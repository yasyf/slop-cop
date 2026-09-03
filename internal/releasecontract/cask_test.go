package releasecontract

import (
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
)

func caskSection(t *testing.T) string {
	t.Helper()
	_, source, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatal("locate test source")
	}
	root := filepath.Clean(filepath.Join(filepath.Dir(source), "..", ".."))
	data, err := os.ReadFile(filepath.Join(root, ".goreleaser.yaml"))
	if err != nil {
		t.Fatal(err)
	}
	var section []string
	for _, line := range strings.Split(string(data), "\n") {
		if strings.HasPrefix(line, "homebrew_casks:") {
			section = []string{line}
			continue
		}
		if section == nil {
			continue
		}
		if trimmed := strings.TrimSpace(line); trimmed != "" && !strings.HasPrefix(trimmed, "#") && !strings.HasPrefix(line, " ") {
			break
		}
		section = append(section, line)
	}
	if section == nil {
		t.Fatal("no homebrew_casks section")
	}
	return strings.Join(section, "\n")
}

func TestCaskInstallHookClearsQuarantine(t *testing.T) {
	cask := caskSection(t)
	for _, required := range []string{
		"hooks:",
		"post:",
		"install:",
		`system_command "/usr/bin/xattr"`,
		`"-dr", "com.apple.quarantine"`,
		"#{staged_path}/slop-cop",
	} {
		if !strings.Contains(cask, required) {
			t.Errorf("cask missing %q", required)
		}
	}
}
