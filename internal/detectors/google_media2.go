package detectors

import (
	"regexp"
	"strings"

	"github.com/yasyf/slop-cop/internal/types"
)

const media2fileExt = `(?:ya?ml|json|go|py|js|ts|tsx|md|txt|sh|toml|xml|csv|proto|tf|conf|ini|sql)`

var (
	media2reFileToken = regexp.MustCompile(`\b[\w.\-/]+\.` + media2fileExt + `\b`)
	media2reFileWhole = regexp.MustCompile(`^[\w.\-/]+\.` + media2fileExt + `$`)
	media2reFlagToken = regexp.MustCompile(`--[a-z][\w-]+`)
	media2reFlagWhole = regexp.MustCompile(`^--[a-z][\w-]+$`)
	media2reCodeSpan  = regexp.MustCompile("`[^`\n]+`")
	media2reLink      = regexp.MustCompile(`\[[^\]\n]*\]\([^)\n]*\)`)
	media2reURL       = regexp.MustCompile(`(?i)\b(?:https?|ftp)://\S+`)
	media2reFence     = regexp.MustCompile("(?s)```.*?```")
	media2rePascal    = regexp.MustCompile(`^[A-Z][a-z0-9]+[A-Za-z0-9]*$`)
	media2reCamel     = regexp.MustCompile(`^[a-z][a-z0-9]*[A-Z][A-Za-z0-9]*$`)
	media2reIdentPath = regexp.MustCompile(`^[A-Za-z0-9_.\-/]+$`)
)

func media2set(words ...string) map[string]bool {
	m := make(map[string]bool, len(words))
	for _, w := range words {
		m[w] = true
	}
	return m
}

var media2categoryNouns = media2set(
	"file", "directory", "folder", "config", "configuration", "manifest", "script",
	"template", "document", "documentation", "class", "method", "function", "field",
	"parameter", "argument", "flag", "option", "command", "subcommand", "package",
	"module", "library", "framework", "table", "column", "row", "resource", "object",
	"service", "api", "endpoint", "header", "variable", "property", "attribute",
	"value", "string", "block", "section", "page", "repository", "repo", "image",
	"container", "job", "policy", "role", "bucket", "dataset", "binary", "archive",
	"extension", "button", "menu", "tab", "checkbox", "dialog", "window", "pane",
	"panel", "icon", "link", "box", "list", "item", "entry", "key", "secret", "token",
	"account", "project", "environment", "region", "zone", "instance", "volume",
	"network", "port", "protocol", "prefix", "suffix", "syntax", "format", "spec",
	"specification", "reference", "runtime", "server", "client", "cluster", "platform",
	"workflow", "action", "event", "hook", "rule", "query", "view", "index", "schema",
	"model", "entity", "record", "namespace", "annotation", "element", "tag", "node",
	"type", "interface", "struct", "enum", "constant", "tool", "utility", "wrapper",
	"helper", "plugin", "snippet", "sample", "example", "setting", "mode", "target",
	"task", "step", "label", "selector", "statement", "expression", "literal",
	"keyword", "version", "release", "daemon", "process", "operator",
	"controller", "chart", "suite", "test", "check", "pipeline", "stage", "response",
	"request", "payload", "message", "topic", "queue", "stream", "log", "report",
	"review", "comment", "excerpt", "run", "verdict", "note", "doc", "guide",
	"tutorial", "output", "input", "result", "state", "layer", "tier", "set",
	"group", "count", "limit", "budget", "path", "url", "uri", "id", "name",
	"number", "size", "status", "error", "warning", "code", "text", "prose",
	"style", "pattern", "regex", "array", "map", "slice", "pointer", "lane",
	"agent", "session", "worker", "cache", "branch", "commit", "remote", "gate",
	"phase", "call", "line", "form", "shape", "case", "part", "pair", "unit",
	"shim", "descriptor", "handler", "adapter", "driver", "middleware", "fragment",
	"artifact", "asset", "bundle", "snapshot", "trace", "metric", "alias", "macro",
)

var media2determiners = media2set("the", "a", "an", "your", "this", "that", "its", "each", "every", "our", "their")

var media2prepositions = media2set("in", "from", "into", "within")

var media2skipWords = media2set("named", "called", "is", "are", "was", "were", "be", "become", "becomes")

var media2literalValues = media2set("true", "false", "null", "nil", "none", "yes", "no", "on", "off", "undefined", "nan")

var media2knownProducts = media2set(
	"node.js", "next.js", "nuxt.js", "vue.js", "three.js", "d3.js", "express.js",
	"react.js", "ember.js", "backbone.js", "chart.js", "jquery.js", "socket.js",
)

type media2span struct{ start, end int }

// DetectGoogleUnqualifiedCodeElement reports a filename, flag, or identifier
// used as a bare noun with no category noun after it.
func DetectGoogleUnqualifiedCodeElement(text string) []types.Violation {
	skips := media2skipRanges(text)
	codeSpans := media2reCodeSpan.FindAllStringIndex(text, -1)
	var out []types.Violation
	for _, m := range codeSpans {
		if media2covered(skips, m[0]) {
			continue
		}
		inner := text[m[0]+1 : m[1]-1]
		if !media2isCodeElement(inner) {
			continue
		}
		if media2unqualified(text, m[0], m[1], true, media2strongShape(inner)) {
			out = append(out, media2violation(text, m[0], m[1]))
		}
	}
	for _, re := range []*regexp.Regexp{media2reFileToken, media2reFlagToken} {
		for _, m := range re.FindAllStringIndex(text, -1) {
			if media2covered(skips, m[0]) || media2inPairs(codeSpans, m[0]) {
				continue
			}
			if media2knownProducts[strings.ToLower(text[m[0]:m[1]])] || media2pathAttached(text, m[0], m[1]) {
				continue
			}
			if media2unqualified(text, m[0], m[1], false, true) {
				out = append(out, media2violation(text, m[0], m[1]))
			}
		}
	}
	return out
}

// media2pathAttached reports whether [start, end) is a fragment of a longer
// path or module name, where the file-token regex stopped at the first
// extension it found: "go.yaml" inside "go.yaml.in/yaml/v4".
func media2pathAttached(text string, start, end int) bool {
	if start > 0 {
		switch text[start-1] {
		case '.', '/', '-', '_':
			return true
		}
	}
	if end >= len(text) {
		return false
	}
	switch text[end] {
	case '/', '-', '_':
		return true
	case '.':
		return end+1 < len(text) && media2isWordByte(text[end+1])
	}
	return false
}

func media2violation(text string, start, end int) types.Violation {
	return types.Violation{
		RuleID:      "unqualified-code-element",
		StartIndex:  start,
		EndIndex:    end,
		MatchedText: text[start:end],
		Explanation: "No category noun follows this code element",
	}
}

func media2skipRanges(text string) []media2span {
	var out []media2span
	for _, re := range []*regexp.Regexp{media2reFence, media2reLink, media2reURL} {
		for _, m := range re.FindAllStringIndex(text, -1) {
			out = append(out, media2span{start: m[0], end: m[1]})
		}
	}
	return out
}

func media2covered(spans []media2span, pos int) bool {
	for _, s := range spans {
		if pos >= s.start && pos < s.end {
			return true
		}
	}
	return false
}

func media2inPairs(pairs [][]int, pos int) bool {
	for _, p := range pairs {
		if pos >= p[0] && pos < p[1] {
			return true
		}
	}
	return false
}

func media2isCodeElement(inner string) bool {
	s := strings.TrimSuffix(strings.TrimSpace(inner), "()")
	if len(s) < 2 || strings.ContainsAny(s, " \t\n") {
		return false
	}
	if media2literalValues[strings.ToLower(s)] || media2knownProducts[strings.ToLower(s)] {
		return false
	}
	if media2strongShape(s) {
		return true
	}
	if media2rePascal.MatchString(s) || media2reCamel.MatchString(s) {
		return true
	}
	return strings.ContainsAny(s, "_/") && media2reIdentPath.MatchString(s)
}

func media2strongShape(inner string) bool {
	s := strings.TrimSpace(inner)
	return media2reFileWhole.MatchString(s) || media2reFlagWhole.MatchString(s)
}

// media2unqualified applies the triggers the span's form earns. A determiner
// in front is the one trigger a bare token in running text gets; a marked-up
// code span also takes a sentence start, and a filename or flag inside one
// takes a preposition.
func media2unqualified(text string, start, end int, marked, strong bool) bool {
	if start > 0 && (text[start-1] == '-' || text[start-1] == '/') {
		return false
	}
	prev := media2precedingWord(text, start)
	if media2skipWords[prev] {
		return false
	}
	trigger := media2determiners[prev]
	if marked && !trigger {
		trigger = media2sentenceStart(text, start) || (strong && media2prepositions[prev])
	}
	if !trigger {
		return false
	}
	next, ok := media2followingWord(text, end)
	if !ok {
		return false
	}
	return !media2categoryNoun(next)
}

func media2categoryNoun(w string) bool {
	if w == "" {
		return false
	}
	if media2categoryNouns[w] {
		return true
	}
	if strings.HasSuffix(w, "es") && media2categoryNouns[strings.TrimSuffix(w, "es")] {
		return true
	}
	return strings.HasSuffix(w, "s") && media2categoryNouns[strings.TrimSuffix(w, "s")]
}

func media2precedingWord(text string, start int) string {
	i := start
	newlines := 0
	for i > 0 {
		c := text[i-1]
		if c == ' ' || c == '\t' || c == '\r' {
			i--
			continue
		}
		if c == '\n' {
			if newlines == 1 {
				return ""
			}
			newlines++
			i--
			continue
		}
		break
	}
	j := i
	for j > 0 && media2isLetter(text[j-1]) {
		j--
	}
	return strings.ToLower(text[j:i])
}

// media2followingWord returns the next prose word after end, lowercased. The
// bool is false where the code element is not a bare noun at all: a reference
// entry ("`--flag`: does x"), an assignment, or an attributive compound
// ("a `--budget`-capped excerpt").
func media2followingWord(text string, end int) (string, bool) {
	if end < len(text) && (text[end] == '-' || text[end] == '/') {
		return "", false
	}
	i := end
	newlines := 0
	for i < len(text) {
		c := text[i]
		if c == ' ' || c == '\t' || c == '\r' {
			i++
			continue
		}
		if c == '\n' {
			if newlines == 1 {
				return "", true
			}
			newlines++
			i++
			continue
		}
		break
	}
	if i >= len(text) {
		return "", true
	}
	if text[i] == ':' || text[i] == '=' {
		return "", false
	}
	if strings.HasPrefix(text[i:], "'s") || strings.HasPrefix(text[i:], "’s") {
		return "", false
	}
	j := i
	for j < len(text) && media2isLetter(text[j]) {
		j++
	}
	if j > i && j+1 < len(text) && text[j] == '-' && media2isLetter(text[j+1]) {
		return "", false
	}
	return strings.ToLower(text[i:j]), true
}

// media2sentenceStart deliberately ignores line starts. A code element
// opening a line is usually a reference entry or a list term, not a bare noun
// in a sentence, so only the start of the text and a terminator earlier on the
// same line count.
func media2sentenceStart(text string, start int) bool {
	i := start
	for i > 0 && (text[i-1] == ' ' || text[i-1] == '\t') {
		i--
	}
	if i == 0 {
		return true
	}
	c := text[i-1]
	if c == '!' || c == '?' {
		return true
	}
	return c == '.' && !media2listMarker(text, i-1)
}

func media2listMarker(text string, dot int) bool {
	i := dot
	for i > 0 && text[i-1] >= '0' && text[i-1] <= '9' {
		i--
	}
	if i == dot {
		return false
	}
	for i > 0 && (text[i-1] == ' ' || text[i-1] == '\t') {
		i--
	}
	return i == 0 || text[i-1] == '\n'
}

func media2isLetter(b byte) bool {
	return (b >= 'a' && b <= 'z') || (b >= 'A' && b <= 'Z')
}

func media2isWordByte(b byte) bool {
	return media2isLetter(b) || (b >= '0' && b <= '9') || b == '_'
}

var media2spellingSubs = []plainSub{
	{from: "colour", to: "color"},
	{from: "colours", to: "colors"},
	{from: "coloured", to: "colored"},
	{from: "behaviour", to: "behavior"},
	{from: "behaviours", to: "behaviors"},
	{from: "organise", to: "organize"},
	{from: "organised", to: "organized"},
	{from: "organising", to: "organizing"},
	{from: "organisation", to: "organization"},
	{from: "organisations", to: "organizations"},
	{from: "initialise", to: "initialize"},
	{from: "initialised", to: "initialized"},
	{from: "initialisation", to: "initialization"},
	{from: "customise", to: "customize"},
	{from: "customised", to: "customized"},
	{from: "optimise", to: "optimize"},
	{from: "optimised", to: "optimized"},
	{from: "optimisation", to: "optimization"},
	{from: "authorise", to: "authorize"},
	{from: "authorisation", to: "authorization"},
	{from: "serialise", to: "serialize"},
	{from: "serialisation", to: "serialization"},
	{from: "normalise", to: "normalize"},
	{from: "synchronise", to: "synchronize"},
	{from: "prioritise", to: "prioritize"},
	{from: "recognise", to: "recognize"},
	{from: "maximise", to: "maximize"},
	{from: "minimise", to: "minimize"},
	{from: "analyse", to: "analyze"},
	{from: "analysed", to: "analyzed"},
	{from: "analysing", to: "analyzing"},
	{from: "licence", to: "license"},
	{from: "licences", to: "licenses"},
	{from: "defence", to: "defense"},
	{from: "centre", to: "center"},
	{from: "centres", to: "centers"},
	{from: "catalogue", to: "catalog"},
	{from: "catalogues", to: "catalogs"},
	{from: "grey", to: "gray"},
	{from: "programme", to: "program"},
	{from: "programmes", to: "programs"},
	{from: "artefact", to: "artifact"},
	{from: "artefacts", to: "artifacts"},
	{from: "whilst", to: "while"},
	{from: "amongst", to: "among"},
	{from: "cancelled", to: "canceled"},
	{from: "cancelling", to: "canceling"},
	{from: "labelled", to: "labeled"},
	{from: "labelling", to: "labeling"},
	{from: "modelling", to: "modeling"},
	{from: "travelling", to: "traveling"},
	{from: "fulfil", to: "fulfill"},
	{from: "enrol", to: "enroll"},
	{from: "enrolment", to: "enrollment"},
	{from: "dialogue box", to: "dialog"},
}

var media2spellingRes = compileSubs(media2spellingSubs)

// media2properNounRisk holds the spellings that are also names — Jane Grey,
// the AeroSpace and Defence Industries Association — so a capitalized match is
// dropped and only the lowercase word is reported.
var media2properNounRisk = media2set("grey", "defence", "centre", "centres", "programme", "programmes")

// DetectGoogleUsEnglishSpelling reports Commonwealth spellings that have a US
// form in Google's developer documentation style guide.
func DetectGoogleUsEnglishSpelling(text string) []types.Violation {
	found := findSubs(text, "us-english-spelling", media2spellingSubs, media2spellingRes)
	out := found[:0]
	for _, v := range found {
		lower := strings.ToLower(v.MatchedText)
		if lower != v.MatchedText && media2properNounRisk[lower] {
			continue
		}
		v.SuggestedChange = media2matchCase(v.MatchedText, v.SuggestedChange)
		out = append(out, v)
	}
	return out
}

func media2matchCase(matched, replacement string) string {
	if matched == "" || replacement == "" || matched[0] < 'A' || matched[0] > 'Z' {
		return replacement
	}
	return strings.ToUpper(replacement[:1]) + replacement[1:]
}
