package detectors

import "testing"

func TestDetectGoogleFutureFeaturePromise(t *testing.T) {
	runHits(t, "future-feature-promise", DetectGoogleFutureFeaturePromise, []hitCase{
		{"plan to add", "Regex filters aren't supported yet; we plan to add them in an upcoming release.", true},
		{"coming soon", "Multi-region replication is coming soon.", true},
		{"eventually", "The CLI will eventually accept a config file.", true},
		{"in the future", "In the future, the daemon reads a config file.", true},
		{"on the roadmap", "Wildcard matching is on the roadmap.", true},
		{"does not yet", "The exporter does not yet support Parquet.", true},
		{"for now", "For now, pass the token on the command line.", true},
		{"stated as shipped", "Regex filters aren't supported.", false},
		{"plain capability", "The CLI accepts flags and environment variables.", false},
		{"soon after", "The VM goes offline soon after you send the shutdown command.", false},
		{"as soon as", "Send the request as soon as the token is issued.", false},
		{"ordinary reference prose", "The daemon listens on port 7000.", false},
		{"ordinary procedure", "Run the migration, then verify the health endpoint.", false},
		{"changelog opt-out", "# Changelog\n\nMulti-region replication is coming soon.\n", false},
		{"versioned headings opt-out", "## v1.2.0\n\nAdds globs.\n\n## v1.1.0\n\nFiltering is planned.\n", false},
	})
}

func TestDetectGoogleFutureTenseBehavior(t *testing.T) {
	runHits(t, "future-tense-behavior", DetectGoogleFutureTenseBehavior, []hitCase{
		{"will send", "Send a query to the service. The server will send an acknowledgment.", true},
		{"would disable", "Setting --retries=0 would disable the backoff loop.", true},
		{"won't retry", "The client won't retry if the flag is unset.", true},
		{"curly won't", "The client won’t retry if the flag is unset.", true},
		{"it'll", "It'll emit one record per row.", true},
		{"is going to", "The parser is going to reject the payload.", true},
		{"present tense", "Send a query to the service. The server sends an acknowledgment.", false},
		{"present conditional", "Setting --retries=0 disables the backoff loop.", false},
		{"present negative", "The client doesn't retry if the flag is unset.", false},
		{"genuine deferral", "Add the filename to the backup list. The file is archived the next time the backup runs.", false},
		{"date anchor", "The API will stop accepting requests on 2027-01-01.", false},
		{"deprecation notice", "The v1 endpoint will stop responding once it is deprecated.", false},
		{"question", "What will happen if the token expires?", false},
		{"block quote", "> The server will send an acknowledgment.", false},
		{"would like", "We would like to hear from you.", false},
		{"would be able", "An operator would be able to restart the pod.", false},
		{"ordinary reference prose", "The function returns an error for an empty input.", false},
		{"ordinary procedure", "Configure the retry budget, then deploy.", false},
	})
}

func TestDetectGoogleImpersonalRecommendation(t *testing.T) {
	runHits(t, "impersonal-recommendation", DetectGoogleImpersonalRecommendation, []hitCase{
		{"it is recommended", "It is recommended that you enable retries before going to production.", true},
		{"users are encouraged", "Users are encouraged to pin an exact version.", true},
		{"it's advisable", "It is advisable to set a deadline on every call.", true},
		{"one should", "One should never commit secrets to the repository.", true},
		{"best practice is to", "Best practice is to rotate keys monthly.", true},
		{"the recommended approach is", "The recommended approach is to shard by tenant.", true},
		{"owned recommendation", "We recommend that you enable retries before going to production.", false},
		{"owned gerund", "We recommend pinning an exact version.", false},
		{"requirement", "You must pin an exact version.", false},
		{"ordinary reference prose", "The client retries three times before failing.", false},
		{"ordinary procedure", "Set the timeout to 30 seconds.", false},
	})
}

func TestDetectGoogleTimeBoundQualifier(t *testing.T) {
	runHits(t, "time-bound-qualifier", DetectGoogleTimeBoundQualifier, []hitCase{
		{"currently", "The following command-line options aren't currently supported.", true},
		{"now supports", "The emulator now supports glob filters.", true},
		{"as of this writing", "As of this writing, the daemon listens on port 7000.", true},
		{"at present", "At present, the API returns JSON only.", true},
		{"is now", "The dashboard is now available in the console.", true},
		{"for the time being", "For the time being, the queue holds 1000 items.", true},
		{"timeless negative", "The following command-line options aren't supported.", false},
		{"timeless capability", "The emulator supports glob filters.", false},
		{"timeless reference", "The daemon listens on port 7000.", false},
		{"now that", "Now that the service is running, send a request.", false},
		{"from now on", "From now on, the token rotates every hour.", false},
		{"version anchor", "Version 2.1 currently ships with the beta flag.", false},
		{"ordinary reference prose", "The exporter writes one file per shard.", false},
		{"ordinary procedure", "Restart the daemon and check the log.", false},
		{"changelog opt-out", "# Release notes\n\nThe emulator now supports glob filters.\n", false},
	})
}
