package detectors

import (
	"regexp"
	"sort"
	"strings"
	"unicode/utf8"

	"github.com/yasyf/slop-cop/internal/types"
)

type linksLink struct {
	start, end         int
	textStart, textEnd int
	text               string
	dest               string
	attrs              string
	isImage            bool
}

var (
	linksreInlineLink = regexp.MustCompile(`(!?)\[([^\]\n]*)\]\(([^)\n]*)\)`)
	linksreRefLink    = regexp.MustCompile(`(!?)\[([^\]\n]*)\]\[([^\]\n]*)\]`)
	linksreRefDef     = regexp.MustCompile(`(?m)^[ \t]{0,3}\[([^\]\n^]+)\]:[ \t]*(\S+)`)
	linksreAnchor     = regexp.MustCompile(`(?i)<a\b([^>]*)>([^<]*)</a>`)
	linksreHref       = regexp.MustCompile(`(?i)href[ \t]*=[ \t]*["']([^"']*)["']`)
	linksreFenceLine  = regexp.MustCompile("(?m)^[ \t]{0,3}(?:`{3,}|~{3,})[^\n]*$")
	linksreCodeSpan   = regexp.MustCompile("`[^`\n]*`")
	linksreWhitespace = regexp.MustCompile(`\s+`)
	linksreH2         = regexp.MustCompile(`(?m)^[ \t]{0,3}##[ \t]`)
	linksreTracking   = regexp.MustCompile(`(?i)(?:^|&)(?:utm_[a-z_]+|gclid|fbclid|mc_cid|mc_eid|ref_src|_ga)=[^&]*`)
)

const linkstrimCut = " \t\n\r.,;:!?\"'`*_“”‘’…"

func linkscodeRanges(text string) [][2]int {
	var out [][2]int
	fences := linksreFenceLine.FindAllStringIndex(text, -1)
	for i := 0; i+1 < len(fences); i += 2 {
		out = append(out, [2]int{fences[i][0], fences[i+1][1]})
	}
	if len(fences)%2 == 1 {
		out = append(out, [2]int{fences[len(fences)-1][0], len(text)})
	}
	for _, idx := range linksreCodeSpan.FindAllStringIndex(text, -1) {
		if linksinsideAny(out, idx[0], idx[1]) {
			continue
		}
		out = append(out, [2]int{idx[0], idx[1]})
	}
	return out
}

func linksinsideAny(ranges [][2]int, start, end int) bool {
	for _, r := range ranges {
		if start >= r[0] && end <= r[1] {
			return true
		}
	}
	return false
}

func linksoverlapsAny(ranges [][2]int, start, end int) bool {
	for _, r := range ranges {
		if start < r[1] && r[0] < end {
			return true
		}
	}
	return false
}

func linkscleanDest(d string) string {
	d = strings.TrimSpace(d)
	if i := strings.IndexAny(d, " \t"); i >= 0 {
		d = d[:i]
	}
	d = strings.TrimPrefix(d, "<")
	return strings.TrimSuffix(d, ">")
}

func linksrefDefs(text string) map[string]string {
	out := map[string]string{}
	for _, m := range linksreRefDef.FindAllStringSubmatch(text, -1) {
		out[strings.ToLower(strings.TrimSpace(m[1]))] = linkscleanDest(m[2])
	}
	return out
}

func linksExtract(text string) []linksLink {
	code := linkscodeRanges(text)
	defs := linksrefDefs(text)
	var out []linksLink
	add := func(l linksLink) {
		if linksinsideAny(code, l.start, l.end) {
			return
		}
		out = append(out, l)
	}
	for _, m := range linksreInlineLink.FindAllStringSubmatchIndex(text, -1) {
		add(linksLink{
			start:     m[0],
			end:       m[1],
			textStart: m[4],
			textEnd:   m[5],
			text:      text[m[4]:m[5]],
			dest:      linkscleanDest(text[m[6]:m[7]]),
			isImage:   m[3] > m[2],
		})
	}
	for _, m := range linksreRefLink.FindAllStringSubmatchIndex(text, -1) {
		label := strings.ToLower(strings.TrimSpace(text[m[6]:m[7]]))
		if label == "" {
			label = strings.ToLower(strings.TrimSpace(text[m[4]:m[5]]))
		}
		dest, ok := defs[label]
		if !ok {
			continue
		}
		add(linksLink{
			start:     m[0],
			end:       m[1],
			textStart: m[4],
			textEnd:   m[5],
			text:      text[m[4]:m[5]],
			dest:      dest,
			isImage:   m[3] > m[2],
		})
	}
	for _, m := range linksreAnchor.FindAllStringSubmatchIndex(text, -1) {
		attrs := text[m[2]:m[3]]
		dest := ""
		if h := linksreHref.FindStringSubmatch(attrs); h != nil {
			dest = strings.TrimSpace(h[1])
		}
		add(linksLink{
			start:     m[0],
			end:       m[1],
			textStart: m[4],
			textEnd:   m[5],
			text:      text[m[4]:m[5]],
			dest:      dest,
			attrs:     attrs,
		})
	}
	sort.SliceStable(out, func(i, j int) bool { return out[i].start < out[j].start })
	return out
}

func linksnormText(s string) string {
	s = linksreWhitespace.ReplaceAllString(strings.ToLower(s), " ")
	return strings.Trim(s, linkstrimCut)
}

func linksnormDest(d string) string {
	frag := ""
	if i := strings.IndexByte(d, '#'); i >= 0 {
		frag, d = d[i:], d[:i]
	}
	query := ""
	if i := strings.IndexByte(d, '?'); i >= 0 {
		query, d = d[i+1:], d[:i]
	}
	query = strings.TrimPrefix(linksreTracking.ReplaceAllString(query, ""), "&")
	if i := strings.Index(d, "://"); i >= 0 {
		if j := strings.IndexByte(d[i+3:], '/'); j >= 0 {
			d = strings.ToLower(d[:i+3+j]) + d[i+3+j:]
		} else {
			d = strings.ToLower(d)
		}
	}
	d = strings.TrimSuffix(d, "/")
	if query != "" {
		d += "?" + query
	}
	return d + frag
}

func linkslineStarts(text string) []int {
	starts := []int{0}
	for i := 0; i < len(text); i++ {
		if text[i] == '\n' {
			starts = append(starts, i+1)
		}
	}
	return starts
}

func linkslineAt(starts []int, pos int) int {
	return sort.SearchInts(starts, pos+1) - 1
}

func linkssectionStarts(text string) []int {
	out := make([]int, 0, 8)
	for _, idx := range linksreH2.FindAllStringIndex(text, -1) {
		out = append(out, idx[0])
	}
	return out
}

func linksanyTabular(lines []string) bool {
	for _, l := range lines {
		if isTabular(l) {
			return true
		}
	}
	return false
}

func linksviolation(ruleID, text string, start, end int, explanation string) types.Violation {
	return types.Violation{
		RuleID:      ruleID,
		StartIndex:  start,
		EndIndex:    end,
		MatchedText: text[start:end],
		Explanation: explanation,
	}
}

// DetectGoogleAmbiguousRepeatedLinkText reports link labels reused for more
// than one destination.
func DetectGoogleAmbiguousRepeatedLinkText(text string) []types.Violation {
	links := linksExtract(text)
	dests := map[string]map[string]bool{}
	for _, l := range links {
		key := linksnormText(l.text)
		if l.dest == "" || utf8.RuneCountInString(key) < 2 {
			continue
		}
		if dests[key] == nil {
			dests[key] = map[string]bool{}
		}
		dests[key][linksnormDest(l.dest)] = true
	}
	var out []types.Violation
	for _, l := range links {
		if l.dest == "" || len(dests[linksnormText(l.text)]) < 2 {
			continue
		}
		out = append(out, linksviolation("ambiguous-repeated-link-text", text, l.start, l.end,
			"same link text, different destinations"))
	}
	return out
}

const linksdupLineWindow = 60

// DetectGoogleDuplicateLinkTarget reports a destination linked again close by
// under the same level-2 heading.
func DetectGoogleDuplicateLinkTarget(text string) []types.Violation {
	links := linksExtract(text)
	if len(links) < 2 {
		return nil
	}
	lineStarts := linkslineStarts(text)
	sectionStarts := linkssectionStarts(text)
	type linksplace struct{ line, section int }
	last := map[string]linksplace{}
	var out []types.Violation
	for _, l := range links {
		if l.dest == "" || strings.HasPrefix(l.dest, "#") {
			continue
		}
		if linksanyTabular(spanLines(text, l.start, l.end)) {
			continue
		}
		key := linksnormDest(l.dest)
		here := linksplace{
			line:    linkslineAt(lineStarts, l.start),
			section: sort.SearchInts(sectionStarts, l.start+1),
		}
		prev, seen := last[key]
		if seen && prev.section == here.section && here.line-prev.line <= linksdupLineWindow {
			out = append(out, linksviolation("duplicate-link-target", text, l.start, l.end,
				"this destination is already linked nearby"))
		}
		last[key] = here
	}
	return out
}

var (
	linksreFootnote   = regexp.MustCompile(`(?m)^[ \t]*\[\^[^\]\n]+\]:[ \t]|\[\^[^\]\s]+\]|<sup>[ \t]*[0-9*\x{2020}\x{2021}]+[ \t]*</sup>|<a\b[^>]*(?i:class="[^"]*footnote|href="#fn)`)
	linksreSuperDigit = regexp.MustCompile(`\p{L}{2,}[\x{00B9}\x{00B2}\x{00B3}\x{2070}-\x{2079}]`)
)

var linksunitWords = map[string]bool{
	"km": true, "cm": true, "mm": true, "nm": true, "um": true,
	"ft": true, "in": true, "mi": true, "yd": true, "sq": true,
}

func linkssupIsExponent(text string, start int) bool {
	letters := 0
	for i := start; i > 0; {
		r, size := utf8.DecodeLastRuneInString(text[:i])
		if letters == 0 && (r >= '0' && r <= '9') {
			return true
		}
		if !linksisLetter(r) {
			break
		}
		letters++
		i -= size
	}
	return letters == 1
}

func linksisLetter(r rune) bool {
	return (r >= 'a' && r <= 'z') || (r >= 'A' && r <= 'Z')
}

// DetectGoogleFootnoteUsage reports footnote markers, definitions, and
// superscript reference numbers.
func DetectGoogleFootnoteUsage(text string) []types.Violation {
	code := linkscodeRanges(text)
	var out []types.Violation
	for _, v := range findAll(text, linksreFootnote, "footnote-usage") {
		if linksoverlapsAny(code, v.StartIndex, v.EndIndex) {
			continue
		}
		if strings.HasPrefix(v.MatchedText, "<sup>") && linkssupIsExponent(text, v.StartIndex) {
			continue
		}
		v.Explanation = "move the content inline"
		out = append(out, v)
	}
	for _, v := range findAll(text, linksreSuperDigit, "footnote-usage") {
		if linksoverlapsAny(code, v.StartIndex, v.EndIndex) {
			continue
		}
		word := strings.ToLower(strings.TrimRightFunc(v.MatchedText, func(r rune) bool { return !linksisLetter(r) }))
		if linksunitWords[word] {
			continue
		}
		v.Explanation = "move the content inline"
		out = append(out, v)
	}
	sort.SliceStable(out, func(i, j int) bool { return out[i].StartIndex < out[j].StartIndex })
	return out
}

const (
	linksmaxLinkWords = 12
	linksmaxLinkChars = 80
)

var linksreInnerSentence = regexp.MustCompile(`[^\p{Lu}][.!?]\s+\p{Lu}`)

// DetectGoogleOverlongLinkText reports link labels stretched past a phrase.
func DetectGoogleOverlongLinkText(text string) []types.Violation {
	var out []types.Violation
	for _, l := range linksExtract(text) {
		if l.isImage {
			continue
		}
		label := strings.TrimSpace(linksreCodeSpan.ReplaceAllString(l.text, ""))
		if label == "" {
			continue
		}
		if len(strings.Fields(label)) <= linksmaxLinkWords &&
			utf8.RuneCountInString(label) <= linksmaxLinkChars &&
			!linksreInnerSentence.MatchString(label) {
			continue
		}
		out = append(out, linksviolation("overlong-link-text", text, l.textStart, l.textEnd,
			"cut the label to a short phrase"))
	}
	return out
}

var linksrePunctuationInLink = []*regexp.Regexp{
	regexp.MustCompile(`\[[^\]\n]*[\p{L}"'\x{201d}\x{2019}][.,;:!?][ \t]*\]\([^)\n]*\)`),
	regexp.MustCompile(`\[[ \t]*["'\x{201c}\x{2018}][^\]\n]*["'\x{201d}\x{2019}][ \t]*\]\([^)\n]*\)`),
	regexp.MustCompile(`(?i)<a\b[^>]*>[^<]*[\p{L}"\x{201d}][.,;:!?][ \t]*</a>`),
	regexp.MustCompile(`(?i)<a\b[^>]*>[ \t]*["\x{201c}][^<]*["\x{201d}][ \t]*</a>`),
}

// DetectGooglePunctuationInsideLink reports sentence punctuation or quotation
// marks swallowed into a link label.
func DetectGooglePunctuationInsideLink(text string) []types.Violation {
	code := linkscodeRanges(text)
	seen := map[int]bool{}
	var out []types.Violation
	for _, re := range linksrePunctuationInLink {
		for _, v := range findAll(text, re, "punctuation-inside-link") {
			if seen[v.StartIndex] || linksinsideAny(code, v.StartIndex, v.EndIndex) {
				continue
			}
			seen[v.StartIndex] = true
			v.Explanation = "move the punctuation outside the link"
			out = append(out, v)
		}
	}
	sort.SliceStable(out, func(i, j int) bool { return out[i].StartIndex < out[j].StartIndex })
	return out
}

var (
	linksreDownloadExt = regexp.MustCompile(`(?i)\.(?:pdf|zip|tar\.gz|tgz|gz|csv|xlsx?|docx?|pptx?|dmg|pkg|exe|msi|iso|apk)(?:[?#][^\s)"']*)?$`)
	linksreDownloadSay = regexp.MustCompile(`(?i)\b(?:downloads?|pdf|zip|csv|spreadsheet|archive|installer|package|file)\b`)
	linksreMailto      = regexp.MustCompile(`(?i)^mailto:`)
	linksreMailSay     = regexp.MustCompile(`(?i)\b(?:e-?mail|write to|contact|send mail)\b`)
	linksreBlankAttr   = regexp.MustCompile(`(?i)target[ \t]*=[ \t]*["']?_blank`)
	linksreBlankSuffix = regexp.MustCompile(`(?i)^\{[^}\n]*_blank`)
	linksreNewTabSay   = regexp.MustCompile(`(?i)opens? in a new (?:tab|window)|new tab|new window`)
)

const linkscontextBytes = 60

func linkscontext(text string, l linksLink, extra int) string {
	start := l.start - extra
	if start < 0 {
		start = 0
	}
	end := l.end + extra
	if end > len(text) {
		end = len(text)
	}
	return text[start:l.start] + " " + l.text + " " + text[l.end:end]
}

// DetectGoogleUnexplainedLinkBehavior reports links that download a file, open
// a mail composer, or seize a new tab without saying so.
func DetectGoogleUnexplainedLinkBehavior(text string) []types.Violation {
	var out []types.Violation
	for _, l := range linksExtract(text) {
		if l.isImage || l.dest == "" {
			continue
		}
		switch {
		case linksreMailto.MatchString(l.dest):
			if linksreMailSay.MatchString(l.text) {
				continue
			}
			out = append(out, linksviolation("unexplained-link-behavior", text, l.start, l.end,
				"say that the link opens an email"))
		case linksreDownloadExt.MatchString(l.dest):
			if linksreDownloadSay.MatchString(linkscontext(text, l, linkscontextBytes)) {
				continue
			}
			out = append(out, linksviolation("unexplained-link-behavior", text, l.start, l.end,
				"say that the link downloads a file and name its type"))
		case linksopensNewTab(text, l):
			if linksreNewTabSay.MatchString(linkscontext(text, l, 2*linkscontextBytes) + " " + l.attrs) {
				continue
			}
			out = append(out, linksviolation("unexplained-link-behavior", text, l.start, l.end,
				"open the link in the current tab, or say that it opens a new one"))
		}
	}
	return out
}

func linksopensNewTab(text string, l linksLink) bool {
	if linksreBlankAttr.MatchString(l.attrs) {
		return true
	}
	end := l.end + linkscontextBytes
	if end > len(text) {
		end = len(text)
	}
	return linksreBlankSuffix.MatchString(text[l.end:end])
}

var linksvagueLabels = map[string]bool{
	"click here": true, "here": true, "this": true, "this link": true,
	"this page": true, "this document": true, "this doc": true,
	"this article": true, "this guide": true, "this tutorial": true,
	"this post": true, "this blog post": true, "this one": true,
	"read this": true, "read this document": true, "read more": true,
	"read the docs": true, "learn more": true, "more": true, "link": true,
	"more info": true, "more information": true, "see more": true,
	"see here": true, "go here": true, "full story": true, "details": true,
}

var linksreClickHere = regexp.MustCompile(`(?i)\bclick here\b`)

// DetectGoogleVagueLinkText reports link labels that identify no destination.
func DetectGoogleVagueLinkText(text string) []types.Violation {
	var out []types.Violation
	for _, l := range linksExtract(text) {
		if l.isImage {
			continue
		}
		if !linksvagueLabels[linksnormText(l.text)] && !linksreClickHere.MatchString(l.text) {
			continue
		}
		out = append(out, linksviolation("vague-link-text", text, l.textStart, l.textEnd,
			"name the destination in the link text"))
	}
	return out
}
