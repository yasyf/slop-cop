package detectors

import (
	"regexp"
	"strings"

	"github.com/yasyf/slop-cop/internal/types"
)

// findAll returns one Violation per match of re in text, attributed to ruleID.
// It mirrors the TypeScript helper of the same name.
func findAll(text string, re *regexp.Regexp, ruleID string) []types.Violation {
	var out []types.Violation
	for _, idx := range re.FindAllStringIndex(text, -1) {
		out = append(out, types.Violation{
			RuleID:      ruleID,
			StartIndex:  idx[0],
			EndIndex:    idx[1],
			MatchedText: text[idx[0]:idx[1]],
		})
	}
	return out
}

// paragraphSplit matches one-or-more blank lines (with optional interior whitespace).
var paragraphSplit = regexp.MustCompile(`\n\s*\n`)

// paragraph is a slice of text with its byte offset in the containing document.
type paragraph struct {
	text  string
	start int
}

// splitParagraphs partitions text on blank-line boundaries, preserving byte
// offsets. Empty/whitespace-only paragraphs are dropped.
func splitParagraphs(text string) []paragraph {
	var out []paragraph
	last := 0
	for _, idx := range paragraphSplit.FindAllStringIndex(text, -1) {
		chunk := text[last:idx[0]]
		if strings.TrimSpace(chunk) != "" {
			out = append(out, paragraph{text: chunk, start: last})
		}
		last = idx[1]
	}
	if last < len(text) && strings.TrimSpace(text[last:]) != "" {
		out = append(out, paragraph{text: text[last:], start: last})
	}
	return out
}

// sentenceEnd matches end-of-sentence punctuation runs followed by whitespace.
var sentenceEnd = regexp.MustCompile(`[.!?]+\s+`)

// splitSentences yields sentences preserving their trailing whitespace,
// matching the TypeScript helper's behaviour. Empty chunks are dropped.
func splitSentences(text string) []string {
	var out []string
	last := 0
	for _, idx := range sentenceEnd.FindAllStringIndex(text, -1) {
		seg := text[last:idx[1]]
		if strings.TrimSpace(seg) != "" {
			out = append(out, seg)
		}
		last = idx[1]
	}
	if last < len(text) {
		seg := text[last:]
		if strings.TrimSpace(seg) != "" {
			out = append(out, seg)
		}
	}
	return out
}

// escapeForRegex escapes the ECMA-style metacharacter set used by the
// original detectors. Keeping the set identical ensures the generated
// regexes match the TS source character for character.
func escapeForRegex(s string) string {
	const specials = `.*+?^${}()|[]\`
	var b strings.Builder
	b.Grow(len(s))
	for _, r := range s {
		if strings.ContainsRune(specials, r) {
			b.WriteByte('\\')
		}
		b.WriteRune(r)
	}
	return b.String()
}
