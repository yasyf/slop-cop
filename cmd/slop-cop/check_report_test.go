package main

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/yasyf/slop-cop/internal/rules"
)

// TestExitCodeMatrix pins the documented exit codes, including the --strict
// signal a caller reads to tell a clean pass from a report full of violations.
func TestExitCodeMatrix(t *testing.T) {
	cases := []struct {
		name string
		err  error
		want int
	}{
		{"clean run", nil, exitOK},
		{"violations found under --strict", errViolationsFound, exitViolations},
		{"violations found, wrapped", fmt.Errorf("check: %w", errViolationsFound), exitViolations},
		{"IO failure", errors.New("reading article.md: no such file"), exitIO},
		{"LLM failure", llmError{err: errors.New("sentence pass")}, exitLLM},
		{"usage failure", usageError{err: errors.New("invalid --standard")}, exitUsage},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			if got := exitCodeFor(c.err); got != c.want {
				t.Fatalf("exitCodeFor(%v) = %d, want %d", c.err, got, c.want)
			}
		})
	}
}

// TestViolationsExitIsOptIn pins the decision that a completed run exits 0
// whatever it found, unless --strict asks otherwise.
func TestViolationsExitIsOptIn(t *testing.T) {
	dirty := writeInput(t, t.TempDir(), "dirty.md", "It's important to note that we must delve into this.\n")

	out, err := runCheck(t, dirty, "--llm-effort=off")
	if err != nil {
		t.Fatalf("violations without --strict returned %v, want nil", err)
	}
	if len(decodeReport(t, out).CountsByRule) == 0 {
		t.Fatal("fixture produced no violations, so the case proves nothing")
	}

	out, err = runCheck(t, dirty, "--llm-effort=off", "--strict")
	if !errors.Is(err, errViolationsFound) {
		t.Fatalf("violations with --strict returned %v, want errViolationsFound", err)
	}
	if len(decodeReport(t, out).CountsByRule) == 0 {
		t.Fatal("errViolationsFound with an empty counts_by_rule")
	}

	clean := writeInput(t, t.TempDir(), "clean.md", "The build runs on every push.\n")
	out, err = runCheck(t, clean, "--llm-effort=off", "--strict")
	if err != nil {
		t.Fatalf("a clean document under --strict returned %v, want nil", err)
	}
	if n := len(decodeReport(t, out).CountsByRule); n != 0 {
		t.Fatalf("a clean run reported %d rules", n)
	}
}

// TestReportCarriesRunMarker checks the field that keeps a truncated or empty
// payload from parsing as a clean pass.
func TestReportCarriesRunMarker(t *testing.T) {
	out := checkOK(t, filepath.Join("testdata", "mixed.md"), "--llm-effort=off")

	var probe struct {
		Ran *bool `json:"ran"`
	}
	if err := json.Unmarshal([]byte(out), &probe); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if probe.Ran == nil || !*probe.Ran {
		t.Fatalf(`report has no "ran": true marker: %s`, out)
	}

	if err := json.Unmarshal([]byte(""), &probe); err == nil {
		t.Fatal("empty stdout decoded without error; it must not read as a report")
	}
	probe.Ran = nil
	if err := json.Unmarshal([]byte(`{"violations":[],"counts_by_rule":{}}`), &probe); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if probe.Ran != nil {
		t.Fatal("a payload without the marker still presented one")
	}
}

// TestReportCarriesBuildIdentity pins the fields that resolve a rule-ID
// mismatch against the binary that produced the report.
func TestReportCarriesBuildIdentity(t *testing.T) {
	rep := decodeReport(t, checkOK(t, filepath.Join("testdata", "mixed.md"), "--llm-effort=off"))
	wantVersion, _ := buildMetadata()
	if rep.Version != wantVersion {
		t.Fatalf("version = %q, want %q", rep.Version, wantVersion)
	}
	wantBinary, err := os.Executable()
	if err != nil {
		t.Fatal(err)
	}
	if rep.BinaryPath != wantBinary {
		t.Fatalf("binary_path = %q, want %q", rep.BinaryPath, wantBinary)
	}
}

// TestReportCarriesRuleSidecar checks the fix guidance rides along once per
// distinct rule rather than once per violation.
func TestReportCarriesRuleSidecar(t *testing.T) {
	rep := decodeReport(t, checkOK(t, filepath.Join("testdata", "mixed.md"), "--llm-effort=off"))
	if len(rep.Rules) != len(rep.CountsByRule) {
		t.Fatalf("rules has %d entries, counts_by_rule has %d", len(rep.Rules), len(rep.CountsByRule))
	}
	for id := range rep.CountsByRule {
		entry, ok := rep.Rules[id]
		if !ok {
			t.Fatalf("no rules entry for %s", id)
		}
		if entry.Name == "" || entry.Tip == "" {
			t.Fatalf("rules[%s] carries no guidance: %+v", id, entry)
		}
	}
}

// TestConfigFiltersTheReport is the end-to-end wiring check: a disabled rule
// leaves the report entirely, and the config that did it is named.
func TestConfigFiltersTheReport(t *testing.T) {
	dir := t.TempDir()
	body := "It's important to note that we must delve into this.\n"
	input := writeInput(t, dir, "doc.md", body)

	before := decodeReport(t, checkOK(t, input, "--llm-effort=off"))
	victim := ""
	for id := range before.CountsByRule {
		if victim == "" || id < victim {
			victim = id
		}
	}
	if victim == "" {
		t.Fatal("fixture produced no violations to suppress")
	}

	writeConfig(t, dir, fmt.Sprintf("disable = [%q]\n", victim))
	after := decodeReport(t, checkOK(t, input, "--llm-effort=off"))
	if _, ok := after.CountsByRule[victim]; ok {
		t.Fatalf("%s survived being disabled", victim)
	}
	if _, ok := after.Rules[victim]; ok {
		t.Fatalf("%s survived in the rules sidecar", victim)
	}
	if after.Config != filepath.Join(dir, configName) {
		t.Fatalf("config = %q, want the file that filtered the run", after.Config)
	}
}

// TestConfigUnknownRuleExitsUsage checks a typo'd rule ID fails loudly rather
// than silently suppressing nothing.
func TestConfigUnknownRuleExitsUsage(t *testing.T) {
	dir := t.TempDir()
	input := writeInput(t, dir, "doc.md", "text\n")
	writeConfig(t, dir, "disable = [\"colon-elaborationn\"]\n")

	out, err := runCheck(t, input, "--llm-effort=off")
	if got := exitCodeFor(err); got != exitUsage {
		t.Fatalf("exit %d, want %d (%v)", got, exitUsage, err)
	}
	if out != "" {
		t.Fatalf("a rejected run still wrote a report: %s", out)
	}
}

// TestFormatCompact pins the projection agents were rebuilding by hand: one
// tab-separated line per violation, then a counts line.
func TestFormatCompact(t *testing.T) {
	input := writeInput(t, t.TempDir(), "doc.md", "Ship it — fast.\nIt's important to note that we must delve into this.\n")
	out, err := runCheck(t, input, "--llm-effort=off", "--format=compact")
	if err != nil && !errors.Is(err, errViolationsFound) {
		t.Fatalf("check: %v", err)
	}

	lines := strings.Split(strings.TrimSuffix(out, "\n"), "\n")
	if len(lines) < 2 {
		t.Fatalf("compact output has %d lines: %q", len(lines), out)
	}
	counts := lines[len(lines)-1]
	if !strings.HasPrefix(counts, "counts\t") {
		t.Fatalf("last line is not the counts line: %q", counts)
	}
	for _, line := range lines[:len(lines)-1] {
		fields := strings.Split(line, "\t")
		if len(fields) != 3 {
			t.Fatalf("violation line has %d fields, want 3: %q", len(fields), line)
		}
		if _, ok := rules.ByID[fields[0]]; !ok {
			t.Fatalf("first field %q is not a rule ID", fields[0])
		}
		if !strings.Contains(fields[1], ":") {
			t.Fatalf("second field %q is not line:col", fields[1])
		}
	}
}

// TestLineColumn checks byte offsets convert with UTF-8 in mind: the em dash
// is one column, not three.
func TestLineColumn(t *testing.T) {
	text := "héllo wörld\nsecond — line\n"
	cases := []struct {
		offset    int
		line, col int
	}{
		{0, 1, 1},
		{len("héllo "), 1, 7},
		{len("héllo wörld\n"), 2, 1},
		{len("héllo wörld\nsecond — "), 2, 10},
	}
	for _, c := range cases {
		line, col := lineColumn(text, c.offset)
		if line != c.line || col != c.col {
			t.Fatalf("lineColumn(%d) = %d:%d, want %d:%d", c.offset, line, col, c.line, c.col)
		}
	}
}

func TestResolveFormat(t *testing.T) {
	for raw, want := range map[string]outputFormat{"": formatJSON, "json": formatJSON, "JSON": formatJSON, "compact": formatCompact} {
		got, err := resolveFormat(raw)
		if err != nil {
			t.Fatalf("resolveFormat(%q): %v", raw, err)
		}
		if got != want {
			t.Fatalf("resolveFormat(%q) = %q, want %q", raw, got, want)
		}
	}
	if _, err := resolveFormat("tsv"); err == nil {
		t.Fatal("expected an error for --format=tsv")
	}
}
