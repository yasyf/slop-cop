package main

import (
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// writeConfig drops a .slopcop.toml into dir and returns its path.
func writeConfig(t *testing.T, dir, body string) string {
	t.Helper()
	file := filepath.Join(dir, configName)
	if err := os.WriteFile(file, []byte(body), 0o600); err != nil {
		t.Fatal(err)
	}
	return file
}

// writeInput drops a fixture at rel under dir, creating parents.
func writeInput(t *testing.T, dir, rel, body string) string {
	t.Helper()
	file := filepath.Join(dir, rel)
	if err := os.MkdirAll(filepath.Dir(file), 0o750); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(file, []byte(body), 0o600); err != nil {
		t.Fatal(err)
	}
	return file
}

func TestResolveRuleFilterNoConfig(t *testing.T) {
	dir := t.TempDir()
	input := writeInput(t, dir, "doc.md", "text\n")
	filter, file, err := resolveRuleFilter("", input)
	if err != nil {
		t.Fatalf("resolveRuleFilter: %v", err)
	}
	if file != "" {
		t.Fatalf("config file = %q, want none", file)
	}
	if !filter.empty() || !filter.allows("colon-elaboration") {
		t.Fatalf("absent config produced a filter: %+v", filter)
	}
}

func TestResolveRuleFilterWalksUp(t *testing.T) {
	dir := t.TempDir()
	want := writeConfig(t, dir, "disable = [\"colon-elaboration\"]\n")
	input := writeInput(t, dir, "docs/guides/doc.md", "text\n")

	filter, file, err := resolveRuleFilter("", input)
	if err != nil {
		t.Fatalf("resolveRuleFilter: %v", err)
	}
	if file != want {
		t.Fatalf("config file = %q, want %q", file, want)
	}
	if filter.allows("colon-elaboration") {
		t.Fatal("disabled rule still allowed")
	}
	if !filter.allows("em-dash-pivot") {
		t.Fatal("an unlisted rule was dropped")
	}
}

func TestResolveRuleFilterEnableOnly(t *testing.T) {
	dir := t.TempDir()
	writeConfig(t, dir, "enable_only = [\"em-dash-pivot\"]\n")
	input := writeInput(t, dir, "doc.md", "text\n")

	filter, _, err := resolveRuleFilter("", input)
	if err != nil {
		t.Fatalf("resolveRuleFilter: %v", err)
	}
	if !filter.allows("em-dash-pivot") {
		t.Fatal("enable_only dropped the rule it names")
	}
	if filter.allows("colon-elaboration") {
		t.Fatal("enable_only kept a rule it does not name")
	}
}

// TestResolveRuleFilterOverrides pins the house-style dismissals the config
// exists to express: a PR-body template gets colon-elaboration and
// listicle-instinct off, and only there.
func TestResolveRuleFilterOverrides(t *testing.T) {
	dir := t.TempDir()
	writeConfig(t, dir, `disable = ["em-dash-pivot"]

[[overrides]]
paths = ["docs/**/*.md", "PR_BODY.md"]
disable = ["colon-elaboration", "listicle-instinct"]
`)
	cases := []struct {
		rel          string
		wantOverride bool
	}{
		{"docs/guides/style.md", true},
		{"docs/style.md", true},
		{"PR_BODY.md", true},
		{"README.md", false},
		{"docsx/style.md", false},
		{"docs/style.txt", false},
	}
	for _, c := range cases {
		t.Run(c.rel, func(t *testing.T) {
			input := writeInput(t, dir, c.rel, "text\n")
			filter, _, err := resolveRuleFilter("", input)
			if err != nil {
				t.Fatalf("resolveRuleFilter: %v", err)
			}
			if got := !filter.allows("colon-elaboration"); got != c.wantOverride {
				t.Fatalf("colon-elaboration disabled = %v, want %v", got, c.wantOverride)
			}
			if filter.allows("em-dash-pivot") {
				t.Fatal("the top-level disable stopped applying under an override")
			}
		})
	}
}

func TestResolveRuleFilterOverrideEnableOnlyWins(t *testing.T) {
	dir := t.TempDir()
	writeConfig(t, dir, `enable_only = ["em-dash-pivot"]

[[overrides]]
paths = ["CHANGELOG.md"]
enable_only = ["colon-elaboration"]
`)
	input := writeInput(t, dir, "CHANGELOG.md", "text\n")
	filter, _, err := resolveRuleFilter("", input)
	if err != nil {
		t.Fatalf("resolveRuleFilter: %v", err)
	}
	if !filter.allows("colon-elaboration") || filter.allows("em-dash-pivot") {
		t.Fatalf("the override's enable_only did not replace the top-level one: %+v", filter)
	}
}

func TestResolveRuleFilterExplicitConfig(t *testing.T) {
	dir, other := t.TempDir(), t.TempDir()
	writeConfig(t, dir, "disable = [\"colon-elaboration\"]\n")
	explicit := writeConfig(t, other, "disable = [\"em-dash-pivot\"]\n")
	input := writeInput(t, dir, "doc.md", "text\n")

	filter, file, err := resolveRuleFilter(explicit, input)
	if err != nil {
		t.Fatalf("resolveRuleFilter: %v", err)
	}
	if file != explicit {
		t.Fatalf("config file = %q, want %q", file, explicit)
	}
	if filter.allows("em-dash-pivot") || !filter.allows("colon-elaboration") {
		t.Fatalf("--config did not override discovery: %+v", filter)
	}
}

func TestResolveRuleFilterUnknownRuleIsUsageError(t *testing.T) {
	dir := t.TempDir()
	writeConfig(t, dir, "disable = [\"colon-elaboration\", \"no-such-rule\"]\n")
	input := writeInput(t, dir, "doc.md", "text\n")

	_, _, err := resolveRuleFilter("", input)
	if err == nil {
		t.Fatal("expected an error for an unknown rule ID")
	}
	if !errors.As(err, new(usageError)) {
		t.Fatalf("error %v is not a usageError; main() would not map it to exit %d", err, exitUsage)
	}
	if !strings.Contains(err.Error(), "no-such-rule") {
		t.Fatalf("error %q does not name the offender", err)
	}
}

func TestResolveRuleFilterUnknownKeyIsUsageError(t *testing.T) {
	dir := t.TempDir()
	writeConfig(t, dir, "disabled = [\"colon-elaboration\"]\n")
	input := writeInput(t, dir, "doc.md", "text\n")

	_, _, err := resolveRuleFilter("", input)
	if !errors.As(err, new(usageError)) {
		t.Fatalf("error %v is not a usageError", err)
	}
}

func TestResolveRuleFilterMissingExplicitConfigIsIOError(t *testing.T) {
	_, _, err := resolveRuleFilter(filepath.Join(t.TempDir(), "absent.toml"), "")
	if err == nil {
		t.Fatal("expected an error for a missing --config path")
	}
	if errors.As(err, new(usageError)) {
		t.Fatalf("a missing --config file should not be a usage error: %v", err)
	}
}

func TestCompileGlob(t *testing.T) {
	cases := []struct {
		pattern string
		path    string
		want    bool
	}{
		{"*.md", "README.md", true},
		{"*.md", "docs/README.md", false},
		{"docs/**/*.md", "docs/a/b/c.md", true},
		{"docs/**/*.md", "docs/c.md", true},
		{"docs/**", "docs/a/b.md", true},
		{"docs/*.md", "docs/a/b.md", false},
		{"?.md", "a.md", true},
		{"?.md", "ab.md", false},
		{"a.b", "axb", false},
	}
	for _, c := range cases {
		re, err := compileGlob(c.pattern)
		if err != nil {
			t.Fatalf("compileGlob(%q): %v", c.pattern, err)
		}
		if got := re.MatchString(c.path); got != c.want {
			t.Fatalf("compileGlob(%q).MatchString(%q) = %v, want %v", c.pattern, c.path, got, c.want)
		}
	}
}
