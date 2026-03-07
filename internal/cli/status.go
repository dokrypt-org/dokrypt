package cli

import (
	"context"
	"fmt"
	"net/http"
	"os"
	"os/signal"
	"strings"
	"time"

	"github.com/spf13/cobra"

	"github.com/dokrypt/dokrypt/internal/common"
	"github.com/dokrypt/dokrypt/internal/config"
	"github.com/dokrypt/dokrypt/internal/container"
)

func newStatusCmd() *cobra.Command {
	var (
		watch   bool
		svcName string
	)

	cmd := &cobra.Command{
		Use:   "status",
		Short: "Show environment status",
		RunE: func(cmd *cobra.Command, args []string) error {
			out := getOutput()

			if watch {
				return statusWatch(cmd, out, svcName)
			}
			return statusOnce(cmd, out, svcName)
		},
	}

	cmd.Flags().BoolVarP(&watch, "watch", "w", false, "live-updating status")
	cmd.Flags().StringVar(&svcName, "service", "", "specific service")

	return cmd
}

func statusOnce(cmd *cobra.Command, out common.Output, svcName string) error {
	cfgPath := getConfigPath()
	cfg, err := config.Parse(cfgPath)
	if err != nil {
		return fmt.Errorf("No dokrypt.yaml found. Run 'dokrypt init' first.")
	}

	state, stateErr := loadState(cfg.Name)
	if stateErr != nil {
		return statusFromDocker(cmd.Context(), out, cfg.Name)
	}

	rt, err := container.NewRuntime("")
	if err != nil {
		return fmt.Errorf("failed to connect to Docker: %w", err)
	}

	fmt.Println()
	headers := []string{"Service", "Status", "Port", "URL"}
	var rows [][]string
	anyRunning := false

	for name, cs := range state.Containers {
		if svcName != "" && name != svcName {
			continue
		}

		status := "○ Stopped"
		port := "—"
		url := "—"

		typeLabel := cs.Image
		if _, isChain := cfg.Chains[name]; isChain {
			typeLabel = cfg.Chains[name].Engine
		} else if svcCfg, isSvc := cfg.Services[name]; isSvc {
			typeLabel = svcCfg.Type
		}

		info, err := rt.InspectContainer(cmd.Context(), cs.ContainerID)
		if err == nil && info.State == "running" {
			for _, portNum := range cs.Ports {
				if portNum > 0 {
					portURL := fmt.Sprintf("http://localhost:%d", portNum)
					if isHTTPHealthy(portURL) {
						status = "● Ready"
					} else {
						status = "● Running"
					}
					port = fmt.Sprintf("%d", portNum)
					url = portURL
					anyRunning = true
					break
				}
			}
		}

		rows = append(rows, []string{
			fmt.Sprintf("%s (%s)", name, typeLabel),
			status, port, url,
		})
	}

	if svcName != "" && len(rows) == 0 {
		out.Info("Service %q not found in the running stack.", svcName)
		return nil
	}

	if !anyRunning {
		out.Info("No Dokrypt stack running in this directory.")
		return nil
	}

	out.Table(headers, rows)
	fmt.Println()
	return nil
}

func statusWatch(cmd *cobra.Command, out common.Output, svcName string) error {
	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, os.Interrupt)
	defer signal.Stop(sigCh)

	ticker := time.NewTicker(2 * time.Second)
	defer ticker.Stop()

	fmt.Print("\033[2J\033[H") // clear screen + move cursor to top
	if err := statusOnce(cmd, out, svcName); err != nil {
		return err
	}

	for {
		select {
		case <-sigCh:
			fmt.Println()
			return nil
		case <-ticker.C:
			fmt.Print("\033[2J\033[H")
			if err := statusOnce(cmd, out, svcName); err != nil {
				return err
			}
		}
	}
}

func statusFromDocker(ctx context.Context, out interface{ Info(string, ...any) }, projectName string) error {
	rt, err := container.NewRuntime("")
	if err != nil {
		return fmt.Errorf("failed to connect to Docker: %w", err)
	}

	containers, err := rt.ListContainers(ctx, container.ListOptions{
		Labels: map[string]string{"dokrypt.project": projectName},
	})
	if err != nil || len(containers) == 0 {
		fmt.Println("No Dokrypt stack running in this directory.")
		return nil
	}

	fmt.Println()
	for _, c := range containers {
		chainName := c.Labels["dokrypt.chain"]
		engineName := c.Labels["dokrypt.engine"]
		status := c.State
		portStr := "—"
		for _, p := range []int{8545, 8546, 8547} {
			if hp, ok := c.Ports[p]; ok {
				portStr = fmt.Sprintf("%d", hp)
				break
			}
		}
		fmt.Printf("  %s (%s)  %s  port:%s\n", chainName, engineName, status, portStr)
	}
	fmt.Println()
	return nil
}

func isHTTPHealthy(url string) bool {
	client := &http.Client{Timeout: 2 * time.Second}
	resp, err := client.Get(url)
	if err != nil {
		resp, err = client.Post(url, "application/json", strings.NewReader(""))
		if err != nil {
			return false
		}
	}
	resp.Body.Close()
	return resp.StatusCode < 500
}
