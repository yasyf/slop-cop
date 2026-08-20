package detectors

import (
	"regexp"
	"strings"

	"github.com/yasyf/slop-cop/internal/types"
)

type punct1line struct {
	text  string
	start int
}

func punct1splitLines(text string) []punct1line {
	out := make([]punct1line, 0, strings.Count(text, "\n")+1)
	offset := 0
	for {
		i := strings.IndexByte(text[offset:], '\n')
		if i < 0 {
			return append(out, punct1line{text: text[offset:], start: offset})
		}
		out = append(out, punct1line{text: text[offset : offset+i], start: offset})
		offset += i + 1
	}
}

func punct1set(words string) map[string]bool {
	out := make(map[string]bool)
	for _, w := range strings.Fields(words) {
		out[w] = true
	}
	return out
}

func punct1violation(text, ruleID string, start, end int, to string) types.Violation {
	return types.Violation{
		RuleID:          ruleID,
		StartIndex:      start,
		EndIndex:        end,
		MatchedText:     text[start:end],
		SuggestedChange: to,
	}
}

func punct1isSentenceStart(text string, at int) bool {
	j := at - 1
	for j >= 0 && (text[j] == ' ' || text[j] == '\t') {
		j--
	}
	if j < 0 {
		return true
	}
	switch text[j] {
	case '\n', '\r', '.', '!', '?', ':', '>', '#', '*', '-', '|', '(', '"':
		return true
	}
	return false
}

func punct1sentenceAround(text string, at int) string {
	start := strings.LastIndexAny(text[:at], ".!?\n") + 1
	end := strings.IndexAny(text[at:], ".!?\n")
	if end < 0 {
		return text[start:]
	}
	return text[start : at+end]
}

func punct1clauseBefore(text string, at int) string {
	return text[strings.LastIndexAny(text[:at], ".!?;\n")+1 : at]
}

func punct1clauseAfter(text string, from int) string {
	if i := strings.IndexAny(text[from:], ".!?;\n"); i >= 0 {
		return text[from : from+i]
	}
	return text[from:]
}

var punct1reWordSplit = regexp.MustCompile(`[^A-Za-z']+`)

var punct1finiteVerbs = punct1set(`
	is are was were am be been has have had do does did
	can could will would shall should may might must
	returns requires uses sends fails applies becomes remains includes
	supports allows causes creates provides expects rejects retries starts
	stops continues exists contains takes makes adds removes throws appears
	depends happens occurs refers gives tries lets tells finds knows seems
	comes goes wants treats handles enables disables prevents ensures
	accepts ignores deletes fetches parses validates raises emits
`)

func punct1hasFiniteVerb(s string) bool {
	for _, w := range punct1reWordSplit.Split(strings.ToLower(s), -1) {
		if punct1finiteVerbs[w] {
			return true
		}
	}
	return false
}

var (
	punct1reAcronymDots  = regexp.MustCompile(`\b(?:[A-Z]\.){2,}`)
	punct1reClippedWord  = regexp.MustCompile(`(?i)\b(app|demo|sync|info|spec|repo|admin|config|blog|intro|prod|dev|ops|lab)\.[ \t]+[a-z]`)
	punct1reNeedsPeriod  = regexp.MustCompile(`\b([Ee]tc|[Aa]pprox)(?:[^.A-Za-z]|$)`)
	punct1reNumberedAbbr = regexp.MustCompile(`\b(Fig|Vol|Sec)[ \t]+\d`)
)

// DetectGoogleAcronymPeriods reports periods threaded through an acronym and
// clipped words that drop or misplace their truncation period.
func DetectGoogleAcronymPeriods(text string) []types.Violation {
	const rule = "acronym-periods"
	var out []types.Violation
	for _, idx := range punct1reAcronymDots.FindAllStringIndex(text, -1) {
		out = append(out, punct1violation(text, rule, idx[0], idx[1],
			strings.ReplaceAll(text[idx[0]:idx[1]], ".", "")))
	}
	for _, idx := range punct1reClippedWord.FindAllStringSubmatchIndex(text, -1) {
		out = append(out, punct1violation(text, rule, idx[2], idx[3]+1, ""))
	}
	for _, re := range []*regexp.Regexp{punct1reNeedsPeriod, punct1reNumberedAbbr} {
		for _, idx := range re.FindAllStringSubmatchIndex(text, -1) {
			out = append(out, punct1violation(text, rule, idx[2], idx[3], text[idx[2]:idx[3]]+"."))
		}
	}
	return out
}

var punct1reAndOr = regexp.MustCompile(`(?i)\band\s*/\s*or\b`)

// DetectGoogleAndOrSlash reports the and/or construction.
func DetectGoogleAndOrSlash(text string) []types.Violation {
	out := findAll(text, punct1reAndOr, "and-or-slash")
	for i := range out {
		out[i].SuggestedChange = "and"
	}
	return out
}

var (
	punct1readOnlyNouns      = strings.Fields(`access mode file files disk volume replica permissions property field view`)
	punct1loadBalancingNouns = strings.Fields(`solution service services configuration algorithm rule rules policy scheme setup layer tier component backend`)
)

func punct1hyphenBefore(open, hyphenated string, nouns []string) []plainSub {
	out := make([]plainSub, 0, len(nouns))
	for _, n := range nouns {
		out = append(out, plainSub{open + " " + n, hyphenated + " " + n})
	}
	return out
}

var punct1compoundBase = []plainSub{
	{"back end", "backend"},
	{"code base", "codebase"},
	{"check box", "checkbox"},
	{"co-locate", "colocate"},
	{"auto-scaling", "autoscaling"},
	{"auto-healing", "autohealing"},
	{"click-through", "clickthrough"},
	{"appendices", "appendixes"},
	{"big endian", "big-endian"},
	{"little endian", "little-endian"},
	{"blue/green", "blue-green"},
	{"right click", "right-click"},
	{"a/b testing", "A/B testing"},
	{"datacenter", "data center"},
	{"data cleansing", "data cleaning"},
	{"datasource", "data source"},
	{"data store", "datastore"},
	{"datatype", "data type"},
	{"e-commerce", "ecommerce"},
	{"e-mail", "email"},
	{"end point", "endpoint"},
	{"file name", "filename"},
	{"filesystem", "file system"},
	{"front-end", "frontend"},
	{"health care", "healthcare"},
	{"host name", "hostname"},
	{"homescreen", "home screen"},
	{"hard-coded", "hardcoded"},
	{"doc set", "documentation set"},
	{"gen AI", "generative AI"},
	{"deep-linking", "deep linking"},
	{"error prone", "error-prone"},
	{"double tap", "double-tap"},
	{"distributed denial of service", "distributed denial-of-service"},
	{"high availability cluster", "high-availability cluster"},
	{"inter-cluster", "intercluster"},
	{"life cycle", "lifecycle"},
	{"life-cycle", "lifecycle"},
	{"live stream", "livestream"},
	{"micro-services", "microservices"},
	{"keyring", "key ring"},
	{"key/value pair", "key-value pair"},
	{"lockscreen", "lock screen"},
	{"long running operation", "long-running operation"},
	{"multicluster", "multi-cluster"},
	{"multiregion", "multi-region"},
	{"multitenancy", "multi-tenancy"},
	{"KB/s", "KBps"},
	{"MB/s", "MBps"},
	{"I-O", "I/O"},
	{"nameserver", "name server"},
	{"name space", "namespace"},
	{"No-SQL", "NoSQL"},
	{"OAuth2", "OAuth 2.0"},
	{"on-premise", "on-premises"},
	{"pre-built", "prebuilt"},
	{"pre-emptible", "preemptible"},
	{"preexisting", "pre-existing"},
	{"pre-recorded", "prerecorded"},
	{"preshared key", "pre-shared key"},
	{"pre-submit", "presubmit"},
	{"resource recordset", "resource record set"},
	{"run book", "runbook"},
	{"peer zone", "peering zone"},
	{"nonkey", "non-key"},
	{"pathname", "path"},
	{"at runtime", "at run time"},
	{"plug-in", "plugin"},
	{"screen shot", "screenshot"},
	{"statusbar", "status bar"},
	{"subcommand", "sub-command"},
	{"sub-tree", "subtree"},
	{"sub-zone", "subzone"},
	{"time stamp", "timestamp"},
	{"time frame", "timeframe"},
	{"tool kit", "toolkit"},
	{"touch screen", "touchscreen"},
	{"transcompile", "transpile"},
	{"trade-off", "tradeoff"},
	{"work-around", "workaround"},
	{"web page", "webpage"},
	{"userbase", "user base"},
	{"walk-through", "walkthrough"},
	{"webserver", "web server"},
	{"white space", "whitespace"},
	{"wild card", "wildcard"},
	{"white paper", "whitepaper"},
	{"Unix like", "Unix-like"},
	{"UTF8", "UTF-8"},
	{"World Wide Web", "web"},
	{"SHA1", "SHA-1"},
	{"time-to-live", "time to live (TTL)"},
	{"3rd party", "third party"},
	{"third party API", "third-party API"},
	{"time zone offset", "time-zone offset"},
}

var punct1compoundSubs = func() []plainSub {
	out := make([]plainSub, 0, len(punct1compoundBase)+len(punct1readOnlyNouns)+len(punct1loadBalancingNouns))
	out = append(out, punct1compoundBase...)
	out = append(out, punct1hyphenBefore("read only", "read-only", punct1readOnlyNouns)...)
	out = append(out, punct1hyphenBefore("load balancing", "load-balancing", punct1loadBalancingNouns)...)
	return out
}()

var punct1compoundCaseSubs = []plainSub{
	{"cURL", "curl"},
	{"HTTPs", "HTTPS"},
	{"FinTech", "fintech"},
	{"UNICODE", "Unicode"},
	{"IPSec", "IPsec"},
	{"Web Application Firewall", "web application firewall"},
}

func punct1compileCaseSubs(subs []plainSub) []*regexp.Regexp {
	out := make([]*regexp.Regexp, 0, len(subs))
	for _, s := range subs {
		out = append(out, regexp.MustCompile(`\b`+
			strings.ReplaceAll(escapeForRegex(s.from), " ", phraseSep)+`\b`))
	}
	return out
}

var (
	punct1compoundRes     = compileSubs(punct1compoundSubs)
	punct1compoundCaseRes = punct1compileCaseSubs(punct1compoundCaseSubs)
)

var (
	punct1reFilePath        = regexp.MustCompile(`(?i)\bfilepath\b`)
	punct1reInLine          = regexp.MustCompile(`(?i)(\bin-line\b)[ \t]*([A-Za-z]*)`)
	punct1rePlural          = regexp.MustCompile(`(?i)\b(indices|matrices)\b`)
	punct1reMathDoc         = regexp.MustCompile(`(?i)\b(eigen\w*|tensors?|determinant|scalars?|transpose|linear algebra|matrix multiplication|portfolio|equity|equities|index fund|stock index|covariance)\b`)
	punct1rePoP             = regexp.MustCompile(`\bPOP\b[ \t]+(?:location|locations|site|sites|region|regions|edge|edges)\b`)
	punct1rePD              = regexp.MustCompile(`\bPD\b`)
	punct1reDiskWord        = regexp.MustCompile(`(?i)\b(disks?|storage|volumes?|snapshots?)\b`)
	punct1reInternet        = regexp.MustCompile(`\bInternet\b`)
	punct1reNextCapital     = regexp.MustCompile(`^[ \t]+[A-Z]`)
	punct1reInternetThings  = regexp.MustCompile(`^[ \t]+of[ \t]+Things\b`)
	punct1reMeridiemPeriods = regexp.MustCompile(`(?i)\ba\.m\./p\.m\.`)
	punct1reRFCNumber       = regexp.MustCompile(`\bRFC(\d{3,5})\b`)
	punct1reBareID          = regexp.MustCompile(`\bId\b`)
)

// punct1identifierNeighbour reports whether [start, end) is welded into a
// larger identifier such as "Claude-Session-Id" or "time.RFC3339", which a
// word boundary alone does not exclude.
func punct1identifierNeighbour(text string, start, end int) bool {
	if start > 0 {
		switch text[start-1] {
		case '-', '_', '.', '/':
			return true
		}
	}
	if end < len(text) {
		switch text[end] {
		case '-', '_', '.', '/':
			return true
		}
	}
	return false
}

func punct1guardedCompounds(text string) []types.Violation {
	const rule = "compound-spelling"
	var out []types.Violation

	for _, idx := range punct1reFilePath.FindAllStringIndex(text, -1) {
		if idx[1] < len(text) {
			switch text[idx[1]] {
			case '.', '/', '(':
				continue
			}
		}
		if idx[0] > 0 && (text[idx[0]-1] == '.' || text[idx[0]-1] == '/') {
			continue
		}
		out = append(out, punct1violation(text, rule, idx[0], idx[1], "path"))
	}

	for _, idx := range punct1reInLine.FindAllStringSubmatchIndex(text, -1) {
		if strings.EqualFold(text[idx[4]:idx[5]], "with") {
			continue
		}
		out = append(out, punct1violation(text, rule, idx[2], idx[3], "inline"))
	}

	if !punct1reMathDoc.MatchString(text) {
		for _, idx := range punct1rePlural.FindAllStringSubmatchIndex(text, -1) {
			to := "indexes"
			if strings.EqualFold(text[idx[2]:idx[3]], "matrices") {
				to = "matrixes"
			}
			out = append(out, punct1violation(text, rule, idx[2], idx[3], to))
		}
	}

	for _, idx := range punct1rePoP.FindAllStringIndex(text, -1) {
		out = append(out, punct1violation(text, rule, idx[0], idx[0]+3, "PoP"))
	}

	for _, idx := range punct1rePD.FindAllStringIndex(text, -1) {
		if !punct1reDiskWord.MatchString(punct1sentenceAround(text, idx[0])) {
			continue
		}
		out = append(out, punct1violation(text, rule, idx[0], idx[1], "persistent disk"))
	}

	for _, idx := range punct1reInternet.FindAllStringIndex(text, -1) {
		if punct1isSentenceStart(text, idx[0]) {
			continue
		}
		rest := text[idx[1]:]
		if punct1reNextCapital.MatchString(rest) || punct1reInternetThings.MatchString(rest) {
			continue
		}
		out = append(out, punct1violation(text, rule, idx[0], idx[1], "internet"))
	}

	for _, idx := range punct1reMeridiemPeriods.FindAllStringIndex(text, -1) {
		out = append(out, punct1violation(text, rule, idx[0], idx[1], "AM/PM"))
	}

	for _, idx := range punct1reRFCNumber.FindAllStringSubmatchIndex(text, -1) {
		if punct1identifierNeighbour(text, idx[0], idx[1]) {
			continue
		}
		out = append(out, punct1violation(text, rule, idx[0], idx[1], "RFC "+text[idx[2]:idx[3]]))
	}

	for _, idx := range punct1reBareID.FindAllStringIndex(text, -1) {
		if punct1identifierNeighbour(text, idx[0], idx[1]) {
			continue
		}
		out = append(out, punct1violation(text, rule, idx[0], idx[1], "ID"))
	}

	return out
}

// DetectGoogleCompoundSpelling reports technical compounds written in a
// spelling other than the settled one.
func DetectGoogleCompoundSpelling(text string) []types.Violation {
	const rule = "compound-spelling"
	out := findSubs(text, rule, punct1compoundSubs, punct1compoundRes)
	out = append(out, findSubs(text, rule, punct1compoundCaseSubs, punct1compoundCaseRes)...)
	return append(out, punct1guardedCompounds(text)...)
}

var (
	punct1reSpliceAdverb = regexp.MustCompile(`(?i),\s+(however|therefore|otherwise|moreover|furthermore|nevertheless|nonetheless|consequently|thus|hence|meanwhile)\s*,`)
	punct1reRunOnAdverb  = regexp.MustCompile(`(?i)[a-z](\s+)(however|therefore|otherwise|nevertheless|consequently)\s+[a-z]`)
	punct1reUnclosedTurn = regexp.MustCompile(`(?i)[;\x{2014}]\s*(however|therefore|otherwise|moreover|furthermore|nevertheless|nonetheless|consequently|meanwhile|that\s+is|for\s+example|for\s+instance|in\s+other\s+words)\s+[^,\s]`)
)

// DetectGoogleConjunctiveAdverbPunctuation reports a conjunctive adverb joined
// to its clauses with a comma splice, no break at all, or no trailing comma.
func DetectGoogleConjunctiveAdverbPunctuation(text string) []types.Violation {
	const rule = "conjunctive-adverb-punctuation"
	var out []types.Violation

	for _, idx := range punct1reSpliceAdverb.FindAllStringSubmatchIndex(text, -1) {
		before := punct1clauseBefore(text, idx[0])
		if len(strings.Fields(before)) < 3 || !punct1hasFiniteVerb(before) {
			continue
		}
		if len(strings.Fields(punct1clauseAfter(text, idx[1]))) < 3 {
			continue
		}
		out = append(out, punct1violation(text, rule, idx[0], idx[1],
			"; "+text[idx[2]:idx[3]]+","))
	}

	for _, idx := range punct1reRunOnAdverb.FindAllStringSubmatchIndex(text, -1) {
		if !punct1hasFiniteVerb(punct1clauseBefore(text, idx[2])) {
			continue
		}
		if !punct1hasFiniteVerb(punct1clauseAfter(text, idx[5])) {
			continue
		}
		out = append(out, punct1violation(text, rule, idx[2], idx[5],
			"; "+text[idx[4]:idx[5]]+","))
	}

	for _, idx := range punct1reUnclosedTurn.FindAllStringSubmatchIndex(text, -1) {
		out = append(out, punct1violation(text, rule, idx[2], idx[3], text[idx[2]:idx[3]]+","))
	}

	return out
}

var punct1reEllipsis = regexp.MustCompile(`\.{3,4}`)

func punct1isFenceLine(trimmed string) bool {
	return strings.HasPrefix(trimmed, "```") || strings.HasPrefix(trimmed, "~~~")
}

func punct1oddCountBefore(s string, b byte) bool {
	return strings.Count(s, string(b))%2 == 1
}

func punct1skipEllipsisLine(line, trimmed string) bool {
	if strings.HasPrefix(trimmed, ">") || strings.HasPrefix(trimmed, "$ ") {
		return true
	}
	return strings.HasPrefix(line, "    ") || strings.HasPrefix(line, "\t")
}

// DetectGoogleEllipsisInProse reports an ellipsis standing in for content the
// sentence should supply.
func DetectGoogleEllipsisInProse(text string) []types.Violation {
	const rule = "ellipsis-in-prose"
	var out []types.Violation
	inFence := false
	for _, ln := range punct1splitLines(text) {
		trimmed := strings.TrimLeft(ln.text, " \t")
		if punct1isFenceLine(trimmed) {
			inFence = !inFence
			continue
		}
		if inFence || punct1skipEllipsisLine(ln.text, trimmed) {
			continue
		}
		for _, idx := range punct1reEllipsis.FindAllStringIndex(ln.text, -1) {
			if idx[0] == 0 || (ln.text[idx[0]-1] != ' ' && ln.text[idx[0]-1] != '\t') {
				continue
			}
			head := ln.text[:idx[0]]
			if punct1oddCountBefore(head, '`') || punct1oddCountBefore(head, '"') {
				continue
			}
			start, end := idx[0], idx[1]
			for start > 0 && (ln.text[start-1] == ' ' || ln.text[start-1] == '\t') {
				start--
			}
			for end < len(ln.text) && (ln.text[end] == ' ' || ln.text[end] == '\t') {
				end++
			}
			out = append(out, punct1violation(text, rule, ln.start+start, ln.start+end, ""))
		}
	}
	return out
}

var (
	punct1reATXPrefix     = regexp.MustCompile(`^[ \t]{0,3}#{1,6}[ \t]+`)
	punct1reATXSuffix     = regexp.MustCompile(`[ \t]+#+[ \t]*$`)
	punct1reSetextRule    = regexp.MustCompile(`^[ \t]{0,3}(?:={2,}|-{2,})[ \t]*$`)
	punct1reHTMLHeading   = regexp.MustCompile(`(?is)<h[1-6][^>]*>([^<]*\.)</h[1-6]>`)
	punct1headingAbbrevs  = punct1set(`etc. inc. ltd. co. jr. sr. al. vs. approx. fig. no.`)
	punct1reHeadingSkipLn = regexp.MustCompile(`^[ \t]{0,6}(?:[-*+>|]|\d+[.)])[ \t]`)
)

func punct1headingPeriod(text string, start, end int) (int, bool) {
	trimmed := strings.TrimRight(text[start:end], " \t\r")
	if len(trimmed) < 2 || !strings.HasSuffix(trimmed, ".") {
		return 0, false
	}
	switch trimmed[len(trimmed)-2] {
	case '.', '!', '?', ':':
		return 0, false
	}
	fields := strings.Fields(trimmed)
	if len(fields) < 2 {
		return 0, false
	}
	last := fields[len(fields)-1]
	if punct1headingAbbrevs[strings.ToLower(last)] {
		return 0, false
	}
	if len(last) == 2 && last[0] >= 'A' && last[0] <= 'Z' {
		return 0, false
	}
	return start + len(trimmed) - 1, true
}

// DetectGoogleHeadingTerminalPeriod reports a heading closed with a period.
func DetectGoogleHeadingTerminalPeriod(text string) []types.Violation {
	const rule = "heading-terminal-period"
	var out []types.Violation
	lines := punct1splitLines(text)
	inFence := false
	for i, ln := range lines {
		if punct1isFenceLine(strings.TrimLeft(ln.text, " \t")) {
			inFence = !inFence
			continue
		}
		if inFence {
			continue
		}
		bodyStart, bodyEnd := -1, -1
		if prefix := punct1reATXPrefix.FindString(ln.text); prefix != "" {
			bodyStart = ln.start + len(prefix)
			bodyEnd = ln.start + len(ln.text)
			if suffix := punct1reATXSuffix.FindStringIndex(ln.text); suffix != nil {
				bodyEnd = ln.start + suffix[0]
			}
		} else if i+1 < len(lines) && punct1reSetextRule.MatchString(lines[i+1].text) &&
			strings.TrimSpace(ln.text) != "" && !punct1reHeadingSkipLn.MatchString(ln.text) &&
			(i == 0 || strings.TrimSpace(lines[i-1].text) == "") {
			bodyStart = ln.start
			bodyEnd = ln.start + len(ln.text)
		}
		if bodyStart < 0 || bodyEnd <= bodyStart {
			continue
		}
		if at, ok := punct1headingPeriod(text, bodyStart, bodyEnd); ok {
			out = append(out, punct1violation(text, rule, at, at+1, ""))
		}
	}
	for _, idx := range punct1reHTMLHeading.FindAllStringSubmatchIndex(text, -1) {
		if at, ok := punct1headingPeriod(text, idx[2], idx[3]); ok {
			out = append(out, punct1violation(text, rule, at, at+1, ""))
		}
	}
	return out
}

var (
	punct1reIntroOpener = regexp.MustCompile(`(?m)(?:^|[.!?]["\x{201D}\x{2019}]?\s+)(However|Therefore|Otherwise|Moreover|Furthermore|Nevertheless|Nonetheless|Consequently|Meanwhile|Instead|Alternatively|Additionally|Similarly|Finally|Typically|Optionally|For[ \t]+example|For[ \t]+instance|In[ \t]+general|In[ \t]+addition|In[ \t]+this[ \t]+case|As[ \t]+a[ \t]+result|By[ \t]+default|On[ \t]+the[ \t]+other[ \t]+hand|In[ \t]+contrast)\s+([A-Za-z]+)`)
	punct1introBlockers = punct1set(`of to from with that than`)
	punct1frontedHeads  = punct1set(`If When After Before Once Unless Although Though While Because To`)
)

func punct1frontedClauses(text string) []types.Violation {
	const rule = "intro-comma-missing"
	var out []types.Violation
	for _, p := range splitParagraphs(text) {
		if isListy(p.text) {
			continue
		}
		offset := p.start
		for _, s := range mergeAbbrev(splitSentences(p.text)) {
			start := offset
			offset += len(s)
			for start < offset && isSpaceByte(text[start]) {
				start++
			}
			line := spanLines(text, start, start)[0]
			if isATXHeading(line) || isTabular(line) {
				continue
			}
			fields := strings.Fields(s)
			if len(fields) < 8 || !punct1frontedHeads[fields[0]] {
				continue
			}
			window := fields
			if len(window) > 18 {
				window = window[:18]
			}
			if strings.ContainsAny(strings.Join(window, " "), ",;:") {
				continue
			}
			out = append(out, punct1violation(text, rule, start, start+len(fields[0]), ""))
		}
	}
	return out
}

// DetectGoogleIntroCommaMissing reports an introductory word, phrase, or
// fronted subordinate clause that runs into the main clause without a comma.
func DetectGoogleIntroCommaMissing(text string) []types.Violation {
	const rule = "intro-comma-missing"
	var out []types.Violation
	for _, idx := range punct1reIntroOpener.FindAllStringSubmatchIndex(text, -1) {
		next := text[idx[4]:idx[5]]
		if punct1introBlockers[strings.ToLower(next)] {
			continue
		}
		if strings.HasSuffix(next, "ed") || strings.HasSuffix(next, "ing") {
			continue
		}
		opener := text[idx[2]:idx[3]]
		if isATXHeading(spanLines(text, idx[2], idx[2])[0]) {
			continue
		}
		out = append(out, punct1violation(text, rule, idx[2], idx[3], opener+","))
	}
	return append(out, punct1frontedClauses(text)...)
}

var (
	punct1reParen        = regexp.MustCompile(`\([^()]*\)`)
	punct1reParenExample = regexp.MustCompile(`(?is)^\((?:for example|for instance|e\.g\.),\s+(.*)\)$`)
	punct1reParenClause  = regexp.MustCompile(`[;?]\s`)
	punct1reParenCite    = regexp.MustCompile(`(?i)^(?:RFC|ISO|IEEE|ANSI|SOC)\s*\d`)
	punct1reParenCode    = regexp.MustCompile("^`[^`]*`$")
)

func punct1skipParen(inner string) bool {
	trimmed := strings.TrimSpace(inner)
	lower := strings.ToLower(trimmed)
	switch {
	case strings.HasPrefix(lower, "see "),
		strings.HasPrefix(lower, "see:"),
		strings.HasPrefix(lower, "refer to "),
		strings.HasPrefix(lower, "for more information"),
		strings.Contains(lower, ", see "):
		return true
	}
	return punct1reParenCite.MatchString(trimmed) || punct1reParenCode.MatchString(trimmed)
}

func punct1midSentence(text string, at int) bool {
	j := at - 1
	for j >= 0 && (text[j] == ' ' || text[j] == '\t') {
		j--
	}
	if j < 0 {
		return false
	}
	c := text[j]
	return c == ',' || c == '_' ||
		(c >= 'a' && c <= 'z') || (c >= 'A' && c <= 'Z') || (c >= '0' && c <= '9')
}

// DetectGoogleLongParenthetical reports a mid-sentence aside long enough to
// suspend the main clause, and an over-elaborated parenthesized example.
func DetectGoogleLongParenthetical(text string) []types.Violation {
	const rule = "long-parenthetical"
	var out []types.Violation
	for _, idx := range punct1reParen.FindAllStringIndex(text, -1) {
		inner := text[idx[0]+1 : idx[1]-1]
		if punct1skipParen(inner) {
			continue
		}
		hit := false
		if m := punct1reParenExample.FindStringSubmatch(text[idx[0]:idx[1]]); m != nil {
			rest := strings.TrimSpace(m[1])
			hit = strings.Contains(rest, ",") || len(strings.Fields(rest)) > 8
		}
		if !hit && punct1midSentence(text, idx[0]) {
			hit = len(inner) >= 80 ||
				(len(inner) >= 40 && punct1reParenClause.MatchString(inner))
		}
		if hit {
			out = append(out, punct1violation(text, rule, idx[0], idx[1], ""))
		}
	}
	return out
}

var (
	punct1reLyHyphen  = regexp.MustCompile(`(?i)\b([A-Za-z]{2,}ly)-[A-Za-z]`)
	punct1lyNonAdverb = punct1set(`
		family supply assembly anomaly apply reply imply comply multiply
		monopoly poly only early holy ugly silly jelly rally ally italy july
		ply panoply friendly daily weekly monthly quarterly yearly hourly
		nightly likely lonely lovely timely orderly elderly costly deadly
		curly surly worldly scholarly unruly bully folly gully rely dolly
		belly medley
	`)
)

// DetectGoogleLyAdverbHyphen reports a hyphen joining an -ly adverb to the
// word it modifies.
func DetectGoogleLyAdverbHyphen(text string) []types.Violation {
	const rule = "ly-adverb-hyphen"
	var out []types.Violation
	for _, idx := range punct1reLyHyphen.FindAllStringSubmatchIndex(text, -1) {
		if punct1lyNonAdverb[strings.ToLower(text[idx[2]:idx[3]])] {
			continue
		}
		out = append(out, punct1violation(text, rule, idx[3], idx[3]+1, " "))
	}
	return out
}
