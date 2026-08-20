package detectors

import "testing"

func TestDetectGoogleMissingTerminalPeriod(t *testing.T) {
	runHits(t, "missing-terminal-period", DetectGoogleMissingTerminalPeriod, []hitCase{
		{"bad_unterminated", "The scheduler retries the request three times before it gives up", true},
		{"good_terminated", "The scheduler retries the request three times before it gives up.", false},
		{"heading", "## The scheduler retries the request three times", false},
		{"list_item", "- The scheduler retries the request three times before it gives up", false},
		{"table_row", "| The scheduler retries the request three times | yes |", false},
		{"fenced", "```\nThe scheduler retries the request three times before it gives up\n```", false},
		{"trailing_url", "Full details live at https://example.com/docs/scheduler-retries", false},
		{"image_only", "![The scheduler retries the request three times](img.png)", false},
		{"colon_lead_in", "The scheduler retries the request in three cases:", false},
		{"short_fragment", "Supported platforms and architectures", false},
		{"noun_phrase", "A short list of supported platforms and architectures", false},
		{"ordinary_prose", "The API returns a token. Store it somewhere safe.", false},
		{"ordinary_question", "Which region does the job run in?", false},
	})
}

func TestDetectGoogleNumberUnitModifierHyphen(t *testing.T) {
	runHits(t, "number-unit-modifier-hyphen", DetectGoogleNumberUnitModifierHyphen, []hitCase{
		{"bad_64_bit", "a 64 bit system", true},
		{"bad_five_minute", "a five minute wait", true},
		{"bad_100000_byte", "100,000 byte files", true},
		{"good_64_bit", "a 64-bit system", false},
		{"good_five_minute", "a five-minute wait", false},
		{"good_plural_unit", "the wait lasted five minutes", false},
		{"good_verb_follows", "Wait one second before retrying.", false},
		{"good_determiner_follows", "After one day the cache expires.", false},
		{"encoding_name", "Offsets are UTF-8 byte offsets, not code units.", false},
		{"participle_follows", "One line saying so replaces the table.", false},
		{"ordinary_prose", "The job ran for five minutes and then exited.", false},
		{"ordinary_prose_2", "Each row of the table holds a single value.", false},
	})
}

func TestDetectGoogleOptionalPluralParens(t *testing.T) {
	runHits(t, "optional-plural-parens", DetectGoogleOptionalPluralParens, []hitCase{
		{"bad_key_s", "To find your API key(s), visit the Credentials page.", true},
		{"bad_port_s", "The linecard can contain port(s).", true},
		{"bad_child_ren", "Remove the child(ren) of the node.", true},
		{"good_key", "To find your API key, visit the Credentials page.", false},
		{"good_ports", "The linecard can contain one or more ports.", false},
		{"code_span", "Call `format(s)` to render the value.", false},
		{"ordinary_prose", "The API accepts a list of keys and returns a token.", false},
		{"ordinary_prose_2", "Set the timeout (in seconds) before you start the job.", false},
	})
}

func TestDetectGoogleOverlongCompoundModifier(t *testing.T) {
	runHits(t, "overlong-compound-modifier", DetectGoogleOverlongCompoundModifier, []hitCase{
		{"bad_edition_specific", "edition-2023-specific test cases", true},
		{"bad_cmek_policy", "a customer-managed-encryption-key policy", true},
		{"good_rewritten", "test cases specific to the 2023 edition", false},
		{"good_two_component", "a policy for customer-managed encryption keys", false},
		{"idiom_state_of_the_art", "This is a state-of-the-art model.", false},
		{"idiom_end_to_end", "The pipeline runs end-to-end tests nightly.", false},
		{"cross_prefix", "Use cross-region-replication settings sparingly.", false},
		{"pre_prefix", "Reproduce the pre-base-layer output byte for byte.", false},
		{"connective_and", "The agent runs a gather-and-verify loop.", false},
		{"connective_to", "Teams need agent-to-agent handoffs mid-run.", false},
		{"flag", "Pass the --log-level-debug flag when you debug.", false},
		{"code_span", "Run `slop-cop-check main.go` before you commit.", false},
		{"iso_date", "Released on 2024-01-15 and mirrored the same day.", false},
		{"ordinary_prose", "The service is cloud-native and scales horizontally.", false},
		{"ordinary_prose_2", "Read the configuration file before you edit it.", false},
	})
}

func TestDetectGoogleParentheticalAcronymPossessive(t *testing.T) {
	runHits(t, "parenthetical-acronym-possessive", DetectGoogleParentheticalAcronymPossessive, []hitCase{
		{"bad_ftc", "The Federal Trade Commission's (FTC's) rule takes effect in June.", true},
		{"good_rewritten", "The rule that the Federal Trade Commission (FTC) issued takes effect in June.", false},
		{"ordinary_prose", "The API (REST) endpoint is stable across releases.", false},
		{"ordinary_prose_2", "Follow the FTC's guidance on disclosures.", false},
	})
}

func TestDetectGoogleParentheticalPeriodPlacement(t *testing.T) {
	runHits(t, "parenthetical-period-placement", DetectGoogleParentheticalPeriodPlacement, []hitCase{
		{"bad_period_inside", "Restart the agent (the change takes effect immediately.)", true},
		{"bad_period_outside", "The scheduler retries on its own. (No manual step is needed).", true},
		{"good_period_outside", "Restart the agent (the change takes effect immediately).", false},
		{"good_period_inside", "The scheduler retries on its own. (No manual step is needed.)", false},
		{"abbreviation", "The client retries transient failures (for example, on 500 errors, etc.)", false},
		{"citation", "Compare the two layouts (see Figure 2.)", false},
		{"ordinary_prose", "Install the CLI (see the installation guide).", false},
		{"ordinary_prose_2", "Set the flag (the default is false).", false},
	})
}

func TestDetectGooglePluralPossessiveApostrophe(t *testing.T) {
	runHits(t, "plural-possessive-apostrophe", DetectGooglePluralPossessiveApostrophe, []hitCase{
		{"bad_models", "Extend the models's capabilities.", true},
		{"bad_users", "Raise the users's quota.", true},
		{"good_models", "Extend the models' capabilities.", false},
		{"good_class", "Raise the storage class's quota.", false},
		{"singular_bus", "The bus's route changed last week.", false},
		{"singular_lens", "Check the lens's focus before you shoot.", false},
		{"singular_index", "The index's size grew overnight.", false},
		{"ordinary_prose", "The company's policy covers every contractor.", false},
		{"ordinary_prose_2", "Each user's session expires after an hour.", false},
	})
}

func TestDetectGooglePostColonCapitalization(t *testing.T) {
	runHits(t, "post-colon-capitalization", DetectGooglePostColonCapitalization, []hitCase{
		{"bad_tone", "Tone: Concise, direct, and free of filler.", true},
		{"bad_limit", "The cache has one limit: Entries expire after an hour.", true},
		{"good_tone", "Tone: concise, direct, and free of filler.", false},
		{"good_limit", "The cache has one limit: entries expire after an hour.", false},
		{"notice_label", "Note: The job restarts on its own after a crash.", false},
		{"bold_notice_label", "**Note**: Restart the daemon after editing the config.", false},
		{"imperative", "Step 1: Open the console and pick a project.", false},
		{"proper_noun", "Supported languages: Python and its standard library.", false},
		{"identifier", "Set this field: MaxRetries controls the ceiling.", false},
		{"title_case_citation", "Wikipedia: Signs of AI Writing", false},
		{"ordinary_prose", "The command has one required flag: the input path.", false},
		{"ordinary_prose_2", "Read the guide before you file a bug.", false},
	})
}

func TestDetectGooglePostVerbCompoundHyphen(t *testing.T) {
	runHits(t, "post-verb-compound-hyphen", DetectGooglePostVerbCompoundHyphen, []hitCase{
		{"bad_well_designed", "The app is well-designed.", true},
		{"bad_real_time", "The logs are written in real-time.", true},
		{"bad_ly_adverb", "The endpoint is widely-used across the fleet.", true},
		{"good_well_designed", "The app is well designed.", false},
		{"good_real_time", "The logs are written in real time.", false},
		{"good_on_premises", "You can deploy the app on-premises.", false},
		{"non_adverb_head", "The comparison is case-sensitive.", false},
		{"non_adverb_head_2", "Its efficiency gain is effort-dependent.", false},
		{"ly_noun_head", "The output is supply-limited.", false},
		{"three_component", "The dependency list is up-to-date.", false},
		{"cross_prefix", "The client is cross-platform.", false},
		{"ordinary_prose", "The scheduler retries the request before it gives up.", false},
		{"ordinary_prose_2", "A read-only replica serves the dashboard.", false},
	})
}
