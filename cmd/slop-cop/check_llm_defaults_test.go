package main

import (
	"os"
	"testing"
)

// realExecutable returns an absolute path to a binary guaranteed to exist
// on every supported platform: the test binary itself, reported by
// os.Executable(). Tests that need "a binary exec.LookPath will find" use
// this instead of hard-coding /bin/sh, which doesn't exist on Windows.
func realExecutable(t *testing.T) string {
	t.Helper()
	p, err := os.Executable()
	if err != nil {
		t.Fatalf("os.Executable: %v", err)
	}
	return p
}

// Tests for the LLM auto-default logic. autoEnableLLM must require BOTH
// a plugin-env signal AND a reachable claude binary so the CLI never
// silently tries to spend credits outside a plugin context.

func TestAutoEnableLLM_NoPluginEnvNoClaudeBin(t *testing.T) {
	t.Setenv("CLAUDE_PLUGIN_ROOT", "")
	t.Setenv("CURSOR_PLUGIN_ROOT", "")
	if autoEnableLLM("nonexistent-binary-xyzzy") {
		t.Fatalf("autoEnableLLM should be false without plugin env")
	}
}

func TestAutoEnableLLM_PluginEnvButNoBinary(t *testing.T) {
	t.Setenv("CLAUDE_PLUGIN_ROOT", "/tmp/fake-plugin")
	t.Setenv("CURSOR_PLUGIN_ROOT", "")
	if autoEnableLLM("nonexistent-binary-xyzzy") {
		t.Fatalf("autoEnableLLM should be false when claude bin missing")
	}
}

func TestAutoEnableLLM_PluginEnvAndBinary(t *testing.T) {
	t.Setenv("CLAUDE_PLUGIN_ROOT", "/tmp/fake-plugin")
	t.Setenv("CURSOR_PLUGIN_ROOT", "")
	if !autoEnableLLM(realExecutable(t)) {
		t.Fatalf("autoEnableLLM should be true with plugin env + reachable bin")
	}
}

func TestAutoEnableLLM_CursorPluginEnvAlone(t *testing.T) {
	// Only CURSOR_PLUGIN_ROOT set; Claude env absent. Still valid trigger.
	t.Setenv("CLAUDE_PLUGIN_ROOT", "")
	t.Setenv("CURSOR_PLUGIN_ROOT", "/tmp/fake-cursor-plugin")
	if !autoEnableLLM(realExecutable(t)) {
		t.Fatalf("autoEnableLLM should honour CURSOR_PLUGIN_ROOT as plugin signal")
	}
}

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
}
