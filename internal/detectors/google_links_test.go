package detectors

import (
	"strings"
	"testing"
)

func TestDetectGoogleAmbiguousRepeatedLinkText(t *testing.T) {
	runHits(t, "ambiguous-repeated-link-text", DetectGoogleAmbiguousRepeatedLinkText, []hitCase{
		{
			"same label, two destinations",
			"See [configuration](/compute/config) … See [configuration](/storage/config).", true,
		},
		{
			"labels name their destinations",
			"See [Compute Engine configuration](/compute/config) … See [Cloud Storage configuration](/storage/config).", false,
		},
		{
			"same label, same destination",
			"See [configuration](/compute/config) and again [configuration](/compute/config).", false,
		},
		{
			"distinct labels and destinations",
			"See [Manage indexes](/indexes) and [Query data](/query) for the two halves of the flow.", false,
		},
		{
			"ordinary prose without links",
			"The scheduler assigns each task to the instance with the smallest queue.", false,
		},
		{
			"markdown link syntax inside a fence",
			"```\nSee [configuration](/compute/config) and [configuration](/storage/config).\n```", false,
		},
	})
}

func TestDetectGoogleDuplicateLinkTarget(t *testing.T) {
	runHits(t, "duplicate-link-target", DetectGoogleDuplicateLinkTarget, []hitCase{
		{
			"same destination twice nearby",
			"…see [Manage indexes](/indexes). …Indexes are described in [Manage indexes](/indexes).", true,
		},
		{
			"second mention carries no link",
			"…see [Manage indexes](/indexes). …Indexes are described there.", false,
		},
		{
			"two different destinations",
			"See [Manage indexes](/indexes) and [Query data](/query) for the two halves of the flow.", false,
		},
		{
			"separate level-2 sections",
			"## Indexes\n\nSee [Manage indexes](/indexes).\n\n## Queries\n\nSee [Manage indexes](/indexes).", false,
		},
		{
			"more than sixty lines apart",
			"See [Manage indexes](/indexes).\n" + strings.Repeat("The queue drains in order.\n", 70) +
				"See [Manage indexes](/indexes).", false,
		},
		{
			"fragment-only destination",
			"Jump back to [the top](#top), or return to [the top](#top) at any point.", false,
		},
		{
			"ordinary prose without links",
			"Each index is rebuilt when the schema changes and the write path stays available.", false,
		},
	})
}

func TestDetectGoogleFootnoteUsage(t *testing.T) {
	runHits(t, "footnote-usage", DetectGoogleFootnoteUsage, []hitCase{
		{"markdown footnote reference", "The quota resets daily.[^1]", true},
		{"superscript marker", "The quota resets daily.<sup>1</sup>", true},
		{"footnote definition", "[^1]: The quota resets at 00:00 UTC.", true},
		{"unicode superscript", "The quota resets daily¹ for every project.", true},
		{"parenthetical instead", "The quota resets daily (at 00:00 UTC).", false},
		{"cross-reference instead", "The quota resets daily. For more information, see [Quotas](/quotas).", false},
		{"exponent notation", "The counter wraps at 2<sup>32</sup> requests.", false},
		{"square kilometres", "The region covers 40 km² of coastline.", false},
		{"character class in code span", "Set the pattern to `[^abc]` before you run the job.", false},
		{"ordinary prose", "The quota resets once a day and applies to every project in the folder.", false},
	})
}

func TestDetectGoogleOverlongLinkText(t *testing.T) {
	runHits(t, "overlong-link-text", DetectGoogleOverlongLinkText, []hitCase{
		{
			"label stretched to a sentence",
			"[If you want to know how the scheduler decides which instance receives the next task, read this explanation of the algorithm](/sched).", true,
		},
		{
			"label carries a sentence boundary",
			"[Set up the agent. Then restart it](/setup)", true,
		},
		{
			"destination title as the label",
			"[Reliable task scheduling on Compute Engine](/sched)", false,
		},
		{
			"image alt text",
			"![A long screenshot of the console showing the instance list with every column visible and the filter applied](/img.png)", false,
		},
		{
			"short label",
			"See [Manage indexes](/indexes) for details.", false,
		},
		{
			"ordinary prose with a link",
			"For more information, see [Cloud Storage pricing](/storage/pricing).", false,
		},
		{
			"version in the label",
			"Install [Node.js v1.2](/node) before you start.", false,
		},
	})
}

func TestDetectGooglePunctuationInsideLink(t *testing.T) {
	runHits(t, "punctuation-inside-link", DetectGooglePunctuationInsideLink, []hitCase{
		{"period inside markdown link", "Read the [setup guide.](/setup)", true},
		{
			"period inside HTML anchor",
			"For more information, see <a href=\"#Test\">Test your code.</a>", true,
		},
		{"quoted markdown label", "Read the [\"setup guide\"](/setup) first.", true},
		{"period outside markdown link", "Read the [setup guide](/setup).", false},
		{
			"period outside HTML anchor",
			"For more information, see <a href=\"#Test\">Test your code</a>.", false,
		},
		{"ordinary link", "See [Manage indexes](/indexes) for details.", false},
		{"version in the label", "Install [Node.js v1.2](/node) before you start.", false},
		{"ellipsis in the label", "See [and so on...](/more) for the rest.", false},
		{
			"ordinary prose without links",
			"The scheduler assigns each task to the instance with the smallest queue.", false,
		},
	})
}

func TestDetectGoogleUnexplainedLinkBehavior(t *testing.T) {
	runHits(t, "unexplained-link-behavior", DetectGoogleUnexplainedLinkBehavior, []hitCase{
		{
			"undeclared PDF download",
			"For more information, see [security features](/docs/security.pdf).", true,
		},
		{
			"undeclared mail composer",
			"<a href=\"mailto:support@example.com\">Technical Support</a>", true,
		},
		{
			"undeclared new tab",
			"<a href=\"/style/accessibility\" target=\"_blank\">Accessible content</a>", true,
		},
		{
			"download announced in the label",
			"For more information, [download the security features PDF](/docs/security.pdf).", false,
		},
		{
			"mail announced in the label",
			"<a href=\"mailto:support@example.com\">send email to Technical Support</a>", false,
		},
		{
			"link stays in the current tab",
			"<a href=\"/style/accessibility\">Accessible content</a>", false,
		},
		{
			"archive announced in the label",
			"Fetch the [release archive](/dl/tool.tar.gz) and extract it.", false,
		},
		{
			"ordinary page link",
			"For more information, see [Cloud Storage buckets](/storage/buckets).", false,
		},
		{
			"ordinary prose without links",
			"The installer writes its logs to the directory you name on the command line.", false,
		},
	})
}

func TestDetectGoogleVagueLinkText(t *testing.T) {
	runHits(t, "vague-link-text", DetectGoogleVagueLinkText, []hitCase{
		{"click here", "Want more? [Click here!](/style/links)", true},
		{"this document", "For more information, see [this document](/style/links).", true},
		{"learn more", "[Learn more](/style/links)", true},
		{
			"destination title as the label",
			"For more information, see [Make headings into link targets](/style/headings).", false,
		},
		{"imperative label", "[Install the CLI](/install).", false},
		{"ordinary link", "See [Manage indexes](/indexes) for details.", false},
		{
			"ordinary prose with a link",
			"The [pricing calculator](/calc) estimates the monthly cost of a project.", false,
		},
		{
			"the word here outside a link",
			"Start here, then open the console and create your first bucket.", false,
		},
	})
}
