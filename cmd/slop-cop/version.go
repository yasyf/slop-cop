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
	var pretty bool
	cmd := &cobra.Command{
		Use:   "version",
		Short: "Print build metadata as JSON.",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			v := version
			c := commit
			if (v == "dev" || c == "") && v != "" {
				if info, ok := debug.ReadBuildInfo(); ok {
					if v == "dev" && info.Main.Version != "" && info.Main.Version != "(devel)" {
						v = info.Main.Version
					}
					if c == "" {
						for _, s := range info.Settings {
							if s.Key == "vcs.revision" {
								c = s.Value
								break
							}
						}
					}
				}
			}
			return writeJSON(map[string]string{
				"version": v,
				"commit":  c,
			}, pretty)
		},
	}
	cmd.Flags().BoolVar(&pretty, "pretty", false, "Indent JSON output.")
	return cmd
}
