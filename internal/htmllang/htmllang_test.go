package htmllang_test

import (
	"strings"
	"testing"

	"github.com/yasyf/slop-cop/internal/detectors"
	"github.com/yasyf/slop-cop/internal/htmllang"
	"github.com/yasyf/slop-cop/internal/lang"
	"github.com/yasyf/slop-cop/internal/types"
)

func TestAnalyzeInvariantsSimple(t *testing.T) {
	src := `<!doctype html>
<html>
  <head><title>Hi</title></head>
  <body>
    <h1>Header</h1>
    <p>Hello <em>world</em>, this is prose.</p>
    <ul>
      <li>one</li>
      <li>two</li>
    </ul>
    <script>var x = "code string in script";</script>
  </body>
</html>`
	lang.AssertAnalyzeInvariants(t, htmllang.Analyzer{}, src)
}

func TestProseKeptTagsMasked(t *testing.T) {
	src := `<p class="greeting">Hello world.</p>`
	masked, _, err := htmllang.Analyzer{}.Analyze(src)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(masked, "Hello world.") {
		t.Fatalf("masked should keep element text; got %q", masked)
	}
	if strings.Contains(masked, "class") {
		t.Fatalf("masked should erase attributes; got %q", masked)
	}
	if strings.Contains(masked, "greeting") {
		t.Fatalf("masked should erase attribute values; got %q", masked)
	}
}

func TestScriptStyleBodiesMasked(t *testing.T) {
	src := `<script>const s = "secret prose";</script><style>body { color: red; }</style><p>Visible prose.</p>`
	masked, _, err := htmllang.Analyzer{}.Analyze(src)
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(masked, "secret prose") {
		t.Fatalf("script body should be masked; got %q", masked)
	}
	if strings.Contains(masked, "color: red") {
		t.Fatalf("style body should be masked; got %q", masked)
	}
	if !strings.Contains(masked, "Visible prose.") {
		t.Fatalf("plain paragraph text should survive; got %q", masked)
	}
}

func TestSuppressRangesHeadingAndList(t *testing.T) {
	src := "<h1>Title here</h1><ul><li>alpha</li><li>beta</li></ul>"
	_, suppress, err := htmllang.Analyzer{}.Analyze(src)
	if err != nil {
		t.Fatal(err)
	}
	var seenHeading, seenLI int
	for _, r := range suppress {
		switch r.Kind {
		case lang.KindHeading:
			seenHeading++
		case lang.KindListItem:
			seenLI++
		}
	}
	if seenHeading == 0 {
		t.Fatalf("expected a heading suppress range; got %+v", suppress)
	}
	if seenLI < 2 {
		t.Fatalf("expected at least 2 list-item suppress ranges; got %+v", suppress)
	}
}

func TestApplySuppressionsDropsHeadingDramaticFragment(t *testing.T) {
	src := "<h1>Wow.</h1>"
	_, suppress, _ := htmllang.Analyzer{}.Analyze(src)
	vs := []types.Violation{
		{RuleID: "dramatic-fragment", StartIndex: 4, EndIndex: 8, MatchedText: "Wow."},
	}
	out := htmllang.Analyzer{}.ApplySuppressions(vs, suppress, src)
	if len(out) != 0 {
		t.Fatalf("dramatic-fragment inside heading should be suppressed; got %+v", out)
	}
}

func TestApplySuppressionsDropsLengthRulesInHeading(t *testing.T) {
	src := "<h1>A heading carrying far more words than any reasonable prose sentence would ever need to carry</h1>"
	_, suppress, _ := htmllang.Analyzer{}.Analyze(src)
	start := strings.Index(src, "A heading")
	end := strings.Index(src, "</h1>")
	vs := []types.Violation{
		{RuleID: "long-sentence", StartIndex: start, EndIndex: end},
		{RuleID: "long-paragraph", StartIndex: start, EndIndex: end},
	}
	out := htmllang.Analyzer{}.ApplySuppressions(vs, suppress, src)
	if len(out) != 0 {
		t.Fatalf("both length rules inside a heading should be suppressed; got %+v", out)
	}
}

func TestApplySuppressionsDropsLongParagraphAcrossListItems(t *testing.T) {
	src := "<ul><li>alpha runs.</li><li>beta runs.</li><li>gamma runs.</li></ul>"
	_, suppress, _ := htmllang.Analyzer{}.Analyze(src)
	vs := []types.Violation{
		{RuleID: "long-paragraph", StartIndex: 0, EndIndex: len(src)},
	}
	out := htmllang.Analyzer{}.ApplySuppressions(vs, suppress, src)
	if len(out) != 0 {
		t.Fatalf("long-paragraph across list items should be suppressed; got %+v", out)
	}
}

// TestApplySuppressionsKeepsLengthRulesInProse is the narrowness proof: the
// document carries a heading and a list, so both suppression ranges are
// live, yet length hits on a paragraph touching neither still stand. The
// violations come from the base detectors rather than by hand, so the spans
// are the ones the pipeline really produces.
func TestApplySuppressionsKeepsLengthRulesInProse(t *testing.T) {
	prose := "The build runs first and then the tests run next and after that the linter " +
		"follows along behind them while the report lands last and none of this is a list " +
		"and none of this is a heading and the paragraph simply sprawls onward, which is " +
		"the whole point of it. The tests run next. The linter follows after that. The " +
		"report lands last. None of this is a list. None of this is a heading. The " +
		"paragraph simply sprawls onward. That is the rule's whole point."
	src := "<h1>Pipeline</h1>\n<ul><li>one</li><li>two</li></ul>\n\n<p>" + prose + "</p>\n"
	masked, suppress, _ := htmllang.Analyzer{}.Analyze(src)
	vs := detectors.RunBase(masked)
	if len(vs) != 2 {
		t.Fatalf("fixture should trip long-sentence and long-paragraph; got %+v", vs)
	}
	out := htmllang.Analyzer{}.ApplySuppressions(vs, suppress, src)
	if len(out) != 2 {
		t.Fatalf("length rules on body prose should survive; got %+v", out)
	}
}

func TestApplySuppressionsRestoresMatchedText(t *testing.T) {
	src := "<p>Hello, world.</p>"
	_, suppress, _ := htmllang.Analyzer{}.Analyze(src)
	vs := []types.Violation{
		{RuleID: "some-rule", StartIndex: 3, EndIndex: 15, MatchedText: "            "},
	}
	out := htmllang.Analyzer{}.ApplySuppressions(vs, suppress, src)
	if len(out) != 1 {
		t.Fatalf("expected 1 violation, got %d", len(out))
	}
	if out[0].MatchedText != "Hello, world" {
		t.Fatalf("MatchedText = %q, want %q", out[0].MatchedText, "Hello, world")
	}
}

func TestRegistryLookup(t *testing.T) {
	a, ok := lang.ByExtension(".html")
	if !ok || a == nil || a.Name() != "html" {
		t.Fatalf("ByExtension .html = (%v, %v)", a, ok)
	}
	a, ok = lang.ByName("html")
	if !ok || a == nil {
		t.Fatalf("ByName html = (%v, %v)", a, ok)
	}
}
