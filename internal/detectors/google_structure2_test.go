package detectors

import "testing"

func TestDetectGoogleKeyboardShortcutInstruction(t *testing.T) {
	runHits(t, "keyboard-shortcut-instruction", DetectGoogleKeyboardShortcutInstruction, []hitCase{
		{"press chord step", "1. Press Ctrl+C, and then press Ctrl+V.", true},
		{"press chord sentence", "Press Cmd+S to save your work.", true},
		{"bare chord after imperative", "Use Ctrl+Shift+P to open the command palette.", true},
		{"named action step", "1. Copy the command, and then paste it.", false},
		{"named action sentence", "Save your work.", false},
		{"shortcuts table", "| Action | Shortcut |\n| --- | --- |\n| Copy | Ctrl+C |", false},
		{"shortcuts section", "## Keyboard shortcuts\n\nUse Ctrl+Shift+P to open the command palette.", false},
		{"descriptive prose", "The Ctrl+C shortcut copies the current selection.", false},
		{"ordinary prose", "Set the request timeout to 30 seconds before you deploy the service.", false},
		{"enter key alone", "Press Enter to continue.", false},
	})
}

func TestDetectGoogleLinkInHeading(t *testing.T) {
	runHits(t, "link-in-heading", DetectGoogleLinkInHeading, []hitCase{
		{"link in heading tail", "## See the [migration guide](/migrate)", true},
		{"heading is a link", "### [Estimate costs](https://example.com/pricing)", true},
		{"html heading anchor", "<h2>Read the <a href=\"/guide\">guide</a></h2>", true},
		{"link below heading", "## Estimate costs\n\nFor pricing details, see the [migration guide](/migrate).", false},
		{"bracketed qualifier", "## Arrays [experimental]", false},
		{"plain heading", "## Configure the firewall", false},
		{"link in body text", "The [pricing page](https://example.com) lists the current rates.", false},
	})
}

func TestDetectGoogleMultiActionStep(t *testing.T) {
	runHits(t, "multi-action-step", DetectGoogleMultiActionStep, []hitCase{
		{"three instructions", "1. Create the bucket. Then upload the sample file. Finally, set the ACL.", true},
		{"two instructions", "1. Click the search box and type `custom function`. Press **Enter**.", true},
		{"joined with and then", "1. Click the search box, type `custom function`, and then press **Enter**.", false},
		{"menu path", "1. Click **Next** > **Finish**.", false},
		{"action then result", "1. Click **Run**. The query results appear after the query runs.", false},
		{"context clause step", "1. In the Google Cloud console, go to the VM instances page.", false},
		{"prose not a step", "Create the bucket. Then upload the sample file.", false},
	})
}

func TestDetectGoogleNumberedHeading(t *testing.T) {
	runHits(t, "numbered-heading", DetectGoogleNumberedHeading, []hitCase{
		{"leading ordinal", "## 1. Create an instance", true},
		{"step word", "## Step 2: Configure the firewall", true},
		{"dotted section number", "### 2.1 Authentication", true},
		{"plain heading", "## Create an instance", false},
		{"plain heading two", "## Configure the firewall", false},
		{"version in name", "## OAuth 2.0 tokens", false},
		{"release heading", "## 1.2.0 (2024-05-01)", false},
		{"sequence word in prose", "Section 3 covers the streaming API.", false},
	})
}

func TestDetectGoogleRunTheFollowingCommand(t *testing.T) {
	runHits(t, "run-the-following-command", DetectGoogleRunTheFollowingCommand, []hitCase{
		{"by running the following command", "In Cloud Shell, deploy the load generator by running the following command:", true},
		{"run the following commands", "1. Run the following commands to configure the firewall:", true},
		{"bare run the following", "1. Run the following:", true},
		{"outcome first", "In Cloud Shell, deploy the load generator:", false},
		{"purpose first", "1. Define a firewall rule to allow internal traffic:", false},
		{"following command as subject", "The following command creates a bucket:", false},
		{"named command", "Run `terraform apply` to provision the cluster.", false},
		{"inside code fence", "```\nRun the following command:\n```", false},
	})
}

func TestDetectGoogleSingleItemList(t *testing.T) {
	runHits(t, "single-item-list", DetectGoogleSingleItemList, []hitCase{
		{"one step procedure", "To clear the entire log, follow this step:\n\n1. Click **Clear logcat**.", true},
		{"one bullet after preamble", "The API supports one action:\n\n- Create", true},
		{"html single item list", "<ul><li>Create</li></ul>", true},
		{"bullet without preamble", "- To clear the entire log, click **Clear logcat**.", false},
		{"folded into prose", "The API supports only the Create action.", false},
		{"two bullets", "Prerequisites:\n\n- A billing account\n- Billing enabled on the project", false},
		{"two steps", "Steps:\n\n1. Open the console.\n2. Click **Save**.", false},
		{"nested sublist", "Options:\n\n- one\n  - detail\n- two", false},
	})
}

func TestDetectGoogleStepLacksImperative(t *testing.T) {
	runHits(t, "step-lacks-imperative", DetectGoogleStepLacksImperative, []hitCase{
		{"background first", "1. You need the project ID later in this document. Retrieve the project ID.", true},
		{"description first", "1. The repository contains the sample data. Clone it.", true},
		{"action first", "1. Clone the repository that contains the sample data.", false},
		{"action then context", "1. Retrieve the project ID. You need it later in this document.", false},
		{"optional prefix", "1. Optional: Type an arbitrary string.", false},
		{"descriptive list item", "1. The scheduler assigns pods to nodes based on resource requests.", false},
		{"context clause step", "1. In the console, go to the VM instances page.", false},
	})
}
