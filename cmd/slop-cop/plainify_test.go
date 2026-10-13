package main

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// runPlainify executes the plainify command in-process and returns the raw
// bytes it wrote to stdout. writeJSON encodes straight to os.Stdout, so the fd
// is redirected to a temp file for the duration of the call.
func runPlainify(t *testing.T, args ...string) (string, error) {
	t.Helper()
	f, err := os.CreateTemp(t.TempDir(), "stdout")
	if err != nil {
		t.Fatal(err)
	}
	orig := os.Stdout
	os.Stdout = f
	cmd := newPlainifyCmd()
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

// shQuote renders s as a single-quoted shell word, so a stub can carry a JSON
// payload in its own body.
func shQuote(s string) string {
	return "'" + strings.ReplaceAll(s, "'", `'\''`) + "'"
}

// claudeReplying puts a claude stub on an otherwise-empty $PATH that answers
// every call with the same rewrite, and returns the path it logs each call's
// arguments to.
func claudeReplying(t *testing.T, plain string) string {
	t.Helper()
	dir := t.TempDir()
	payload, err := json.Marshal(map[string]string{"plain": plain})
	if err != nil {
		t.Fatal(err)
	}
	events := []map[string]any{{
		"type":              "result",
		"subtype":           "success",
		"is_error":          false,
		"result":            string(payload),
		"structured_output": json.RawMessage(payload),
	}}
	raw, err := json.Marshal(events)
	if err != nil {
		t.Fatal(err)
	}
	argsLog := filepath.Join(dir, "args")
	// The stub runs on shell builtins alone: spawning cat costs a second
	// execve per call, which under the endpoint-security agents that inspect
	// every exec on a Mac is ~0.3s rather than microseconds.
	script := fmt.Sprintf(
		"#!/bin/sh\nwhile IFS= read -r line; do :; done\nprintf '%%s\\n' \"$*\" >> %q\nprintf '%%s' %s\n",
		argsLog, shQuote(string(raw)),
	)
	bin := t.TempDir()
	//nolint:gosec // G306: an executable stub on a test-owned temp PATH.
	if err := os.WriteFile(filepath.Join(bin, "claude"), []byte(script), 0o755); err != nil {
		t.Fatal(err)
	}
	t.Setenv("PATH", bin)
	return argsLog
}

func writeTemp(t *testing.T, name, body string) string {
	t.Helper()
	path := filepath.Join(t.TempDir(), name)
	//nolint:gosec // G306: a fixture in a test-owned temp dir.
	if err := os.WriteFile(path, []byte(body), 0o644); err != nil {
		t.Fatal(err)
	}
	return path
}

// TestPlainifyCommandReport pins the single-input envelope: the rewrite, its
// word count, and both constraint fields, with violations an empty array
// rather than null.
func TestPlainifyCommandReport(t *testing.T) {
	claudeReplying(t, "The worker moved to the new transport.")
	draft := writeTemp(t, "draft.md", "We leveraged a novel transport abstraction.")

	out, err := runPlainify(t, draft)
	if err != nil {
		t.Fatalf("plainify: %v", err)
	}
	var got plainifyResult
	if err := json.Unmarshal([]byte(out), &got); err != nil {
		t.Fatalf("decode report: %v\n%s", err, out)
	}
	if got.Plain != "The worker moved to the new transport." {
		t.Fatalf("plain=%q", got.Plain)
	}
	if got.Words != 7 || got.Truncated {
		t.Fatalf("words=%d truncated=%v, want 7 and false", got.Words, got.Truncated)
	}
	if !json.Valid([]byte(out)) || !strings.Contains(out, `"violations":[]`) {
		t.Fatalf("violations did not serialise as an empty array:\n%s", out)
	}
}

// TestPlainifyCommandJSONMode covers the batch contract: an array in, an array
// of the same length and order out, each element carrying its entry's id.
func TestPlainifyCommandJSONMode(t *testing.T) {
	claudeReplying(t, "The worker moved.")
	entries := writeTemp(t, "entries.json", `[
	  {"id": "DQ4", "text": "We leveraged a novel transport abstraction."},
	  {"id": "DQ7", "text": "The subsystem exhibits nondeterministic characteristics."}
	]`)

	out, err := runPlainify(t, entries, "--json")
	if err != nil {
		t.Fatalf("plainify --json: %v", err)
	}
	var got []plainifyEntryResult
	if err := json.Unmarshal([]byte(out), &got); err != nil {
		t.Fatalf("decode report: %v\n%s", err, out)
	}
	if len(got) != 2 {
		t.Fatalf("results=%d, want 2:\n%s", len(got), out)
	}
	for i, wantID := range []string{"DQ4", "DQ7"} {
		if got[i].ID != wantID {
			t.Fatalf("result %d has id %q, want %q", i, got[i].ID, wantID)
		}
		if got[i].Plain != "The worker moved." || got[i].Words != 3 {
			t.Fatalf("result %d is %+v", i, got[i])
		}
	}
}

// TestPlainifyCommandJSONModeRejectsNull covers the one non-array input JSON
// decodes without complaint: null would otherwise print an empty array and
// exit 0, while an empty array is a real answer.
func TestPlainifyCommandJSONModeRejectsNull(t *testing.T) {
	claudeReplying(t, "unused")

	if _, err := runPlainify(t, writeTemp(t, "null.json", "null"), "--json"); err == nil {
		t.Fatal("expected null entries to fail")
	}
	out, err := runPlainify(t, writeTemp(t, "empty.json", "[]"), "--json")
	if err != nil {
		t.Fatalf("plainify --json on an empty array: %v", err)
	}
	if strings.TrimSpace(out) != "[]" {
		t.Fatalf("empty array printed %q, want []", out)
	}
}

// TestPlainifyCommandReportsSurvivingViolations proves a forbidden id the
// model kept lands in the report instead of being scrubbed or erroring out.
func TestPlainifyCommandReportsSurvivingViolations(t *testing.T) {
	claudeReplying(t, "DQ4 moved.")
	draft := writeTemp(t, "draft.md", "DQ4 was migrated.")

	out, err := runPlainify(t, draft, "--forbid", `\b(DQ|A|Q|V)\d+\b`)
	if err != nil {
		t.Fatalf("plainify: %v", err)
	}
	var got plainifyResult
	if err := json.Unmarshal([]byte(out), &got); err != nil {
		t.Fatalf("decode report: %v\n%s", err, out)
	}
	if got.Plain != "DQ4 moved." {
		t.Fatalf("plain=%q, want the model's answer untouched", got.Plain)
	}
	if len(got.Violations) != 1 || got.Violations[0].Match != "DQ4" {
		t.Fatalf("violations=%+v, want one DQ4 match", got.Violations)
	}
}

// TestPlainifyCommandGlossaryReachesTheModel closes the loop on --glossary and
// --name-by-title: both land in the system prompt claude is invoked with.
func TestPlainifyCommandGlossaryReachesTheModel(t *testing.T) {
	argsLog := claudeReplying(t, "The worker transport moved.")
	glossary := writeTemp(t, "titles.json", `{"DQ4": "Worker transport"}`)
	draft := writeTemp(t, "draft.md", "DQ4 was migrated.")

	if _, err := runPlainify(t, draft, "--name-by-title", "--glossary", glossary); err != nil {
		t.Fatalf("plainify: %v", err)
	}
	//nolint:gosec // G304: a test-owned temp path, not user input.
	args, err := os.ReadFile(argsLog)
	if err != nil {
		t.Fatal(err)
	}
	for _, want := range []string{"Name every item you refer to by its title", `DQ4 is titled "Worker transport"`} {
		if !strings.Contains(string(args), want) {
			t.Fatalf("the system prompt is missing %q:\n%s", want, args)
		}
	}
}

// TestPlainifyCommandRejectsBadFlags maps each malformed constraint to the
// usage exit class rather than a claude call.
func TestPlainifyCommandRejectsBadFlags(t *testing.T) {
	draft := writeTemp(t, "draft.md", "prose")
	cases := []struct {
		name string
		args []string
	}{
		{"unparsable forbid pattern", []string{draft, "--forbid", "("}},
		{"negative word budget", []string{draft, "--max-words", "-1"}},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			_, err := runPlainify(t, c.args...)
			if err == nil {
				t.Fatal("expected a usage error")
			}
			if !errors.As(err, new(usageError)) {
				t.Fatalf("error %v is not a usageError", err)
			}
		})
	}
}
