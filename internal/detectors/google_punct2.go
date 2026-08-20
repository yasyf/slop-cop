package detectors

import (
	"regexp"
	"strings"

	"github.com/yasyf/slop-cop/internal/types"
)

// punct2thirdPersonVerbs is a closed list of third-person-singular verbs used
// to decide whether a run of words is a sentence. Nouns that share a spelling
// with a verb (logs, points, needs, sets, uses, matches, records, works) are
// deliberately absent: a caption or noun phrase must never read as a sentence.
var punct2thirdPersonVerbs = []string{
	"accepts", "adds", "allows", "appears", "applies", "arrives", "becomes",
	"begins", "belongs", "closes", "completes", "contains", "continues",
	"converts", "creates", "defines", "deletes", "depends", "describes",
	"determines", "disables", "displays", "emits", "enables", "ensures",
	"exceeds", "executes", "exists", "expands", "expects", "expires", "exposes",
	"extends", "fails", "fetches", "finds", "follows", "generates", "gets",
	"gives", "goes", "grants", "happens", "helps", "ignores", "improves",
	"includes", "increases", "indicates", "inherits", "initializes", "installs",
	"invokes", "involves", "iterates", "knows", "launches", "lets", "listens",
	"makes", "manages", "merges", "occurs", "opens", "operates", "optimizes",
	"overrides", "owns", "parses", "performs", "polls", "populates", "prevents",
	"prints", "produces", "provides", "publishes", "raises", "receives",
	"reduces", "refers", "refreshes", "reflects", "rejects", "relies", "reloads",
	"removes", "renders", "replaces", "requires", "resolves", "responds",
	"restarts", "restores", "retries", "retrieves", "returns", "reverts",
	"rewrites", "runs", "selects", "sends", "serves", "shows", "simplifies",
	"skips", "sorts", "specifies", "starts", "stays", "stops", "submits",
	"subscribes", "succeeds", "supplies", "supports", "syncs", "takes", "tells",
	"terminates", "throws", "tracks", "transmits", "treats", "tries",
	"truncates", "validates", "verifies", "waits", "wants", "warns", "wraps",
	"writes", "yields",
}

var punct2reFiniteVerb = regexp.MustCompile(`(?i)\b(?:is|are|was|were|can|could|will|would|should|must|may|might|has|have|had|do|does|did|` +
	strings.Join(punct2thirdPersonVerbs, "|") + `)\b`)

func punct2inCodeSpan(text string, idx int) bool {
	lineStart := strings.LastIndexByte(text[:idx], '\n') + 1
	return strings.Count(text[lineStart:idx], "`")%2 == 1
}

func punct2lineAt(text string, idx int) string {
	lineStart := strings.LastIndexByte(text[:idx], '\n') + 1
	lineEnd := len(text)
	if p := strings.IndexByte(text[idx:], '\n'); p >= 0 {
		lineEnd = idx + p
	}
	return text[lineStart:lineEnd]
}

var (
	punct2reBlockMarker = regexp.MustCompile(`^[ \t]{0,6}(?:[-*+][ \t]|\d+[.)][ \t]|>|\||!?\[|#|<|=|_)`)
	punct2reImageOnly   = regexp.MustCompile(`^!?\[[^\]]*\]\([^)]*\)$`)
	punct2reTrailingURL = regexp.MustCompile(`https?://\S+$`)
	punct2reTermPunct   = regexp.MustCompile(`[.?!:;,\x{2014}]$`)
	punct2reWordTail    = regexp.MustCompile(`[\p{L}\p{N}%]$`)
)

// DetectGoogleMissingTerminalPeriod reports running-prose paragraphs whose last
// line is a sentence with no closing punctuation.
func DetectGoogleMissingTerminalPeriod(text string) []types.Violation {
	var out []types.Violation
	for _, p := range splitParagraphs(text) {
		if strings.Contains(p.text, "```") || strings.Contains(p.text, "~~~") {
			continue
		}
		body := strings.TrimRight(p.text, " \t\r\n")
		if body == "" {
			continue
		}
		lineStart := strings.LastIndexByte(body, '\n') + 1
		line := body[lineStart:]
		trimmed := strings.TrimSpace(line)
		if trimmed == "" {
			continue
		}
		if isATXHeading(line) || isTabular(line) || punct2reBlockMarker.MatchString(line) {
			continue
		}
		if strings.HasPrefix(line, "    ") || strings.HasPrefix(line, "\t") || strings.HasPrefix(trimmed, "**") {
			continue
		}
		if punct2reImageOnly.MatchString(trimmed) || punct2reTrailingURL.MatchString(trimmed) {
			continue
		}
		core := strings.TrimRight(trimmed, ")]}\"'\u201D\u2019")
		if punct2reTermPunct.MatchString(core) || !punct2reWordTail.MatchString(core) {
			continue
		}
		if proseWordCount(trimmed) < 5 || !punct2reFiniteVerb.MatchString(trimmed) {
			continue
		}
		start := p.start + lineStart + len(line) - len(strings.TrimLeft(line, " \t"))
		end := p.start + len(body)
		out = append(out, types.Violation{
			RuleID:      "missing-terminal-period",
			StartIndex:  start,
			EndIndex:    end,
			MatchedText: text[start:end],
		})
	}
	return out
}

var (
	punct2reNumberUnit = regexp.MustCompile(`(?i:\b(?:\d[\d,]*|one|two|three|four|five|six|seven|eight|nine|ten|twelve)[ \t](?:bit|byte|word|character|digit|minute|second|hour|day|week|month|year|core|node|thread|page|step|line|column|row|element|item|foot|inch|meter|mile|volt|watt|degree))[ \t]([a-z]{2,})\b`)
	punct2reSpaceRun   = regexp.MustCompile(`[ \t]+`)
)

var punct2unitFollowStop = map[string]bool{
	"is": true, "are": true, "was": true, "were": true, "be": true, "been": true,
	"of": true, "in": true, "on": true, "to": true, "for": true, "after": true,
	"before": true, "later": true, "ago": true, "from": true, "and": true,
	"or": true, "per": true, "that": true, "which": true, "the": true,
	"an": true, "this": true, "these": true, "those": true, "it": true,
	"its": true, "you": true, "we": true, "they": true, "there": true,
	"here": true, "their": true, "his": true, "her": true, "will": true,
	"can": true, "could": true, "would": true, "should": true, "must": true,
	"may": true, "might": true, "has": true, "have": true, "had": true,
	"do": true, "does": true, "did": true, "not": true, "no": true,
	"only": true, "just": true, "also": true, "still": true, "again": true,
	"even": true, "when": true, "while": true, "if": true, "so": true,
	"but": true, "as": true, "at": true, "with": true, "by": true,
	"into": true, "than": true, "then": true, "each": true, "old": true,
	"long": true, "away": true, "apart": true, "worth": true, "until": true,
	"since": true, "during": true, "about": true, "over": true, "under": true,
	"out": true, "up": true, "down": true, "off": true, "because": true,
	"unless": true, "how": true, "why": true, "what": true, "where": true,
	"who": true, "earlier": true, "without": true, "within": true,
	"apiece": true,
}

// DetectGoogleNumberUnitModifierHyphen reports a number and a spelled-out
// singular unit that modify the noun after them without a hyphen.
func DetectGoogleNumberUnitModifierHyphen(text string) []types.Violation {
	var out []types.Violation
	for _, m := range punct2reNumberUnit.FindAllStringSubmatchIndex(text, -1) {
		follow := strings.ToLower(text[m[2]:m[3]])
		if punct2unitFollowStop[follow] || strings.HasSuffix(follow, "ing") ||
			len(follow) >= 4 && strings.HasSuffix(follow, "ly") {
			continue
		}
		if m[0] > 0 && (strings.IndexByte("-./", text[m[0]-1]) >= 0 || punct2isWordByte(text[m[0]-1])) {
			continue
		}
		if punct2inCodeSpan(text, m[0]) {
			continue
		}
		start, end := trimSpan(text, m[0], m[2])
		matched := text[start:end]
		out = append(out, types.Violation{
			RuleID:          "number-unit-modifier-hyphen",
			StartIndex:      start,
			EndIndex:        end,
			MatchedText:     matched,
			SuggestedChange: punct2reSpaceRun.ReplaceAllString(matched, "-"),
		})
	}
	return out
}

var punct2reOptionalPlural = regexp.MustCompile(`\b([A-Za-z]{2,})\((s|es|ies|ren)\)`)

// DetectGoogleOptionalPluralParens reports a plural ending wrapped in
// parentheses.
func DetectGoogleOptionalPluralParens(text string) []types.Violation {
	var out []types.Violation
	for _, m := range punct2reOptionalPlural.FindAllStringSubmatchIndex(text, -1) {
		if m[0] > 0 && strings.IndexByte("._$", text[m[0]-1]) >= 0 {
			continue
		}
		if punct2inCodeSpan(text, m[0]) {
			continue
		}
		line := punct2lineAt(text, m[0])
		if strings.HasPrefix(line, "    ") || strings.HasPrefix(line, "\t") {
			continue
		}
		base := text[m[2]:m[3]]
		suffix := text[m[4]:m[5]]
		replacement := base + suffix
		if suffix == "ies" {
			replacement = strings.TrimSuffix(base, "y") + "ies"
		}
		out = append(out, types.Violation{
			RuleID:          "optional-plural-parens",
			StartIndex:      m[0],
			EndIndex:        m[1],
			MatchedText:     text[m[0]:m[1]],
			SuggestedChange: replacement,
		})
	}
	return out
}

var punct2reHyphenChain = regexp.MustCompile(`([A-Za-z0-9]+(?:-[A-Za-z0-9]+){2,})[ \t]+([a-z]+)`)

var punct2chainConnectives = map[string]bool{
	"to": true, "by": true, "of": true, "the": true, "a": true, "an": true,
	"and": true, "or": true, "nor": true, "in": true, "on": true, "at": true,
	"for": true, "with": true, "from": true, "as": true, "into": true,
	"per": true, "then": true, "than": true, "over": true, "under": true,
	"out": true, "up": true, "down": true, "off": true, "upon": true,
	"versus": true, "vs": true, "is": true, "are": true, "you": true,
	"it": true, "we": true, "so": true, "but": true, "if": true,
	"via": true, "since": true, "without": true, "before": true,
	"after": true, "between": true, "across": true, "within": true,
	"against": true, "through": true, "during": true, "plus": true,
	"minus": true, "all": true, "any": true, "no": true, "not": true,
	"one": true, "two": true, "be": true, "was": true, "were": true,
	"this": true, "that": true,
}

var punct2modifierPrefixes = map[string]bool{
	"non": true, "self": true, "cross": true, "pre": true, "post": true,
	"re": true, "sub": true, "inter": true, "intra": true, "anti": true,
	"semi": true, "co": true, "micro": true, "mid": true, "over": true,
	"under": true, "auto": true, "bi": true, "de": true, "multi": true,
	"un": true, "pseudo": true, "quasi": true, "per": true, "no": true,
}

var punct2chainFollowStop = map[string]bool{
	"is": true, "are": true, "was": true, "were": true, "be": true,
	"been": true, "has": true, "have": true, "had": true, "can": true,
	"could": true, "will": true, "would": true, "should": true, "must": true,
	"may": true, "might": true, "do": true, "does": true, "did": true,
	"and": true, "or": true, "but": true, "in": true, "on": true, "to": true,
	"for": true, "from": true, "with": true, "as": true, "at": true,
	"by": true, "of": true, "that": true, "which": true, "when": true,
	"then": true, "also": true, "only": true, "too": true, "so": true,
	"if": true, "not": true, "no": true, "runs": true, "works": true,
	"lets": true, "means": true, "makes": true, "takes": true, "gives": true,
	"returns": true, "requires": true, "supports": true, "uses": true,
	"rather": true, "instead": true, "versus": true, "than": true,
	"ran": true, "per": true, "first": true, "both": true, "either": true,
	"again": true, "here": true, "there": true, "now": true, "later": true,
	"before": true, "after": true, "during": true, "while": true,
	"until": true, "since": true, "unless": true, "without": true,
	"within": true, "across": true, "between": true, "against": true,
	"through": true, "all": true, "any": true, "each": true, "every": true,
}

var punct2reNumericChain = regexp.MustCompile(`^[0-9]+(?:-[0-9]+)+$`)

// DetectGoogleOverlongCompoundModifier reports a hyphenated modifier of three
// or more components standing in front of a noun.
func DetectGoogleOverlongCompoundModifier(text string) []types.Violation {
	var out []types.Violation
	for _, m := range punct2reHyphenChain.FindAllStringSubmatchIndex(text, -1) {
		start, end := m[2], m[3]
		if start > 0 && (strings.IndexByte("-/`_.'", text[start-1]) >= 0 || punct2isWordByte(text[start-1])) {
			continue
		}
		if start >= 3 && text[start-3:start] == "’" {
			continue
		}
		chain := strings.ToLower(text[start:end])
		if punct2reNumericChain.MatchString(chain) {
			continue
		}
		parts := strings.Split(chain, "-")
		if punct2modifierPrefixes[parts[0]] || punct2hasConnective(parts) || punct2hasAlnumPart(parts) {
			continue
		}
		if punct2chainFollowStop[strings.ToLower(text[m[4]:m[5]])] {
			continue
		}
		if punct2inCodeSpan(text, start) {
			continue
		}
		out = append(out, types.Violation{
			RuleID:      "overlong-compound-modifier",
			StartIndex:  start,
			EndIndex:    end,
			MatchedText: text[start:end],
		})
	}
	return out
}

func punct2hasConnective(parts []string) bool {
	for _, p := range parts[1:] {
		if punct2chainConnectives[p] {
			return true
		}
	}
	return false
}

// punct2hasAlnumPart reports a component mixing letters and digits, the mark of
// a resource name (tnt-usw2-0ddq7rb) rather than a modifier.
func punct2hasAlnumPart(parts []string) bool {
	for _, p := range parts {
		var letter, digit bool
		if len(p) < 4 {
			continue
		}
		for i := 0; i < len(p); i++ {
			if p[i] >= '0' && p[i] <= '9' {
				digit = true
			} else {
				letter = true
			}
		}
		if letter && digit {
			return true
		}
	}
	return false
}

func punct2isWordByte(b byte) bool {
	return b >= 'a' && b <= 'z' || b >= 'A' && b <= 'Z' || b >= '0' && b <= '9'
}

var (
	punct2reAcronymPoss  = regexp.MustCompile(`\([A-Z]{2,}['\x{2019}]s\)`)
	punct2reNamePossTail = regexp.MustCompile(`[A-Z][A-Za-z ]+['\x{2019}]s[ \t]+$`)
)

// DetectGoogleParentheticalAcronymPossessive reports a possessive stacked on a
// parenthetical initialism.
func DetectGoogleParentheticalAcronymPossessive(text string) []types.Violation {
	out := make([]types.Violation, 0, 4)
	for _, idx := range punct2reAcronymPoss.FindAllStringIndex(text, -1) {
		start := idx[0]
		if p := punct2reNamePossTail.FindStringIndex(text[:start]); p != nil {
			start = p[0]
		}
		out = append(out, types.Violation{
			RuleID:      "parenthetical-acronym-possessive",
			StartIndex:  start,
			EndIndex:    idx[1],
			MatchedText: text[start:idx[1]],
		})
	}
	return out
}

var (
	punct2rePeriodInside  = regexp.MustCompile(`[A-Za-z0-9,;:][ \t]*(\([a-z][^()]*\.\))`)
	punct2rePeriodOutside = regexp.MustCompile(`(?m)(?:^|[.!?]["\x{201D}]?[ \t\n]+)(\([A-Z][^()]*[^.!?)]\)\.)`)
	punct2reAbbrevEnd     = regexp.MustCompile(`(?i)\b(?:etc|inc|ltd|e\.g|i\.e|u\.s|vs|corp|co|approx|fig|al)\.$`)
)

// DetectGoogleParentheticalPeriodPlacement reports a sentence period on the
// wrong side of a closing parenthesis.
func DetectGoogleParentheticalPeriodPlacement(text string) []types.Violation {
	var out []types.Violation
	for _, m := range punct2rePeriodInside.FindAllStringSubmatchIndex(text, -1) {
		content := strings.TrimSpace(text[m[2]+1 : m[3]-1])
		if punct2reAbbrevEnd.MatchString(content) || strings.HasPrefix(strings.ToLower(content), "see ") {
			continue
		}
		if proseWordCount(content) < 4 || strings.HasSuffix(content, "..") || strings.Contains(content, "…") {
			continue
		}
		if punct2inCodeSpan(text, m[2]) {
			continue
		}
		out = append(out, types.Violation{
			RuleID:          "parenthetical-period-placement",
			StartIndex:      m[3] - 2,
			EndIndex:        m[3],
			MatchedText:     text[m[3]-2 : m[3]],
			SuggestedChange: ").",
		})
	}
	for _, m := range punct2rePeriodOutside.FindAllStringSubmatchIndex(text, -1) {
		content := text[m[2]+1 : m[3]-2]
		if !punct2reFiniteVerb.MatchString(content) {
			continue
		}
		if punct2inCodeSpan(text, m[2]) {
			continue
		}
		out = append(out, types.Violation{
			RuleID:          "parenthetical-period-placement",
			StartIndex:      m[3] - 2,
			EndIndex:        m[3],
			MatchedText:     text[m[3]-2 : m[3]],
			SuggestedChange: ".)",
		})
	}
	return out
}

var (
	punct2rePluralPossCons = regexp.MustCompile(`\b[A-Za-z]*[bcdfgklmnprtvwxz]s['\x{2019}]s\b`)
	punct2rePluralPossIes  = regexp.MustCompile(`\b[A-Za-z]+ies['\x{2019}]s\b`)
	punct2reStripPossS     = regexp.MustCompile(`s\z`)
)

var punct2singularSibilants = map[string]bool{
	"lens": true, "news": true, "series": true, "species": true, "rabies": true,
	"means": true, "corps": true, "alms": true,
}

// DetectGooglePluralPossessiveApostrophe reports a plural possessive written
// with a second s after the apostrophe.
func DetectGooglePluralPossessiveApostrophe(text string) []types.Violation {
	var out []types.Violation
	spans := punct2rePluralPossCons.FindAllStringIndex(text, -1)
	spans = append(spans, punct2rePluralPossIes.FindAllStringIndex(text, -1)...)
	seen := make(map[int]bool, len(spans))
	for _, idx := range spans {
		if seen[idx[0]] {
			continue
		}
		seen[idx[0]] = true
		if idx[0] > 0 && strings.IndexByte(".-/", text[idx[0]-1]) >= 0 {
			continue
		}
		matched := text[idx[0]:idx[1]]
		stem := strings.ToLower(matched[:strings.IndexAny(matched, "'\u2019")])
		if len(stem) < 4 || punct2singularSibilants[stem] || strings.HasSuffix(stem, "ics") {
			continue
		}
		if punct2inCodeSpan(text, idx[0]) {
			continue
		}
		out = append(out, types.Violation{
			RuleID:          "plural-possessive-apostrophe",
			StartIndex:      idx[0],
			EndIndex:        idx[1],
			MatchedText:     matched,
			SuggestedChange: punct2reStripPossS.ReplaceAllString(matched, ""),
		})
	}
	return out
}

var (
	punct2rePostColon    = regexp.MustCompile(`:[ \t]+([A-Z][a-z]{2,})`)
	punct2reNoticeLabel  = regexp.MustCompile(`(?i)(?:^|[\s>])[*_]{0,2}(?:note|caution|warning|important|key point|tip|success|prerequisites?|beta|deprecated|example)[*_]{0,2}$`)
	punct2reBoldRunIn    = regexp.MustCompile(`^[ \t]*(?:(?:[-*+]|\d+[.)])[ \t]+)?\*\*[^*]+\*\*[ \t]*$`)
	punct2reConfigKey    = regexp.MustCompile(`^[ \t]*[a-z][a-z0-9_.-]*$`)
	punct2reColonSchemes = regexp.MustCompile(`(?i)(?:^|[\s(])(?:https?|ftp|mailto|file|data|urn)$`)
	punct2reInnerCapital = regexp.MustCompile(`[ \t][A-Z]`)
)

var punct2imperativeVerbs = map[string]bool{
	"run": true, "set": true, "use": true, "add": true, "open": true,
	"close": true, "create": true, "delete": true, "install": true,
	"configure": true, "check": true, "enable": true, "disable": true,
	"start": true, "stop": true, "restart": true, "copy": true, "paste": true,
	"click": true, "select": true, "choose": true, "enter": true, "type": true,
	"replace": true, "remove": true, "update": true, "upgrade": true,
	"download": true, "upload": true, "import": true, "export": true,
	"save": true, "load": true, "send": true, "read": true, "write": true,
	"edit": true, "apply": true, "deploy": true, "build": true, "test": true,
	"verify": true, "ensure": true, "make": true, "get": true, "put": true,
	"call": true, "wait": true, "retry": true, "review": true, "follow": true,
	"see": true, "avoid": true, "prefer": true, "keep": true, "leave": true,
	"include": true, "specify": true, "provide": true, "define": true,
	"register": true, "visit": true, "consider": true, "confirm": true,
	"repeat": true, "return": true, "sign": true, "switch": true, "try": true,
}

// DetectGooglePostColonCapitalization reports a capitalized fragment following
// a colon inside a sentence.
func DetectGooglePostColonCapitalization(text string) []types.Violation {
	var out []types.Violation
	for _, m := range punct2rePostColon.FindAllStringSubmatchIndex(text, -1) {
		word := text[m[2]:m[3]]
		if m[3] < len(text) {
			next := text[m[3]]
			if next == '_' || next == '.' || next == '-' || next == '(' || next >= 'A' && next <= 'Z' {
				continue
			}
		}
		if punct2properNouns[word] || punct2imperativeVerbs[strings.ToLower(word)] {
			continue
		}
		if m[0] > 0 && text[m[0]-1] >= '0' && text[m[0]-1] <= '9' {
			continue
		}
		prefix := text[:m[0]]
		if punct2reNoticeLabel.MatchString(prefix) || punct2reColonSchemes.MatchString(prefix) {
			continue
		}
		line := punct2lineAt(text, m[0])
		if isATXHeading(line) || isTabular(line) {
			continue
		}
		lineStart := strings.LastIndexByte(prefix, '\n') + 1
		if punct2reBoldRunIn.MatchString(prefix[lineStart:]) || punct2reConfigKey.MatchString(prefix[lineStart:]) {
			continue
		}
		if punct2inCodeSpan(text, m[2]) {
			continue
		}
		rest := text[m[2]:]
		if i := strings.IndexAny(rest, ".!?\n"); i >= 0 {
			rest = rest[:i]
		}
		if punct2reFiniteVerb.MatchString(rest) || punct2reInnerCapital.MatchString(rest) {
			continue
		}
		out = append(out, types.Violation{
			RuleID:          "post-colon-capitalization",
			StartIndex:      m[2],
			EndIndex:        m[3],
			MatchedText:     word,
			SuggestedChange: strings.ToLower(word[:1]) + word[1:],
		})
	}
	return out
}

var punct2properNouns = map[string]bool{
	"Google": true, "Claude": true, "Anthropic": true, "OpenAI": true,
	"Python": true, "Golang": true, "Java": true, "JavaScript": true,
	"TypeScript": true, "Docker": true, "Kubernetes": true, "Linux": true,
	"Windows": true, "GitHub": true, "GitLab": true, "Azure": true,
	"Cloud": true, "Node": true, "React": true, "Django": true, "Flask": true,
	"Rails": true, "Postgres": true, "PostgreSQL": true, "MySQL": true,
	"Redis": true, "Mongo": true, "MongoDB": true, "Kafka": true,
	"Terraform": true, "Ansible": true, "Jenkins": true, "Slack": true,
	"Chrome": true, "Firefox": true, "Safari": true, "Android": true,
	"Apple": true, "Microsoft": true, "Amazon": true, "Meta": true,
	"Nvidia": true, "Intel": true, "Ubuntu": true, "Debian": true,
	"Fedora": true, "Alpine": true, "Homebrew": true, "Bash": true,
	"Emacs": true, "Xcode": true, "Gradle": true, "Maven": true,
	"Bazel": true, "Sentry": true, "Datadog": true, "Grafana": true,
	"Prometheus": true, "Elasticsearch": true, "Kibana": true, "Nginx": true,
	"Apache": true, "Firebase": true, "Stripe": true, "Shopify": true,
	"Salesforce": true, "Oracle": true, "Adobe": true, "Figma": true,
	"Notion": true, "Jira": true, "Confluence": true, "Asana": true,
	"Trello": true, "Zoom": true, "Outlook": true, "Gmail": true,
	"Drive": true, "Docs": true, "Sheets": true, "Maps": true,
	"YouTube": true, "Netflix": true, "Spotify": true, "Monday": true,
	"Tuesday": true, "Wednesday": true, "Thursday": true, "Friday": true,
	"Saturday": true, "Sunday": true, "January": true, "February": true,
	"March": true, "April": true, "June": true, "July": true, "August": true,
	"September": true, "October": true, "November": true, "December": true,
	"English": true, "Spanish": true, "French": true, "German": true,
	"Chinese": true, "Japanese": true, "Korean": true, "Russian": true,
	"Arabic": true, "Hindi": true, "American": true, "European": true,
}

var (
	punct2rePostVerbCompound = regexp.MustCompile(`(?i:\b(?:is|are|was|were|be|been|being|becomes|became|seems|seemed|remains|looks))[ \t]+([a-z]+-[a-z]+)\b`)
	punct2reRealTime         = regexp.MustCompile(`(?i:\bin)[ \t]+(real-time)\b`)
)

var punct2plainAdverbs = map[string]bool{
	"well": true, "best": true, "better": true, "worse": true, "worst": true,
	"ill": true, "much": true, "most": true, "least": true, "more": true,
	"less": true,
}

var punct2nonAdverbLy = map[string]bool{
	"early": true, "silly": true, "curly": true, "burly": true, "surly": true,
	"daily": true, "reply": true, "apply": true, "imply": true, "italy": true,
	"family": true, "supply": true, "likely": true, "timely": true,
	"costly": true, "deadly": true, "orderly": true, "elderly": true,
	"weekly": true, "monthly": true, "quarterly": true, "yearly": true,
	"hourly": true, "nightly": true, "friendly": true, "lonely": true,
	"lovely": true, "assembly": true, "anomaly": true, "monopoly": true,
	"multiply": true, "unlikely": true, "wobbly": true, "bubbly": true,
	"ghastly": true, "sickly": true, "portly": true, "prickly": true,
}

// DetectGooglePostVerbCompoundHyphen reports an adverb-led compound sitting in
// predicate position, where the hyphen is unnecessary.
func DetectGooglePostVerbCompoundHyphen(text string) []types.Violation {
	var out []types.Violation
	matches := punct2rePostVerbCompound.FindAllStringSubmatchIndex(text, -1)
	matches = append(matches, punct2reRealTime.FindAllStringSubmatchIndex(text, -1)...)
	seen := make(map[int]bool, len(matches))
	for _, m := range matches {
		start, end := m[2], m[3]
		if seen[start] {
			continue
		}
		seen[start] = true
		if end < len(text) && text[end] == '-' {
			continue
		}
		compound := text[start:end]
		if compound != "real-time" && !punct2isAdverbHead(compound[:strings.IndexByte(compound, '-')]) {
			continue
		}
		if punct2inCodeSpan(text, start) {
			continue
		}
		out = append(out, types.Violation{
			RuleID:          "post-verb-compound-hyphen",
			StartIndex:      start,
			EndIndex:        end,
			MatchedText:     compound,
			SuggestedChange: strings.Replace(compound, "-", " ", 1),
		})
	}
	return out
}

func punct2isAdverbHead(head string) bool {
	if punct2plainAdverbs[head] {
		return true
	}
	return len(head) >= 5 && strings.HasSuffix(head, "ly") && !punct2nonAdverbLy[head]
}
