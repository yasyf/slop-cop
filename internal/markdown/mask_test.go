package markdown

import (
	"strings"
	"testing"

	"github.com/yasyf/slop-cop/internal/detectors"
	"github.com/yasyf/slop-cop/internal/types"
)

// ── Shared invariant helper ─────────────────────────────────────────────────

// assertInvariants checks the contract Analyze promises about its output.
func assertInvariants(t *testing.T, src, masked string, suppress []Range, fm Range) {
	t.Helper()
	if len(masked) != len(src) {
		t.Fatalf("length mismatch: len(masked)=%d len(src)=%d", len(masked), len(src))
	}
	for i := 0; i < len(src); i++ {
		if src[i] == '\n' && masked[i] != '\n' {
			t.Fatalf("newline at %d not preserved (masked=%q)", i, masked[i])
		}
	}
	for i := 0; i < len(masked); i++ {
		if masked[i] == src[i] {
			continue
		}
		if masked[i] != ' ' {
			t.Fatalf("masked byte at %d is %q, want ' ' or original %q", i, masked[i], src[i])
		}
	}
	for _, r := range suppress {
		if r.Start < 0 || r.End > len(src) || r.Start > r.End {
			t.Fatalf("bad suppress range %+v for src len %d", r, len(src))
		}
		if r.Kind == 0 {
			t.Fatalf("suppress range %+v has zero Kind", r)
		}
	}
	// Check sort order.
	for i := 1; i < len(suppress); i++ {
		a, b := suppress[i-1], suppress[i]
		if a.Start > b.Start {
			t.Fatalf("suppress not sorted at index %d: %+v > %+v", i, a, b)
		}
	}
	if fm.Start >= 0 {
		if fm.End > len(src) || fm.Start > fm.End {
			t.Fatalf("bad front-matter range %+v for src len %d", fm, len(src))
		}
	}
}

// ── Fenced code blocks ─────────────────────────────────────────────────────

func TestAnalyze_FencedCodeBash(t *testing.T) {
	src := "pre text\n\n```bash\nutilize this thing\n```\n\npost text\n"
	masked, _, _ := Analyze(src)
	assertInvariants(t, src, masked, nil, Range{Start: -1, End: -1})
	// Code line should be masked.
	if strings.Contains(masked, "utilize") {
		t.Fatalf("expected 'utilize' masked, got masked=%q", masked)
	}
	// Surrounding prose preserved.
	if !strings.Contains(masked, "pre text") || !strings.Contains(masked, "post text") {
		t.Fatalf("surrounding prose mangled: %q", masked)
	}
	// Detector should not fire on slop inside the code block.
	if hasRule(detectors.RunClient(masked), "elevated-register") {
		t.Fatalf("elevated-register fired inside fenced code")
	}
}

func TestAnalyze_FencedCodeUnlabelled(t *testing.T) {
	src := "```\nrobust and pivotal tapestry\n```\n"
	masked, _, _ := Analyze(src)
	assertInvariants(t, src, masked, nil, Range{Start: -1, End: -1})
	if hasRule(detectors.RunClient(masked), "overused-intensifiers") {
		t.Fatalf("overused-intensifiers fired inside fenced code")
	}
}

func TestAnalyze_FencedCodeTilde(t *testing.T) {
	src := "~~~\nleverage the synergy\n~~~\n"
	masked, _, _ := Analyze(src)
	assertInvariants(t, src, masked, nil, Range{Start: -1, End: -1})
	if hasRule(detectors.RunClient(masked), "overused-intensifiers") {
		t.Fatalf("fired on ~~~ code")
	}
}

func TestAnalyze_IndentedCodeBlock(t *testing.T) {
	src := "some prose\n\n    utilize this\n    robust framework\n\nmore prose\n"
	masked, _, _ := Analyze(src)
	assertInvariants(t, src, masked, nil, Range{Start: -1, End: -1})
	// Indented code should not fire slop rules.
	if hasRule(detectors.RunClient(masked), "elevated-register") {
		t.Fatalf("elevated-register fired inside indented code block")
	}
}

func TestAnalyze_FenceInsideListItem(t *testing.T) {
	src := "- item with code:\n\n    ```\n    utilize this\n    ```\n\n- other item\n"
	masked, _, _ := Analyze(src)
	assertInvariants(t, src, masked, nil, Range{Start: -1, End: -1})
	if hasRule(detectors.RunClient(masked), "elevated-register") {
		t.Fatalf("elevated-register fired inside fence in list item")
	}
}

func TestAnalyze_UnterminatedFence(t *testing.T) {
	src := "start\n\n```bash\nutilize this\nforever\n"
	masked, _, _ := Analyze(src)
	assertInvariants(t, src, masked, nil, Range{Start: -1, End: -1})
	if hasRule(detectors.RunClient(masked), "elevated-register") {
		t.Fatalf("elevated-register fired in unterminated fence")
	}
}

// ── Inline code ────────────────────────────────────────────────────────────

func TestAnalyze_InlineCodeSingleTick(t *testing.T) {
	src := "Avoid `utilize` in prose; prefer use.\n"
	masked, _, _ := Analyze(src)
	assertInvariants(t, src, masked, nil, Range{Start: -1, End: -1})
	// `utilize` inside code span must not fire.
	// "prefer use" is bare prose and should be unchanged.
	if !strings.Contains(masked, "prefer use") {
		t.Fatalf("prose mangled: %q", masked)
	}
	// Only backticks + inner should be spaces.
	if strings.Contains(masked, "utilize") {
		t.Fatalf("utilize not masked: %q", masked)
	}
}

func TestAnalyze_InlineCodeDoubleTickContainingBacktick(t *testing.T) {
	src := "refer to ``a `b` c`` in prose\n"
	masked, _, _ := Analyze(src)
	assertInvariants(t, src, masked, nil, Range{Start: -1, End: -1})
	// Inside double-tick code span, content including literal backtick masked.
	if strings.Contains(masked, "`b`") {
		t.Fatalf("inner backticks not masked: %q", masked)
	}
}

func TestAnalyze_InlineCodeWithURL(t *testing.T) {
	src := "see `https://example.com/robust` for details\n"
	masked, _, _ := Analyze(src)
	assertInvariants(t, src, masked, nil, Range{Start: -1, End: -1})
	if hasRule(detectors.RunClient(masked), "overused-intensifiers") {
		t.Fatalf("robust inside code span fired overused-intensifiers")
	}
}

// ── Links & images ─────────────────────────────────────────────────────────

func TestAnalyze_InlineLinkDestinationMasked(t *testing.T) {
	src := "See [click](https://example.com/utilize) for more.\n"
	masked, _, _ := Analyze(src)
	assertInvariants(t, src, masked, nil, Range{Start: -1, End: -1})
	// Link text `click` preserved.
	if !strings.Contains(masked, "[click]") {
		t.Fatalf("link text lost: %q", masked)
	}
	// URL `utilize` must not fire.
	if hasRule(detectors.RunClient(masked), "elevated-register") {
		t.Fatalf("elevated-register fired in link URL")
	}
}

func TestAnalyze_ImageDestinationMasked(t *testing.T) {
	src := "![alt text](robust.png)\n"
	masked, _, _ := Analyze(src)
	assertInvariants(t, src, masked, nil, Range{Start: -1, End: -1})
	if !strings.Contains(masked, "![alt text]") {
		t.Fatalf("alt text lost: %q", masked)
	}
	if hasRule(detectors.RunClient(masked), "overused-intensifiers") {
		t.Fatalf("overused-intensifiers fired on image URL 'robust.png'")
	}
}

func TestAnalyze_ImageInLink(t *testing.T) {
	src := "[![alt](img.png)](https://x.com/utilize)\n"
	masked, _, _ := Analyze(src)
	assertInvariants(t, src, masked, nil, Range{Start: -1, End: -1})
	if hasRule(detectors.RunClient(masked), "elevated-register") {
		t.Fatalf("elevated-register fired on image-in-link URL")
	}
}

func TestAnalyze_LinkBalancedParens(t *testing.T) {
	// The Wikipedia case — URL contains a literal `)` inside balanced parens.
	src := "See [page](https://en.wikipedia.org/wiki/Markdown_(parser)) now.\n"
	masked, _, _ := Analyze(src)
	assertInvariants(t, src, masked, nil, Range{Start: -1, End: -1})
	if !strings.Contains(masked, "[page]") {
		t.Fatalf("link text lost: %q", masked)
	}
	if !strings.Contains(masked, "now.") {
		t.Fatalf("trailing prose lost: %q", masked)
	}
	// The entire `(https://...)` destination including the inner `)` must be
	// masked — otherwise `now.` would be inside the mask and mangled, which
	// the previous check would catch.
}

func TestAnalyze_LinkAngleDestination(t *testing.T) {
	src := "[label](<https://x.com/has spaces>) end\n"
	masked, _, _ := Analyze(src)
	assertInvariants(t, src, masked, nil, Range{Start: -1, End: -1})
	if !strings.Contains(masked, "end") {
		t.Fatalf("trailing prose lost: %q", masked)
	}
}

func TestAnalyze_LinkTitleMasked(t *testing.T) {
	// The word `robust` lives inside the title "...".
	src := `[x](https://a.b "robust framework") y` + "\n"
	masked, _, _ := Analyze(src)
	assertInvariants(t, src, masked, nil, Range{Start: -1, End: -1})
	if hasRule(detectors.RunClient(masked), "overused-intensifiers") {
		t.Fatalf("overused-intensifiers fired inside link title")
	}
}

func TestAnalyze_Autolink(t *testing.T) {
	src := "see <https://x.com/utilize> now\n"
	masked, _, _ := Analyze(src)
	assertInvariants(t, src, masked, nil, Range{Start: -1, End: -1})
	if hasRule(detectors.RunClient(masked), "elevated-register") {
		t.Fatalf("elevated-register fired inside autolink")
	}
}

func TestAnalyze_ReferenceDefinition(t *testing.T) {
	src := "See [reference][ref] for details.\n\n[ref]: https://x.com/utilize \"robust frame\"\n"
	masked, _, _ := Analyze(src)
	assertInvariants(t, src, masked, nil, Range{Start: -1, End: -1})
	if hasRule(detectors.RunClient(masked), "elevated-register") {
		t.Fatalf("elevated-register fired inside reference definition URL")
	}
	if hasRule(detectors.RunClient(masked), "overused-intensifiers") {
		t.Fatalf("overused-intensifiers fired inside reference definition title")
	}
	// The prose before the ref-def should still be there.
	if !strings.Contains(masked, "See [reference][ref] for details.") {
		t.Fatalf("prose mangled: %q", masked)
	}
}

func TestAnalyze_LinkTextIsProse(t *testing.T) {
	// Link text itself is prose — slop there should still fire.
	src := "Read [utilize this now](https://x.com) today.\n"
	masked, _, _ := Analyze(src)
	assertInvariants(t, src, masked, nil, Range{Start: -1, End: -1})
	if !hasRule(detectors.RunClient(masked), "elevated-register") {
		t.Fatalf("elevated-register did NOT fire on link text 'utilize': %q", masked)
	}
}

// ── HTML ───────────────────────────────────────────────────────────────────

func TestAnalyze_HTMLBlock(t *testing.T) {
	src := "prose before\n\n<div>\n  utilize this\n</div>\n\nprose after\n"
	masked, _, _ := Analyze(src)
	assertInvariants(t, src, masked, nil, Range{Start: -1, End: -1})
	if hasRule(detectors.RunClient(masked), "elevated-register") {
		t.Fatalf("elevated-register fired inside HTML block")
	}
	if !strings.Contains(masked, "prose before") || !strings.Contains(masked, "prose after") {
		t.Fatalf("surrounding prose lost: %q", masked)
	}
}

func TestAnalyze_InlineHTML(t *testing.T) {
	src := "This is <b>important</b> stuff\n"
	masked, _, _ := Analyze(src)
	assertInvariants(t, src, masked, nil, Range{Start: -1, End: -1})
	// Tags `<b>` / `</b>` should be masked but surrounding text preserved.
	if !strings.Contains(masked, "This is") || !strings.Contains(masked, "stuff") {
		t.Fatalf("surrounding prose lost: %q", masked)
	}
	if strings.Contains(masked, "<b>") || strings.Contains(masked, "</b>") {
		t.Fatalf("inline HTML tags not masked: %q", masked)
	}
}

// ── YAML front matter ─────────────────────────────────────────────────────

func TestAnalyze_FrontMatterMasked(t *testing.T) {
	src := "---\ntitle: robust framework\ndate: 2026-01-01\n---\n\nBody text here.\n"
	masked, _, fm := Analyze(src)
	assertInvariants(t, src, masked, nil, fm)
	if fm.Start != 0 {
		t.Fatalf("front matter range not detected: %+v", fm)
	}
	if hasRule(detectors.RunClient(masked), "overused-intensifiers") {
		t.Fatalf("robust in front matter fired overused-intensifiers")
	}
	if !strings.Contains(masked, "Body text here.") {
		t.Fatalf("body after front matter lost: %q", masked)
	}
}

func TestAnalyze_EmptyFrontMatter(t *testing.T) {
	src := "---\n\n---\nBody here\n"
	masked, _, fm := Analyze(src)
	assertInvariants(t, src, masked, nil, fm)
	if !strings.Contains(masked, "Body here") {
		t.Fatalf("body lost: %q", masked)
	}
}

func TestAnalyze_NoFrontMatter(t *testing.T) {
	src := "Regular content with no front matter.\n"
	masked, _, fm := Analyze(src)
	assertInvariants(t, src, masked, nil, fm)
	if fm.Start != -1 {
		t.Fatalf("phantom front matter detected: %+v", fm)
	}
	if masked != src {
		t.Fatalf("plain content was altered:\n src=%q\n got=%q", src, masked)
	}
}

// ── Headings (suppression, not masking) ────────────────────────────────────

func TestAnalyze_ATXHeadingSuppressRange(t *testing.T) {
	src := "intro\n\n## Usage\n\nbody\n"
	masked, suppress, _ := Analyze(src)
	assertInvariants(t, src, masked, suppress, Range{Start: -1, End: -1})
	headingIdx := strings.Index(src, "## Usage")
	var found bool
	for _, r := range suppress {
		if r.Kind == KindHeading && r.Start <= headingIdx && r.End >= headingIdx+len("## Usage") {
			found = true
		}
	}
	if !found {
		t.Fatalf("no KindHeading range overlaps '## Usage'. suppress=%+v", suppress)
	}
}

func TestAnalyze_SetextHeadingSuppressRange(t *testing.T) {
	src := "before\n\nUsage\n=====\n\nbody\n"
	masked, suppress, _ := Analyze(src)
	assertInvariants(t, src, masked, suppress, Range{Start: -1, End: -1})
	titleIdx := strings.Index(src, "Usage\n=====")
	titleEnd := titleIdx + len("Usage\n=====")
	var found bool
	for _, r := range suppress {
		if r.Kind == KindHeading && r.Start <= titleIdx && r.End >= titleEnd {
			found = true
		}
	}
	if !found {
		t.Fatalf("setext heading range missing. suppress=%+v", suppress)
	}
}

func TestAnalyze_HeadingContainingSlop(t *testing.T) {
	// Heading text is NOT masked; we only collect it as a suppress range.
	src := "## The robust tapestry\n\nbody\n"
	masked, suppress, _ := Analyze(src)
	assertInvariants(t, src, masked, suppress, Range{Start: -1, End: -1})
	if masked != src {
		t.Fatalf("heading bytes unexpectedly masked")
	}
	if !hasRule(detectors.RunClient(masked), "overused-intensifiers") {
		t.Fatalf("overused-intensifiers did not fire on heading text (it should; dramatic-fragment suppression happens in the CLI post-filter)")
	}
	// Heading range exists.
	var found bool
	for _, r := range suppress {
		if r.Kind == KindHeading {
			found = true
		}
	}
	if !found {
		t.Fatalf("no KindHeading range reported")
	}
}

func TestAnalyze_HeadingInBlockquote(t *testing.T) {
	src := "> # Subsection\n>\n> body\n"
	masked, suppress, _ := Analyze(src)
	assertInvariants(t, src, masked, suppress, Range{Start: -1, End: -1})
	var found bool
	for _, r := range suppress {
		if r.Kind == KindHeading {
			found = true
		}
	}
	if !found {
		t.Fatalf("heading inside blockquote not reported. suppress=%+v", suppress)
	}
}

// ── Lists (suppression) ────────────────────────────────────────────────────

func TestAnalyze_UnorderedListItems(t *testing.T) {
	src := "- Foo.\n- Bar.\n- Baz.\n"
	masked, suppress, _ := Analyze(src)
	assertInvariants(t, src, masked, suppress, Range{Start: -1, End: -1})
	var count int
	for _, r := range suppress {
		if r.Kind == KindListItem {
			count++
		}
	}
	if count != 3 {
		t.Fatalf("expected 3 KindListItem, got %d. suppress=%+v", count, suppress)
	}
}

func TestAnalyze_NestedOrderedList(t *testing.T) {
	src := "1. outer\n   1. inner\n   2. inner2\n2. outer2\n"
	masked, suppress, _ := Analyze(src)
	assertInvariants(t, src, masked, suppress, Range{Start: -1, End: -1})
	var count int
	for _, r := range suppress {
		if r.Kind == KindListItem {
			count++
		}
	}
	if count < 4 {
		t.Fatalf("expected >=4 KindListItem across nested list, got %d. suppress=%+v", count, suppress)
	}
}

// ── Plain-text no-op ───────────────────────────────────────────────────────

func TestAnalyze_PlainProseNoOp(t *testing.T) {
	src := "This is ordinary prose with no markdown features at all. Just sentences.\n"
	masked, suppress, fm := Analyze(src)
	assertInvariants(t, src, masked, suppress, fm)
	if masked != src {
		t.Fatalf("plain prose was altered:\n src=%q\n got=%q", src, masked)
	}
	if fm.Start != -1 {
		t.Fatalf("phantom front matter")
	}
}

func TestAnalyze_Empty(t *testing.T) {
	masked, suppress, fm := Analyze("")
	assertInvariants(t, "", masked, suppress, fm)
}

// ── Overlaps / CountOverlapping helpers ────────────────────────────────────

func TestOverlapsAndCount(t *testing.T) {
	spans := []Range{
		{Start: 0, End: 10, Kind: KindListItem},
		{Start: 10, End: 20, Kind: KindListItem},
		{Start: 30, End: 40, Kind: KindHeading},
	}
	if !Overlaps(5, 15, spans, KindListItem) {
		t.Fatalf("Overlaps missed [5,15) vs list items")
	}
	if Overlaps(30, 40, spans, KindListItem) {
		t.Fatalf("Overlaps false-positive on wrong kind")
	}
	if got := CountOverlapping(5, 15, spans, KindListItem); got != 2 {
		t.Fatalf("CountOverlapping [5,15) = %d, want 2", got)
	}
}

// ── Helpers ────────────────────────────────────────────────────────────────

func hasRule(vs []types.Violation, ruleID string) bool {
	for _, v := range vs {
		if v.RuleID == ruleID {
			return true
		}
	}
	return false
}
