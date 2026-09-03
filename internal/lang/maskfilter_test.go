package lang_test

import (
	"testing"

	"github.com/yasyf/slop-cop/internal/lang"
	"github.com/yasyf/slop-cop/internal/rules"
	"github.com/yasyf/slop-cop/internal/types"
)

// TestDropMaskMatchesKeepsWordBoundaries asserts a masked run collapses to the
// single space it stood in for rather than vanishing: deleting it outright
// would glue the words either side together and lose the `\b` the detector
// matched on.
func TestDropMaskMatchesKeepsWordBoundaries(t *testing.T) {
	original := "This is robust<!-- an aside -->and fast enough.\n"
	spans := []lang.Range{{Start: 14, End: 31, Kind: lang.KindMasked}}
	vs := []types.Violation{{RuleID: "overused-intensifiers", StartIndex: 8, EndIndex: 14, MatchedText: "robust"}}
	if out := lang.DropMaskMatches(vs, spans, original); len(out) != 1 {
		t.Fatalf("hit on real prose beside a mask should survive; got %+v", out)
	}
}

// TestDropMaskMatchesDropsFillerOnlyHits asserts a hit whose whole match is
// masking filler goes away, and one on the same span with prose in it stays.
func TestDropMaskMatchesDropsFillerOnlyHits(t *testing.T) {
	filler := "The lookup is layer-agnostic (`internal/lang/suppress_data.go`) and works.\n"
	spans := []lang.Range{{Start: 30, End: 62, Kind: lang.KindMasked}}
	vs := []types.Violation{{RuleID: "parenthetical-qualifier", StartIndex: 29, EndIndex: 63}}
	if out := lang.DropMaskMatches(vs, spans, filler); len(out) != 0 {
		t.Fatalf("filler-only parenthetical should be dropped; got %+v", out)
	}

	prose := "The lookup is layer-agnostic (it never inspects a layer) and works.\n"
	vs = []types.Violation{{RuleID: "parenthetical-qualifier", StartIndex: 29, EndIndex: 56}}
	if out := lang.DropMaskMatches(vs, nil, prose); len(out) != 1 {
		t.Fatalf("prose parenthetical should survive; got %+v", out)
	}
}

// TestDeterministicReadsRequiresLLMNotTheTier pins the predicate against the
// stray tier `false-range` carries: it runs client-side, so the filter must
// test it like any other detector rule. Keying off LLMTier exempted it.
func TestDeterministicReadsRequiresLLMNotTheTier(t *testing.T) {
	r, ok := rules.ByID["false-range"]
	if !ok || r.RequiresLLM || r.LLMTier == "" {
		t.Fatalf("fixture assumption gone: false-range = %+v", r)
	}
	original := "The problem ranges from trivial <!-- an aside --> to catastrophic.\n"
	spans := []lang.Range{{Start: 32, End: 49, Kind: lang.KindMasked}}
	vs := []types.Violation{{RuleID: "false-range", StartIndex: 0, EndIndex: 20}}
	if out := lang.DropMaskMatches(vs, spans, original); len(out) != 0 {
		t.Fatalf("false-range must be filtered like any client-side rule; got %+v", out)
	}
}
