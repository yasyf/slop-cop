package llm

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	spawnllm "github.com/yasyf/spawnllm/go"
)

// The stub bodies in this package run on shell builtins alone. Spawning cat
// costs a second execve per call, which under the endpoint-security agents
// that inspect every exec on a Mac is ~0.3s rather than microseconds.
const drainStdin = "while IFS= read -r line; do :; done\n"

// shQuote renders s as a single-quoted shell word, so a stub can carry a JSON
// payload in its own body.
func shQuote(s string) string {
	return "'" + strings.ReplaceAll(s, "'", `'\''`) + "'"
}

// printPayload is a stub command writing payload to stdout.
func printPayload(payload string) string {
	return "printf '%s' " + shQuote(payload)
}

// codexWriteResult is a stub fragment scanning argv for -o and writing payload
// to the path after it, which is where spawnllm reads a codex result from.
func codexWriteResult(payload string) string {
	return "while [ $# -gt 0 ]; do\n\tif [ \"$1\" = -o ]; then printf '%s' " +
		shQuote(payload) + " > \"$2\"; exit 0; fi\n\tshift\ndone\n"
}

// writeStub puts an executable named name, running body, on an
// otherwise-empty $PATH.
func writeStub(t *testing.T, name, body string) {
	t.Helper()
	dir := t.TempDir()
	//nolint:gosec // G306: an executable stub on a test-owned temp PATH.
	if err := os.WriteFile(filepath.Join(dir, name), []byte("#!/bin/sh\n"+body), 0o755); err != nil {
		t.Fatalf("write fake %s: %v", name, err)
	}
	t.Setenv("PATH", dir)
}

// fakeClaude puts an executable named claude on an otherwise-empty $PATH
// that ignores its arguments and prints payload to stdout.
func fakeClaude(t *testing.T, payload string) {
	t.Helper()
	writeStub(t, "claude", drainStdin+printPayload(payload)+"\n")
}

// fakeCodex puts an executable named codex on an otherwise-empty $PATH that
// answers with payload.
func fakeCodex(t *testing.T, payload string) {
	t.Helper()
	writeStub(t, "codex", codexWriteResult(payload))
}

// TestRunSchemaStreamArray replays a real captured `claude -p --output-format
// json` response — a JSON array of stream events whose terminal type=="result"
// element carries the schema payload — and proves RunSchema extracts and
// decodes it.
func TestRunSchemaStreamArray(t *testing.T) {
	//nolint:gosec // G304: a fixed testdata path, not user input.
	fixture, err := os.ReadFile(filepath.Join("testdata", "claude_stream_array.json"))
	if err != nil {
		t.Fatal(err)
	}
	fakeClaude(t, string(fixture))

	cfg := Config{Provider: spawnllm.ProviderClaude, Model: DefaultRewriteModel, Timeout: 30 * time.Second}
	var env violationsEnvelope
	if err := RunSchema(context.Background(), cfg, SentenceSystemPrompt, "text", json.RawMessage(ViolationToolSchema), &env); err != nil {
		t.Fatalf("RunSchema: %v", err)
	}
	if env.Violations == nil {
		t.Fatalf("violations not decoded: envelope %+v", env)
	}
	if len(env.Violations) != 0 {
		t.Fatalf("fixture carries zero violations, got %d", len(env.Violations))
	}
}

// TestRunSchemaPanicsOnAnUnroutedProvider proves the provider is a real switch
// rather than a label: a Config naming a backend RunSchema has no spec for
// crashes instead of falling through to a default one, so a routing bug can
// never reach an auto-enabled pass as an ordinary fail-open LLM error.
func TestRunSchemaPanicsOnAnUnroutedProvider(t *testing.T) {
	defer func() {
		got := recover()
		if got == nil {
			t.Fatal("an unrouted provider must panic, not pick a backend")
		}
		if want := `llm: unsupported provider "gemini"`; got != want {
			t.Fatalf("panic = %v, want %q", got, want)
		}
	}()
	cfg := Config{Provider: spawnllm.ProviderGemini, Model: "gemini-3-pro", Timeout: time.Second}
	var env violationsEnvelope
	//nolint:errcheck // the call panics before it returns.
	_ = RunSchema(context.Background(), cfg, SentenceSystemPrompt, "text", json.RawMessage(ViolationToolSchema), &env)
}

// TestRunSentenceResolvesByteOffsets drives the sentence pass end to end
// against a synthetic codex response and checks that matched text resolves to
// UTF-8 byte offsets in the original input.
func TestRunSentenceResolvesByteOffsets(t *testing.T) {
	const text = "Café rules. We utilize this approach."
	fakeCodex(t, `{"violations":[{"ruleId":"elevated-register","matchedText":"utilize","explanation":"register","suggestedChange":"use"}]}`)

	vs, err := RunSentence(context.Background(), text, Options{Timeout: 30 * time.Second})
	if err != nil {
		t.Fatalf("RunSentence: %v", err)
	}
	if len(vs) != 1 {
		t.Fatalf("violations=%d, want 1: %+v", len(vs), vs)
	}
	v := vs[0]
	wantStart := len("Café rules. We ")
	wantEnd := wantStart + len("utilize")
	if v.RuleID != "elevated-register" || v.MatchedText != "utilize" {
		t.Fatalf("violation %+v", v)
	}
	if v.StartIndex != wantStart || v.EndIndex != wantEnd {
		t.Fatalf("offsets=[%d,%d], want [%d,%d]", v.StartIndex, v.EndIndex, wantStart, wantEnd)
	}
}

// fakeRetryableCodex puts an executable named codex on an otherwise-empty
// $PATH that works for delay, then fails with an error spawnllm classifies as
// transient — the input that drives its retry-with-backoff loop.
func fakeRetryableCodex(t *testing.T, delay time.Duration) {
	t.Helper()
	dir := t.TempDir()
	script := fmt.Sprintf("#!/bin/sh\nsleep %.2f\necho 'rate limit exceeded' >&2\nexit 1\n", delay.Seconds())
	//nolint:gosec // G306: an executable stub on a test-owned temp PATH.
	if err := os.WriteFile(filepath.Join(dir, "codex"), []byte(script), 0o755); err != nil {
		t.Fatalf("write fake codex: %v", err)
	}
	t.Setenv("PATH", dir)
}

// TestRunSchemaBoundsTheRetryLoop pins Config.Timeout as a bound on the whole
// call rather than on one attempt. The stub fails well inside the per-attempt
// bound and retryably, so spawnllm retries it up to five times with backoff;
// while only RunSpec.Timeout carried the budget, that made a
// --sentence-timeout=20s run against a rate-limited codex measure 150s.
func TestRunSchemaBoundsTheRetryLoop(t *testing.T) {
	fakeRetryableCodex(t, 200*time.Millisecond)

	cfg := Config{Provider: spawnllm.ProviderCodex, Model: DefaultSentenceModel, Timeout: 3 * time.Second}
	var env violationsEnvelope
	start := time.Now()
	err := RunSchema(context.Background(), cfg, SentenceSystemPrompt, "text", json.RawMessage(ViolationToolSchema), &env)
	elapsed := time.Since(start)

	if err == nil {
		t.Fatal("a provider that always fails must return an error")
	}
	if !strings.Contains(err.Error(), "timed out after 3s including retries") {
		t.Fatalf("error %q is not the exhausted whole-call budget", err)
	}
	if elapsed > 30*time.Second {
		t.Fatalf("call took %s against a %s budget: the retry loop is unbounded", elapsed, cfg.Timeout)
	}
}
