// Package lang defines the mask-and-suppress contract every input-language
// mode implements. A language analyzer turns source text into a byte-aligned
// "masked" copy where non-prose regions are overwritten with ASCII spaces, so
// slop-cop's prose detectors can run without matching inside code, URLs,
// tags, or comments' surrounding syntax. The raw structural ranges are also
// returned so post-filters can suppress rule-specific false positives (e.g.
// headings shouldn't trigger "dramatic-fragment").
//
// New language packages register themselves in init() via Register, keyed by
// canonical name and by file extensions for auto-detection.
package lang

import "github.com/yasyf/slop-cop/internal/types"

// Kind identifies what a structural Range corresponds to. Values are strings
// so they can round-trip through JSON without ambiguity and so new language
// modules can add new kinds without touching this file's const block.
type Kind string

const (
	// KindHeading covers heading-like prose (markdown ATX/setext, HTML h1-h6).
	KindHeading Kind = "heading"
	// KindListItem covers a single ordered/unordered list item.
	KindListItem Kind = "list-item"
	// KindCodeBlock covers fenced/indented code in markdown or
	// <pre>/<code>/<script>/<style> in HTML. Usually redundant with masking
	// but kept so kind-based filters stay symmetric across languages.
	KindCodeBlock Kind = "code-block"
	// KindComment covers any single-line or block comment in source code.
	KindComment Kind = "comment"
	// KindJSDoc covers JSDoc/TSDoc blocks (/** ... */) specifically.
	KindJSDoc Kind = "jsdoc"
	// KindStringLiteral covers the prose interior of a string literal.
	KindStringLiteral Kind = "string-literal"
	// KindJSXText covers text children between JSX opening and closing tags.
	KindJSXText Kind = "jsx-text"
)

// Range is a half-open byte span [Start, End) into the original source,
// tagged with the structural kind that produced it.
type Range struct {
	Start int  `json:"start"`
	End   int  `json:"end"`
	Kind  Kind `json:"kind,omitempty"`
}

// Analyzer is the mask-and-suppress contract every input-language mode
// implements. Implementations must be safe for concurrent use; a single
// registered Analyzer instance is shared across all calls.
//
// Analyze invariants:
//   - len(masked) == len(src)
//   - every '\n' in src preserved at the same offset in masked
//   - bytes outside any mask region are byte-identical to src
//   - masked bytes are ASCII space (0x20), except preserved newlines
//
// ApplySuppressions drops violations that are structural false positives for
// this language and re-populates MatchedText from the original source (since
// detectors ran against the masked copy where those bytes were spaces).
type Analyzer interface {
	Name() string
	Analyze(src string) (masked string, suppress []Range, err error)
	ApplySuppressions(violations []types.Violation, suppress []Range, original string) []types.Violation
}

// Overlaps reports whether [vStart, vEnd) overlaps any Range in spans whose
// Kind matches want. Shared helper for ApplySuppressions implementations.
func Overlaps(vStart, vEnd int, spans []Range, want Kind) bool {
	for _, s := range spans {
		if s.Kind != want {
			continue
		}
		if vStart < s.End && vEnd > s.Start {
			return true
		}
	}
	return false
}

// CountOverlapping returns the number of Ranges of the given kind that
// [vStart, vEnd) touches. Used by suppressions that only fire when a
// violation straddles multiple structural elements (e.g. staccato-burst
// across consecutive list items).
func CountOverlapping(vStart, vEnd int, spans []Range, want Kind) int {
	n := 0
	for _, s := range spans {
		if s.Kind != want {
			continue
		}
		if vStart < s.End && vEnd > s.Start {
			n++
		}
	}
	return n
}

// RestoreMatchedText copies the original-source slice [start, end) back
// into v.MatchedText when the indices are valid. Detectors ran against the
// masked string where those bytes were spaces, so every ApplySuppressions
// implementation needs this step before returning.
func RestoreMatchedText(v *types.Violation, original string) {
	if v.StartIndex >= 0 && v.EndIndex <= len(original) && v.EndIndex >= v.StartIndex {
		v.MatchedText = original[v.StartIndex:v.EndIndex]
	}
}
