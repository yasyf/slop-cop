package markdown

import (
	"strings"
	"testing"

	"github.com/yasyf/slop-cop/internal/types"
)

// TestApplySuppressions_DropsDramaticFragmentOnHeadings asserts the post
// filter removes a dramatic-fragment hit that falls inside a heading span
// and preserves an unrelated hit alongside it.
func TestApplySuppressions_DropsDramaticFragmentOnHeadings(t *testing.T) {
	vs := []types.Violation{
		{RuleID: "dramatic-fragment", StartIndex: 0, EndIndex: 10, MatchedText: ""},
		{RuleID: "overused-intensifiers", StartIndex: 12, EndIndex: 18, MatchedText: ""},
	}
	original := "## Heading\n\nrobust prose\n"
	suppress := []Range{
		{Start: 0, End: 10, Kind: KindHeading},
	}
	out := ApplySuppressions(vs, suppress, original)
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

// TestApplySuppressions_DropsStaccatoAcrossListItems asserts a staccato
// burst that straddles 2+ consecutive list items is removed.
func TestApplySuppressions_DropsStaccatoAcrossListItems(t *testing.T) {
	vs := []types.Violation{
		{RuleID: "staccato-burst", StartIndex: 0, EndIndex: 30, MatchedText: ""},
	}
	original := "- First item.\n- Second item.\n- Third item.\n"
	suppress := []Range{
		{Start: 0, End: 14, Kind: KindListItem},
		{Start: 14, End: 29, Kind: KindListItem},
		{Start: 29, End: 43, Kind: KindListItem},
	}
	out := ApplySuppressions(vs, suppress, original)
	if len(out) != 0 {
		t.Fatalf("expected staccato-burst to be dropped, got %+v", out)
	}
}

// TestApplySuppressions_KeepsStaccatoInSingleItem asserts a staccato burst
// fully contained in one list item is still a real hit.
func TestApplySuppressions_KeepsStaccatoInSingleItem(t *testing.T) {
	vs := []types.Violation{
		{RuleID: "staccato-burst", StartIndex: 2, EndIndex: 40, MatchedText: ""},
	}
	original := "- A. B. C. D. continuation of the item prose.\n"
	suppress := []Range{
		{Start: 0, End: len(original), Kind: KindListItem},
	}
	out := ApplySuppressions(vs, suppress, original)
	if len(out) != 1 {
		t.Fatalf("expected staccato-burst to survive within a single list item, got %+v", out)
	}
}

// TestApplySuppressions_DropsLongSentenceOnSetextHeading asserts a long
// setext heading is not a runaway sentence. The setext form is the case the
// detector's own textual ATX guard cannot see, so only the parse catches it.
func TestApplySuppressions_DropsLongSentenceOnSetextHeading(t *testing.T) {
	title := "A heading carrying far more words than any reasonable prose sentence would ever need to carry"
	original := title + "\n" + strings.Repeat("=", len(title)) + "\n\nBody prose.\n"
	_, suppress, _ := Analyze(original)
	vs := []types.Violation{
		{RuleID: "long-sentence", StartIndex: 0, EndIndex: len(title)},
	}
	out := ApplySuppressions(vs, suppress, original)
	if len(out) != 0 {
		t.Fatalf("expected long-sentence on a setext heading to be dropped, got %+v", out)
	}
}

// TestApplySuppressions_DropsLengthRulesOnATXHeading asserts both length
// rules drop on an ATX heading.
func TestApplySuppressions_DropsLengthRulesOnATXHeading(t *testing.T) {
	heading := "## A heading carrying far more words than any reasonable prose sentence would ever need to carry"
	original := heading + "\n\nBody prose.\n"
	_, suppress, _ := Analyze(original)
	vs := []types.Violation{
		{RuleID: "long-sentence", StartIndex: 0, EndIndex: len(heading)},
		{RuleID: "long-paragraph", StartIndex: 0, EndIndex: len(heading)},
	}
	out := ApplySuppressions(vs, suppress, original)
	if len(out) != 0 {
		t.Fatalf("expected both length rules on an ATX heading to be dropped, got %+v", out)
	}
}

// TestApplySuppressions_DropsLongParagraphAcrossListItems asserts a loose
// list — which parses as one paragraph carrying every item's sentences — is
// the list's natural shape, not a sprawling paragraph.
func TestApplySuppressions_DropsLongParagraphAcrossListItems(t *testing.T) {
	original := strings.Repeat("- The item carries one short sentence.\n\n", 8)
	_, suppress, _ := Analyze(original)
	vs := []types.Violation{
		{RuleID: "long-paragraph", StartIndex: 0, EndIndex: len(original)},
	}
	out := ApplySuppressions(vs, suppress, original)
	if len(out) != 0 {
		t.Fatalf("expected long-paragraph across a loose list to be dropped, got %+v", out)
	}
}

// TestApplySuppressions_KeepsLongParagraphInProse is the narrowness proof:
// the document carries a heading and a list, so both suppression ranges are
// live, yet an eight-sentence prose wall that touches neither still stands.
func TestApplySuppressions_KeepsLongParagraphInProse(t *testing.T) {
	wall := "The build runs first. The tests run next. The linter follows after that. " +
		"The report lands last. None of this is a list. None of this is a heading. " +
		"The paragraph simply sprawls onward. That is the rule's whole point.\n"
	original := "## Pipeline\n\n- one item\n- another item\n\n" + wall
	_, suppress, _ := Analyze(original)
	start := strings.Index(original, wall)
	vs := []types.Violation{
		{RuleID: "long-paragraph", StartIndex: start, EndIndex: start + len(wall) - 1},
	}
	out := ApplySuppressions(vs, suppress, original)
	if len(out) != 1 {
		t.Fatalf("expected an 8-sentence prose paragraph to survive, got %+v", out)
	}
}

// TestApplySuppressions_KeepsLongSentenceInProse asserts a 45-word body
// sentence still reports even in a document whose heading ranges are live.
func TestApplySuppressions_KeepsLongSentenceInProse(t *testing.T) {
	sentence := "The release pipeline builds every target from a single runner and then " +
		"uploads each archive to the tagged release, and because the toolchain is pinned " +
		"in the repository the same commit produces the same bytes on every machine that " +
		"anyone happens to run it on."
	original := "## Releases\n\n" + sentence + "\n"
	_, suppress, _ := Analyze(original)
	start := strings.Index(original, sentence)
	vs := []types.Violation{
		{RuleID: "long-sentence", StartIndex: start, EndIndex: start + len(sentence)},
	}
	out := ApplySuppressions(vs, suppress, original)
	if len(out) != 1 {
		t.Fatalf("expected a 45-word body sentence to survive, got %+v", out)
	}
}
