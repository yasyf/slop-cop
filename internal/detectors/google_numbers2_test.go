package detectors

import "testing"

func TestDetectGooglePercentFormat(t *testing.T) {
	runHits(t, "percent-format", DetectGooglePercentFormat, []hitCase{
		{"space before sign", "The cache absorbs 40 % of reads.", true},
		{"sentence opens on a percentage", "40% of the files were skipped.", true},
		{"spelled after a numeral", "Roughly 40 percent of reads hit the cache.", true},
		{"trailing space before sign", "Coverage rose to 98 %.", true},
		{"closed sign", "The cache absorbs 40% of reads.", false},
		{"spelled out at sentence start", "Forty percent of the files were skipped.", false},
		{"plain duration", "The build finished in 40 seconds.", false},
		{"percentile is not a percentage", "The 95th percentile latency is 40ms.", false},
	})
}

func TestDetectGoogleRateSlash(t *testing.T) {
	runHits(t, "rate-slash", DetectGoogleRateSlash, []hitCase{
		{"requests per day", "requests/day", true},
		{"queries per second", "10,000 queries/second", true},
		{"bandwidth", "2.5 Gb/s", true},
		{"abbreviated unit", "It sustains 5,000 writes/s under load.", true},
		{"spelled rate", "requests per day", false},
		{"spelled rate with count", "10,000 queries per second", false},
		{"settled abbreviation", "2.5 Gbps", false},
		{"url path", "The endpoint is /api/v1/users/1234.", false},
		{"repo path", "The handler lives in internal/detectors/helpers.go.", false},
		{"table row keeps the slash", "| queries/second | 10,000 |", false},
		{"paired nouns", "Serve the input/output pair.", false},
	})
}

func TestDetectGoogleSentenceInitialNumeral(t *testing.T) {
	runHits(t, "sentence-initial-numeral", DetectGoogleSentenceInitialNumeral, []hitCase{
		{"opening quantity", "512 MB of memory is the default for this runtime.", true},
		{"opening count", "15 directories are created during setup.", true},
		{"after a terminator", "The default is small. 15 directories are created.", true},
		{"after a quoted terminator", "He said \"done.\" 15 files remained.", true},
		{"spelled out", "Fifteen directories are created during setup.", false},
		{"recast sentence", "By default, the runtime gets 512 MB of memory.", false},
		{"four-digit year", "2017 was the first year of the project.", false},
		{"ordered list marker", "1. Install the CLI.", false},
		{"wrapped line", "The runtime allocates\n512 MB of memory.", false},
		{"percentage is another rule", "40% of the files were skipped.", false},
		{"version string", "1.2.3 is the current release.", false},
		{"after an abbreviation", "Use a lock file, e.g. 15 entries are pinned there.", false},
		{"table row", "| 15 | directories |", false},
		{"clock time", "The cutoff is 12:30 UTC.", false},
		{"list item", "- 512 MB is the default.", false},
	})
}

func TestDetectGoogleSpelledOutOrdinals(t *testing.T) {
	runHits(t, "spelled-out-ordinals", DetectGoogleSpelledOutOrdinals, []hitCase{
		{"third", "Retry the 3rd request.", true},
		{"twenty-first", "The 21st field is reserved.", true},
		{"twelfth", "The 12th column holds the checksum.", true},
		{"superscript source form", "The 3<sup>rd</sup> attempt failed.", true},
		{"spelled third", "Retry the third request.", false},
		{"spelled twenty-first", "The twenty-first field is reserved.", false},
		{"cardinal duration", "Set the timeout to 30 seconds.", false},
		{"version number", "Version 3 of the API is stable.", false},
		{"issue reference", "See issue #3rd for context.", false},
	})
}

func TestDetectGoogleTwoDigitYear(t *testing.T) {
	runHits(t, "two-digit-year", DetectGoogleTwoDigitYear, []hitCase{
		{"abbreviated month year", "Shipped in Jan '17.", true},
		{"two-digit year after a day", "The cutoff is January 19, 17.", true},
		{"all-numeric date", "The log rotates on 01-19-17.", true},
		{"after by", "We ship by '25 at the latest.", true},
		{"four-digit year", "Shipped in January 2017.", false},
		{"full date", "The cutoff is January 19, 2017.", false},
		{"version number", "The release notes cover version 1.17 and later.", false},
		{"quoted number", "The array holds '17' as a string.", false},
		{"phone number", "Call 212-555-1234 for support.", false},
		{"quoted flag value", "The flag defaults to 'on' for new projects.", false},
	})
}

func TestDetectGoogleUnitSpaceMissing(t *testing.T) {
	runHits(t, "unit-space-missing", DetectGoogleUnitSpaceMissing, []hitCase{
		{"capacity", "Attach a 64GB disk.", true},
		{"duration", "Timeout after 500ms.", true},
		{"length", "25mm clearance", true},
		{"binary capacity", "Each 4KiB page is mapped once.", true},
		{"bandwidth", "A 10Gbps uplink is provisioned.", true},
		{"separated capacity", "Attach a 64&nbsp;GB disk.", false},
		{"separated duration", "Timeout after 500&nbsp;ms.", false},
		{"resolution shorthand", "The image is 1080p at 60 Hz.", false},
		{"architecture identifiers", "Build for x86_64 with h264 support.", false},
		{"version strings", "The container image is version 2.4.1 and 3.10.2.", false},
		{"bare count", "Set the retry budget to 3 attempts.", false},
		{"thousands suffix", "The token budget is 200k characters.", false},
	})
}

func TestDetectGoogleWordFractionHyphen(t *testing.T) {
	runHits(t, "word-fraction-hyphen", DetectGoogleWordFractionHyphen, []hitCase{
		{"open two thirds", "two thirds of the shards", true},
		{"open one half", "Reserve one half of the quota.", true},
		{"fraction glyph", "Reserve ½ of the quota.", true},
		{"hyphenated", "two-thirds of the shards", false},
		{"decimal", "Reserve 0.5 of the quota.", false},
		{"quarter as a period", "The migration ran in one quarter the time.", false},
		{"fiscal quarter", "We shipped it in the first quarter of the year.", false},
		{"plural count", "Split the work into four quarters.", false},
		{"hyphenated compound", "Only one third-party dependency remains.", false},
	})
}
