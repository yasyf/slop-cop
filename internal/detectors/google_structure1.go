package detectors

import (
	"regexp"
	"sort"
	"strings"
	"unicode"

	"github.com/yasyf/slop-cop/internal/types"
)

type structure1Line struct {
	text  string
	start int
	code  bool
}

var (
	structure1reFenceTick  = regexp.MustCompile("^[ \t]{0,3}```")
	structure1reFenceTilde = regexp.MustCompile(`^[ \t]{0,3}~~~`)
)

func structure1Lines(text string) []structure1Line {
	raw := strings.Split(text, "\n")
	out := make([]structure1Line, 0, len(raw))
	off := 0
	fence := ""
	front := false
	for i, l := range raw {
		trimmedRight := strings.TrimRight(l, " \t\r")
		code := false
		switch {
		case i == 0 && trimmedRight == "---":
			front, code = true, true
		case front:
			code = true
			if trimmedRight == "---" || trimmedRight == "..." {
				front = false
			}
		case fence != "":
			code = true
			if strings.HasPrefix(strings.TrimLeft(l, " \t"), fence) {
				fence = ""
			}
		case structure1reFenceTick.MatchString(l):
			fence, code = "```", true
		case structure1reFenceTilde.MatchString(l):
			fence, code = "~~~", true
		}
		out = append(out, structure1Line{text: l, start: off, code: code})
		off += len(l) + 1
	}
	return out
}

func structure1codeAt(lines []structure1Line, off int) bool {
	i := sort.Search(len(lines), func(i int) bool { return lines[i].start > off }) - 1
	if i < 0 {
		return false
	}
	return lines[i].code
}

func structure1indent(s string) int {
	n := 0
	for _, c := range s {
		switch c {
		case ' ':
			n++
		case '\t':
			n += 4
		default:
			return n
		}
	}
	return n
}

type structure1Heading struct {
	level     int
	lineStart int
	lineEnd   int
	textStart int
	textEnd   int
	text      string
}

var (
	structure1reATX      = regexp.MustCompile(`^[ \t]{0,3}(#{1,6})[ \t]+(.*)$`)
	structure1reSetextH1 = regexp.MustCompile(`^[ \t]{0,3}=+[ \t]*\r?$`)
	structure1reSetextH2 = regexp.MustCompile(`^[ \t]{0,3}-{3,}[ \t]*\r?$`)
	structure1reHTMLHead = regexp.MustCompile(`(?is)<h([1-6])[^>]*>(.*?)</h[1-6]>`)
	structure1reListItem = regexp.MustCompile(`^([ \t]*)([-*+]|\d+[.)])[ \t]+(\S.*)$`)
	structure1reQuote    = regexp.MustCompile(`^[ \t]{0,3}>`)
)

func structure1Headings(text string) []structure1Heading {
	lines := structure1Lines(text)
	var out []structure1Heading
	for i, ln := range lines {
		if ln.code {
			continue
		}
		if m := structure1reATX.FindStringSubmatchIndex(ln.text); m != nil {
			body := strings.TrimRight(ln.text[m[4]:m[5]], " \t\r")
			body = strings.TrimRight(strings.TrimRight(body, "#"), " \t")
			if body == "" {
				continue
			}
			out = append(out, structure1Heading{
				level:     m[3] - m[2],
				lineStart: ln.start,
				lineEnd:   ln.start + len(ln.text),
				textStart: ln.start + m[4],
				textEnd:   ln.start + m[4] + len(body),
				text:      body,
			})
			continue
		}
		if i == 0 {
			continue
		}
		level := 0
		switch {
		case structure1reSetextH1.MatchString(ln.text):
			level = 1
		case structure1reSetextH2.MatchString(ln.text):
			level = 2
		default:
			continue
		}
		prev := lines[i-1]
		if prev.code || strings.TrimSpace(prev.text) == "" {
			continue
		}
		if structure1reATX.MatchString(prev.text) || structure1reListItem.MatchString(prev.text) ||
			structure1reQuote.MatchString(prev.text) || strings.Contains(prev.text, "|") {
			continue
		}
		lead := len(prev.text) - len(strings.TrimLeft(prev.text, " \t"))
		body := strings.TrimRight(prev.text[lead:], " \t\r")
		out = append(out, structure1Heading{
			level:     level,
			lineStart: prev.start,
			lineEnd:   ln.start + len(ln.text),
			textStart: prev.start + lead,
			textEnd:   prev.start + lead + len(body),
			text:      body,
		})
	}
	for _, m := range structure1reHTMLHead.FindAllStringSubmatchIndex(text, -1) {
		if structure1codeAt(lines, m[0]) {
			continue
		}
		out = append(out, structure1Heading{
			level:     int(text[m[2]] - '0'),
			lineStart: m[0],
			lineEnd:   m[1],
			textStart: m[4],
			textEnd:   m[5],
			text:      text[m[4]:m[5]],
		})
	}
	sort.SliceStable(out, func(i, j int) bool { return out[i].lineStart < out[j].lineStart })
	return out
}

var (
	structure1reMDLink    = regexp.MustCompile(`\[([^\]]*)\]\([^)]*\)`)
	structure1reCodeSpan  = regexp.MustCompile("`[^`]*`")
	structure1reHTMLTag   = regexp.MustCompile(`<[^>]+>`)
	structure1reStarEmph  = regexp.MustCompile(`\*{1,3}|~~`)
	structure1reUnderEmph = regexp.MustCompile(`(^|[ \t])_{1,3}|_{1,3}([ \t]|$)`)
	structure1reOptional  = regexp.MustCompile(`(?i)^optional\s*:\s*`)
)

func structure1stripInline(s string) string {
	s = structure1reMDLink.ReplaceAllString(s, "$1")
	s = structure1reCodeSpan.ReplaceAllString(s, " ")
	s = structure1reHTMLTag.ReplaceAllString(s, " ")
	s = structure1reStarEmph.ReplaceAllString(s, "")
	s = structure1reUnderEmph.ReplaceAllString(s, "$1$2")
	return strings.TrimSpace(s)
}

func structure1words(s string) []string {
	return strings.FieldsFunc(strings.ToLower(s), func(r rune) bool {
		return !unicode.IsLetter(r) && !unicode.IsDigit(r)
	})
}

func structure1jaccard(a, b map[string]bool) float64 {
	inter := 0
	for k := range a {
		if b[k] {
			inter++
		}
	}
	union := len(a) + len(b) - inter
	if union == 0 {
		return 0
	}
	return float64(inter) / float64(union)
}

func structure1subset(a, b map[string]bool) bool {
	for k := range a {
		if !b[k] {
			return false
		}
	}
	return true
}

var (
	structure1reAltStepLead = regexp.MustCompile(`(?i)^[ \t]{0,6}\d+[.)][ \t]*(?:Alternatively|Optionally|Or,)\b`)
	structure1reAltStepBody = regexp.MustCompile(`(?i)\b(?:alternatively|you can also|another way to|if you prefer|or you can|instead, you can|as an alternative)\b`)
	structure1reCallout     = regexp.MustCompile(`(?i)^[ \t]{0,6}(?:>|(?:\*\*|__)?(?:note|caution|warning|important|tip|key point)(?:\*\*|__)?[ \t]*:)`)
)

// DetectGoogleAlternativeInStep reports a second way to do the same thing
// offered inside a numbered step.
func DetectGoogleAlternativeInStep(text string) []types.Violation {
	lines := structure1Lines(text)
	var out []types.Violation
	for i, ln := range lines {
		if ln.code || !structure1reOrderedItem.MatchString(ln.text) {
			continue
		}
		indent := structure1indent(ln.text)
		end := ln.start + len(ln.text)
		callout := structure1reCallout.MatchString(ln.text)
		for j := i + 1; j < len(lines); j++ {
			nx := lines[j]
			if nx.code || strings.TrimSpace(nx.text) == "" {
				break
			}
			if structure1indent(nx.text) <= indent || structure1reListItem.MatchString(nx.text) {
				break
			}
			if structure1reCallout.MatchString(nx.text) {
				callout = true
			}
			end = nx.start + len(nx.text)
		}
		if callout {
			continue
		}
		seg := text[ln.start:end]
		var spans [][]int
		if m := structure1reAltStepLead.FindStringIndex(seg); m != nil {
			spans = append(spans, m)
		}
		for _, m := range structure1reAltStepBody.FindAllStringIndex(seg, -1) {
			if len(spans) > 0 && m[0] < spans[0][1] {
				continue
			}
			spans = append(spans, m)
		}
		for _, m := range spans {
			out = append(out, types.Violation{
				RuleID:      "alternative-in-step",
				StartIndex:  ln.start + m[0],
				EndIndex:    ln.start + m[1],
				MatchedText: seg[m[0]:m[1]],
			})
		}
	}
	return out
}

var structure1reOrderedItem = regexp.MustCompile(`^[ \t]{0,6}\d+[.)][ \t]+\S`)

var (
	structure1sectionSubs = []plainSub{
		{"these sections describe", "the following sections describe"},
		{"described in this section", "described in the following sections"},
	}
	structure1sectionRes   = compileSubs(structure1sectionSubs)
	structure1reSectionRef = []*regexp.Regexp{
		regexp.MustCompile(`(?i)\b(?:this|these)\s+sections?\s+(?:describe|describes|list|lists|cover|covers|explain|explains|show|shows|walk|walks)\b`),
		regexp.MustCompile(`(?i)\b(?:described|listed|covered|explained|shown|detailed)\s+in\s+(?:this|these)\s+sections?\b`),
		regexp.MustCompile(`(?i)\bin\s+(?:this|these)\s+sections?\s*[,:]`),
	}
)

// DetectGoogleAmbiguousSectionReference reports a forward-looking preview that
// points at the subsections after it as "this section" or "these sections".
func DetectGoogleAmbiguousSectionReference(text string) []types.Violation {
	hs := structure1Headings(text)
	var out []types.Violation
	for i, h := range hs {
		enclosing := 0
		for j := i - 1; j >= 0; j-- {
			if hs[j].level < h.level {
				enclosing = hs[j].level
				break
			}
		}
		if h.level <= enclosing {
			continue
		}
		regionStart := 0
		if i > 0 {
			regionStart = hs[i-1].lineEnd
		}
		if regionStart >= h.lineStart {
			continue
		}
		paras := splitParagraphs(text[regionStart:h.lineStart])
		if len(paras) == 0 {
			continue
		}
		p := paras[len(paras)-1]
		base := regionStart + p.start
		hits := findSubs(p.text, "ambiguous-section-reference", structure1sectionSubs, structure1sectionRes)
		for k := range hits {
			hits[k].StartIndex += base
			hits[k].EndIndex += base
		}
		out = append(out, hits...)
		for _, re := range structure1reSectionRef {
			for _, m := range re.FindAllStringIndex(p.text, -1) {
				if structure1overlapsHit(hits, base+m[0], base+m[1]) {
					continue
				}
				out = append(out, types.Violation{
					RuleID:      "ambiguous-section-reference",
					StartIndex:  base + m[0],
					EndIndex:    base + m[1],
					MatchedText: p.text[m[0]:m[1]],
				})
			}
		}
	}
	return out
}

func structure1overlapsHit(hits []types.Violation, start, end int) bool {
	for _, h := range hits {
		if start < h.EndIndex && h.StartIndex < end {
			return true
		}
	}
	return false
}

const structure1docNoun = `(?:diagram|image|figure|screenshot|table|example|code|snippet|section|command|output|list|steps?|procedure|paragraph|note|instructions?|chart)`

var (
	structure1directionalSubs = []plainSub{
		{"the above diagram", "the preceding diagram"},
		{"the table below", "the following table"},
		{"as shown above", "as shown earlier"},
		{"see below", "see the following section"},
	}
	structure1directionalRes = compileSubs(structure1directionalSubs)
	structure1reDirectional  = []*regexp.Regexp{
		regexp.MustCompile(`(?i)\b(?:the\s+)?(?:above|below)\s+` + structure1docNoun + `\b`),
		regexp.MustCompile(`(?i)\b` + structure1docNoun + `\s+(?:above|below)\b`),
		regexp.MustCompile(`(?i)\bas\s+(?:shown|described|noted|mentioned|discussed|explained|listed|defined|stated|outlined|seen)\s+(?:above|below)\b`),
		regexp.MustCompile(`(?i)\bsee\s+(?:the\s+)?[a-z ]{0,20}(?:above|below)\b`),
		regexp.MustCompile(`(?i)\b(?:right|left|top|bottom|upper|lower)[- ]hand\s+(?:side|corner|column|panel|pane|menu)\b`),
		regexp.MustCompile(`(?i)\bin\s+the\s+(?:upper|lower|top|bottom)[- ]?(?:right|left)\b`),
		regexp.MustCompile(`(?i)\bin\s+the\s+(?:left|right|upper|lower|top|bottom)[- ]?(?:side\s+)?(?:panel|pane|menu|column|corner|nav|navigation)\b`),
	}
	structure1reDirectionalSide = regexp.MustCompile(`(?i)\bon\s+the\s+(?:right|left)(?:\s+side)?\b`)
	structure1sideFollow        = map[string]bool{
		"side": true, "sides": true, "of": true, "pane": true, "panel": true,
		"menu": true, "column": true, "nav": true, "navigation": true,
		"corner": true, "edge": true, "toolbar": true, "sidebar": true,
		"tab": true, "tabs": true, "list": true,
	}
)

// DetectGoogleDirectionalReference reports content located by its position on
// the page rather than by name or sequence.
func DetectGoogleDirectionalReference(text string) []types.Violation {
	lines := structure1Lines(text)
	hits := findSubs(text, "directional-reference", structure1directionalSubs, structure1directionalRes)
	out := make([]types.Violation, 0, len(hits))
	for _, h := range hits {
		if !structure1codeAt(lines, h.StartIndex) {
			out = append(out, h)
		}
	}
	add := func(start, end int) {
		if structure1codeAt(lines, start) || structure1overlapsHit(hits, start, end) {
			return
		}
		out = append(out, types.Violation{
			RuleID:      "directional-reference",
			StartIndex:  start,
			EndIndex:    end,
			MatchedText: text[start:end],
		})
	}
	for _, re := range structure1reDirectional {
		for _, m := range re.FindAllStringIndex(text, -1) {
			if structure1endsDirectional(text[m[0]:m[1]]) && structure1numericFollows(text, m[1]) {
				continue
			}
			add(m[0], m[1])
		}
	}
	for _, m := range structure1reDirectionalSide.FindAllStringIndex(text, -1) {
		if m[1] < len(text) && text[m[1]] == '-' {
			continue
		}
		w := structure1nextWord(text, m[1])
		if w != "" && !structure1sideFollow[w] {
			continue
		}
		add(m[0], m[1])
	}
	return out
}

func structure1endsDirectional(s string) bool {
	l := strings.ToLower(s)
	return strings.HasSuffix(l, "above") || strings.HasSuffix(l, "below")
}

func structure1numericFollows(text string, i int) bool {
	for i < len(text) && (text[i] == ' ' || text[i] == '\t') {
		i++
	}
	return i < len(text) && (text[i] >= '0' && text[i] <= '9' || text[i] == '$')
}

func structure1nextWord(text string, i int) string {
	for i < len(text) && (text[i] == ' ' || text[i] == '\t') {
		i++
	}
	j := i
	for j < len(text) && structure1isWordByte(text[j]) {
		j++
	}
	return strings.ToLower(text[i:j])
}

func structure1isWordByte(b byte) bool {
	return b >= 'a' && b <= 'z' || b >= 'A' && b <= 'Z' || b >= '0' && b <= '9'
}

var (
	structure1reGerund   = regexp.MustCompile(`(?i)^[a-z]{3,}ing$`)
	structure1gerundSubs = []plainSub{
		{"Creating an instance", "Create an instance"},
		{"Migrating to the cloud", "Migration to the cloud"},
		{"Getting started", "Get started"},
	}
	structure1gerundRes   = compileSubs(structure1gerundSubs)
	structure1gerundAllow = map[string]bool{
		"billing": true, "pricing": true, "networking": true, "logging": true,
		"monitoring": true, "marketing": true, "engineering": true,
		"training": true, "onboarding": true, "troubleshooting": true,
		"licensing": true, "versioning": true, "caching": true, "routing": true,
		"indexing": true, "sharding": true, "hashing": true, "scheduling": true,
	}
)

// DetectGoogleGerundHeading reports a multi-word heading that opens on an -ing
// verb instead of a plain verb or a noun phrase.
func DetectGoogleGerundHeading(text string) []types.Violation {
	var out []types.Violation
	for _, h := range structure1Headings(text) {
		body := structure1reOptional.ReplaceAllString(structure1stripInline(h.text), "")
		fields := strings.Fields(body)
		if len(fields) < 2 {
			continue
		}
		first := strings.Trim(fields[0], "*_`\"'")
		if !structure1reGerund.MatchString(first) || structure1gerundAllow[strings.ToLower(first)] {
			continue
		}
		v := types.Violation{
			RuleID:      "gerund-heading",
			StartIndex:  h.textStart,
			EndIndex:    h.textEnd,
			MatchedText: text[h.textStart:h.textEnd],
		}
		if subs := findSubs(h.text, "gerund-heading", structure1gerundSubs, structure1gerundRes); len(subs) > 0 {
			v.SuggestedChange = subs[0].SuggestedChange
		}
		out = append(out, v)
	}
	return out
}

// DetectGoogleHeadingLevelSkip reports a heading that jumps more than one level
// below its predecessor, and every level-1 heading after the first.
func DetectGoogleHeadingLevelSkip(text string) []types.Violation {
	hs := structure1Headings(text)
	if len(hs) < 2 {
		return nil
	}
	var out []types.Violation
	h1 := 0
	for i, h := range hs {
		if h.level == 1 {
			h1++
			if h1 > 1 {
				out = append(out, types.Violation{
					RuleID:      "heading-level-skip",
					StartIndex:  h.textStart,
					EndIndex:    h.textEnd,
					MatchedText: text[h.textStart:h.textEnd],
					Explanation: "second level-1 heading",
				})
			}
		}
		if i == 0 || h.level-hs[i-1].level < 2 {
			continue
		}
		out = append(out, types.Violation{
			RuleID:      "heading-level-skip",
			StartIndex:  h.textStart,
			EndIndex:    h.textEnd,
			MatchedText: text[h.textStart:h.textEnd],
			Explanation: "level " + itoa(hs[i-1].level) + " to level " + itoa(h.level),
		})
	}
	return out
}

var structure1titleStop = map[string]bool{
	"a": true, "an": true, "the": true, "of": true, "for": true, "to": true,
	"in": true, "on": true, "with": true, "and": true, "or": true,
	"your": true, "this": true,
}

func structure1titleTokens(s string) map[string]bool {
	out := map[string]bool{}
	for _, w := range structure1words(structure1stripInline(s)) {
		if structure1titleStop[w] {
			continue
		}
		out[w] = true
	}
	return out
}

// DetectGoogleHeadingRepeatsTitle reports a section heading that restates the
// page title.
func DetectGoogleHeadingRepeatsTitle(text string) []types.Violation {
	hs := structure1Headings(text)
	var title map[string]bool
	for _, h := range hs {
		if h.level == 1 {
			title = structure1titleTokens(h.text)
			break
		}
	}
	if len(title) == 0 {
		return nil
	}
	var out []types.Violation
	for _, h := range hs {
		if h.level < 2 {
			continue
		}
		toks := structure1titleTokens(h.text)
		if len(toks) == 0 {
			continue
		}
		same := len(toks) == len(title) && structure1subset(toks, title)
		near := len(toks) >= 2 && len(title) >= 2 && structure1jaccard(title, toks) >= 0.9
		if !same && !near {
			continue
		}
		out = append(out, types.Violation{
			RuleID:      "heading-repeats-title",
			StartIndex:  h.textStart,
			EndIndex:    h.textEnd,
			MatchedText: text[h.textStart:h.textEnd],
			Explanation: "repeats the page title",
		})
	}
	return out
}

var (
	structure1reSuspect   = regexp.MustCompile(`^[A-Z][a-z]{2,}$`)
	structure1reAlphaOnly = regexp.MustCompile(`^[A-Za-z]+$`)
	structure1reWord      = regexp.MustCompile(`[A-Za-z][A-Za-z]+`)
	structure1funcWords   = map[string]bool{
		"The": true, "A": true, "An": true, "And": true, "Or": true,
		"But": true, "For": true, "Nor": true, "To": true, "In": true,
		"Of": true, "With": true, "By": true, "On": true, "From": true,
		"At": true, "As": true, "Is": true, "Are": true, "Your": true,
		"You": true, "Their": true, "Its": true,
	}
	structure1properNouns = map[string]bool{
		"Google": true, "Cloud": true, "Console": true, "Compute": true,
		"Engine": true, "Storage": true, "Firebase": true, "Android": true,
		"Chrome": true, "Docker": true, "Kubernetes": true, "Linux": true,
		"Windows": true, "Java": true, "Python": true, "Node": true,
		"React": true, "Angular": true, "Terraform": true, "Ansible": true,
		"Jenkins": true, "Slack": true, "Markdown": true, "Unicode": true,
		"Studio": true, "Workspace": true, "Postgres": true, "Redis": true,
		"Kafka": true, "Spark": true, "Hadoop": true, "Jupyter": true,
		"Anthropic": true, "Claude": true, "Azure": true, "Amazon": true,
		"Apple": true, "Microsoft": true, "Firefox": true, "Safari": true,
		"Ubuntu": true, "Debian": true, "Alpine": true, "Homebrew": true,
		"Maven": true, "Gradle": true, "Rust": true, "Swift": true,
		"Kotlin": true, "Ruby": true, "Rails": true, "Django": true,
		"Flask": true, "Nginx": true, "Apache": true, "Grafana": true,
		"Prometheus": true, "Datadog": true, "Looker": true, "Sheets": true,
		"Drive": true, "Gmail": true, "Maps": true, "Analytics": true,
		"Dataflow": true, "Dataproc": true, "Bigtable": true, "Spanner": true,
		"Firestore": true, "Vertex": true, "Gemini": true,
	}
)

func structure1bodyLower(text string, hs []structure1Heading) map[string]bool {
	out := map[string]bool{}
	for _, ln := range structure1Lines(text) {
		if ln.code {
			continue
		}
		start, end := ln.start, ln.start+len(ln.text)
		inHeading := false
		for _, h := range hs {
			if start < h.lineEnd && h.lineStart < end {
				inHeading = true
				break
			}
		}
		if inHeading {
			continue
		}
		for _, w := range structure1reWord.FindAllString(ln.text, -1) {
			if w == strings.ToLower(w) {
				out[w] = true
			}
		}
	}
	return out
}

// DetectGoogleHeadingSentenceCase reports a heading capitalized like a book
// title rather than like a sentence.
func DetectGoogleHeadingSentenceCase(text string) []types.Violation {
	hs := structure1Headings(text)
	if len(hs) == 0 {
		return nil
	}
	lower := structure1bodyLower(text, hs)
	var out []types.Violation
	for _, h := range hs {
		body := structure1reOptional.ReplaceAllString(structure1stripInline(h.text), "")
		toks := strings.Fields(body)
		if len(toks) < 2 {
			continue
		}
		funcWord := false
		allCapped := true
		capped := 0
		corroborated := 0
		for i := 1; i < len(toks); i++ {
			prev := toks[i-1]
			if strings.HasSuffix(prev, ".") || strings.HasSuffix(prev, ":") ||
				strings.HasSuffix(prev, "?") || strings.HasSuffix(prev, "-") {
				continue
			}
			t := strings.Trim(toks[i], `"'(),.:;?!`)
			if t == "" || !structure1reAlphaOnly.MatchString(t) {
				continue
			}
			if structure1funcWords[t] {
				funcWord = true
			}
			if len(t) > 1 && t == strings.ToUpper(t) {
				continue
			}
			if unicode.IsUpper(rune(t[0])) {
				capped++
			} else {
				allCapped = false
			}
			if structure1reSuspect.MatchString(t) && !structure1properNouns[t] && lower[strings.ToLower(t)] {
				corroborated++
			}
		}
		if !funcWord && (len(toks) < 4 || !allCapped || capped < 2 || corroborated < 2) {
			continue
		}
		out = append(out, types.Violation{
			RuleID:      "heading-sentence-case",
			StartIndex:  h.textStart,
			EndIndex:    h.textEnd,
			MatchedText: text[h.textStart:h.textEnd],
			Explanation: "use sentence case",
		})
	}
	return out
}

type structure1item struct {
	start   int
	end     int
	text    string
	ordered bool
	hasSub  bool
}

var (
	structure1reDotAlnum = regexp.MustCompile(`\.[A-Za-z0-9]`)
	structure1reTaskBox  = regexp.MustCompile(`^\[[ xX]\][ \t]*`)
)

// DetectGoogleInconsistentListPunctuation reports a list whose items disagree
// about capitalization or end punctuation.
func DetectGoogleInconsistentListPunctuation(text string) []types.Violation {
	lines := structure1Lines(text)
	var out []types.Violation
	var cur []structure1item
	curIndent := -1
	blank := false
	flush := func() {
		out = append(out, structure1listGroup(text, cur)...)
		cur = nil
		curIndent = -1
	}
	for _, ln := range lines {
		if ln.code {
			if len(cur) > 0 {
				cur[len(cur)-1].hasSub = true
			}
			flush()
			blank = true
			continue
		}
		if strings.TrimSpace(ln.text) == "" {
			blank = true
			continue
		}
		ind := structure1indent(ln.text)
		m := structure1reListItem.FindStringSubmatchIndex(ln.text)
		if m == nil {
			if len(cur) > 0 && ind > curIndent && !blank {
				it := &cur[len(cur)-1]
				it.end = ln.start + len(strings.TrimRight(ln.text, " \t\r"))
				it.text += " " + strings.TrimSpace(ln.text)
			} else {
				flush()
			}
			blank = false
			continue
		}
		if len(cur) > 0 && ind > curIndent {
			cur[len(cur)-1].hasSub = true
			blank = false
			continue
		}
		ordered := ln.text[m[4]] >= '0' && ln.text[m[4]] <= '9'
		if len(cur) > 0 && (ind < curIndent || cur[0].ordered != ordered) {
			flush()
		}
		if len(cur) == 0 {
			curIndent = ind
		}
		body := strings.TrimRight(ln.text[m[6]:m[7]], " \t\r")
		cur = append(cur, structure1item{
			start:   ln.start + m[6],
			end:     ln.start + m[6] + len(body),
			text:    body,
			ordered: ordered,
		})
		blank = false
	}
	flush()
	return out
}

func structure1listGroup(text string, items []structure1item) []types.Violation {
	if len(items) < 3 {
		return nil
	}
	codeFirst := 0
	for _, it := range items {
		if strings.HasPrefix(structure1bareText(it.text), "`") {
			codeFirst++
		}
	}
	if codeFirst*2 >= len(items) {
		return nil
	}
	var usable []structure1item
	for _, it := range items {
		if it.hasSub && strings.HasSuffix(it.text, ":") {
			continue
		}
		if strings.HasPrefix(it.text, "```") || structure1skipItem(it.text) {
			continue
		}
		usable = append(usable, it)
	}
	if len(usable) < 3 {
		return nil
	}
	ends := make([]bool, len(usable))
	ups := make([]bool, len(usable))
	continued := 0
	for i, it := range usable {
		s := strings.TrimRight(structure1stripInline(it.text), " \t")
		closed := strings.TrimRight(s, "\"'`)]}’”")
		if closed != "" && strings.IndexByte(",;", closed[len(closed)-1]) >= 0 {
			continued++
		}
		ends[i] = closed != "" && strings.IndexByte(".?!", closed[len(closed)-1]) >= 0
		ups[i] = structure1firstAlphaUpper(s)
	}
	dims := [][]bool{ends, ups}
	if continued >= 2 {
		dims = [][]bool{ups}
	}
	flagged := map[int]bool{}
	for _, flags := range dims {
		yes, no := 0, 0
		for _, b := range flags {
			if b {
				yes++
			} else {
				no++
			}
		}
		if yes == 0 || no == 0 {
			continue
		}
		var want bool
		switch {
		case yes < no:
			want = true
		case no < yes:
			want = false
		default:
			want = !flags[0]
		}
		for i, b := range flags {
			if b == want {
				flagged[i] = true
			}
		}
	}
	idx := make([]int, 0, len(flagged))
	for i := range flagged {
		idx = append(idx, i)
	}
	sort.Ints(idx)
	out := make([]types.Violation, 0, len(idx))
	for _, i := range idx {
		out = append(out, types.Violation{
			RuleID:      "inconsistent-list-punctuation",
			StartIndex:  usable[i].start,
			EndIndex:    usable[i].end,
			MatchedText: text[usable[i].start:usable[i].end],
			Explanation: "inconsistent with the rest of the list",
		})
	}
	return out
}

func structure1bareText(s string) string {
	s = strings.TrimLeft(s, "*_ \t")
	if m := structure1reTaskBox.FindString(s); m != "" {
		s = strings.TrimLeft(s[len(m):], "*_ \t")
	}
	return s
}

func structure1skipItem(s string) bool {
	f := strings.Fields(structure1bareText(s))
	if len(f) == 0 {
		return true
	}
	t := f[0]
	switch {
	case strings.HasPrefix(t, "`"), strings.HasPrefix(t, "<"),
		strings.HasPrefix(t, "http://"), strings.HasPrefix(t, "https://"):
		return true
	case t[0] >= '0' && t[0] <= '9':
		return true
	case strings.ContainsAny(t, "_/-"):
		return true
	}
	return structure1reDotAlnum.MatchString(t)
}

func structure1firstAlphaUpper(s string) bool {
	for _, r := range s {
		if unicode.IsLetter(r) {
			return unicode.IsUpper(r)
		}
	}
	return false
}

var structure1introStop = map[string]bool{
	"a": true, "an": true, "the": true, "this": true, "that": true,
	"these": true, "those": true, "in": true, "on": true, "of": true,
	"to": true, "for": true, "with": true, "and": true, "or": true,
	"you": true, "your": true, "we": true, "will": true, "how": true,
	"section": true, "page": true, "document": true, "tutorial": true,
	"following": true, "describes": true, "describe": true, "shows": true,
	"show": true, "explains": true, "explain": true, "complete": true,
	"tasks": true,
}

func structure1stem(w string) string {
	switch {
	case len(w) > 5 && strings.HasSuffix(w, "ing"):
		w = w[:len(w)-3]
	case len(w) > 3 && strings.HasSuffix(w, "s") && !strings.HasSuffix(w, "ss"):
		w = w[:len(w)-1]
	}
	if len(w) > 4 && strings.HasSuffix(w, "e") {
		w = w[:len(w)-1]
	}
	return w
}

func structure1introTokens(s string) map[string]bool {
	out := map[string]bool{}
	for _, w := range structure1words(structure1stripInline(s)) {
		if structure1introStop[w] {
			continue
		}
		st := structure1stem(w)
		if structure1introStop[st] {
			continue
		}
		out[st] = true
	}
	return out
}

// DetectGoogleIntroRestatesHeading reports a section's first sentence when it
// only repeats the heading the reader just read.
func DetectGoogleIntroRestatesHeading(text string) []types.Violation {
	hs := structure1Headings(text)
	var out []types.Violation
	for i, h := range hs {
		bodyEnd := len(text)
		if i+1 < len(hs) {
			bodyEnd = hs[i+1].lineStart
		}
		if h.lineEnd >= bodyEnd {
			continue
		}
		paras := splitParagraphs(text[h.lineEnd:bodyEnd])
		if len(paras) == 0 {
			continue
		}
		p := paras[0]
		trimmed := strings.TrimSpace(p.text)
		if isListy(p.text) || structure1reListItem.MatchString(trimmed) ||
			structure1reQuote.MatchString(trimmed) || isTabular(trimmed) ||
			strings.HasPrefix(trimmed, "```") || strings.HasPrefix(trimmed, "~~~") ||
			strings.HasPrefix(trimmed, "<") || structure1reATX.MatchString(trimmed) {
			continue
		}
		sentences := mergeAbbrev(splitSentences(p.text))
		if len(sentences) == 0 {
			continue
		}
		first := sentences[0]
		head := structure1introTokens(h.text)
		sent := structure1introTokens(first)
		hollow := len(sent) == 0 && len(strings.Fields(first)) >= 4
		extra := 0
		for k := range sent {
			if !head[k] {
				extra++
			}
		}
		maxExtra := 3
		if len(head) < 2 {
			maxExtra = 0
		}
		restates := len(head) > 0 && structure1subset(head, sent) && extra <= maxExtra
		similar := len(head) >= 2 && structure1jaccard(head, sent) >= 0.7
		if !hollow && !restates && !similar {
			continue
		}
		start := h.lineEnd + p.start
		s, e := trimSpan(text, start, start+len(first))
		out = append(out, types.Violation{
			RuleID:      "intro-restates-heading",
			StartIndex:  s,
			EndIndex:    e,
			MatchedText: text[s:e],
		})
	}
	return out
}
