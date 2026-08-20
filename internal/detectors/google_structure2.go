package detectors

import (
	"regexp"
	"strings"

	"github.com/yasyf/slop-cop/internal/types"
)

type structure2span struct{ start, end int }

type structure2line struct {
	text   string
	start  int
	fenced bool
}

var structure2reFenceDelim = regexp.MustCompile("^[ \t]{0,3}(?:```|~~~)")

func structure2lines(text string) []structure2line {
	out := make([]structure2line, 0, strings.Count(text, "\n")+1)
	inFence := false
	for pos := 0; pos <= len(text); {
		line := text[pos:]
		next := len(text) + 1
		if i := strings.IndexByte(line, '\n'); i >= 0 {
			line = line[:i]
			next = pos + i + 1
		}
		ln := structure2line{text: line, start: pos, fenced: inFence}
		if structure2reFenceDelim.MatchString(line) {
			ln.fenced = true
			inFence = !inFence
		}
		out = append(out, ln)
		pos = next
	}
	return out
}

func structure2fencedRanges(text string) []structure2span {
	var out []structure2span
	for _, ln := range structure2lines(text) {
		if !ln.fenced {
			continue
		}
		end := ln.start + len(ln.text)
		if n := len(out); n > 0 && out[n-1].end+1 >= ln.start {
			out[n-1].end = end
			continue
		}
		out = append(out, structure2span{ln.start, end})
	}
	return out
}

func structure2inSpans(spans []structure2span, pos int) bool {
	for _, s := range spans {
		if pos >= s.start && pos < s.end {
			return true
		}
	}
	return false
}

func structure2indentOf(line string) int {
	return len(line) - len(strings.TrimLeft(line, " \t"))
}

var (
	structure2reATX           = regexp.MustCompile(`^[ \t]{0,3}(#{1,6})[ \t]+(\S.*)$`)
	structure2reHTMLHeading   = regexp.MustCompile(`(?is)<h[1-6][^>]*>(.*?)</h[1-6][ \t]*>`)
	structure2reOrderedMarker = regexp.MustCompile(`^[ \t]{0,6}\d{1,3}[.)][ \t]+`)
	structure2reListMarker    = regexp.MustCompile(`^[ \t]{0,6}(?:[-*+]|\d{1,3}[.)])[ \t]+`)
	structure2reLeadingMarkup = regexp.MustCompile("^[*_`\\s]+")
	structure2reStepPrefix    = regexp.MustCompile(`(?i)^(?:optional|note|tip|important|caution|warning)[ \t]*:[ \t]*`)
	structure2reContextClause = regexp.MustCompile(`(?i)^(?:to|in|on|if|when|from|under|after|before|for)\b[^,]{0,60},\s*`)
	structure2reSeqAdverb     = regexp.MustCompile(`(?i)^(?:then|next|finally|first|second|third|lastly|afterwards?|also|now)\b,?\s+`)
	structure2reImperative    = regexp.MustCompile(`(?i)^(?:click|select|choose|enter|type|open|close|run|use|set|add|remove|delete|create|install|configure|copy|paste|specify|update|start|stop|download|upload|deploy|save|press|go\s+to|navigate\s+to|replace|verify|repeat|check|clone|retrieve|build|enable|disable|restart|edit|rename|submit|apply|export|import|grant|expose|connect|launch|publish|upgrade|migrate|restore|generate|switch|wait|provision|review|confirm|attach|mount|scroll|drag|sign\s+in|log\s+in|turn\s+on|turn\s+off)\b`)
	structure2reBackground    = regexp.MustCompile(`(?i)^(?:you|your|we|the|this|that|these|those|there|it|its|an?|first|next|now|note|after|once|because|since|make\s+sure)\b`)
)

func structure2headings(text string) []structure2span {
	var out []structure2span
	for _, ln := range structure2lines(text) {
		if ln.fenced {
			continue
		}
		if m := structure2reATX.FindStringSubmatchIndex(ln.text); m != nil {
			out = append(out, structure2span{ln.start + m[4], ln.start + m[5]})
		}
	}
	for _, m := range structure2reHTMLHeading.FindAllStringSubmatchIndex(text, -1) {
		out = append(out, structure2span{m[2], m[3]})
	}
	return out
}

func structure2trimLeadingMarkup(s string) (string, int) {
	if m := structure2reLeadingMarkup.FindStringIndex(s); m != nil && m[0] == 0 {
		return s[m[1]:], m[1]
	}
	return s, 0
}

func structure2stripStepLead(s string, seq bool) string {
	s, _ = structure2trimLeadingMarkup(s)
	s = structure2reStepPrefix.ReplaceAllString(s, "")
	s, _ = structure2trimLeadingMarkup(s)
	if seq {
		s = structure2reSeqAdverb.ReplaceAllString(s, "")
	}
	s = structure2reContextClause.ReplaceAllString(s, "")
	s, _ = structure2trimLeadingMarkup(s)
	return s
}

func structure2orderedItems(text string) []structure2span {
	var out []structure2span
	lines := structure2lines(text)
	for i := 0; i < len(lines); i++ {
		ln := lines[i]
		if ln.fenced {
			continue
		}
		m := structure2reOrderedMarker.FindStringIndex(ln.text)
		if m == nil {
			continue
		}
		indent := structure2indentOf(ln.text)
		start := ln.start + m[1]
		end := ln.start + len(ln.text)
		for j := i + 1; j < len(lines); j++ {
			nx := lines[j]
			if nx.fenced || strings.TrimSpace(nx.text) == "" {
				break
			}
			if structure2indentOf(nx.text) <= indent || structure2reListMarker.MatchString(nx.text) {
				break
			}
			end = nx.start + len(nx.text)
			i = j
		}
		if start < end {
			out = append(out, structure2span{start, end})
		}
	}
	return out
}

var (
	structure2rePressChord  = regexp.MustCompile(`(?i)\bpress\s+(?:and\s+hold\s+)?(?:Ctrl|Control|Cmd|Command|\x{2318}|Alt|Option|\x{2325}|Shift|\x{21E7}|Meta|Win)\s*\+\s*\S`)
	structure2reBareChord   = regexp.MustCompile(`\b(?:Ctrl|Cmd|Alt|Shift|Meta)\s*\+\s*(?:Shift\s*\+\s*)?[A-Za-z0-9]\b`)
	structure2reShortcutRef = regexp.MustCompile(`(?i)shortcut|hotkey|keymap|key binding`)

	structure2shortcutSubs = []plainSub{
		{"press Ctrl+C", "copy"},
		{"press Ctrl+V", "paste"},
		{"press Cmd+S", "save"},
	}
	structure2shortcutRes = compileSubs(structure2shortcutSubs)
)

func structure2shortcutSuggestions(text string) map[int]string {
	out := make(map[int]string)
	for _, v := range findSubs(text, "keyboard-shortcut-instruction", structure2shortcutSubs, structure2shortcutRes) {
		out[v.StartIndex] = v.SuggestedChange
	}
	return out
}

func structure2sentenceAt(line string, pos int) string {
	off := 0
	for _, s := range mergeAbbrev(splitSentences(line)) {
		if pos >= off && pos < off+len(s) {
			return s
		}
		off += len(s)
	}
	return line
}

// DetectGoogleKeyboardShortcutInstruction reports instructions expressed as a
// key chord instead of as the action the chord performs.
func DetectGoogleKeyboardShortcutInstruction(text string) []types.Violation {
	suggestions := structure2shortcutSuggestions(text)
	var out []types.Violation
	suppressed := 0
	for _, ln := range structure2lines(text) {
		if ln.fenced {
			continue
		}
		if m := structure2reATX.FindStringSubmatchIndex(ln.text); m != nil {
			level := m[3] - m[2]
			if suppressed > 0 && level <= suppressed {
				suppressed = 0
			}
			if structure2reShortcutRef.MatchString(ln.text[m[4]:m[5]]) {
				suppressed = level
			}
			continue
		}
		if suppressed > 0 || isTabular(ln.text) {
			continue
		}
		pressed := structure2rePressChord.FindAllStringIndex(ln.text, -1)
		for _, idx := range pressed {
			out = append(out, structure2shortcutViolation(text, ln.start+idx[0], ln.start+idx[1], suggestions))
		}
		for _, idx := range structure2reBareChord.FindAllStringIndex(ln.text, -1) {
			if structure2overlaps(pressed, idx) {
				continue
			}
			if !structure2reImperative.MatchString(structure2stripStepLead(structure2sentenceAt(ln.text, idx[0]), true)) {
				continue
			}
			out = append(out, structure2shortcutViolation(text, ln.start+idx[0], ln.start+idx[1], suggestions))
		}
	}
	return out
}

func structure2overlaps(spans [][]int, idx []int) bool {
	for _, s := range spans {
		if idx[0] < s[1] && s[0] < idx[1] {
			return true
		}
	}
	return false
}

func structure2shortcutViolation(text string, start, end int, suggestions map[int]string) types.Violation {
	return types.Violation{
		RuleID:          "keyboard-shortcut-instruction",
		StartIndex:      start,
		EndIndex:        end,
		MatchedText:     text[start:end],
		SuggestedChange: suggestions[start],
	}
}

var structure2headingLinkRes = []*regexp.Regexp{
	regexp.MustCompile(`\[[^\]\n]+\](?:\([^)\n]*\)|\[[^\]\n]*\]|[ \t]{2,})`),
	regexp.MustCompile(`(?is)<a[ \t\r\n][^>]*href[ \t]*=[^>]*>`),
	regexp.MustCompile(`<https?://[^>\s]+>`),
}

// DetectGoogleLinkInHeading reports links carried inside a heading, where link
// styling and heading styling are indistinguishable.
func DetectGoogleLinkInHeading(text string) []types.Violation {
	var out []types.Violation
	for _, h := range structure2headings(text) {
		raw := text[h.start:h.end]
		for _, re := range structure2headingLinkRes {
			for _, idx := range re.FindAllStringIndex(raw, -1) {
				if idx[0] > 0 && raw[idx[0]-1] == '!' {
					continue
				}
				ts, te := trimSpan(text, h.start+idx[0], h.start+idx[1])
				if ts >= te {
					continue
				}
				out = append(out, types.Violation{
					RuleID:      "link-in-heading",
					StartIndex:  ts,
					EndIndex:    te,
					MatchedText: text[ts:te],
				})
			}
		}
	}
	return out
}

// DetectGoogleMultiActionStep reports a numbered step that carries two or more
// separate instructions.
func DetectGoogleMultiActionStep(text string) []types.Violation {
	var out []types.Violation
	for _, item := range structure2orderedItems(text) {
		n := 0
		for _, s := range mergeAbbrev(splitSentences(text[item.start:item.end])) {
			if structure2reImperative.MatchString(structure2stripStepLead(s, true)) {
				n++
			}
		}
		if n < 2 {
			continue
		}
		ts, te := trimSpan(text, item.start, item.end)
		out = append(out, types.Violation{
			RuleID:      "multi-action-step",
			StartIndex:  ts,
			EndIndex:    te,
			MatchedText: text[ts:te],
			Explanation: itoa(n) + " instructions",
		})
	}
	return out
}

var (
	structure2reSeqWordHeading = regexp.MustCompile(`(?i)^(?:Step|Part|Phase|Stage|Section|Chapter)\s+\d+\b`)
	structure2reNumHeading     = regexp.MustCompile(`^\d{1,2}[ \t]*[.):\x{2013}\x{2014}-][ \t]+\S`)
	structure2reDottedHeading  = regexp.MustCompile(`^\d{1,2}\.\d{1,2}[ \t]+\S`)
	structure2reVersionHeading = regexp.MustCompile(`(?i)^\d{1,2}\.\d{1,3}[ \t]*(?:[-\x{2013}\x{2014}(\[]|release|beta|alpha|rc\b|preview)`)
)

// DetectGoogleNumberedHeading reports a heading that carries a sequence number
// the heading hierarchy already encodes.
func DetectGoogleNumberedHeading(text string) []types.Violation {
	var out []types.Violation
	for _, h := range structure2headings(text) {
		body, off := structure2trimLeadingMarkup(text[h.start:h.end])
		hit := structure2reSeqWordHeading.MatchString(body)
		if !hit && (structure2reNumHeading.MatchString(body) || structure2reDottedHeading.MatchString(body)) {
			hit = !structure2reVersionHeading.MatchString(body)
		}
		if !hit {
			continue
		}
		ts, te := trimSpan(text, h.start+off, h.end)
		if ts >= te {
			continue
		}
		out = append(out, types.Violation{
			RuleID:      "numbered-heading",
			StartIndex:  ts,
			EndIndex:    te,
			MatchedText: text[ts:te],
		})
	}
	return out
}

var structure2runFollowingRes = []*regexp.Regexp{
	regexp.MustCompile(`(?i)\b(?:by\s+)?(?:run|runs|running|execute|executes|executing|issue|issuing|invoke|invoking|enter|entering|type|typing)\s+the\s+following\s+(?:command|commands|code|snippet|script|scripts|line)s?\b`),
	regexp.MustCompile(`(?i)\brun\s+the\s+(?:command|code|script)\s+below\b`),
	regexp.MustCompile(`(?im)^[ \t]*(?:[-*+]|\d{1,3}[.)])?[ \t]*Run the following:?[ \t]*$`),
}

// DetectGoogleRunTheFollowingCommand reports a step that describes the
// mechanics of a code block instead of what the command accomplishes.
func DetectGoogleRunTheFollowingCommand(text string) []types.Violation {
	fenced := structure2fencedRanges(text)
	var out []types.Violation
	for _, re := range structure2runFollowingRes {
		for _, idx := range re.FindAllStringIndex(text, -1) {
			if structure2inSpans(fenced, idx[0]) {
				continue
			}
			ts, te := trimSpan(text, idx[0], idx[1])
			if ts >= te {
				continue
			}
			out = append(out, types.Violation{
				RuleID:      "run-the-following-command",
				StartIndex:  ts,
				EndIndex:    te,
				MatchedText: text[ts:te],
			})
		}
	}
	return out
}

type structure2list struct {
	indent int
	marker string
	items  []structure2span
	nested bool
	lead   bool
}

var (
	structure2reListItemLine = regexp.MustCompile(`^([ \t]*)([-*+]|\d{1,3}[.)])(?:[ \t]+\S.*)?$`)
	structure2reFollowStep   = regexp.MustCompile(`(?i)\b(?:follows?\s+th(?:is|e\s+following)\s+step|do\s+the\s+following\s+step)\b`)
	structure2reHTMLList     = regexp.MustCompile(`(?is)<(?:ul|ol)\b[^>]*>(.*?)</(?:ul|ol)[ \t]*>`)
	structure2reHTMLListItem = regexp.MustCompile(`(?i)<li\b`)
	structure2reHTMLNested   = regexp.MustCompile(`(?i)<(?:ul|ol)\b`)
)

func structure2isThematicBreak(line string) bool {
	t := strings.Map(func(r rune) rune {
		if r == ' ' || r == '\t' || r == '\r' {
			return -1
		}
		return r
	}, line)
	if len(t) < 3 {
		return false
	}
	c := t[0]
	if c != '-' && c != '*' && c != '_' {
		return false
	}
	return strings.Count(t, string(c)) == len(t)
}

func structure2isPreambleLine(line string) bool {
	t := strings.TrimSpace(line)
	if !strings.HasSuffix(t, ":") || isATXHeading(line) || structure2reListItemLine.MatchString(line) {
		return false
	}
	return len(strings.Fields(t)) >= 2
}

// DetectGoogleSingleItemList reports a list that promises a set and delivers
// exactly one item.
func DetectGoogleSingleItemList(text string) []types.Violation {
	fenced := structure2fencedRanges(text)
	var out []types.Violation
	for _, idx := range structure2reFollowStep.FindAllStringIndex(text, -1) {
		if structure2inSpans(fenced, idx[0]) {
			continue
		}
		out = append(out, types.Violation{
			RuleID:      "single-item-list",
			StartIndex:  idx[0],
			EndIndex:    idx[1],
			MatchedText: text[idx[0]:idx[1]],
		})
	}
	var cur *structure2list
	blank := false
	prev := ""
	flush := func() {
		if cur == nil {
			return
		}
		if len(cur.items) == 1 && !cur.nested && (cur.lead || cur.marker == "1") {
			ts, te := trimSpan(text, cur.items[0].start, cur.items[0].end)
			if ts < te {
				out = append(out, types.Violation{
					RuleID:      "single-item-list",
					StartIndex:  ts,
					EndIndex:    te,
					MatchedText: text[ts:te],
				})
			}
		}
		cur = nil
	}
	for _, ln := range structure2lines(text) {
		if ln.fenced {
			blank = false
			continue
		}
		if strings.TrimSpace(ln.text) == "" {
			blank = true
			continue
		}
		m := structure2reListItemLine.FindStringSubmatchIndex(ln.text)
		if m != nil && !structure2isThematicBreak(ln.text) {
			indent := m[3] - m[2]
			marker := ln.text[m[4]:m[5]]
			if marker[0] >= '0' && marker[0] <= '9' {
				marker = "1"
			}
			span := structure2span{ln.start, ln.start + len(ln.text)}
			switch {
			case cur != nil && indent > cur.indent:
				cur.nested = true
				cur.items[len(cur.items)-1].end = span.end
			case cur != nil && indent == cur.indent && marker == cur.marker:
				cur.items = append(cur.items, span)
			default:
				flush()
				cur = &structure2list{
					indent: indent,
					marker: marker,
					lead:   structure2isPreambleLine(prev),
					items:  []structure2span{span},
				}
			}
			blank = false
			continue
		}
		if cur != nil {
			if blank && structure2indentOf(ln.text) <= cur.indent {
				flush()
			} else {
				cur.items[len(cur.items)-1].end = ln.start + len(ln.text)
			}
		}
		prev = ln.text
		blank = false
	}
	flush()
	for _, m := range structure2reHTMLList.FindAllStringSubmatchIndex(text, -1) {
		inner := text[m[2]:m[3]]
		if structure2reHTMLNested.MatchString(inner) {
			continue
		}
		if len(structure2reHTMLListItem.FindAllStringIndex(inner, -1)) != 1 {
			continue
		}
		out = append(out, types.Violation{
			RuleID:      "single-item-list",
			StartIndex:  m[0],
			EndIndex:    m[1],
			MatchedText: text[m[0]:m[1]],
		})
	}
	return out
}

func structure2isNonProseStep(content string) bool {
	t := strings.TrimSpace(content)
	if t == "" {
		return true
	}
	switch t[0] {
	case '`', '$', '<', '|', '#':
		return true
	}
	return !strings.Contains(t, " ")
}

func structure2stepHasAction(sentences []string) bool {
	for _, s := range sentences {
		if structure2reImperative.MatchString(structure2stripStepLead(s, true)) {
			return true
		}
	}
	return false
}

// DetectGoogleStepLacksImperative reports a numbered step whose first sentence
// describes a situation instead of naming the action.
func DetectGoogleStepLacksImperative(text string) []types.Violation {
	var out []types.Violation
	for _, item := range structure2orderedItems(text) {
		content := text[item.start:item.end]
		if structure2isNonProseStep(content) {
			continue
		}
		sentences := mergeAbbrev(splitSentences(content))
		if len(sentences) == 0 {
			continue
		}
		first := structure2stripStepLead(sentences[0], false)
		if !structure2reBackground.MatchString(first) || proseWordCount(first) < 4 {
			continue
		}
		if !structure2stepHasAction(sentences) {
			continue
		}
		ts, te := trimSpan(text, item.start, item.end)
		if ts >= te {
			continue
		}
		out = append(out, types.Violation{
			RuleID:      "step-lacks-imperative",
			StartIndex:  ts,
			EndIndex:    te,
			MatchedText: text[ts:te],
		})
	}
	return out
}
