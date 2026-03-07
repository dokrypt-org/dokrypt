package cli

import (
	goruntime "runtime"

	"github.com/spf13/cobra"
)

func newVersionCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "version",
		Short: "Show version information",
		RunE: func(cmd *cobra.Command, args []string) error {
			out := getOutput()
			out.Info("dokrypt %s", Version)
			out.Info("  commit: %s", Commit)
			out.Info("  built: %s", Date)
			out.Info("  go: %s", goruntime.Version())
			out.Info("  os/arch: %s/%s", goruntime.GOOS, goruntime.GOARCH)
			return nil
		},
	}
}
