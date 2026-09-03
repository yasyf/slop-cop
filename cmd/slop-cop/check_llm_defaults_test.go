package main

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/spf13/cobra"
)

// pathWithCodex returns a PATH holding exactly one entry: a directory
// containing an executable named codex.
func pathWithCodex(t *testing.T) string {
	t.Helper()
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "codex"), []byte("#!/bin/sh\n"), 0o755); err != nil { //nolint:gosec // G306: an executable stub on a test-owned temp PATH.
		t.Fatalf("write fake codex: %v", err)
	}
	return dir
}

// Tests for autoEnableLLM. The LLM passes are auto-enabled whenever the
// codex CLI is reachable on $PATH.

func TestAutoEnableLLM(t *testing.T) {
	t.Setenv("PATH", pathWithCodex(t))
	if !autoEnableLLM() {
		t.Fatalf("autoEnableLLM: expected true with codex on PATH")
	}
	t.Setenv("PATH", t.TempDir())
	if autoEnableLLM() {
		t.Fatalf("autoEnableLLM: expected false with codex absent from PATH")
	}
}

// Tests for resolveEffort. Precedence: --llm-effort > --llm-deep > --llm > auto.

// newCheckForTest builds the check command fresh per test so Flags().Changed
// reflects only the arguments that particular case set.
func newCheckForTest(t *testing.T, args []string) *cobra.Command {
	t.Helper()
	cmd := newCheckCmd()
	cmd.SetArgs(args)
	// ParseFlags advances only to the first non-flag arg; we don't need to
	// Execute, just to populate the flag state so resolveEffort can query
	// cmd.Flags().Changed(...).
	if err := cmd.ParseFlags(args); err != nil {
		t.Fatalf("ParseFlags(%v): %v", args, err)
	}
	return cmd
}

// TestResolveEffort exercises the full precedence table. codexOnPath drives
// whether the auto path picks "low" or "off".
func TestResolveEffort(t *testing.T) {
	cases := []struct {
		name        string
		flags       []string
		codexOnPath bool
		wantEff     llmEffort
		wantAuto    bool
	}{
		// Explicit --llm-effort is authoritative.
		{"effort=off explicit", []string{"--llm-effort=off"}, true, effortOff, false},
		{"effort=low explicit", []string{"--llm-effort=low"}, true, effortLow, false},
		{"effort=high explicit", []string{"--llm-effort=high"}, false, effortHigh, false},
		{"effort=auto with codex", []string{"--llm-effort=auto"}, true, effortLow, true},
		{"effort=auto without codex", []string{"--llm-effort=auto"}, false, effortOff, true},

		// --llm-deep alias.
		{"--llm-deep=true", []string{"--llm-deep"}, false, effortHigh, false},
		{"--llm-deep=false", []string{"--llm-deep=false"}, true, effortOff, false},

		// --llm alias.
		{"--llm=true", []string{"--llm"}, false, effortLow, false},
		{"--llm=false", []string{"--llm=false"}, true, effortOff, false},

		// --llm-deep wins over --llm when both present.
		{"--llm + --llm-deep", []string{"--llm", "--llm-deep"}, false, effortHigh, false},

		// --llm-effort wins over both aliases.
		{"effort=low + --llm-deep", []string{"--llm-effort=low", "--llm-deep"}, false, effortLow, false},

		// No flags → auto.
		{"default with codex", nil, true, effortLow, true},
		{"default without codex", nil, false, effortOff, true},
	}

	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			if c.codexOnPath {
				t.Setenv("PATH", pathWithCodex(t))
			} else {
				t.Setenv("PATH", t.TempDir())
			}

			cmd := newCheckForTest(t, c.flags)
			effortFlag, _ := cmd.Flags().GetString("llm-effort")
			llm, _ := cmd.Flags().GetBool("llm")
			deep, _ := cmd.Flags().GetBool("llm-deep")

			eff, auto, err := resolveEffort(cmd, effortFlag, llm, deep)
			if err != nil {
				t.Fatalf("resolveEffort: %v", err)
			}
			if eff != c.wantEff {
				t.Fatalf("effort=%q, want %q", eff, c.wantEff)
			}
			if auto != c.wantAuto {
				t.Fatalf("auto=%v, want %v", auto, c.wantAuto)
			}
		})
	}
}

func TestResolveEffort_InvalidFlag(t *testing.T) {
	cmd := newCheckForTest(t, []string{"--llm-effort=turbo"})
	effortFlag, _ := cmd.Flags().GetString("llm-effort")
	if _, _, err := resolveEffort(cmd, effortFlag, false, false); err == nil {
		t.Fatalf("expected error for --llm-effort=turbo")
	}
}
