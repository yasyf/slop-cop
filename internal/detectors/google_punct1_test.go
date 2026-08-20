package detectors

import "testing"

func TestDetectGoogleAcronymPeriods(t *testing.T) {
	runHits(t, "acronym-periods", DetectGoogleAcronymPeriods, []hitCase{
		{"usa", "The U.S.A. region shards separately.", true},
		{"api", "Configure the A.P.I. endpoint.", true},
		{"etc without period", "Read the retry budget, timeout, etc before filing a bug.", true},
		{"usa clean", "The USA region shards separately.", false},
		{"api clean", "Configure the API endpoint.", false},
		{"etc with period", "Read the retry budget, timeout, etc. before filing a bug.", false},
		{"plain prose", "The service retries the request after a short delay.", false},
		{"dotted filename", "Set config.yaml to enable debug logging.", false},
		{"spaced initials", "Written by J. R. R. Tolkien in 1937.", false},
		{"etcd", "Store the lease in etcd before restarting.", false},
	})
}

func TestDetectGoogleAndOrSlash(t *testing.T) {
	runHits(t, "and-or-slash", DetectGoogleAndOrSlash, []hitCase{
		{"view and edit", "You can view and/or edit your own data.", true},
		{"raw or processed", "Export raw and/or processed events.", true},
		{"spaced slash", "You can view and / or edit your own data.", true},
		{"plain and", "You can view and edit your own data.", false},
		{"spelled out", "Export raw events, processed events, or both.", false},
		{"ordinary conjunction", "The client retries and the server logs the failure.", false},
		{"path with or", "Set the mode to append or overwrite before you upload.", false},
	})
}

func TestDetectGoogleCompoundSpelling(t *testing.T) {
	runHits(t, "compound-spelling", DetectGoogleCompoundSpelling, []hitCase{
		{"datacenter", "Provision a replica in a second datacenter.", true},
		{"e-mail", "Store the e-mail address as a string.", true},
		{"code base", "Clone the code base and enable auto-scaling.", true},
		{"life-cycle", "The bucket has a life-cycle policy.", true},
		{"https case", "Requests over HTTPs are rejected.", true},
		{"read only noun", "Grant the account read only access to the bucket.", true},
		{"load balancing noun", "Attach a load balancing policy to the group.", true},
		{"rfc without space", "The header follows RFC2318 exactly.", true},
		{"bare id", "Set the Id field to a unique value.", true},
		{"data center", "Provision a replica in a second data center.", false},
		{"email", "Store the email address as a string.", false},
		{"codebase", "Clone the codebase and enable autoscaling.", false},
		{"lifecycle", "The bucket has a lifecycle policy.", false},
		{"https clean", "Requests over HTTPS are rejected.", false},
		{"read-only clean", "Grant the account read-only access to the bucket.", false},
		{"ordinary retry prose", "The scheduler retries failed jobs with exponential backoff.", false},
		{"ordinary console prose", "Open the console and select the project you want to configure.", false},
		{"in-line with", "Keep the change in-line with the existing convention.", false},
		{"go package path", "Join the segments with filepath.Join before opening the file.", false},
		{"matrices in math doc", "The eigenvector solver multiplies two matrices per step.", false},
		{"pd without disk", "The PD team reviewed the mock before the launch.", false},
		{"qualified rfc constant", "Format the value with time.RFC3339 before sending it.", false},
		{"hyphenated id header", "The Claude-Session-Id trailer is stripped from the subject.", false},
	})
}

func TestDetectGoogleConjunctiveAdverbPunctuation(t *testing.T) {
	runHits(t, "conjunctive-adverb-punctuation", DetectGoogleConjunctiveAdverbPunctuation, []hitCase{
		{"run on otherwise", "The field must have a value otherwise the server rejects the request.", true},
		{"comma splice therefore", "The cache is warm, therefore, the first read is fast.", true},
		{"missing trailing comma", "The field must have a value; otherwise the server rejects the request.", true},
		{"semicolon otherwise", "The field must have a value; otherwise, the server rejects the request.", false},
		{"semicolon therefore", "The cache is warm; therefore, the first read is fast.", false},
		{"interrupter however", "This value, however, is optional.", false},
		{"plain sentence", "The default timeout is 30 seconds.", false},
		{"semicolon without adverb", "Set the flag; the client then retries the request.", false},
		{"coordinating conjunction", "The API returns a 404, and the client retries the request.", false},
	})
}

func TestDetectGoogleEllipsisInProse(t *testing.T) {
	runHits(t, "ellipsis-in-prose", DetectGoogleEllipsisInProse, []hitCase{
		{"dramatic pause", "The answer is ... wait for it ... that you shouldn't do this.", true},
		{"trailing off", "The client handles retries, backoff, ...", true},
		{"stated plainly", "The answer is that you shouldn't do this.", false},
		{"listed plainly", "The client handles retries and backoff.", false},
		{"plain instruction", "Set the timeout to 30 seconds.", false},
		{"code span", "Run `git log --oneline ... --graph` to see the history.", false},
		{"quoted material", `The RFC says "the client retries ... within the window."`, false},
		{"fenced block", "Try this:\n\n```\nrun ... --all\n```\n", false},
		{"block quote", "> the client retries ... within the window", false},
	})
}

func TestDetectGoogleHeadingTerminalPeriod(t *testing.T) {
	runHits(t, "heading-terminal-period", DetectGoogleHeadingTerminalPeriod, []hitCase{
		{"atx h2", "## Configure the retention policy.", true},
		{"atx h3", "### What the scheduler does.", true},
		{"setext", "Configure the retention policy.\n------------------------------\n", true},
		{"setext after blank line", "Intro paragraph.\n\nConfigure the retention policy.\n====\n", true},
		{"html heading", "<h2>Configure the retention policy.</h2>", true},
		{"atx clean", "## Configure the retention policy", false},
		{"atx question", "## Why does the upload retry?", false},
		{"body prose", "Configure the retention policy. Then save your changes.", false},
		{"single token filename", "## config.yaml.", false},
		{"heading with ellipsis", "## Loading the retention policy...", false},
		{"abbreviation", "## Retention, replication, etc.", false},
		{"shell comment in fence", "```bash\n# 2. PATH fallback (CI, scripting, or an install).\n```\n", false},
		{"thematic break after paragraph", "Detectors run over the whole file.\nThe mask stays unchanged.\n---\n", false},
	})
}

func TestDetectGoogleIntroCommaMissing(t *testing.T) {
	runHits(t, "intro-comma-missing", DetectGoogleIntroCommaMissing, []hitCase{
		{"however opener", "However the retry budget still applies to streaming calls.", true},
		{"fronted if clause", "If the token expires before the upload finishes the client silently drops the request.", true},
		{"however with comma", "However, the retry budget still applies to streaming calls.", false},
		{"fronted if with comma", "If the token expires before the upload finishes, the client silently drops the request.", false},
		{"no opener", "The retry budget applies to streaming calls.", false},
		{"instead of phrase", "Instead of the default, use a custom retry policy.", false},
		{"in addition to phrase", "In addition to the retry budget, the client applies a deadline.", false},
		{"short fronted clause", "To learn more, see the reference.", false},
		{"fronted when with comma", "When the job completes, the console shows the final status.", false},
		{"imperative sentence", "Configure the retry budget before you deploy the service to production.", false},
	})
}

func TestDetectGoogleLongParenthetical(t *testing.T) {
	runHits(t, "long-parenthetical", DetectGoogleLongParenthetical, []hitCase{
		{"overlong example", "Enter a hex color (for instance, if you want a deep forest green rather than the default teal, enter 228B22 here), and then click OK.", true},
		{"terse example", "Enter a hex color (for example, 228B22), and then click OK.", false},
		{"promoted to sentence", "Enter a hex color, and then click OK. For a deep forest green, enter 228B22.", false},
		{"short aside", "Set the deadline (in seconds) before you send the request.", false},
		{"cross reference", "See the reference (for more information about retry budgets, see the retry guide in the operations handbook).", false},
		{"code span aside", "Set the deadline (`--deadline`) before you send the request.", false},
		{"plain prose", "The client retries the request twice before it gives up.", false},
	})
}

func TestDetectGoogleLyAdverbHyphen(t *testing.T) {
	runHits(t, "ly-adverb-hyphen", DetectGoogleLyAdverbHyphen, []hitCase{
		{"publicly", "publicly-available implementations", true},
		{"fully", "a fully-managed service", true},
		{"highly", "a highly-available cluster", true},
		{"publicly clean", "publicly available implementations", false},
		{"fully clean", "a fully managed service", false},
		{"read-only", "The bucket is read-only for the service account.", false},
		{"supply-chain", "The supply-chain attack surface is small.", false},
		{"family-owned", "Use a family-owned mirror for the packages.", false},
		{"plain prose", "The service scales automatically when load increases.", false},
	})
}
