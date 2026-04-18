package main

import (
	"testing"

	"github.com/yasyf/slop-cop/internal/detectors"
	"github.com/yasyf/slop-cop/internal/markdown"
	"github.com/yasyf/slop-cop/internal/types"
)

// Integration tests for the pieces the `check` command composes, exercised
// without spawning a child process.

func TestResolveMarkdown_Auto(t *testing.T) {
	cases := []struct {
		mode string
		path string
		want bool
	}{
		{"auto", "README.md", true},
		{"auto", "README.MD", true},
		{"auto", "doc.MarkDown", true},
		{"auto", "page.mdx", true},
		{"auto", "notes.txt", false},
		{"auto", "", false},
		{"auto", "-", false},
		{"on", "anything.txt", true},
		{"on", "", true},
		{"off", "README.md", false},
	}
	for _, c := range cases {
		got, err := resolveMarkdown(c.mode, c.path)
		if err != nil {
			t.Fatalf("resolveMarkdown(%q,%q) error: %v", c.mode, c.path, err)
		}
		if got != c.want {
			t.Fatalf("resolveMarkdown(%q,%q) = %v, want %v", c.mode, c.path, got, c.want)
		}
	}
}

func TestResolveMarkdown_Invalid(t *testing.T) {
	// "auto|on|off" are the only accepted values; aliases like "yes"/"true"
	// would mask typos, so we require strict input.
	for _, mode := range []string{"maybe", "yes", "true", "no", "false", "1", "0"} {
		if _, err := resolveMarkdown(mode, ""); err == nil {
			t.Fatalf("expected error for --markdown=%q", mode)
		}
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

	// Markdown mode: mask + detect + suppress.
	masked, suppress, _ := markdown.Analyze(src)
	if len(masked) != len(src) {
		t.Fatalf("markdown.Analyze broke length invariant: %d != %d", len(masked), len(src))
	}
	got := markdown.ApplySuppressions(detectors.RunClient(masked), suppress, src)

	// `utilize` should survive only on the heading line (link URL, code span,
	// and fenced block are all masked; no ref-def/autolink in this fixture).
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
		// Offset invariant on every returned violation.
		if v.StartIndex < 0 || v.EndIndex > len(src) || v.EndIndex < v.StartIndex {
			t.Fatalf("violation %+v has out-of-range offsets for src length %d", v, len(src))
		}
		if src[v.StartIndex:v.EndIndex] != v.MatchedText {
			t.Fatalf("offset invariant violated: %+v vs src slice %q", v, src[v.StartIndex:v.EndIndex])
		}
		// No survivor may be a dramatic-fragment on a heading or a
		// staccato-burst across two+ list items.
		if v.RuleID == "dramatic-fragment" && markdown.Overlaps(v.StartIndex, v.EndIndex, suppress, markdown.KindHeading) {
			t.Fatalf("dramatic-fragment survived on a heading span: %+v", v)
		}
		if v.RuleID == "staccato-burst" && markdown.CountOverlapping(v.StartIndex, v.EndIndex, suppress, markdown.KindListItem) >= 2 {
			t.Fatalf("staccato-burst survived across list items: %+v", v)
		}
	}
}
