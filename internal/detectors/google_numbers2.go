package detectors

import (
	"regexp"
	"strconv"
	"strings"

	"github.com/yasyf/slop-cop/internal/types"
)

var (
	numbers2rePercentSpace   = regexp.MustCompile(`\b\d[\d,.]*[ \t]+%`)
	numbers2rePercentWord    = regexp.MustCompile(`(?i)\b\d[\d,.]*[ \t]+percent\b`)
	numbers2rePercentInitial = regexp.MustCompile(`(?m)(?:^|[.!?][ \t]+)(\d[\d,.]*[ \t]*%)`)
	numbers2reLeadingNumber  = regexp.MustCompile(`^\d[\d,.]*`)
)

// DetectGooglePercentFormat reports percentages written with a space before the
// sign, spelled as the word "percent" after a numeral, or opening a sentence
// with digits instead of words.
func DetectGooglePercentFormat(text string) []types.Violation {
	const rule = "percent-format"
	var out []types.Violation
	closed := make(map[int]bool)
	for _, idx := range numbers2rePercentSpace.FindAllStringIndex(text, -1) {
		closed[idx[0]] = true
		match := text[idx[0]:idx[1]]
		out = append(out, types.Violation{
			RuleID:          rule,
			StartIndex:      idx[0],
			EndIndex:        idx[1],
			MatchedText:     match,
			Explanation:     "A percent sign closes up against its number.",
			SuggestedChange: strings.Join(strings.Fields(match), ""),
		})
	}
	for _, idx := range numbers2rePercentWord.FindAllStringIndex(text, -1) {
		closed[idx[0]] = true
		match := text[idx[0]:idx[1]]
		out = append(out, types.Violation{
			RuleID:          rule,
			StartIndex:      idx[0],
			EndIndex:        idx[1],
			MatchedText:     match,
			Explanation:     "A numeral takes the percent sign, not the word.",
			SuggestedChange: numbers2reLeadingNumber.FindString(match) + "%",
		})
	}
	for _, m := range numbers2rePercentInitial.FindAllStringSubmatchIndex(text, -1) {
		if closed[m[2]] {
			continue
		}
		out = append(out, types.Violation{
			RuleID:      rule,
			StartIndex:  m[2],
			EndIndex:    m[3],
			MatchedText: text[m[2]:m[3]],
			Explanation: "A percentage that opens a sentence is spelled out.",
		})
	}
	return out
}

var (
	numbers2reRateSlash = regexp.MustCompile(`(?i)\b(requests?|queries|operations?|calls?|reads?|writes?|files?|bytes?|jobs?|users?|events?|messages?|rows?|transactions?)[ \t]*/[ \t]*(second|sec|minute|min|hour|hr|day|week|month|year|s|d|h)\b`)
	numbers2reRateBits  = regexp.MustCompile(`\b(Gb|Mb|kb|GB|MB|kB|TB)[ \t]*/[ \t]*(?:second|sec|s)\b`)
)

var numbers2rateUnits = map[string]string{
	"s":      "second",
	"sec":    "second",
	"second": "second",
	"min":    "minute",
	"minute": "minute",
	"h":      "hour",
	"hr":     "hour",
	"hour":   "hour",
	"d":      "day",
	"day":    "day",
	"week":   "week",
	"month":  "month",
	"year":   "year",
}

// DetectGoogleRateSlash reports rates written with a division slash where prose
// wants "per", and bandwidth units split by a slash instead of using their
// settled abbreviation.
func DetectGoogleRateSlash(text string) []types.Violation {
	const rule = "rate-slash"
	var out []types.Violation
	for _, m := range numbers2reRateSlash.FindAllStringSubmatchIndex(text, -1) {
		if numbers2inTable(text, m[0], m[1]) {
			continue
		}
		noun := text[m[2]:m[3]]
		unit := numbers2rateUnits[strings.ToLower(text[m[4]:m[5]])]
		out = append(out, types.Violation{
			RuleID:          rule,
			StartIndex:      m[0],
			EndIndex:        m[1],
			MatchedText:     text[m[0]:m[1]],
			Explanation:     "Prose spells a rate with \"per\".",
			SuggestedChange: noun + " per " + unit,
		})
	}
	for _, m := range numbers2reRateBits.FindAllStringSubmatchIndex(text, -1) {
		if numbers2inTable(text, m[0], m[1]) {
			continue
		}
		out = append(out, types.Violation{
			RuleID:          rule,
			StartIndex:      m[0],
			EndIndex:        m[1],
			MatchedText:     text[m[0]:m[1]],
			Explanation:     "Bandwidth has a settled abbreviation.",
			SuggestedChange: text[m[2]:m[3]] + "ps",
		})
	}
	return out
}

func numbers2inTable(text string, start, end int) bool {
	for _, line := range spanLines(text, start, end) {
		if isTabular(line) {
			return true
		}
	}
	return false
}

var numbers2reSentenceNumeral = regexp.MustCompile(`(?m)(?:^|[.!?]["'”’)\]]?\s+)(\d+(?:,\d{3})*)`)

// DetectGoogleSentenceInitialNumeral reports a sentence that opens on digits
// rather than on words.
func DetectGoogleSentenceInitialNumeral(text string) []types.Violation {
	const rule = "sentence-initial-numeral"
	var out []types.Violation
	for _, m := range numbers2reSentenceNumeral.FindAllStringSubmatchIndex(text, -1) {
		start, end := m[2], m[3]
		if m[0] == start {
			if !numbers2opensBlock(text, start) {
				continue
			}
		} else if endsWithAbbrev(text[:m[0]+1]) {
			continue
		}
		numeral := text[start:end]
		if numbers2rejectNumeral(text, numeral, end) {
			continue
		}
		v := types.Violation{
			RuleID:      rule,
			StartIndex:  start,
			EndIndex:    end,
			MatchedText: numeral,
			Explanation: "A sentence opens on a word, not on digits.",
		}
		if n, err := strconv.Atoi(numeral); err == nil {
			v.SuggestedChange = numbers2capitalize(numbers2cardinalWord(n))
		}
		out = append(out, v)
	}
	return out
}

func numbers2rejectNumeral(text, numeral string, end int) bool {
	if len(numeral) == 4 && !strings.Contains(numeral, ",") {
		if n, err := strconv.Atoi(numeral); err == nil && n >= 1000 && n <= 2999 {
			return true
		}
	}
	rest := text[end:]
	if rest == "" {
		return false
	}
	switch rest[0] {
	case '.', ')':
		return true
	case ':':
		return len(rest) > 1 && rest[1] >= '0' && rest[1] <= '9'
	case '_':
		return true
	}
	if numbers2isASCIILetter(rest[0]) {
		return true
	}
	trimmed := strings.TrimLeft(rest, " \t")
	return strings.HasPrefix(trimmed, "%")
}

func numbers2opensBlock(text string, idx int) bool {
	if idx == 0 {
		return true
	}
	lineStart := strings.LastIndexByte(text[:idx], '\n')
	if lineStart <= 0 {
		return true
	}
	prevStart := strings.LastIndexByte(text[:lineStart], '\n') + 1
	prev := strings.TrimRight(text[prevStart:lineStart], " \t\r")
	if prev == "" || isATXHeading(prev) {
		return true
	}
	prev = strings.TrimRight(prev, `"'”’)]`)
	if prev == "" {
		return false
	}
	switch prev[len(prev)-1] {
	case '.', '!', '?', ':':
		return true
	}
	return false
}

var (
	numbers2reOrdinalDigits = regexp.MustCompile(`(?i)\b(\d+)(?:st|nd|rd|th)\b`)
	numbers2reOrdinalSup    = regexp.MustCompile(`(?i)\b(\d+)<sup>(?:st|nd|rd|th)</sup>`)
)

// DetectGoogleSpelledOutOrdinals reports ordinals written as digits plus a
// suffix where prose wants the word.
func DetectGoogleSpelledOutOrdinals(text string) []types.Violation {
	const rule = "spelled-out-ordinals"
	var out []types.Violation
	for _, re := range []*regexp.Regexp{numbers2reOrdinalDigits, numbers2reOrdinalSup} {
		for _, m := range re.FindAllStringSubmatchIndex(text, -1) {
			if m[0] > 0 && (text[m[0]-1] == '$' || text[m[0]-1] == '#') {
				continue
			}
			v := types.Violation{
				RuleID:      rule,
				StartIndex:  m[0],
				EndIndex:    m[1],
				MatchedText: text[m[0]:m[1]],
				Explanation: "Prose spells ordinals as words.",
			}
			if n, err := strconv.Atoi(text[m[2]:m[3]]); err == nil {
				v.SuggestedChange = numbers2ordinalWord(n)
			}
			out = append(out, v)
		}
	}
	return out
}

const numbers2months = `January|February|March|April|May|June|July|August|September|October|November|December|Jan|Feb|Mar|Apr|Jun|Jul|Aug|Sep|Sept|Oct|Nov|Dec`

var (
	numbers2reMonthDayYear   = regexp.MustCompile(`(?i)\b(?:` + numbers2months + `)\b\.?[ \t]+\d{1,2},[ \t]*'?\d{2}\b`)
	numbers2reApostropheYear = regexp.MustCompile(`['\x{2019}]\d{2}\b`)
	numbers2reYearContext    = regexp.MustCompile(`(?i)(?:` + numbers2months + `)|\b(?:in|since|by)\b`)
	numbers2reDashedDate     = regexp.MustCompile(`\b\d{2}-\d{2}-\d{2}\b`)
)

const numbers2yearWindow = 15

// DetectGoogleTwoDigitYear reports a year abbreviated to its last two digits.
func DetectGoogleTwoDigitYear(text string) []types.Violation {
	const rule = "two-digit-year"
	var out []types.Violation
	emit := func(start, end int) {
		out = append(out, types.Violation{
			RuleID:      rule,
			StartIndex:  start,
			EndIndex:    end,
			MatchedText: text[start:end],
			Explanation: "A year carries all four digits.",
		})
	}
	for _, idx := range numbers2reMonthDayYear.FindAllStringIndex(text, -1) {
		emit(idx[0], idx[1])
	}
	for _, idx := range numbers2reApostropheYear.FindAllStringIndex(text, -1) {
		window := idx[0] - numbers2yearWindow
		if window < 0 {
			window = 0
		}
		if numbers2reYearContext.MatchString(text[window:idx[0]]) {
			emit(idx[0], idx[1])
		}
	}
	for _, idx := range numbers2reDashedDate.FindAllStringIndex(text, -1) {
		emit(idx[0], idx[1])
	}
	return out
}

var (
	numbers2reFusedUnit  = regexp.MustCompile(`\b\d+(?:\.\d+)?(?:KiB|MiB|GiB|TiB|GB|MB|KB|kB|TB|PB|Mbps|Gbps|kbps|bps|kHz|MHz|GHz|Hz|ms|ns|mm|cm|km|kg|mg|px|pt|dpi|kW|mA|bytes|byte|bits|bit)\b`)
	numbers2reUnitNumber = regexp.MustCompile(`^\d+(?:\.\d+)?`)
)

// DetectGoogleUnitSpaceMissing reports a quantity fused to its unit symbol.
func DetectGoogleUnitSpaceMissing(text string) []types.Violation {
	const rule = "unit-space-missing"
	var out []types.Violation
	for _, idx := range numbers2reFusedUnit.FindAllStringIndex(text, -1) {
		if numbers2fusedIsIdentifier(text, idx[0], idx[1]) {
			continue
		}
		match := text[idx[0]:idx[1]]
		number := numbers2reUnitNumber.FindString(match)
		out = append(out, types.Violation{
			RuleID:          rule,
			StartIndex:      idx[0],
			EndIndex:        idx[1],
			MatchedText:     match,
			Explanation:     "A number and its unit are separated by a space.",
			SuggestedChange: number + " " + match[len(number):],
		})
	}
	return out
}

func numbers2fusedIsIdentifier(text string, start, end int) bool {
	if start > 0 && strings.IndexByte("._-", text[start-1]) >= 0 {
		return true
	}
	if end >= len(text) {
		return false
	}
	if text[end] == '-' {
		return true
	}
	return text[end] == '.' && end+1 < len(text) && numbers2isASCIIAlnum(text[end+1])
}

var (
	numbers2reOpenFraction  = regexp.MustCompile(`(?i)\b((?:one|two|three|four|five|six|seven|eight|nine|ten)[ \t]+(?:halves|half|thirds?|fourths?|quarters?|fifths?|sixths?|sevenths?|eighths?|ninths?|tenths?))\s+of\b`)
	numbers2reFractionGlyph = regexp.MustCompile(`[\x{00BC}\x{00BD}\x{00BE}\x{2150}-\x{215E}]`)
	numbers2reInnerSpace    = regexp.MustCompile(`[ \t]+`)
)

var numbers2fractionDecimals = map[string]string{
	"¼": "0.25",
	"½": "0.5",
	"¾": "0.75",
	"⅒": "0.1",
	"⅕": "0.2",
	"⅖": "0.4",
	"⅗": "0.6",
	"⅘": "0.8",
	"⅛": "0.125",
	"⅜": "0.375",
	"⅝": "0.625",
	"⅞": "0.875",
}

// DetectGoogleWordFractionHyphen reports a fraction spelled as two open words
// or set with a precomposed fraction glyph.
func DetectGoogleWordFractionHyphen(text string) []types.Violation {
	const rule = "word-fraction-hyphen"
	var out []types.Violation
	for _, m := range numbers2reOpenFraction.FindAllStringSubmatchIndex(text, -1) {
		if m[2] > 0 && text[m[2]-1] == '-' {
			continue
		}
		match := text[m[2]:m[3]]
		out = append(out, types.Violation{
			RuleID:          rule,
			StartIndex:      m[2],
			EndIndex:        m[3],
			MatchedText:     match,
			Explanation:     "A spelled-out fraction is hyphenated.",
			SuggestedChange: numbers2reInnerSpace.ReplaceAllString(match, "-"),
		})
	}
	for _, idx := range numbers2reFractionGlyph.FindAllStringIndex(text, -1) {
		match := text[idx[0]:idx[1]]
		out = append(out, types.Violation{
			RuleID:          rule,
			StartIndex:      idx[0],
			EndIndex:        idx[1],
			MatchedText:     match,
			Explanation:     "A fraction glyph reads as one character; prefer the decimal.",
			SuggestedChange: numbers2fractionDecimals[match],
		})
	}
	return out
}

var (
	numbers2cardinalUnits = []string{"zero", "one", "two", "three", "four", "five", "six", "seven", "eight", "nine", "ten", "eleven", "twelve", "thirteen", "fourteen", "fifteen", "sixteen", "seventeen", "eighteen", "nineteen"}
	numbers2cardinalTens  = []string{"", "", "twenty", "thirty", "forty", "fifty", "sixty", "seventy", "eighty", "ninety"}
	numbers2ordinalUnits  = []string{"", "first", "second", "third", "fourth", "fifth", "sixth", "seventh", "eighth", "ninth", "tenth", "eleventh", "twelfth", "thirteenth", "fourteenth", "fifteenth", "sixteenth", "seventeenth", "eighteenth", "nineteenth"}
	numbers2ordinalTens   = []string{"", "", "twentieth", "thirtieth", "fortieth", "fiftieth", "sixtieth", "seventieth", "eightieth", "ninetieth"}
)

func numbers2cardinalWord(n int) string {
	switch {
	case n < 0 || n > 99:
		return ""
	case n < 20:
		return numbers2cardinalUnits[n]
	case n%10 == 0:
		return numbers2cardinalTens[n/10]
	default:
		return numbers2cardinalTens[n/10] + "-" + numbers2cardinalUnits[n%10]
	}
}

func numbers2ordinalWord(n int) string {
	switch {
	case n < 1 || n > 99:
		return ""
	case n < 20:
		return numbers2ordinalUnits[n]
	case n%10 == 0:
		return numbers2ordinalTens[n/10]
	default:
		return numbers2cardinalTens[n/10] + "-" + numbers2ordinalUnits[n%10]
	}
}

func numbers2capitalize(s string) string {
	if s == "" {
		return ""
	}
	return strings.ToUpper(s[:1]) + s[1:]
}

func numbers2isASCIILetter(b byte) bool {
	return (b >= 'a' && b <= 'z') || (b >= 'A' && b <= 'Z')
}

func numbers2isASCIIAlnum(b byte) bool {
	return numbers2isASCIILetter(b) || (b >= '0' && b <= '9')
}
