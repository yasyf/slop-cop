package detectors

import (
	"regexp"
	"strings"

	"github.com/yasyf/slop-cop/internal/types"
)

const voice1apos = `(?:'|\x{2019})`

func voice1recase(matched, replacement string) string {
	if replacement == "" || matched == "" || matched[0] < 'A' || matched[0] > 'Z' {
		return replacement
	}
	return strings.ToUpper(replacement[:1]) + replacement[1:]
}

func voice1normalize(s string) string {
	return strings.Join(strings.Fields(strings.ToLower(s)), " ")
}

func voice1lookback(text string, start int) string {
	const window = 8
	if start > window {
		return text[start-window : start]
	}
	return text[:start]
}

var voice1reAnthropomorphism = regexp.MustCompile(`(?i)\b(?:(?:the|a|an|your)\s+)?` +
	`(?:object|service|server|client|API|SDK|CLI|function|method|class|module|library|package|` +
	`parser|compiler|linter|database|system|app|application|script|browser|PC|computer|machine|` +
	`device|model|agent|job|process|container|cluster|queue|scheduler|widget|driver|kernel|` +
	`daemon|endpoint)s?\s+` +
	`(?:(?:also|then|now|already|still)\s+)?` +
	`(?:(?:does|do|did|ca|wo|would|should|might|must)\s*n` + voice1apos + `?t\s+|` +
	`(?:does|do|did|can|will|would|should|might|may|must)\s+not\s+|cannot\s+|` +
	`(?:can|will|may|might|must|should|would)\s+)?` +
	`(?:sees|see|saw|thinks|think|knows|know|believes|believe|wants|want|wishes|understands|` +
	`understand|remembers|remember|forgets|forget|forgot|decides|decide|feels|likes|hates|loves|` +
	`expects|expect|tells|tell|told|tries|try|complains|complain|refuses|refuse|prefers|prefer|` +
	`realizes|cares|care|is\s+happy|is\s+confused|gets\s+confused|is\s+aware|is\s+smart|` +
	`is\s+clever)\b`)

var voice1anthroVerbs = map[string]string{
	"sees":       "detects",
	"see":        "detect",
	"saw":        "detected",
	"tells":      "specifies",
	"tell":       "specify",
	"told":       "specified",
	"thinks":     "evaluates",
	"think":      "evaluate",
	"knows":      "has access to",
	"know":       "have access to",
	"wants":      "requires",
	"want":       "require",
	"expects":    "requires",
	"expect":     "require",
	"decides":    "determines",
	"decide":     "determine",
	"understand": "support",
	"remembers":  "stores",
	"remember":   "store",
	"forgets":    "discards",
	"forget":     "discard",
	"complains":  "reports an error",
	"complain":   "report an error",
	"refuses":    "rejects",
	"refuse":     "reject",
}

func voice1swapAnthroVerb(matched string) string {
	i := strings.LastIndexAny(matched, " \t\n")
	if i < 0 {
		return ""
	}
	repl, ok := voice1anthroVerbs[strings.ToLower(matched[i+1:])]
	if !ok {
		return ""
	}
	return matched[:i+1] + repl
}

// DetectGoogleAnthropomorphism reports software given a human faculty.
func DetectGoogleAnthropomorphism(text string) []types.Violation {
	out := findAll(text, voice1reAnthropomorphism, "anthropomorphism")
	for i := range out {
		out[i].SuggestedChange = voice1swapAnthroVerb(out[i].MatchedText)
	}
	return out
}

var voice1reExclamation = regexp.MustCompile(`[A-Za-z0-9,;:"')\]]\s*!(?:\s|$|["')\]*])`)

// DetectGoogleExclamationPoint reports an exclamation point closing a
// documentation sentence.
func DetectGoogleExclamationPoint(text string) []types.Violation {
	out := findAll(text, voice1reExclamation, "exclamation-point")
	for i := range out {
		out[i].StartIndex += strings.IndexByte(out[i].MatchedText, '!')
		out[i].EndIndex = out[i].StartIndex + 1
		out[i].MatchedText = "!"
		out[i].SuggestedChange = "."
	}
	return out
}

var voice1slangSubs = []plainSub{
	{"tl;dr", "Summary"},
	{"tldr", "Summary"},
	{"ymmv", "results vary"},
	{"rtfm", "For more information, see"},
	{"afaik", "as far as I know"},
	{"asap", "as soon as possible"},
	{"aka", "also known as"},
	{"imho", ""},
	{"imo", ""},
	{"iirc", ""},
	{"fwiw", ""},
	{"eli5", ""},
	{"icymi", ""},
	{"btw", ""},
	{"nbd", ""},
	{"ftw", ""},
	{"fyi", ""},
	{"lol", ""},
	{"rofl", ""},
	{"lmk", ""},
	{"tbh", ""},
	{"lgtm", ""},
	{"smh", ""},
	{"brb", ""},
	{"voila", ""},
	{"woot", ""},
	{"yay", ""},
}

var voice1slangRes = compileSubs(voice1slangSubs)

// DetectGoogleInternetSlang reports chat-register abbreviations.
func DetectGoogleInternetSlang(text string) []types.Violation {
	return findSubs(text, "internet-slang", voice1slangSubs, voice1slangRes)
}

var (
	voice1reContractionStacked = regexp.MustCompile(`(?i)\b[a-z]+` + voice1apos +
		`[a-z]+` + voice1apos + `(?:ve|ll|nt|d|t)\b`)
	voice1reContractionRe = regexp.MustCompile(`(?i)\b[a-z]{2,}` + voice1apos + `re\b`)
	voice1reContractionIs = regexp.MustCompile(`(?i)\b[a-z]{3,}` + voice1apos +
		`s\s+(?:an?|the|not|now|no\s+longer|called|based)\b`)
	voice1reContractionRare = regexp.MustCompile(`(?i)\b(?:` +
		`(?:it|there|that|who|what|how)` + voice1apos + `(?:d|ll)|` +
		`(?:might|must|sha|need|ought|dare)n` + voice1apos + `t|` +
		`(?:would|could|should|might|must)` + voice1apos + `ve|` +
		`ain` + voice1apos + `t|y` + voice1apos + `all)\b`)
)

var (
	voice1reContractionOkStems = map[string]bool{
		"you": true, "we": true, "they": true, "these": true, "those": true, "here": true,
	}
	voice1isContractionOkStems = map[string]bool{
		"that": true, "there": true, "here": true, "what": true, "who": true, "she": true,
		"one": true, "everything": true, "something": true, "nothing": true, "this": true,
		"let": true, "how": true,
	}
)

var voice1contractionExpansions = map[string]string{
	"it'd":         "it would",
	"there'd":      "there would",
	"that'd":       "that would",
	"who'd":        "who would",
	"what'd":       "what did",
	"how'd":        "how did",
	"it'll":        "it will",
	"there'll":     "there will",
	"that'll":      "that will",
	"who'll":       "who will",
	"what'll":      "what will",
	"mightn't":     "might not",
	"mustn't":      "must not",
	"shan't":       "shall not",
	"needn't":      "do not need to",
	"oughtn't":     "ought not",
	"daren't":      "dare not",
	"would've":     "would have",
	"could've":     "could have",
	"should've":    "should have",
	"might've":     "might have",
	"must've":      "must have",
	"ain't":        "is not",
	"y'all":        "you",
	"mightn't've":  "might not have",
	"mustn't've":   "must not have",
	"couldn't've":  "could not have",
	"shouldn't've": "should not have",
	"wouldn't've":  "would not have",
	"it'd've":      "it would have",
}

func voice1aposSplit(s string) (int, int) {
	if i := strings.IndexByte(s, '\''); i >= 0 {
		return i, 1
	}
	if i := strings.Index(s, "\u2019"); i >= 0 {
		return i, len("\u2019")
	}
	return -1, 0
}

func voice1expandContraction(matched string) string {
	key := strings.ReplaceAll(strings.ToLower(matched), "\u2019", "'")
	return voice1recase(matched, voice1contractionExpansions[key])
}

// DetectGoogleNonstandardContraction reports invented, stacked, or uncommon
// contractions.
func DetectGoogleNonstandardContraction(text string) []types.Violation {
	const rule = "nonstandard-contraction"
	var out []types.Violation

	for _, v := range findAll(text, voice1reContractionStacked, rule) {
		v.SuggestedChange = voice1expandContraction(v.MatchedText)
		out = append(out, v)
	}

	for _, v := range findAll(text, voice1reContractionRe, rule) {
		i, _ := voice1aposSplit(v.MatchedText)
		stem := v.MatchedText[:i]
		if voice1reContractionOkStems[strings.ToLower(stem)] {
			continue
		}
		v.SuggestedChange = stem + " are"
		out = append(out, v)
	}

	for _, v := range findAll(text, voice1reContractionIs, rule) {
		if v.EndIndex < len(text) && text[v.EndIndex] == '-' {
			continue
		}
		i, n := voice1aposSplit(v.MatchedText)
		stem := v.MatchedText[:i]
		if voice1isContractionOkStems[strings.ToLower(stem)] {
			continue
		}
		v.SuggestedChange = stem + " is" + v.MatchedText[i+n+1:]
		out = append(out, v)
	}

	for _, v := range findAll(text, voice1reContractionRare, rule) {
		v.SuggestedChange = voice1expandContraction(v.MatchedText)
		out = append(out, v)
	}

	return out
}

var (
	voice1rePassiveByAgent = regexp.MustCompile(
		`(?i:\b(?:is|are|was|were|be|been|being|gets?|got)\s+(?:[a-z]+ly\s+)?` +
			`(?:[a-z]+(?:ed|en|wn)|` + strings.Join(irregularParticiples, "|") + `)\s+by\s+)` +
			`(?:(?i:you|your|the|a|an|our|its|their|this|each)(?:\s+[a-z]+)?|[A-Z][a-z]+(?:\s+[A-Z][a-z]+)?)\b`)
	voice1reByIdiom = regexp.MustCompile(`(?i)\bby\s+(?:default|design|the\s+time|now|contrast|` +
		`comparison|hand|convention|accident|up\s+to|a\s+factor)\b`)
	voice1reByCriterion = regexp.MustCompile(`(?i)\b(?:sorted|ordered|grouped|keyed|indexed|` +
		`sharded|partitioned|filtered|ranked|followed|preceded|separated|delimited|denoted|` +
		`represented|multiplied|divided|prefixed|suffixed|terminated|accompanied)\s+by\b`)
	voice1reTrailingFunction = regexp.MustCompile(`(?i)\s+(?:at|in|on|to|for|from|with|before|` +
		`after|and|or|when|if|as|during|using|over|under|than|that|which|so|but|because)$`)
)

// DetectGooglePassiveByAgent reports a passive clause that names its actor in a
// trailing `by` phrase.
func DetectGooglePassiveByAgent(text string) []types.Violation {
	var out []types.Violation
	for _, v := range findAll(text, voice1rePassiveByAgent, "passive-by-agent") {
		if voice1reByIdiom.MatchString(v.MatchedText) || voice1reByCriterion.MatchString(v.MatchedText) {
			continue
		}
		if tail := voice1reTrailingFunction.FindString(v.MatchedText); tail != "" {
			v.EndIndex -= len(tail)
			v.MatchedText = text[v.StartIndex:v.EndIndex]
		}
		out = append(out, v)
	}
	return out
}

var voice1rePlaceholderOpener = regexp.MustCompile(`(?im)(?:^|[.!?;]\s+|,\s+)(?:` +
	`please\s+note(?:\s+that)?|note\s+that|at\s+this\s+time|at\s+the\s+present\s+time|` +
	`it\s+should\s+be\s+noted\s+that|` +
	`as\s+(?:you\s+can\s+see|you\s+know|previously\s+(?:noted|mentioned)))\b[ \t]*`)

func voice1firstLetter(s string) int {
	return strings.IndexFunc(s, func(r rune) bool {
		return r >= 'a' && r <= 'z' || r >= 'A' && r <= 'Z'
	})
}

// DetectGooglePlaceholderPhrase reports an opener that announces a sentence
// instead of making it.
func DetectGooglePlaceholderPhrase(text string) []types.Violation {
	out := findAll(text, voice1rePlaceholderOpener, "placeholder-phrase")
	for i := range out {
		out[i].StartIndex += voice1firstLetter(out[i].MatchedText)
		out[i].MatchedText = text[out[i].StartIndex:out[i].EndIndex]
		if n := voice1normalize(out[i].MatchedText); strings.HasPrefix(n, "at this time") ||
			strings.HasPrefix(n, "at the present time") {
			out[i].SuggestedChange = voice1recase(out[i].MatchedText, "now")
		}
	}
	return out
}

var (
	voice1rePleaseNote = regexp.MustCompile(`(?i)\bplease\s+note(?:\s+that)?[ \t]*`)
	voice1rePleaseStep = regexp.MustCompile(`(?im)(?:^[ \t]*(?:[-*+]|\d+[.)])?[ \t]*|[.!?]\s+)` +
		`please\s+[a-z]+`)
)

var voice1pleaseImposition = map[string]bool{
	"contact": true, "reach": true, "email": true, "file": true, "report": true,
	"open": true, "submit": true, "excuse": true, "be": true, "note": true,
}

// DetectGooglePleaseInInstructions reports `please` softening a step.
func DetectGooglePleaseInInstructions(text string) []types.Violation {
	const rule = "please-in-instructions"
	out := findAll(text, voice1rePleaseNote, rule)
	for _, v := range findAll(text, voice1rePleaseStep, rule) {
		v.StartIndex += strings.Index(strings.ToLower(v.MatchedText), "please")
		v.MatchedText = text[v.StartIndex:v.EndIndex]
		fields := strings.Fields(v.MatchedText)
		if len(fields) != 2 || voice1pleaseImposition[strings.ToLower(fields[1])] {
			continue
		}
		v.SuggestedChange = voice1recase(v.MatchedText, fields[1])
		out = append(out, v)
	}
	return out
}

var (
	voice1reReaderEndUser = regexp.MustCompile(`(?i)\b(?:your\s+(?:app|application|service|users|` +
		`customers)|end\s+users?)\b`)
	voice1reReaderThirdActs = regexp.MustCompile(`(?i)\bthe\s+(?:users?|developers?|readers?|` +
		`customers?|administrators?)\s+(?:can|must|should|might|will|needs?\s+to|has\s+to|` +
		`have\s+to|is\s+expected\s+to|are\s+expected\s+to|clicks?|selects?|runs?|enters?|opens?|` +
		`navigates?|types?|sees?)\b`)
	voice1reReaderThirdHead = regexp.MustCompile(`(?i)^the\s+(?:users?|developers?|readers?|` +
		`customers?|administrators?)`)
	voice1reReaderPossessive = regexp.MustCompile(`(?i)\bthe\s+users?` + voice1apos + `s?\s+[a-z]+\b`)
	voice1reReaderPossHead   = regexp.MustCompile(`(?i)^the\s+users?` + voice1apos + `s?`)
	voice1reReaderTaught     = regexp.MustCompile(`(?i)\b(?:shows?|teaches?|walks?|guides?|tells?)\s+` +
		`the\s+(?:users?|developers?|readers?)\s+(?:how|where|what|why|when|through)\b`)
	voice1reReaderTaughtHead = regexp.MustCompile(`(?i)\bthe\s+(?:users?|developers?|readers?)\b`)
	voice1reReaderJoint      = regexp.MustCompile(`(?i)\b(?:we|let\s+us)\s+` +
		`(?:can\s+|now\s+|then\s+|next\s+|first\s+)*` +
		`(?:create|run|start|open|configure|install|deploy|add|build|set|click|select|navigate|` +
		`examine|begin|continue|verify|check|look|take)\b`)
	voice1reReaderLets = regexp.MustCompile(`(?i)\blet` + voice1apos + `s\s+[a-z]+\b`)
	voice1reReaderOur  = regexp.MustCompile(`(?i)\bour\s+(?:project|cluster|app|application|` +
		`instance|bucket|repo|repository|config|configuration|file|script|example|sample|table)s?\b`)
)

// DetectGoogleReaderAddressPerson reports the reader addressed in the third
// person, or joint first-person framing of steps only the reader performs.
func DetectGoogleReaderAddressPerson(text string) []types.Violation {
	const rule = "reader-address-person"
	var out []types.Violation

	off := 0
	for _, sentence := range mergeAbbrev(splitSentences(text)) {
		start := off
		off += len(sentence)
		if voice1reReaderEndUser.MatchString(sentence) {
			continue
		}
		for _, v := range findAll(sentence, voice1reReaderThirdActs, rule) {
			v.StartIndex += start
			v.EndIndex += start
			v.SuggestedChange = voice1recase(v.MatchedText,
				voice1reReaderThirdHead.ReplaceAllString(v.MatchedText, "you"))
			out = append(out, v)
		}
		for _, v := range findAll(sentence, voice1reReaderPossessive, rule) {
			v.StartIndex += start
			v.EndIndex += start
			v.SuggestedChange = voice1recase(v.MatchedText,
				voice1reReaderPossHead.ReplaceAllString(v.MatchedText, "your"))
			out = append(out, v)
		}
		for _, v := range findAll(sentence, voice1reReaderTaught, rule) {
			v.StartIndex += start
			v.EndIndex += start
			v.SuggestedChange = voice1reReaderTaughtHead.ReplaceAllString(v.MatchedText, "you")
			out = append(out, v)
		}
	}

	out = append(out, findAll(text, voice1reReaderJoint, rule)...)
	out = append(out, findAll(text, voice1reReaderLets, rule)...)
	for _, v := range findAll(text, voice1reReaderOur, rule) {
		v.SuggestedChange = voice1recase(v.MatchedText,
			"your"+v.MatchedText[len("our"):])
		out = append(out, v)
	}
	return out
}

var (
	voice1reSuperlative = regexp.MustCompile(`(?i)\b(?:the\s+)?(?:best|worst|fastest|slowest|` +
		`simplest|easiest|cheapest|safest|most\s+(?:powerful|secure|reliable|scalable|advanced|` +
		`efficient|affordable|popular)|unmatched|unparalleled|unbeatable|industry-leading|` +
		`world-class|state-of-the-art|number\s+one|only\s+solution)\b`)
	voice1reAbsolute = regexp.MustCompile(`(?i)\b(?:never|always)\s+(?:fails?|breaks?|works?|` +
		`loses?|drops?|succeeds?|available|secure|correct|consistent|up)\b`)
	voice1reSuperlativeNoun = regexp.MustCompile(`(?i)^[\s-]*(?:practices?|effort|cases?)\b`)
	voice1reUpToDate        = regexp.MustCompile(`(?i)^[\s-]*to\b`)
	voice1reAtBest          = regexp.MustCompile(`(?i)\bat\s+$`)
)

var voice1superlativeSuggestions = map[string]string{
	"the best":         "the recommended",
	"the simplest":     "the shortest",
	"always available": "designed for high availability",
	"never fails":      "retries automatically",
	"never fail":       "retry automatically",
}

// DetectGoogleSuperlativeProductClaim reports an uncheckable superlative or
// absolute claim about a product.
func DetectGoogleSuperlativeProductClaim(text string) []types.Violation {
	const rule = "superlative-product-claim"
	var out []types.Violation

	for _, v := range findAll(text, voice1reSuperlative, rule) {
		if voice1reSuperlativeNoun.MatchString(text[v.EndIndex:]) {
			continue
		}
		if voice1reAtBest.MatchString(voice1lookback(text, v.StartIndex)) {
			continue
		}
		v.SuggestedChange = voice1recase(v.MatchedText,
			voice1superlativeSuggestions[voice1normalize(v.MatchedText)])
		out = append(out, v)
	}

	for _, v := range findAll(text, voice1reAbsolute, rule) {
		if strings.HasSuffix(strings.ToLower(v.MatchedText), "up") &&
			voice1reUpToDate.MatchString(text[v.EndIndex:]) {
			continue
		}
		v.SuggestedChange = voice1recase(v.MatchedText,
			voice1superlativeSuggestions[voice1normalize(v.MatchedText)])
		out = append(out, v)
	}

	return out
}
