package detectors

import "testing"

func TestDetectGoogleAdjacentNumerals(t *testing.T) {
	runHits(t, "adjacent-numerals", DetectGoogleAdjacentNumerals, []hitCase{
		{"byte files", "The job writes 15 100,000-byte files.", true},
		{"mb volumes", "Allocate 3 512 MB volumes.", true},
		{"spelled out", "The job writes fifteen 100,000-byte files.", false},
		{"separated by words", "The job writes 15 of the 100,000-byte files.", false},
		{"status code", "The API returns 200 OK for every request.", false},
		{"two independent values", "Set the timeout to 30 seconds and the retry limit to 5.", false},
		{"single unit phrase", "The file is 512 MB.", false},
		{"comma separated", "Copy 10, 20-byte chunks.", false},
	})
}

func TestDetectGoogleAmPmFormat(t *testing.T) {
	runHits(t, "am-pm-format", DetectGoogleAmPmFormat, []hitCase{
		{"no space lowercase", "3pm", true},
		{"periods", "3 p.m.", true},
		{"no space uppercase", "3:45PM", true},
		{"hour only", "3 PM", false},
		{"hour and minutes", "3:45 PM", false},
		{"meeting time", "The meeting starts at 9 AM in room 4.", false},
		{"ambient sensors", "The cluster has 5 ambient sensors.", false},
		{"version number", "Version 2 is faster than the previous build.", false},
	})
}

func TestDetectGoogleAmbiguousNumericDate(t *testing.T) {
	runHits(t, "ambiguous-numeric-date", DetectGoogleAmbiguousNumericDate, []hitCase{
		{"dotted", "Support ended 02.12.2017.", true},
		{"slashed", "The change landed 12/02/2017.", true},
		{"two digit year", "The migration window opens on 03/04/26.", true},
		{"month spelled out", "Support ended February 12, 2017.", false},
		{"iso order", "The change landed 2017-04-15.", false},
		{"month spelled out again", "The migration window opens on March 4, 2026.", false},
		{"version string", "Upgrade to version 1.2.2020 before the deadline.", false},
		{"semantic version", "Python 3.9.20 is supported.", false},
		{"ratio", "The ratio is 3/4 of the total.", false},
		{"ipv4 address", "The server listens on 192.168.10.10.", false},
	})
}

func TestDetectGoogleCurrencyFormat(t *testing.T) {
	runHits(t, "currency-format", DetectGoogleCurrencyFormat, []hitCase{
		{"space thousands", "$10 000 in fees", true},
		{"separator after decimal", "$0.006,653 per vCPU hour", true},
		{"trailing symbol", "The plan costs 10$.", true},
		{"comma thousands", "$10,000 in fees", false},
		{"unpunctuated decimal", "$0.006653 per vCPU hour", false},
		{"leading symbol", "The plan costs $10.", false},
		{"shell variable", "Set the 3 $PATH entries.", false},
		{"price with cents", "The tier costs $1,299.99 per month.", false},
		{"price range", "Prices range from $5 to $10 per user.", false},
	})
}

func TestDetectGoogleDateAbbreviationConsistency(t *testing.T) {
	runHits(t, "date-abbreviation-consistency", DetectGoogleDateAbbreviationConsistency, []hitCase{
		{"short day long month", "Mon, September 3, 2018", true},
		{"trailing period", "Sept. 3, 2018", true},
		{"long day short month", "Tuesday, Apr 27, 2021", true},
		{"all short", "Mon, Sep 3, 2018", false},
		{"all long", "Tuesday, April 27, 2021", false},
		{"may has no abbreviation", "The release shipped on May 4, 2020.", false},
		{"weekdays without a date", "We deploy every Monday, and the report runs on Friday.", false},
	})
}

func TestDetectGoogleKSuffixThousands(t *testing.T) {
	runHits(t, "k-suffix-thousands", DetectGoogleKSuffixThousands, []hitCase{
		{"detached suffix", "limited to 55 k download operations", true},
		{"no noun", "You are limited to 55k per day.", true},
		{"suffix with noun", "limited to 55k download operations and 20k upload operations per day", false},
		{"noun follows", "The bucket holds 12k objects.", false},
		{"kilobyte unit", "Set the limit to 3 kB per request.", false},
		{"hyphenated k", "Compare 5 k-means clusters.", false},
	})
}

func TestDetectGoogleLeadingZeroDecimal(t *testing.T) {
	runHits(t, "leading-zero-decimal", DetectGoogleLeadingZeroDecimal, []hitCase{
		{"bare tolerance", "The tolerance is .3 inches.", true},
		{"bare duration", "Sleep for .5 seconds.", true},
		{"leading zero tolerance", "The tolerance is 0.3 inches.", false},
		{"leading zero duration", "Sleep for 0.5 seconds.", false},
		{"file extension", "Download the .7z archive.", false},
		{"section number", "See section 2.5 for details.", false},
		{"package name", "Use Node.js 18 in production.", false},
	})
}

func TestDetectGoogleMonthYearComma(t *testing.T) {
	runHits(t, "month-year-comma", DetectGoogleMonthYearComma, []hitCase{
		{"january", "She was hired in January, 2017.", true},
		{"october", "The API shipped in October, 2021.", true},
		{"january without comma", "She was hired in January 2017.", false},
		{"october without comma", "The API shipped in October 2021.", false},
		{"clause comma before a count", "In March, 4000 units shipped.", false},
		{"month and year in a phrase", "The release notes for January 2017 are online.", false},
	})
}

func TestDetectGoogleNumeralUnderTen(t *testing.T) {
	runHits(t, "numeral-under-ten", DetectGoogleNumeralUnderTen, []hitCase{
		{"menu options", "The menu has 4 options.", true},
		{"wait minutes", "Wait 5 minutes before retrying.", true},
		{"menu options spelled out", "The menu has four options.", false},
		{"wait minutes spelled out", "Wait five minutes before retrying.", false},
		{"technical rate", "6 queries per second", false},
		{"larger number in the sentence", "The menu contains 15 options but 6 of them are deselected.", false},
		{"product version", "Python 3 introduced type hints.", false},
		{"unit of memory", "The disk holds 4 GB of data.", false},
		{"step reference", "See step 3 for details.", false},
		{"ordered list marker", "1. Install the package", false},
		{"percentage", "The error rate stayed under 5%.", false},
		{"protocol version", "HTTP 2 multiplexes streams.", false},
		{"numeric range", "Pick 2–4 concrete options.", false},
		{"table row", "| fable-5 | 2 | 9 | 9 | Orchestration |", false},
	})
}
