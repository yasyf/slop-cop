package main

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"testing"

	"github.com/yasyf/slop-cop/internal/readability"
	"github.com/yasyf/slop-cop/internal/rules"
	"github.com/yasyf/slop-cop/internal/types"
)

// runCheck executes the check command in-process and returns the raw bytes it
// wrote to stdout. writeJSON encodes straight to os.Stdout, so the fd is
// redirected to a temp file for the duration of the call.
func runCheck(t *testing.T, args ...string) (string, error) {
	t.Helper()
	f, err := os.CreateTemp(t.TempDir(), "stdout")
	if err != nil {
		t.Fatal(err)
	}
	orig := os.Stdout
	os.Stdout = f
	cmd := newCheckCmd()
	cmd.SilenceUsage = true
	cmd.SilenceErrors = true
	cmd.SetArgs(args)
	runErr := cmd.Execute()
	os.Stdout = orig
	if err := f.Close(); err != nil {
		t.Fatal(err)
	}
	out, err := os.ReadFile(f.Name())
	if err != nil {
		t.Fatal(err)
	}
	return string(out), runErr
}

// checkOK runs check and fails on anything but a clean run or the
// violations-found signal, which is not an error.
func checkOK(t *testing.T, args ...string) string {
	t.Helper()
	out, err := runCheck(t, args...)
	if err != nil && !errors.Is(err, errViolationsFound) {
		t.Fatalf("check: %v", err)
	}
	return out
}

func decodeReport(t *testing.T, out string) checkReport {
	t.Helper()
	var rep checkReport
	if err := json.Unmarshal([]byte(out), &rep); err != nil {
		t.Fatalf("decode report: %v\n%s", err, out)
	}
	return rep
}

// TestStandardSlopMatchesPreBaseLayerGoldens is the release gate for the whole
// base-layer change. The goldens are the reports the pre-base-layer binary
// emitted for these exact fixture bytes, so a byte match proves --standard=slop
// reproduces the old tool's output — same violations, same offsets, same
// counts, same field order. A drift here means the base layer leaked into the
// slop layer; regenerating the goldens defeats the gate. The sanctioned
// regenerations so far dropped the six hits that only matched the spaces
// masking left behind (lang.DropMaskMatches), the nine that sat on list
// structure (lang.Suppress), and one staccato-burst that was a table's rows.
func TestStandardSlopMatchesPreBaseLayerGoldens(t *testing.T) {
	cases := []struct {
		fixture string
		golden  string
		want    int
	}{
		{"readme.md", "readme.slop.golden.json", 20},
		{"skill.md", "skill.slop.golden.json", 38},
	}
	for _, c := range cases {
		t.Run(c.fixture, func(t *testing.T) {
			want, err := os.ReadFile(filepath.Join("testdata", c.golden))
			if err != nil {
				t.Fatal(err)
			}
			got := legacyProjection(t, checkOK(t, filepath.Join("testdata", c.fixture), "--llm-effort=off", "--standard=slop"))
			if got != string(want) {
				t.Fatalf("--standard=slop drifted from %s at %s", c.golden, firstDiff(string(want), got))
			}
			if n := len(decodeReport(t, got).Violations); n != c.want {
				t.Fatalf("golden carries %d violations, want %d", n, c.want)
			}
		})
	}
}

// TestStandardDefaultRunsBothLayers checks the default is `all`: the base
// layer contributes rule IDs the slop layer never emits, and the slop layer's
// own hits survive alongside them.
func TestStandardDefaultRunsBothLayers(t *testing.T) {
	fixture := filepath.Join("testdata", "mixed.md")
	all := checkOK(t, fixture, "--llm-effort=off")
	slop := checkOK(t, fixture, "--llm-effort=off", "--standard=slop")

	allRep, slopRep := decodeReport(t, all), decodeReport(t, slop)
	baseIDs := ruleIDs(rules.Base)
	fired := 0
	for id := range allRep.CountsByRule {
		if baseIDs[id] {
			fired++
		}
	}
	if fired == 0 {
		t.Fatalf("default run emitted no base-layer rule: %v", allRep.CountsByRule)
	}
	for id, n := range slopRep.CountsByRule {
		if allRep.CountsByRule[id] != n {
			t.Fatalf("default run dropped slop hits for %s: %d, want %d", id, allRep.CountsByRule[id], n)
		}
	}
	if len(allRep.Violations) <= len(slopRep.Violations) {
		t.Fatalf("default run has %d violations, slop-only has %d; the base layer added nothing", len(allRep.Violations), len(slopRep.Violations))
	}
}

// TestStandardDefaultCountsCategoryBase pins the `base` key in
// counts_by_category, which is how an agent tells the two layers apart in a
// single report.
func TestStandardDefaultCountsCategoryBase(t *testing.T) {
	rep := decodeReport(t, checkOK(t, filepath.Join("testdata", "mixed.md"), "--llm-effort=off"))
	if rep.CountsByCategory[types.CategoryBase] == 0 {
		t.Fatalf("counts_by_category has no base key: %v", rep.CountsByCategory)
	}
}

// TestStandardBaseEmitsNoSlopOnlyRules proves the routing is exclusive.
// elevated-register is the one shared ID: D10 has the base layer's plain-word
// detector reuse it rather than adding a parallel rule, so it is allowed
// through while every other slop rule must be absent.
func TestStandardBaseEmitsNoSlopOnlyRules(t *testing.T) {
	for _, fixture := range []string{"mixed.md", "readme.md", "skill.md"} {
		t.Run(fixture, func(t *testing.T) {
			out := checkOK(t, filepath.Join("testdata", fixture), "--llm-effort=off", "--standard=base")
			allowed := ruleIDs(rules.Base)
			allowed["elevated-register"] = true
			for id := range decodeReport(t, out).CountsByRule {
				if !allowed[id] {
					t.Fatalf("--standard=base emitted slop-only rule %s", id)
				}
			}
		})
	}
}

// TestStandardInvalidIsUsageError checks a bad --standard fails as a usage
// error, which main() maps to the flag/usage exit code.
func TestStandardInvalidIsUsageError(t *testing.T) {
	out, err := runCheck(t, filepath.Join("testdata", "mixed.md"), "--standard=bogus")
	if err == nil {
		t.Fatal("expected an error for --standard=bogus")
	}
	if !errors.As(err, new(usageError)) {
		t.Fatalf("error %v is not a usageError; main() would not map it to exit %d", err, exitUsage)
	}
	if out != "" {
		t.Fatalf("a rejected run still wrote a report: %s", out)
	}
}

func TestResolveStandard(t *testing.T) {
	cases := []struct {
		raw  string
		want standardLayer
	}{
		{"", standardAll},
		{"all", standardAll},
		{"ALL", standardAll},
		{"slop", standardSlop},
		{"Slop", standardSlop},
		{"base", standardBase},
		{"BASE", standardBase},
	}
	for _, c := range cases {
		got, err := resolveStandard(c.raw)
		if err != nil {
			t.Fatalf("resolveStandard(%q): %v", c.raw, err)
		}
		if got != c.want {
			t.Fatalf("resolveStandard(%q) = %q, want %q", c.raw, got, c.want)
		}
	}
	for _, raw := range []string{"bogus", "both", "ste", "all,base"} {
		if _, err := resolveStandard(raw); err == nil {
			t.Fatalf("resolveStandard(%q): expected an error", raw)
		}
	}
}

// TestStandardCataloguePartitionsTheRules pins the contract the LLM prompt
// depends on: slop, then base, then google, with All the exact concatenation.
func TestStandardCataloguePartitionsTheRules(t *testing.T) {
	slop, base := standardSlop.catalogue(), standardBase.catalogue()
	google, all := standardGoogle.catalogue(), standardAll.catalogue()
	if len(all) != len(slop)+len(base)+len(google) {
		t.Fatalf("all=%d, slop=%d, base=%d, google=%d; All is not the concatenation",
			len(all), len(slop), len(base), len(google))
	}
	layers := []struct {
		name  string
		rules []types.ViolationRule
	}{{"slop", slop}, {"base", base}, {"google", google}}
	at := 0
	for _, layer := range layers {
		for i, r := range layer.rules {
			if all[at+i].ID != r.ID {
				t.Fatalf("All[%d] = %s, want the %s rule %s", at+i, all[at+i].ID, layer.name, r.ID)
			}
		}
		at += len(layer.rules)
	}
}

// firstDiff locates where two reports part company and windows both sides
// around it, so a drifted golden reports the change rather than 8KB of JSON.
func firstDiff(want, got string) string {
	i := 0
	for i < len(want) && i < len(got) && want[i] == got[i] {
		i++
	}
	return fmt.Sprintf("byte %d:\n want: …%s…\n  got: …%s…", i, window(want, i), window(got, i))
}

func window(s string, at int) string {
	lo := max(at-60, 0)
	hi := min(at+120, len(s))
	return s[lo:hi]
}

// legacyProjection re-encodes a report as the pre-google envelope the goldens
// pin, so the gate keeps grading violations, offsets, and counts while the
// envelope around them grows new fields.
func legacyProjection(t *testing.T, out string) string {
	t.Helper()
	rep := decodeReport(t, out)
	var buf bytes.Buffer
	enc := json.NewEncoder(&buf)
	enc.SetEscapeHTML(false)
	if err := enc.Encode(legacyReport{
		TextLength:       rep.TextLength,
		Violations:       rep.Violations,
		CountsByRule:     rep.CountsByRule,
		CountsByCategory: rep.CountsByCategory,
		Lang:             rep.Lang,
		LLMEffort:        rep.LLMEffort,
		Readability:      rep.Readability,
	}); err != nil {
		t.Fatal(err)
	}
	return buf.String()
}

// legacyReport is the checkReport shape the goldens were captured from.
type legacyReport struct {
	TextLength       int                             `json:"text_length"`
	Violations       []types.Violation               `json:"violations"`
	CountsByRule     map[string]int                  `json:"counts_by_rule"`
	CountsByCategory map[types.ViolationCategory]int `json:"counts_by_category"`
	Lang             string                          `json:"lang"`
	LLMEffort        string                          `json:"llm_effort"`
	Readability      *readability.Report             `json:"readability,omitempty"`
}

func ruleIDs(rs []types.ViolationRule) map[string]bool {
	out := make(map[string]bool, len(rs))
	for _, r := range rs {
		out[r.ID] = true
	}
	return out
}
