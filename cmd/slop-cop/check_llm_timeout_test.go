package main

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// pathWithRetryableCodex returns a PATH holding exactly one entry: a directory
// containing a codex that fails fast with an error spawnllm classifies as
// transient, so the call spends its budget on the retry loop rather than on a
// single attempt.
func pathWithRetryableCodex(t *testing.T) string {
	t.Helper()
	dir := t.TempDir()
	script := "#!/bin/sh\nsleep 0.2\necho 'rate limit exceeded' >&2\nexit 1\n"
	if err := os.WriteFile(filepath.Join(dir, "codex"), []byte(script), 0o755); err != nil { //nolint:gosec // G306: an executable stub on a test-owned temp PATH.
		t.Fatalf("write fake codex: %v", err)
	}
	return dir
}

// TestAutoPassTimeoutKeepsClientResults pins what a wedged provider CLI costs:
// the LLM tier, and nothing else. The auto-enabled sentence pass exhausts its
// budget, reports that under llm.sentence.error, and the client-side rules
// still return their violations under exit code 0.
func TestAutoPassTimeoutKeepsClientResults(t *testing.T) {
	t.Setenv("PATH", pathWithRetryableCodex(t))

	out, err := runCheck(t, filepath.Join("testdata", "readme.md"), "--lang=markdown", "--sentence-timeout=3s")
	if err != nil {
		t.Fatalf("an auto-enabled pass must fail open, got %v", err)
	}
	rep := decodeReport(t, out)

	if len(rep.Violations) == 0 {
		t.Fatal("client-side violations were lost with the LLM pass")
	}
	if rep.LLM == nil || rep.LLM.Sentence == nil {
		t.Fatalf("no sentence status in %+v", rep.LLM)
	}
	got := rep.LLM.Sentence
	if got.Ran || !got.Auto {
		t.Fatalf("sentence status = %+v, want an auto pass that did not run", got)
	}
	if !strings.Contains(got.Error, "timed out after 3s including retries") {
		t.Fatalf("llm.sentence.error = %q, does not report the exhausted budget", got.Error)
	}
}
