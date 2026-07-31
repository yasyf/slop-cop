package llm

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"testing"
	"time"
)

// fakeClaude puts an executable named claude on an otherwise-empty $PATH
// that ignores its arguments and prints the response file to stdout.
func fakeClaude(t *testing.T, responsePath string) {
	t.Helper()
	dir := t.TempDir()
	script := fmt.Sprintf("#!/bin/sh\nexec /bin/cat %q\n", responsePath)
	//nolint:gosec // G306: an executable stub on a test-owned temp PATH.
	if err := os.WriteFile(filepath.Join(dir, "claude"), []byte(script), 0o755); err != nil {
		t.Fatalf("write fake claude: %v", err)
	}
	t.Setenv("PATH", dir)
}

// TestRunSchemaStreamArray replays a real captured `claude -p --output-format
// json` response — a JSON array of stream events whose terminal type=="result"
// element carries the schema payload — and proves RunSchema extracts and
// decodes it.
func TestRunSchemaStreamArray(t *testing.T) {
	fixture, err := filepath.Abs(filepath.Join("testdata", "claude_stream_array.json"))
	if err != nil {
		t.Fatal(err)
	}
	fakeClaude(t, fixture)

	cfg := Config{Model: DefaultSentenceModel, Timeout: 30 * time.Second}
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

// TestRunSentenceResolvesByteOffsets drives the sentence pass end to end
// against a synthetic stream-array response and checks that matched text
// resolves to UTF-8 byte offsets in the original input.
func TestRunSentenceResolvesByteOffsets(t *testing.T) {
	const text = "Café rules. We utilize this approach."
	payload := `{"violations":[{"ruleId":"elevated-register","matchedText":"utilize","explanation":"register","suggestedChange":"use"}]}`
	events := []map[string]any{
		{"type": "system", "subtype": "init"},
		{"type": "assistant"},
		{
			"type":              "result",
			"subtype":           "success",
			"is_error":          false,
			"result":            payload,
			"structured_output": json.RawMessage(payload),
		},
	}
	raw, err := json.Marshal(events)
	if err != nil {
		t.Fatal(err)
	}
	response := filepath.Join(t.TempDir(), "response.json")
	//nolint:gosec // G306: a fixture copy in a test-owned temp dir.
	if err := os.WriteFile(response, raw, 0o644); err != nil {
		t.Fatal(err)
	}
	fakeClaude(t, response)

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
