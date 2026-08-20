package detectors

import "testing"

func TestDetectGoogleAllCapsEmphasis(t *testing.T) {
	runHits(t, "all-caps-emphasis", DetectGoogleAllCapsEmphasis, []hitCase{
		{"shouted negation", "Do NOT delete the volume before the snapshot finishes.", true},
		{"shouted only", "This is the ONLY supported upgrade path.", true},
		{"alt text", "alt=\"WARNING ICON\"", true},
		{"contraction instead", "Don't delete the volume before the snapshot finishes.", false},
		{"lowercase only", "This is the only supported upgrade path.", false},
		{"sentence-case alt text", "alt=\"Warning icon.\"", false},
		{"abbreviations", "The API returns JSON over HTTPS and caches the ETag.", false},
		{"more abbreviations", "Set the TTL to 300 seconds in the DNS record.", false},
		{"notice label", "NOTE: back up the volume first.", false},
		{"bold notice label", "> **WARNING:** the rollout is not atomic.", false},
		{"code span", "Add `NOT NULL` to the column definition.", false},
		{"angle placeholder", "Replace <NONE> with a region name.", false},
		{"env var shape", "Export NOT_READY=1 before the run.", false},
		{"shouted obligation", "Before you touch the file, you MUST:", true},
		{"shouted prohibition", "Do NOT bulk-read the cited files yourself.", true},
		{"alert warning", "> [!WARNING]\n> The rollout is not atomic.", false},
		{"alert note", "> [!NOTE]\n> Back up the volume first.", false},
		{"alert tip", "> [!TIP]\n> Reuse the cached build.", false},
		{"alert important", "> [!IMPORTANT]\n> Rotate the key before the deadline.", false},
		{"alert caution", "> [!CAUTION]\n> This deletes the volume.", false},
		{"nested alert", "- > [!NOTE]\n  > Back up the volume first.", false},
		{"bare filename", "Consult NOTICE before broadening use.", false},
		{"filename as link text", "See [NOTICE](NOTICE) for provenance.", false},
		{"filename with extension", "Provenance lives in NOTICE.md.", false},
		{"other doc filenames", "The repo carries LICENSE, README, AGENTS, CLAUDE, and CONTRIBUTING.", false},
		{"image alt text", "![WARNING SIGN](sign.png)", true},
	})
}

func TestDetectGoogleAmpersandConjunction(t *testing.T) {
	runHits(t, "ampersand-conjunction", DetectGoogleAmpersandConjunction, []hitCase{
		{"sentence conjunction", "Build & deploy the service.", true},
		{"heading conjunction", "## Quotas & limits", true},
		{"line-leading ampersand", "The job runs nightly.\n& it retries on failure.", true},
		{"spelled out", "Build and deploy the service.", false},
		{"spelled out heading", "## Quotas and limits", false},
		{"ui label", "Click **File & Print**.", false},
		{"query string", "The page uses ?a=1&b=2 for pagination.", false},
		{"shell operator", "Run `make build && make test` locally.", false},
		{"company name", "Contact AT&T for the circuit ID.", false},
		{"trailing background operator", "Start it with nohup server &\n", false},
		{"html entity", "Escape it as a &amp; b in the template.", false},
		{"ordinary prose", "The scheduler retries the request after a short delay.", false},
	})
}

func TestDetectGoogleHyphenatedWordCapitalization(t *testing.T) {
	runHits(t, "hyphenated-word-capitalization", DetectGoogleHyphenatedWordCapitalization, []hitCase{
		{"sentence start", "Re-Run the migration after the schema lands.", true},
		{"heading", "## Re-Indexing the catalog", true},
		{"second sentence", "The schema lands first. Re-Run the migration afterwards.", true},
		{"lowercase second element", "Re-run the migration after the schema lands.", false},
		{"proper adjective", "Non-English locales load a separate bundle.", false},
		{"mid-sentence compound", "The service uses a well-known port for TLS.", false},
		{"lowercase compound", "Configure the load-balancer timeout before you deploy.", false},
		{"title-cased proper name", "Cross-Origin Resource Sharing must be enabled.", false},
		{"trademark compound", "Wi-Fi credentials are stored in the keychain.", false},
		{"code span", "Pass `Re-Run` as the header value.\n", false},
	})
}

func TestDetectGoogleListItemSentenceCase(t *testing.T) {
	runHits(t, "list-item-sentence-case", DetectGoogleListItemSentenceCase, []hitCase{
		{"list item", "- Restart The Build Server", true},
		{"table row", "| Field Name | Default Value |", true},
		{"caption", "Table 2. Supported Regions And Their Endpoints", true},
		{"html list item", "<li>Restart The Build Server</li>", true},
		{"sentence-case list item", "- Restart the build server", false},
		{"sentence-case table row", "| Field name | Default value |", false},
		{"sentence-case caption", "Table 2. Supported regions and their endpoints", false},
		{"body prose", "The build server restarts after each deploy finishes.", false},
		{"single-word cells", "| Name | Type | Default | Required |", false},
		{"separator row", "| --- | --- |", false},
		{"product name", "- Enable the Compute Engine API for the project", false},
		{"ui label in bold", "1. Click **Save As** and choose a folder", false},
		{"sentence-case with clause", "- Restart the build server, then check the logs", false},
	})
}

func TestDetectGoogleTermIntroductionFormatting(t *testing.T) {
	runHits(t, "term-introduction-formatting", DetectGoogleTermIntroductionFormatting, []hitCase{
		{"bold definition", "A **shard** is a horizontal slice of the table.", true},
		{"quoted term", "The term \"idempotent\" means the call can be repeated safely.", true},
		{"known as with bold", "The pattern is known as **fan-out** in the docs.", true},
		{"italic definition", "A _shard_ is a horizontal slice of the table.", false},
		{"italic term", "The term _idempotent_ means the call can be repeated safely.", false},
		{"ordinary called", "The scheduler is called every five minutes.", false},
		{"ordinary definition", "A retry policy is applied to every request.", false},
		{"identifier in code font", "A `shard` is a horizontal slice of the table.", false},
	})
}

func TestDetectGoogleUnderlineNonLink(t *testing.T) {
	runHits(t, "underline-non-link", DetectGoogleUnderlineNonLink, []hitCase{
		{"underline element", "<u>Important</u>: back up the volume first.", true},
		{"css on a paragraph", "p { text-decoration: underline; }", true},
		{"bare insert element", "<ins>Added in v2.</ins>", true},
		{"italics instead", "_Important_: back up the volume first.", false},
		{"ordinary prose", "Underline the risks in your incident report.", false},
		{"decoration removed", "The link uses text-decoration: none in the stylesheet.", false},
		{"anchor selector", "a:hover { text-decoration: underline; }", false},
		{"anchor inline style", "See the <a href=\"x\" style=\"text-decoration: underline\">link</a>.", false},
		{"unordered list", "<ul><li>one</li></ul>", false},
		{"change tracking", "<ins datetime=\"2020-01-01\">new</ins>", false},
		{"code span", "Wrap it in `<u>` only for legacy mail clients.", false},
	})
}
