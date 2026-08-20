package detectors

import "testing"

func TestDetectGoogleDoubleNegative(t *testing.T) {
	runHits(t, "double-negative", DetectGoogleDoubleNegative, []hitCase{
		{"prevent-you-from", "A missing path won't prevent you from continuing.", true},
		{"isnt-uncommon", "It isn't uncommon for the job to fail without a retry.", true},
		{"dont-forget-to-not", "Don't forget to not disable the audit log.", true},
		{"not-uncommon", "It is not uncommon for the job to retry.", true},
		{"not-run-without", "The agent does not start without a service account.", true},
		{"not-unless-adjacent", "Do not delete the bucket unless you have a backup.", true},
		{"good-continue", "You can continue without a path.", false},
		{"good-often-fails", "The job often fails and then retries.", false},
		{"good-keep-enabled", "Keep the audit log enabled.", false},
		{"ordinary-single-negation", "The service does not restart automatically after a crash.", false},
		{"ordinary-no-negation", "Set the timeout to 30 seconds before you run the migration.", false},
		{"ordinary-clause-separated", "If you cannot connect, check that the port is open.", false},
		{"ordinary-negations-far-apart", "The client cannot retry when the server does not respond.", false},
		{"ordinary-negations-across-clauses", "Never commit a token, and do not paste one into an issue.", false},
		{"ordinary-code-span", "Pass `--no-cache` when the build is not reproducible.", false},
	})
}

func TestDetectGoogleIdiomColloquialism(t *testing.T) {
	runHits(t, "idiom-colloquialism", DetectGoogleIdiomColloquialism, []hitCase{
		{"ballpark-figure", "This gives you a ballpark figure for the monthly cost.", true},
		{"under-the-hood", "Under the hood, the scheduler batches by priority.", true},
		{"silver-bullet", "Caching is not a silver bullet for latency.", true},
		{"rule-of-thumb", "As a rule of thumb, keep payloads under 1 MB.", true},
		{"red-tape", "The approval process adds red tape.", true},
		{"good-estimate", "This gives you an estimate of the monthly cost.", false},
		{"good-internally", "Internally, the scheduler batches by priority.", false},
		{"ordinary-scheduler", "The scheduler batches requests by priority.", false},
		{"ordinary-estimate-cost", "Estimate the monthly cost before you deploy.", false},
	})
}

func TestDetectGoogleMultiwordForSingleWord(t *testing.T) {
	runHits(t, "multiword-for-single-word", DetectGoogleMultiwordForSingleWord, []hitCase{
		{"number-of-ability-regular", "A number of clients have the ability to retry on a regular basis.", true},
		{"exception-timely", "With the exception of the first run, the job completes in a timely manner.", true},
		{"is-able-to", "The worker is able to resume after a restart.", true},
		{"majority-of", "The majority of requests finish in under a second.", true},
		{"good-some-clients", "Some clients can retry regularly.", false},
		{"good-except-for", "Except for the first run, the job completes quickly.", false},
		{"ordinary-most-clients", "Most clients retry after a timeout.", false},
		{"ordinary-three-regions", "The service supports three regions and one global endpoint.", false},
	})
}

func TestDetectGoogleNoteAsCrossReference(t *testing.T) {
	runHits(t, "note-as-cross-reference", DetectGoogleNoteAsCrossReference, []hitCase{
		{"note-for-more-information", "**Note:** For more information, see [Firewall rules](/vpc/firewall).", true},
		{"note-see-also", "**Note:** See also [Regions](/regions).", true},
		{"good-plain-cross-reference", "For more information, see [Firewall rules](/vpc/firewall).", false},
		{"ordinary-note-without-link", "**Note:** The bucket name must be globally unique.", false},
		{"ordinary-note-two-sentences", "**Note:** Buckets are regional. For more information, see [Regions](/regions).", false},
		{"ordinary-note-with-substance", "**Note:** Quotas apply per project, as [Quotas](/quotas) explains in detail.", false},
	})
}

func TestDetectGoogleNoteCarriesRequiredInfo(t *testing.T) {
	runHits(t, "note-carries-required-info", DetectGoogleNoteCarriesRequiredInfo, []hitCase{
		{"note-before-you-begin", "**Note:** Before you begin, enable the Compute Engine API.", true},
		{"note-run-command", "**Note:** Run `terraform init` before applying the plan.", true},
		{"note-you-must", "**Note:** You must grant the service account the Editor role.", true},
		{"good-prose-then-steps", "Before you begin, enable the Compute Engine API.\n\n1. Run `terraform init`.\n2. Apply the plan.", false},
		{"ordinary-note-describes-behavior", "**Note:** The API returns a 429 when you exceed the quota.", false},
		{"ordinary-warning-is-not-a-step-carrier", "**Warning:** Run `terraform destroy` to remove everything.", false},
	})
}

func TestDetectGoogleNoticePileup(t *testing.T) {
	runHits(t, "notice-pileup", DetectGoogleNoticePileup, []hitCase{
		{"adjacent-notices", "**Note:** Buckets are regional.\n\n**Caution:** Deleting a bucket is permanent.", true},
		{"four-notices", "**Note:** One.\n\nProse.\n\n**Tip:** Two.\n\nProse.\n\n**Caution:** Three.\n\nProse.\n\n**Warning:** Four.", true},
		{"good-single-notice", "Buckets are regional, so pick the region closest to your workload.\n\n**Warning:** Deleting a bucket permanently destroys its contents.", false},
		{"ordinary-two-separated-notices", "**Note:** Buckets are regional.\n\nPick the region closest to your workload so that reads stay fast.\n\n**Warning:** Deleting a bucket destroys its contents.", false},
		{"ordinary-no-notices", "Buckets are regional. Pick the region closest to your workload.", false},
	})
}

func TestDetectGoogleNoticeSeverityMismatch(t *testing.T) {
	runHits(t, "notice-severity-mismatch", DetectGoogleNoticeSeverityMismatch, []hitCase{
		{"note-permanently-destroys", "**Note:** Deleting the cluster permanently destroys its persistent volumes.", true},
		{"caution-security-risk", "**Caution:** Putting the password on the command line is a security risk.", true},
		{"note-cannot-be-undone", "**Note:** The migration cannot be undone.", true},
		{"good-warning-permanent", "**Warning:** Deleting the cluster permanently destroys its persistent volumes.", false},
		{"good-warning-exposes", "**Warning:** Putting the password on the command line exposes it to the process list.", false},
		{"ordinary-note-benign", "**Note:** The cache refreshes every five minutes.", false},
		{"ordinary-prose-permanent", "Deleting a bucket is permanent, so copy the objects first.", false},
	})
}

func TestDetectGooglePlainWordSwap(t *testing.T) {
	runHits(t, "plain-word-swap", DetectGooglePlainWordSwap, []hitCase{
		{"commences-subsequently-utilizes", "The scheduler commences the job and subsequently utilizes the remaining quota.", true},
		{"prior-to-ascertain", "Prior to deployment, ascertain that the service account exists.", true},
		{"via", "Deploy the service via Terraform.", true},
		{"in-order-to", "Run the migration script in order to rebuild the index.", true},
		{"allows-you-to", "The console allows you to filter by label.", true},
		{"ie", "Use the primary region, i.e. the one closest to your users.", true},
		{"versus", "Compare gRPC vs. REST for streaming workloads.", true},
		{"good-starts-then-uses", "The scheduler starts the job and then uses the remaining quota.", false},
		{"good-before-you-deploy", "Before you deploy, check that the service account exists.", false},
		{"good-by-using", "Deploy the service by using Terraform.", false},
		{"good-plain-infinitive", "Run the migration script to rebuild the index.", false},
		{"guard-financial-leverage", "The debt leverage ratio stays constant across quarters.", false},
		{"guard-utilization", "CPU utilization stayed under 40 percent.", false},
		{"ordinary-set-retry-limit", "Set the retry limit to five and restart the worker.", false},
		{"ordinary-json-response", "The API returns a JSON object that lists every region.", false},
	})
}

func TestDetectGoogleSuccessNoticeInStaticDoc(t *testing.T) {
	runHits(t, "success-notice-in-static-doc", DetectGoogleSuccessNoticeInStaticDoc, []hitCase{
		{"success-deployed", "**Success:** You've successfully deployed an application to GKE.", true},
		{"success-blockquote", "> **Success:** Your cluster is now running.", true},
		{"good-prose-outcome", "The deployment is complete when the service reports a Ready status.", false},
		{"templated-success", "{{#if deployed}}\n**Success:** You deployed the app.\n{{/if}}", false},
		{"ordinary-note", "**Note:** The deployment succeeded once the pods report Ready.", false},
		{"ordinary-prose", "Check the console to confirm that the deployment succeeded.", false},
	})
}
