package detectors

import "testing"

func TestDetectGoogleUndefinedAbbreviation(t *testing.T) {
	runHits(t, "undefined-abbreviation", DetectGoogleUndefinedAbbreviation, []hitCase{
		{"bare protocol acronym", "Establish BGP sessions using a router on the peer network.", true},
		{"bare product acronym", "Attach the VPC SC perimeter before you enable the connector.", true},
		{"heading with nothing below it", "## Configure the MIG\n\nThe MIG scales automatically based on load.", true},
		{"tooling acronym", "The MCP server exposes the same query surface.", true},
		{"markup acronym", "The analyzer masks JSX attributes before the detectors run.", true},
		{"plural acronym", "Agents do not want TUIs, they want a structured report.", true},
		{"version control acronym", "The VCS lane picks Graphite when the config is live.", true},
		{"encoding acronym", "The classifier picks TOON when the shape is tabular.", true},
		{"nested encoding acronym", "Deeply nested payloads encode as TRON instead.", true},
		{"line format acronym", "Write the records as JSONL so each line parses alone.", true},
		{"protocol acronym", "Reach for the LSP when the answer must be exhaustive.", true},
		{"binary format acronym", "The blob ships as WASM on release tags only.", true},
		{"two letter acronym", "The CC field carries the second reviewer.", true},
		{"index acronym", "The AA index ranked the models on the same harness.", true},
		{"expanded at first use", "Establish Border Gateway Protocol (BGP) sessions using a router on the peer network. Restart BGP after the change.", false},
		{"product expanded at first use", "Attach the VPC Service Controls (VPC SC) perimeter before you enable the connector.", false},
		{"heading expanded in the paragraph below", "## Configure the MIG\n\nA managed instance group (MIG) scales automatically based on load.", false},
		{"expansion follows the abbreviation", "The MIG (managed instance group) scales automatically based on load.", false},
		{"universally known abbreviations", "The API returns JSON over HTTPS, and the CLI writes it to a CSV file.", false},
		{"code span", "Set the `BGP` field on the router resource.", false},
		{"bare caps filename", "Read NOTICE before you reuse any of the ported content.", false},
		{"caps filename with an extension", "The shared rules live in AGENTS.md and travel with the repository.", false},
		{"caps filename as link text", "See [NOTICE](NOTICE) for the provenance boundary.", false},
		{"agent doc filenames", "CLAUDE defers to AGENTS, and the active SKILL file repeats neither.", false},
		{"plural caps filename", "The READMEs in each package repeat the same install line.", false},
		{"license file", "The Go source is MIT; see LICENSE for the full text.", false},
		{"caps emphasis on modals", "A plan without it is NOT complete and MUST be revised.", false},
		{"caps emphasis on verbs", "The guard BLOCKS the command and ALWAYS reports why.", false},
		{"caps emphasis on adverbs", "The orchestrator NEVER runs the sweep inline.", false},
		{"caps emphasis on quantifiers", "Dispatch ALL that are ready, ANY that unblock, NONE that wait.", false},
		{"caps admonitions", "WARNING: the cache is not invalidated. NOTE the ordering.", false},
		{"ordinary prose", "Run the command from the repository root and read the output.", false},
		{"no abbreviations at all", "Deploy the service, then confirm that the health check passes.", false},
	})
}

func TestDetectGoogleUnderspecifiedTerm(t *testing.T) {
	runHits(t, "underspecified-term", DetectGoogleUnderspecifiedTerm, []hitCase{
		{"mime type", "Set the Content-Type header to the correct MIME type.", true},
		{"bare mobile", "The SDK also runs on mobile.", true},
		{"cellular data", "Turn off cellular data before you sync.", true},
		{"ingest as a verb", "Ingest the CSV file into the table.", true},
		{"epoch time", "Convert the value to epoch time.", true},
		{"key features", "This page lists the key features of the API.", true},
		{"interconnect type", "Choose the interconnect type that matches your peering.", true},
		{"media type", "Set the Content-Type header to the correct media type.", false},
		{"mobile devices", "The SDK also runs on mobile devices.", false},
		{"mobile data", "Turn off mobile data before you sync.", false},
		{"plain verb", "Copy the CSV file into the table.", false},
		{"unix epoch time", "Convert the value to Unix epoch time.", false},
		{"mobile as a modifier", "The mobile SDK ships a separate binary.", false},
		{"ingestion is a settled noun", "The data ingestion pipeline runs nightly.", false},
		{"ordinary prose", "Run the command from the repository root and read the output.", false},
		{"encryption key", "Rotate the encryption key before the certificate expires.", false},
	})
}

func TestDetectGoogleUnitNumberAgreement(t *testing.T) {
	runHits(t, "unit-number-agreement", DetectGoogleUnitNumberAgreement, []hitCase{
		{"one with a plural unit", "Wait 1 seconds before retrying.", true},
		{"zero with a singular unit", "The threshold is 0 degree.", true},
		{"count with a singular unit", "Provision 3 node.", true},
		{"decimal with a singular unit", "The delay is 0.5 second.", true},
		{"one with a singular unit", "Wait 1 second before retrying.", false},
		{"zero with a plural unit", "The threshold is 0 degrees.", false},
		{"count with a plural unit", "Provision 3 nodes.", false},
		{"one point zero takes the plural", "Wait 1.0 seconds before retrying.", false},
		{"compound modifier", "Deploy a 3 node cluster in each region.", false},
		{"unrelated nouns", "The build produced 12 artifacts across 4 targets.", false},
		{"ordinary prose", "Run the command from the repository root and read the output.", false},
	})
}

func TestDetectGoogleVerbNounCompound(t *testing.T) {
	runHits(t, "verb-noun-compound", DetectGoogleVerbNounCompound, []hitCase{
		{"closed form as a verb", "To setup the cluster, run the script.", true},
		{"open form as a noun", "Complete the set up before you deploy.", true},
		{"log in", "Log in with your credentials.", true},
		{"sign into", "Sign into the console.", true},
		{"closed form after a modal", "The request will timeout after 30 seconds.", true},
		{"bare third-party noun", "The connector talks to a third-party.", true},
		{"open form as a verb", "To set up the cluster, run the script.", false},
		{"closed form as a noun", "Complete the setup before you deploy.", false},
		{"sign in", "Sign in with your credentials.", false},
		{"sign in to", "Sign in to the console.", false},
		{"log as a noun", "Check the log in the audit bucket.", false},
		{"logging as a subject", "Structured logging in production is enabled by default.", false},
		{"third-party as a modifier", "The connector talks to a third-party service.", false},
		{"ordinary prose", "Run the command from the repository root and read the output.", false},
		{"timeout as a noun", "The service exposes a timeout setting in the configuration file.", false},
	})
}

func TestDetectGoogleVersionRangeComparator(t *testing.T) {
	runHits(t, "version-range-comparator", DetectGoogleVersionRangeComparator, []hitCase{
		{"version and above", "This flag works on version 2.4 and above.", true},
		{"or higher", "Use Python 3.9 or higher.", true},
		{"or lower", "Use version 2.2 or lower.", true},
		{"trailing plus", "Requires Node 18.0+ for the new runtime.", true},
		{"higher on this page", "See the table higher on this page for the full list.", true},
		{"under an explicit version", "The field is ignored under version 2.4.", true},
		{"version and later", "This flag works on version 2.4 and later.", false},
		{"or later", "Use Python 3.9 or later.", false},
		{"or earlier", "Use version 2.2 or earlier.", false},
		{"count threshold", "Set the retry budget to 3 or higher.", false},
		{"version in a tag name", "The tag is v1.2.3 and the branch is main.", false},
		{"size in prose", "Download the 1.5 GB archive and extract it.", false},
		{"ordinary prose", "Run the command from the repository root and read the output.", false},
	})
}
