package jslang_test

import (
	"strings"
	"testing"

	_ "github.com/yasyf/slop-cop/internal/jslang"
	"github.com/yasyf/slop-cop/internal/lang"
	"github.com/yasyf/slop-cop/internal/types"
)

func analyzer(t *testing.T, name string) lang.Analyzer {
	t.Helper()
	a, ok := lang.ByName(name)
	if !ok || a == nil {
		t.Fatalf("lang.ByName(%q) not registered", name)
	}
	return a
}

func TestInvariantsJS(t *testing.T) {
	a := analyzer(t, "js")
	lang.AssertAnalyzeInvariants(t, a, `
// A short comment.
const greeting = "Hello, world.";
const multi = \u0060tagged ${greeting} template\u0060;
function foo(x) { return x / 2; }
const re = /abc/g;
`)
}

func TestCommentAndStringProseKeptJS(t *testing.T) {
	src := `// It's important to note this comment.
const s = "Hello, dear reader.";
const x = 42;`
	a := analyzer(t, "js")
	masked, _, err := a.Analyze(src)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(masked, "It's important to note this comment.") {
		t.Fatalf("line comment text should survive; masked=%q", masked)
	}
	if !strings.Contains(masked, "Hello, dear reader.") {
		t.Fatalf("string literal text should survive; masked=%q", masked)
	}
	if strings.Contains(masked, "const") {
		t.Fatalf("keyword 'const' should be masked; masked=%q", masked)
	}
	if strings.Contains(masked, "42") {
		t.Fatalf("number literal should be masked; masked=%q", masked)
	}
}

func TestJSDocDetectedAsJSDocKind(t *testing.T) {
	src := `/**
 * Greets the user.
 * @param name The user's name.
 */
function greet(name) {}`
	a := analyzer(t, "js")
	_, suppress, err := a.Analyze(src)
	if err != nil {
		t.Fatal(err)
	}
	foundJSDoc := false
	for _, r := range suppress {
		if r.Kind == lang.KindJSDoc {
			foundJSDoc = true
			break
		}
	}
	if !foundJSDoc {
		t.Fatalf("expected KindJSDoc suppress range; got %+v", suppress)
	}
}

func TestJSXText(t *testing.T) {
	src := `const Foo = () => <div>Hello from JSX.</div>;`
	a := analyzer(t, "jsx")
	masked, _, err := a.Analyze(src)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(masked, "Hello from JSX.") {
		t.Fatalf("JSX text should survive; masked=%q", masked)
	}
	if strings.Contains(masked, "<div>") {
		t.Fatalf("JSX tag should be masked; masked=%q", masked)
	}
}

func TestTemplateLiteralQuasi(t *testing.T) {
	src := "const m = `Hello, ${name}, how are you today?`;"
	a := analyzer(t, "js")
	masked, _, err := a.Analyze(src)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(masked, "Hello, ") {
		t.Fatalf("template quasi prefix should survive; masked=%q", masked)
	}
	if !strings.Contains(masked, ", how are you today?") {
		t.Fatalf("template quasi suffix should survive; masked=%q", masked)
	}
	if strings.Contains(masked, "${name}") {
		t.Fatalf("template substitution should be masked; masked=%q", masked)
	}
}

func TestTSGenericsNotTreatedAsJSX(t *testing.T) {
	src := `function identity<T>(x: T): T { return x; }
const arr: Array<string> = [];`
	a := analyzer(t, "ts")
	lang.AssertAnalyzeInvariants(t, a, src)
	// No prose content → masked should be all spaces/newlines.
	masked, _, _ := a.Analyze(src)
	for i, b := range []byte(masked) {
		if b != ' ' && b != '\n' {
			t.Fatalf("offset %d: expected space/newline, got %q (masked=%q)", i, b, masked)
		}
	}
}

func TestTSXPreservesJSXText(t *testing.T) {
	src := `const C = () => <span>Some prose.</span>;`
	a := analyzer(t, "tsx")
	masked, _, err := a.Analyze(src)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(masked, "Some prose.") {
		t.Fatalf("TSX jsx_text should survive; masked=%q", masked)
	}
}

func TestRegistryLookup(t *testing.T) {
	for _, name := range []string{"js", "jsx", "ts", "tsx"} {
		if a, ok := lang.ByName(name); !ok || a == nil {
			t.Fatalf("lang.ByName(%q) missing", name)
		}
	}
	for ext, want := range map[string]string{
		".js":  "js",
		".mjs": "js",
		".cjs": "js",
		".jsx": "jsx",
		".ts":  "ts",
		".mts": "ts",
		".cts": "ts",
		".tsx": "tsx",
	} {
		a, ok := lang.ByExtension(ext)
		if !ok || a == nil || a.Name() != want {
			t.Fatalf("ByExtension(%q) = (%v, %v); want name %q", ext, a, ok, want)
		}
	}
}

func TestBrokenInputDoesNotPanic(t *testing.T) {
	// Deliberately malformed: unclosed template, unclosed tag.
	src := "function broken( { const x = `foo ${ <Bar>"
	a := analyzer(t, "tsx")
	masked, _, err := a.Analyze(src)
	if err != nil {
		t.Fatalf("broken input should parse with ERROR nodes, not error: %v", err)
	}
	if len(masked) != len(src) {
		t.Fatalf("length invariant broken: %d vs %d", len(masked), len(src))
	}
}

func TestApplySuppressionsRestoresMatchedText(t *testing.T) {
	src := `// Hello there.`
	a := analyzer(t, "js")
	_, suppress, _ := a.Analyze(src)
	vs := []types.Violation{
		{RuleID: "some-rule", StartIndex: 3, EndIndex: 15, MatchedText: "            "},
	}
	out := a.ApplySuppressions(vs, suppress, src)
	if len(out) != 1 {
		t.Fatalf("expected 1 violation, got %d", len(out))
	}
	if out[0].MatchedText != "Hello there." {
		t.Fatalf("MatchedText = %q, want %q", out[0].MatchedText, "Hello there.")
	}
}

func TestDramaticFragmentInJSDocSuppressed(t *testing.T) {
	src := `/**
 * @param name
 */
function f(name) {}`
	a := analyzer(t, "js")
	_, suppress, _ := a.Analyze(src)
	// Find a jsdoc range and build a violation inside it.
	var jsdoc lang.Range
	for _, r := range suppress {
		if r.Kind == lang.KindJSDoc {
			jsdoc = r
			break
		}
	}
	if jsdoc.End == 0 {
		t.Fatal("no JSDoc range found")
	}
	vs := []types.Violation{{RuleID: "dramatic-fragment", StartIndex: jsdoc.Start, EndIndex: jsdoc.End}}
	out := a.ApplySuppressions(vs, suppress, src)
	if len(out) != 0 {
		t.Fatalf("dramatic-fragment in JSDoc should be suppressed; got %+v", out)
	}
}
