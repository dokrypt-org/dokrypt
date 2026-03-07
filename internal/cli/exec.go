package cli

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"

	"github.com/dokrypt/dokrypt/internal/config"
	"github.com/dokrypt/dokrypt/internal/container"
)

func newExecCmd() *cobra.Command {
	var (
		interactive bool
		tty         bool
	)

	cmd := &cobra.Command{
		Use:   "exec <service> <command> [args...]",
		Short: "Execute command in service container",
		Long: `Execute a command inside a running service container.

Examples:
  dokrypt exec ethereum sh
  dokrypt exec ethereum cast block latest
  dokrypt exec ipfs ipfs id
  dokrypt exec ethereum cat /proc/1/cmdline`,
		Args: cobra.MinimumNArgs(2),
		RunE: func(cmd *cobra.Command, args []string) error {
			svcName := args[0]
			execCmd := args[1:]

			cfg, err := config.Parse(getConfigPath())
			if err != nil {
				return fmt.Errorf("No dokrypt.yaml found. Run 'dokrypt init' first.")
			}

			state, err := loadState(cfg.Name)
			if err != nil {
				return fmt.Errorf("No Dokrypt stack running. Run 'dokrypt up' first.")
			}

			cs, ok := state.Containers[svcName]
			if !ok {
				return fmt.Errorf("Service '%s' not found. Available: %s", svcName, containerNames(state))
			}

			rt, err := container.NewRuntime("")
			if err != nil {
				return fmt.Errorf("Docker is not running.")
			}

			info, err := rt.InspectContainer(cmd.Context(), cs.ContainerID)
			if err != nil || info.State != "running" {
				info, err = rt.InspectContainer(cmd.Context(), cs.ContainerName)
				if err != nil || info.State != "running" {
					return fmt.Errorf("Service '%s' is not running.", svcName)
				}
				cs.ContainerID = info.ID
			}

			result, err := rt.ExecInContainer(cmd.Context(), cs.ContainerID, execCmd, container.ExecOptions{
				Stdin:       os.Stdin,
				Stdout:      os.Stdout,
				Stderr:      os.Stderr,
				Interactive: interactive,
				TTY:         tty,
			})
			if err != nil {
				return fmt.Errorf("exec failed: %w", err)
			}

			if result != nil {
				if result.Stdout != "" {
					fmt.Print(result.Stdout)
				}
				if result.Stderr != "" {
					fmt.Fprint(os.Stderr, result.Stderr)
				}
				if result.ExitCode != 0 {
					return fmt.Errorf("command exited with code %d", result.ExitCode)
				}
			}

			return nil
		},
	}

	cmd.Flags().BoolVarP(&interactive, "interactive", "i", false, "keep stdin open")
	cmd.Flags().BoolVarP(&tty, "tty", "t", false, "allocate a pseudo-TTY")

	cmd.Flags().SetInterspersed(false)

	return cmd
}
