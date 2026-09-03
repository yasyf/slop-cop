package detectors

import (
	"regexp"
	"sort"
	"strings"
	"unicode"

	"github.com/yasyf/slop-cop/internal/types"
)

const (
	code2ruleIdentifier = "unformatted-code-identifier"
	code2ruleUILabel    = "uppercase-ui-label"
	code2ruleVisual     = "visual-appearance-reference"
)

var code2reSkipSpan = regexp.MustCompile(strings.Join([]string{
	"(?s:```.*?```)",
	"(?s:~~~.*?~~~)",
	"(?s:``[^`]*``)",
	"(?:`[^`\n]*`)",
	"(?is:<code[^>]*>.*?</code>)",
	"(?m:^[ \t]{0,3}#{1,6}[ \t][^\n]*)",
	`(?:\[[^\[\]\n]*\](?:\([^)\n]*\)|\[[^\]\n]*\])?)`,
	`(?:\]\([^)\n]*\))`,
	`(?:<[^>\n]+>)`,
	`(?i:(?:https?://|www\.)[^\s)\]<>"']+)`,
}, "|"))

func code2mask(text string) string {
	spans := code2reSkipSpan.FindAllStringIndex(text, -1)
	if len(spans) == 0 {
		return text
	}
	b := []byte(text)
	for _, idx := range spans {
		for i := idx[0]; i < idx[1]; i++ {
			if b[i] != '\n' {
				b[i] = ' '
			}
		}
	}
	return string(b)
}

var code2reIdentPlain = []*regexp.Regexp{
	regexp.MustCompile(`\b[a-z][a-z0-9]*[A-Z][A-Za-z0-9]*\b`),
	regexp.MustCompile(`\b[A-Z][a-z0-9]+[A-Z][A-Za-z0-9]*\b`),
	regexp.MustCompile(`\b[a-z0-9]+_[a-z0-9_]+\b`),
	regexp.MustCompile(`\b[A-Za-z_][A-Za-z0-9_]*\(\)`),
	regexp.MustCompile(`\b[\w.-]+\.(?:go|py|js|ts|tsx|jsx|json|ya?ml|html|css|sh|toml|xml|proto|sql|cfg|ini|env|conf|log|csv|txt|tf|jar)\b`),
	regexp.MustCompile(`\b[A-Z][A-Z0-9]*(?:_[A-Z0-9]+)+\b`),
}

var code2reIdentGroup = []*regexp.Regexp{
	regexp.MustCompile(`(?m)(?:^|\s)(/(?:[\w.-]+/)+[\w.-]*)`),
	regexp.MustCompile(`(?m)(?:^|\s)(--[a-z][\w-]{2,})\b`),
}

var (
	code2reHTTPVerb   = regexp.MustCompile(`\b(?:GET|POST|PUT|PATCH|DELETE)\b`)
	code2reStatusCode = regexp.MustCompile(`\b[1-5]\d\d\b`)
)

var code2httpContext = map[string]bool{
	"http": true, "https": true, "request": true, "requests": true,
	"endpoint": true, "endpoints": true, "method": true, "methods": true,
	"api": true, "url": true, "uri": true, "verb": true, "curl": true,
	"response": true, "route": true, "handler": true, "payload": true,
}

var code2statusContext = map[string]bool{
	"status": true, "statuses": true, "error": true, "errors": true,
	"response": true, "responses": true, "http": true, "https": true,
}

var code2properNouns = code2lowerSet([]string{
	"JavaScript", "TypeScript", "CoffeeScript", "ActionScript", "GitHub", "GitLab",
	"YouTube", "PostgreSQL", "JavaDoc", "PowerShell", "GraphQL", "WebSocket",
	"WebSockets", "JavaBeans", "WebAssembly", "WebRTC", "WebGL", "WebKit",
	"OpenAI", "OpenAPI", "OpenSSL", "OpenSSH", "OpenSearch", "OpenTelemetry",
	"DeepMind", "ChatGPT", "SharePoint", "OneDrive", "PowerPoint", "PowerBI",
	"LinkedIn", "MySQL", "MariaDB", "MongoDB", "DynamoDB", "CockroachDB",
	"BigQuery", "BigTable", "CloudSQL", "CloudFront", "CloudFlare", "CloudWatch",
	"CloudFormation", "AppEngine", "FireStore", "PubSub", "TestFlight",
	"JetBrains", "IntelliJ", "WebStorm", "PyCharm", "GoLand", "DataGrip",
	"VSCode", "ChromeOS", "FreeBSD", "NetBSD", "OpenBSD", "RabbitMQ",
	"ElasticSearch", "PagerDuty", "DigitalOcean", "PlayStation", "AirPods",
	"MacBook", "McAfee", "McDonald", "TikTok", "WhatsApp", "SnapChat",
	"WordPress", "MailChimp", "HubSpot", "SourceTree", "GitKraken",
	"TensorFlow", "PyTorch", "NumPy", "SciPy", "JupyterLab", "MathML",
	"MathJax", "LaTeX", "BibTeX", "JSONPath", "SwiftUI", "UIKit", "SQLAlchemy",
	"iOS", "iPadOS", "macOS", "tvOS", "watchOS", "iPhone", "iPad", "iMac",
	"iCloud", "iTunes", "eBay", "eSIM", "jQuery", "npm", "PyPI", "OAuth",
	"PoC", "PoCs", "IoT", "QoS", "SoC", "SoCs", "ToS", "DoS", "DDoS",
	"gRPC", "mTLS", "eBPF", "gVisor", "xUnit", "dApp", "dApps",
	"Node.js", "Next.js", "Nuxt.js", "Vue.js", "Nest.js", "Ember.js",
	"Backbone.js", "Chart.js", "Three.js", "Express.js", "Alpine.js", "D3.js",
})

// code2unitAbbrevs are byte-size units whose mixed case is the unit's own
// spelling, not camel case dragged out of source.
var code2unitAbbrevs = code2lowerSet([]string{
	"KiB", "MiB", "GiB", "TiB", "PiB", "EiB", "ZiB", "YiB", "kB",
})

func code2lowerSet(words []string) map[string]bool {
	out := make(map[string]bool, len(words))
	for _, w := range words {
		out[strings.ToLower(w)] = true
	}
	return out
}

func code2cleanToken(w string) string {
	return strings.ToLower(strings.Trim(w, ".,;:!?()[]{}\"'`*_"))
}

func code2contextTokens(text string, start, end, n int) []string {
	lo := start - 120
	if lo < 0 {
		lo = 0
	}
	hi := end + 120
	if hi > len(text) {
		hi = len(text)
	}
	before := strings.Fields(text[lo:start])
	if len(before) > n {
		before = before[len(before)-n:]
	}
	after := strings.Fields(text[end:hi])
	if len(after) > n {
		after = after[:n]
	}
	out := make([]string, 0, len(before)+len(after))
	for _, w := range before {
		out = append(out, code2cleanToken(w))
	}
	for _, w := range after {
		out = append(out, code2cleanToken(w))
	}
	return out
}

func code2dedupeOverlap(v []types.Violation) []types.Violation {
	sort.SliceStable(v, func(i, j int) bool {
		if v[i].StartIndex != v[j].StartIndex {
			return v[i].StartIndex < v[j].StartIndex
		}
		return v[i].EndIndex > v[j].EndIndex
	})
	out := v[:0]
	last := -1
	for _, x := range v {
		if x.StartIndex < last {
			continue
		}
		out = append(out, x)
		last = x.EndIndex
	}
	return out
}

// DetectGoogleUnformattedCodeIdentifier reports identifiers, filenames, paths,
// flags, and HTTP tokens that are left in body font instead of code font.
func DetectGoogleUnformattedCodeIdentifier(text string) []types.Violation {
	masked := code2mask(text)
	var out []types.Violation
	for _, re := range code2reIdentPlain {
		for _, idx := range re.FindAllStringIndex(masked, -1) {
			out = code2appendIdent(out, text, idx[0], idx[1])
		}
	}
	for _, re := range code2reIdentGroup {
		for _, m := range re.FindAllStringSubmatchIndex(masked, -1) {
			end := m[3]
			for end > m[2] && strings.IndexByte(".,;:", text[end-1]) >= 0 {
				end--
			}
			out = code2appendIdent(out, text, m[2], end)
		}
	}
	for _, idx := range code2reHTTPVerb.FindAllStringIndex(masked, -1) {
		if code2hasHTTPContext(masked, idx[0], idx[1]) {
			out = code2appendIdent(out, text, idx[0], idx[1])
		}
	}
	for _, idx := range code2reStatusCode.FindAllStringIndex(masked, -1) {
		if code2hasAnyToken(masked, idx[0], idx[1], 3, code2statusContext) {
			out = code2appendIdent(out, text, idx[0], idx[1])
		}
	}
	return code2dedupeOverlap(out)
}

func code2appendIdent(out []types.Violation, text string, start, end int) []types.Violation {
	matched := text[start:end]
	lowered := strings.ToLower(matched)
	if code2properNouns[lowered] || code2unitAbbrevs[lowered] {
		return out
	}
	return append(out, types.Violation{
		RuleID:          code2ruleIdentifier,
		StartIndex:      start,
		EndIndex:        end,
		MatchedText:     matched,
		SuggestedChange: "`" + matched + "`",
	})
}

func code2hasAnyToken(text string, start, end, n int, words map[string]bool) bool {
	for _, tok := range code2contextTokens(text, start, end, n) {
		if words[tok] {
			return true
		}
	}
	return false
}

func code2hasHTTPContext(text string, start, end int) bool {
	for _, tok := range code2contextTokens(text, start, end, 5) {
		if code2httpContext[tok] || strings.HasPrefix(tok, "/") {
			return true
		}
	}
	return false
}

var code2reUILabel = regexp.MustCompile(
	`\b((?i:click|tap|select|choose|go to))[ \t]+\*{0,2}([A-Z]{3,}(?:[ \t]+[A-Z]{2,})*)\*{0,2}`)

var code2uiAcronyms = map[string]bool{
	"OK": true, "SQL": true, "API": true, "VM": true, "ID": true, "URL": true,
	"JSON": true, "XML": true, "HTTP": true, "HTTPS": true, "IP": true,
	"CSV": true, "PDF": true, "UI": true, "CPU": true, "GPU": true, "SSH": true,
	"DNS": true, "TLS": true, "SSL": true, "YAML": true, "CLI": true,
	"SDK": true, "IAM": true, "VPC": true, "CI": true, "CD": true, "OS": true,
	"RPC": true, "JWT": true, "TTL": true, "SSO": true, "MFA": true,
}

// DetectGoogleUppercaseUILabel reports a UI label reproduced in all caps.
func DetectGoogleUppercaseUILabel(text string) []types.Violation {
	var out []types.Violation
	for _, m := range code2reUILabel.FindAllStringSubmatchIndex(text, -1) {
		verb := text[m[2]:m[3]]
		if verb == strings.ToUpper(verb) {
			continue
		}
		label := text[m[4]:m[5]]
		if code2allAcronyms(label) {
			continue
		}
		out = append(out, types.Violation{
			RuleID:          code2ruleUILabel,
			StartIndex:      m[4],
			EndIndex:        m[5],
			MatchedText:     label,
			SuggestedChange: code2sentenceCase(label),
		})
	}
	return out
}

func code2allAcronyms(label string) bool {
	for _, w := range strings.Fields(label) {
		if !code2uiAcronyms[w] {
			return false
		}
	}
	return true
}

func code2sentenceCase(label string) string {
	words := strings.Fields(label)
	for i, w := range words {
		if code2uiAcronyms[w] {
			continue
		}
		lowered := []rune(strings.ToLower(w))
		if i == 0 {
			lowered[0] = unicode.ToUpper(lowered[0])
		}
		words[i] = string(lowered)
	}
	return strings.Join(words, " ")
}

var code2visualSubs = []plainSub{
	{"click the bell icon", "click Notifications"},
	{"the button with three lines", "Menu"},
	{"the green check mark", "the Ready status"},
}

var code2visualRes = compileSubs(code2visualSubs)

var code2reVisual = []*regexp.Regexp{
	regexp.MustCompile(`\b(?i:click|tap|select|press|choose|find|look for)\s+(?i:on\s+)?(?i:the)\s+(?:[a-z-]+\s+){0,3}(?i:icon|glyph|symbol|arrow|button with|three (?:dots|lines|bars))\b`),
	regexp.MustCompile(`\b(?i:click|tap|select|press)\s+(?i:the)\s+(?i:icon|button|link|arrow|control)\s*[.,;:]`),
	regexp.MustCompile(`(?i)\bthe (?:red|green|blue|yellow|orange|gray|grey|black|white) (?:button|link|text|box|indicator|light|bar|highlight|check ?mark|banner)\b`),
	regexp.MustCompile(`(?i)\bthe (?:bell|gear|cog|hamburger|kebab|magnifying glass|pencil|trash|floppy|plus|minus) (?:icon|button|symbol)\b`),
}

var code2arrowDirections = map[string]bool{
	"up": true, "down": true, "left": true, "right": true,
	"back": true, "forward": true, "next": true, "previous": true,
}

var code2greenLightVerbs = map[string]bool{
	"gave": true, "give": true, "given": true, "gives": true, "giving": true,
	"got": true, "get": true, "gets": true, "getting": true,
}

// DetectGoogleVisualAppearanceReference reports a control named by its look, or
// pointed at without a label.
func DetectGoogleVisualAppearanceReference(text string) []types.Violation {
	out := findSubs(text, code2ruleVisual, code2visualSubs, code2visualRes)
	tabled := append([]types.Violation(nil), out...)
	for _, re := range code2reVisual {
		for _, idx := range re.FindAllStringIndex(text, -1) {
			if code2isKeyboardArrow(text, idx[0], idx[1]) || code2isGreenLightIdiom(text, idx[0], idx[1]) {
				continue
			}
			if code2overlaps(tabled, idx[0], idx[1]) {
				continue
			}
			start, end := trimSpan(text, idx[0], idx[1])
			for end > start && strings.IndexByte(".,;:", text[end-1]) >= 0 {
				end--
			}
			out = append(out, types.Violation{
				RuleID:      code2ruleVisual,
				StartIndex:  start,
				EndIndex:    end,
				MatchedText: text[start:end],
			})
		}
	}
	return code2dedupeOverlap(out)
}

func code2overlaps(spans []types.Violation, start, end int) bool {
	for _, s := range spans {
		if start < s.EndIndex && s.StartIndex < end {
			return true
		}
	}
	return false
}

func code2isKeyboardArrow(text string, start, end int) bool {
	words := strings.Fields(strings.ToLower(strings.Trim(text[start:end], ".,;: ")))
	if len(words) == 0 || words[len(words)-1] != "arrow" {
		return false
	}
	if len(words) >= 2 && code2arrowDirections[words[len(words)-2]] {
		return true
	}
	return strings.HasPrefix(code2nextToken(text, end), "key")
}

func code2isGreenLightIdiom(text string, start, end int) bool {
	if !strings.EqualFold(strings.TrimSpace(text[start:end]), "the green light") {
		return false
	}
	return code2greenLightVerbs[code2prevToken(text, start)]
}

func code2prevToken(text string, start int) string {
	lo := start - 40
	if lo < 0 {
		lo = 0
	}
	words := strings.Fields(text[lo:start])
	if len(words) == 0 {
		return ""
	}
	return code2cleanToken(words[len(words)-1])
}

func code2nextToken(text string, end int) string {
	hi := end + 40
	if hi > len(text) {
		hi = len(text)
	}
	words := strings.Fields(text[end:hi])
	if len(words) == 0 {
		return ""
	}
	return code2cleanToken(words[0])
}
