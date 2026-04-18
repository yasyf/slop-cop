package main

import (
	"runtime/debug"

	"github.com/spf13/cobra"
)

// version metadata is overridable at build time via -ldflags.
var (
	version = "dev"
	commit  = ""
)

func newVersionCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "version",
		Short: "Print build metadata as JSON.",
		Args:  cobra.NoArgs,
	}
	pretty := addPrettyFlag(cmd)
	cmd.RunE = func(cmd *cobra.Command, args []string) error {
		v, c := buildMetadata()
		return writeJSON(map[string]string{"version": v, "commit": c}, *pretty)
	}
	return cmd
}

// buildMetadata returns the version + commit for this build. Values injected
// via -ldflags win; otherwise we fall back to runtime/debug.BuildInfo so
// `go install` users still get a sensible module version and VCS revision.
func buildMetadata() (ver, com string) {
	ver, com = version, commit
	info, ok := debug.ReadBuildInfo()
	if !ok {
		return ver, com
	}
	if ver == "dev" && info.Main.Version != "" && info.Main.Version != "(devel)" {
		ver = info.Main.Version
	}
	if com == "" {
		for _, s := range info.Settings {
			if s.Key == "vcs.revision" {
				com = s.Value
				break
			}
		}
	}
	return ver, com
}
