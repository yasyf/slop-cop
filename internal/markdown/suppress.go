package markdown

import (
	"github.com/yasyf/slop-cop/internal/lang"
	"github.com/yasyf/slop-cop/internal/types"
)

// ApplySuppressions drops violations that correspond to false positives on
// markdown structural elements, and re-populates MatchedText from the
// original input so consumers never see the masked whitespace.
//
// Suppression rules currently applied:
//
//   - `dramatic-fragment` inside an ATX or setext heading range is a false
//     positive; headings are not "short dramatic paragraphs".
//   - `long-sentence` inside a heading range is a title, not a runaway
//     sentence. The detector's own ATX guard is textual, so only the parse
//     sees the setext (`===`/`---` underline) form.
//   - `staccato-burst` that straddles two or more consecutive list items is
//     the list's natural rhythm, not a rhetorical device.
//   - `long-paragraph` inside a heading range, or straddling two or more
//     list items, is the document's structure showing through: a loose list
//     parses as one paragraph carrying every item's sentences.
//
// Pass the result of Analyze(src) as `suppress` and the original source as
// `original`. The returned slice is a fresh allocation; callers need not
// worry about aliasing with the input.
func ApplySuppressions(vs []types.Violation, suppress []lang.Range, original string) []types.Violation {
	out := make([]types.Violation, 0, len(vs))
	for _, v := range vs {
		if lang.SuppressedByKind(v, suppress) {
			continue
		}
		switch v.RuleID {
		case "dramatic-fragment", "long-sentence":
			if lang.Overlaps(v.StartIndex, v.EndIndex, suppress, lang.KindHeading) {
				continue
			}
		case "staccato-burst":
			if lang.CountOverlapping(v.StartIndex, v.EndIndex, suppress, lang.KindListItem) >= 2 {
				continue
			}
		case "long-paragraph":
			if lang.Overlaps(v.StartIndex, v.EndIndex, suppress, lang.KindHeading) {
				continue
			}
			if lang.CountOverlapping(v.StartIndex, v.EndIndex, suppress, lang.KindListItem) >= 2 {
				continue
			}
		}
		lang.RestoreMatchedText(&v, original)
		out = append(out, v)
	}
	return out
}
