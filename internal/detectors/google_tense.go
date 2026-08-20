package detectors

import (
	"regexp"
	"sort"
	"strings"

	"github.com/yasyf/slop-cop/internal/types"
)

func tensealt(alts []string) string {
	out := make([]string, 0, len(alts))
	for _, a := range alts {
		out = append(out, strings.ReplaceAll(a, " ", phraseSep))
	}
	return strings.Join(out, "|")
}

func tensenorm(s string) string {
	s = strings.ToLower(strings.Join(strings.Fields(s), " "))
	return strings.ReplaceAll(s, "’", "'")
}

func tenseshift(text string, v types.Violation, off int) types.Violation {
	v.StartIndex += off
	v.EndIndex += off
	v.MatchedText = text[v.StartIndex:v.EndIndex]
	return v
}

func tensespan(text, ruleID string, start, end int) types.Violation {
	return types.Violation{
		RuleID:      ruleID,
		StartIndex:  start,
		EndIndex:    end,
		MatchedText: text[start:end],
	}
}

func tenseoverlapsAny(v types.Violation, hits []types.Violation) bool {
	for _, h := range hits {
		if v.StartIndex < h.EndIndex && h.StartIndex < v.EndIndex {
			return true
		}
	}
	return false
}

func tensemergeHits(primary, secondary []types.Violation) []types.Violation {
	out := make([]types.Violation, 0, len(primary)+len(secondary))
	out = append(out, primary...)
	for _, v := range secondary {
		if !tenseoverlapsAny(v, primary) {
			out = append(out, v)
		}
	}
	sort.SliceStable(out, func(i, j int) bool { return out[i].StartIndex < out[j].StartIndex })
	return out
}

var (
	tensechangelogTitleRe = regexp.MustCompile(
		`(?im)^[ \t]{0,3}#{1,6}[ \t]+(?:the[ \t]+)?` +
			`(?:changelog|change[ \t]+log|release[ \t]+notes|release[ \t]+history|` +
			`what['\x{2019}]s[ \t]+new|unreleased|upcoming[ \t]+releases?|roadmap)\b`)
	tensechangelogVersionRe = regexp.MustCompile(`(?im)^[ \t]{0,3}#{1,6}[ \t]+\[?v?\d+\.\d+`)
)

func tenseisChangelog(text string) bool {
	if tensechangelogTitleRe.MatchString(text) {
		return true
	}
	return len(tensechangelogVersionRe.FindAllStringIndex(text, -1)) >= 2
}

var tenseblockQuoteRe = regexp.MustCompile(`^[ \t]{0,3}>`)

func tenseisBlockQuote(line string) bool { return tenseblockQuoteRe.MatchString(line) }

var tensefuturePromiseRe = regexp.MustCompile(`(?i)\b(?:` + tensealt([]string{
	`in (?:a|the) future (?:release|version|update)`,
	`at some point in the future`,
	`in the future`,
	`future release`,
	`upcoming release`,
	`in an upcoming`,
	`coming soon`,
	`available soon`,
	`will (?:be available|be supported|support|ship|land)`,
	`we (?:plan|intend|aim) to`,
	`plans to (?:add|support|ship)`,
	`(?:is|are) planned`,
	`planned for`,
	`on (?:our|the) roadmap`,
	`(?:does|do|is|are|has|have) not yet`,
	`(?:doesn|don|isn|aren|hasn|haven)['\x{2019}]?t yet`,
	`not yet (?:supported|available|implemented)`,
	`(?:supported|available|implemented) yet`,
	`at a later date`,
	`in a later release`,
	`stay tuned`,
	`watch this space`,
	`for now`,
	`eventually`,
	`soon`,
}) + `)\b`)

var (
	tensesoonTailRe = regexp.MustCompile(`(?i)^\s*([a-z]+)(?:\s+([a-z]+))?`)
	tensesoonHeadRe = regexp.MustCompile(`(?i)([a-z]+)\s*$`)
)

func tensesoonIsPromise(text string, start, end int) bool {
	if m := tensesoonHeadRe.FindStringSubmatch(text[:start]); m != nil {
		switch strings.ToLower(m[1]) {
		case "as", "how":
			return false
		}
	}
	m := tensesoonTailRe.FindStringSubmatch(text[end:])
	if m == nil {
		return true
	}
	for _, w := range m[1:] {
		switch strings.ToLower(w) {
		case "after", "as":
			return false
		}
	}
	return true
}

var tensefuturePromiseSuggest = map[string]string{
	"does not yet":        "doesn't",
	"do not yet":          "don't",
	"is not yet":          "isn't",
	"are not yet":         "aren't",
	"has not yet":         "hasn't",
	"have not yet":        "haven't",
	"doesn't yet":         "doesn't",
	"don't yet":           "don't",
	"isn't yet":           "isn't",
	"aren't yet":          "aren't",
	"hasn't yet":          "hasn't",
	"haven't yet":         "haven't",
	"doesnt yet":          "doesn't",
	"dont yet":            "don't",
	"isnt yet":            "isn't",
	"arent yet":           "aren't",
	"hasnt yet":           "hasn't",
	"havent yet":          "haven't",
	"supported yet":       "supported",
	"available yet":       "available",
	"implemented yet":     "implemented",
	"not yet supported":   "not supported",
	"not yet available":   "not available",
	"not yet implemented": "not implemented",
}

// DetectGoogleFutureFeaturePromise reports sentences that commit the project to
// work it has not shipped.
func DetectGoogleFutureFeaturePromise(text string) []types.Violation {
	if tenseisChangelog(text) {
		return nil
	}
	var out []types.Violation
	for _, v := range findAll(text, tensefuturePromiseRe, "future-feature-promise") {
		if strings.EqualFold(v.MatchedText, "soon") &&
			!tensesoonIsPromise(text, v.StartIndex, v.EndIndex) {
			continue
		}
		v.SuggestedChange = tensefuturePromiseSuggest[tensenorm(v.MatchedText)]
		out = append(out, v)
	}
	return out
}

const tensefutureCap = 3

var (
	tensefutureWillRe = regexp.MustCompile(
		`(?i)\b(?:will|shall|won['\x{2019}]?t)\s+` +
			`(?:not\s+|never\s+|then\s+|also\s+|only\s+|still\s+)?[a-z]+\b`)
	tensefutureContractRe = regexp.MustCompile(
		`(?i)\b(?:it|that|this|you|we|they|there)['\x{2019}]ll\s+[a-z]+\b`)
	tensefutureGoingRe = regexp.MustCompile(`(?i)\b(?:is|are|am)\s+going to\s+[a-z]+\b`)
	tensefutureWouldRe = regexp.MustCompile(
		`(?i)\bwould\s+(?:then\s+|also\s+|not\s+|never\s+|no longer\s+)?([a-z]+)\b`)
)

var tensefutureDeferRe = regexp.MustCompile(`(?i)\b(?:` + tensealt([]string{
	`next time`,
	`the next`,
	`after (?:you|the|it|that)`,
	`once (?:you|the|it)`,
	`when (?:you|the|it)`,
	`later`,
	`afterward`,
	`subsequently`,
	`until`,
	`asynchronous(?:ly)?`,
	`in (?:a few|\d+) (?:seconds|minutes|hours|days|weeks)`,
	`within \d+`,
	`by the time`,
	`on the next`,
	`eventually`,
}) + `)\b`)

var tensefutureDateRe = regexp.MustCompile(
	`(?i)\b(?:on|after|starting|beginning|by)\s+` +
		`(?:(?:19|20)\d{2}|january|february|march|april|may|june|july|august|` +
		`september|october|november|december)\b`)

var tensewouldExempt = map[string]bool{
	"like": true, "rather": true, "prefer": true,
	"expect": true, "have": true, "be": true,
}

var tensefutureSuggest = map[string]string{
	"will be":           "is",
	"will return":       "returns",
	"will contain":      "contains",
	"would cause":       "causes",
	"would then remove": "removes",
}

// DetectGoogleFutureTenseBehavior reports future and conditional framing of
// behavior that happens on every invocation.
func DetectGoogleFutureTenseBehavior(text string) []types.Violation {
	var out []types.Violation
	for _, p := range splitParagraphs(text) {
		hits := 0
		off := 0
		for _, s := range mergeAbbrev(splitSentences(p.text)) {
			start := p.start + off
			off += len(s)
			if hits >= tensefutureCap {
				break
			}
			if tensefutureSkip(text, s, start) {
				continue
			}
			for _, v := range tensefutureSentenceHits(s) {
				if hits >= tensefutureCap {
					break
				}
				v = tenseshift(text, v, start)
				v.SuggestedChange = tensefutureSuggest[tensenorm(v.MatchedText)]
				out = append(out, v)
				hits++
			}
		}
	}
	return out
}

func tensefutureSkip(text, sentence string, start int) bool {
	if strings.HasSuffix(strings.TrimRight(sentence, " \t\r\n"), "?") {
		return true
	}
	lines := spanLines(text, start, start+len(sentence))
	if isATXHeading(lines[0]) {
		return true
	}
	for _, l := range lines {
		if tenseisBlockQuote(l) || isTabular(l) {
			return true
		}
	}
	if strings.Contains(strings.ToLower(sentence), "deprecat") {
		return true
	}
	return tensefutureDeferRe.MatchString(sentence) || tensefutureDateRe.MatchString(sentence)
}

func tensefutureSentenceHits(sentence string) []types.Violation {
	var out []types.Violation
	add := func(start, end int) {
		out = append(out, types.Violation{RuleID: "future-tense-behavior", StartIndex: start, EndIndex: end})
	}
	for _, re := range []*regexp.Regexp{tensefutureWillRe, tensefutureContractRe, tensefutureGoingRe} {
		for _, idx := range re.FindAllStringIndex(sentence, -1) {
			add(idx[0], idx[1])
		}
	}
	for _, m := range tensefutureWouldRe.FindAllStringSubmatchIndex(sentence, -1) {
		if tensewouldExempt[strings.ToLower(sentence[m[2]:m[3]])] {
			continue
		}
		add(m[0], m[1])
	}
	sort.SliceStable(out, func(i, j int) bool { return out[i].StartIndex < out[j].StartIndex })
	return out
}

var tenseimpersonalSubs = []plainSub{
	{"it is recommended that you", "we recommend that you"},
	{"it's recommended to", "we recommend"},
	{"it is advisable to", "we recommend"},
	{"users are encouraged to", "we recommend that you"},
	{"one should", "you can"},
	{"best practice is to", "we recommend"},
}

var tenseimpersonalRes = compileSubs(tenseimpersonalSubs)

var tenseimpersonalRe = regexp.MustCompile(`(?i)(?:` + tensealt([]string{
	`\bit(?:['\x{2019}]s|\s+is)\s+(?:strongly\s+|highly\s+|generally\s+)?` +
		`(?:recommended|suggested|advised|advisable|encouraged|considered best practice|a good idea|best)\b`,
	`\b(?:users|developers|customers|readers|you)\s+(?:are|is)\s+(?:encouraged|advised|expected)\s+to\b`,
	`\bone should\b`,
	`\bthe recommended (?:approach|practice|way|option) is\b`,
	`\bbest practice is to\b`,
}) + `)`)

// DetectGoogleImpersonalRecommendation reports advice that names no one behind
// it.
func DetectGoogleImpersonalRecommendation(text string) []types.Violation {
	subs := findSubs(text, "impersonal-recommendation", tenseimpersonalSubs, tenseimpersonalRes)
	return tensemergeHits(subs, findAll(text, tenseimpersonalRe, "impersonal-recommendation"))
}

var tensetimeBoundRe = regexp.MustCompile(`(?i)\b(?:` + tensealt([]string{
	`currently`,
	`presently`,
	`at present`,
	`at this time`,
	`at the moment`,
	`as of (?:this|the) writing`,
	`at the time of (?:this )?writing`,
	`as of today`,
	`as things stand`,
	`for the time being`,
}) + `)\b`)

var (
	tensenowAfterAuxRe   = regexp.MustCompile(`(?i)\b(?:is|are|can|will)(\s+)now\b`)
	tensenowBeforeVerbRe = regexp.MustCompile(`(?i)\bnow(\s+)` +
		`(?:supports?|includes?|provides?|offers?|allows?|lets?|has|have|contains?|returns?|accepts?)\b`)
	tenseversionAnchorRe = regexp.MustCompile(
		`(?i)\b(?:v?\d+\.\d+|version\s+\d|release\s+\d|\d{4}-\d{2}-\d{2})\b`)
)

// DetectGoogleTimeBoundQualifier reports qualifiers that date a sentence. Every
// flagged span is exactly the text to delete, so a "now supports" hit covers
// "now " and an "is now" hit covers " now".
func DetectGoogleTimeBoundQualifier(text string) []types.Violation {
	if tenseisChangelog(text) {
		return nil
	}
	var out []types.Violation
	for _, p := range splitParagraphs(text) {
		off := 0
		for _, s := range mergeAbbrev(splitSentences(p.text)) {
			start := p.start + off
			off += len(s)
			if tenseversionAnchorRe.MatchString(s) {
				continue
			}
			for _, v := range findAll(s, tensetimeBoundRe, "time-bound-qualifier") {
				out = append(out, tenseshift(text, v, start))
			}
			for _, m := range tensenowAfterAuxRe.FindAllStringSubmatchIndex(s, -1) {
				out = append(out, tensespan(text, "time-bound-qualifier", start+m[2], start+m[1]))
			}
			for _, m := range tensenowBeforeVerbRe.FindAllStringSubmatchIndex(s, -1) {
				out = append(out, tensespan(text, "time-bound-qualifier", start+m[0], start+m[3]))
			}
		}
	}
	return out
}
