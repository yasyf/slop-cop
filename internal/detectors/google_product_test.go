package detectors

import "testing"

func TestDetectGoogleBrandNameAsVerb(t *testing.T) {
	runHits(t, "brand-name-as-verb", DetectGoogleBrandNameAsVerb, []hitCase{
		{"google the message id", "If the error is unfamiliar, google the message ID.", true},
		{"dockerize", "Dockerize the service before you deploy it.", true},
		{"googled", "We googled the stack trace and found the upstream issue.", true},
		{"photoshopped", "The screenshot in the report was photoshopped.", true},
		{"slack me", "Slack me the deploy link when the rollout finishes.", true},
		{"slacked me", "The on-call engineer slacked me the incident summary.", true},
		{"zoom them", "You can Zoom them once the meeting bridge is up.", true},
		{"search for", "If the error is unfamiliar, search for the message ID.", false},
		{"docker image", "Package the service as a Docker image before you deploy it.", false},
		{"google as noun", "Sign in with your Google account before you continue.", false},
		{"google cloud", "Deploy the container image to Google Cloud Run.", false},
		{"slack as noun", "Migrate the workspace to Slack and archive the old channels.", false},
		{"zoom as noun", "The Zoom meeting link expires after 24 hours.", false},
		{"ordinary prose", "Restart the collector and wait for the metrics to settle.", false},
		{"ordinary prose 2", "The client retries the request three times before it gives up.", false},
	})
}

func TestDetectGoogleBrandNameCasing(t *testing.T) {
	runHits(t, "brand-name-casing", DetectGoogleBrandNameCasing, []hitCase{
		{"github", "Clone the repository from Github.", true},
		{"javascript and nodejs", "The SDK is written in Javascript and runs on NodeJS.", true},
		{"macos", "Install the agent on MacOS 14 or later.", true},
		{"wifi", "Connect the device to a WiFi network first.", true},
		{"npm in prose", "Install NPM globally before you start.", true},
		{"oauth", "The endpoint accepts an Oauth bearer token.", true},
		{"github correct", "Clone the repository from GitHub.", false},
		{"javascript correct", "The SDK is written in JavaScript and runs on Node.js.", false},
		{"macos correct", "Install the agent on macOS 14 or later.", false},
		{"npm lowercase", "Run the test suite with npm test before you open a pull request.", false},
		{"all caps heading", "NPM PACKAGE LAYOUT", false},
		{"identifier is not a word", "Read the token from the OAUTH_CLIENT_SECRET variable.", false},
		{"ordinary prose", "The service authenticates with OAuth 2.0 and caches the token for an hour.", false},
		{"ordinary prose 2", "Each worker writes its results to Cloud Storage before it exits.", false},
	})
}

func TestDetectGoogleProductNameAbbreviation(t *testing.T) {
	runHits(t, "product-name-abbreviation", DetectGoogleProductNameAbbreviation, []hitCase{
		{"gcp", "Deploy the service to GCP.", true},
		{"gcs and bq", "Store the transcript in GCS and query it with BQ.", true},
		{"k8s", "The chart installs a k8s operator in the cluster.", true},
		{"postgres", "Run the migration against Postgres before you restart the API.", true},
		{"js and ts", "The SDK ships JS and TS type definitions.", true},
		{"google cloud", "Deploy the service to Google Cloud.", false},
		{"cloud storage", "Store the transcript in Cloud Storage and query it with BigQuery.", false},
		{"file extension", "Rename the handler to index.JS after the build.", false},
		{"all caps heading", "GCP GKE MIGRATION NOTES", false},
		{"mongodb spelled out", "The driver connects to MongoDB over TLS.", false},
		{"ordinary prose", "The dashboard renders in the browser and refreshes every 30 seconds.", false},
		{"ordinary prose 2", "Compile the TypeScript sources before you publish the package.", false},
	})
}

func TestDetectGoogleTrailingForExampleComma(t *testing.T) {
	runHits(t, "trailing-for-example-comma", DetectGoogleTrailingForExampleComma, []hitCase{
		{"instance name", "Enter a name for the instance, for example, my-instance-99.", true},
		{"algorithm", "Choose an encryption algorithm, for example, AES-256.", true},
		{"for instance", "Pick a region, for instance, us-central1.", true},
		{"parenthesized", "Enter a name for the instance (for example, my-instance-99).", false},
		{"such as", "Choose a strong encryption algorithm, such as AES-256.", false},
		{"parenthetical mid sentence", "Some drivers, for example, do not support batching.", false},
		{"clause with finite verb", "The flags, for example, are ignored when the cache is warm.", false},
		{"clause with inflected lead", "The team, for example, owns the ingest pipeline.", false},
		{"example plus trailing clause", "Rotate the key, for example, every 90 days, before the audit.", false},
		{"ordinary prose", "This applies to managed instances only.", false},
		{"ordinary prose 2", "The exporter batches spans and flushes them every five seconds.", false},
	})
}
