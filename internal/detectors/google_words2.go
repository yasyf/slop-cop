package detectors

import (
	"regexp"
	"strings"

	"github.com/yasyf/slop-cop/internal/types"
)

type words2sub struct {
	pat string
	to  string
}

func words2phrase(s string) string { return strings.ReplaceAll(s, "_", phraseSep) }

func words2compile(subs []words2sub) ([]plainSub, []*regexp.Regexp) {
	tbl := make([]plainSub, 0, len(subs))
	res := make([]*regexp.Regexp, 0, len(subs))
	for _, s := range subs {
		tbl = append(tbl, plainSub{from: s.pat, to: s.to})
		res = append(res, regexp.MustCompile(words2phrase(s.pat)))
	}
	return tbl, res
}

func words2set(words ...string) map[string]bool {
	m := make(map[string]bool, len(words))
	for _, w := range words {
		m[w] = true
	}
	return m
}

func words2isWordByte(b byte) bool {
	return b == '_' || (b >= '0' && b <= '9') || (b >= 'a' && b <= 'z') || (b >= 'A' && b <= 'Z')
}

func words2isSpaceByte(b byte) bool {
	return b == ' ' || b == '\t' || b == '\n' || b == '\r'
}

func words2prevWord(text string, pos int) string {
	i := pos
	for i > 0 && words2isSpaceByte(text[i-1]) {
		i--
	}
	end := i
	for i > 0 && words2isWordByte(text[i-1]) {
		i--
	}
	return text[i:end]
}

func words2nextWord(text string, pos int) string {
	i := pos
	for i < len(text) && words2isSpaceByte(text[i]) {
		i++
	}
	start := i
	for i < len(text) && words2isWordByte(text[i]) {
		i++
	}
	return text[start:i]
}

func words2applyCase(vs []types.Violation) []types.Violation {
	for i, v := range vs {
		if v.SuggestedChange == "" || v.MatchedText == "" {
			continue
		}
		if c := v.MatchedText[0]; c >= 'A' && c <= 'Z' {
			vs[i].SuggestedChange = strings.ToUpper(v.SuggestedChange[:1]) + v.SuggestedChange[1:]
		}
	}
	return vs
}

func words2findGroupSubs(text, ruleID string, subs []plainSub, res []*regexp.Regexp) []types.Violation {
	var out []types.Violation
	for i, re := range res {
		for _, m := range re.FindAllStringSubmatchIndex(text, -1) {
			if len(m) < 4 || m[2] < 0 {
				continue
			}
			out = append(out, types.Violation{
				RuleID:          ruleID,
				StartIndex:      m[2],
				EndIndex:        m[3],
				MatchedText:     text[m[2]:m[3]],
				SuggestedChange: subs[i].to,
			})
		}
	}
	return out
}

func words2singular(w string) string {
	low := strings.ToLower(w)
	switch {
	case len(w) > 3 && strings.HasSuffix(low, "ies"):
		return w[:len(w)-3] + "y"
	case len(w) > 4 && (strings.HasSuffix(low, "sses") || strings.HasSuffix(low, "shes") ||
		strings.HasSuffix(low, "ches") || strings.HasSuffix(low, "xes") || strings.HasSuffix(low, "zes")):
		return w[:len(w)-2]
	default:
		return w[:len(w)-1]
	}
}

var (
	words2reOneOrMore   = regexp.MustCompile(`(?i)\b` + words2phrase(`one_or_more_`) + `((?:[a-z][a-z-]*[ \t]+){0,2}[a-z][a-z-]*)\b`)
	words2reMoreThanOne = regexp.MustCompile(`(?i)\b([a-z][a-z'-]*)[ \t]+(` + words2phrase(`more_than_one_`) + `([a-z][a-z-]*))\b`)
)

var (
	words2irregularPlurals = words2set("data", "criteria", "children", "people", "indices", "matrices", "media", "schemata", "men", "women")
	words2oneOrMoreLead    = words2set(
		"of", "the", "a", "an", "and", "or", "to", "in", "for", "from", "with", "on", "at", "by", "than",
		"may", "might", "can", "could", "will", "would", "should", "must", "have", "has", "had",
		"do", "does", "did", "is", "are", "was", "were", "be", "been", "being", "if", "that", "these", "those", "such",
	)
	words2alwaysS = words2set(
		"access", "address", "analysis", "basis", "bus", "business", "class", "https", "process", "series", "status",
		"focus", "index", "lens", "news", "os", "dns", "ios", "gas", "plus", "versus", "thus", "this", "its",
		"is", "was", "has", "does", "less", "unless",
	)
	words2moreThanOneLead = words2set(
		"create", "creates", "creating", "add", "adds", "adding", "have", "has", "specify", "specifies",
		"use", "uses", "using", "select", "selects", "configure", "configures", "define", "defines",
		"provide", "provides", "set", "sets", "return", "returns", "contain", "contains", "include", "includes",
		"support", "supports", "pass", "passes", "attach", "attaches", "enter", "enters", "choose", "chooses",
		"install", "installs", "run", "runs", "make", "makes", "assign", "assigns", "delete", "deletes",
		"list", "lists", "allow", "allows", "require", "requires", "expect", "expects", "accept", "accepts",
		"upload", "uploads", "register", "registers", "declare", "declares", "of", "with", "in", "for", "to", "from", "into", "than",
	)
)

// DetectGoogleOneOrMoreAgreement reports "one or more" followed by a singular
// noun phrase, and "more than one" followed by a plural noun.
func DetectGoogleOneOrMoreAgreement(text string) []types.Violation {
	const rule = "one-or-more-agreement"
	var out []types.Violation
	for _, m := range words2reOneOrMore.FindAllStringSubmatchIndex(text, -1) {
		tokens := strings.Fields(text[m[2]:m[3]])
		if len(tokens) == 0 || words2oneOrMoreLead[strings.ToLower(tokens[0])] {
			continue
		}
		plural := false
		for _, t := range tokens {
			low := strings.ToLower(t)
			if strings.HasSuffix(low, "s") || words2irregularPlurals[low] {
				plural = true
				break
			}
		}
		if plural {
			continue
		}
		out = append(out, types.Violation{
			RuleID:      rule,
			StartIndex:  m[0],
			EndIndex:    m[1],
			MatchedText: text[m[0]:m[1]],
			Explanation: "one or more takes a plural noun and a plural verb.",
		})
	}
	for _, m := range words2reMoreThanOne.FindAllStringSubmatchIndex(text, -1) {
		if !words2moreThanOneLead[strings.ToLower(text[m[2]:m[3]])] {
			continue
		}
		token := text[m[6]:m[7]]
		low := strings.ToLower(token)
		if !strings.HasSuffix(low, "s") || words2alwaysS[low] {
			continue
		}
		out = append(out, types.Violation{
			RuleID:          rule,
			StartIndex:      m[4],
			EndIndex:        m[5],
			MatchedText:     text[m[4]:m[5]],
			Explanation:     "more than one takes a singular noun.",
			SuggestedChange: text[m[4]:m[6]] + words2singular(token),
		})
	}
	return out
}

var words2abbrevSubs, words2abbrevRes = words2buildAbbrev()

func words2buildAbbrev() ([]plainSub, []*regexp.Regexp) {
	pairs := []struct{ abbr, exp string }{
		{"PDF", `portable_document_format`},
		{"HTML", `hypertext_markup_language`},
		{"XML", `extensible_markup_language`},
		{"JSON", `javascript_object_notation`},
		{"API", `application_programming_interface`},
		{"URL", `uniform_resource_locator`},
		{"USB", `universal_serial_bus`},
		{"RAM", `random[-\s]access_memory`},
		{"PC", `personal_computer`},
		{"REST", `representational_state_transfer`},
		{"DVD", `digital_versatile_disc`},
		{"AI", `artificial_intelligence`},
		{"CSV", `comma[-\s]separated_values`},
	}
	subs := make([]plainSub, 0, len(pairs)*2)
	res := make([]*regexp.Regexp, 0, len(pairs)*2)
	for _, p := range pairs {
		exp := words2phrase(p.exp)
		subs = append(subs, plainSub{from: p.exp, to: p.abbr})
		res = append(res, regexp.MustCompile(`(?i)\b`+exp+`[ \t]*\((?-i:`+p.abbr+`)\)`))
		subs = append(subs, plainSub{from: p.abbr, to: p.abbr})
		res = append(res, regexp.MustCompile(`\b`+p.abbr+`[ \t]*\((?i:`+exp+`)\)`))
	}
	return subs, res
}

// DetectGoogleOverExplainedAbbreviation reports an abbreviation every reader
// already knows introduced alongside its spelled-out expansion.
func DetectGoogleOverExplainedAbbreviation(text string) []types.Violation {
	return findSubs(text, "over-explained-abbreviation", words2abbrevSubs, words2abbrevRes)
}

var (
	words2rePerDeterminer = regexp.MustCompile(`(?i)\b(?:as[ \t]+)?per[ \t]+(the|your|our|my|this|that|their|his|her)\b`)
	words2rePerProper     = regexp.MustCompile(`\b[Pp]er[ \t]+([A-Z][A-Za-z]+)\b`)
)

var (
	words2perRateNouns = words2set(
		"second", "seconds", "minute", "minutes", "hour", "hours", "day", "days", "week", "weeks",
		"month", "months", "year", "years", "request", "requests", "query", "queries", "node", "nodes",
		"user", "users", "operation", "operations", "call", "calls", "byte", "bytes", "row", "rows",
		"record", "records", "message", "messages", "item", "items", "page", "pages", "transaction",
		"transactions", "session", "sessions", "token", "tokens", "core", "cores", "cpu", "vcpu",
		"thread", "threads", "connection", "connections", "file", "files", "job", "jobs", "task", "tasks",
		"event", "events", "read", "reads", "write", "writes", "unit", "units", "capita", "percent",
	)
	words2perRateLead = words2set(
		"gb", "mb", "kb", "tb", "pb", "gib", "mib", "kib", "tib", "ms", "ns", "us", "hz", "khz", "mhz", "ghz",
		"mbps", "gbps", "kbps", "iops", "qps", "rps", "requests", "queries", "operations", "calls", "bytes",
		"reads", "writes", "transactions", "dollars", "cents", "characters", "tokens",
	)
)

func words2isUpperToken(s string) bool {
	for i := 0; i < len(s); i++ {
		if s[i] >= 'a' && s[i] <= 'z' {
			return false
		}
	}
	return true
}

func words2hasDigit(s string) bool {
	for i := 0; i < len(s); i++ {
		if s[i] >= '0' && s[i] <= '9' {
			return true
		}
	}
	return false
}

// DetectGooglePerOutsideRates reports "per" used to mean "according to" or
// "for each" rather than to state a rate.
func DetectGooglePerOutsideRates(text string) []types.Violation {
	const rule = "per-outside-rates"
	var out []types.Violation
	for _, m := range words2rePerDeterminer.FindAllStringSubmatchIndex(text, -1) {
		out = append(out, types.Violation{
			RuleID:          rule,
			StartIndex:      m[0],
			EndIndex:        m[1],
			MatchedText:     text[m[0]:m[1]],
			SuggestedChange: "according to " + text[m[2]:m[3]],
		})
	}
	for _, m := range words2rePerProper.FindAllStringSubmatchIndex(text, -1) {
		noun := text[m[2]:m[3]]
		if words2perRateNouns[strings.ToLower(noun)] || words2isUpperToken(noun) {
			continue
		}
		lead := strings.ToLower(words2prevWord(text, m[0]))
		if words2hasDigit(lead) || words2perRateLead[lead] {
			continue
		}
		out = append(out, types.Violation{
			RuleID:          rule,
			StartIndex:      m[0],
			EndIndex:        m[1],
			MatchedText:     text[m[0]:m[1]],
			SuggestedChange: "for each " + noun,
		})
	}
	return words2applyCase(out)
}

var words2rePiedPiping = regexp.MustCompile(`(?i)\b(?:about|across|after|against|among|around|at|before|behind|below|beneath|beside|between|beyond|by|during|for|from|in|inside|into|near|of|off|on|onto|over|through|throughout|to|toward|towards|under|until|upon|with|within|without)[ \t\r\n]+(?:which|whom)\b`)

var (
	words2piedFollow = words2set("case", "event", "time", "point", "respect", "order", "addition", "contrast", "fact", "particular", "of", "one", "ones")
	words2piedLead   = words2set(
		"order", "way", "ways", "manner", "extent", "degree", "rate", "speed", "point", "time", "times",
		"case", "cases", "frequency", "ease", "sequence", "direction", "level", "form", "position",
		"moment", "stage", "angle", "context", "respect", "terms", "light", "addition", "contrast",
		"both", "all", "some", "none", "each", "many", "one", "two", "three", "four", "five",
		"any", "most", "several", "few", "neither", "either", "half", "part",
		"decision", "decisions", "question", "questions", "choice", "choices", "detail", "details",
		"information", "guidance", "clarity", "confusion", "debate", "discussion", "doubt", "matter",
		"depends", "depending", "regardless", "unclear", "note", "notes",
		"decide", "decides", "decided", "deciding", "debate", "debates", "debated", "ask", "asks", "asked",
		"wonder", "wonders", "wondered", "know", "knows", "knew", "learn", "learns", "learned",
		"determine", "determines", "determined", "choose", "chooses", "chose", "choosing",
		"discuss", "discusses", "discussed", "explain", "explains", "explained", "describe", "describes",
		"document", "documents", "indicate", "indicates", "control", "controls", "unsure", "explicit",
	)

	words2reWhichInfinitive = regexp.MustCompile(`^[ \t]+(?:[A-Za-z][A-Za-z-]*[ \t]+){0,3}to[ \t]+[a-z]`)
)

// DetectGooglePiedPiping reports a preposition moved ahead of "which" or
// "whom" instead of left where speech would put it.
func DetectGooglePiedPiping(text string) []types.Violation {
	const rule = "pied-piping"
	var out []types.Violation
	for _, idx := range words2rePiedPiping.FindAllStringIndex(text, -1) {
		if words2piedFollow[strings.ToLower(words2nextWord(text, idx[1]))] {
			continue
		}
		if words2piedLead[strings.ToLower(words2prevWord(text, idx[0]))] {
			continue
		}
		if words2reWhichInfinitive.MatchString(text[idx[1]:]) {
			continue
		}
		out = append(out, types.Violation{
			RuleID:      rule,
			StartIndex:  idx[0],
			EndIndex:    idx[1],
			MatchedText: text[idx[0]:idx[1]],
			Explanation: "Let the preposition sit where speech puts it, including at the end.",
		})
	}
	return out
}

var words2rePluralUnit = regexp.MustCompile(`\b\d+(?:\.\d+)?[ \t]*(?:GiB|MiB|KiB|TiB|GB|MB|KB|TB|PB|EB|kHz|MHz|GHz|Hz|Mbps|Gbps|Kbps|ms|ns|us|dpi|kg|km|cm|mm)s\b`)

// DetectGooglePluralizedUnitAbbreviation reports a unit symbol given a plural
// "s", which no unit symbol takes.
func DetectGooglePluralizedUnitAbbreviation(text string) []types.Violation {
	const rule = "pluralized-unit-abbreviation"
	out := make([]types.Violation, 0, 4)
	for _, idx := range words2rePluralUnit.FindAllStringIndex(text, -1) {
		match := text[idx[0]:idx[1]]
		out = append(out, types.Violation{
			RuleID:          rule,
			StartIndex:      idx[0],
			EndIndex:        idx[1],
			MatchedText:     match,
			SuggestedChange: strings.TrimSuffix(match, "s"),
		})
	}
	return out
}

var words2reSibilantPlural = regexp.MustCompile(`\b[A-Z]{1,5}(?:SH|CH|S|X)s\b`)

// DetectGoogleSibilantAbbreviationPlural reports an abbreviation ending in a
// sibilant pluralized with a bare "s" instead of "es".
func DetectGoogleSibilantAbbreviationPlural(text string) []types.Violation {
	const rule = "sibilant-abbreviation-plural"
	out := make([]types.Violation, 0, 4)
	for _, idx := range words2reSibilantPlural.FindAllStringIndex(text, -1) {
		match := text[idx[0]:idx[1]]
		out = append(out, types.Violation{
			RuleID:          rule,
			StartIndex:      idx[0],
			EndIndex:        idx[1],
			MatchedText:     match,
			SuggestedChange: strings.TrimSuffix(match, "s") + "es",
		})
	}
	return out
}

var (
	words2reAbbrevIntro = regexp.MustCompile(`[ \t]?\(([A-Z][A-Za-z0-9]*[A-Z0-9])\)`)
	words2reWordRun     = regexp.MustCompile(`[A-Za-z][A-Za-z-]*`)
	words2abbrevStops   = words2set("of", "the", "and", "for", "in", "to", "a", "an", "on", "at", "with")
)

func words2lettersUpper(s string) string {
	var b strings.Builder
	for i := 0; i < len(s); i++ {
		switch c := s[i]; {
		case c >= 'a' && c <= 'z':
			b.WriteByte(c - 32)
		case c >= 'A' && c <= 'Z':
			b.WriteByte(c)
		}
	}
	return b.String()
}

func words2initials(words []string, skipStops bool) string {
	var b strings.Builder
	for _, w := range words {
		if skipStops && words2abbrevStops[strings.ToLower(w)] {
			continue
		}
		for _, part := range strings.Split(w, "-") {
			if part == "" {
				continue
			}
			b.WriteString(words2lettersUpper(part[:1]))
		}
	}
	return b.String()
}

func words2isIntroduction(prefix, abbr string) bool {
	target := words2lettersUpper(abbr)
	if len(target) < 2 {
		return false
	}
	words := words2reWordRun.FindAllString(prefix, -1)
	if len(words) > 8 {
		words = words[len(words)-8:]
	}
	for n := len(target); n <= len(target)+3 && n <= len(words); n++ {
		run := words[len(words)-n:]
		if words2initials(run, false) == target || words2initials(run, true) == target {
			return true
		}
	}
	return false
}

func words2countStandalone(text, word string, skipAt int) int {
	count := 0
	for i := 0; i+len(word) <= len(text); {
		j := strings.Index(text[i:], word)
		if j < 0 {
			break
		}
		at := i + j
		i = at + len(word)
		if at == skipAt {
			continue
		}
		if at > 0 && words2isWordByte(text[at-1]) {
			continue
		}
		if i < len(text) && words2isWordByte(text[i]) {
			continue
		}
		count++
	}
	return count
}

// DetectGoogleSingleUseAbbreviation reports an abbreviation introduced in
// parentheses that the document then uses at most once.
func DetectGoogleSingleUseAbbreviation(text string) []types.Violation {
	const rule = "single-use-abbreviation"
	var out []types.Violation
	for _, m := range words2reAbbrevIntro.FindAllStringSubmatchIndex(text, -1) {
		start := m[2] - 160
		if start < 0 {
			start = 0
		}
		prefix := text[start : m[2]-1]
		if i := strings.LastIndexAny(prefix, ".!?;:\n()"); i >= 0 {
			prefix = prefix[i+1:]
		}
		trimmed := strings.TrimRight(prefix, " \t")
		if trimmed == "" || !words2isWordByte(trimmed[len(trimmed)-1]) {
			continue
		}
		abbr := text[m[2]:m[3]]
		if !words2isIntroduction(trimmed, abbr) {
			continue
		}
		if words2countStandalone(text, abbr, m[2]) > 1 {
			continue
		}
		out = append(out, types.Violation{
			RuleID:      rule,
			StartIndex:  m[0],
			EndIndex:    m[1],
			MatchedText: text[m[0]:m[1]],
			Explanation: "An abbreviation the page barely uses earns nothing.",
		})
	}
	return out
}

var words2shorthandSubs, words2shorthandRes = words2compile([]words2sub{
	{`(?im)(?:^|[\s(\[])(w/o)(?:[\s.,;:!?)\]]|$)`, "without"},
	{`(?im)(?:^|[\s(\[])(w/)\s`, "with"},
	{`(?im)(?:^|[\s(\[])(b/c)(?:[\s.,;:!?)\]]|$)`, "because"},
	{`(?im)(?:^|[\s(\[])(c/o)(?:[\s.,;:!?)\]]|$)`, "care of"},
	{`(?im)(?:^|[\s(\[])(a/k/a)(?:[\s.,;:!?)\]]|$)`, "also known as"},
	{`(?im)(?:^|[\s(\[])(y/n)(?:[\s.,;:!?)\]]|$)`, "yes or no"},
	{`(?im)(?:^|[\s(\[])(approx\.)(?:\s|$)`, "approximately"},
	{`(?im)(?:^|[\s(\[])(misc\.)(?:\s|$)`, "miscellaneous"},
	{`(?im)(?:^|[\s(\[])(mgmt\.)(?:\s|$)`, "management"},
	{`(?im)(?:^|[\s(\[])(dept\.)(?:\s|$)`, "department"},
	{`(?im)(?:^|[\s(\[])(thru)(?:[\s.,;:!?)\]]|$)`, "through"},
	{`(?im)(?:^|[\s(\[])(tho)(?:[\s.,;:!?)\]]|$)`, "though"},
	{`(?m)\s(@)\s`, "at"},
})

var words2reMultiplier = regexp.MustCompile(`(?m)\b((\d+(?:\.\d+)?)[xX])(?:[\s.,;:)\]]|$)`)

// DetectGoogleSpelledOutShorthand reports slashed shorthand, clipped words, and
// multiplier "x" standing in for a word the reader shouldn't have to decode.
func DetectGoogleSpelledOutShorthand(text string) []types.Violation {
	const rule = "spelled-out-shorthand"
	out := words2findGroupSubs(text, rule, words2shorthandSubs, words2shorthandRes)
	for _, m := range words2reMultiplier.FindAllStringSubmatchIndex(text, -1) {
		out = append(out, types.Violation{
			RuleID:          rule,
			StartIndex:      m[2],
			EndIndex:        m[3],
			MatchedText:     text[m[2]:m[3]],
			SuggestedChange: text[m[4]:m[5]] + " times",
		})
	}
	return words2applyCase(out)
}

var words2uiSubs, words2uiRes = words2compile([]words2sub{
	{`(?i)\bcell[ \t]?phones?\b`, "mobile phone"},
	{`(?i)\bsmart[ \t]?phones?\b`, "mobile phone"},
	{`(?i)\bomnibox(?:es)?\b`, "address bar"},
	{`(?i)\baction_bar\b`, "app bar"},
	{`(?i)\bandroid_(?:device|phone|tablet)s?\b`, "Android-powered device"},
	{`(?i)\b(?:developer|dev)_keys?\b`, "API key"},
	{`(?i)\bAPI_Console_keys?\b`, "API key"},
	{`(?i)\bchapters?\b`, "page or section"},
	{`(?i)\btap_(?:and|&)_hold\b`, "touch & hold"},
	{`(?i)\btouch_and_hold\b`, "touch & hold"},
})

var (
	words2reAccountName = regexp.MustCompile(`(?i)\b` + words2phrase(`account_names?`) + `\b`)
	words2reTypeVerb    = regexp.MustCompile(`(?i)\b(type)[ \t]+(?:your|this)\b`)
	words2reTypeQuote   = regexp.MustCompile("(?i)\\b(type)[ \t]+[\"'`]")
	words2reTypeExempt  = regexp.MustCompile(`(?i)\b(?:data type|machine type|content type|type of|mime type|media type|return type|type system|file type|node type|instance type|type parameter)\b`)
)

var (
	words2accountLead = words2set("storage", "service", "billing", "bucket", "aws", "azure", "gcp", "domain", "cloud", "root", "iam", "project", "subscription", "tenant")
	words2typeLead    = words2set("", "and", "then", "or", "to", "please", "you", "user", "users", "they", "we", "now", "first", "next", "also", "optionally")
)

func words2sentenceAround(text string, pos int) string {
	start := 0
	if i := strings.LastIndexAny(text[:pos], ".!?\n"); i >= 0 {
		start = i + 1
	}
	end := len(text)
	if i := strings.IndexAny(text[pos:], ".!?\n"); i >= 0 {
		end = pos + i + 1
	}
	return text[start:end]
}

// DetectGoogleUIDeviceNaming reports a loose name for a device, browser
// surface, credential, or doc structure that has a settled name.
func DetectGoogleUIDeviceNaming(text string) []types.Violation {
	const rule = "ui-device-naming"
	out := findSubs(text, rule, words2uiSubs, words2uiRes)
	for _, idx := range words2reAccountName.FindAllStringIndex(text, -1) {
		if words2accountLead[strings.ToLower(words2prevWord(text, idx[0]))] {
			continue
		}
		out = append(out, types.Violation{
			RuleID:          rule,
			StartIndex:      idx[0],
			EndIndex:        idx[1],
			MatchedText:     text[idx[0]:idx[1]],
			SuggestedChange: "username",
		})
	}
	for _, re := range []*regexp.Regexp{words2reTypeVerb, words2reTypeQuote} {
		for _, m := range re.FindAllStringSubmatchIndex(text, -1) {
			if words2reTypeExempt.MatchString(words2sentenceAround(text, m[2])) {
				continue
			}
			if !words2typeLead[strings.ToLower(words2prevWord(text, m[2]))] {
				continue
			}
			out = append(out, types.Violation{
				RuleID:          rule,
				StartIndex:      m[2],
				EndIndex:        m[3],
				MatchedText:     text[m[2]:m[3]],
				SuggestedChange: "enter",
			})
		}
	}
	return words2applyCase(out)
}
