package lang

import (
	"strings"

	"github.com/yasyf/slop-cop/internal/detectors"
	"github.com/yasyf/slop-cop/internal/types"
)

// DropMaskMatches removes violations that only exist because masking replaced
// non-prose bytes with spaces. Several detector patterns carry length
// quantifiers a run of filler satisfies — `[^)]{20,}` inside parentheses, for
// instance — so a masked code span reads to them as a long parenthetical.
//
// The test is the rule's own pattern, run twice. Every detector layer sees the
// masked text the first pass saw, and again with each run of [KindMasked] bytes
// collapsed to the single space it stood in for. A violation is dropped only
// when it is in the first set and absent from the second: the detectors made
// it, and the prose alone does not sustain it. Half-masked matches are covered
// too, since the remaining prose either satisfies the pattern on its own or it
// does not.
//
// Nothing here consults rule metadata, because no field records which pass
// emitted a violation. A hit the detectors never made — an LLM pass's, whatever
// its rule — is absent from the first set and kept untouched, and that stays
// true for `false-range`, which runs client-side while carrying a stray
// sentence tier that no metadata predicate reads correctly for both passes.
func DropMaskMatches(vs []types.Violation, spans []Range, original string) []types.Violation {
	masked, prose, offset := splitOnMask(original, spans)
	if len(prose) == len(original) {
		return vs
	}
	survived := firedSpans(prose)
	// Deferred, not conditional: a violation the collapsed text reproduces is
	// kept whatever made it, so the masked-text pass is only needed once one
	// fails to reproduce. Documents with no artifact never pay for it.
	var produced map[firedSpan]struct{}
	out := make([]types.Violation, 0, len(vs))
	for _, v := range vs {
		if v.StartIndex < 0 || v.EndIndex > len(original) || v.EndIndex < v.StartIndex {
			out = append(out, v)
			continue
		}
		if _, ok := survived[firedSpan{v.RuleID, offset[v.StartIndex], offset[v.EndIndex]}]; ok {
			out = append(out, v)
			continue
		}
		if produced == nil {
			produced = firedSpans(masked)
		}
		if _, ok := produced[firedSpan{v.RuleID, v.StartIndex, v.EndIndex}]; !ok {
			out = append(out, v)
		}
	}
	return out
}

// splitOnMask returns the two texts the filter compares — original with every
// masked run turned to spaces, which is what the detectors first saw, and the
// same with each run collapsed to one space — plus a table mapping every
// original byte offset (0..len) to its offset in the collapsed text.
//
// Newlines survive both, as they do under masking. The space a collapsed run
// leaves behind keeps the word and sentence boundaries the filler stood in for,
// so only the length it contributed is gone.
func splitOnMask(original string, spans []Range) (string, string, []int) {
	isMask := make([]bool, len(original))
	for _, s := range spans {
		if s.Kind != KindMasked {
			continue
		}
		start, end := s.Start, s.End
		if start < 0 {
			start = 0
		}
		if end > len(original) {
			end = len(original)
		}
		for i := start; i < end; i++ {
			isMask[i] = true
		}
	}
	var masked, prose strings.Builder
	masked.Grow(len(original))
	prose.Grow(len(original))
	offset := make([]int, len(original)+1)
	inRun := false
	for i := 0; i < len(original); i++ {
		offset[i] = prose.Len()
		if isMask[i] && original[i] != '\n' {
			masked.WriteByte(' ')
			if !inRun {
				prose.WriteByte(' ')
				inRun = true
			}
			continue
		}
		inRun = false
		masked.WriteByte(original[i])
		prose.WriteByte(original[i])
	}
	offset[len(original)] = prose.Len()
	return masked.String(), prose.String(), offset
}

// firedSpan identifies one detector hit by rule and span, so two runs can be
// compared position by position.
type firedSpan struct {
	ruleID string
	start  int
	end    int
}

// firedSpans runs every detector layer over text and returns the hits it
// produced. All three run whatever layers the caller selected, because a rule
// ID does not name the layer that emitted it: the base layer reports its plain
// word swaps under the slop layer's `elevated-register`, so a per-rule
// shortcut would miss hits it never gave the right detector a chance to make.
func firedSpans(text string) map[firedSpan]struct{} {
	out := map[firedSpan]struct{}{}
	for _, run := range []func(string) []types.Violation{
		detectors.RunClient,
		detectors.RunBase,
		detectors.RunGoogle,
	} {
		for _, v := range run(text) {
			out[firedSpan{v.RuleID, v.StartIndex, v.EndIndex}] = struct{}{}
		}
	}
	return out
}
