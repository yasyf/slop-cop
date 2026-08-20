package detectors

import "testing"

func TestDetectGoogleCheckboxVerb(t *testing.T) {
	runHits(t, "checkbox-verb", DetectGoogleCheckboxVerb, []hitCase{
		{"check the checkbox", "Check the **Automatically check for updates** checkbox.", true},
		{"uncheck", "Uncheck the **Bookmarks** checkbox.", true},
		{"choose a label", "Choose **Advanced settings**.", true},
		{"deselect near a check box", "To clear it, deselect the check box.", true},
		{"unchecked state", "The checkbox is unchecked by default.", true},
		{"select the checkbox", "Select the **Automatically check for updates** checkbox.", false},
		{"clear the checkbox", "Clear the **Bookmarks** checkbox.", false},
		{"select a label", "Select **Advanced settings**.", false},
		{"check the logs", "Run the tests and check the logs for errors.", false},
		{"checksum", "The service returns a checksum for each upload.", false},
		{"unchecked exception", "An unchecked exception aborts the request.", false},
	})
}

func TestDetectGoogleExtensionAsFileType(t *testing.T) {
	runHits(t, "extension-as-file-type", DetectGoogleExtensionAsFileType, []hitCase{
		{"png file", "Upload a .png file.", true},
		{"sh file", "Run the .sh file.", true},
		{"yaml file", "Edit the .yaml file and redeploy.", true},
		{"format name", "Upload a PNG file.", false},
		{"bash file", "Run the Bash file.", false},
		{"extension noun", "Files with the \x60.png\x60 extension are PNG files.", false},
		{"full filename", "Open config.yaml and set the region.", false},
		{"format in prose", "The API returns JSON, and the client writes it to disk.", false},
		{"version number", "Release 1.2.3 shipped on Monday.", false},
	})
}

func TestDetectGoogleImpreciseUiNoun(t *testing.T) {
	runHits(t, "imprecise-ui-noun", DetectGoogleImpreciseUINoun, []hitCase{
		{"zippy", "To expand the **Advanced options** section, click the zippy.", true},
		{"menu item", "In the **File** menu, choose the **Open** menu item.", true},
		{"text box", "In the **Instance text box**, enter a name.", true},
		{"console sub-page as tab", "Open the **Instances** tab.", true},
		{"bold label as area", "The values appear in the **Results** area.", true},
		{"expander arrow", "To expand the **Advanced options** section, click the expander arrow.", false},
		{"select the command", "In the **File** menu, select **Open**.", false},
		{"field", "In the **Instance** field, enter a name.", false},
		{"list of instances", "The API returns a list of instances.", false},
		{"menu bar", "The menu bar lists the available commands.", false},
		{"tab key", "Press the Tab key to move to the next field.", false},
	})
}

func TestDetectGoogleInflectedCodeElement(t *testing.T) {
	runHits(t, "inflected-code-element", DetectGoogleInflectedCodeElement, []hitCase{
		{"possessive code span", "\x60ADDRESS\x60's value is defined in \x60settings.h\x60.", true},
		{"http verb as predicate", "\x60POST\x60 the data.", true},
		{"call possessive", "Check getUser()'s response code.", true},
		{"pluralized code span", "Register both \x60handler\x60s before you start.", true},
		{"snake_case possessive", "The max_retries's default is 3.", true},
		{"noun carries the inflection", "The \x60ADDRESS\x60 constant's value is defined in the \x60settings.h\x60 file.", false},
		{"post request", "To add the data, send a \x60POST\x60 request.", false},
		{"call in a noun phrase", "Check the response code returned by \x60getUser()\x60.", false},
		{"flag in code font", "Use the \x60--verbose\x60 flag to print each step.", false},
		{"ordinary possessive", "The user's session expires after an hour.", false},
		{"brand possessive", "macOS's keychain stores the token.", false},
	})
}

func TestDetectGoogleOutputIntroPhrase(t *testing.T) {
	runHits(t, "output-intro-phrase", DetectGoogleOutputIntroPhrase, []hitCase{
		{"you should see", "You should see the following output:", true},
		{"sample output", "Sample output:", true},
		{"lead-in above a fence", "You should see this\n\n```\nok\n```\n", true},
		{"results above a fence", "Results:\n\n```\nok\n```\n", true},
		{"standard similar", "The output is similar to the following:", false},
		{"standard exact", "The output is the following:", false},
		{"phrase inside a fence", "```\nSample output: ok\n```\n", false},
		{"see in running prose", "You should see a confirmation email within a few minutes.", false},
		{"returns in running prose", "The command returns the instance list.", false},
		{"bare results line", "Results: the migration finished in under a minute", false},
	})
}

func TestDetectGoogleToggleAsVerb(t *testing.T) {
	runHits(t, "toggle-as-verb", DetectGoogleToggleAsVerb, []hitCase{
		{"toggle the setting", "Toggle the Wi-Fi setting.", true},
		{"to toggle", "To toggle dark mode, open **Settings**.", true},
		{"toggle on", "Toggle on the debug flag.", true},
		{"toggling", "Toggling the value restarts the daemon.", true},
		{"turn on", "To turn on the setting, click the **Wi-Fi** toggle.", false},
		{"toggle position", "In **Settings**, click the **Magic mode** toggle to the on position.", false},
		{"setting is off", "The dark mode setting is off by default.", false},
		{"toggles as a plural noun", "Two toggles appear at the top of the page.", false},
	})
}

func TestDetectGoogleUiLabelDecoration(t *testing.T) {
	runHits(t, "ui-label-decoration", DetectGoogleUILabelDecoration, []hitCase{
		{"quoted labels", "Select \"New Activity\", and then click the \"Next\" button.", true},
		{"click on", "Click on Save.", true},
		{"trailing button noun", "Click the **Next** button.", true},
		{"bold label", "Select the **New activity** checkbox, and then click **Next**.", false},
		{"bold button label", "Click **Save**.", false},
		{"no ui instruction", "The console shows the current status.", false},
		{"lowercase target", "Tap the notification to open the app.", false},
	})
}

func TestDetectGoogleUiPreposition(t *testing.T) {
	runHits(t, "ui-preposition", DetectGoogleUIPreposition, []hitCase{
		{"on a dialog", "On the **Alert** dialog, click **OK**.", true},
		{"in a page", "In the **Create an instance** page, click **Add**.", true},
		{"on a field", "On the **Name** field, enter a value.", true},
		{"in a dialog", "In the **Alert** dialog, click **OK**.", false},
		{"on a page", "On the **Create an instance** page, click **Add**.", false},
		{"mailing list", "Post the question on the mailing list.", false},
		{"stored in the cache", "The values are stored in the cache.", false},
	})
}

func TestDetectGoogleUnfencedCommandLine(t *testing.T) {
	runHits(t, "unfenced-command-line", DetectGoogleUnfencedCommandLine, []hitCase{
		{"command in running prose", "Run gcloud compute instances list --zones=us-central1-a to see your VMs.", true},
		{"prompted line", "$ kubectl get pods\n", true},
		{"bare command line", "kubectl get pods --all-namespaces\n", true},
		{"fenced command", "To list your VMs, run the following command:\n\n```\ngcloud compute instances list\n```\n", false},
		{"inline code span", "Use \x60gcloud compute instances list\x60 to see your VMs.", false},
		{"tool named in prose", "The Git history shows every change.", false},
		{"sentence about a tool", "git is a distributed version control system that many teams use\n", false},
		{"no command at all", "Install the CLI, and then authenticate.", false},
	})
}
