package lang

import (
	"strings"

	"github.com/yasyf/slop-cop/internal/detectors"
	"github.com/yasyf/slop-cop/internal/rules"
	"github.com/yasyf/slop-cop/internal/types"
)

// DropMaskMatches removes violations that only exist because masking replaced
// non-prose bytes with spaces. Several detector patterns carry length
// quantifiers a run of filler satisfies — `[^)]{20,}` inside parentheses, for
// instance — so a masked code span reads to them as a long parenthetical.
//
// The test is the rule's own pattern, not a heuristic: each run of [KindMasked]
// bytes collapses to the single space it stands in for, every deterministic
// layer is re-run over that prose-only text, and a violation is kept only when
// its rule fires again at the same span. Half-masked matches are covered too,
// since the remaining prose either satisfies the pattern on its own or it
// doesn't.
//
// Rules an LLM pass can emit are left alone: no detector reproduces them, so
// re-running the layers proves nothing about them.
func DropMaskMatches(vs []types.Violation, spans []Range, original string) []types.Violation {
	prose, offset := collapseMasked(original, spans)
	if len(prose) == len(original) {
		return vs
	}
	fired := firedSpans(prose)
	out := make([]types.Violation, 0, len(vs))
	for _, v := range vs {
		if !deterministic(v.RuleID) || v.StartIndex < 0 || v.EndIndex > len(original) || v.EndIndex < v.StartIndex {
			out = append(out, v)
			continue
		}
		if _, ok := fired[firedSpan{v.RuleID, offset[v.StartIndex], offset[v.EndIndex]}]; ok {
			out = append(out, v)
		}
	}
	return out
}

// collapseMasked returns original with each run of masked bytes reduced to one
// space, plus a table mapping every original byte offset (0..len) to its offset
// in the returned string. Newlines survive as they do under masking, and the
// surviving space keeps the word and sentence boundaries the filler stood in
// for — only the length it contributed is gone.
func collapseMasked(original string, spans []Range) (string, []int) {
	masked := make([]bool, len(original))
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
			masked[i] = true
		}
	}
	var b strings.Builder
	b.Grow(len(original))
	offset := make([]int, len(original)+1)
	inRun := false
	for i := 0; i < len(original); i++ {
		offset[i] = b.Len()
		if masked[i] && original[i] != '\n' {
			if !inRun {
				b.WriteByte(' ')
				inRun = true
			}
			continue
		}
		inRun = false
		b.WriteByte(original[i])
	}
	offset[len(original)] = b.Len()
	return b.String(), offset
}

// firedSpan identifies one detector hit by rule and span, so a re-run can be
// compared against the original run position by position.
type firedSpan struct {
	ruleID string
	start  int
	end    int
}

// firedSpans re-runs every detector layer over prose and returns the hits it
// produced. All three run whatever layers the caller selected, because a rule
// ID does not name the layer that emitted it: the base layer reports its plain
// word swaps under the slop layer's `elevated-register`, so a per-rule
// shortcut would drop hits it never gave the right detector a chance to make.
func firedSpans(prose string) map[firedSpan]struct{} {
	out := map[firedSpan]struct{}{}
	for _, run := range []func(string) []types.Violation{
		detectors.RunClient,
		detectors.RunBase,
		detectors.RunGoogle,
	} {
		for _, v := range run(prose) {
			out[firedSpan{v.RuleID, v.StartIndex, v.EndIndex}] = struct{}{}
		}
	}
	return out
}

// deterministic reports whether a rule is produced by the detectors alone. A
// rule either pass can emit is excluded: an LLM hit would never survive a
// detector re-run.
func deterministic(ruleID string) bool {
	r, ok := rules.ByID[ruleID]
	return ok && !r.RequiresLLM && r.LLMTier == ""
}
