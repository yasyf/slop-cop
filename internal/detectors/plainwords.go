package detectors

import (
	"regexp"
	"strings"

	"github.com/yasyf/slop-cop/internal/types"
)

// phraseSep separates the words of a multi-word entry. It absorbs a wrapped
// line, so "in order\nto" and "in order\r\nto" match while "in order\n\nto"
// — a paragraph break — does not.
const phraseSep = `(?:[ \t]+|[ \t]*\r?\n[ \t]*)`

// compileSubs precompiles one whole-word regex per substitution, once, at
// package init.
func compileSubs(subs []plainSub) []*regexp.Regexp {
	out := make([]*regexp.Regexp, 0, len(subs))
	for _, s := range subs {
		out = append(out, regexp.MustCompile(`(?i)\b`+
			strings.ReplaceAll(escapeForRegex(s.from), " ", phraseSep)+`\b`))
	}
	return out
}

var (
	plainWordRes   = compileSubs(plainWordSubs)
	plainPhraseRes = compileSubs(plainPhraseSubs)
)

// DetectPlainWord reports elevated vocabulary that has a plain replacement,
// emitting the existing "elevated-register" rule with the replacement as its
// suggested change. It is a sibling of [DetectElevatedRegister], not a
// modification of it: the upstream list and detector stay byte-untouched, and
// this one runs in [RunBase] only.
func DetectPlainWord(text string) []types.Violation {
	out := findSubs(text, "elevated-register", plainWordSubs, plainWordRes)
	out = append(out, findSubs(text, "elevated-register", plainPhraseSubs, plainPhraseRes)...)
	kept := out[:0]
	for _, v := range out {
		if hyphenAttached(text, v.StartIndex, v.EndIndex) {
			continue
		}
		kept = append(kept, v)
	}
	return kept
}

// hyphenAttached reports whether [start, end) is one half of a hyphenated
// compound, where the elevated word is doing coinage rather than padding:
// "AI-assisted" is not "AI-helped".
func hyphenAttached(text string, start, end int) bool {
	return (start > 0 && text[start-1] == '-') || (end < len(text) && text[end] == '-')
}
