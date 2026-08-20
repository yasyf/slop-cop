package detectors

import (
	"regexp"
	"sort"
	"strings"

	"github.com/yasyf/slop-cop/internal/types"
)

var media1reInlineSpan = regexp.MustCompile("`[^`\n]*`|\"[^\"\n]*\"|“[^”\n]*”")

func media1maskInline(text string) string {
	return media1reInlineSpan.ReplaceAllStringFunc(text, func(m string) string {
		return strings.Repeat(" ", len(m))
	})
}

func media1lineSpans(text string) [][2]int {
	out := make([][2]int, 0, strings.Count(text, "\n")+1)
	start := 0
	for i := 0; i < len(text); i++ {
		if text[i] != '\n' {
			continue
		}
		end := i
		if end > start && text[end-1] == '\r' {
			end--
		}
		out = append(out, [2]int{start, end})
		start = i + 1
	}
	return append(out, [2]int{start, len(text)})
}

func media1window(text string, from, n int) string {
	if from >= len(text) {
		return ""
	}
	to := from + n
	if to > len(text) {
		to = len(text)
	}
	return text[from:to]
}

func media1dropOverlapping(hits []types.Violation) []types.Violation {
	sort.SliceStable(hits, func(i, j int) bool {
		if hits[i].StartIndex != hits[j].StartIndex {
			return hits[i].StartIndex < hits[j].StartIndex
		}
		return hits[i].EndIndex > hits[j].EndIndex
	})
	kept := make([]types.Violation, 0, len(hits))
	end := -1
	for _, h := range hits {
		if h.StartIndex < end {
			continue
		}
		kept = append(kept, h)
		if h.EndIndex > end {
			end = h.EndIndex
		}
	}
	return kept
}

func media1retext(text string, hits []types.Violation) []types.Violation {
	for i := range hits {
		hits[i].MatchedText = text[hits[i].StartIndex:hits[i].EndIndex]
	}
	return hits
}

type media1notice struct {
	label      string
	rawLabel   string
	sigilStart int
	sigilEnd   int
	bodyStart  int
	blockStart int
	blockEnd   int
	body       string
}

var (
	media1reNoticeLabel = regexp.MustCompile(`(?i)^([ \t]*(?:>[ \t]*)*)([*_]{0,2})(note|caution|warning|important|success|tip|key point)([*_]{0,2})[ \t]*:[*_]{0,2}[ \t]*`)
	media1reAlertLabel  = regexp.MustCompile(`(?i)^[ \t]*(?:>[ \t]*)+\[!(note|caution|warning|important|tip)\][ \t]*$`)
	media1reQuoteMark   = regexp.MustCompile(`(?m)^[ \t]*(?:>[ \t]*)+`)
)

func media1stripQuote(s string) string {
	return media1reQuoteMark.ReplaceAllStringFunc(s, func(m string) string {
		return strings.Repeat(" ", len(m))
	})
}

func media1isNoticeStart(line string) bool {
	return media1reAlertLabel.MatchString(line) || media1reNoticeLabel.MatchString(line)
}

func media1likelyBareNotice(rawLabel, body string) bool {
	if rawLabel == "" || rawLabel[0] < 'A' || rawLabel[0] > 'Z' {
		return false
	}
	return proseWordCount(body) >= 3
}

func media1notices(text string) []media1notice {
	lines := media1lineSpans(text)
	var out []media1notice
	for i := 0; i < len(lines); i++ {
		lineStart, lineEnd := lines[i][0], lines[i][1]
		line := text[lineStart:lineEnd]
		var n media1notice
		bare := false
		if m := media1reAlertLabel.FindStringSubmatchIndex(line); m != nil {
			n.rawLabel = line[m[2]:m[3]]
			n.sigilStart, n.sigilEnd = lineStart, lineEnd
			n.bodyStart = lineEnd
			if n.bodyStart < len(text) {
				n.bodyStart++
			}
		} else if m := media1reNoticeLabel.FindStringSubmatchIndex(line); m != nil {
			n.rawLabel = line[m[6]:m[7]]
			n.sigilStart = lineStart
			n.sigilEnd = lineStart + m[1]
			n.bodyStart = lineStart + m[1]
			bare = m[2] == m[3] && m[4] == m[5] && m[8] == m[9]
		} else {
			continue
		}
		n.label = strings.ToLower(n.rawLabel)
		j := i + 1
		for j < len(lines) {
			next := text[lines[j][0]:lines[j][1]]
			if strings.Trim(next, " \t\r>") == "" || media1isNoticeStart(next) {
				break
			}
			j++
		}
		n.blockStart = lineStart
		n.blockEnd = lines[j-1][1]
		i = j - 1
		if n.bodyStart > n.blockEnd {
			n.bodyStart = n.blockEnd
		}
		n.body = media1stripQuote(text[n.bodyStart:n.blockEnd])
		if bare && !media1likelyBareNotice(n.rawLabel, n.body) {
			continue
		}
		for n.sigilEnd > n.sigilStart && isSpaceByte(text[n.sigilEnd-1]) {
			n.sigilEnd--
		}
		out = append(out, n)
	}
	return out
}

func media1sigilViolation(text string, n media1notice, ruleID, explanation, suggestion string) types.Violation {
	return types.Violation{
		RuleID:          ruleID,
		StartIndex:      n.sigilStart,
		EndIndex:        n.sigilEnd,
		MatchedText:     text[n.sigilStart:n.sigilEnd],
		Explanation:     explanation,
		SuggestedChange: suggestion,
	}
}

func media1blockViolation(text string, n media1notice, ruleID, explanation string) types.Violation {
	start, end := trimSpan(text, n.blockStart, n.blockEnd)
	return types.Violation{
		RuleID:      ruleID,
		StartIndex:  start,
		EndIndex:    end,
		MatchedText: text[start:end],
		Explanation: explanation,
	}
}

var (
	media1reNegation    = regexp.MustCompile(`(?i)\b(?:not|no|never|none|neither|nor|without|unless|cannot|fails? to|failed to|prevents? [a-z]+ from|un(?:able|likely|necessary|available|supported)|[a-z]+n't)\b`)
	media1reNotUn       = regexp.MustCompile(`(?i)\b(?:not|[a-z]+n't)\s+(?:uncommon|unusual|unlikely|unable|unavailable|unsupported|unnecessary|unclear|unimportant|unreasonable|unhelpful|unsafe|unrelated|unknown|invalid|incorrect|incomplete|impossible|improbable|illegal|illogical|irreversible|irrelevant|irregular|non-?trivial|non-?zero|non-?empty)\b`)
	media1reNotExcept   = regexp.MustCompile(`(?i)\b(?:not|[a-z]+n't)\s+[a-z]+\s+(?:without|unless|except)\b`)
	media1reDontForget  = regexp.MustCompile(`(?i)\bdo(?:n't| not)\s+(?:forget|neglect|fail)\s+to\b`)
	media1reClauseSplit = regexp.MustCompile(`[,;:]`)
)

var media1negSubs = []plainSub{
	{"it is not uncommon", "it is common"},
	{"don't forget to", "remember to"},
	{"won't prevent you from", "lets you"},
}

var media1negSubRes = compileSubs(media1negSubs)

// Past this byte gap two negations read as independent clauses, not a stack.
const media1negGap = 20

func media1stackedPair(sentence string) (int, int, bool) {
	found := media1reNegation.FindAllStringIndex(sentence, -1)
	for i := 1; i < len(found); i++ {
		gapStart, gapEnd := found[i-1][1], found[i][0]
		if gapEnd-gapStart > media1negGap || media1reClauseSplit.MatchString(sentence[gapStart:gapEnd]) {
			continue
		}
		return found[i-1][0], found[i][1], true
	}
	return 0, 0, false
}

// DetectGoogleDoubleNegative reports stacked negations the reader has to cancel
// out before they learn what they can do.
func DetectGoogleDoubleNegative(text string) []types.Violation {
	masked := media1maskInline(text)
	narrow := findSubs(masked, "double-negative", media1negSubs, media1negSubRes)
	for _, re := range []*regexp.Regexp{media1reNotUn, media1reNotExcept, media1reDontForget} {
		narrow = append(narrow, findAll(masked, re, "double-negative")...)
	}
	narrow = media1dropOverlapping(narrow)
	out := narrow
	for _, p := range splitParagraphs(masked) {
		offset := p.start
		for _, sentence := range mergeAbbrev(splitSentences(p.text)) {
			base := offset
			offset += len(sentence)
			from, to, ok := media1stackedPair(sentence)
			if !ok {
				continue
			}
			start, end := base+from, base+to
			covered := false
			for _, h := range narrow {
				if h.StartIndex < end && h.EndIndex > start {
					covered = true
					break
				}
			}
			if covered {
				continue
			}
			out = append(out, types.Violation{RuleID: "double-negative", StartIndex: start, EndIndex: end})
		}
	}
	return media1retext(text, media1dropOverlapping(out))
}

var media1idiomSubs = []plainSub{
	{"ballpark figure", "estimate"},
	{"silver bullet", "complete solution"},
	{"in a nutshell", "in summary"},
	{"at the end of the day", "ultimately"},
	{"moving forward", "from now on"},
	{"bells and whistles", "extra features"},
	{"under the hood", "internally"},
	{"rule of thumb", "general guideline"},
	{"on the fly", "dynamically"},
	{"deep dive", "detailed explanation"},
	{"heavy lifting", "the difficult work"},
	{"kick off", "start"},
	{"kicks off", "starts"},
	{"kicked off", "started"},
	{"wrap up", "finish"},
	{"wraps up", "finishes"},
	{"wrapped up", "finished"},
	{"ballpark", ""},
	{"back burner", ""},
	{"hang in there", ""},
	{"piece of cake", ""},
	{"touch base", ""},
	{"apples to apples", ""},
	{"cut corners", ""},
	{"hit the ground running", ""},
	{"game changer", ""},
	{"no-brainer", ""},
	{"break the bank", ""},
	{"down the road", ""},
	{"the last mile", ""},
	{"a home run", ""},
	{"ahead of the curve", ""},
	{"drink from the firehose", ""},
	{"drink from the fire hose", ""},
	{"think outside the box", ""},
	{"thinking outside the box", ""},
	{"red tape", ""},
	{"monday morning quarterback", ""},
	{"hail mary", ""},
	{"slam dunk", ""},
	{"out of left field", ""},
}

var media1idiomRes = compileSubs(media1idiomSubs)

// DetectGoogleIdiomColloquialism reports figurative English standing in for the
// literal statement it decorates.
func DetectGoogleIdiomColloquialism(text string) []types.Violation {
	return media1dropOverlapping(findSubs(text, "idiom-colloquialism", media1idiomSubs, media1idiomRes))
}

var media1multiwordSubs = []plainSub{
	{"in spite of the fact that", "although"},
	{"in the majority of cases", "usually"},
	{"a sufficient amount of", "enough"},
	{"with the exception of", "except for"},
	{"a large number of", "many"},
	{"on a regular basis", "regularly"},
	{"in a timely manner", "quickly"},
	{"for the purpose of", "for"},
	{"has the ability to", "can"},
	{"have the ability to", "can"},
	{"in the near future", "soon"},
	{"the majority of", "most"},
	{"a majority of", "most"},
	{"at a later date", "later"},
	{"a number of", "some"},
	{"a variety of", "several"},
	{"is able to", "can"},
	{"are able to", "can"},
}

var media1multiwordRes = compileSubs(media1multiwordSubs)

// DetectGoogleMultiwordForSingleWord reports stock phrases that expand a single
// word into several without adding meaning.
func DetectGoogleMultiwordForSingleWord(text string) []types.Violation {
	return media1dropOverlapping(findSubs(text, "multiword-for-single-word", media1multiwordSubs, media1multiwordRes))
}

var (
	media1reCrossRefOpener = regexp.MustCompile(`(?i)^\s*(?:for (?:more|further|additional) (?:information|details)|for details|to learn more|learn more|read more|see also|see|check out)\b`)
	media1reLink           = regexp.MustCompile(`\[[^\]\n]+\]\([^)\n]*\)|\[[^\]\n]+\]\[[^\]\n]*\]|<a\s[^>\n]*href=|https?://`)
)

// DetectGoogleNoteAsCrossReference reports a callout whose whole body is a
// pointer to another page.
func DetectGoogleNoteAsCrossReference(text string) []types.Violation {
	var out []types.Violation
	for _, n := range media1notices(text) {
		body := strings.TrimSpace(n.body)
		if body == "" || !media1reLink.MatchString(body) {
			continue
		}
		if len(mergeAbbrev(splitSentences(body))) != 1 {
			continue
		}
		if !media1reCrossRefOpener.MatchString(body) {
			continue
		}
		out = append(out, media1sigilViolation(text, n, "note-as-cross-reference",
			"The callout body is only a cross-reference.", ""))
	}
	return out
}

var (
	media1rePrereqBody     = regexp.MustCompile(`(?i)^\s*(?:before you (?:begin|start|continue|proceed)|you (?:must|need to) (?:first )?\w+|make sure (?:that )?you|ensure (?:that )?you|you should have (?:already )?|first,)`)
	media1reProceduralBody = regexp.MustCompile(`(?i)^\s*(?:click|run|enter|type|select|open|install|create|add|set|copy|paste|navigate to|go to|choose|configure|replace|download|deploy|save|restart|delete)[ \t]+\S`)
)

// DetectGoogleNoteCarriesRequiredInfo reports a callout holding a prerequisite
// or an instruction the reader cannot skip.
func DetectGoogleNoteCarriesRequiredInfo(text string) []types.Violation {
	var out []types.Violation
	for _, n := range media1notices(text) {
		procedural := (n.label == "note" || n.label == "caution") && media1reProceduralBody.MatchString(n.body)
		if !media1rePrereqBody.MatchString(n.body) && !procedural {
			continue
		}
		out = append(out, media1blockViolation(text, n, "note-carries-required-info",
			"The callout carries a prerequisite or a step the reader needs."))
	}
	return out
}

// DetectGoogleNoticePileup reports callouts that crowd each other out.
func DetectGoogleNoticePileup(text string) []types.Violation {
	notices := media1notices(text)
	if len(notices) < 2 {
		return nil
	}
	words := proseWordCount(text)
	var out []types.Violation
	for i, n := range notices {
		flag := i >= 3
		if i >= 2 && words < 400*len(notices) {
			flag = true
		}
		if i > 0 && strings.Trim(text[notices[i-1].blockEnd:n.blockStart], " \t\r\n>") == "" {
			flag = true
		}
		if !flag {
			continue
		}
		out = append(out, media1blockViolation(text, n, "notice-pileup",
			"This callout stacks on the one before it."))
	}
	return out
}

var media1reSevere = regexp.MustCompile(`(?i)\b(?:permanent(?:ly)?|irreversibl[ey]|(?:cannot|can't) be (?:undone|reversed|recovered)|unrecoverable|data loss|lose (?:your |any )?(?:data|work|money|access)|destroy(?:s|ed)? (?:all|every|the)|security (?:risk|breach|vulnerability|hole)|expose(?:s|d)? (?:your )?(?:password|credential|secret|private key|token))\b`)

func media1promoteToWarning(sigil, rawLabel string) string {
	replacement := "Warning"
	if rawLabel == strings.ToUpper(rawLabel) && rawLabel != strings.ToLower(rawLabel) {
		replacement = "WARNING"
	}
	at := strings.Index(strings.ToLower(sigil), strings.ToLower(rawLabel))
	if at < 0 {
		return ""
	}
	return sigil[:at] + replacement + sigil[at+len(rawLabel):]
}

// DetectGoogleNoticeSeverityMismatch reports a Note or Caution describing a
// consequence that warrants a Warning.
func DetectGoogleNoticeSeverityMismatch(text string) []types.Violation {
	var out []types.Violation
	for _, n := range media1notices(text) {
		if n.label != "note" && n.label != "caution" {
			continue
		}
		if !media1reSevere.MatchString(n.body) {
			continue
		}
		sigil := text[n.sigilStart:n.sigilEnd]
		out = append(out, media1sigilViolation(text, n, "notice-severity-mismatch",
			"The consequence described here warrants a Warning.", media1promoteToWarning(sigil, n.rawLabel)))
	}
	return out
}

var media1plainSubs = []plainSub{
	{"commence", "start"},
	{"commences", "starts"},
	{"commenced", "started"},
	{"utilize", "use"},
	{"utilizes", "uses"},
	{"utilized", "used"},
	{"utilizing", "using"},
	{"leverage", "use"},
	{"leverages", "uses"},
	{"leveraged", "used"},
	{"leveraging", "using"},
	{"consequently", "so"},
	{"subsequently", "then"},
	{"endeavor", "try"},
	{"endeavors", "tries"},
	{"endeavored", "tried"},
	{"facilitate", "help"},
	{"facilitates", "helps"},
	{"facilitated", "helped"},
	{"ascertain", "find out"},
	{"ascertains", "finds out"},
	{"ascertained", "found out"},
	{"terminate", "stop"},
	{"terminates", "stops"},
	{"terminated", "stopped"},
	{"desire", "want"},
	{"desires", "wants"},
	{"desired", "that you want"},
	{"desiring", "wanting"},
	{"execute", "run"},
	{"executes", "runs"},
	{"executed", "ran"},
	{"executing", "running"},
	{"functionality", "capabilities"},
	{"down-scope", "reduce the scope of"},
	{"downscope", "reduce the scope of"},
	{"down-scopes", "reduces the scope of"},
	{"downscopes", "reduces the scope of"},
	{"down-scoped", "reduced the scope of"},
	{"downscoped", "reduced the scope of"},
	{"due to the fact that", "because"},
	{"at this point in time", "now"},
	{"in the event that", "if"},
	{"subsequent to", "after"},
	{"prior to", "before"},
	{"in order to", "to"},
	{"sufficient", "enough"},
	{"approximately", "about"},
	{"methodology", "method"},
	{"comprised of", "consists of"},
	{"comprises", "consists of"},
	{"comprise", "consist of"},
	{"via", "by using"},
	{"vice versa", ""},
	{"wish", "want"},
	{"wishes", "wants"},
	{"wished", "wanted"},
	{"learnings", "what you learned"},
	{"enables you to", "lets you"},
	{"enable you to", "let you"},
	{"enabling you to", "letting you"},
	{"allows you to", "lets you"},
	{"allow you to", "let you"},
	{"allowing you to", "letting you"},
}

var media1plainRes = compileSubs(media1plainSubs)

var (
	media1rePermission     = regexp.MustCompile(`(?i)\b(?:enables?|enabling|allows?|allowing)\s+(?:users?|developers?|customers?|callers?)\s+to\b`)
	media1reIE             = regexp.MustCompile(`(?i)\bi\.e\.`)
	media1reVersus         = regexp.MustCompile(`(?i)[A-Za-z0-9)\]]\s+(vs\.?)\s+[A-Za-z0-9(\[]`)
	media1reBareInfinitive = regexp.MustCompile(`(?i)\bto\s+[a-z]+`)
	media1rePrevWord       = regexp.MustCompile(`([A-Za-z]+)[^A-Za-z]*$`)
)

var media1leverageGuard = map[string]bool{
	"financial": true,
	"operating": true,
	"debt":      true,
	"ratio":     true,
	"high":      true,
	"low":       true,
}

func media1prevWord(text string, start int) string {
	low := start - 40
	if low < 0 {
		low = 0
	}
	m := media1rePrevWord.FindStringSubmatch(text[low:start])
	if m == nil {
		return ""
	}
	return strings.ToLower(m[1])
}

// DetectGooglePlainWordSwap reports formal or roundabout wording that has a
// plain everyday equivalent.
func DetectGooglePlainWordSwap(text string) []types.Violation {
	hits := findSubs(text, "plain-word-swap", media1plainSubs, media1plainRes)
	hits = append(hits, findAll(text, media1rePermission, "plain-word-swap")...)
	for _, m := range media1reIE.FindAllStringIndex(text, -1) {
		hits = append(hits, types.Violation{
			RuleID:          "plain-word-swap",
			StartIndex:      m[0],
			EndIndex:        m[1],
			MatchedText:     text[m[0]:m[1]],
			SuggestedChange: "that is",
		})
	}
	for _, m := range media1reVersus.FindAllStringSubmatchIndex(text, -1) {
		hits = append(hits, types.Violation{
			RuleID:          "plain-word-swap",
			StartIndex:      m[2],
			EndIndex:        m[3],
			MatchedText:     text[m[2]:m[3]],
			SuggestedChange: "versus",
		})
	}
	kept := hits[:0]
	for _, h := range hits {
		matched := strings.ToLower(h.MatchedText)
		if strings.HasPrefix(matched, "leverag") && media1leverageGuard[media1prevWord(text, h.StartIndex)] {
			continue
		}
		if strings.HasPrefix(matched, "in order") && media1reBareInfinitive.MatchString(media1window(text, h.EndIndex, 30)) {
			continue
		}
		kept = append(kept, h)
	}
	return media1dropOverlapping(kept)
}

var media1reTemplate = regexp.MustCompile(`\{\{|\{%|\{#|<template\b|<%`)

// DetectGoogleSuccessNoticeInStaticDoc reports a Success callout on a page that
// cannot observe the outcome it announces.
func DetectGoogleSuccessNoticeInStaticDoc(text string) []types.Violation {
	var out []types.Violation
	for _, n := range media1notices(text) {
		if n.label != "success" {
			continue
		}
		low := n.blockStart - 200
		if low < 0 {
			low = 0
		}
		if media1reTemplate.MatchString(text[low:n.blockEnd]) {
			continue
		}
		out = append(out, media1blockViolation(text, n, "success-notice-in-static-doc",
			"A static page cannot confirm what the reader just did."))
	}
	return out
}
