package detectors

import (
	"regexp"
	"strconv"
	"strings"
	"unicode"
	"unicode/utf8"

	"github.com/yasyf/slop-cop/internal/types"
)

func numbers1isWordByte(b byte) bool {
	return b == '_' || (b >= '0' && b <= '9') || (b >= 'a' && b <= 'z') || (b >= 'A' && b <= 'Z')
}

func numbers1isAlnumByte(b byte) bool {
	return (b >= '0' && b <= '9') || (b >= 'a' && b <= 'z') || (b >= 'A' && b <= 'Z')
}

func numbers1endsWithDigit(s string) bool {
	return s != "" && s[len(s)-1] >= '0' && s[len(s)-1] <= '9'
}

func numbers1isCapitalized(s string) bool {
	r, _ := utf8.DecodeRuneInString(s)
	return unicode.IsUpper(r)
}

func numbers1isAcronym(s string) bool {
	if utf8.RuneCountInString(s) < 2 {
		return false
	}
	letters := 0
	for _, r := range s {
		if unicode.IsDigit(r) {
			continue
		}
		if !unicode.IsUpper(r) {
			return false
		}
		letters++
	}
	return letters > 0
}

var numbers1reAdjacentNumerals = regexp.MustCompile(`\b([0-9][0-9,]*)\s+([0-9][0-9,]*)(?:-[a-z]+|\s+(?:GB|MB|KB|kB|byte|bytes|bit|bits))\b`)

// DetectGoogleAdjacentNumerals reports a numeral immediately followed by a
// second numeral that heads a hyphenated compound or a unit phrase.
func DetectGoogleAdjacentNumerals(text string) []types.Violation {
	var out []types.Violation
	for _, m := range numbers1reAdjacentNumerals.FindAllStringSubmatchIndex(text, -1) {
		if m[0] > 0 && strings.IndexByte(`-,./xX`, text[m[0]-1]) >= 0 {
			continue
		}
		if !numbers1endsWithDigit(text[m[2]:m[3]]) || !numbers1endsWithDigit(text[m[4]:m[5]]) {
			continue
		}
		out = append(out, types.Violation{
			RuleID:      "adjacent-numerals",
			StartIndex:  m[0],
			EndIndex:    m[1],
			MatchedText: text[m[0]:m[1]],
			Explanation: "Two numerals sit side by side.",
		})
	}
	return out
}

var (
	numbers1reAmPm      = regexp.MustCompile(`(?i)\b(\d{1,2})(:\d{2})?([ \t\x{00A0}]*)(a\.m\.|p\.m\.|a\.m|p\.m|am|pm)`)
	numbers1reAmPmExact = regexp.MustCompile(`^\d{1,2}(?::\d{2})? (?:AM|PM)$`)
)

// DetectGoogleAmPmFormat reports a clock time whose meridiem marker is
// lowercase, punctuated, or not separated from the digits by one space.
func DetectGoogleAmPmFormat(text string) []types.Violation {
	var out []types.Violation
	for _, m := range numbers1reAmPm.FindAllStringSubmatchIndex(text, -1) {
		if numbers1reAmPmExact.MatchString(text[m[0]:m[1]]) {
			continue
		}
		if m[0] > 0 && strings.IndexByte(".:", text[m[0]-1]) >= 0 {
			continue
		}
		if m[1] < len(text) && numbers1isWordByte(text[m[1]]) {
			continue
		}
		clock := text[m[2]:m[3]]
		if m[4] >= 0 {
			clock += text[m[4]:m[5]]
		}
		marker := "AM"
		if text[m[8]] == 'p' || text[m[8]] == 'P' {
			marker = "PM"
		}
		out = append(out, types.Violation{
			RuleID:          "am-pm-format",
			StartIndex:      m[0],
			EndIndex:        m[1],
			MatchedText:     text[m[0]:m[1]],
			SuggestedChange: clock + " " + marker,
		})
	}
	return out
}

var (
	numbers1reSlashDate  = regexp.MustCompile(`\b(\d{1,4})/(\d{1,2})/(\d{2,4})\b`)
	numbers1reDotDate    = regexp.MustCompile(`\b(\d{1,4})\.(\d{1,2})\.(\d{4})\b`)
	numbers1reDashDate   = regexp.MustCompile(`\b(\d{1,2})-(\d{1,2})-(\d{2})\b`)
	numbers1reVersionCue = regexp.MustCompile(`(?i)(?:\b(?:version|release|rev|build|v)\.?\s*$|\b(?:upgrade|update|upgraded|updated|downgrade|downgraded)\s+to\s+$)`)
)

func numbers1leadPart(s string) bool {
	n, err := strconv.Atoi(s)
	if err != nil {
		return false
	}
	if len(s) == 4 {
		return n >= 1000 && n <= 2999
	}
	if len(s) > 2 {
		return false
	}
	return n >= 1 && n <= 31
}

func numbers1monthPart(s string) bool {
	n, err := strconv.Atoi(s)
	return err == nil && n >= 1 && n <= 12
}

func numbers1yearPart(s string) bool {
	n, err := strconv.Atoi(s)
	if err != nil {
		return false
	}
	if len(s) == 4 {
		return n >= 1000 && n <= 2999
	}
	return len(s) == 2
}

func numbers1inLongerRun(text string, start, end int) bool {
	if start > 0 && strings.IndexByte(`./-`, text[start-1]) >= 0 {
		return true
	}
	if end < len(text) && strings.IndexByte(`./-`, text[end]) >= 0 {
		return end+1 < len(text) && text[end+1] >= '0' && text[end+1] <= '9'
	}
	return false
}

func numbers1hasVersionCue(text string, start int) bool {
	from := start - 20
	if from < 0 {
		from = 0
	}
	return numbers1reVersionCue.MatchString(text[from:start])
}

func numbers1numericDates(text string, re *regexp.Regexp) []types.Violation {
	var out []types.Violation
	for _, m := range re.FindAllStringSubmatchIndex(text, -1) {
		if !numbers1leadPart(text[m[2]:m[3]]) || !numbers1monthPart(text[m[4]:m[5]]) || !numbers1yearPart(text[m[6]:m[7]]) {
			continue
		}
		if numbers1inLongerRun(text, m[0], m[1]) || numbers1hasVersionCue(text, m[0]) {
			continue
		}
		out = append(out, types.Violation{
			RuleID:      "ambiguous-numeric-date",
			StartIndex:  m[0],
			EndIndex:    m[1],
			MatchedText: text[m[0]:m[1]],
			Explanation: "An all-numeric date reads month-first in some countries and day-first in others.",
		})
	}
	return out
}

// DetectGoogleAmbiguousNumericDate reports a date written entirely in digits,
// where the ordering of day and month is ambiguous across locales.
func DetectGoogleAmbiguousNumericDate(text string) []types.Violation {
	out := numbers1numericDates(text, numbers1reSlashDate)
	out = append(out, numbers1numericDates(text, numbers1reDotDate)...)
	return append(out, numbers1numericDates(text, numbers1reDashDate)...)
}

var (
	numbers1reCurrencySpaceAfter = regexp.MustCompile(`\$[ \t\x{00A0}]+\d`)
	numbers1reCurrencySpaceGroup = regexp.MustCompile(`\$\d{1,3}(?:[ \x{00A0}\x{202F}]\d{3})+`)
	numbers1reCurrencyDecimalSep = regexp.MustCompile(`\$\d*\.\d{1,3}[, \t\x{00A0}]\d`)
	numbers1reCurrencyTrailing   = regexp.MustCompile(`\b(\d[\d,]*(?:\.\d+)?)[ \t]?\$`)
	numbers1reCurrencyDollarWord = regexp.MustCompile(`(?i)\b(\d[\d,.]*)\s+dollars\b`)
	numbers1thousandsComma       = strings.NewReplacer(" ", ",", "\u00A0", ",", "\u202F", ",")
)

// DetectGoogleCurrencyFormat reports a dollar amount whose symbol, thousands
// separator, or decimal fraction is punctuated the wrong way.
func DetectGoogleCurrencyFormat(text string) []types.Violation {
	var out []types.Violation
	add := func(start, end int, suggestion string) {
		out = append(out, types.Violation{
			RuleID:          "currency-format",
			StartIndex:      start,
			EndIndex:        end,
			MatchedText:     text[start:end],
			SuggestedChange: suggestion,
		})
	}
	for _, idx := range numbers1reCurrencySpaceAfter.FindAllStringIndex(text, -1) {
		matched := text[idx[0]:idx[1]]
		add(idx[0], idx[1], "$"+matched[len(matched)-1:])
	}
	for _, idx := range numbers1reCurrencySpaceGroup.FindAllStringIndex(text, -1) {
		add(idx[0], idx[1], numbers1thousandsComma.Replace(text[idx[0]:idx[1]]))
	}
	for _, idx := range numbers1reCurrencyDecimalSep.FindAllStringIndex(text, -1) {
		runes := []rune(text[idx[0]:idx[1]])
		add(idx[0], idx[1], string(runes[:len(runes)-2])+string(runes[len(runes)-1]))
	}
	for _, m := range numbers1reCurrencyTrailing.FindAllStringSubmatchIndex(text, -1) {
		if m[0] > 0 && text[m[0]-1] == '$' {
			continue
		}
		if m[1] < len(text) && (numbers1isWordByte(text[m[1]]) || text[m[1]] == '{' || text[m[1]] == '(') {
			continue
		}
		add(m[0], m[1], "$"+text[m[2]:m[3]])
	}
	for _, m := range numbers1reCurrencyDollarWord.FindAllStringSubmatchIndex(text, -1) {
		add(m[0], m[1], "$"+text[m[2]:m[3]])
	}
	return out
}

const (
	numbers1dayAlt   = `(?:Monday|Mon|Tuesday|Tues|Tue|Wednesday|Wed|Thursday|Thurs|Thur|Thu|Friday|Fri|Saturday|Sat|Sunday|Sun)`
	numbers1monthAlt = `(?:January|Jan|February|Feb|March|Mar|April|Apr|May|June|Jun|July|Jul|August|Aug|September|Sept|Sep|October|Oct|November|Nov|December|Dec)`
)

var (
	numbers1reDateMixed  = regexp.MustCompile(`\b(` + numbers1dayAlt + `)\.?,?[ \t]+(` + numbers1monthAlt + `)\.?[ \t]+\d{1,2}\b`)
	numbers1reDatePeriod = regexp.MustCompile(`\b(Jan|Feb|Mar|Apr|Jun|Jul|Aug|Sept|Sep|Oct|Nov|Dec|Mon|Tues|Tue|Wed|Thurs|Thur|Thu|Fri|Sat|Sun)\.[ \t]+(\d{1,4})\b`)

	numbers1shortDates = map[string]string{
		"mon": "Mon", "tue": "Tue", "tues": "Tue", "wed": "Wed", "thu": "Thu",
		"thur": "Thu", "thurs": "Thu", "fri": "Fri", "sat": "Sat", "sun": "Sun",
		"jan": "Jan", "feb": "Feb", "mar": "Mar", "apr": "Apr", "jun": "Jun",
		"jul": "Jul", "aug": "Aug", "sep": "Sep", "sept": "Sep", "oct": "Oct",
		"nov": "Nov", "dec": "Dec",
	}
	numbers1longDates = map[string]string{
		"monday": "Mon", "tuesday": "Tue", "wednesday": "Wed", "thursday": "Thu",
		"friday": "Fri", "saturday": "Sat", "sunday": "Sun",
		"january": "Jan", "february": "Feb", "march": "Mar", "april": "Apr",
		"june": "Jun", "july": "Jul", "august": "Aug", "september": "Sep",
		"october": "Oct", "november": "Nov", "december": "Dec",
	}
)

func numbers1abbreviate(word string) string {
	lower := strings.ToLower(word)
	if short, ok := numbers1shortDates[lower]; ok {
		return short
	}
	if short, ok := numbers1longDates[lower]; ok {
		return short
	}
	return word
}

// DetectGoogleDateAbbreviationConsistency reports a date that abbreviates some
// of its elements but not others, or that punctuates an abbreviation.
func DetectGoogleDateAbbreviationConsistency(text string) []types.Violation {
	var out []types.Violation
	for _, m := range numbers1reDateMixed.FindAllStringSubmatchIndex(text, -1) {
		day := strings.ToLower(text[m[2]:m[3]])
		month := strings.ToLower(text[m[4]:m[5]])
		_, dayShort := numbers1shortDates[day]
		_, dayLong := numbers1longDates[day]
		_, monthShort := numbers1shortDates[month]
		_, monthLong := numbers1longDates[month]
		if (!dayShort || !monthLong) && (!dayLong || !monthShort) {
			continue
		}
		separator := strings.ReplaceAll(text[m[3]:m[4]], ".", "")
		tail := strings.ReplaceAll(text[m[5]:m[1]], ".", "")
		out = append(out, types.Violation{
			RuleID:          "date-abbreviation-consistency",
			StartIndex:      m[0],
			EndIndex:        m[1],
			MatchedText:     text[m[0]:m[1]],
			SuggestedChange: numbers1abbreviate(text[m[2]:m[3]]) + separator + numbers1abbreviate(text[m[4]:m[5]]) + tail,
		})
	}
	for _, m := range numbers1reDatePeriod.FindAllStringSubmatchIndex(text, -1) {
		out = append(out, types.Violation{
			RuleID:          "date-abbreviation-consistency",
			StartIndex:      m[0],
			EndIndex:        m[1],
			MatchedText:     text[m[0]:m[1]],
			SuggestedChange: numbers1abbreviate(text[m[2]:m[3]]) + " " + text[m[4]:m[5]],
		})
	}
	return out
}

var (
	numbers1reKSpaced  = regexp.MustCompile(`\b(\d+)[ \t]+k\b`)
	numbers1reKSuffix  = regexp.MustCompile(`\b\d+k\b`)
	numbers1reKBadTail = regexp.MustCompile(`^(?:[.!?]|\s+(?:is|are|was|were|per|in|on|for|to|and|or)\b)`)
)

// DetectGoogleKSuffixThousands reports a thousands suffix detached from its
// number or left without the noun it counts.
func DetectGoogleKSuffixThousands(text string) []types.Violation {
	var out []types.Violation
	for _, m := range numbers1reKSpaced.FindAllStringSubmatchIndex(text, -1) {
		if m[1] < len(text) && text[m[1]] == '-' {
			continue
		}
		out = append(out, types.Violation{
			RuleID:          "k-suffix-thousands",
			StartIndex:      m[0],
			EndIndex:        m[1],
			MatchedText:     text[m[0]:m[1]],
			SuggestedChange: text[m[2]:m[3]] + "k",
		})
	}
	for _, idx := range numbers1reKSuffix.FindAllStringIndex(text, -1) {
		if idx[1] < len(text) && !numbers1reKBadTail.MatchString(text[idx[1]:]) {
			continue
		}
		out = append(out, types.Violation{
			RuleID:      "k-suffix-thousands",
			StartIndex:  idx[0],
			EndIndex:    idx[1],
			MatchedText: text[idx[0]:idx[1]],
			Explanation: "A bare thousands suffix reads as kilobytes; name what it counts.",
		})
	}
	return out
}

var numbers1reBareDecimal = regexp.MustCompile(`(?:^|[^0-9A-Za-z.\x{2026}])(\.\d+)`)

// DetectGoogleLeadingZeroDecimal reports a value below one written without the
// zero in front of its decimal point.
func DetectGoogleLeadingZeroDecimal(text string) []types.Violation {
	var out []types.Violation
	for _, m := range numbers1reBareDecimal.FindAllStringSubmatchIndex(text, -1) {
		if m[2] > 0 && strings.IndexByte(`/\`, text[m[2]-1]) >= 0 {
			continue
		}
		if m[3] < len(text) {
			next := text[m[3]]
			if (next >= 'a' && next <= 'z') || (next >= 'A' && next <= 'Z') {
				continue
			}
			if next == '.' && m[3]+1 < len(text) && numbers1isAlnumByte(text[m[3]+1]) {
				continue
			}
		}
		out = append(out, types.Violation{
			RuleID:          "leading-zero-decimal",
			StartIndex:      m[2],
			EndIndex:        m[3],
			MatchedText:     text[m[2]:m[3]],
			SuggestedChange: "0" + text[m[2]:m[3]],
		})
	}
	return out
}

var numbers1reMonthYearComma = regexp.MustCompile(`\b(January|February|March|April|May|June|July|August|September|October|November|December),[ \t]+((?:19|20|21)\d{2})\b`)

// DetectGoogleMonthYearComma reports a comma between a month and a year, which
// belongs only in the full month-day-year form.
func DetectGoogleMonthYearComma(text string) []types.Violation {
	out := make([]types.Violation, 0, 4)
	for _, m := range numbers1reMonthYearComma.FindAllStringSubmatchIndex(text, -1) {
		out = append(out, types.Violation{
			RuleID:          "month-year-comma",
			StartIndex:      m[0],
			EndIndex:        m[1],
			MatchedText:     text[m[0]:m[1]],
			SuggestedChange: text[m[2]:m[3]] + " " + text[m[4]:m[5]],
		})
	}
	return out
}

var (
	numbers1reLoneDigit  = regexp.MustCompile(`\b[0-9]\b`)
	numbers1reMultiDigit = regexp.MustCompile(`[0-9]{2,}`)
	numbers1reTimesDigit = regexp.MustCompile(`^[ \t]*[x\x{00D7}][ \t]*[0-9]`)

	numbers1priorTokens = map[string]bool{
		"version": true, "v": true, "step": true, "chapter": true, "section": true,
		"page": true, "figure": true, "fig": true, "table": true, "part": true,
		"level": true, "tier": true, "port": true, "phase": true, "release": true,
		"http": true, "api": true, "code": true, "id": true, "index": true,
		"line": true, "column": true, "row": true, "day": true, "week": true,
		"month": true, "quarter": true, "node": true, "worker": true, "replica": true,
		"group": true, "type": true, "class": true, "task": true, "job": true,
		"round": true, "stage": true, "layer": true, "zone": true, "region": true,
		"slot": true, "attempt": true, "retry": true, "build": true, "appendix": true,
		"item": true, "priority": true, "rank": true, "exit": true, "exits": true,
		"returns": true, "entry": true, "sha": true, "utf": true, "ipv": true,
	}

	numbers1unitTokens = map[string]bool{
		"gb": true, "mb": true, "kb": true, "tb": true, "pb": true, "eb": true,
		"kib": true, "mib": true, "gib": true, "tib": true, "bit": true, "bits": true,
		"byte": true, "bytes": true, "px": true, "pixel": true, "pixels": true,
		"ms": true, "ns": true, "us": true, "s": true, "min": true, "h": true,
		"hr": true, "hz": true, "khz": true, "mhz": true, "ghz": true, "db": true,
		"bps": true, "kbps": true, "mbps": true, "gbps": true, "mm": true, "cm": true,
		"m": true, "km": true, "kg": true, "kw": true, "ma": true, "vcpu": true,
		"vcpus": true, "cpus": true, "cores": true, "threads": true, "qps": true,
		"rps": true, "tps": true, "iops": true, "queries": true, "requests": true,
		"operations": true, "degrees": true, "characters": true, "columns": true,
		"rows": true,
	}

	numbers1properNouns = map[string]bool{
		"python": true, "java": true, "angular": true, "react": true, "vue": true,
		"django": true, "rails": true, "ruby": true, "php": true, "windows": true,
		"macos": true, "ios": true, "android": true, "ubuntu": true, "debian": true,
		"kubernetes": true, "docker": true, "postgres": true, "postgresql": true,
		"mysql": true, "redis": true, "kafka": true, "spark": true, "swift": true,
		"kotlin": true, "scala": true, "rust": true, "typescript": true,
		"javascript": true, "ecmascript": true, "oauth": true, "cuda": true,
		"opengl": true, "directx": true, "gradle": true, "maven": true, "npm": true,
		"webpack": true, "nginx": true, "apache": true, "bootstrap": true,
		"jquery": true, "laravel": true, "symfony": true, "flask": true,
		"spring": true, "tomcat": true, "elasticsearch": true, "mongodb": true,
		"sqlite": true, "oracle": true, "unicode": true, "xcode": true,
		"jenkins": true, "ansible": true, "terraform": true, "helm": true,
		"istio": true, "envoy": true, "grafana": true, "prometheus": true,
		"cassandra": true, "hadoop": true, "airflow": true, "pytorch": true,
		"tensorflow": true, "keras": true, "pandas": true, "numpy": true,
		"perl": true, "lua": true, "dart": true, "flutter": true, "unity": true,
		"blender": true, "opencv": true, "ffmpeg": true, "elixir": true,
		"erlang": true, "haskell": true, "clojure": true, "fortran": true,
		"matlab": true, "chrome": true, "firefox": true, "safari": true,
		"gnome": true, "bash": true, "powershell": true, "node": true,
		"deno": true, "bun": true, "svelte": true, "hugo": true, "jekyll": true,
	}
)

func numbers1sentenceBounds(text string, at int) (int, int) {
	start := 0
	for i := at - 1; i >= 0; i-- {
		c := text[i]
		if c == '.' || c == '!' || c == '?' || c == '\n' {
			start = i + 1
			break
		}
	}
	for start < at && (text[start] == ' ' || text[start] == '\t' || text[start] == '\r') {
		start++
	}
	end := len(text)
	for i := at + 1; i < len(text); i++ {
		c := text[i]
		if c == '.' || c == '!' || c == '?' || c == '\n' {
			end = i
			break
		}
	}
	return start, end
}

func numbers1prevToken(text string, at int) (string, int) {
	i := at
	for i > 0 && (text[i-1] == ' ' || text[i-1] == '\t') {
		i--
	}
	end := i
	for i > 0 && numbers1isWordByte(text[i-1]) {
		i--
	}
	return text[i:end], i
}

func numbers1nextToken(text string, at int) string {
	i := at
	for i < len(text) && (text[i] == ' ' || text[i] == '\t') {
		i++
	}
	j := i
	for j < len(text) && numbers1isWordByte(text[j]) {
		j++
	}
	return text[i:j]
}

var (
	numbers1prevRunes = []string{"€", "£", "¥", "₹", "№", "–", "—", "−", "…", "×", "·"}
	numbers1nextRunes = []string{"°", "–", "—", "−", "…", "×", "·", "′", "″"}
)

func numbers1lineLead(text string, start int) bool {
	lineStart := strings.LastIndexByte(text[:start], '\n') + 1
	return strings.Trim(text[lineStart:start], " \t#>-*+") == ""
}

func numbers1plainDigitContext(text string, start, end int) bool {
	if start > 0 {
		if strings.IndexByte("`.,-:/$#%_+=*^~<>@\\([{&'\"|", text[start-1]) >= 0 {
			return false
		}
		for _, sign := range numbers1prevRunes {
			if strings.HasSuffix(text[:start], sign) {
				return false
			}
		}
	}
	if end < len(text) {
		next := text[end]
		if strings.IndexByte("`-:/%_+=*^~<>@\\)]}&'\"|", next) >= 0 {
			return false
		}
		if (next == '.' || next == ',') && end+1 < len(text) && numbers1isAlnumByte(text[end+1]) {
			return false
		}
		for _, sign := range numbers1nextRunes {
			if strings.HasPrefix(text[end:], sign) {
				return false
			}
		}
		if (next == '.' || next == ')') && numbers1lineLead(text, start) {
			return false
		}
	}
	return true
}

// DetectGoogleNumeralUnderTen reports a count below ten written as a digit in
// ordinary prose, where a word reads better than a numeral.
func DetectGoogleNumeralUnderTen(text string) []types.Violation {
	var out []types.Violation
	for _, idx := range numbers1reLoneDigit.FindAllStringIndex(text, -1) {
		start, end := idx[0], idx[1]
		if !numbers1plainDigitContext(text, start, end) || isTabular(spanLines(text, start, end)[0]) {
			continue
		}
		sentenceStart, sentenceEnd := numbers1sentenceBounds(text, start)
		if numbers1reMultiDigit.MatchString(text[sentenceStart:sentenceEnd]) {
			continue
		}
		prev, prevStart := numbers1prevToken(text, start)
		if prev != "" {
			lower := strings.ToLower(prev)
			if numbers1priorTokens[lower] || numbers1isAcronym(prev) {
				continue
			}
			if numbers1isCapitalized(prev) && (prevStart > sentenceStart || numbers1properNouns[lower]) {
				continue
			}
		}
		if numbers1unitTokens[strings.ToLower(numbers1nextToken(text, end))] {
			continue
		}
		if numbers1reTimesDigit.MatchString(text[end:]) {
			continue
		}
		out = append(out, types.Violation{
			RuleID:      "numeral-under-ten",
			StartIndex:  start,
			EndIndex:    end,
			MatchedText: text[start:end],
			Explanation: "Spell out a count below ten in ordinary prose.",
		})
	}
	return out
}
