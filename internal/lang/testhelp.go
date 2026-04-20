package lang

import (
	"strings"
	"testing"
)

// AssertAnalyzeInvariants runs Analyzer.Analyze on src and verifies the
// cross-language masking invariants described on the Analyzer interface.
// Every language package should call this from its Analyze tests so the
// pipeline's core assumption (byte-aligned masked text, preserved newlines)
// can't silently regress.
func AssertAnalyzeInvariants(t *testing.T, a Analyzer, src string) {
	t.Helper()
	masked, _, err := a.Analyze(src)
	if err != nil {
		t.Fatalf("%s.Analyze(%d bytes): %v", a.Name(), len(src), err)
	}
	if len(masked) != len(src) {
		t.Fatalf("%s: len(masked)=%d, len(src)=%d", a.Name(), len(masked), len(src))
	}
	for i := 0; i < len(src); i++ {
		switch {
		case src[i] == '\n':
			if masked[i] != '\n' {
				t.Fatalf("%s: newline at offset %d not preserved (got %q)", a.Name(), i, masked[i])
			}
		case masked[i] == src[i]:
			// unmasked byte, fine.
		case masked[i] == ' ':
			// masked byte, fine.
		default:
			t.Fatalf("%s: offset %d differs from src but is not a space (src=%q masked=%q)",
				a.Name(), i, src[i], masked[i])
		}
	}
	// Spot-check: if the source contains at least one non-space byte that
	// survived, the analyzer didn't accidentally mask everything.
	if len(src) > 0 && strings.TrimSpace(src) != "" && strings.TrimSpace(masked) == "" {
		t.Logf("%s: every non-space byte was masked — fine if the input is pure code, suspicious otherwise", a.Name())
	}
}
