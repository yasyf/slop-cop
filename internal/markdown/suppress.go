package markdown

import "github.com/yasyf/slop-cop/internal/types"

// ApplySuppressions drops violations that correspond to false positives on
// markdown structural elements, and re-populates MatchedText from the
// original input so consumers never see the masked whitespace.
//
// Suppression rules currently applied:
//
//   - `dramatic-fragment` inside an ATX or setext heading range is a false
//     positive; headings are not "short dramatic paragraphs".
//   - `staccato-burst` that straddles two or more consecutive list items is
//     the list's natural rhythm, not a rhetorical device.
//
// Pass the result of Analyze(src) as `suppress` and the original source as
// `original`. The returned slice is a fresh allocation; callers need not
// worry about aliasing with the input.
func ApplySuppressions(vs []types.Violation, suppress []Range, original string) []types.Violation {
	out := make([]types.Violation, 0, len(vs))
	for _, v := range vs {
		switch v.RuleID {
		case "dramatic-fragment":
			if Overlaps(v.StartIndex, v.EndIndex, suppress, KindHeading) {
				continue
			}
		case "staccato-burst":
			if CountOverlapping(v.StartIndex, v.EndIndex, suppress, KindListItem) >= 2 {
				continue
			}
		}
		if v.StartIndex >= 0 && v.EndIndex <= len(original) && v.EndIndex >= v.StartIndex {
			v.MatchedText = original[v.StartIndex:v.EndIndex]
		}
		out = append(out, v)
	}
	return out
}
