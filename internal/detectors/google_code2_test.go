package detectors

import "testing"

func TestDetectGoogleUnformattedCodeIdentifier(t *testing.T) {
	runHits(t, "unformatted-code-identifier", DetectGoogleUnformattedCodeIdentifier, []hitCase{
		{"conf file and path", "Open the pg_hba.conf file, which is typically in /etc/postgresql/13/main.", true},
		{"call and filename", "Call getUserToken() before you read config.yaml.", true},
		{"long flag", "Pass --source-file to the import command.", true},
		{"status code", "The server answers with a 429 status code.", true},
		{"http verb with request", "Send a GET request to the users endpoint.", true},
		{"screaming snake constant", "Set DATABASE_URL before you start.", true},
		{"pascal identifier", "The UserAccount record holds the billing address.", true},
		{"bare symbol in prose", "Reach for findReferences when the answer must be exhaustive.", true},
		{"bare tool name in prose", "Spawn the WebFetch agent with the URL and your prompt.", true},
		{"bare source path in prose", "The detector prompt lives in internal/llm/prompts.go today.", true},
		{"symbol beside a link", "See [tropes.md](https://tropes.fyi/tropes-md) before you call findReferences.", true},
		{"conf file and path in code font", "Open the `pg_hba.conf` file, which is in the `/etc/postgresql/13/main` directory.", false},
		{"call and filename in code font", "Call `getUserToken()` before you read `config.yaml`.", false},
		{"long flag in code font", "Pass the `--source-file` flag to the import command.", false},
		{"status code in code font", "The server answers with a `429` status code.", false},
		{"fenced block", "Run this:\n\n```\ngetUserToken()\n```\n", false},
		{"heading", "# Call getUserToken\n\nRead the section that follows.\n", false},
		{"url", "See https://example.com/api_v2/getUser for details.", false},
		{"inline link text", "See [LLM_PROSE_TELLS.md](https://example.com/prompts/LLM_PROSE_TELLS.md) for the list.", false},
		{"inline link text after destination masking", "See [LLM_PROSE_TELLS.md]                                        for the list.", false},
		{"reference link text", "See [tropes.md][tropes] for the list.", false},
		{"source path as link text", "Read [internal/llm/prompts.go](https://example.com/blob/main/internal/llm/prompts.go).", false},
		{"source path as link destination", "Read [the detector prompt](https://example.com/blob/main/internal/llm/prompts.go).", false},
		{"documentation filenames in prose", "Read AGENTS.md and CLAUDE.md before you start.", false},
		{"notice file in prose", "Provenance and compliance guidance live in NOTICE.", false},
		{"allowlisted proper nouns", "Use JavaScript and TypeScript on GitHub with macOS and iOS.", false},
		{"bare number without status context", "The team reviewed 404 support tickets last quarter.", false},
		{"caps verb without http context", "Click DELETE to remove the row from the table.", false},
		{"ordinary prose", "The server returns a response as soon as the upload finishes.", false},
		{"ordinary prose with proper nouns", "We shipped the release on Tuesday and told the team on Wednesday.", false},
	})
}

func TestDetectGoogleUppercaseUILabel(t *testing.T) {
	runHits(t, "uppercase-ui-label", DetectGoogleUppercaseUILabel, []hitCase{
		{"single word label", "Click **REFRESH**.", true},
		{"two word label", "Click **NEW PROJECT**, and then click **New Activity**.", true},
		{"unbolded label", "Go to ADVANCED SETTINGS to change the region.", true},
		{"sentence case label", "Click **Refresh**.", false},
		{"sentence case labels", "Click **New project**, and then click **New activity**.", false},
		{"allowlisted acronym", "Click **API** to open the reference.", false},
		{"allowlisted acronym pair", "Select JSON API from the menu.", false},
		{"sql keyword", "Use SELECT FROM orders to read the table.", false},
		{"ordinary prose", "Tap the Save button to store your work.", false},
		{"ordinary prose with verb", "Select the option that matches your region, and then continue.", false},
	})
}

func TestDetectGoogleVisualAppearanceReference(t *testing.T) {
	runHits(t, "visual-appearance-reference", DetectGoogleVisualAppearanceReference, []hitCase{
		{"bare icon", "Click the icon.", true},
		{"three lines", "Click the button with three lines.", true},
		{"green check mark", "Look for the green check mark.", true},
		{"bell icon", "Click the bell icon to see recent alerts.", true},
		{"colored control", "The red banner reports the last failure.", true},
		{"named control", "Click **Settings and utilities**.", false},
		{"named menu", "Click **Menu**.", false},
		{"named status", "Look for the Ready status.", false},
		{"capitalized label with icon", "Click the Settings icon in the top toolbar.", false},
		{"keyboard arrow", "Press the up arrow key to scroll back through your history.", false},
		{"green light idiom", "The team gave the green light to ship on Friday.", false},
		{"ordinary prose", "Click Save to store the file in your workspace.", false},
		{"ordinary prose with control", "The control panel lists every environment you can deploy to.", false},
	})
}
