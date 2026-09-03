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

// clearEffortEnv drops the effort variable the test process may have
// inherited, so a case states its own conditions.
func clearEffortEnv(t *testing.T) {
	t.Helper()
	t.Setenv(envLLMEffort, "")
}

// Tests for resolveEffort. Precedence:
// --llm-effort > --no-llm > --llm-deep > --llm > $SLOP_COP_LLM > auto.

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
func callResolveEffort(t *testing.T, cmd *cobra.Command) (llmEffort, bool, error) {
	t.Helper()
	effortFlag, _ := cmd.Flags().GetString("llm-effort")
	llm, _ := cmd.Flags().GetBool("llm")
	deep, _ := cmd.Flags().GetBool("llm-deep")
	noLLM, _ := cmd.Flags().GetBool("no-llm")
	return resolveEffort(cmd, effortFlag, llm, deep, noLLM)
}

func TestResolveEffort(t *testing.T) {
	cases := []struct {
		name        string
		flags       []string
		env         string
		codexOnPath bool
		wantEff     llmEffort
		wantAuto    bool
	}{
		// Explicit --llm-effort is authoritative.
		{"effort=off explicit", []string{"--llm-effort=off"}, "", true, effortOff, false},
		{"effort=low explicit", []string{"--llm-effort=low"}, "", true, effortLow, false},
		{"effort=high explicit", []string{"--llm-effort=high"}, "", false, effortHigh, false},
		{"effort=auto with codex", []string{"--llm-effort=auto"}, "", true, effortLow, true},
		{"effort=auto without codex", []string{"--llm-effort=auto"}, "", false, effortOff, true},

		// --no-llm alias.
		{"--no-llm", []string{"--no-llm"}, "", true, effortOff, false},
		{"--no-llm=false falls through to auto", []string{"--no-llm=false"}, "", true, effortLow, true},

		// --llm-deep alias.
		{"--llm-deep=true", []string{"--llm-deep"}, "", false, effortHigh, false},
		{"--llm-deep=false", []string{"--llm-deep=false"}, "", true, effortOff, false},

		// --llm alias.
		{"--llm=true", []string{"--llm"}, "", false, effortLow, false},
		{"--llm=false", []string{"--llm=false"}, "", true, effortOff, false},

		// --llm-deep wins over --llm when both present.
		{"--llm + --llm-deep", []string{"--llm", "--llm-deep"}, "", false, effortHigh, false},

		// --no-llm names an outcome, not a tier, so it wins over both.
		{"--no-llm + --llm-deep", []string{"--no-llm", "--llm-deep"}, "", true, effortOff, false},
		{"--no-llm + --llm", []string{"--no-llm", "--llm"}, "", true, effortOff, false},

		// --llm-effort wins over every alias.
		{"effort=low + --llm-deep", []string{"--llm-effort=low", "--llm-deep"}, "", false, effortLow, false},
		{"effort=high + --no-llm", []string{"--llm-effort=high", "--no-llm"}, "", false, effortHigh, false},

		// $SLOP_COP_LLM applies when no flag does, and loses to every flag.
		{"env off", nil, "off", true, effortOff, false},
		{"env high", nil, "high", false, effortHigh, false},
		{"env auto with codex", nil, "auto", true, effortLow, true},
		{"env is case-insensitive", nil, "OFF", true, effortOff, false},
		{"env empty falls through to auto", nil, "", true, effortLow, true},
		{"env loses to --llm", []string{"--llm"}, "off", false, effortLow, false},
		{"env loses to --llm-effort", []string{"--llm-effort=high"}, "off", false, effortHigh, false},

		// No flags, no env → auto.
		{"default with codex", nil, "", true, effortLow, true},
		{"default without codex", nil, "", false, effortOff, true},
	}

	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			t.Setenv(envLLMEffort, c.env)
			if c.codexOnPath {
				t.Setenv("PATH", pathWithCodex(t))
			} else {
				t.Setenv("PATH", t.TempDir())
			}

			eff, auto, err := callResolveEffort(t, newCheckForTest(t, c.flags))
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
	clearEffortEnv(t)
	if _, _, err := callResolveEffort(t, newCheckForTest(t, []string{"--llm-effort=turbo"})); err == nil {
		t.Fatalf("expected error for --llm-effort=turbo")
	}
}

func TestResolveEffort_InvalidEnv(t *testing.T) {
	t.Setenv(envLLMEffort, "turbo")
	if _, _, err := callResolveEffort(t, newCheckForTest(t, nil)); err == nil {
		t.Fatalf("expected error for %s=turbo", envLLMEffort)
	}
}
