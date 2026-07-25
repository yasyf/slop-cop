package releasecontract

import (
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
)

func TestPostreleaseWorkflows(t *testing.T) {
	_, source, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatal("locate test source")
	}
	root := filepath.Clean(filepath.Join(filepath.Dir(source), "..", ".."))
	read := func(name string) string {
		t.Helper()
		data, err := os.ReadFile(filepath.Join(root, ".github", "workflows", name))
		if err != nil {
			t.Fatal(err)
		}
		return string(data)
	}
	ci := read("ci.yml")
	guides := read("guides.yml")
	release := read("release.yml")
	for name, workflow := range map[string]string{"CI": ci, "Guides": guides} {
		for _, required := range []string{
			"commit:",
			"required: true",
			"${{ inputs.commit || github.sha }}",
		} {
			if !strings.Contains(workflow, required) {
				t.Errorf("%s workflow missing %q", name, required)
			}
		}
	}
	if checkouts, refs := strings.Count(ci, "uses: actions/checkout@v7"), strings.Count(ci, "ref: ${{ inputs.commit || github.sha }}"); checkouts != refs {
		t.Errorf("CI exact checkout refs = %d, want %d", refs, checkouts)
	}
	for _, required := range []string{
		"needs: descriptor",
		"actions: write",
		"TAG: ${{ github.ref_name }}",
		`echo "sha=$(git rev-parse HEAD)" >> "$GITHUB_OUTPUT"`,
		"for workflow in ci.yml guides.yml",
		"X-GitHub-Api-Version: 2026-03-10",
		"gh run watch",
		`wait "$pid" || status=1`,
	} {
		if !strings.Contains(release, required) {
			t.Errorf("Release workflow missing %q", required)
		}
	}
}
