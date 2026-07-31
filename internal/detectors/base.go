package detectors

import (
	"fmt"
	"regexp"
	"sort"
	"strings"
	"unicode"

	"github.com/yasyf/slop-cop/internal/types"
)

// RunBase executes every base-layer (clarity) detector over text and returns a
// merged, deduplicated slice of violations. It is a sibling of [RunClient],
// never a superset: the two layers are selected independently, so
// `--standard=slop` reproduces the pre-base-layer output exactly.
func RunBase(text string) []types.Violation {
	detectors := []func(string) []types.Violation{
		DetectLongSentence,
		DetectLongParagraph,
		DetectPassiveVoice,
		DetectPaddedVerb,
		DetectMissingHyphen,
		DetectExpletiveOpener,
		DetectNominalization,
		DetectPlainWord,
	}
	all := make([]types.Violation, 0, len(detectors))
	for _, d := range detectors {
		all = append(all, d(text)...)
	}
	return Deduplicate(all)
}

const (
	longSentenceWords      = 40
	longParagraphSentences = 6
)

var (
	reParenSpan  = regexp.MustCompile(`\([^)]*\)`)
	reQuotedSpan = regexp.MustCompile(`"[^"]*"|\x{201C}[^\x{201D}]*\x{201D}`)
)

// proseWordCount counts the words of s with every parenthetical and quoted span
// collapsed to a single token, so an aside or a quoted string weighs once
// instead of by its length. It never exceeds len(strings.Fields(s)).
//
// It must NEVER be retrofitted into DetectStaccatoBurst or
// DetectDramaticFragment: both are pinned to plain strings.Fields by the
// upstream parity suite.
func proseWordCount(s string) int {
	s = reParenSpan.ReplaceAllString(s, "x")
	s = reQuotedSpan.ReplaceAllString(s, "x")
	return len(strings.Fields(s))
}

var (
	reATXHeading = regexp.MustCompile(`^[ \t]{0,3}#{1,6}[ \t]`)
	reListMarker = regexp.MustCompile(`(?m)^[ \t]{0,6}(?:[-*+]|\d+[.)])[ \t]+`)
	reAbbrevTail = regexp.MustCompile(`(?:e\.g\.|i\.e\.|Fig\.|\b[A-Za-z]\.)$`)
)

func isATXHeading(line string) bool { return reATXHeading.MatchString(line) }

// isTabular reports whether line is a pipe-table row. goldmark parses those as
// plain paragraphs (no GFM extension), so they reach the detectors as prose.
func isTabular(line string) bool {
	t := strings.TrimSpace(line)
	return strings.HasPrefix(t, "|") && strings.Count(t, "|") >= 2
}

// isListy reports whether s is shaped like a list rather than a paragraph.
func isListy(s string) bool {
	return len(reListMarker.FindAllStringIndex(s, -1)) >= 2
}

// mergeAbbrev rejoins sentences that splitSentences cut at an abbreviation
// ("e.g.", "i.e.", "Fig.", a single initial) instead of at a real terminator.
// A merged group is joined once from its members rather than accumulated
// chunk by chunk, and the abbreviation test reads the last member rather than
// the whole group — a group's tail is its last member's tail. Both matter: a
// document of N merging sentences is O(N) work, where growing a string per
// chunk made it O(N²).
//
// Joining preserves total length, so the running-offset idiom still
// reconstructs byte positions.
func mergeAbbrev(sentences []string) []string {
	out := make([]string, 0, len(sentences))
	start := 0
	for i, s := range sentences {
		if i+1 < len(sentences) && endsWithAbbrev(s) {
			continue
		}
		if i == start {
			out = append(out, s)
		} else {
			out = append(out, strings.Join(sentences[start:i+1], ""))
		}
		start = i + 1
	}
	return out
}

func endsWithAbbrev(sentence string) bool {
	return reAbbrevTail.MatchString(strings.TrimRight(sentence, " \t\r\n"))
}

// spanLines returns the source lines that [start, end) touches.
func spanLines(text string, start, end int) []string {
	lineStart := strings.LastIndexByte(text[:start], '\n') + 1
	lineEnd := len(text)
	if p := strings.IndexByte(text[end:], '\n'); p >= 0 {
		lineEnd = end + p
	}
	return strings.Split(text[lineStart:lineEnd], "\n")
}

// trimSpan narrows [start, end) to the non-whitespace text it contains.
func trimSpan(text string, start, end int) (int, int) {
	for start < end && isSpaceByte(text[start]) {
		start++
	}
	for end > start && isSpaceByte(text[end-1]) {
		end--
	}
	return start, end
}

func isSpaceByte(b byte) bool {
	return b == ' ' || b == '\t' || b == '\n' || b == '\r'
}

// DetectLongSentence reports sentences past the base layer's word budget.
func DetectLongSentence(text string) []types.Violation {
	var out []types.Violation
	for _, p := range splitParagraphs(text) {
		sentences := mergeAbbrev(splitSentences(p.text))
		off := 0
		for _, s := range sentences {
			start := p.start + off
			end := start + len(s)
			off += len(s)
			words := proseWordCount(s)
			if words <= longSentenceWords {
				continue
			}
			lines := spanLines(text, start, end)
			if isATXHeading(lines[0]) || anyTabular(lines) || isListy(s) {
				continue
			}
			ts, te := trimSpan(text, start, end)
			out = append(out, types.Violation{
				RuleID:      "long-sentence",
				StartIndex:  ts,
				EndIndex:    te,
				MatchedText: text[ts:te],
				Explanation: itoa(words) + " words",
			})
		}
	}
	return out
}

// DetectLongParagraph reports paragraphs that run past the base layer's
// sentence budget.
func DetectLongParagraph(text string) []types.Violation {
	var out []types.Violation
	for _, p := range splitParagraphs(text) {
		lines := strings.Split(p.text, "\n")
		if isATXHeading(lines[0]) || anyTabular(lines) || isListy(p.text) {
			continue
		}
		n := len(mergeAbbrev(splitSentences(p.text)))
		if n <= longParagraphSentences {
			continue
		}
		ts, te := trimSpan(text, p.start, p.start+len(p.text))
		out = append(out, types.Violation{
			RuleID:      "long-paragraph",
			StartIndex:  ts,
			EndIndex:    te,
			MatchedText: text[ts:te],
			Explanation: itoa(n) + " sentences",
		})
	}
	return out
}

func anyTabular(lines []string) bool {
	for _, l := range lines {
		if isTabular(l) {
			return true
		}
	}
	return false
}

var rePassive = regexp.MustCompile(`(?i)\b(?:am|is|are|was|were|be|been|being)\s+` +
	`(?:(?:not|never|always|often|very|also|still|just|\w+ly)\s+){0,2}(?:\w+ed|` +
	strings.Join(irregularParticiples, "|") + `)\s+by\s+([\w'\x{2019}-]+)(?:\s+([\w'\x{2019}-]+))?`)

// DetectPassiveVoice reports passives that name their agent with `by`. The
// agentless passive is the LLM tier's job: RE2 plus a word list cannot tell
// "is deprecated" from "is stored".
func DetectPassiveVoice(text string) []types.Violation {
	var out []types.Violation
	for _, m := range rePassive.FindAllStringSubmatchIndex(text, -1) {
		head := strings.ToLower(text[m[2]:m[3]])
		if byDeterminers[head] && m[4] >= 0 {
			head = strings.ToLower(text[m[4]:m[5]])
		}
		if byNonAgentive[head] || strings.HasSuffix(head, "ing") {
			continue
		}
		out = append(out, types.Violation{
			RuleID:      "passive-voice",
			StartIndex:  m[0],
			EndIndex:    m[1],
			MatchedText: text[m[0]:m[1]],
		})
	}
	return out
}

var paddedVerbRes = compileSubs(paddedVerbSubs)

// DetectPaddedVerb reports verb phrases padded out with an auxiliary frame.
func DetectPaddedVerb(text string) []types.Violation {
	return findSubs(text, "padded-verb", paddedVerbSubs, paddedVerbRes)
}

var nominalizationRes = compileSubs(nominalizationSubs)

// DetectNominalization reports light-verb frames that bury a verb in a noun.
func DetectNominalization(text string) []types.Violation {
	return findSubs(text, "nominalization", nominalizationSubs, nominalizationRes)
}

// findSubs emits one violation per match of each precompiled substitution,
// carrying the replacement as the suggested change.
func findSubs(text, ruleID string, subs []plainSub, res []*regexp.Regexp) []types.Violation {
	var out []types.Violation
	for i, re := range res {
		for _, idx := range re.FindAllStringIndex(text, -1) {
			out = append(out, types.Violation{
				RuleID:          ruleID,
				StartIndex:      idx[0],
				EndIndex:        idx[1],
				MatchedText:     text[idx[0]:idx[1]],
				SuggestedChange: subs[i].to,
			})
		}
	}
	return out
}

// determinerAlt is the leading alternation both compound patterns share,
// built from byDeterminers so the two lists cannot drift apart. Sorted for a
// stable pattern, which map iteration would not give.
var determinerAlt = func() string {
	alts := make([]string, 0, len(byDeterminers))
	for d := range byDeterminers {
		alts = append(alts, d)
	}
	sort.Strings(alts)
	return `(?:` + strings.Join(alts, "|") + `)`
}()

// An open compound needs the noun it modifies to follow, so its trailing word
// is required; a phrasal-verb noun stands on its own ("complete the check
// in."), so its trailing word is optional.
var (
	hyphenCompoundRes     = compileCompounds(hyphenCompounds, `%s`)
	hyphenNounCompoundRes = compileCompounds(hyphenNounCompounds, `(?:%s)?`)
	hyphenLocativeNounRes = compileCompounds(hyphenLocativeNouns, `(?:%s)?`)
)

func compileCompounds(compounds []string, tail string) []*regexp.Regexp {
	out := make([]*regexp.Regexp, 0, len(compounds))
	for _, c := range compounds {
		follow := fmt.Sprintf(tail, phraseSep+`([\w'\x{2019}-]+)`)
		out = append(out, regexp.MustCompile(`(?i)\b`+determinerAlt+phraseSep+
			`(`+strings.ReplaceAll(escapeForRegex(c), " ", phraseSep)+`)`+follow))
	}
	return out
}

// DetectMissingHyphen reports a compound that needs a hyphen.
//
// The three compound classes are read differently. An open compound
// ("open source", "command line") takes a hyphen only when it modifies a
// noun, so a following verb or preposition means predicate position and no
// hyphen: "the tool is open source" stands. A phrasal-verb noun ("roll out",
// "follow up") is already a noun once a determiner introduces it, and nothing
// following can undo that, so it fires on whatever comes next — including the
// reduced relative clause in "the roll out your team scheduled". A locative
// one ("check in", "work around") reads either way, so an object after it
// means the pair is a noun and a preposition: "deposit the check in the bank."
func DetectMissingHyphen(text string) []types.Violation {
	out := findCompounds(text, hyphenCompounds, hyphenCompoundRes, hyphenFollowStop)
	out = append(out, findCompounds(text, hyphenNounCompounds, hyphenNounCompoundRes, nil)...)
	return append(out, findCompounds(text, hyphenLocativeNouns, hyphenLocativeNounRes, hyphenObjectStop)...)
}

func findCompounds(text string, compounds []string, res []*regexp.Regexp, stop map[string]bool) []types.Violation {
	var out []types.Violation
	for i, re := range res {
		for _, m := range re.FindAllStringSubmatchIndex(text, -1) {
			if m[4] >= 0 && stop[strings.ToLower(text[m[4]:m[5]])] {
				continue
			}
			compound := text[m[2]:m[3]]
			if isTitleCase(compound) {
				continue
			}
			out = append(out, types.Violation{
				RuleID:          "missing-hyphen",
				StartIndex:      m[2],
				EndIndex:        m[3],
				MatchedText:     compound,
				SuggestedChange: strings.ReplaceAll(compounds[i], " ", "-"),
			})
		}
	}
	return out
}

// isTitleCase reports whether every word of the compound is capitalised, which
// marks a proper noun rather than a compound modifier. Apple spells its
// product "Command Line Tools" deliberately, and hyphenating it would be wrong
// rather than merely unwanted. A compound always follows a determiner here, so
// it is never sentence-initial and capitals carry their usual meaning.
func isTitleCase(compound string) bool {
	for _, word := range strings.Fields(compound) {
		r, _ := firstRune(word)
		if !unicode.IsUpper(r) {
			return false
		}
	}
	return true
}

var (
	reStructuralMarker = regexp.MustCompile(`^[ \t]*(?:(?:>|#{1,6}|[-*+]|\d+[.)])[ \t]+)*`)
	reExpletiveOpener  = regexp.MustCompile(`(?i)^there\s+(?:is|are|was|were)\s+(\S+)`)
	reEmphasisEdge     = regexp.MustCompile(`^[*_]+|[*_]+$`)
)

// DetectExpletiveOpener reports a sentence that opens on "there is/are/was/
// were" and so delays its subject. Existential negations ("There are no
// guarantees.") and colon-terminated list openers ("There are three modes:")
// are carved out; a colon-enumerating opener that continues on the same line
// ("There are three modes: fast, slow, off.") still fires.
func DetectExpletiveOpener(text string) []types.Violation {
	var out []types.Violation
	for _, p := range splitParagraphs(text) {
		off := 0
		for _, s := range splitSentences(p.text) {
			start := p.start + off
			off += len(s)
			marker := len(reStructuralMarker.FindString(s))
			m := reExpletiveOpener.FindStringSubmatchIndex(s[marker:])
			if m == nil {
				continue
			}
			word := strings.ToLower(reEmphasisEdge.ReplaceAllString(s[marker+m[2]:marker+m[3]], ""))
			if expletiveNegations[strings.Trim(word, ".,;:!?")] {
				continue
			}
			if endsLineWithColon(text, start+marker) {
				continue
			}
			out = append(out, types.Violation{
				RuleID:      "expletive-opener",
				StartIndex:  start + marker + m[0],
				EndIndex:    start + marker + m[1],
				MatchedText: text[start+marker+m[0] : start+marker+m[1]],
			})
		}
	}
	return out
}

func endsLineWithColon(text string, pos int) bool {
	end := len(text)
	if p := strings.IndexByte(text[pos:], '\n'); p >= 0 {
		end = pos + p
	}
	return strings.HasSuffix(strings.TrimRight(text[pos:end], " \t\r"), ":")
}
