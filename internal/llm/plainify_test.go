package llm

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"testing"
	"time"
)

// plainResponse renders one captured-shape `claude -p --output-format json`
// reply — the stream array whose terminal type=="result" element carries the
// schema payload — for a given rewrite.
func plainResponse(t *testing.T, plain string) string {
	t.Helper()
	payload, err := json.Marshal(plainifyEnvelope{Plain: plain})
	if err != nil {
		t.Fatal(err)
	}
	events := []map[string]any{
		{"type": "system", "subtype": "init"},
		{"type": "assistant"},
		{
			"type":              "result",
			"subtype":           "success",
			"is_error":          false,
			"result":            string(payload),
			"structured_output": json.RawMessage(payload),
		},
	}
	raw, err := json.Marshal(events)
	if err != nil {
		t.Fatal(err)
	}
	return string(raw)
}

// fakeClaudeSequence puts a claude stub on an otherwise-empty $PATH that
// answers the Nth call with plains[N-1] and appends each call's arguments to
// the returned log file, so a test can read back the system prompt each
// attempt was given.
func fakeClaudeSequence(t *testing.T, plains ...string) string {
	t.Helper()
	state := t.TempDir()
	counter := filepath.Join(state, "calls")
	argsLog := filepath.Join(state, "args")

	var arms strings.Builder
	for i, plain := range plains {
		fmt.Fprintf(&arms, "\t%d) %s ;;\n", i+1, printPayload(plainResponse(t, plain)))
	}
	writeStub(t, "claude", fmt.Sprintf(`%[1]sprintf '%%s\n' "$*" >> %[2]q
n=0
if [ -f %[3]q ]; then read -r n < %[3]q; fi
n=$((n+1))
printf '%%s\n' "$n" > %[3]q
case "$n" in
%[4]s	*) exit 1 ;;
esac
`, drainStdin, argsLog, counter, arms.String()))
	return argsLog
}

func readArgsLog(t *testing.T, path string) string {
	t.Helper()
	//nolint:gosec // G304: a test-owned temp path, not user input.
	b, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	return string(b)
}

func testOptions() Options { return Options{Timeout: 30 * time.Second} }

// TestPlainifyPromptCarriesEveryConstraint proves each constraint reaches the
// model, and that an unconstrained call carries no constraint block at all.
func TestPlainifyPromptCarriesEveryConstraint(t *testing.T) {
	bare := BuildPlainifySystemPrompt(PlainifyConstraints{})
	for _, principle := range plainifyPrinciples {
		if !strings.Contains(bare, principle) {
			t.Fatalf("bare prompt is missing a standing principle: %q", principle)
		}
	}
	if strings.Contains(bare, "constraints") {
		t.Fatalf("bare prompt carries a constraint block:\n%s", bare)
	}

	full := BuildPlainifySystemPrompt(PlainifyConstraints{
		MaxWords:    120,
		Forbid:      []*regexp.Regexp{regexp.MustCompile(`\bDQ\d+\b`)},
		NameByTitle: true,
		Glossary:    map[string]string{"DQ4": "Worker transport", "DQ1": "Session identity"},
	})
	for _, want := range []string{
		"at most 120 words",
		`\bDQ\d+\b`,
		"Name every item you refer to by its title",
		`DQ4 is titled "Worker transport"`,
	} {
		if !strings.Contains(full, want) {
			t.Fatalf("constrained prompt is missing %q:\n%s", want, full)
		}
	}
	if strings.Index(full, "DQ1 is titled") > strings.Index(full, "DQ4 is titled") {
		t.Fatalf("glossary entries are not sorted by identifier:\n%s", full)
	}
}

// TestPlainifyRetriesOnceAndKeepsTheRetry drives a rewrite that busts the word
// budget on the first attempt and fits on the second.
func TestPlainifyRetriesOnceAndKeepsTheRetry(t *testing.T) {
	argsLog := fakeClaudeSequence(t, "one two three four five", "one two three")

	c := PlainifyConstraints{MaxWords: 4}
	res, err := Plainify(context.Background(), "a long claudish paragraph", c, testOptions())
	if err != nil {
		t.Fatalf("Plainify: %v", err)
	}
	if res.Plain != "one two three" {
		t.Fatalf("plain=%q, want the retry's answer", res.Plain)
	}
	if res.Words != 3 || res.Truncated {
		t.Fatalf("words=%d truncated=%v, want 3 and false", res.Words, res.Truncated)
	}
	if len(res.Violations) != 0 {
		t.Fatalf("violations=%+v, want none", res.Violations)
	}

	args := readArgsLog(t, argsLog)
	if n := strings.Count(args, "--json-schema"); n != 2 {
		t.Fatalf("claude was called %d times, want 2:\n%s", n, args)
	}
	if !strings.Contains(args, "ran to 5 words against a limit of 4") {
		t.Fatalf("the retry never named the miss:\n%s", args)
	}
}

// TestPlainifyReportsWhatTheRetryStillMisses is the no-silent-drop half: a
// model that ignores the constraints twice yields a report, not an error and
// not a scrubbed rewrite.
func TestPlainifyReportsWhatTheRetryStillMisses(t *testing.T) {
	const stubborn = "DQ4 and DQ7 both moved to the new transport"
	fakeClaudeSequence(t, stubborn, stubborn)

	c := PlainifyConstraints{
		MaxWords: 3,
		Forbid:   []*regexp.Regexp{regexp.MustCompile(`\bDQ\d+\b`)},
	}
	res, err := Plainify(context.Background(), "claudish input", c, testOptions())
	if err != nil {
		t.Fatalf("Plainify: %v", err)
	}
	if res.Plain != stubborn {
		t.Fatalf("plain=%q, want the model's answer untouched", res.Plain)
	}
	if !res.Truncated || res.Words != 9 {
		t.Fatalf("words=%d truncated=%v, want 9 and true", res.Words, res.Truncated)
	}
	want := []PlainViolation{
		{Pattern: `\bDQ\d+\b`, Match: "DQ4"},
		{Pattern: `\bDQ\d+\b`, Match: "DQ7"},
	}
	if fmt.Sprint(res.Violations) != fmt.Sprint(want) {
		t.Fatalf("violations=%+v, want %+v", res.Violations, want)
	}
}

// TestPlainifyLeavesAMetRewriteAlone checks the single-call path: no retry
// fires when the first answer already meets the constraints.
func TestPlainifyLeavesAMetRewriteAlone(t *testing.T) {
	argsLog := fakeClaudeSequence(t, "  the worker moved  ")

	res, err := Plainify(context.Background(), "input", PlainifyConstraints{MaxWords: 10}, testOptions())
	if err != nil {
		t.Fatalf("Plainify: %v", err)
	}
	if res.Plain != "the worker moved" {
		t.Fatalf("plain=%q, want the trimmed answer", res.Plain)
	}
	if n := strings.Count(readArgsLog(t, argsLog), "--json-schema"); n != 1 {
		t.Fatalf("claude was called %d times, want 1", n)
	}
}

// fakeClaudeByMarker puts a claude stub on $PATH that answers from a marker
// found in the text it is handed on stdin, sleeping for the marker's index so
// the entries finish out of the order they were dispatched in. A marker
// mapped to the empty rewrite exits 1 instead.
func fakeClaudeByMarker(t *testing.T, markers []string, fail string) {
	t.Helper()
	var arms strings.Builder
	for i, marker := range markers {
		if marker == fail {
			fmt.Fprintf(&arms, "  *%s*) exit 1 ;;\n", marker)
			continue
		}
		delay := float64(len(markers)-i) / 20.0
		fmt.Fprintf(&arms, "  *%s*) /bin/sleep %.2f; %s ;;\n", marker, delay, printPayload(plainResponse(t, "plain "+marker)))
	}
	writeStub(t, "claude", "body=\n"+
		"while IFS= read -r line || [ -n \"$line\" ]; do body=\"$body$line\"; done\n"+
		"case \"$body\" in\n"+arms.String()+"  *) exit 1 ;;\nesac\n")
}

func markerEntries(markers []string) []PlainEntry {
	entries := make([]PlainEntry, 0, len(markers))
	for _, marker := range markers {
		entries = append(entries, PlainEntry{ID: "id-" + marker, Text: "claudish " + marker + " prose"})
	}
	return entries
}

// TestPlainifyBatchKeepsEntryOrder proves results come back paired with the
// entry that produced them. There are more entries than concurrency slots, so
// the run blocks on the semaphore, and each stub sleeps for its position so
// the calls finish in reverse order.
func TestPlainifyBatchKeepsEntryOrder(t *testing.T) {
	markers := []string{"alpha", "beta", "gamma", "delta", "epsilon", "zeta"}
	if len(markers) <= plainifyConcurrency {
		t.Fatalf("the batch needs more than %d entries to block on a slot", plainifyConcurrency)
	}
	fakeClaudeByMarker(t, markers, "")

	results, err := PlainifyBatch(context.Background(), markerEntries(markers), PlainifyConstraints{}, testOptions())
	if err != nil {
		t.Fatalf("PlainifyBatch: %v", err)
	}
	if len(results) != len(markers) {
		t.Fatalf("results=%d, want %d", len(results), len(markers))
	}
	for i, marker := range markers {
		if want := "plain " + marker; results[i].Plain != want {
			t.Fatalf("result %d is %q, want %q", i, results[i].Plain, want)
		}
	}
}

// TestPlainifyBatchNamesTheFailingEntry fails one entry away from index 0, so
// an error attributed by position rather than by index shows up.
func TestPlainifyBatchNamesTheFailingEntry(t *testing.T) {
	markers := []string{"alpha", "beta", "gamma", "delta", "epsilon", "zeta"}
	fakeClaudeByMarker(t, markers, "epsilon")

	_, err := PlainifyBatch(context.Background(), markerEntries(markers), PlainifyConstraints{}, testOptions())
	if err == nil {
		t.Fatal("PlainifyBatch: expected the stub's failure to propagate")
	}
	if !strings.Contains(err.Error(), "entry id-epsilon") {
		t.Fatalf("error %q does not name the failing entry", err)
	}
}
