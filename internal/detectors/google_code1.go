package detectors

import (
	"regexp"
	"sort"
	"strings"

	"github.com/yasyf/slop-cop/internal/types"
)

func code1hit(ruleID, text string, start, end int, suggest string) types.Violation {
	return types.Violation{
		RuleID:          ruleID,
		StartIndex:      start,
		EndIndex:        end,
		MatchedText:     text[start:end],
		SuggestedChange: suggest,
	}
}

// code1dropOverlaps keeps the longest match at each position and discards any
// later violation that overlaps one already kept.
func code1dropOverlaps(vs []types.Violation) []types.Violation {
	if len(vs) < 2 {
		return vs
	}
	sort.SliceStable(vs, func(i, j int) bool {
		if vs[i].StartIndex != vs[j].StartIndex {
			return vs[i].StartIndex < vs[j].StartIndex
		}
		return vs[i].EndIndex > vs[j].EndIndex
	})
	out := vs[:0]
	end := -1
	for _, v := range vs {
		if v.StartIndex < end {
			continue
		}
		out = append(out, v)
		end = v.EndIndex
	}
	return out
}

func code1isWordByte(b byte) bool {
	return b == '_' || (b >= '0' && b <= '9') || (b >= 'a' && b <= 'z') || (b >= 'A' && b <= 'Z')
}

func code1firstWord(s string) string {
	s = strings.TrimLeft(s, " \t")
	i := 0
	for i < len(s) && code1isWordByte(s[i]) {
		i++
	}
	return strings.ToLower(s[:i])
}

func code1lastWord(s string) string {
	s = strings.TrimRight(s, " \t\x60")
	i := len(s)
	for i > 0 && code1isWordByte(s[i-1]) {
		i--
	}
	return strings.ToLower(s[i:])
}

// code1swapPrefix replaces the first prefixLen bytes of matched with repl,
// matching the capitalization of the text it replaces.
func code1swapPrefix(matched string, prefixLen int, repl string) string {
	if prefixLen > len(matched) || len(repl) == 0 {
		return ""
	}
	head := strings.ToLower(repl)
	if matched[0] >= 'A' && matched[0] <= 'Z' {
		head = strings.ToUpper(repl[:1]) + strings.ToLower(repl[1:])
	}
	return head + matched[prefixLen:]
}

func code1window(text string, start, end, radius int) string {
	lo := start - radius
	if lo < 0 {
		lo = 0
	}
	hi := end + radius
	if hi > len(text) {
		hi = len(text)
	}
	return text[lo:hi]
}

var (
	code1checkboxSubs = []plainSub{
		{"uncheck", "clear"},
		{"unchecks", "clears"},
		{"unchecking", "clearing"},
		{"unselect", "clear"},
		{"unselects", "clears"},
		{"unselecting", "clearing"},
		{"unselected", "not selected"},
	}
	code1checkboxRes = compileSubs(code1checkboxSubs)

	code1reUnchecked     = regexp.MustCompile(`(?i)\bunchecked\b`)
	code1reDeselect      = regexp.MustCompile(`(?i)\bdeselect(?:s|ed|ing)?\b`)
	code1reCheckboxNear  = regexp.MustCompile(`(?i)\bcheck ?(?:box|boxes|mark|marks)\b`)
	code1reUIContext     = regexp.MustCompile(`(?i)\b(?:check ?box(?:es)?|check ?marks?|option|setting|toggle|radio button)\b`)
	code1reCheckCheckbox = regexp.MustCompile(`(?im)(?:^|[.!?:;][ \t]+|\bto[ \t]+|\band[ \t]+(?:then[ \t]+)?)[ \t]*(check(?:s|ed|ing)?\b[^.\n]{0,60}?\bcheckbox\b)`)
	code1reCheckboxState = regexp.MustCompile(`(?i)\bcheckbox\b[^.\n]{0,40}\bis[ \t]+(?:un)?checked\b`)
	code1reCheckLabel    = regexp.MustCompile(`(?im)(?:^|[.!?:;][ \t]+)[ \t]*(check[ \t]+(?:the[ \t]+)?\*\*)`)
	code1reChooseLabel   = regexp.MustCompile(`(?i)\bchoose[ \t]+(?:the[ \t]+)?[*\x60]`)
)

// DetectGoogleCheckboxVerb reports check/uncheck used for a checkbox, where
// select and clear name the action and the resulting state unambiguously.
func DetectGoogleCheckboxVerb(text string) []types.Violation {
	const rule = "checkbox-verb"
	out := findSubs(text, rule, code1checkboxSubs, code1checkboxRes)
	for _, idx := range code1reUnchecked.FindAllStringIndex(text, -1) {
		if code1reUIContext.MatchString(code1window(text, idx[0], idx[1], 60)) {
			out = append(out, code1hit(rule, text, idx[0], idx[1], "not selected"))
		}
	}
	for _, idx := range code1reDeselect.FindAllStringIndex(text, -1) {
		if code1reCheckboxNear.MatchString(code1window(text, idx[0], idx[1], 60)) {
			out = append(out, code1hit(rule, text, idx[0], idx[1], "clear"))
		}
	}
	for _, m := range code1reCheckCheckbox.FindAllStringSubmatchIndex(text, -1) {
		v := code1hit(rule, text, m[2], m[3], "")
		if strings.EqualFold(v.MatchedText[:5], "check") {
			v.SuggestedChange = code1swapPrefix(v.MatchedText, 5, "select")
		}
		out = append(out, v)
	}
	for _, idx := range code1reCheckboxState.FindAllStringIndex(text, -1) {
		out = append(out, code1hit(rule, text, idx[0], idx[1], ""))
	}
	for _, m := range code1reCheckLabel.FindAllStringSubmatchIndex(text, -1) {
		v := code1hit(rule, text, m[2], m[3], "")
		v.SuggestedChange = code1swapPrefix(v.MatchedText, 5, "select")
		out = append(out, v)
	}
	for _, idx := range code1reChooseLabel.FindAllStringIndex(text, -1) {
		v := code1hit(rule, text, idx[0], idx[1], "")
		v.SuggestedChange = code1swapPrefix(v.MatchedText, 6, "select")
		out = append(out, v)
	}
	return code1dropOverlaps(out)
}

var code1extNames = map[string]string{
	"adoc":  "AsciiDoc",
	"csv":   "CSV",
	"exe":   "executable",
	"gif":   "GIF",
	"img":   "disk image",
	"ipynb": "Jupyter notebook",
	"jar":   "JAR",
	"jpeg":  "JPEG",
	"jpg":   "JPEG",
	"json":  "JSON",
	"md":    "Markdown",
	"pdf":   "PDF",
	"png":   "PNG",
	"ps":    "PostScript",
	"ps1":   "PowerShell",
	"py":    "Python",
	"sh":    "Bash",
	"sql":   "SQL",
	"svg":   "SVG",
	"tar":   "tar archive",
	"tf":    "Terraform",
	"tif":   "TIFF",
	"tiff":  "TIFF",
	"txt":   "text",
	"wasm":  "WebAssembly",
	"yaml":  "YAML",
	"yml":   "YAML",
	"zip":   "ZIP",
}

var (
	code1reFileExt = regexp.MustCompile(`(?i)(?:^|[\s(\[])\x60?(\.(?:adoc|csv|exe|gif|img|ipynb|jar|jpe?g|json|md|pdf|png|ps1?|py|sh|sql|svg|tar|tf|tiff?|txt|wasm|ya?ml|zip))\b`)

	code1extNouns = map[string]bool{
		"file": true, "files": true, "format": true, "formats": true,
		"image": true, "images": true, "document": true, "documents": true,
	}
	code1extSkipNouns = map[string]bool{
		"extension": true, "extensions": true, "suffix": true, "suffixes": true,
	}
	code1extDeterminers = map[string]bool{"a": true, "an": true, "the": true}
)

// DetectGoogleExtensionAsFileType reports a dotted filename extension standing
// in for the name of a file format.
func DetectGoogleExtensionAsFileType(text string) []types.Violation {
	const rule = "extension-as-file-type"
	var out []types.Violation
	for _, m := range code1reFileExt.FindAllStringSubmatchIndex(text, -1) {
		start, end := m[2], m[3]
		name, known := code1extNames[strings.ToLower(text[start+1:end])]
		if !known {
			continue
		}
		rest := strings.TrimLeft(text[end:], "\x60")
		if strings.HasPrefix(rest, ".") {
			continue
		}
		next := code1firstWord(rest)
		if code1extSkipNouns[next] {
			continue
		}
		if !code1extNouns[next] && !code1extDeterminers[code1lastWord(text[:start])] {
			continue
		}
		if start > 0 && text[start-1] == '\x60' {
			start--
		}
		if end < len(text) && text[end] == '\x60' {
			end++
		}
		out = append(out, code1hit(rule, text, start, end, name))
	}
	return out
}

var (
	code1uiNounSubs = []plainSub{
		{"hamburger icon", "Menu button"},
		{"hamburger menu", "Menu button"},
		{"hamburger button", "Menu button"},
		{"kebab menu", "More options menu"},
		{"zippy", "expander arrow"},
		{"expando", "expander arrow"},
		{"disclosure triangle", "expander arrow"},
		{"disclosure widget", "expander arrow"},
		{"drop-down list", "list"},
		{"dropdown list", "list"},
		{"drop down list", "list"},
		{"drop-down box", "list"},
		{"dropdown box", "list"},
		{"drop-down menu", "menu"},
		{"dropdown menu", "menu"},
		{"drop down menu", "menu"},
		{"drop-downs", "lists"},
		{"dropdowns", "lists"},
		{"drop-down", "list"},
		{"dropdown", "list"},
		{"pop-up window", "dialog"},
		{"popup window", "dialog"},
		{"pop-ups", "dialogs"},
		{"popups", "dialogs"},
		{"pop-up", "dialog"},
		{"popup", "dialog"},
		{"navigation bar", "navigation menu"},
		{"navigation pane", "navigation menu"},
		{"navigation panel", "navigation menu"},
		{"navigation window", "navigation menu"},
		{"menu item", "command"},
		{"menu items", "commands"},
		{"menu choice", "command"},
		{"menu choices", "commands"},
		{"menu option", "command"},
		{"menu options", "commands"},
		{"text box", "field"},
		{"textbox", "field"},
		{"text boxes", "fields"},
		{"textboxes", "fields"},
		{"dialog box", "dialog"},
		{"dialog boxes", "dialogs"},
	}
	code1uiNounRes = compileSubs(code1uiNounSubs)

	code1reUIArea = regexp.MustCompile(`(?i)\bthe[ \t]+\*\*[^*\n]{1,40}\*\*[ \t]+(?:area|column)\b`)
	code1reUITab  = regexp.MustCompile(`\b(?i:the)[ \t]+(?:\*\*)?[A-Z][A-Za-z0-9]{0,20}(?:[ ][A-Z][A-Za-z0-9]{0,20}){0,2}(?:\*\*)?[ \t]+tab\b`)
)

// DetectGoogleImpreciseUINoun reports slang or near-synonyms for a named
// interface element.
func DetectGoogleImpreciseUINoun(text string) []types.Violation {
	const rule = "imprecise-ui-noun"
	out := findSubs(text, rule, code1uiNounSubs, code1uiNounRes)
	for _, idx := range code1reUIArea.FindAllStringIndex(text, -1) {
		v := code1hit(rule, text, idx[0], idx[1], "")
		if strings.HasSuffix(strings.ToLower(v.MatchedText), "area") {
			v.SuggestedChange = v.MatchedText[:len(v.MatchedText)-4] + "section"
		}
		out = append(out, v)
	}
	for _, idx := range code1reUITab.FindAllStringIndex(text, -1) {
		v := code1hit(rule, text, idx[0], idx[1], "")
		v.SuggestedChange = v.MatchedText[:len(v.MatchedText)-3] + "page"
		out = append(out, v)
	}
	return code1dropOverlaps(out)
}

var (
	code1reCodeInflect = regexp.MustCompile(`\x60[^\x60\n]+\x60(?:'s|\x{2019}s|s|es|ed|ing)\b`)
	code1reCodeVerb    = regexp.MustCompile(`(?i)\x60(?:GET|POST|PUT|DELETE|PATCH)\x60(?:ting|ing|s|ed)?[ \t]+(?:the|a|an|your)\b`)
	code1reCamelPoss   = regexp.MustCompile(`\b[a-z]+[A-Z][A-Za-z0-9]*['\x{2019}]s\b`)
	code1reSnakePoss   = regexp.MustCompile(`\b[a-z][a-z0-9]*_[a-z0-9_]+['\x{2019}]s\b`)
	code1reCallPoss    = regexp.MustCompile(`\b[A-Za-z_][A-Za-z0-9_]*\(\)['\x{2019}]s`)

	code1camelSkip = map[string]bool{
		"iphone": true, "ipad": true, "ipod": true, "ios": true, "ipados": true,
		"imac": true, "icloud": true, "itunes": true, "ebay": true,
		"macos": true, "watchos": true, "tvos": true, "javascript": true,
		"ipv4": true, "ipv6": true, "youtube": true,
	}
)

// DetectGoogleInflectedCodeElement reports a code element that has been
// pluralized, made possessive, or conjugated, leaving a string the reader
// cannot type or search for.
func DetectGoogleInflectedCodeElement(text string) []types.Violation {
	const rule = "inflected-code-element"
	var out []types.Violation
	for _, re := range []*regexp.Regexp{code1reCodeInflect, code1reCodeVerb, code1reCallPoss} {
		for _, idx := range re.FindAllStringIndex(text, -1) {
			out = append(out, code1hit(rule, text, idx[0], idx[1], ""))
		}
	}
	for _, re := range []*regexp.Regexp{code1reCamelPoss, code1reSnakePoss} {
		for _, idx := range re.FindAllStringIndex(text, -1) {
			word := text[idx[0]:idx[1]]
			bare := strings.ToLower(word[:strings.LastIndexAny(word, "'’")])
			if code1camelSkip[bare] {
				continue
			}
			out = append(out, code1hit(rule, text, idx[0], idx[1], ""))
		}
	}
	return code1dropOverlaps(out)
}

var (
	code1reFence        = regexp.MustCompile(`^[ \t]{0,3}(?:\x60{3,}|~{3,})`)
	code1reOutputIntro  = regexp.MustCompile(`(?i)\b(?:you (?:should|will|'ll|\x{2019}ll) see\b|the output should (?:look|be)\b|sample output\b|example output\b|this (?:will )?(?:returns?|outputs?|prints?)\b|it (?:will )?prints?\b)`)
	code1reOutputWeak   = regexp.MustCompile(`(?i)\bresults?[ \t]*:`)
	code1outputStandard = "The output is similar to the following"
)

// DetectGoogleOutputIntroPhrase reports a nonstandard lead-in for sample
// command output.
func DetectGoogleOutputIntroPhrase(text string) []types.Violation {
	const rule = "output-intro-phrase"
	var out []types.Violation
	lines := strings.Split(text, "\n")
	off := 0
	inFence := false
	for i, line := range lines {
		start := off
		off += len(line) + 1
		if code1reFence.MatchString(line) {
			inFence = !inFence
			continue
		}
		if inFence || isATXHeading(line) {
			continue
		}
		fenced := code1fenceFollows(lines, i)
		colon := strings.HasSuffix(strings.TrimRight(line, " \t"), ":")
		if colon || fenced {
			for _, idx := range code1reOutputIntro.FindAllStringIndex(line, -1) {
				out = append(out, code1hit(rule, text, start+idx[0], start+idx[1], code1outputStandard))
			}
		}
		if fenced {
			for _, idx := range code1reOutputWeak.FindAllStringIndex(line, -1) {
				out = append(out, code1hit(rule, text, start+idx[0], start+idx[1], code1outputStandard+":"))
			}
		}
	}
	return code1dropOverlaps(out)
}

func code1fenceFollows(lines []string, i int) bool {
	for j := i + 1; j < len(lines) && j <= i+3; j++ {
		if strings.TrimSpace(lines[j]) == "" {
			continue
		}
		return code1reFence.MatchString(lines[j]) || strings.HasPrefix(lines[j], "    ")
	}
	return false
}

var code1reToggleVerb = regexp.MustCompile(`(?i)\b(?:to[ \t]+toggle\b|toggle[ \t]+(?:on|off)\b|toggled\b|toggling\b|toggles[ \t]+(?:the|a|an|it|this|that|between)\b|toggle[ \t]+(?:the[ \t]+)?[^.\n]{0,30}?\b(?:setting|option|switch|control)s?\b)`)

// DetectGoogleToggleAsVerb reports toggle used as a verb, which leaves the
// reader without an end state to aim for.
func DetectGoogleToggleAsVerb(text string) []types.Violation {
	return findAll(text, code1reToggleVerb, "toggle-as-verb")
}

var (
	code1reQuotedLabel = regexp.MustCompile(`(?i)\b(click|tap|select|choose|press)[ \t]+(?:on[ \t]+)?(?:the[ \t]+)?["\x{201C}]([^"\x{201D}\n]{1,40})["\x{201D}]`)
	code1reButtonLabel = regexp.MustCompile(`(?i:\b(click|tap))[ \t]+the[ \t]+\*{0,2}([A-Z][\w ]{0,30}?)\*{0,2}[ \t]+button\b`)
	code1reClickOn     = regexp.MustCompile(`(?i)\bclick on\b`)
)

// DetectGoogleUILabelDecoration reports a UI label wrapped in quotation marks
// or padded with its element type.
func DetectGoogleUILabelDecoration(text string) []types.Violation {
	const rule = "ui-label-decoration"
	var out []types.Violation
	for _, re := range []*regexp.Regexp{code1reQuotedLabel, code1reButtonLabel} {
		for _, m := range re.FindAllStringSubmatchIndex(text, -1) {
			verb := text[m[2]:m[3]]
			label := strings.TrimSpace(text[m[4]:m[5]])
			out = append(out, code1hit(rule, text, m[0], m[1], verb+" **"+label+"**"))
		}
	}
	for _, idx := range code1reClickOn.FindAllStringIndex(text, -1) {
		out = append(out, code1hit(rule, text, idx[0], idx[1], text[idx[0]:idx[0]+5]))
	}
	return code1dropOverlaps(out)
}

var (
	code1reOnContainer = regexp.MustCompile(`(?i)\bon[ \t]+the[ \t]+([\w *]{1,40}?)[ \t](?:dialog|field|list|menu|pane|window)\b`)
	code1reInSurface   = regexp.MustCompile(`(?i)\bin[ \t]+the[ \t]+([\w *]{1,40}?)[ \t](?:page|tab|toolbar)\b`)
)

// DetectGoogleUIPreposition reports "on" used for a container the reader looks
// inside, or "in" used for a surface the reader acts on.
func DetectGoogleUIPreposition(text string) []types.Violation {
	const rule = "ui-preposition"
	var out []types.Violation
	for _, m := range code1reOnContainer.FindAllStringSubmatchIndex(text, -1) {
		if !code1isUILabel(text[m[2]:m[3]]) {
			continue
		}
		v := code1hit(rule, text, m[0], m[1], "")
		v.SuggestedChange = code1swapPrefix(v.MatchedText, 2, "in")
		out = append(out, v)
	}
	for _, m := range code1reInSurface.FindAllStringSubmatchIndex(text, -1) {
		if !code1isUILabel(text[m[2]:m[3]]) {
			continue
		}
		v := code1hit(rule, text, m[0], m[1], "")
		v.SuggestedChange = code1swapPrefix(v.MatchedText, 2, "on")
		out = append(out, v)
	}
	return code1dropOverlaps(out)
}

// code1isUILabel reports whether label reads as the visible name of a control
// — marked up, or capitalized — rather than as an ordinary noun phrase.
func code1isUILabel(label string) bool {
	if strings.Contains(label, "*") {
		return true
	}
	return len(label) > 0 && label[0] >= 'A' && label[0] <= 'Z'
}

const code1cmdAlt = `gcloud|gsutil|kubectl|kubeadm|bq|npx|npm|pnpm|yarn|git|curl|wget|docker|pip3|pip|python3|terraform|cargo|apt-get|yum|brew|ssh|scp|sudo`

var (
	code1rePromptCmd = regexp.MustCompile(`^[ \t]{0,3}\$[ \t]*(?:` + code1cmdAlt + `)[ \t]+[\w./-]+`)
	code1reBareCmd   = regexp.MustCompile(`^[ \t]{0,3}(?:` + code1cmdAlt + `)[ \t]+[\w./-]+`)
	code1reRunCmd    = regexp.MustCompile(`(?i:\b(?:run|execute))[ \t]+(?:the[ \t]+following[ \t]+commands?:?[ \t]*)?(?:` + code1cmdAlt + `)[ \t]+[\w./-]+`)
	code1reLinkSpan  = regexp.MustCompile(`\]\([^)\n]*\)|<[^>\n]*>`)
)

// DetectGoogleUnfencedCommandLine reports a command left in running prose,
// where the renderer is free to mangle its spacing and special characters.
func DetectGoogleUnfencedCommandLine(text string) []types.Violation {
	const rule = "unfenced-command-line"
	var out []types.Violation
	off := 0
	inFence := false
	for _, line := range strings.Split(text, "\n") {
		start := off
		off += len(line) + 1
		if code1reFence.MatchString(line) {
			inFence = !inFence
			continue
		}
		if inFence || strings.HasPrefix(line, "    ") || strings.HasPrefix(line, "\t") || isTabular(line) {
			continue
		}
		var spans [][]int
		if idx := code1rePromptCmd.FindStringIndex(line); idx != nil {
			spans = append(spans, idx)
		} else if idx := code1reBareCmd.FindStringIndex(line); idx != nil && code1isCommandLine(line) {
			spans = append(spans, idx)
		}
		spans = append(spans, code1reRunCmd.FindAllStringIndex(line, -1)...)
		for _, idx := range spans {
			if code1inlineCodeAt(line, idx[0]) || code1inLinkAt(line, idx[0]) {
				continue
			}
			v := code1hit(rule, text, start+idx[0], start+idx[1], "")
			v.Explanation = "Put the command in a code block, or wrap a short mention in backticks."
			out = append(out, v)
		}
	}
	return code1dropOverlaps(out)
}

// code1isCommandLine reports whether a line that opens with a tool name reads
// as an invocation rather than as a sentence about the tool.
func code1isCommandLine(line string) bool {
	trimmed := strings.TrimSpace(line)
	if strings.HasSuffix(trimmed, ".") || strings.HasSuffix(trimmed, ",") || strings.HasSuffix(trimmed, ":") {
		return false
	}
	fields := strings.Fields(trimmed)
	for _, f := range fields[1:] {
		if strings.HasPrefix(f, "-") || strings.ContainsAny(f, "/=") {
			return true
		}
	}
	return len(fields) <= 4
}

func code1inlineCodeAt(line string, pos int) bool {
	return strings.Count(line[:pos], "\x60")%2 == 1
}

func code1inLinkAt(line string, pos int) bool {
	for _, idx := range code1reLinkSpan.FindAllStringIndex(line, -1) {
		if pos >= idx[0] && pos < idx[1] {
			return true
		}
	}
	return false
}
