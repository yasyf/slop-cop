package main

import (
	"testing"

	"github.com/yasyf/slop-cop/internal/detectors"
	"github.com/yasyf/slop-cop/internal/markdown"
	"github.com/yasyf/slop-cop/internal/types"
)

// These tests exercise the pieces the `check` command composes — the
// resolveMarkdown decision, the applyMarkdownSuppressions post-filter, and
// the offset invariant we expose to consumers — without spawning a child
// process. The `check` RunE function drives these helpers directly.

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
	if _, err := resolveMarkdown("maybe", ""); err == nil {
		t.Fatalf("expected error for --markdown=maybe")
	}
}

// TestCheckMarkdown_EndToEnd walks a realistic markdown document through
// the same steps the `check` RunE does and asserts the markdown-mode
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

	// Plain mode: many markdown-structure hits.
	plainViolations := detectors.RunClient(src)
	plainHas := func(id string) bool {
		for _, v := range plainViolations {
			if v.RuleID == id {
				return true
			}
		}
		return false
	}
	if !plainHas("dramatic-fragment") {
		t.Fatalf("plain mode should fire dramatic-fragment on headings; got %+v", plainViolations)
	}
	if !plainHas("elevated-register") {
		t.Fatalf("plain mode should fire elevated-register on the unmasked 'utilize' in code/URL")
	}

	// Markdown mode: run detectors on the masked copy + apply suppressions.
	masked, suppress, _ := markdown.Analyze(src)
	if len(masked) != len(src) {
		t.Fatalf("markdown.Analyze broke length invariant: %d != %d", len(masked), len(src))
	}
	mdViolations := detectors.RunClient(masked)
	mdViolations = applyMarkdownSuppressions(mdViolations, suppress, src)

	// `utilize` should only survive on the heading ("# Heading With utilize")
	// and any prose link-text contexts. Code span + URL + fenced block are
	// masked; the ref-def / autolink cases are absent here.
	elevatedHits := 0
	for _, v := range mdViolations {
		if v.RuleID == "elevated-register" {
			elevatedHits++
			// Each surviving hit must actually be on the word "utilize" in the
			// original source (heading only in this fixture).
			if v.MatchedText != "utilize" {
				t.Fatalf("elevated-register matchedText = %q, want 'utilize'", v.MatchedText)
			}
			if src[v.StartIndex:v.EndIndex] != v.MatchedText {
				t.Fatalf("offset invariant violated for %+v (src slice = %q)", v, src[v.StartIndex:v.EndIndex])
			}
		}
	}
	if elevatedHits == 0 {
		t.Fatalf("expected at least one elevated-register hit (on the heading text), got none")
	}

	// dramatic-fragment should be suppressed on the headings.
	for _, v := range mdViolations {
		if v.RuleID == "dramatic-fragment" {
			// If any survive, ensure they don't overlap a heading.
			if markdown.Overlaps(v.StartIndex, v.EndIndex, suppress, markdown.KindHeading) {
				t.Fatalf("dramatic-fragment survived on a heading span: %+v", v)
			}
		}
	}

	// staccato-burst across the bulleted list should be suppressed.
	for _, v := range mdViolations {
		if v.RuleID == "staccato-burst" {
			if markdown.CountOverlapping(v.StartIndex, v.EndIndex, suppress, markdown.KindListItem) >= 2 {
				t.Fatalf("staccato-burst survived across list items: %+v", v)
			}
		}
	}

	// Offset invariant for every remaining violation.
	for _, v := range mdViolations {
		if v.StartIndex < 0 || v.EndIndex > len(src) || v.EndIndex < v.StartIndex {
			t.Fatalf("violation %+v has out-of-range offsets for src length %d", v, len(src))
		}
		if src[v.StartIndex:v.EndIndex] != v.MatchedText {
			t.Fatalf("offset invariant violated: %+v vs src slice %q", v, src[v.StartIndex:v.EndIndex])
		}
	}
}

// TestApplyMarkdownSuppressions_DropsDramaticFragmentOnHeadings constructs
// synthetic inputs so the post-filter is exercised deterministically.
func TestApplyMarkdownSuppressions_DropsDramaticFragmentOnHeadings(t *testing.T) {
	vs := []types.Violation{
		{RuleID: "dramatic-fragment", StartIndex: 0, EndIndex: 10, MatchedText: ""},
		{RuleID: "overused-intensifiers", StartIndex: 12, EndIndex: 18, MatchedText: ""},
	}
	original := "## Heading\n\nrobust prose\n"
	suppress := []markdown.Range{
		{Start: 0, End: 10, Kind: markdown.KindHeading},
	}
	out := applyMarkdownSuppressions(vs, suppress, original)
	if len(out) != 1 {
		t.Fatalf("expected 1 surviving violation, got %d: %+v", len(out), out)
	}
	if out[0].RuleID != "overused-intensifiers" {
		t.Fatalf("wrong survivor: %+v", out[0])
	}
	if out[0].MatchedText != "robust" {
		t.Fatalf("matchedText re-slice failed: got %q", out[0].MatchedText)
	}
}

func TestApplyMarkdownSuppressions_DropsStaccatoAcrossListItems(t *testing.T) {
	// A staccato burst spanning three list items should be dropped.
	vs := []types.Violation{
		{RuleID: "staccato-burst", StartIndex: 0, EndIndex: 30, MatchedText: ""},
	}
	original := "- First item.\n- Second item.\n- Third item.\n"
	suppress := []markdown.Range{
		{Start: 0, End: 14, Kind: markdown.KindListItem},
		{Start: 14, End: 29, Kind: markdown.KindListItem},
		{Start: 29, End: 43, Kind: markdown.KindListItem},
	}
	out := applyMarkdownSuppressions(vs, suppress, original)
	if len(out) != 0 {
		t.Fatalf("expected staccato-burst to be dropped, got %+v", out)
	}
}

func TestApplyMarkdownSuppressions_KeepsStaccatoInSingleItem(t *testing.T) {
	// A staccato burst fully inside one list item is still real.
	vs := []types.Violation{
		{RuleID: "staccato-burst", StartIndex: 2, EndIndex: 40, MatchedText: ""},
	}
	original := "- A. B. C. D. continuation of the item prose.\n"
	suppress := []markdown.Range{
		{Start: 0, End: len(original), Kind: markdown.KindListItem},
	}
	out := applyMarkdownSuppressions(vs, suppress, original)
	if len(out) != 1 {
		t.Fatalf("expected staccato-burst to survive within a single list item, got %+v", out)
	}
}
