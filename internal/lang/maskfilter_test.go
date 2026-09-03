package lang_test

import (
	"testing"

	"github.com/yasyf/slop-cop/internal/detectors"
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

// TestHitNoDetectorProducedIsKept pins the absence of a false negative: a
// violation the detectors never made — an LLM pass's — is not in the masked
// run either, so the filter leaves it alone whatever its rule. `false-range`
// is the rule that makes this concrete, because it runs client-side while
// carrying a stray sentence tier, so the LLM pass can report it too and no
// metadata field separates the two provenances.
func TestHitNoDetectorProducedIsKept(t *testing.T) {
	r, ok := rules.ByID["false-range"]
	if !ok || r.RequiresLLM || r.LLMTier == "" {
		t.Fatalf("fixture assumption gone: false-range = %+v", r)
	}
	original := "The problem ranges from trivial <!-- an aside --> to catastrophic.\n"
	spans := []lang.Range{{Start: 32, End: 49, Kind: lang.KindMasked}}
	vs := []types.Violation{{RuleID: "false-range", StartIndex: 0, EndIndex: 20}}
	if out := lang.DropMaskMatches(vs, spans, original); len(out) != 1 {
		t.Fatalf("a hit no detector produced must survive; got %+v", out)
	}
}

// TestDetectorHitIsJudgedByItsRuleNotItsMetadata is the companion: real
// detector hits over the masked text, of which the mask-only parenthetical is
// dropped and the `false-range` in the surrounding prose is kept. No rule is
// exempt from the filter and none is falsely dropped by it, so the same rule
// that survives here as an LLM hit at an unproduced span is still judged on
// its pattern when a detector actually made it.
func TestDetectorHitIsJudgedByItsRuleNotItsMetadata(t *testing.T) {
	original := "The bug (`internal/lang/suppress_data.go`) appeared from nowhere.\n"
	spans := []lang.Range{{Start: 9, End: 41, Kind: lang.KindMasked}}
	masked := []byte(original)
	for i := spans[0].Start; i < spans[0].End; i++ {
		masked[i] = ' '
	}
	vs := detectors.RunClient(string(masked))
	if len(vs) != 2 {
		t.Fatalf("fixture should trip parenthetical-qualifier and false-range; got %+v", vs)
	}
	out := lang.DropMaskMatches(vs, spans, original)
	if len(out) != 1 || out[0].RuleID != "false-range" {
		t.Fatalf("false-range should survive and the masked parenthetical should not; got %+v", out)
	}
}
