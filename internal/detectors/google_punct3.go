package detectors

import (
	"regexp"
	"strings"
	"unicode/utf8"

	"github.com/yasyf/slop-cop/internal/types"
)

func punct3lower(token string) string {
	return strings.ToLower(strings.Trim(token, `.,;:!?()[]"'`+"`"))
}

func punct3lineAt(text string, idx int) string {
	start := strings.LastIndexByte(text[:idx], '\n') + 1
	end := len(text)
	if p := strings.IndexByte(text[start:], '\n'); p >= 0 {
		end = start + p
	}
	return text[start:end]
}

var (
	punct3reQuoteTrailPunct = regexp.MustCompile(`[A-Za-z0-9)]["\x{201D}]\s*[.,]`)
	punct3reQuoteAfterMark  = regexp.MustCompile(`[?!]["\x{201D}]\s*\.`)
	punct3reLiteralToken    = regexp.MustCompile(`^[A-Za-z0-9_./\-]+$`)
	punct3reAllCapsToken    = regexp.MustCompile(`^[A-Z0-9_.\-]*[A-Z][A-Z0-9_.\-]*$`)
)

func punct3quotedInner(text string, closeAt int, closing rune) (int, string, bool) {
	opener := `"`
	if closing != '"' {
		opener = "“"
	}
	i := strings.LastIndex(text[:closeAt], opener)
	if i < 0 || strings.Contains(text[i:closeAt], "\n") {
		return 0, "", false
	}
	return i, text[i+len(opener) : closeAt], true
}

func punct3prevNonSpace(text string, i int) byte {
	for i > 0 {
		b := text[i-1]
		if b != ' ' && b != '\t' {
			return b
		}
		i--
	}
	return 0
}

func punct3nextNonSpace(text string, i int) byte {
	for i < len(text) {
		b := text[i]
		if b != ' ' && b != '\t' {
			return b
		}
		i++
	}
	return 0
}

func punct3structuralQuote(text string, openAt, after int) bool {
	switch punct3prevNonSpace(text, openAt) {
	case '{', '[', '=':
		return true
	}
	if text[after-1] != ',' {
		return false
	}
	switch punct3nextNonSpace(text, after) {
	case 0, '\n', '"', '}', ']':
		return true
	}
	return false
}

func punct3isLiteralQuote(inner string) bool {
	if len(inner) >= 2 && strings.HasPrefix(inner, "`") && strings.HasSuffix(inner, "`") {
		return true
	}
	if inner == "" || strings.ContainsAny(inner, " \t") {
		return false
	}
	if punct3reAllCapsToken.MatchString(inner) {
		return true
	}
	return punct3reLiteralToken.MatchString(inner) && strings.ContainsAny(inner, "_/.-")
}

// DetectGoogleQuotePeriodPlacement reports a sentence-final comma or period
// stranded outside the closing quotation mark, and a period piled onto a
// quoted question or exclamation.
func DetectGoogleQuotePeriodPlacement(text string) []types.Violation {
	var out []types.Violation
	for _, m := range punct3reQuoteTrailPunct.FindAllStringIndex(text, -1) {
		quoteAt := m[0] + 1
		q, qw := utf8.DecodeRuneInString(text[quoteAt:])
		openAt, inner, ok := punct3quotedInner(text, quoteAt, q)
		if !ok || punct3isLiteralQuote(inner) || punct3structuralQuote(text, openAt, m[1]) {
			continue
		}
		out = append(out, types.Violation{
			RuleID:          "quote-period-placement",
			StartIndex:      m[0],
			EndIndex:        m[1],
			MatchedText:     text[m[0]:m[1]],
			SuggestedChange: text[m[0]:quoteAt] + text[m[1]-1:m[1]] + text[quoteAt:quoteAt+qw],
		})
	}
	for _, m := range punct3reQuoteAfterMark.FindAllStringIndex(text, -1) {
		quoteAt := m[0] + 1
		_, qw := utf8.DecodeRuneInString(text[quoteAt:])
		out = append(out, types.Violation{
			RuleID:          "quote-period-placement",
			StartIndex:      m[0],
			EndIndex:        m[1],
			MatchedText:     text[m[0]:m[1]],
			SuggestedChange: text[m[0] : quoteAt+qw],
		})
	}
	return out
}

var (
	punct3reQuotedCodeSpan = regexp.MustCompile("[\"\\x{201C}']`[^`\\n\\s](?:[^`\\n]*[^`\\n\\s])?`[\"\\x{201D}']")
	punct3reQuotedCodeTag  = regexp.MustCompile(`["\x{201C}']<code>[^<\n]*</code>["\x{201D}']`)
	punct3reQuotedTerm     = regexp.MustCompile(`(?i)\bthe\s+"[A-Za-z_][\w.]*"\s+(?:method|function|class|field|parameter|flag|command|attribute)\b`)
)

func punct3stripOuterQuotes(s string) string {
	_, first := utf8.DecodeRuneInString(s)
	_, last := utf8.DecodeLastRuneInString(s)
	return s[first : len(s)-last]
}

// DetectGoogleQuotesAroundCodeSpan reports quotation marks wrapped around a
// code span, and a quoted identifier that should have been code font.
func DetectGoogleQuotesAroundCodeSpan(text string) []types.Violation {
	var out []types.Violation
	for _, re := range []*regexp.Regexp{punct3reQuotedCodeSpan, punct3reQuotedCodeTag} {
		for _, m := range re.FindAllStringIndex(text, -1) {
			out = append(out, types.Violation{
				RuleID:          "quotes-around-code-span",
				StartIndex:      m[0],
				EndIndex:        m[1],
				MatchedText:     text[m[0]:m[1]],
				SuggestedChange: punct3stripOuterQuotes(text[m[0]:m[1]]),
			})
		}
	}
	for _, m := range punct3reQuotedTerm.FindAllStringIndex(text, -1) {
		matched := text[m[0]:m[1]]
		out = append(out, types.Violation{
			RuleID:          "quotes-around-code-span",
			StartIndex:      m[0],
			EndIndex:        m[1],
			MatchedText:     matched,
			SuggestedChange: strings.ReplaceAll(matched, `"`, "`"),
		})
	}
	return out
}

var punct3reRangeWordDash = regexp.MustCompile(
	`(?i)\b(from|between)\s+([0-9]+(?:[.,][0-9]+)*)\s*[-\x{2013}\x{2014}]\s*([0-9]+(?:[.,][0-9]+)*)`)

func punct3isYearMonth(op1, op2 string) bool {
	return len(op1) == 4 && len(op2) == 2 &&
		!strings.ContainsAny(op1, ".,") && !strings.ContainsAny(op2, ".,")
}

// DetectGoogleRangeWordDashMix reports a numeric range that opens with `from`
// or `between` and then closes with a dash instead of `to` or `and`.
func DetectGoogleRangeWordDashMix(text string) []types.Violation {
	var out []types.Violation
	for _, m := range punct3reRangeWordDash.FindAllStringSubmatchIndex(text, -1) {
		if m[1] < len(text) {
			if r, _ := utf8.DecodeRuneInString(text[m[1]:]); r == '-' || r == '–' || r == '—' {
				continue
			}
		}
		intro, op1, op2 := text[m[2]:m[3]], text[m[4]:m[5]], text[m[6]:m[7]]
		if punct3isYearMonth(op1, op2) {
			continue
		}
		partner := " to "
		if strings.EqualFold(intro, "between") {
			partner = " and "
		}
		out = append(out, types.Violation{
			RuleID:          "range-word-dash-mix",
			StartIndex:      m[0],
			EndIndex:        m[1],
			MatchedText:     text[m[0]:m[1]],
			SuggestedChange: intro + " " + op1 + partner + op2,
		})
	}
	return out
}

var punct3reCommaSeries = regexp.MustCompile(
	`,\s+([A-Za-z][\w'\x{2019}-]*(?:\s+[A-Za-z][\w'\x{2019}-]*){0,3})\s+(and|or)\s+([A-Za-z][\w'\x{2019}-]*)`)

var punct3reSemiSeries = regexp.MustCompile(`[A-Za-z0-9)\]]\s+(and|or)\s+([A-Za-z][\w'\x{2019}-]*)`)

var punct3serialStop = map[string]bool{
	"is": true, "are": true, "was": true, "were": true, "be": true, "been": true,
	"can": true, "could": true, "will": true, "would": true, "should": true,
	"must": true, "may": true, "might": true, "has": true, "have": true, "had": true,
	"do": true, "does": true, "did": true, "you": true, "we": true, "it": true,
	"they": true, "this": true, "that": true, "there": true,
}

var punct3serialAfter = map[string]bool{
	"then": true, "you": true, "i": true, "we": true, "they": true, "it": true,
	"he": true, "she": true, "this": true, "that": true, "there": true,
	"so": true, "if": true, "when": true,
}

var punct3serialLead = map[string]bool{
	"and": true, "or": true, "but": true, "nor": true, "so": true, "yet": true,
	"then": true, "thus": true, "however": true, "therefore": true, "instead": true,
	"otherwise": true, "because": true, "which": true, "who": true, "where": true,
	"while": true, "although": true, "though": true, "if": true, "when": true,
	"after": true, "before": true, "since": true, "unless": true, "until": true,
}

var punct3serialOpener = map[string]bool{
	"after": true, "when": true, "whenever": true, "if": true, "while": true,
	"before": true, "once": true, "unless": true, "until": true, "although": true,
	"though": true, "because": true, "since": true, "as": true, "first": true,
	"next": true, "then": true, "finally": true, "however": true, "instead": true,
	"otherwise": true, "therefore": true, "meanwhile": true, "in": true, "on": true,
	"at": true, "to": true, "with": true, "by": true, "for": true, "during": true,
	"without": true, "given": true, "despite": true, "rather": true, "from": true,
	"under": true, "within": true, "through": true,
}

var punct3whHead = map[string]bool{
	"what": true, "who": true, "whom": true, "whose": true, "which": true,
	"when": true, "where": true, "how": true, "why": true, "whether": true,
}

func punct3commaSeries(s string, base int) []types.Violation {
	var out []types.Violation
	firstComma := strings.IndexByte(s, ',')
	for _, m := range punct3reCommaSeries.FindAllStringSubmatchIndex(s, -1) {
		fields := strings.Fields(s[m[2]:m[3]])
		if len(fields) == 0 || punct3serialLead[punct3lower(fields[0])] {
			continue
		}
		stopped := false
		for _, f := range fields {
			if punct3serialStop[punct3lower(f)] {
				stopped = true
				break
			}
		}
		if stopped || punct3serialAfter[punct3lower(s[m[6]:m[7]])] {
			continue
		}
		if strings.ContainsRune(s[m[7]:], ',') {
			continue
		}
		runStart := 0
		if pc := strings.LastIndexByte(s[:m[0]], ','); pc >= 0 {
			runStart = pc + 1
		}
		if proseWordCount(s[runStart:m[0]]) > 5 {
			continue
		}
		if m[0] == firstComma {
			lead := strings.Fields(s[:m[0]])
			if len(lead) <= 3 || punct3serialOpener[punct3lower(lead[0])] {
				continue
			}
		}
		out = append(out, types.Violation{
			RuleID:          "serial-comma",
			StartIndex:      base + m[2],
			EndIndex:        base + m[5],
			MatchedText:     s[m[2]:m[5]],
			SuggestedChange: s[m[2]:m[3]] + "," + s[m[3]:m[5]],
		})
	}
	return out
}

func punct3semiSeries(s string, base int) []types.Violation {
	last := strings.LastIndexByte(s, ';')
	if last < 0 {
		return nil
	}
	run := s[last+1:]
	m := punct3reSemiSeries.FindStringSubmatchIndex(run)
	if m == nil {
		return nil
	}
	head := ""
	if f := strings.Fields(run); len(f) > 0 {
		head = punct3lower(f[0])
	}
	if !punct3whHead[head] || head != punct3lower(run[m[4]:m[5]]) {
		return nil
	}
	start, end := last+1+m[0], last+1+m[3]
	_, w := utf8.DecodeRuneInString(s[start:])
	return []types.Violation{{
		RuleID:          "serial-comma",
		StartIndex:      base + start,
		EndIndex:        base + end,
		MatchedText:     s[start:end],
		SuggestedChange: s[start:start+w] + ";" + s[start+w:end],
	}}
}

// DetectGoogleSerialComma reports a series of three or more items whose final
// item arrives with no separator before its `and` or `or`.
func DetectGoogleSerialComma(text string) []types.Violation {
	var out []types.Violation
	for _, p := range splitParagraphs(text) {
		off := 0
		for _, s := range mergeAbbrev(splitSentences(p.text)) {
			base := p.start + off
			off += len(s)
			lines := spanLines(text, base, base+len(s))
			if isATXHeading(lines[0]) || anyTabular(lines) || isListy(s) {
				continue
			}
			out = append(out, punct3commaSeries(s, base)...)
			out = append(out, punct3semiSeries(s, base)...)
		}
	}
	return out
}

var (
	punct3reSingleQuoted = regexp.MustCompile(`(?m)(?:^|[\s(\[])'([A-Za-z][^'\n]{1,80})'(?:[\s.,;:!?)\]]|$)`)
	punct3reCodeSpan     = regexp.MustCompile("`[^`\\n]*`")
	punct3reDoubleQuoted = regexp.MustCompile(`"[^"\n]*"|\x{201C}[^\x{201D}\n]*\x{201D}`)
)

func punct3codeCallSite(text string, openAt int) bool {
	if openAt > 1 && text[openAt-1] == '(' {
		b := text[openAt-2]
		if b == '_' || b == '.' || ('a' <= b && b <= 'z') || ('A' <= b && b <= 'Z') || ('0' <= b && b <= '9') {
			return true
		}
	}
	switch punct3prevNonSpace(text, openAt) {
	case ':', '=':
		return true
	}
	return false
}

func punct3within(ranges [][]int, start, end int) bool {
	for _, r := range ranges {
		if r[0] <= start && end <= r[1] {
			return true
		}
	}
	return false
}

func punct3strictlyWithin(ranges [][]int, start, end int) bool {
	for _, r := range ranges {
		if r[0] < start && end <= r[1] {
			return true
		}
	}
	return false
}

// DetectGoogleSingleQuotesInProse reports single quotation marks used for a
// top-level quotation, where double marks belong.
func DetectGoogleSingleQuotesInProse(text string) []types.Violation {
	matches := punct3reSingleQuoted.FindAllStringSubmatchIndex(text, -1)
	if len(matches) == 0 {
		return nil
	}
	code := punct3reCodeSpan.FindAllStringIndex(text, -1)
	quoted := punct3reDoubleQuoted.FindAllStringIndex(text, -1)
	var out []types.Violation
	for _, m := range matches {
		inner := text[m[2]:m[3]]
		if strings.ContainsAny(inner, "={}<$/:") {
			continue
		}
		start, end := m[2]-1, m[3]+1
		if punct3codeCallSite(text, start) {
			continue
		}
		if punct3within(code, start, end) || punct3strictlyWithin(quoted, start, end) {
			continue
		}
		out = append(out, types.Violation{
			RuleID:          "single-quotes-in-prose",
			StartIndex:      start,
			EndIndex:        end,
			MatchedText:     text[start:end],
			SuggestedChange: `"` + inner + `"`,
		})
	}
	return out
}

var (
	punct3reSpacedBoth  = regexp.MustCompile(`[A-Za-z0-9,)]\x20+-\x20+[A-Za-z0-9(]`)
	punct3reSpaceAfter  = regexp.MustCompile(`\b[A-Za-z]{2,}-\x20`)
	punct3reDelimLine   = regexp.MustCompile(`^[-\s|:+]+$`)
	punct3reHyphenatedW = regexp.MustCompile(`[A-Za-z]-[A-Za-z]`)
)

func punct3frontMatterEnd(text string) int {
	if !strings.HasPrefix(text, "---\n") {
		return 0
	}
	rest := text[4:]
	for off := 0; off < len(rest); {
		nl := strings.IndexByte(rest[off:], '\n')
		line := rest[off:]
		if nl >= 0 {
			line = rest[off : off+nl]
		}
		if strings.TrimRight(line, " \t\r") == "---" {
			if nl < 0 {
				return len(text)
			}
			return 4 + off + nl + 1
		}
		if nl < 0 {
			break
		}
		off += nl + 1
	}
	return 0
}

func punct3suspendedHyphen(rest string) bool {
	if i := strings.IndexByte(rest, '\n'); i >= 0 {
		rest = rest[:i]
	}
	toks := strings.Fields(rest)
	if len(toks) > 4 {
		toks = toks[:4]
	}
	conj := false
	for i, t := range toks {
		if i >= 3 {
			break
		}
		switch punct3lower(t) {
		case "or", "and", "to":
			conj = true
		}
	}
	if !conj {
		return false
	}
	for _, t := range toks {
		if punct3reHyphenatedW.MatchString(t) {
			return true
		}
	}
	return false
}

// DetectGoogleSpacedHyphen reports whitespace around a hyphen, which reads as
// a dash the sentence never asked for.
func DetectGoogleSpacedHyphen(text string) []types.Violation {
	skipBefore := punct3frontMatterEnd(text)
	var out []types.Violation
	for _, m := range punct3reSpacedBoth.FindAllStringIndex(text, -1) {
		if m[0] < skipBefore {
			continue
		}
		if punct3isDigit(text[m[0]]) && punct3isDigit(text[m[1]-1]) {
			continue
		}
		line := punct3lineAt(text, m[0])
		if isTabular(line) || punct3reDelimLine.MatchString(strings.TrimSpace(line)) {
			continue
		}
		if reListMarker.MatchString(line) {
			lineStart := strings.LastIndexByte(text[:m[0]], '\n') + 1
			if first := punct3reSpacedBoth.FindStringIndex(line); first != nil && lineStart+first[0] == m[0] {
				continue
			}
		}
		out = append(out, types.Violation{
			RuleID:      "spaced-hyphen",
			StartIndex:  m[0] + 1,
			EndIndex:    m[1] - 1,
			MatchedText: text[m[0]+1 : m[1]-1],
		})
	}
	for _, m := range punct3reSpaceAfter.FindAllStringIndex(text, -1) {
		if m[0] < skipBefore || punct3suspendedHyphen(text[m[1]:]) {
			continue
		}
		line := punct3lineAt(text, m[0])
		if isTabular(line) || punct3reDelimLine.MatchString(strings.TrimSpace(line)) {
			continue
		}
		out = append(out, types.Violation{
			RuleID:          "spaced-hyphen",
			StartIndex:      m[0],
			EndIndex:        m[1],
			MatchedText:     text[m[0]:m[1]],
			SuggestedChange: text[m[0] : m[1]-1],
		})
	}
	return out
}

func punct3isDigit(b byte) bool { return b >= '0' && b <= '9' }

var (
	punct3reUILabelVerb = regexp.MustCompile(
		"\\b(?i:click|select|choose|tap|press|open)\\s+(?:\\*\\*|__|`)?" +
			"[A-Z][A-Za-z0-9]*(?:[ ][A-Za-z0-9]+){0,2}(?:\\.\\.\\.|\\x{2026})")
	punct3reUILabelBold = regexp.MustCompile(`(?:\*\*|__)[A-Z][^*_\n]{0,32}(?:\.\.\.|\x{2026})(?:\*\*|__)`)
)

func punct3ellipsisSpan(m string) (int, int) {
	dots := strings.LastIndex(m, "...")
	uni := strings.LastIndex(m, "…")
	if uni > dots {
		return uni, uni + len("…")
	}
	if dots >= 0 {
		return dots, dots + 3
	}
	return -1, -1
}

// DetectGoogleUILabelEllipsis reports a UI label quoted with the trailing
// ellipsis the interface adds.
func DetectGoogleUILabelEllipsis(text string) []types.Violation {
	var out []types.Violation
	seen := map[int]bool{}
	for _, re := range []*regexp.Regexp{punct3reUILabelVerb, punct3reUILabelBold} {
		for _, m := range re.FindAllStringIndex(text, -1) {
			lo, hi := punct3ellipsisSpan(text[m[0]:m[1]])
			if lo < 0 {
				continue
			}
			start, end := m[0]+lo, m[0]+hi
			if seen[start] {
				continue
			}
			seen[start] = true
			out = append(out, types.Violation{
				RuleID:      "ui-label-ellipsis",
				StartIndex:  start,
				EndIndex:    end,
				MatchedText: text[start:end],
			})
		}
	}
	return out
}

var (
	punct3reURLPeriod = regexp.MustCompile(`(?:https?://|ftp://|www\.)[^\s<>()\[\]"']*[A-Za-z0-9/]\.(?:\s|$)`)
	punct3reURLGapDot = regexp.MustCompile(`(?:https?://|www\.)[^\s]+[ ]+\.`)
)

// DetectGoogleURLTerminalPeriod reports a bare URL left at the end of a
// sentence, where the period reads as part of the address.
func DetectGoogleURLTerminalPeriod(text string) []types.Violation {
	var out []types.Violation
	for _, re := range []*regexp.Regexp{punct3reURLPeriod, punct3reURLGapDot} {
		for _, m := range re.FindAllStringIndex(text, -1) {
			s, e := trimSpan(text, m[0], m[1])
			out = append(out, types.Violation{
				RuleID:      "url-terminal-period",
				StartIndex:  s,
				EndIndex:    e,
				MatchedText: text[s:e],
			})
		}
	}
	return out
}
