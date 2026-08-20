package detectors

import (
	"regexp"
	"strings"

	"github.com/yasyf/slop-cop/internal/types"
)

func words3set(words string) map[string]bool {
	out := make(map[string]bool)
	for _, w := range strings.Fields(words) {
		out[w] = true
	}
	return out
}

var (
	words3reCodeSpan     = regexp.MustCompile("`[^`\n]*`")
	words3reWord         = regexp.MustCompile(`[A-Za-z][A-Za-z0-9'\x{2019}-]*`)
	words3reTrailingWord = regexp.MustCompile(`([A-Za-z][A-Za-z'\x{2019}-]*)[^A-Za-z]*$`)
	words3reNextWord     = regexp.MustCompile(`^(?:[ \t]+|[ \t]*\r?\n[ \t]*)([A-Za-z][A-Za-z'\x{2019}-]*)`)
	words3reParenGroup   = regexp.MustCompile(`\(([^()\n]{1,120})\)`)
)

func words3inCodeSpan(spans [][]int, pos int) bool {
	for _, s := range spans {
		if pos >= s[0] && pos < s[1] {
			return true
		}
	}
	return false
}

func words3identAdjacent(text string, start, end int) bool {
	if start > 0 && strings.IndexByte(`/-_.$@\`, text[start-1]) >= 0 {
		return true
	}
	return end < len(text) && strings.IndexByte(`/-_(=\`, text[end]) >= 0
}

func words3prevWord(text string, pos int) string {
	lo := pos - 48
	if lo < 0 {
		lo = 0
	}
	m := words3reTrailingWord.FindStringSubmatch(text[lo:pos])
	if m == nil {
		return ""
	}
	return strings.ToLower(m[1])
}

func words3nextWord(text string, pos int) string {
	hi := pos + 48
	if hi > len(text) {
		hi = len(text)
	}
	m := words3reNextWord.FindStringSubmatch(text[pos:hi])
	if m == nil {
		return ""
	}
	return strings.ToLower(m[1])
}

func words3norm(s string) string { return strings.ToLower(strings.Join(strings.Fields(s), " ")) }

func words3matchCase(src, repl string) string {
	if src == "" || repl == "" || src[0] < 'A' || src[0] > 'Z' {
		return repl
	}
	return strings.ToUpper(repl[:1]) + repl[1:]
}

func words3lineAt(text string, pos int) string {
	start := strings.LastIndexByte(text[:pos], '\n') + 1
	end := len(text)
	if p := strings.IndexByte(text[pos:], '\n'); p >= 0 {
		end = pos + p
	}
	return text[start:end]
}

var words3knownAbbrev = words3set(`
	AI API CLI CPU CSV DNS DVD EU FAQ GB GPU HTML HTTP HTTPS ID IDE IO IP JSON
	KB MB MIB ML OK OS PDF PNG RAM REST ROM SDK SQL SSH TB TCP TLS UDP UI UK
	URI URL US USB UTC UX VM XML YAML
	NOTE TODO TIP FIXME HACK XXX WARN TRUE FALSE NULL NONE ALL ANY NEW OLD YES
	NO NOT AND OR MUST MAY SHALL SHOULD CAN ONLY THE IF IS IT OF ON TO UP WE AS
	AT BY IN SO DO PASS FAIL DONE OFF ONE TWO END
	GET POST PUT PATCH DELETE HEAD TRACE
	CSS JS TS TSV TOML INI ENV PATH HOME ASCII UTF MD RFC ISO ANSI SI LTS EOL
	EOF GUI TTY SSD HDD NIC LAN WAN VPN VPC SSL JWT ACL ARN AWS GCP IAM RDS CDN
	DHCP FTP SFTP SMTP IMAP POP SSO LDAP SAML RBAC CORS CRUD DOM SVG JPEG JPG
	GIF ZIP TAR GZIP RPM DEB NPM PIP RSA AES SHA HMAC TTL MAC BIOS UEFI ARM
	RISC GPL MIT BSD EULA OSS SLA SLO SLI KPI ETA FYI AKA ASAP IMO PR MR CI CD
	QA DEV PROD TEST UAT ORM MVC MVP POC ADR SPA PWA SEO RSS GPS HDMI VGA LED
	LCD XLS DOC PPT ODF EPUB CRC UUID GUID ULID INF MIN MAX SUM AVG STD ABS LOG
	EXP TPU NPU DPU NVME SATA PCIE RAID RPC GRPC SOAP WSDL XSD GMT PST EST AM
	PM PB EB KIB GIB TIB JVM JDK JRE JAR WAR CLR NET PHP LLM RAG NLP OCR ASR
	TTS GAN CNN RNN LSTM BERT PII PHI KYC AML FIFO LIFO LRU MRU ETL OLAP OLTP
	DAG BI XSS CSRF DDOS DOS MITM URN QUIC SCP OWASP NIST GDPR HIPAA PCI FIPS
	CVE CWE OAUTH IBM NASA IEEE ACM GNU FSF
`)

var words3englishAllCaps = words3set(`
	ALWAYS NEVER EVER STILL JUST VERY REALLY EXACTLY ALSO EVEN RATHER
	BLOCK BLOCKS BLOCKED ALLOW ALLOWS DENY STOP START MAKE KEEP DROP ADD
	USE USES USED NEED NEEDS READ WRITE OPEN CLOSE RUN RUNS FIX WORK WORKS
	EVERY EACH SOME BOTH MANY MOST MORE LESS FEW
	BEFORE AFTER FIRST LAST NEXT THEN THAN THIS THAT THESE THOSE
	WHEN WHERE WHAT WHY HOW WHO WHICH HERE THERE NOW SOON LATER
	WILL WOULD COULD MIGHT CANNOT WONT DONT DOES DID
	GOOD BAD BEST WORST SAFE REAL
	WITH FROM INTO ONTO OVER UNDER ABOVE BELOW OUT DOWN PER
	ARE WAS WERE BEEN HAVE HAS HAD BE GO GOES
	YOU YOUR OUR THEY THEM ITS
	WARNING CAUTION DANGER IMPORTANT REQUIRED
	NOTHING ANYTHING EVERYTHING SOMETHING
`)

var words3docFilename = words3set(`
	README NOTICE LICENSE LICENCE COPYING CHANGELOG CONTRIBUTING CODEOWNERS
	AUTHORS SECURITY MAKEFILE DOCKERFILE AGENTS CLAUDE GEMINI SKILL DOCS
`)

var (
	words3reAbbrevToken = regexp.MustCompile(`\b[A-Z][A-Z0-9]{1,5}s?\b`)
	words3reFileSuffix  = regexp.MustCompile(`^\.[a-z][a-z0-9]{0,4}\b`)
)

const (
	words3abbrevWindow = 80
	words3headingScan  = 400
)

// DetectGoogleUndefinedAbbreviation reports the first use of an abbreviation
// that the surrounding prose never expands.
func DetectGoogleUndefinedAbbreviation(text string) []types.Violation {
	const rule = "undefined-abbreviation"
	spans := words3reCodeSpan.FindAllStringIndex(text, -1)
	seen := make(map[string]bool)
	var out []types.Violation
	for _, idx := range words3reAbbrevToken.FindAllStringIndex(text, -1) {
		token := text[idx[0]:idx[1]]
		base := strings.TrimSuffix(token, "s")
		if len(base) < 2 || words3hasDigit(base) || words3knownAbbrev[base] {
			continue
		}
		if words3englishAllCaps[base] || words3docFilename[base] {
			continue
		}
		if words3inCodeSpan(spans, idx[0]) || words3identAdjacent(text, idx[0], idx[1]) {
			continue
		}
		if words3reFileSuffix.MatchString(text[idx[1]:]) {
			continue
		}
		line := words3lineAt(text, idx[0])
		if words3shoutyLine(line) {
			continue
		}
		if seen[base] {
			continue
		}
		seen[base] = true
		lo := idx[0] - words3abbrevWindow
		if lo < 0 {
			lo = 0
		}
		hi := idx[1] + words3abbrevWindow
		if isATXHeading(line) {
			hi = idx[1] + words3headingScan
		}
		if hi > len(text) {
			hi = len(text)
		}
		if words3definedIn(text[lo:hi], base) {
			continue
		}
		out = append(out, types.Violation{
			RuleID:      rule,
			StartIndex:  idx[0],
			EndIndex:    idx[1],
			MatchedText: token,
			Explanation: "An abbreviation is spelled out at its first use.",
		})
	}
	return out
}

func words3hasDigit(s string) bool {
	for i := 0; i < len(s); i++ {
		if s[i] >= '0' && s[i] <= '9' {
			return true
		}
	}
	return false
}

func words3shoutyLine(line string) bool {
	words := words3reWord.FindAllString(line, -1)
	if len(words) < 2 {
		return false
	}
	for _, w := range words {
		if strings.ToUpper(w) != w {
			return false
		}
	}
	return true
}

func words3definedIn(segment, abbr string) bool {
	upper := strings.ToUpper(abbr)
	for _, m := range words3reParenGroup.FindAllStringSubmatchIndex(segment, -1) {
		words := words3reWord.FindAllString(segment[m[2]:m[3]], -1)
		if len(words) == 0 {
			continue
		}
		if strings.EqualFold(words[len(words)-1], abbr) {
			if strings.HasSuffix(words3initials(segment[:m[0]]), upper) {
				return true
			}
			continue
		}
		if strings.HasSuffix(words3initials(segment[m[2]:m[3]]), upper) {
			return true
		}
	}
	return false
}

func words3initials(s string) string {
	words := words3reWord.FindAllString(s, -1)
	if len(words) > 12 {
		words = words[len(words)-12:]
	}
	var b strings.Builder
	for _, w := range words {
		b.WriteByte(w[0])
	}
	return strings.ToUpper(b.String())
}

var words3underspecSubs = []plainSub{
	{"MIME type", "media type"},
	{"MIME types", "media types"},
	{"MIME-type", "media type"},
	{"unix time", "Unix epoch time"},
	{"epoch time", "Unix epoch time"},
	{"cellular data", "mobile data"},
	{"cellular network", "mobile network"},
	{"cellular networks", "mobile networks"},
	{"interconnect type", "connection type"},
	{"interconnect types", "connection types"},
}

var words3underspecRes = compileSubs(words3underspecSubs)

var (
	words3reKeyNoun = regexp.MustCompile(`(?i)\bkey[ \t]+(features?|benefits?|components?|concepts?|differences?|points?|factors?|takeaways?|advantages?|insights?)\b`)
	words3reMobile  = regexp.MustCompile(`(?i)\bmobile\b`)
	words3reIngest  = regexp.MustCompile(`(?i)\bingest(?:s|ed|ing)?\b`)
)

var words3keyIsALiteralKey = words3set(`
	api encryption ssh public private primary foreign license access secret
	signing host partition sort shift arrow escape enter modifier session
	cryptographic account service registry
`)

var words3mobileConnectors = words3set(`and or but`)

var words3ingestForms = map[string]string{
	"ingest":    "import",
	"ingests":   "imports",
	"ingested":  "imported",
	"ingesting": "importing",
}

// DetectGoogleUnderspecifiedTerm reports technical nouns that name several
// things at once where a settled fuller form exists.
func DetectGoogleUnderspecifiedTerm(text string) []types.Violation {
	const rule = "underspecified-term"
	spans := words3reCodeSpan.FindAllStringIndex(text, -1)
	matches := findSubs(text, rule, words3underspecSubs, words3underspecRes)
	out := matches[:0]
	for _, v := range matches {
		if words3norm(v.MatchedText) == "epoch time" && words3prevWord(text, v.StartIndex) == "unix" {
			continue
		}
		out = append(out, v)
	}
	for _, m := range words3reKeyNoun.FindAllStringSubmatchIndex(text, -1) {
		if words3keyIsALiteralKey[words3prevWord(text, m[0])] {
			continue
		}
		out = append(out, types.Violation{
			RuleID:          rule,
			StartIndex:      m[0],
			EndIndex:        m[1],
			MatchedText:     text[m[0]:m[1]],
			Explanation:     `"Key" rates the item instead of naming it.`,
			SuggestedChange: text[m[2]:m[3]],
		})
	}
	for _, idx := range words3reMobile.FindAllStringIndex(text, -1) {
		if idx[0] > 0 && text[idx[0]-1] == '-' {
			continue
		}
		if next := words3nextWord(text, idx[1]); next != "" && !words3mobileConnectors[next] {
			continue
		}
		out = append(out, types.Violation{
			RuleID:          rule,
			StartIndex:      idx[0],
			EndIndex:        idx[1],
			MatchedText:     text[idx[0]:idx[1]],
			Explanation:     `"Mobile" on its own names neither the device nor the network.`,
			SuggestedChange: words3matchCase(text[idx[0]:idx[1]], "mobile devices"),
		})
	}
	for _, idx := range words3reIngest.FindAllStringIndex(text, -1) {
		if words3inCodeSpan(spans, idx[0]) {
			continue
		}
		match := text[idx[0]:idx[1]]
		out = append(out, types.Violation{
			RuleID:          rule,
			StartIndex:      idx[0],
			EndIndex:        idx[1],
			MatchedText:     match,
			Explanation:     `"Ingest" is reserved for a genuinely heavy-processing load.`,
			SuggestedChange: words3matchCase(match, words3ingestForms[strings.ToLower(match)]),
		})
	}
	return out
}

var words3reUnitAgreement = regexp.MustCompile(`\b(\d+(?:\.\d+)?)[ \t]+(second|minute|hour|day|week|month|year|degree|byte|bit|pixel|character|core|node|replica|request|instance|attempt|retry|item|row|column)(s?)\b`)

var words3unitFollowers = words3set(`
	of in to for per and or with from at on is are was were that which than
	into over under after before then but by as when while unless if because
`)

// DetectGoogleUnitNumberAgreement reports a spelled-out unit whose number
// disagrees with it.
func DetectGoogleUnitNumberAgreement(text string) []types.Violation {
	const rule = "unit-number-agreement"
	var out []types.Violation
	for _, m := range words3reUnitAgreement.FindAllStringSubmatchIndex(text, -1) {
		if m[0] > 0 && strings.IndexByte(`$#-.`, text[m[0]-1]) >= 0 {
			continue
		}
		number := text[m[2]:m[3]]
		unit := text[m[4]:m[5]]
		plural := m[7] > m[6]
		var suggestion string
		switch {
		case number == "1" && plural:
			suggestion = number + " " + unit
		case number != "1" && !plural:
			next := words3nextWord(text, m[1])
			if next != "" && !words3unitFollowers[next] {
				continue
			}
			suggestion = number + " " + words3pluralize(unit)
		default:
			continue
		}
		out = append(out, types.Violation{
			RuleID:          rule,
			StartIndex:      m[0],
			EndIndex:        m[1],
			MatchedText:     text[m[0]:m[1]],
			Explanation:     "A spelled-out unit is singular only after exactly 1.",
			SuggestedChange: suggestion,
		})
	}
	return out
}

func words3pluralize(unit string) string {
	if strings.HasSuffix(unit, "y") {
		return unit[:len(unit)-1] + "ies"
	}
	return unit + "s"
}

var (
	words3reClosedVerb = regexp.MustCompile(`(?i)\b((?:to|can|could|must|should|will|would|please|don['\x{2019}]t|doesn['\x{2019}]t|didn['\x{2019}]t|helps you|before you|after you|then)[ \t]+)(setup|startup|timeout|signin|signout)\b`)
	words3reOpenNoun   = regexp.MustCompile(`(?i)\b((?:the|a|an|your|our|its|any|initial|one-time)[ \t]+)(set[ \t]+up|start[ \t]+up|time[ \t]+out|sign[ \t]+in|sign[ \t]+out)\b`)
	words3reThirdParty = regexp.MustCompile(`(?i)\b((?:a|the|any|each)[ \t]+)third-party\b`)
)

var words3openForms = map[string]string{
	"setup":   "set up",
	"startup": "start up",
	"timeout": "time out",
	"signin":  "sign in",
	"signout": "sign out",
}

var words3closedForms = map[string]string{
	"set up":   "setup",
	"start up": "startup",
	"time out": "timeout",
	"sign in":  "sign-in",
	"sign out": "sign-out",
}

var words3fixedTermSubs = []plainSub{
	{"log in", "sign in"},
	{"login", "sign-in"},
	{"log out", "sign out"},
	{"logout", "sign-out"},
	{"logging in", "signing in"},
	{"logging out", "signing out"},
	{"sign into", "sign in to"},
}

var words3fixedTermRes = compileSubs(words3fixedTermSubs)

var words3logIsANoun = words3set(`
	the a an this that these those your our its their each every any some one
	no error audit build access system application server event debug change
	git binary json structured
`)

var words3loggingFollowers = words3set(`to of and or`)

// DetectGoogleVerbNounCompound reports a compound spelled for the wrong part of
// speech, plus the fixed terms Google settles in one direction.
func DetectGoogleVerbNounCompound(text string) []types.Violation {
	const rule = "verb-noun-compound"
	spans := words3reCodeSpan.FindAllStringIndex(text, -1)
	var out []types.Violation
	for _, m := range words3reClosedVerb.FindAllStringSubmatchIndex(text, -1) {
		if words3inCodeSpan(spans, m[0]) {
			continue
		}
		out = append(out, types.Violation{
			RuleID:          rule,
			StartIndex:      m[0],
			EndIndex:        m[1],
			MatchedText:     text[m[0]:m[1]],
			Explanation:     "The verb is two words; the closed form is the noun.",
			SuggestedChange: text[m[2]:m[3]] + words3openForms[strings.ToLower(text[m[4]:m[5]])],
		})
	}
	for _, m := range words3reOpenNoun.FindAllStringSubmatchIndex(text, -1) {
		if words3inCodeSpan(spans, m[0]) {
			continue
		}
		out = append(out, types.Violation{
			RuleID:          rule,
			StartIndex:      m[0],
			EndIndex:        m[1],
			MatchedText:     text[m[0]:m[1]],
			Explanation:     "The noun is one word; the open form is the verb.",
			SuggestedChange: text[m[2]:m[3]] + words3closedForms[words3norm(text[m[4]:m[5]])],
		})
	}
	for _, v := range findSubs(text, rule, words3fixedTermSubs, words3fixedTermRes) {
		if words3inCodeSpan(spans, v.StartIndex) || !words3fixedTermFires(text, v) {
			continue
		}
		v.Explanation = "Google settles this term in one direction."
		v.SuggestedChange = words3matchCase(v.MatchedText, v.SuggestedChange)
		out = append(out, v)
	}
	for _, m := range words3reThirdParty.FindAllStringSubmatchIndex(text, -1) {
		if words3nextWord(text, m[1]) != "" || words3inCodeSpan(spans, m[0]) {
			continue
		}
		out = append(out, types.Violation{
			RuleID:          rule,
			StartIndex:      m[0],
			EndIndex:        m[1],
			MatchedText:     text[m[0]:m[1]],
			Explanation:     "The noun is two open words; the hyphen belongs to the modifier.",
			SuggestedChange: text[m[2]:m[3]] + "third party",
		})
	}
	return out
}

func words3fixedTermFires(text string, v types.Violation) bool {
	switch words3norm(v.MatchedText) {
	case "log in", "log out":
		return !words3logIsANoun[words3prevWord(text, v.StartIndex)]
	case "login", "logout":
		return !words3identAdjacent(text, v.StartIndex, v.EndIndex)
	case "logging in", "logging out":
		next := words3nextWord(text, v.EndIndex)
		return next == "" || words3loggingFollowers[next]
	}
	return true
}

var (
	words3reVersionRange = regexp.MustCompile(`(?i)\b(?:(?:version|v)\.?[ \t]*\d+(?:\.\d+)*|\d+\.\d+(?:\.\d+)*)[ \t]+(?:and|or)[ \t]+(above|higher|below|lower)\b`)
	words3reVersionPlus  = regexp.MustCompile(`\b(v?\d+(?:\.\d+)+)\+`)
	words3reDocPosition  = regexp.MustCompile(`(?i)\b(higher|lower)([ \t]+(?:in|on)[ \t]+this[ \t]+(?:document|page|section))\b`)
	words3reUnderVersion = regexp.MustCompile(`(?i)\bunder[ \t]+((?:version|v)\.?[ \t]*\d+(?:\.\d+)*)\b`)
)

var words3timeAxis = map[string]string{
	"above":  "later",
	"higher": "later",
	"below":  "earlier",
	"lower":  "earlier",
}

var words3docAxis = map[string]string{
	"higher": "earlier",
	"lower":  "later",
}

// DetectGoogleVersionRangeComparator reports a version range stated as
// magnitude rather than as release order.
func DetectGoogleVersionRangeComparator(text string) []types.Violation {
	const rule = "version-range-comparator"
	const explanation = "Releases are ordered in time, not by magnitude."
	spans := words3reCodeSpan.FindAllStringIndex(text, -1)
	var out []types.Violation
	for _, m := range words3reVersionRange.FindAllStringSubmatchIndex(text, -1) {
		if words3inCodeSpan(spans, m[0]) {
			continue
		}
		out = append(out, types.Violation{
			RuleID:          rule,
			StartIndex:      m[0],
			EndIndex:        m[1],
			MatchedText:     text[m[0]:m[1]],
			Explanation:     explanation,
			SuggestedChange: text[m[0]:m[2]] + words3timeAxis[strings.ToLower(text[m[2]:m[3]])],
		})
	}
	for _, m := range words3reVersionPlus.FindAllStringSubmatchIndex(text, -1) {
		if words3inCodeSpan(spans, m[0]) || words3plusIsFused(text, m[1]) {
			continue
		}
		out = append(out, types.Violation{
			RuleID:          rule,
			StartIndex:      m[0],
			EndIndex:        m[1],
			MatchedText:     text[m[0]:m[1]],
			Explanation:     explanation,
			SuggestedChange: text[m[2]:m[3]] + " or later",
		})
	}
	for _, m := range words3reDocPosition.FindAllStringSubmatchIndex(text, -1) {
		if words3inCodeSpan(spans, m[0]) {
			continue
		}
		out = append(out, types.Violation{
			RuleID:          rule,
			StartIndex:      m[0],
			EndIndex:        m[1],
			MatchedText:     text[m[0]:m[1]],
			Explanation:     "A position in a document is earlier or later, not higher or lower.",
			SuggestedChange: words3matchCase(text[m[2]:m[3]], words3docAxis[strings.ToLower(text[m[2]:m[3]])]) + text[m[4]:m[5]],
		})
	}
	for _, m := range words3reUnderVersion.FindAllStringSubmatchIndex(text, -1) {
		if words3inCodeSpan(spans, m[0]) {
			continue
		}
		out = append(out, types.Violation{
			RuleID:          rule,
			StartIndex:      m[0],
			EndIndex:        m[1],
			MatchedText:     text[m[0]:m[1]],
			Explanation:     explanation,
			SuggestedChange: text[m[2]:m[3]] + " and earlier",
		})
	}
	return out
}

func words3plusIsFused(text string, end int) bool {
	if end >= len(text) {
		return false
	}
	b := text[end]
	return b == '+' || b == '=' || b == '_' || (b >= '0' && b <= '9') || (b >= 'a' && b <= 'z') || (b >= 'A' && b <= 'Z')
}
