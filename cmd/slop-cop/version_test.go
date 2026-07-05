package main

import (
	"bytes"
	"testing"
)

// TestVersionFlag pins the plugin-installer contract: `slop-cop --version`
// prints exactly the stamped version string, one line, no decoration. The
// installer compares it (v-stripped) against the target release tag.
func TestVersionFlag(t *testing.T) {
	cases := []struct {
		name    string
		stamped string
		want    string
	}{
		{"bare ldflags stamp", "0.1.99", "0.1.99\n"},
		{"v-prefixed stamp", "v0.2.0", "v0.2.0\n"},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			prev := version
			version = c.stamped
			t.Cleanup(func() { version = prev })

			root := newRoot()
			var out bytes.Buffer
			root.SetOut(&out)
			root.SetArgs([]string{"--version"})
			if err := root.Execute(); err != nil {
				t.Fatalf("Execute(--version) err = %v", err)
			}
			if got := out.String(); got != c.want {
				t.Fatalf("--version output = %q, want %q", got, c.want)
			}
		})
	}
}
