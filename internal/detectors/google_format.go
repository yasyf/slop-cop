package detectors

import (
	"regexp"
	"strings"

	"github.com/yasyf/slop-cop/internal/types"
)

type formatspan struct{ start, end int }

var (
	formatreFencedCode = regexp.MustCompile("(?is)```.*?```|~~~.*?~~~|<pre\\b.*?</pre>|<code\\b.*?</code>")
	formatreInlineCode = regexp.MustCompile("`[^`\n]*`")
)

func formatcodeSpans(text string) []formatspan {
	out := make([]formatspan, 0, 8)
	for _, m := range formatreFencedCode.FindAllStringIndex(text, -1) {
		out = append(out, formatspan{m[0], m[1]})
	}
	for _, m := range formatreInlineCode.FindAllStringIndex(text, -1) {
		out = append(out, formatspan{m[0], m[1]})
	}
	return out
}

func formatcovers(spans []formatspan, i int) bool {
	for _, s := range spans {
		if i >= s.start && i < s.end {
			return true
		}
	}
	return false
}

var formatproperNouns = map[string]bool{
	"African": true, "Amazon": true, "American": true, "Android": true, "Apache": true,
	"Apple": true, "April": true, "Arm": true, "Asian": true, "August": true,
	"Australian": true, "Bayesian": true, "Benz": true, "Boolean": true, "British": true,
	"Canadian": true, "Chinese": true, "Chrome": true, "Cloud": true, "Cola": true,
	"December": true, "Debian": true, "Django": true, "Docker": true, "Dutch": true,
	"English": true, "European": true, "Excel": true, "February": true, "Fi": true,
	"Firebase": true, "Firefox": true, "French": true, "Friday": true, "Gaussian": true,
	"German": true, "GitHub": true, "GitLab": true, "Google": true, "Greek": true,
	"Intel": true, "Italian": true, "January": true, "Japanese": true, "Java": true,
	"JavaScript": true, "July": true, "June": true, "Kafka": true, "Korean": true,
	"Kubernetes": true, "Latin": true, "Linux": true, "Mac": true, "Mail": true,
	"March": true, "Markdown": true, "May": true, "Microsoft": true, "Monday": true,
	"Mozilla": true, "Nginx": true, "Node": true, "November": true, "Nvidia": true,
	"October": true, "Oracle": true, "Packard": true, "Portuguese": true, "Postgres": true,
	"Python": true, "Rails": true, "Ray": true, "React": true, "Redis": true,
	"Royce": true, "Russian": true, "Safari": true, "Saturday": true, "September": true,
	"Slack": true, "Spanish": true, "Sunday": true, "Thursday": true, "Tuesday": true,
	"Ubuntu": true, "Unix": true, "Watson": true, "Wednesday": true, "Windows": true,
	"Word": true,
}

var formatdocFilenames = map[string]bool{
	"AGENTS": true, "CLAUDE": true, "CONTRIBUTING": true,
	"LICENSE": true, "NOTICE": true, "README": true,
}

var formatreAllCaps = regexp.MustCompile(`\b(?:ABSOLUTELY|ALL|ALWAYS|ANY|CAUTION|CRITICAL|DANGER|DEFINITELY|DEPRECATED|ESSENTIAL|EVERY|EXTREMELY|IMPORTANT|MUST|NEVER|NONE|NOTICE|NOTE|NOT|ONLY|PLEASE|REALLY|REQUIRED|SHOULD|VERY|WARNING)(?:'S|S)?\b`)

func formatisPlaceholder(text string, start, end int) bool {
	if start == 0 || end >= len(text) {
		return false
	}
	return (text[start-1] == '<' && text[end] == '>') ||
		(text[start-1] == '{' && text[end] == '}')
}

func formatonlyBlockMarkers(text string, from, to int) bool {
	for i := from; i < to; i++ {
		switch text[i] {
		case ' ', '\t', '>', '*', '_', '-', '+':
		default:
			return false
		}
	}
	return true
}

func formatisNoticeLabel(text string, start, end int) bool {
	lineStart := strings.LastIndexByte(text[:start], '\n') + 1
	if !formatonlyBlockMarkers(text, lineStart, start) {
		return false
	}
	i := end
	for i < len(text) && (text[i] == '*' || text[i] == '_') {
		i++
	}
	return i < len(text) && text[i] == ':'
}

func formatisAlertMarker(text string, start, end int) bool {
	if start < 2 || end >= len(text) ||
		text[start-2] != '[' || text[start-1] != '!' || text[end] != ']' {
		return false
	}
	lineStart := strings.LastIndexByte(text[:start-2], '\n') + 1
	return formatonlyBlockMarkers(text, lineStart, start-2)
}

// DetectGoogleAllCapsEmphasis reports ordinary words set in all caps to carry
// emphasis, which screen readers may spell out letter by letter.
func DetectGoogleAllCapsEmphasis(text string) []types.Violation {
	code := formatcodeSpans(text)
	var out []types.Violation
	for _, m := range formatreAllCaps.FindAllStringIndex(text, -1) {
		if formatcovers(code, m[0]) ||
			formatdocFilenames[text[m[0]:m[1]]] ||
			formatisPlaceholder(text, m[0], m[1]) ||
			formatisNoticeLabel(text, m[0], m[1]) ||
			formatisAlertMarker(text, m[0], m[1]) {
			continue
		}
		out = append(out, types.Violation{
			RuleID:      "all-caps-emphasis",
			StartIndex:  m[0],
			EndIndex:    m[1],
			MatchedText: text[m[0]:m[1]],
			Explanation: "all caps used for emphasis",
		})
	}
	return out
}

var (
	formatreAmpersandConj = regexp.MustCompile(`[\p{L}\p{N}][ \t]+&[ \t]+[\p{L}\p{N}]`)
	formatreAmpersandLead = regexp.MustCompile(`(?m)^[ \t]*&[ \t]`)
	formatreEntity        = regexp.MustCompile(`^&(?:#[0-9]+|#[xX][0-9a-fA-F]+|[a-zA-Z][a-zA-Z0-9]*);`)
	formatreUILabelSpan   = regexp.MustCompile(`\*\*[^*\n]+\*\*|__[^_\n]+__|"[^"\n]*"|\x{201C}[^\x{201D}]*\x{201D}`)
)

// DetectGoogleAmpersandConjunction reports an ampersand standing in for the
// word "and" outside a literal UI or menu name.
func DetectGoogleAmpersandConjunction(text string) []types.Violation {
	code := formatcodeSpans(text)
	labels := make([]formatspan, 0, 8)
	for _, m := range formatreUILabelSpan.FindAllStringIndex(text, -1) {
		labels = append(labels, formatspan{m[0], m[1]})
	}
	seen := make(map[int]bool)
	var out []types.Violation
	add := func(at int) {
		if seen[at] || formatcovers(code, at) || formatcovers(labels, at) ||
			formatreEntity.MatchString(text[at:]) {
			return
		}
		seen[at] = true
		out = append(out, types.Violation{
			RuleID:          "ampersand-conjunction",
			StartIndex:      at,
			EndIndex:        at + 1,
			MatchedText:     "&",
			Explanation:     "ampersand used as a conjunction",
			SuggestedChange: "and",
		})
	}
	for _, m := range formatreAmpersandConj.FindAllStringIndex(text, -1) {
		add(m[0] + strings.IndexByte(text[m[0]:m[1]], '&'))
	}
	for _, m := range formatreAmpersandLead.FindAllStringIndex(text, -1) {
		add(m[0] + strings.IndexByte(text[m[0]:m[1]], '&'))
	}
	return out
}

var formatreHyphenCap = regexp.MustCompile(`(?m)(?:^[ \t]{0,3}#{1,6}[ \t]+|^|[.!?][ \t]+)([A-Z][a-z]+)-([A-Z][a-z]+)`)

func formatnextWordCapped(text string, end int) bool {
	i := end
	for i < len(text) && (text[i] == ' ' || text[i] == '\t') {
		i++
	}
	return i > end && i < len(text) && text[i] >= 'A' && text[i] <= 'Z'
}

// DetectGoogleHyphenatedWordCapitalization reports a hyphenated compound whose
// second element is capitalized at the start of a sentence or heading.
func DetectGoogleHyphenatedWordCapitalization(text string) []types.Violation {
	code := formatcodeSpans(text)
	var out []types.Violation
	for _, m := range formatreHyphenCap.FindAllStringSubmatchIndex(text, -1) {
		start, end := m[2], m[5]
		first, second := text[m[2]:m[3]], text[m[4]:m[5]]
		if formatcovers(code, start) ||
			formatproperNouns[first] || formatproperNouns[second] ||
			formatnextWordCapped(text, end) {
			continue
		}
		out = append(out, types.Violation{
			RuleID:          "hyphenated-word-capitalization",
			StartIndex:      start,
			EndIndex:        end,
			MatchedText:     text[start:end],
			Explanation:     "only the first element of a hyphenated word takes the capital",
			SuggestedChange: first + "-" + strings.ToLower(second),
		})
	}
	return out
}

var (
	formatreListItem   = regexp.MustCompile(`(?m)^[ \t]{0,6}(?:[-*+]|\d+[.)])[ \t]+(.*)$`)
	formatreCaption    = regexp.MustCompile(`(?m)^[ \t]*(?:Figure|Table|Example)[ \t]+\d+[ \t]*[.:][ \t]+(.*)$`)
	formatreTableRow   = regexp.MustCompile(`(?m)^[ \t]*\|.*\|[ \t]*$`)
	formatreStrongSpan = regexp.MustCompile(`\*\*[^*\n]+\*\*|__[^_\n]+__`)
	formatreMDLink     = regexp.MustCompile(`\[[^\]\n]*\]\([^)\n]*\)`)
	formatreBareURL    = regexp.MustCompile(`https?://\S+`)
	formatreHTMLTag    = regexp.MustCompile(`<[^>\n]*>`)
	formatreEmphMark   = regexp.MustCompile(`[*_~]{1,3}`)
	formatreTitleToken = regexp.MustCompile(`^[A-Z][a-z]+$`)
)

var formatreHTMLCells = func() []*regexp.Regexp {
	tags := []string{"li", "td", "th", "caption", "figcaption"}
	out := make([]*regexp.Regexp, 0, len(tags))
	for _, t := range tags {
		out = append(out, regexp.MustCompile(`(?is)<`+t+`\b[^>]*>(.*?)</`+t+`>`))
	}
	return out
}()

func formatstripMarkup(s string) string {
	s = formatreInlineCode.ReplaceAllString(s, " x ")
	s = formatreStrongSpan.ReplaceAllString(s, " x ")
	s = formatreMDLink.ReplaceAllString(s, " x ")
	s = formatreBareURL.ReplaceAllString(s, " x ")
	s = formatreHTMLTag.ReplaceAllString(s, " ")
	return formatreEmphMark.ReplaceAllString(s, "")
}

func formathasLetter(s string) bool {
	for _, r := range s {
		if (r >= 'a' && r <= 'z') || (r >= 'A' && r <= 'Z') {
			return true
		}
	}
	return false
}

func formattokens(s string) []string {
	var out []string
	for _, f := range strings.Fields(s) {
		t := strings.Trim(f, ".,;:!?()[]{}\"'“”‘’")
		if t == "" || !formathasLetter(t) {
			continue
		}
		out = append(out, t)
	}
	return out
}

func formattitleCased(fragments []string) bool {
	run, nonFirst := 0, 0
	for _, f := range fragments {
		toks := formattokens(formatstripMarkup(f))
		if len(toks) == 0 {
			continue
		}
		for _, t := range toks {
			if formatproperNouns[t] {
				return false
			}
		}
		nonFirst += len(toks) - 1
		for i := 1; i < len(toks); i++ {
			if !formatreTitleToken.MatchString(toks[i]) {
				break
			}
			run++
		}
	}
	return run >= 2 && nonFirst > 0 && run*2 >= nonFirst
}

func formatsplitCells(row string) []string {
	var cells []string
	var cur strings.Builder
	escaped := false
	for i := 0; i < len(row); i++ {
		c := row[i]
		switch {
		case escaped:
			cur.WriteByte(c)
			escaped = false
		case c == '\\':
			escaped = true
		case c == '|':
			cells = append(cells, cur.String())
			cur.Reset()
		default:
			cur.WriteByte(c)
		}
	}
	cells = append(cells, cur.String())
	out := cells[:0]
	for _, c := range cells {
		if strings.TrimSpace(c) != "" {
			out = append(out, c)
		}
	}
	return out
}

func formatisSeparatorRow(cells []string) bool {
	for _, c := range cells {
		if strings.Trim(c, "-: \t") != "" {
			return false
		}
	}
	return true
}

// DetectGoogleListItemSentenceCase reports title case in a list item, table
// row, or figure caption, all of which take sentence case.
func DetectGoogleListItemSentenceCase(text string) []types.Violation {
	code := formatcodeSpans(text)
	var out []types.Violation
	flag := func(start, end int, fragments []string) {
		if formatcovers(code, start) || !formattitleCased(fragments) {
			return
		}
		s, e := trimSpan(text, start, end)
		if s >= e {
			return
		}
		out = append(out, types.Violation{
			RuleID:      "list-item-sentence-case",
			StartIndex:  s,
			EndIndex:    e,
			MatchedText: text[s:e],
			Explanation: "list items, tables, and captions take sentence case",
		})
	}
	for _, re := range []*regexp.Regexp{formatreListItem, formatreCaption} {
		for _, m := range re.FindAllStringSubmatchIndex(text, -1) {
			flag(m[2], m[3], []string{text[m[2]:m[3]]})
		}
	}
	for _, re := range formatreHTMLCells {
		for _, m := range re.FindAllStringSubmatchIndex(text, -1) {
			flag(m[2], m[3], []string{text[m[2]:m[3]]})
		}
	}
	for _, m := range formatreTableRow.FindAllStringIndex(text, -1) {
		cells := formatsplitCells(text[m[0]:m[1]])
		if len(cells) == 0 || formatisSeparatorRow(cells) {
			continue
		}
		flag(m[0], m[1], cells)
	}
	return out
}

var (
	formatreTermIntro  = regexp.MustCompile(`(?i)(?:\bthe (?:term|word|phrase|letter)\b|\bcalled\b|\bknown as\b|\breferred to as\b|\bwe call\b)\s+(?:\*\*[^*\n]{1,60}\*\*|__[^_\n]{1,60}__|"[^"\n]{1,60}"|<b\b|<strong\b)`)
	formatreTermDef    = regexp.MustCompile(`(?m)^(?:A|An|The)\s+(?:\*\*[^*\n]{1,60}\*\*|"[^"\n]{1,60}")\s+(?:is|are|refers to|means)\b`)
	formatreStrongTerm = regexp.MustCompile(`\*\*([^*\n]{1,60})\*\*|__([^_\n]{1,60})__`)
	formatreQuotedTerm = regexp.MustCompile(`"([^"\n]{1,60})"`)
)

func formatitalicize(s string) string {
	out := formatreStrongTerm.ReplaceAllString(s, "_${1}${2}_")
	out = formatreQuotedTerm.ReplaceAllString(out, "_${1}_")
	if out == s {
		return ""
	}
	return out
}

// DetectGoogleTermIntroductionFormatting reports a term introduced in bold or
// quotation marks where the style guide calls for italics.
func DetectGoogleTermIntroductionFormatting(text string) []types.Violation {
	code := formatcodeSpans(text)
	var out []types.Violation
	for _, re := range []*regexp.Regexp{formatreTermIntro, formatreTermDef} {
		for _, m := range re.FindAllStringIndex(text, -1) {
			matched := text[m[0]:m[1]]
			if formatcovers(code, m[0]) || strings.Contains(matched, "`") {
				continue
			}
			out = append(out, types.Violation{
				RuleID:          "term-introduction-formatting",
				StartIndex:      m[0],
				EndIndex:        m[1],
				MatchedText:     matched,
				Explanation:     "a term being defined takes italics",
				SuggestedChange: formatitalicize(matched),
			})
		}
	}
	return out
}

var (
	formatreUnderlineTag  = regexp.MustCompile(`(?i)<u\b[^>]*>`)
	formatreInsTag        = regexp.MustCompile(`(?i)<ins\b[^>]*>`)
	formatreTextDecor     = regexp.MustCompile(`(?i)text-decoration(?:-line)?\s*:\s*[^;"'\n]*underline`)
	formatreChangeTrack   = regexp.MustCompile(`(?i)\b(?:datetime|cite)\s*=`)
	formatreAnchorContext = regexp.MustCompile(`(?i)(?:^|[\s,>~+])a(?::[a-z-]+)?\s*[,{][^{}]*$`)
)

func formatanchorScoped(text string, at int) bool {
	lo := strings.LastIndexByte(text[:at], '}') + 1
	if at-lo > 240 {
		lo = at - 240
	}
	for lo < at && text[lo]&0xC0 == 0x80 {
		lo++
	}
	window := text[lo:at]
	return strings.Contains(strings.ToLower(window), "<a ") ||
		formatreAnchorContext.MatchString(window)
}

// DetectGoogleUnderlineNonLink reports underlined text that is not a link.
func DetectGoogleUnderlineNonLink(text string) []types.Violation {
	code := formatcodeSpans(text)
	var out []types.Violation
	emit := func(start, end int) {
		out = append(out, types.Violation{
			RuleID:      "underline-non-link",
			StartIndex:  start,
			EndIndex:    end,
			MatchedText: text[start:end],
			Explanation: "underline signals a link",
		})
	}
	for _, m := range formatreUnderlineTag.FindAllStringIndex(text, -1) {
		if formatcovers(code, m[0]) {
			continue
		}
		emit(m[0], m[1])
	}
	for _, m := range formatreInsTag.FindAllStringIndex(text, -1) {
		if formatcovers(code, m[0]) || formatreChangeTrack.MatchString(text[m[0]:m[1]]) {
			continue
		}
		emit(m[0], m[1])
	}
	for _, m := range formatreTextDecor.FindAllStringIndex(text, -1) {
		if formatcovers(code, m[0]) || formatanchorScoped(text, m[0]) {
			continue
		}
		emit(m[0], m[1])
	}
	return out
}
