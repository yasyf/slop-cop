package detectors

import (
	"regexp"
	"strings"

	"github.com/yasyf/slop-cop/internal/types"
)

const productBrandObject = `(?:it|me|us|them|him|her|you|the|this|that|these|those|a|an|your|our|my|their)`

var productBrandVerbPats = []struct {
	re *regexp.Regexp
	to string
}{
	{regexp.MustCompile(`(?i)\bgoogled\b`), "searched for"},
	{regexp.MustCompile(`(?i)\bgoogling\b`), "searching for"},
	{regexp.MustCompile(`(?i)\bgoogles\b`), "searches for"},
	{regexp.MustCompile(`(?i)\bphotoshopped\b`), "edited in Photoshop"},
	{regexp.MustCompile(`(?i)\bphotoshopping\b`), "editing in Photoshop"},
	{regexp.MustCompile(`(?i)\bphotoshops\b`), "edits in Photoshop"},
	{regexp.MustCompile(`(?i)\bxeroxed\b`), "photocopied"},
	{regexp.MustCompile(`(?i)\bxeroxing\b`), "photocopying"},
	{regexp.MustCompile(`(?i)\bxeroxes\b`), "photocopies"},
	{regexp.MustCompile(`(?i)\bvenmoed\b`), "paid with Venmo"},
	{regexp.MustCompile(`(?i)\bvenmoing\b`), "paying with Venmo"},
	{regexp.MustCompile(`(?i)\bskyped\b`), "called on Skype"},
	{regexp.MustCompile(`(?i)\bskyping\b`), "calling on Skype"},
	{regexp.MustCompile(`(?i)\bfacetimed\b`), "called on FaceTime"},
	{regexp.MustCompile(`(?i)\bfacetiming\b`), "calling on FaceTime"},
	{regexp.MustCompile(`(?i)\bdockerize\b`), "package as a Docker image"},
	{regexp.MustCompile(`(?i)\bdockerizes\b`), "packages as a Docker image"},
	{regexp.MustCompile(`(?i)\bdockerized\b`), "packaged as a Docker image"},
	{regexp.MustCompile(`(?i)\bdockerizing\b`), "packaging as a Docker image"},
}

var (
	productReBrandSlackObject = regexp.MustCompile(`(?i)\bslack(?:ed|ing)\s+(?:me|us|them|him|her|you)\b`)
	productReBrandObject      = regexp.MustCompile(`\b(?:(?i:google|photoshop|venmo|facetime|chromecast)|Slack)\s+` + productBrandObject + `\b`)
	productReBrandZoomObject  = regexp.MustCompile(`\b(?:to|can|could|should|must|will|would|please|just)\s+Zoom\s+(?:it|me|us|them|him|her|you)\b`)
)

const productBrandVerbExplanation = "Name the action with a verb and keep the brand name as a noun."

// DetectGoogleBrandNameAsVerb reports a brand name used as a verb.
func DetectGoogleBrandNameAsVerb(text string) []types.Violation {
	var out []types.Violation
	for _, p := range productBrandVerbPats {
		for _, v := range findAll(text, p.re, "brand-name-as-verb") {
			v.SuggestedChange = p.to
			out = append(out, v)
		}
	}
	for _, re := range []*regexp.Regexp{productReBrandSlackObject, productReBrandObject, productReBrandZoomObject} {
		for _, v := range findAll(text, re, "brand-name-as-verb") {
			v.Explanation = productBrandVerbExplanation
			out = append(out, v)
		}
	}
	return out
}

var productCasingSubs = []plainSub{
	{"Github", "GitHub"},
	{"GitHUb", "GitHub"},
	{"Javascript", "JavaScript"},
	{"JavaScipt", "JavaScript"},
	{"Typescript", "TypeScript"},
	{"NodeJS", "Node.js"},
	{"NodeJs", "Node.js"},
	{"Node.JS", "Node.js"},
	{"Bigquery", "BigQuery"},
	{"BigTable", "Bigtable"},
	{"MacOS", "macOS"},
	{"Mac OS X", "macOS"},
	{"Iphone", "iPhone"},
	{"IPhone", "iPhone"},
	{"Ipad", "iPad"},
	{"PubSub", "Pub/Sub"},
	{"Pubsub", "Pub/Sub"},
	{"GMail", "Gmail"},
	{"Youtube", "YouTube"},
	{"WiFi", "Wi-Fi"},
	{"Wifi", "Wi-Fi"},
	{"Chrome OS", "ChromeOS"},
	{"FireStore", "Firestore"},
	{"NPM", "npm"},
	{"Ebay", "eBay"},
	{"EBay", "eBay"},
	{"Gitlab", "GitLab"},
	{"Stackoverflow", "Stack Overflow"},
	{"Linkedin", "LinkedIn"},
	{"Powershell", "PowerShell"},
	{"Sqlite", "SQLite"},
	{"Mysql", "MySQL"},
	{"MongoDb", "MongoDB"},
	{"Oauth", "OAuth"},
	{"OAUTH", "OAuth"},
	{"Postgresql", "PostgreSQL"},
}

var productAbbrevSubs = []plainSub{
	{"GCP", "Google Cloud"},
	{"GKE", "Google Kubernetes Engine"},
	{"GCS", "Cloud Storage"},
	{"BQ", "BigQuery"},
	{"K8s", "Kubernetes"},
	{"k8s", "Kubernetes"},
	{"VSCode", "Visual Studio Code"},
	{"VS Code", "Visual Studio Code"},
	{"Postgres", "PostgreSQL"},
	{"Mongo", "MongoDB"},
}

var (
	productCasingRes = productCompileExact(productCasingSubs)
	productAbbrevRes = productCompileExact(productAbbrevSubs)
)

func productCompileExact(subs []plainSub) []*regexp.Regexp {
	out := make([]*regexp.Regexp, 0, len(subs))
	for _, s := range subs {
		out = append(out, regexp.MustCompile(`\b`+
			strings.ReplaceAll(escapeForRegex(s.from), " ", phraseSep)+`\b`))
	}
	return out
}

func productDropShouted(text string, in []types.Violation) []types.Violation {
	kept := in[:0]
	for _, v := range in {
		if productShouted(text, v.StartIndex, v.EndIndex) {
			continue
		}
		kept = append(kept, v)
	}
	return kept
}

func productShouted(text string, start, end int) bool {
	match := text[start:end]
	if match != strings.ToUpper(match) {
		return false
	}
	lines := spanLines(text, start, end)
	if len(lines) != 1 {
		return false
	}
	words := 0
	for _, w := range strings.Fields(lines[0]) {
		w = strings.Trim(w, `.,:;!?()[]"'`)
		if len(w) < 2 || !productIsAlpha(w) {
			continue
		}
		words++
		if w != strings.ToUpper(w) {
			return false
		}
	}
	return words >= 2
}

func productIsAlpha(s string) bool {
	for _, r := range s {
		if (r < 'a' || r > 'z') && (r < 'A' || r > 'Z') {
			return false
		}
	}
	return true
}

// DetectGoogleBrandNameCasing reports a brand name written with the wrong capitalization.
func DetectGoogleBrandNameCasing(text string) []types.Violation {
	return productDropShouted(text, findSubs(text, "brand-name-casing", productCasingSubs, productCasingRes))
}

var productReScriptAbbrev = regexp.MustCompile(`\b(?:JS|TS)\b`)

// DetectGoogleProductNameAbbreviation reports a product name shortened to an
// abbreviation the guide spells out.
func DetectGoogleProductNameAbbreviation(text string) []types.Violation {
	out := findSubs(text, "product-name-abbreviation", productAbbrevSubs, productAbbrevRes)
	for _, idx := range productReScriptAbbrev.FindAllStringIndex(text, -1) {
		if idx[0] > 0 && (text[idx[0]-1] == '.' || text[idx[0]-1] == '/') {
			continue
		}
		to := "JavaScript"
		if text[idx[0]:idx[1]] == "TS" {
			to = "TypeScript"
		}
		out = append(out, types.Violation{
			RuleID:          "product-name-abbreviation",
			StartIndex:      idx[0],
			EndIndex:        idx[1],
			MatchedText:     text[idx[0]:idx[1]],
			SuggestedChange: to,
		})
	}
	return productDropShouted(text, out)
}

var productReTrailingExample = regexp.MustCompile(`,\s+((?i:for example|for instance)),\s+([^.;:!?]{1,60})[.!?](?:\s|$)`)

var productFiniteVerbs = map[string]struct{}{
	"am": {}, "is": {}, "are": {}, "was": {}, "were": {}, "be": {}, "been": {},
	"has": {}, "have": {}, "had": {}, "do": {}, "does": {}, "did": {},
	"can": {}, "could": {}, "will": {}, "would": {}, "shall": {}, "should": {},
	"may": {}, "might": {}, "must": {},
	"return": {}, "returns": {}, "set": {}, "sets": {}, "show": {}, "shows": {},
	"use": {}, "uses": {}, "contain": {}, "contains": {}, "include": {}, "includes": {},
	"mean": {}, "means": {}, "require": {}, "requires": {}, "need": {}, "needs": {},
	"appear": {}, "appears": {}, "work": {}, "works": {}, "run": {}, "runs": {},
	"get": {}, "gets": {}, "become": {}, "becomes": {}, "remain": {}, "remains": {},
	"add": {}, "adds": {}, "apply": {}, "applies": {}, "let": {}, "lets": {},
}

var productReInflectedLead = regexp.MustCompile(`^[a-z]+(?:s|ed|ing)$`)

func productClauseLike(fields []string) bool {
	for i, f := range fields {
		w := strings.ToLower(strings.Trim(f, `,"'()[]`))
		if _, ok := productFiniteVerbs[w]; ok {
			return true
		}
		if i == 0 && productReInflectedLead.MatchString(f) {
			return true
		}
	}
	return false
}

// DetectGoogleTrailingForExampleComma reports a trailing "for example" clause
// left dangling after a comma.
func DetectGoogleTrailingForExampleComma(text string) []types.Violation {
	var out []types.Violation
	for _, m := range productReTrailingExample.FindAllStringSubmatchIndex(text, -1) {
		start, end := trimSpan(text, m[4], m[5])
		example := text[start:end]
		fields := strings.Fields(example)
		if len(fields) == 0 || len(fields) > 6 || strings.Contains(example, ",") || productClauseLike(fields) {
			continue
		}
		out = append(out, types.Violation{
			RuleID:          "trailing-for-example-comma",
			StartIndex:      m[0],
			EndIndex:        end,
			MatchedText:     text[m[0]:end],
			Explanation:     "Parenthesize a short trailing example, or introduce it with \"such as\".",
			SuggestedChange: " (" + text[m[2]:m[3]] + ", " + example + ")",
		})
	}
	return out
}
