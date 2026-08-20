package detectors

import (
	"regexp"
	"sort"
	"strings"
	"unicode"

	"github.com/yasyf/slop-cop/internal/types"
)

const words1punct = `.,;:!?"'()[]{}`

var words1reWord = regexp.MustCompile(`[A-Za-z]+`)

var words1expansionStop = map[string]bool{
	"of": true, "the": true, "and": true, "for": true, "in": true, "to": true,
}

func words1words(s string) []string {
	var out []string
	for _, f := range strings.Fields(s) {
		if m := words1reWord.FindString(f); m != "" {
			out = append(out, strings.ToLower(m))
		}
	}
	return out
}

func words1initials(w []string, skipStop bool) string {
	var b strings.Builder
	for _, s := range w {
		if skipStop && words1expansionStop[s] {
			continue
		}
		b.WriteByte(s[0])
	}
	return b.String()
}

func words1isInitialism(abbrev string, w []string) bool {
	a := strings.ToLower(abbrev)
	if a == "" || len(w) < 2 {
		return false
	}
	return a == words1initials(w, false) || a == words1initials(w, true)
}

func words1upperCount(s string) int {
	n := 0
	for _, r := range s {
		if unicode.IsUpper(r) {
			n++
		}
	}
	return n
}

func words1tailWords(s string, n int) []string {
	if len(s) > 160 {
		s = s[len(s)-160:]
	}
	f := strings.Fields(s)
	if len(f) > n {
		f = f[len(f)-n:]
	}
	out := make([]string, 0, len(f))
	for _, w := range f {
		out = append(out, strings.ToLower(strings.Trim(w, words1punct)))
	}
	return out
}

func words1sorted(v []types.Violation) []types.Violation {
	sort.SliceStable(v, func(i, j int) bool { return v[i].StartIndex < v[j].StartIndex })
	return v
}

var words1reAbbrevFirst = regexp.MustCompile(`\b([A-Z][A-Za-z0-9]{1,5})[ \t]*\(([a-z][^)\n]{2,60})\)`)

// DetectGoogleAbbreviationBeforeExpansion reports an abbreviation introduced
// ahead of the term it stands for.
func DetectGoogleAbbreviationBeforeExpansion(text string) []types.Violation {
	var out []types.Violation
	for _, m := range words1reAbbrevFirst.FindAllStringSubmatchIndex(text, -1) {
		abbrev := text[m[2]:m[3]]
		expansion := text[m[4]:m[5]]
		if words1upperCount(abbrev) < 2 {
			continue
		}
		if !words1isInitialism(abbrev, words1words(expansion)) {
			continue
		}
		out = append(out, types.Violation{
			RuleID:          "abbreviation-before-expansion",
			StartIndex:      m[0],
			EndIndex:        m[1],
			MatchedText:     text[m[0]:m[1]],
			SuggestedChange: expansion + " (" + abbrev + ")",
		})
	}
	return out
}

var (
	words1reAcronymPlural = regexp.MustCompile(`\b[A-Z]{2,6}['\x{2019}]s\b`)
	words1reYearPlural    = regexp.MustCompile(`\b(?:19|20)\d{2}['\x{2019}]s\b`)
	words1reProductPlural = regexp.MustCompile(`\b(?:Google Cloud|Google|Android|Chrome|Kubernetes|` +
		`BigQuery|Gmail|Firebase|YouTube|Docker|GitHub|Slack|AWS|Azure|Terraform|Postgres|Linux)` +
		`(?:['\x{2019}]s|s)\b`)
	words1reNextWord = regexp.MustCompile(`^[ \t]*([A-Za-z]+)`)
)

var words1pluralAgreeing = map[string]bool{
	"are": true, "were": true, "have": true, "do": true, "include": true,
	"provide": true, "support": true, "require": true, "allow": true,
	"remain": true, "differ": true,
}

var words1apostropheStrip = strings.NewReplacer("'", "", "’", "")

func words1readsAsPlural(text string, end int) bool {
	rest := text[end:]
	if rest == "" {
		return true
	}
	switch rest[0] {
	case ',', ';', ':', '.', '!', '?', ')', ']', '\n', '\r':
		return true
	}
	m := words1reNextWord.FindStringSubmatch(rest)
	return m != nil && words1pluralAgreeing[strings.ToLower(m[1])]
}

// DetectGoogleApostrophePlural reports an apostrophe used to form a plural, and
// a product name bent into a plural or a possessive.
func DetectGoogleApostrophePlural(text string) []types.Violation {
	var out []types.Violation
	for _, v := range findAll(text, words1reAcronymPlural, "apostrophe-plural") {
		if !words1readsAsPlural(text, v.EndIndex) {
			continue
		}
		v.SuggestedChange = words1apostropheStrip.Replace(v.MatchedText)
		out = append(out, v)
	}
	for _, v := range findAll(text, words1reYearPlural, "apostrophe-plural") {
		v.SuggestedChange = words1apostropheStrip.Replace(v.MatchedText)
		out = append(out, v)
	}
	for _, v := range findAll(text, words1reProductPlural, "apostrophe-plural") {
		base := v.MatchedText
		if i := strings.LastIndexAny(base, "'’"); i >= 0 {
			base = base[:i]
		} else {
			base = strings.TrimSuffix(base, "s")
		}
		v.SuggestedChange = base
		out = append(out, v)
	}
	return words1sorted(out)
}

var words1reCompoundAgree = regexp.MustCompile(
	`(?i)\b([a-z][a-z-]{2,})[ \t]+and[ \t]+([a-z][a-z-]{2,})[ \t]+` +
		`(is|was|has|does|isn['\x{2019}]t|wasn['\x{2019}]t|hasn['\x{2019}]t|doesn['\x{2019}]t)\b`)

var words1compoundLead = map[string]bool{
	"between": true, "among": true, "amongst": true, "of": true, "from": true,
	"with": true, "for": true, "in": true, "on": true, "at": true, "by": true,
	"to": true, "into": true, "across": true, "over": true, "under": true,
	"about": true, "than": true, "and": true, "or": true, "both": true,
	"when": true, "where": true, "while": true, "if": true, "whether": true,
	"including": true, "versus": true, "vs": true,
}

var words1compoundScope = map[string]bool{
	"each": true, "every": true, "either": true, "neither": true, "no": true,
}

var words1compoundIdiom = map[string]bool{
	"trial and error":           true,
	"research and development":  true,
	"read and write":            true,
	"back and forth":            true,
	"cause and effect":          true,
	"black and white":           true,
	"supply and demand":         true,
	"bread and butter":          true,
	"give and take":             true,
	"peace and quiet":           true,
	"command and control":       true,
	"search and replace":        true,
	"terms and conditions":      true,
	"health and safety":         true,
	"drag and drop":             true,
	"copy and paste":            true,
	"business and technology":   true,
	"extract and transform":     true,
	"monitoring and alerting":   true,
	"backup and restore":        true,
	"authentication and access": true,
}

var words1compoundStop = map[string]bool{
	"this": true, "that": true, "these": true, "those": true, "one": true,
	"none": true, "some": true, "any": true, "all": true, "each": true,
	"every": true, "either": true, "neither": true, "which": true, "who": true,
	"what": true, "there": true, "here": true, "above": true, "below": true,
	"beyond": true, "back": true, "forth": true, "more": true, "less": true,
	"then": true, "again": true, "only": true, "still": true, "yet": true,
	"now": true, "not": true, "the": true, "and": true, "but": true,
	"such": true, "own": true, "same": true, "other": true, "another": true,
}

var words1pluralVerb = map[string]string{
	"is": "are", "was": "were", "has": "have", "does": "do",
	"isn't": "aren't", "wasn't": "weren't", "hasn't": "haven't", "doesn't": "don't",
}

// DetectGoogleCompoundSubjectAgreement reports two subjects joined by "and"
// taking a singular verb.
func DetectGoogleCompoundSubjectAgreement(text string) []types.Violation {
	var out []types.Violation
	for _, m := range words1reCompoundAgree.FindAllStringSubmatchIndex(text, -1) {
		a := strings.ToLower(text[m[2]:m[3]])
		b := strings.ToLower(text[m[4]:m[5]])
		if a == b || words1compoundStop[a] || words1compoundStop[b] {
			continue
		}
		if words1compoundIdiom[a+" and "+b] {
			continue
		}
		lead := words1tailWords(text[:m[0]], 3)
		if len(lead) > 0 && words1compoundLead[lead[len(lead)-1]] {
			continue
		}
		scoped := false
		for _, w := range lead {
			if words1compoundScope[w] {
				scoped = true
			}
		}
		if scoped {
			continue
		}
		verb := strings.ReplaceAll(strings.ToLower(text[m[6]:m[7]]), "’", "'")
		out = append(out, types.Violation{
			RuleID:          "compound-subject-agreement",
			StartIndex:      m[0],
			EndIndex:        m[1],
			MatchedText:     text[m[0]:m[1]],
			SuggestedChange: text[m[0]:m[6]] + words1pluralVerb[verb],
		})
	}
	return out
}

var words1docSelfSubs = []plainSub{
	{"this article", "this document"},
	{"this page", "this document"},
	{"this doc", "this document"},
	{"this topic", "this document"},
}

var words1docSelfRes = compileSubs(words1docSelfSubs)

// DetectGoogleDocSelfReference reports an ad hoc noun for the page itself.
func DetectGoogleDocSelfReference(text string) []types.Violation {
	return findSubs(text, "doc-self-reference", words1docSelfSubs, words1docSelfRes)
}

const words1actionVerbs = `(?i:add|configure|create|delete|deploy|disable|edit|enable|export|` +
	`import|install|open|remove|restart|run|set[ \t]+up|start|stop|update|upgrade|view|write)`

var words1articleContexts = []*regexp.Regexp{
	regexp.MustCompile(`(?m)^[ \t]{0,3}#{1,6}[ \t]+(` + words1actionVerbs + `\b[^\n]*)$`),
	regexp.MustCompile(`(?m)^[ \t]{0,3}(` + words1actionVerbs + `\b[^\n]*)\n[ \t]{0,3}(?:={2,}|-{2,})[ \t]*$`),
	regexp.MustCompile(`(?m)^title:[ \t]*["'\x{2019}]?(` + words1actionVerbs + `\b[^\n"']*)$`),
	regexp.MustCompile(`(?m)^[ \t]{0,6}(?:[-*+]|\d+[.)])[ \t]+(` + words1actionVerbs + `\b[^\n]*)$`),
}

var (
	words1reLeadVerb = regexp.MustCompile(`^` + words1actionVerbs + `\b`)
	words1reToken    = regexp.MustCompile(`\S+`)
)

var words1determiner = map[string]bool{
	"a": true, "an": true, "the": true, "your": true, "this": true,
	"that": true, "these": true, "those": true, "all": true, "any": true,
	"each": true, "both": true, "no": true, "multiple": true, "several": true,
	"two": true, "three": true, "my": true, "our": true, "its": true,
	"their": true, "one": true, "some": true, "every": true, "another": true,
	"per": true, "new": true,
}

var words1articleConnector = map[string]bool{
	"for": true, "with": true, "to": true, "in": true, "on": true, "at": true,
	"from": true, "by": true, "of": true, "and": true, "or": true,
	"using": true, "via": true, "when": true, "if": true, "as": true,
	"into": true, "before": true, "after": true, "that": true, "without": true,
	"over": true, "under": true, "between": true, "across": true,
	"through": true, "during": true, "based": true, "than": true,
}

var words1massNoun = map[string]bool{
	"access": true, "authentication": true, "authorization": true,
	"billing": true, "data": true, "information": true, "logging": true,
	"memory": true, "monitoring": true, "storage": true, "support": true,
	"traffic": true, "latency": true, "throughput": true, "networking": true,
	"security": true, "bandwidth": true, "capacity": true, "compliance": true,
	"connectivity": true, "content": true, "encryption": true,
	"decryption": true, "compression": true, "feedback": true,
	"governance": true, "hardware": true, "help": true, "infrastructure": true,
	"integration": true, "isolation": true, "output": true, "input": true,
	"performance": true, "redundancy": true, "reliability": true,
	"replication": true, "scalability": true, "software": true,
	"telemetry": true, "uptime": true, "downtime": true, "validation": true,
	"verification": true, "visibility": true, "availability": true,
	"observability": true, "concurrency": true, "middleware": true,
	"firmware": true, "mail": true, "email": true, "code": true,
}

// DetectGoogleDroppedArticle reports a heading, title, or list-item noun phrase
// that carries no article, determiner, or quantifier.
func DetectGoogleDroppedArticle(text string) []types.Violation {
	var out []types.Violation
	for _, re := range words1articleContexts {
		for _, m := range re.FindAllStringSubmatchIndex(text, -1) {
			if v, ok := words1articleHit(text, m[2], m[3]); ok {
				out = append(out, v)
			}
		}
	}
	out = words1sorted(out)
	var uniq []types.Violation
	for _, v := range out {
		if n := len(uniq); n > 0 && uniq[n-1].StartIndex == v.StartIndex {
			continue
		}
		uniq = append(uniq, v)
	}
	return uniq
}

func words1articleHit(text string, start, end int) (types.Violation, bool) {
	seg := strings.TrimRight(text[start:end], " \t\r")
	verb := words1reLeadVerb.FindString(seg)
	if verb == "" {
		return types.Violation{}, false
	}
	rest := seg[len(verb):]
	if strings.ContainsAny(rest, "`<") {
		return types.Violation{}, false
	}
	spans := words1reToken.FindAllStringIndex(rest, -1)
	n := 0
	for _, sp := range spans {
		raw := rest[sp[0]:sp[1]]
		if words1articleConnector[strings.ToLower(strings.Trim(raw, words1punct))] {
			break
		}
		n++
		if strings.ContainsAny(raw, ",:;") {
			break
		}
	}
	if n < 2 {
		return types.Violation{}, false
	}
	for _, sp := range spans[:n] {
		w := strings.ToLower(strings.Trim(rest[sp[0]:sp[1]], words1punct))
		if w == "" || words1determiner[w] || (w[0] >= '0' && w[0] <= '9') {
			return types.Violation{}, false
		}
	}
	last := strings.Trim(rest[spans[n-1][0]:spans[n-1][1]], words1punct)
	if last == "" || unicode.IsUpper(rune(last[0])) {
		return types.Violation{}, false
	}
	lw := strings.ToLower(last)
	if strings.HasSuffix(lw, "s") || strings.HasSuffix(lw, "ing") || words1massNoun[lw] {
		return types.Violation{}, false
	}
	base := start + len(verb)
	return types.Violation{
		RuleID:      "dropped-article",
		StartIndex:  base + spans[0][0],
		EndIndex:    base + spans[n-1][1],
		MatchedText: text[base+spans[0][0] : base+spans[n-1][1]],
	}, true
}

var (
	words1reParenAbbrev = regexp.MustCompile(`\(([A-Za-z]{2,6})\)`)
	words1reTrailWords  = regexp.MustCompile(`(?:[A-Za-z][A-Za-z-]*[ \t]+){1,6}$`)
	words1reWordSpan    = regexp.MustCompile(`[A-Za-z][A-Za-z-]*`)
)

func words1abbrevNumber(abbrev string, w []string) (plural, ok bool) {
	if words1isInitialism(abbrev, w) {
		return false, true
	}
	l := strings.ToLower(abbrev)
	if len(l) > 2 && strings.HasSuffix(l, "s") && words1isInitialism(l[:len(l)-1], w) {
		return true, true
	}
	return false, false
}

func words1isPluralNoun(w string) bool {
	return len(w) > 3 && strings.HasSuffix(w, "s") &&
		!strings.HasSuffix(w, "ss") && !strings.HasSuffix(w, "us") &&
		!strings.HasSuffix(w, "is") && !strings.HasSuffix(w, "as")
}

func words1dropPluralS(s string) string {
	if n := len(s); n > 0 && (s[n-1] == 's' || s[n-1] == 'S') {
		return s[:n-1]
	}
	return s
}

// DetectGoogleExpansionNumberMismatch reports a spelled-out term and its
// parenthesized abbreviation disagreeing in number.
func DetectGoogleExpansionNumberMismatch(text string) []types.Violation {
	var out []types.Violation
	for _, m := range words1reParenAbbrev.FindAllStringSubmatchIndex(text, -1) {
		abbrev := text[m[2]:m[3]]
		lead := words1reTrailWords.FindStringIndex(text[:m[0]])
		if lead == nil {
			continue
		}
		spans := words1reWordSpan.FindAllStringIndex(text[lead[0]:lead[1]], -1)
		words := make([]string, 0, len(spans))
		for _, sp := range spans {
			words = append(words, strings.ToLower(text[lead[0]+sp[0]:lead[0]+sp[1]]))
		}
		for k := 2; k <= len(words); k++ {
			cand := words[len(words)-k:]
			plural, ok := words1abbrevNumber(abbrev, cand)
			if !ok {
				continue
			}
			if plural != words1isPluralNoun(cand[len(cand)-1]) {
				start := lead[0] + spans[len(spans)-k][0]
				expansion := strings.TrimRight(text[start:m[0]], " \t")
				fixed := expansion + " (" + abbrev + "s)"
				if plural {
					fixed = expansion + " (" + words1dropPluralS(abbrev) + ")"
				}
				out = append(out, types.Violation{
					RuleID:          "expansion-number-mismatch",
					StartIndex:      start,
					EndIndex:        m[1],
					MatchedText:     text[start:m[1]],
					SuggestedChange: fixed,
				})
			}
			break
		}
	}
	return out
}

var (
	words1reArticleA  = regexp.MustCompile(`\b([Aa])[ \t]+([A-Z][A-Z0-9]{1,5})\b`)
	words1reArticleAn = regexp.MustCompile(`\b([Aa]n)[ \t]+([A-Z][A-Z0-9]{1,5})\b`)
)

// words1vowelSound lists letter-by-letter initialisms whose first letter is
// said with a leading vowel sound. It is a pronunciation table, not a rule:
// extend it as new initialisms appear.
var words1vowelSound = map[string]bool{
	"ABI": true, "ACL": true, "AES": true, "AMI": true, "AMQP": true,
	"API": true, "APK": true, "ARM": true, "ARN": true, "ASN": true,
	"AST": true, "ATM": true, "AWS": true,
	"EBS": true, "EC2": true, "ECS": true, "EFS": true, "EKS": true,
	"EOF": true, "EOL": true, "ESB": true, "ETL": true,
	"FPGA": true, "FQDN": true, "FTP": true, "FTPS": true,
	"HA": true, "HDD": true, "HDFS": true, "HMAC": true, "HSM": true,
	"HTML": true, "HTTP": true, "HTTPS": true,
	"IAM": true, "ICMP": true, "ID": true, "IDE": true, "IDP": true,
	"IMAP": true, "IOPS": true, "IP": true, "IPC": true, "IRC": true,
	"ISO": true, "ISP": true,
	"LB": true, "LDAP": true, "LLM": true, "LRU": true, "LTE": true,
	"LTS": true, "LXC": true,
	"MB": true, "MD5": true, "MFA": true, "ML": true, "MQTT": true,
	"MTU": true, "MX": true,
	"NDA": true, "NFS": true, "NPM": true, "NS": true,
	"ODBC": true, "OIDC": true, "OOM": true, "ORM": true, "OS": true,
	"OTP":  true,
	"RBAC": true, "RDBMS": true, "RDP": true, "RDS": true, "RFC": true,
	"RPC": true, "RSA": true, "RSS": true, "RTC": true, "RTP": true,
	"RTT": true,
	"S3":  true, "SAP": true, "SDK": true, "SFTP": true, "SHA": true,
	"SLA": true, "SLI": true, "SLO": true, "SMS": true, "SMTP": true,
	"SNMP": true, "SNS": true, "SOA": true, "SPF": true, "SQS": true,
	"SRE": true, "SRV": true, "SSD": true, "SSE": true, "SSH": true,
	"SSL": true, "SSO": true, "STS": true,
	"XHR": true, "XML": true, "XSD": true, "XSLT": true, "XSS": true,
}

// words1saidAsWord lists abbreviations pronounced as words that open with a
// consonant sound, so they take "a" even when their first letter would not.
var words1saidAsWord = map[string]bool{
	"SQL": true, "SAML": true, "SIM": true, "SOAP": true, "SAAS": true,
	"NAT": true, "LAN": true, "MAN": true, "RAID": true, "FHIR": true,
	"NASA": true, "NATO": true, "MIME": true, "LIFO": true, "FIFO": true,
	"REST": true, "RAM": true, "ROM": true, "MAC": true, "SAN": true,
	"SATA": true, "SPA": true, "FUSE": true, "SCSI": true, "SIEM": true,
	"NAS": true, "NIC": true, "LAMP": true, "MIPS": true, "ROI": true,
}

const words1consonantLetters = "BCDGJKPQTUVWYZ"

// DetectGoogleIndefiniteArticleAbbreviation reports "a" or "an" chosen by the
// abbreviation's first letter rather than by how it is said aloud.
func DetectGoogleIndefiniteArticleAbbreviation(text string) []types.Violation {
	var out []types.Violation
	for _, m := range words1reArticleA.FindAllStringSubmatchIndex(text, -1) {
		if !words1vowelSound[text[m[4]:m[5]]] {
			continue
		}
		article := "an"
		if text[m[2]] == 'A' {
			article = "An"
		}
		out = append(out, types.Violation{
			RuleID:          "indefinite-article-abbreviation",
			StartIndex:      m[0],
			EndIndex:        m[1],
			MatchedText:     text[m[0]:m[1]],
			SuggestedChange: article + text[m[3]:m[5]],
		})
	}
	for _, m := range words1reArticleAn.FindAllStringSubmatchIndex(text, -1) {
		abbrev := text[m[4]:m[5]]
		if !strings.ContainsRune(words1consonantLetters, rune(abbrev[0])) && !words1saidAsWord[abbrev] {
			continue
		}
		if words1vowelSound[abbrev] {
			continue
		}
		article := "a"
		if text[m[2]] == 'A' {
			article = "A"
		}
		out = append(out, types.Violation{
			RuleID:          "indefinite-article-abbreviation",
			StartIndex:      m[0],
			EndIndex:        m[1],
			MatchedText:     text[m[0]:m[1]],
			SuggestedChange: article + text[m[3]:m[5]],
		})
	}
	return words1sorted(out)
}

var (
	words1reLatin       = regexp.MustCompile(`(?i)\b(i\.e\.|e\.g\.|i\.e|e\.g|viz\.|cf\.|et[ \t]+al\.)(?:[,;:)\]\s]|$)`)
	words1reForInstance = regexp.MustCompile(`(?i)\bfor[ \t]+instance\b`)
	words1reForExample  = regexp.MustCompile(`(?i)\b(for[ \t]+example)[ \t]+[a-z]`)
)

var words1latinSuggest = map[string]string{
	"i.e.":   "that is",
	"i.e":    "that is",
	"e.g.":   "for example",
	"e.g":    "for example",
	"viz.":   "namely",
	"cf.":    "compare",
	"et al.": "and others",
}

// words1instanceNoun holds the words after which "for instance" is the noun
// "instance" doing ordinary work rather than a Latin-flavored lead-in.
var words1instanceNoun = map[string]bool{
	"type": true, "types": true, "group": true, "groups": true,
	"name": true, "names": true, "id": true, "ids": true, "size": true,
	"sizes": true, "count": true, "counts": true, "template": true,
	"templates": true, "metadata": true, "variables": true, "state": true,
	"identifier": true, "identifiers": true, "creation": true,
}

func words1isLeadIn(text string, start int) bool {
	for i := start - 1; i >= 0; i-- {
		switch text[i] {
		case ' ', '\t':
			continue
		case '.', ',', ';', ':', '(', '\n', '\r', '-':
			return true
		default:
			return false
		}
	}
	return true
}

func words1matchCase(src, replacement string) string {
	if src == "" || replacement == "" || !unicode.IsUpper(rune(src[0])) {
		return replacement
	}
	return strings.ToUpper(replacement[:1]) + replacement[1:]
}

// DetectGoogleLatinAbbreviation reports Latin abbreviations and the "for
// instance" variant of "for example".
func DetectGoogleLatinAbbreviation(text string) []types.Violation {
	var out []types.Violation
	for _, m := range words1reLatin.FindAllStringSubmatchIndex(text, -1) {
		matched := text[m[2]:m[3]]
		key := strings.ToLower(strings.Join(strings.Fields(matched), " "))
		out = append(out, types.Violation{
			RuleID:          "latin-abbreviation",
			StartIndex:      m[2],
			EndIndex:        m[3],
			MatchedText:     matched,
			SuggestedChange: words1matchCase(matched, words1latinSuggest[key]),
		})
	}
	for _, v := range findAll(text, words1reForInstance, "latin-abbreviation") {
		if n := words1reNextWord.FindStringSubmatch(text[v.EndIndex:]); n != nil &&
			words1instanceNoun[strings.ToLower(n[1])] {
			continue
		}
		v.SuggestedChange = words1matchCase(v.MatchedText, "for example")
		out = append(out, v)
	}
	for _, m := range words1reForExample.FindAllStringSubmatchIndex(text, -1) {
		if !words1isLeadIn(text, m[2]) {
			continue
		}
		out = append(out, types.Violation{
			RuleID:          "latin-abbreviation",
			StartIndex:      m[2],
			EndIndex:        m[3],
			MatchedText:     text[m[2]:m[3]],
			SuggestedChange: text[m[2]:m[3]] + ",",
		})
	}
	return words1sorted(out)
}

var (
	words1reNeitherOr = regexp.MustCompile(`(?i)\bneither\b[^.!?\n]{1,60}\bor\b`)
	words1reNor       = regexp.MustCompile(`(?i)\bnor\b`)
)

func words1norCase(or string) string {
	switch or {
	case "OR":
		return "NOR"
	case "Or":
		return "Nor"
	}
	return "nor"
}

// DetectGoogleNeitherNorPairing reports "neither" paired with "or".
func DetectGoogleNeitherNorPairing(text string) []types.Violation {
	var out []types.Violation
	for _, v := range findAll(text, words1reNeitherOr, "neither-nor-pairing") {
		if words1reNor.MatchString(v.MatchedText) || strings.ContainsAny(v.MatchedText, ",;") {
			continue
		}
		n := len(v.MatchedText)
		v.SuggestedChange = v.MatchedText[:n-2] + words1norCase(v.MatchedText[n-2:])
		out = append(out, v)
	}
	return out
}
