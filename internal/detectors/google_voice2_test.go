package detectors

import "testing"

func TestDetectGoogleThatWhichClause(t *testing.T) {
	runHits(t, "that-which-clause", DetectGoogleThatWhichClause, []hitCase{
		{"comma before that", "The echidna, that has a long snout, is furry.", true},
		{"restrictive which", "Delete the bucket which holds the stale objects.", true},
		{"restrictive which defaults", "Set the retention field which defaults to 30 days.", true},
		{"which contains", "Open the file which contains the service account key.", true},
		{"hard-wrapped antecedent", "Set the retention\nfield which defaults to 30 days.", true},
		{"comma before that plus verb", "The default retention, that applies to every bucket, is 30 days.", true},
		{"restrictive that", "The echidna that has a long snout is furry.", false},
		{"restrictive that bucket", "Delete the bucket that holds the stale objects.", false},
		{"nonrestrictive which", "Set the retention field, which defaults to 30 days.", false},
		{"preposition before which", "It depends on which region you deploy to.", false},
		{"interrogative verb", "Determine which flag the job reads.", false},
		{"no matter which", "No matter which flag you set, the job retries.", false},
		{"which one", "Inspect the two buckets and pick which one holds the key.", false},
		{"of which", "The exporter writes three files, all of which are gzipped.", false},
		{"parenthetical that is", "The bearer token, that is, the Authorization value, must be set.", false},
		{"demonstrative that", "If the request fails, that error is logged to stderr.", false},
		{"demonstrative that process", "When the queue drains, that process exits.", false},
		{"interrogative which plus noun", "Ask the user which file to read.", false},
		{"interrogative which after adverb", "Say precisely which prerequisite is outstanding.", false},
		{"imperative antecedent", "Filter which requests get spans.", false},
		{"adverb antecedent", "Use manual mode when you know exactly which operations are measured.", false},
		{"list item imperative", "- Customize which errors are captured.", false},
		{"embedded question object", "Tell the reviewer which claims are high-stakes.", false},
		{"demonstrative subject predicate", "If the user takes the offer, that is a new request, not a bug.", false},
		{"comma splice demonstrative", "Do not re-run ship, that cuts a new commit.", false},
		{"coordinated that complements", "Check that the build is current, that the tag is signed.", false},
		{"ordinary reference prose", "The daemon listens on port 7000 and writes a pid file.", false},
		{"ordinary procedure", "Run the migration, then verify the health endpoint.", false},
	})
}

func TestDetectGoogleTrivializingDifficulty(t *testing.T) {
	runHits(t, "trivializing-difficulty", DetectGoogleTrivializingDifficulty, []hitCase{
		{"simply", "Simply set the flag and redeploy.", true},
		{"easily", "You can easily add a second region later.", true},
		{"all you have to do", "All you have to do is regenerate the token.", true},
		{"just skips", "BigQuery just skips the row.", true},
		{"just run", "Just run the installer and restart the daemon.", true},
		{"it's that simple", "Restart the pod. It's that simple.", true},
		{"of course", "Of course, you must set the token before the first call.", true},
		{"straightforward", "The migration is straightforward.", true},
		{"sentence adverb clearly", "Clearly, you must set the token before the first call.", true},
		{"no adverb", "Set the flag and redeploy.", false},
		{"manner adverb clearly", "Label the replay sections clearly so reviewers can read them.", false},
		{"no easily", "You can add a second region later.", false},
		{"bare imperative", "Regenerate the token.", false},
		{"no just", "BigQuery skips the row.", false},
		{"quick reference", "See the quick reference for the full flag list.", false},
		{"quickstart", "The quickstart walks through the first deploy.", false},
		{"proper noun", "Amazon Simple Storage Service stores the exported objects.", false},
		{"more than just", "The adapter is more than just a thin wrapper.", false},
		{"or just", "Set the flag, or just delete the file.", false},
		{"comparative just as", "The base rules fire on human prose just as readily.", false},
		{"comparative just like", "The wrapper behaves just like the CLI.", false},
		{"ordinary reference prose", "The daemon listens on port 7000 and writes a pid file.", false},
		{"ordinary procedure", "Run the migration, then verify the health endpoint.", false},
	})
}
