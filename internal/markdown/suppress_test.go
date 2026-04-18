package markdown

import (
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
