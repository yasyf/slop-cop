package main

import (
	"testing"

	"github.com/yasyf/slop-cop/internal/detectors"
	"github.com/yasyf/slop-cop/internal/lang"
	"github.com/yasyf/slop-cop/internal/types"
)

// Integration tests for the language-selection and masking pipeline the
// `check` command composes, exercised without spawning a child process.

func TestResolveLangAutoByExtension(t *testing.T) {
	cases := []struct {
		path string
		want string
	}{
		{"README.md", "markdown"},
		{"README.MD", "markdown"},
		{"doc.MarkDown", "markdown"},
		{"page.mdx", "markdown"},
		{"index.html", "html"},
		{"index.HTM", "html"},
		{"app.jsx", "jsx"},
		{"App.TSX", "tsx"},
		{"util.ts", "ts"},
		{"util.mts", "ts"},
		{"util.cts", "ts"},
		{"util.js", "js"},
		{"util.mjs", "js"},
		{"util.cjs", "js"},
		{"notes.txt", "text"},
		{"", "text"},
		{"-", "text"},
	}
	for _, c := range cases {
		_, name, err := resolveLang("auto", c.path)
		if err != nil {
			t.Fatalf("resolveLang(auto,%q) err=%v", c.path, err)
		}
		if name != c.want {
			t.Fatalf("resolveLang(auto,%q) = %q, want %q", c.path, name, c.want)
		}
	}
}

func TestResolveLangExplicitOverride(t *testing.T) {
	_, name, err := resolveLang("jsx", "README.md")
	if err != nil {
		t.Fatal(err)
	}
	if name != "jsx" {
		t.Fatalf("--lang=jsx over .md should resolve to jsx; got %q", name)
	}
}

func TestResolveLangExplicitText(t *testing.T) {
	a, name, err := resolveLang("text", "README.md")
	if err != nil {
		t.Fatal(err)
	}
	if a != nil || name != "text" {
		t.Fatalf("--lang=text should yield (nil, text); got (%v, %q)", a, name)
	}
}

func TestResolveLangInvalid(t *testing.T) {
	if _, _, err := resolveLang("pascal", "x.md"); err == nil {
		t.Fatal("expected error for --lang=pascal")
	}
}

// TestCheckMarkdown_EndToEnd walks a realistic markdown document through the
// same steps the `check` RunE does and asserts the markdown-mode
// suppressions fire and every returned violation carries an accurate
// matchedText slice back into the original input.
func TestCheckMarkdown_EndToEnd(t *testing.T) {
	src := `---
title: robust overview
---

# Heading With utilize

Prose with [click](https://example.com/utilize) and ` + "`utilize` in code" + `.

## Usage

Another paragraph here that contains robust prose.

- First.
- Second.
- Third.

` + "```bash\nutilize this in code\n```\n"

	hasRule := func(vs []types.Violation, id string) bool {
		for _, v := range vs {
			if v.RuleID == id {
				return true
			}
		}
		return false
	}

	// Plain mode: markdown structure should yield several false positives.
	plain := detectors.RunClient(src)
	if !hasRule(plain, "dramatic-fragment") {
		t.Fatalf("plain mode should fire dramatic-fragment on headings; got %+v", plain)
	}
	if !hasRule(plain, "elevated-register") {
		t.Fatalf("plain mode should fire elevated-register on the unmasked code/URL 'utilize'")
	}

	a, ok := lang.ByName("markdown")
	if !ok || a == nil {
		t.Fatal("markdown analyzer not registered")
	}
	masked, suppress, err := a.Analyze(src)
	if err != nil {
		t.Fatal(err)
	}
	if len(masked) != len(src) {
		t.Fatalf("Analyze broke length invariant: %d != %d", len(masked), len(src))
	}
	got := a.ApplySuppressions(detectors.RunClient(masked), suppress, src)

	elevated := 0
	for _, v := range got {
		if v.RuleID != "elevated-register" {
			continue
		}
		elevated++
		if v.MatchedText != "utilize" {
			t.Fatalf("elevated-register matchedText = %q, want 'utilize'", v.MatchedText)
		}
	}
	if elevated == 0 {
		t.Fatalf("expected at least one elevated-register hit (on the heading text)")
	}

	for _, v := range got {
		if v.StartIndex < 0 || v.EndIndex > len(src) || v.EndIndex < v.StartIndex {
			t.Fatalf("violation %+v has out-of-range offsets for src length %d", v, len(src))
		}
		if src[v.StartIndex:v.EndIndex] != v.MatchedText {
			t.Fatalf("offset invariant violated: %+v vs src slice %q", v, src[v.StartIndex:v.EndIndex])
		}
		if v.RuleID == "dramatic-fragment" && lang.Overlaps(v.StartIndex, v.EndIndex, suppress, lang.KindHeading) {
			t.Fatalf("dramatic-fragment survived on a heading span: %+v", v)
		}
		if v.RuleID == "staccato-burst" && lang.CountOverlapping(v.StartIndex, v.EndIndex, suppress, lang.KindListItem) >= 2 {
			t.Fatalf("staccato-burst survived across list items: %+v", v)
		}
	}
}
