package detectors

import "testing"

func TestDetectGoogleQuotePeriodPlacement(t *testing.T) {
	runHits(t, "quote-period-placement", DetectGoogleQuotePeriodPlacement, []hitCase{
		{"period outside", `The commit message should read "Fixed typo".`, true},
		{"period after question", `New users always ask "Why?".`, true},
		{"period after title", `See the section titled "Care and feeding of the emu".`, true},
		{"period inside", `The commit message should read "Fixed typo."`, false},
		{"question mark alone", `New users always ask "Why?"`, false},
		{"literal keeps period outside", `Set the mode to "read-only".`, false},
		{"literal flag", `Set the flag to "--dry-run".`, false},
		{"no quotes", "The build failed because the config file was missing.", false},
		{"code span", "Use the `--verbose` flag to see more output.", false},
	})
}

func TestDetectGoogleQuotesAroundCodeSpan(t *testing.T) {
	runHits(t, "quotes-around-code-span", DetectGoogleQuotesAroundCodeSpan, []hitCase{
		{"quoted flag", "Set the flag to \"`--verbose`\".", true},
		{"quoted method", "Call the \"`get`\" method.", true},
		{"quoted parameter name", `Set the "maxRetries" parameter.`, true},
		{"bare code span", "Set the flag to `--verbose`.", false},
		{"bare method span", "Call the `get` method.", false},
		{"quotes inside code", "The constant `city` has the value `\"San Francisco\"`.", false},
		{"plain prose", "The parser reads the file and returns a syntax tree.", false},
		{"quoted phrase", `Quote the value "San Francisco" in the report.`, false},
	})
}

func TestDetectGoogleRangeWordDashMix(t *testing.T) {
	runHits(t, "range-word-dash-mix", DetectGoogleRangeWordDashMix, []hitCase{
		{"from with dash", "Upload from 8-20 files.", true},
		{"between with dash", "Expect between 5-10 minutes.", true},
		{"from with to", "Upload from 8 to 20 files.", false},
		{"between with and", "Expect between 5 and 10 minutes.", false},
		{"bare dash range", "Expect 5-10 minutes.", false},
		{"iso date", "The report covers data from 2024-01-15 onward.", false},
		{"hyphenated word", "Read from read-only storage.", false},
	})
}

func TestDetectGoogleSerialComma(t *testing.T) {
	runHits(t, "serial-comma", DetectGoogleSerialComma, []hitCase{
		{"verb series", "The client caches tokens, retries failed calls and logs every response.", true},
		{"noun series", "Supported formats are JSON, YAML and XML.", true},
		{"semicolon series", "Focus on what users need most; what is cheapest to fix and what fits the time you have.", true},
		{"verb series fixed", "The client caches tokens, retries failed calls, and logs every response.", false},
		{"noun series fixed", "Supported formats are JSON, YAML, and XML.", false},
		{"semicolon series fixed", "Focus on what users need most; what is cheapest to fix; and what fits the time you have.", false},
		{"two clauses", "The parser reads the file and returns a syntax tree.", false},
		{"introductory clause", "When the build completes, the tests run and the report uploads.", false},
		{"semicolon joins clauses", "The build failed; the logs show a timeout and a retry.", false},
	})
}

func TestDetectGoogleSingleQuotesInProse(t *testing.T) {
	runHits(t, "single-quotes-in-prose", DetectGoogleSingleQuotesInProse, []hitCase{
		{"single quoted term", "This forms an 'island' inside the network.", true},
		{"single quoted sentence", `She said, 'I heard him shout "Help".'`, true},
		{"double quoted term", `This forms an "island" inside the network.`, false},
		{"nested single quotes", `She said, "I heard him shout 'Help.'"`, false},
		{"contraction and possessive", "Don't change the user's password.", false},
		{"plural possessive", "The tests' output lands in the logs directory.", false},
	})
}

func TestDetectGoogleSpacedHyphen(t *testing.T) {
	runHits(t, "spaced-hyphen", DetectGoogleSpacedHyphen, []hitCase{
		{"parenthetical dashes", "The retry limit - the number of attempts - is configurable.", true},
		{"spaced compound", "a well - designed app", true},
		{"trailing space after hyphen", "scan at one- hour intervals", true},
		{"parentheses instead", "The retry limit (the number of attempts) is configurable.", false},
		{"closed compound", "a well-designed app", false},
		{"suspended hyphens", "scan at one-, two-, or three-hour intervals", false},
		{"long flag", "Use the --verbose flag to see more output.", false},
		{"hyphenated modifier", "The build-time flag controls the output.", false},
		{"numeric span", "Scores of 30 - 5 appeared in the log.", false},
	})
}

func TestDetectGoogleUiLabelEllipsis(t *testing.T) {
	runHits(t, "ui-label-ellipsis", DetectGoogleUILabelEllipsis, []hitCase{
		{"bold label with dots", "Click **Save...**.", true},
		{"plain label with ellipsis", "Select Export… from the menu.", true},
		{"bold label", "Click **Save**.", false},
		{"bold label in sentence", "Select **Export** from the menu.", false},
		{"plain prose", "The installer copies the files and then reboots.", false},
		{"trailing ellipsis in prose", "Open the log file and wait for the build to finish...", false},
	})
}

func TestDetectGoogleUrlTerminalPeriod(t *testing.T) {
	runHits(t, "url-terminal-period", DetectGoogleURLTerminalPeriod, []hitCase{
		{"url ends sentence", "Read the retention policy at http://example.com/policy/.", true},
		{"url mid sentence", "The retention policy at http://example.com/policy/ explains the defaults.", false},
		{"markdown link", "Read the [retention policy](http://example.com/policy/).", false},
		{"no url", "Run the installer and then restart the service.", false},
		{"url followed by prose", "See the docs at https://example.com/guide for details.", false},
	})
}
