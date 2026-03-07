package cli

import (
	"bufio"
	"fmt"
	"io"
	"os"
	"os/signal"
	"sync"
	"syscall"

	"github.com/spf13/cobra"

	"github.com/dokrypt/dokrypt/internal/config"
	"github.com/dokrypt/dokrypt/internal/container"
)

var serviceColors = []string{
	"\033[34m", // blue
	"\033[36m", // cyan
	"\033[35m", // magenta
	"\033[33m", // yellow
	"\033[32m", // green
}

const colorReset = "\033[0m"

func newLogsCmd() *cobra.Command {
	var (
		svcName    string
		follow     bool
		tailLines  string
		since      string
		timestamps bool
	)

	cmd := &cobra.Command{
		Use:   "logs",
		Short: "Stream service logs",
		RunE: func(cmd *cobra.Command, args []string) error {
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

			targets := make(map[string]string) // name -> container ID
			if svcName != "" {
				cs, ok := state.Containers[svcName]
				if !ok {
					return fmt.Errorf("Service '%s' not found. Available: %s", svcName, containerNames(state))
				}
				targets[svcName] = cs.ContainerID
			} else {
				for name, cs := range state.Containers {
					targets[name] = cs.ContainerID
				}
			}

			if len(targets) == 0 {
				fmt.Println("No services running.")
				return nil
			}

			sigCh := make(chan os.Signal, 1)
			signal.Notify(sigCh, os.Interrupt, syscall.SIGTERM)

			ctx := cmd.Context()
			logOpts := container.LogOptions{
				Follow:     follow,
				Tail:       tailLines,
				Since:      since,
				Timestamps: timestamps,
				Stdout:     true,
				Stderr:     true,
			}

			if len(targets) == 1 {
				for name, id := range targets {
					reader, err := rt.ContainerLogs(ctx, id, logOpts)
					if err != nil {
						return fmt.Errorf("failed to get logs for %s: %w", name, err)
					}
					defer reader.Close()

					go func() {
						<-sigCh
						reader.Close()
					}()

					streamDockerLogs(reader, "", os.Stdout)
					return nil
				}
			}

			var wg sync.WaitGroup
			colorIdx := 0
			for name, id := range targets {
				reader, err := rt.ContainerLogs(ctx, id, logOpts)
				if err != nil {
					fmt.Fprintf(os.Stderr, "failed to get logs for %s: %v\n", name, err)
					continue
				}

				color := serviceColors[colorIdx%len(serviceColors)]
				colorIdx++
				prefix := fmt.Sprintf("%s[%-12s]%s ", color, name, colorReset)

				wg.Add(1)
				go func(r io.ReadCloser, pfx string) {
					defer wg.Done()
					defer r.Close()
					streamDockerLogs(r, pfx, os.Stdout)
				}(reader, prefix)
			}

			go func() {
				<-sigCh
				fmt.Println()
				os.Exit(0)
			}()

			wg.Wait()
			return nil
		},
	}

	cmd.Flags().StringVarP(&svcName, "service", "s", "", "specific service")
	cmd.Flags().BoolVarP(&follow, "follow", "f", false, "follow mode")
	cmd.Flags().StringVar(&tailLines, "tail", "50", "last N lines")
	cmd.Flags().StringVar(&since, "since", "", "logs since duration (e.g. 5m, 1h)")
	cmd.Flags().BoolVar(&timestamps, "timestamps", false, "show timestamps")

	return cmd
}

func streamDockerLogs(reader io.Reader, prefix string, out io.Writer) {
	scanner := bufio.NewScanner(reader)
	scanner.Buffer(make([]byte, 0, 64*1024), 1024*1024)
	for scanner.Scan() {
		line := scanner.Text()
		if len(line) > 8 && (line[0] == 1 || line[0] == 2) {
			line = line[8:]
		}
		if prefix != "" {
			fmt.Fprintf(out, "%s%s\n", prefix, line)
		} else {
			fmt.Fprintln(out, line)
		}
	}
}

func containerNames(state *ProjectState) string {
	names := ""
	for name := range state.Containers {
		if names != "" {
			names += ", "
		}
		names += name
	}
	return names
}
