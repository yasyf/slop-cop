package detectors

import "testing"

func TestDetectGoogleAlternativeInStep(t *testing.T) {
	runHits(t, "alternative-in-step", DetectGoogleAlternativeInStep, []hitCase{
		{"alternatively in step", "1. Click **Create instance**. Alternatively, you can run `gcloud compute instances create`.", true},
		{"you can also in step", "2. Set the region in the console. You can also set it in the config file.", true},
		{"or you can in step", "3. Wait for the build to finish. Or you can poll the API.", true},
		{"alternatively opens step", "1. Alternatively, run the gcloud command.", true},
		{"single method step", "1. Click **Create instance**.", false},
		{"alternative under its own heading", "## Create an instance by using the gcloud CLI\n\n1. Run `gcloud compute instances create`.", false},
		{"alternative in a paragraph", "Alternatively, you can install slop-cop from source.", false},
		{"plain procedure", "1. Install the CLI.\n2. Authenticate with your account.\n3. Create the instance.", false},
		{"note callout inside a step", "1. Deploy the service.\n    Note: alternatively, the service deploys itself.", false},
	})
}

func TestDetectGoogleAmbiguousSectionReference(t *testing.T) {
	runHits(t, "ambiguous-section-reference", DetectGoogleAmbiguousSectionReference, []hitCase{
		{"these sections describe", "These sections describe the views in the editor.\n\n### Data view", true},
		{"described in this section", "## Overview\n\nThe views are described in this section.\n\n### Data view\n\n### Code view", true},
		{"in this section colon", "## Overview\n\nIn this section, you configure the following:\n\n### Firewall rules", true},
		{"the following sections", "The following sections describe the views in the editor.\n\n### Data view", false},
		{"no following subsection", "In this section, you configure the firewall rules for the network.", false},
		{"ordinary preview", "## Overview\n\nThe service retries failed requests with exponential backoff.\n\n### Retries\n\n### Backoff", false},
		{"section named in prose", "The retry budget is documented separately.\n\n## Retries", false},
	})
}

func TestDetectGoogleDirectionalReference(t *testing.T) {
	runHits(t, "directional-reference", DetectGoogleDirectionalReference, []hitCase{
		{"diagram below", "In the diagram below, the resolver sits between the client and the app.", true},
		{"as shown above", "As shown above, the request fails.", true},
		{"right-hand side", "Click the menu on the right-hand side.", true},
		{"above diagram", "The above diagram shows the resolver.", true},
		{"table below", "The table below lists the quotas.", true},
		{"see below", "For the full list, see below.", true},
		{"upper right", "The status appears in the upper right of the console.", true},
		{"left navigation", "In the left navigation, click **Settings**.", true},
		{"button on the right", "Click the button on the right.", true},
		{"following diagram", "In the following diagram, the resolver sits between the client and the app.", false},
		{"preceding section", "As described in the preceding section, the request fails.", false},
		{"named menu", "In the navigation menu, click **Menu**.", false},
		{"numeric above", "Set the threshold above 50% to trigger an alert.", false},
		{"numeric below", "The latency stayed below 20 ms for the whole week.", false},
		{"below the target", "The error rate stayed below the target for the whole week.", false},
		{"on the right track", "The retry heuristic is on the right track.", false},
		{"see the following section", "For the full list, see the following section.", false},
	})
}

func TestDetectGoogleGerundHeading(t *testing.T) {
	runHits(t, "gerund-heading", DetectGoogleGerundHeading, []hitCase{
		{"creating an instance", "## Creating an instance", true},
		{"transferring data sets", "### Transferring data sets", true},
		{"getting started", "# Getting started", true},
		{"create an instance", "## Create an instance", false},
		{"transfer data sets", "### Transfer data sets", false},
		{"noun phrase heading", "## Introduction to BigQuery monitoring", false},
		{"deverbal noun heading", "## Monitoring your costs", false},
		{"single word heading", "## Billing", false},
		{"gerund in prose", "Creating an instance takes about a minute.", false},
	})
}

func TestDetectGoogleHeadingLevelSkip(t *testing.T) {
	runHits(t, "heading-level-skip", DetectGoogleHeadingLevelSkip, []hitCase{
		{"h2 to h4", "## Deploying\n\n#### Rollback", true},
		{"two level-1 headings", "# Guide\n\n# Reference", true},
		{"h1 to h3", "# Guide\n\n### Rollback", true},
		{"h2 to h3", "## Deploying\n\n### Rollback", false},
		{"h1 to h2", "# Guide\n\n## Reference", false},
		{"decrease is fine", "# Guide\n\n## Setup\n\n### Install\n\n## Usage", false},
		{"no headings", "Some ordinary prose with no headings at all.\n\nA second paragraph.", false},
		{"hashes in a code fence", "# Title\n\n```sh\n# a comment\n#### also a comment\n```\n\n## Next", false},
		{"front matter fence", "---\ntitle: Foo\n---\n\n# Title\n\n## Next", false},
	})
}

func TestDetectGoogleHeadingRepeatsTitle(t *testing.T) {
	runHits(t, "heading-repeats-title", DetectGoogleHeadingRepeatsTitle, []hitCase{
		{"section repeats title", "# Create and start VM instances\n\n## Create and start VM instances", true},
		{"section repeats title with stopwords", "# Create and start VM instances\n\n## Create and start the VM instances", true},
		{"narrowed sections", "# Create and start VM instances\n\n## Create a VM\n\n## Start a VM", false},
		{"ordinary readme", "# slop-cop\n\n## Install\n\n## Usage\n\n## How it works", false},
		{"no title", "## Install\n\n## Usage", false},
	})
}

func TestDetectGoogleHeadingSentenceCase(t *testing.T) {
	runHits(t, "heading-sentence-case", DetectGoogleHeadingSentenceCase, []hitCase{
		{"title case article", "## Create A New Instance", true},
		{"title case function words", "# Getting Started With The Cloud API", true},
		{"title case possessive", "## Configure Your Firewall", true},
		{"all capitalized corroborated", "# Guide\n\nThe network firewall settings control ingress.\n\n## Configure Network Firewall Settings\n\nMore prose follows here.", true},
		{"sentence case", "## Create a new instance", false},
		{"sentence case with proper nouns", "# Get started with the Cloud API", false},
		{"product names", "## Use BigQuery with Looker Studio", false},
		{"sentence case body", "# Guide\n\nThe network firewall settings control ingress.\n\n## Configure network firewall settings\n\nMore prose follows here.", false},
		{"numbered step heading", "## Step 1: Create the instance", false},
		{"acronyms", "## Configure the API and the CLI", false},
	})
}

func TestDetectGoogleInconsistentListPunctuation(t *testing.T) {
	runHits(t, "inconsistent-list-punctuation", DetectGoogleInconsistentListPunctuation, []hitCase{
		{"mixed periods", "- Big.\n- Small\n- Gratuitous.", true},
		{"mixed periods and case", "1. Set the quota\n2. Deploy the service.\n3. verify the health check", true},
		{"uniform fragments", "- Big\n- Small\n- Gratuitous", false},
		{"uniform steps", "1. Set the quota\n2. Deploy the service\n3. Verify the health check", false},
		{"uniform sentences", "- Install the CLI.\n- Authenticate with your account.\n- Create the instance.", false},
		{"term list of flags", "- `--verbose` prints more.\n- `--quiet` prints less\n- `--json` emits JSON.", false},
		{"two items only", "- Big.\n- Small", false},
		{"continuation list", "Record per candidate:\n\n- the section heading,\n- the verbatim prose stem,\n- any fenced code examples.", false},
		{"checkbox list of options", "- [ ] `signals=` pre-filter\n- [ ] `max_fires` set deliberately\n- [ ] `model` left at the default\n- [ ] For stateless checks, pass `agent=False`", false},
		{"colon item introduces a code block", "1. Read the rules.\n2. Check the workflows.\n3. Incident archaeology. Run:\n\n```sh\ngit log --oneline\n```", false},
		{"prose not a list", "The quota is set. The service deploys. The health check passes.", false},
	})
}

func TestDetectGoogleIntroRestatesHeading(t *testing.T) {
	runHits(t, "intro-restates-heading", DetectGoogleIntroRestatesHeading, []hitCase{
		{"this section describes", "## Customize the buttons\n\nThis section describes how to customize the buttons.", true},
		{"contentless objectives intro", "## Objectives\n\nIn the following tutorial, you will complete the following tasks:", true},
		{"context instead of restatement", "## Customize the buttons\n\nButton layouts are per-workspace, so changes here do not affect other members of your team.", false},
		{"list under the heading", "## Objectives\n\n- Create an instance\n- Snapshot an instance", false},
		{"prerequisites", "## Prerequisites\n\nInstall the gcloud CLI and authenticate before you begin.", false},
		{"troubleshooting intro", "## Troubleshoot connection errors\n\nA dropped connection usually means the firewall closed the port.", false},
		{"no heading", "This section describes how to customize the buttons.", false},
		{"one word heading with content", "## Test\n\nThe hooks carry inline tests. Run them against the module directory.", false},
		{"html block under a heading", "## HTML\n\n<div class=\"note\">\n  html content lives here\n</div>", false},
		{"gerund heading restated", "## Configuring the service\n\nThis section describes how to configure the service.", true},
	})
}
