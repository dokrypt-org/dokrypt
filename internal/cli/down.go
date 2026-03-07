package cli

import (
	"fmt"
	"time"

	"github.com/spf13/cobra"

	"github.com/dokrypt/dokrypt/internal/config"
	"github.com/dokrypt/dokrypt/internal/container"
)

func newDownCmd() *cobra.Command {
	var (
		volumes  bool
		svcNames []string
		timeout  time.Duration
	)

	cmd := &cobra.Command{
		Use:   "down",
		Short: "Stop all services",
		Long:  "Stops all running chains and services.",
		RunE: func(cmd *cobra.Command, args []string) error {
			out := getOutput()
			ctx := cmd.Context()

			cfg, err := config.Parse(getConfigPath())
			if err != nil {
				return fmt.Errorf("No dokrypt.yaml found. Run 'dokrypt init' first.")
			}

			rt, err := container.NewRuntime("")
			if err != nil {
				return fmt.Errorf("Docker is not running. Please start Docker and try again.")
			}

			svcFilter := make(map[string]bool, len(svcNames))
			for _, s := range svcNames {
				svcFilter[s] = true
			}
			filterActive := len(svcFilter) > 0

			fmt.Println()

			state, stateErr := loadState(cfg.Name)
			if stateErr == nil {
				for name, cs := range state.Containers {
					if filterActive && !svcFilter[name] {
						continue
					}
					fmt.Printf("Stopping %s...", name)
					if err := rt.StopContainer(ctx, cs.ContainerID, timeout); err != nil {
						_ = rt.StopContainer(ctx, cs.ContainerName, timeout)
					}
					_ = rt.RemoveContainer(ctx, cs.ContainerID, true)
					_ = rt.RemoveContainer(ctx, cs.ContainerName, true)
					fmt.Println(" done")
				}
			} else {
				containers, err := rt.ListContainers(ctx, container.ListOptions{
					All:    true,
					Labels: map[string]string{"dokrypt.project": cfg.Name},
				})
				if err == nil && len(containers) > 0 {
					for _, c := range containers {
						name := c.Labels["dokrypt.chain"]
						if name == "" {
							name = c.Name
						}
						if filterActive && !svcFilter[name] {
							continue
						}
						fmt.Printf("Stopping %s...", name)
						_ = rt.StopContainer(ctx, c.ID, timeout)
						_ = rt.RemoveContainer(ctx, c.ID, true)
						fmt.Println(" done")
					}
				} else {
					out.Info("No Dokrypt stack running in this directory.")
					return nil
				}
			}

			if !filterActive {
				networkName := fmt.Sprintf("dokrypt-%s", cfg.Name)
				fmt.Printf("Removing network %s...", networkName)
				networks, _ := rt.ListNetworks(ctx)
				for _, n := range networks {
					if n.Name == networkName {
						_ = rt.RemoveNetwork(ctx, n.ID)
						break
					}
				}
				fmt.Println(" done")

				_ = removeState(cfg.Name)
			}

			if volumes {
				vm := container.NewVolumeManager(rt)
				projectVolumes, listErr := vm.List(ctx, cfg.Name)
				if listErr == nil && len(projectVolumes) > 0 {
					for _, v := range projectVolumes {
						fmt.Printf("Removing volume %s...", v.Name)
						if err := vm.Remove(ctx, v.Name, true); err != nil {
							out.Warning("Failed to remove volume %s: %v", v.Name, err)
						} else {
							fmt.Println(" done")
						}
					}
				} else if len(projectVolumes) == 0 {
					out.Info("No volumes to remove.")
				}
			}

			fmt.Println()
			out.Success("Stack stopped.")
			fmt.Println()

			return nil
		},
	}

	cmd.Flags().BoolVar(&volumes, "volumes", false, "also remove volumes")
	cmd.Flags().StringSliceVar(&svcNames, "service", nil, "stop specific service(s)")
	cmd.Flags().DurationVar(&timeout, "timeout", 30*time.Second, "shutdown timeout")

	return cmd
}

func newRestartCmd() *cobra.Command {
	var svcName string

	cmd := &cobra.Command{
		Use:   "restart",
		Short: "Restart services",
		Long:  "Restarts individual service containers or the entire stack.",
		RunE: func(cmd *cobra.Command, args []string) error {
			out := getOutput()
			ctx := cmd.Context()

			cfg, err := config.Parse(getConfigPath())
			if err != nil {
				return fmt.Errorf("No dokrypt.yaml found. Run 'dokrypt init' first.")
			}

			state, err := loadState(cfg.Name)
			if err != nil {
				return fmt.Errorf("No Dokrypt stack running. Run 'dokrypt up' first.")
			}

			rt, err := container.NewRuntime("")
			if err != nil {
				return fmt.Errorf("Docker is not running.")
			}

			fmt.Println()

			if svcName != "" {
				cs, ok := state.Containers[svcName]
				if !ok {
					return fmt.Errorf("Service '%s' not found. Available: %s", svcName, containerNames(state))
				}

				fmt.Printf("Restarting %s...", svcName)
				_ = rt.StopContainer(ctx, cs.ContainerID, 10*time.Second)
				if err := rt.StartContainer(ctx, cs.ContainerID); err != nil {
					fmt.Println(" failed")
					return fmt.Errorf("failed to restart %s: %w", svcName, err)
				}
				fmt.Println(" done")
			} else {
				for name, cs := range state.Containers {
					fmt.Printf("Restarting %s...", name)
					_ = rt.StopContainer(ctx, cs.ContainerID, 10*time.Second)
					if err := rt.StartContainer(ctx, cs.ContainerID); err != nil {
						fmt.Println(" failed")
						out.Warning("Failed to restart %s: %v", name, err)
						continue
					}
					fmt.Println(" done")
				}
			}

			fmt.Println()
			out.Success("Restart complete.")
			fmt.Println()
			return nil
		},
	}

	cmd.Flags().StringVar(&svcName, "service", "", "restart specific service")
	return cmd
}
