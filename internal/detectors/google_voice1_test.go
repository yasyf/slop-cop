package detectors

import "testing"

func TestDetectGoogleAnthropomorphism(t *testing.T) {
	runHits(t, "anthropomorphism", DetectGoogleAnthropomorphism, []hitCase{
		{"object tells", "The delimiter object tells the splitter where the string should break.", true},
		{"PC sees", "The PC sees a new device.", true},
		{"compiler doesn't know", "The compiler doesn't know about the flag.", true},
		{"service wants", "The service wants a bearer token.", true},
		{"object specifies", "The delimiter object specifies where to split the string.", false},
		{"PC detects", "The PC detects a new device.", false},
		{"compiler has no definition", "The compiler has no definition for the flag.", false},
		{"model learns is ML jargon", "The model learns from the training set.", false},
		{"plain behavior", "The service returns a 404 when the record is missing.", false},
		{"plain mechanism", "The parser reads the header before the body.", false},
	})
}

func TestDetectGoogleExclamationPoint(t *testing.T) {
	runHits(t, "exclamation-point", DetectGoogleExclamationPoint, []hitCase{
		{"build finished", "The build finished in under a second!", true},
		{"deployment live", "Your deployment is live!", true},
		{"period instead", "The build finished in under a second.", false},
		{"period instead again", "Your deployment is live.", false},
		{"inequality operator", "Use the != operator to compare the two values.", false},
		{"negation prefix", "Set the flag to !enabled before running the job.", false},
		{"shebang", "#!/bin/sh is the shebang line.", false},
		{"ordinary step", "Run the command and check the exit code.", false},
	})
}

func TestDetectGoogleInternetSlang(t *testing.T) {
	runHits(t, "internet-slang", DetectGoogleInternetSlang, []hitCase{
		{"tldr", "tl;dr: use the batch endpoint.", true},
		{"fwiw", "The retry budget is per host, fwiw.", true},
		{"imo", "IMO the second approach scales better.", true},
		{"summary heading", "Summary: use the batch endpoint.", false},
		{"plain statement", "The retry budget applies per host.", false},
		{"direct claim", "The second approach scales better.", false},
		{"ordinary reference", "The API returns a list of resources.", false},
		{"ordinary instruction", "Configure the retry budget per host.", false},
	})
}

func TestDetectGoogleNonstandardContraction(t *testing.T) {
	runHits(t, "nonstandard-contraction", DetectGoogleNonstandardContraction, []hitCase{
		{"invented re", "The guides're updated every release.", true},
		{"noun plus is", "The browser's not required for this step.", true},
		{"there would", "There'd be no retry if the token expires first.", true},
		{"stacked", "You mightn't've noticed the change.", true},
		{"spelled out", "The guides are updated every release.", false},
		{"common contraction", "The browser isn't required for this step.", false},
		{"there would spelled out", "There would be no retry if the token expires first.", false},
		{"common negation", "The service doesn't restart automatically.", false},
		{"you are", "You're able to configure the timeout.", false},
		{"possessive", "The client's configuration file lives in /etc.", false},
		{"it is", "It's the default value.", false},
		{"hyphenated compound", "The server's not-found handler returns 404.", false},
		{"name possessive", "O'Reilly's documentation covers the topic.", false},
	})
}

func TestDetectGooglePassiveByAgent(t *testing.T) {
	runHits(t, "passive-by-agent", DetectGooglePassiveByAgent, []hitCase{
		{"signed by library", "The request is signed by the client library before it is sent.", true},
		{"read by daemon", "Configuration values are read by the daemon at startup.", true},
		{"written by person", "The module was written by Jane Doe.", true},
		{"active library", "The client library signs the request before sending it.", false},
		{"active daemon", "The daemon reads configuration values at startup.", false},
		{"agentless passive", "Requests are retried automatically after a timeout.", false},
		{"criterion by", "Sort the results by name.", false},
		{"keyed by criterion", "Records are keyed by their primary identifier.", false},
		{"by default idiom", "The file is compressed by default.", false},
		{"by the time idiom", "The cache is refreshed by the time the request lands.", false},
		{"by a factor idiom", "The latency improved by a factor of three.", false},
	})
}

func TestDetectGooglePlaceholderPhrase(t *testing.T) {
	runHits(t, "placeholder-phrase", DetectGooglePlaceholderPhrase, []hitCase{
		{"please note that", "Please note that the endpoint rejects requests larger than 4 MiB.", true},
		{"at this time", "At this time, the connector supports one region.", true},
		{"note that mid paragraph", "The limit is 4 MiB. Note that the header counts toward it.", true},
		{"fact first", "The endpoint rejects requests larger than 4 MiB.", false},
		{"scope stated", "The connector supports one region.", false},
		{"temporal not an opener", "The job runs at this time each day.", false},
		{"take note of", "Take note of the exit code before retrying.", false},
	})
}

func TestDetectGooglePleaseInInstructions(t *testing.T) {
	runHits(t, "please-in-instructions", DetectGooglePleaseInInstructions, []hitCase{
		{"please click", "Please click Deploy to start the rollout.", true},
		{"please note", "Please note that the endpoint rejects requests larger than 4 MiB.", true},
		{"numbered step", "1. Please install the CLI.", true},
		{"bare imperative", "Click Deploy to start the rollout.", false},
		{"fact first", "The endpoint rejects requests larger than 4 MiB.", false},
		{"bare numbered step", "1. Install the CLI.", false},
		{"genuine imposition", "If the problem persists, please contact support.", false},
		{"ordinary step", "Run the migration before restarting the service.", false},
		{"ordinary statement", "The dialog asks for confirmation.", false},
	})
}

func TestDetectGoogleReaderAddressPerson(t *testing.T) {
	runHits(t, "reader-address-person", DetectGoogleReaderAddressPerson, []hitCase{
		{"shows the user how", "This guide shows the user how to deploy an app to their organization.", true},
		{"lets look", "Now let's look at what the response contains.", true},
		{"we install our project", "We install the CLI and then configure our project.", true},
		{"the user must", "The user must accept the terms before continuing.", true},
		{"the users possessive", "The user's browser stores the token.", true},
		{"shows you how", "This guide shows you how to deploy an app to your organization.", false},
		{"no joint framing", "The response contains the following fields.", false},
		{"bare imperative", "Install the CLI, and then configure your project.", false},
		{"reader's own end user", "Your app asks the user for consent before reading their contacts.", false},
		{"organizational we", "We recommend using the batch endpoint.", false},
		{"lets is not let us", "The flag lets you disable caching.", false},
		{"guarded possessive", "Your app stores the user's token on the device.", false},
	})
}

func TestDetectGoogleSuperlativeProductClaim(t *testing.T) {
	runHits(t, "superlative-product-claim", DetectGoogleSuperlativeProductClaim, []hitCase{
		{"fastest way", "This is the fastest way to load data into the warehouse.", true},
		{"always available", "The replica set is always available.", true},
		{"best way", "This is the best way to migrate the schema.", true},
		{"never fails", "The queue never fails under load.", true},
		{"scoped comparison", "For bulk loads over 1 GiB, this path was faster in our benchmark; see the performance comparison.", false},
		{"designed for availability", "The replica set is designed for high availability; see the SLA for the guaranteed uptime.", false},
		{"best practices", "Follow these best practices when configuring the cluster.", false},
		{"worst case", "In the worst case, the retry budget is exhausted.", false},
		{"up to date", "Keep the dependency list always up to date.", false},
		{"best effort", "The service uses a best-effort delivery model.", false},
	})
}
