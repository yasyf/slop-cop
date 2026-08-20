package detectors

import "testing"

func TestDetectGoogleOneOrMoreAgreement(t *testing.T) {
	runHits(t, "one-or-more-agreement", DetectGoogleOneOrMoreAgreement, []hitCase{
		{"singular after one or more", "If one or more test fail, a warning is triggered.", true},
		{"plural after more than one", "You can create more than one instances at a time.", true},
		{"plural after one or more", "If one or more tests fail, a warning is triggered.", false},
		{"singular after more than one", "You can create more than one instance at a time.", false},
		{"one or more of", "Choose one or more of the following options.", false},
		{"plural head noun", "Provide one or more tags for the resource.", false},
		{"irregular plural", "Send one or more media files.", false},
		{"singular verb takes the subject", "If more than one matches the filter, the request fails.", false},
		{"always-s noun", "You can specify more than one address for the record.", false},
		{"wrapped plural", "The pipeline reads one or more\nconfiguration files at startup.", false},
		{"ordinary prose", "The cache absorbs most reads before they reach the disk.", false},
		{"ordinary prose with counts", "Each request carries two headers and one body.", false},
	})
}

func TestDetectGoogleOverExplainedAbbreviation(t *testing.T) {
	runHits(t, "over-explained-abbreviation", DetectGoogleOverExplainedAbbreviation, []hitCase{
		{"expansion then abbreviation", "Export a portable document format (PDF) report.", true},
		{"api expansion", "Call the application programming interface (API) endpoint.", true},
		{"abbreviation then expansion", "The API (application programming interface) is versioned.", true},
		{"hyphenated expansion", "The node has 8 GB of random-access memory (RAM).", true},
		{"bare abbreviation", "Export a PDF report.", false},
		{"bare api", "Call the API endpoint.", false},
		{"version in parentheses", "The API (v2) endpoint accepts JSON.", false},
		{"abbreviation worth expanding", "Configure the network time protocol (NTP) offset.", false},
		{"ordinary prose", "The report renders as a PDF and an HTML page.", false},
		{"ordinary prose with aside", "The endpoint returns JSON (never XML) for every request.", false},
	})
}

func TestDetectGooglePerOutsideRates(t *testing.T) {
	runHits(t, "per-outside-rates", DetectGooglePerOutsideRates, []hitCase{
		{"as per your request", "As per your request, the quota was raised.", true},
		{"per a proper noun", "Create one policy per Pod.", true},
		{"per the style guide", "Per the style guide, use sentence case.", true},
		{"reworded request", "In response to your request, the quota was raised.", false},
		{"for each", "Create one policy for each Pod.", false},
		{"rate", "The API handles 500 requests per second.", false},
		{"unit rate", "Storage costs two cents per GB each month.", false},
		{"rate with a count", "The quota allows 20 GB per Pod.", false},
		{"ordinary prose", "Each Pod gets its own service account.", false},
		{"ordinary prose with a rate", "Billing is computed per month and charged in arrears.", false},
	})
}

func TestDetectGooglePiedPiping(t *testing.T) {
	runHits(t, "pied-piping", DetectGooglePiedPiping, []hitCase{
		{"with which", "See the client library documentation for the language with which you're interacting.", true},
		{"in which", "Select the project in which the bucket resides.", true},
		{"trailing preposition", "See the client library documentation for the language you're interacting with.", false},
		{"trailing preposition on a clause", "Select the project the bucket resides in.", false},
		{"in which case", "The token may expire, in which case you refresh it.", false},
		{"the order in which", "The order in which the rules run matters.", false},
		{"both of which", "Two flags are set, both of which are optional.", false},
		{"embedded question", "Every caller decides about which of the two modes it wants.", false},
		{"embedded question with an infinitive", "The team debated about which storage class to use.", false},
		{"ordinary prose", "The bucket that the project owns is regional.", false},
		{"ordinary prose with a relative clause", "The handler that parses the payload runs first.", false},
	})
}

func TestDetectGooglePluralizedUnitAbbreviation(t *testing.T) {
	runHits(t, "pluralized-unit-abbreviation", DetectGooglePluralizedUnitAbbreviation, []hitCase{
		{"gigabytes", "The instance has 64 GBs of memory.", true},
		{"terabytes", "Allocate 2 TBs of disk.", true},
		{"clock speed", "The core runs at 3.2 GHzs.", true},
		{"singular symbol", "The instance has 64 GB of memory.", false},
		{"singular terabyte", "Allocate 2 TB of disk.", false},
		{"milliseconds", "The job finished in 300 ms.", false},
		{"decade", "Latency budgets tightened in the 1990s.", false},
		{"ordinary prose", "The disk holds 500 GB and the cache holds 2 GB.", false},
		{"spelled-out unit", "The instance has 64 gigabytes of memory.", false},
	})
}

func TestDetectGoogleSibilantAbbreviationPlural(t *testing.T) {
	runHits(t, "sibilant-abbreviation-plural", DetectGoogleSibilantAbbreviationPlural, []hitCase{
		{"operating systems", "Both OSs ship the driver.", true},
		{"messages", "Two SMSs were delivered.", true},
		{"name servers", "The zone lists three DNSs.", true},
		{"already es", "Both OSes ship the driver.", false},
		{"already es on sms", "Two SMSes were delivered.", false},
		{"non-sibilant abbreviations", "The FAQs cover both APIs.", false},
		{"more non-sibilant abbreviations", "Three VMs share one URL.", false},
		{"ordinary prose", "The IDs and SDKs are listed in the appendix.", false},
		{"bucket name with a digit", "The S3 bucket holds the archive.", false},
	})
}

func TestDetectGoogleSingleUseAbbreviation(t *testing.T) {
	runHits(t, "single-use-abbreviation", DetectGoogleSingleUseAbbreviation, []hitCase{
		{"two unused abbreviations", "The internet of things (IoT) service connects to sensors in low Earth orbit (LEO).", true},
		{"defined once", "Configure the network time protocol (NTP) offset before you continue.", true},
		{"reused abbreviation", "The internet of things (IoT) service connects to sensors. IoT devices authenticate with a device certificate, and the IoT registry stores it.", false},
		{"no abbreviation", "Configure the network time protocol offset before you continue.", false},
		{"plain aside", "The service exports a report (see the appendix for details).", false},
		{"parenthetical that is not an introduction", "Files are written to disk (usually within a second).", false},
		{"markdown link target", "See the [contributor guide](CONTRIBUTING) before you open a pull request.", false},
		{"ordinary prose", "The scheduler retries the request twice before it gives up.", false},
	})
}

func TestDetectGoogleSpelledOutShorthand(t *testing.T) {
	runHits(t, "spelled-out-shorthand", DetectGoogleSpelledOutShorthand, []hitCase{
		{"multiplier", "Throughput is now 10x faster.", true},
		{"without", "Deploy w/o the sidecar.", true},
		{"approx", "Approx. 200 requests per second.", true},
		{"with", "Deploy w/ the sidecar.", true},
		{"because", "The retry is skipped b/c the token expired.", true},
		{"spelled multiplier", "Throughput is now 10 times faster.", false},
		{"spelled without", "Deploy without the sidecar.", false},
		{"spelled approximately", "Approximately 200 requests per second.", false},
		{"pixel dimensions", "The image is 1080x1920 pixels.", false},
		{"repo path", "The handler lives in internal/detectors/helpers.go.", false},
		{"email address", "Write to support@example.com for help.", false},
		{"ordinary prose", "The build finished in three minutes without a cache hit.", false},
	})
}

func TestDetectGoogleUiDeviceNaming(t *testing.T) {
	runHits(t, "ui-device-naming", DetectGoogleUIDeviceNaming, []hitCase{
		{"cell phone", "Install the app on your cell phone.", true},
		{"omnibox", "Type the URL into the omnibox.", true},
		{"account name", "Sign in with your account name.", true},
		{"type your", "Type your project ID.", true},
		{"type a quoted string", "Type \"yes\" to confirm.", true},
		{"developer key", "Use the developer key from the console.", true},
		{"mobile phone", "Install the app on your mobile phone.", false},
		{"address bar", "Type the URL into the address bar.", false},
		{"username", "Sign in with your username.", false},
		{"enter", "Enter your project ID.", false},
		{"storage account name", "Set the storage account name in the config.", false},
		{"data type", "The data type of the field is a string.", false},
		{"machine type", "Set the machine type and type the value you want.", false},
		{"ordinary prose", "Open the app bar and choose a project.", false},
	})
}
