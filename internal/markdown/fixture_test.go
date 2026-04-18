package markdown

import (
	"encoding/json"
	"flag"
	"os"
	"path/filepath"
	"testing"

	"github.com/yasyf/slop-cop/internal/detectors"
	"github.com/yasyf/slop-cop/internal/types"
)

// -update rewrites fixture.md.golden from the current detector output. Use
// only when the fixture source or detector behaviour changes intentionally.
var updateGolden = flag.Bool("update", false, "rewrite testdata/*.golden")

// TestAnalyze_Fixture runs the real client detector over the masked
// fixture.md and snapshot-compares the violations against a committed
// golden file. Any change to Analyze or to a detector that alters what the
// fixture produces will fail this test until -update is used.
func TestAnalyze_Fixture(t *testing.T) {
	fixturePath := filepath.Join("testdata", "fixture.md")
	goldenPath := filepath.Join("testdata", "fixture.md.golden")

	src, err := os.ReadFile(fixturePath)
	if err != nil {
		t.Fatalf("read fixture: %v", err)
	}

	masked, suppress, fm := Analyze(string(src))
	assertInvariants(t, string(src), masked, suppress, fm)

	violations := detectors.RunClient(masked)
	// Sanity: every violation must still map to a valid substring of the
	// original source.
	for _, v := range violations {
		if v.StartIndex < 0 || v.EndIndex > len(src) || v.EndIndex < v.StartIndex {
			t.Fatalf("invalid offsets on violation %+v", v)
		}
	}

	payload := struct {
		Violations []types.Violation `json:"violations"`
	}{Violations: violations}
	out, err := json.MarshalIndent(payload, "", "  ")
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}
	out = append(out, '\n')

	if *updateGolden {
		if err := os.WriteFile(goldenPath, out, 0o644); err != nil {
			t.Fatalf("write golden: %v", err)
		}
		return
	}

	want, err := os.ReadFile(goldenPath)
	if err != nil {
		t.Fatalf("read golden (run with -update to create): %v", err)
	}
	if string(out) != string(want) {
		t.Fatalf("fixture output drift; re-run with -update if intentional.\n--- got ---\n%s\n--- want ---\n%s", out, want)
	}
}
