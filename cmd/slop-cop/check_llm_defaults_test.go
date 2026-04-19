package main

import (
	"os"
	"testing"

	"github.com/spf13/cobra"
)

// realExecutable returns an absolute path to a binary guaranteed to exist
// on every supported platform: the test binary itself. Tests that need
// "a binary exec.LookPath will find" use this instead of hard-coding
// /bin/sh, which doesn't exist on Windows.
func realExecutable(t *testing.T) string {
	t.Helper()
	p, err := os.Executable()
	if err != nil {
		t.Fatalf("os.Executable: %v", err)
	}
	return p
}

// Tests for pluginEnvActive + autoEnableLLM. autoEnableLLM must require
// BOTH a plugin-env signal AND a reachable claude binary so the CLI never
// silently tries to spend credits outside a plugin context.

func TestPluginEnvActive(t *testing.T) {
	t.Setenv("CLAUDE_PLUGIN_ROOT", "")
	t.Setenv("CURSOR_PLUGIN_ROOT", "")
	if pluginEnvActive() {
		t.Fatalf("pluginEnvActive: expected false with both env empty")
	}
	t.Setenv("CLAUDE_PLUGIN_ROOT", "/x")
	if !pluginEnvActive() {
		t.Fatalf("pluginEnvActive: expected true with CLAUDE_PLUGIN_ROOT set")
	}
	t.Setenv("CLAUDE_PLUGIN_ROOT", "")
	t.Setenv("CURSOR_PLUGIN_ROOT", "/y")
	if !pluginEnvActive() {
		t.Fatalf("pluginEnvActive: expected true with CURSOR_PLUGIN_ROOT set")
	}
}

func TestAutoEnableLLM(t *testing.T) {
	cases := []struct {
		name    string
		claude  string
		cEnv    string
		curEnv  string
		want    bool
	}{
		{"no env, no bin", "nonexistent-binary-xyzzy", "", "", false},
		{"claude env, missing bin", "nonexistent-binary-xyzzy", "/tmp/fake", "", false},
		{"cursor env, missing bin", "nonexistent-binary-xyzzy", "", "/tmp/fake", false},
		{"claude env, real bin", "", "/tmp/fake", "", true},
		{"cursor env, real bin", "", "", "/tmp/fake", true},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			t.Setenv("CLAUDE_PLUGIN_ROOT", c.cEnv)
			t.Setenv("CURSOR_PLUGIN_ROOT", c.curEnv)
			bin := c.claude
			if bin == "" {
				bin = realExecutable(t)
			}
			if got := autoEnableLLM(bin); got != c.want {
				t.Fatalf("autoEnableLLM=%v, want %v", got, c.want)
			}
		})
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

// TestResolveEffort exercises the full precedence table. realBin is used
// whenever the auto path should pick "high"; "missing-bin" when it should
// pick "off".
func TestResolveEffort(t *testing.T) {
	realBin := realExecutable(t)
	const missingBin = "nonexistent-binary-xyzzy"

	cases := []struct {
		name      string
		flags     []string
		pluginEnv bool
		bin       string
		wantEff   llmEffort
		wantAuto  bool
	}{
		// Explicit --llm-effort is authoritative.
		{"effort=off explicit", []string{"--llm-effort=off"}, true, realBin, effortOff, false},
		{"effort=low explicit", []string{"--llm-effort=low"}, true, realBin, effortLow, false},
		{"effort=high explicit", []string{"--llm-effort=high"}, false, missingBin, effortHigh, false},
		{"effort=auto under plugin", []string{"--llm-effort=auto"}, true, realBin, effortHigh, true},
		{"effort=auto outside plugin", []string{"--llm-effort=auto"}, false, missingBin, effortOff, true},

		// --llm-deep alias.
		{"--llm-deep=true", []string{"--llm-deep"}, false, missingBin, effortHigh, false},
		{"--llm-deep=false", []string{"--llm-deep=false"}, true, realBin, effortOff, false},

		// --llm alias.
		{"--llm=true", []string{"--llm"}, false, missingBin, effortLow, false},
		{"--llm=false", []string{"--llm=false"}, true, realBin, effortOff, false},

		// --llm-deep wins over --llm when both present.
		{"--llm + --llm-deep", []string{"--llm", "--llm-deep"}, false, missingBin, effortHigh, false},

		// --llm-effort wins over both aliases.
		{"effort=low + --llm-deep", []string{"--llm-effort=low", "--llm-deep"}, false, missingBin, effortLow, false},

		// No flags → auto.
		{"default under plugin", nil, true, realBin, effortHigh, true},
		{"default outside plugin", nil, false, missingBin, effortOff, true},
	}

	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			if c.pluginEnv {
				t.Setenv("CLAUDE_PLUGIN_ROOT", "/tmp/fake-plugin")
			} else {
				t.Setenv("CLAUDE_PLUGIN_ROOT", "")
			}
			t.Setenv("CURSOR_PLUGIN_ROOT", "")

			cmd := newCheckForTest(t, c.flags)
			effortFlag, _ := cmd.Flags().GetString("llm-effort")
			llm, _ := cmd.Flags().GetBool("llm")
			deep, _ := cmd.Flags().GetBool("llm-deep")

			eff, auto, err := resolveEffort(cmd, effortFlag, llm, deep, c.bin)
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
	t.Setenv("CLAUDE_PLUGIN_ROOT", "")
	t.Setenv("CURSOR_PLUGIN_ROOT", "")
	cmd := newCheckForTest(t, []string{"--llm-effort=turbo"})
	effortFlag, _ := cmd.Flags().GetString("llm-effort")
	if _, _, err := resolveEffort(cmd, effortFlag, false, false, realExecutable(t)); err == nil {
		t.Fatalf("expected error for --llm-effort=turbo")
	}
}
