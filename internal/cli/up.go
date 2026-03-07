package cli

import (
	"context"
	"errors"
	"fmt"
	"math/big"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/spf13/cobra"

	"github.com/dokrypt/dokrypt/internal/config"
	"github.com/dokrypt/dokrypt/internal/engine"
)

func newUpCmd() *cobra.Command {
	var (
		detach    bool
		build     bool
		svcNames  []string
		fresh     bool
		fork      string
		forkBlock uint64
		snapshot  string
		profile   string
		timeout   time.Duration
	)

	cmd := &cobra.Command{
		Use:   "up",
		Short: "Start all services",
		Long:  "Starts all chains and services defined in dokrypt.yaml.",
		RunE: func(cmd *cobra.Command, args []string) error {
			out := getOutput()
			startTime := time.Now()

			cfgPath := getConfigPath()
			cfg, err := config.ParseWithProfile(cfgPath, profile)
			if err != nil {
				var pathErr *os.PathError
				if errors.As(err, &pathErr) {
					return fmt.Errorf("No dokrypt.yaml found. Run 'dokrypt init' first.")
				}
				return fmt.Errorf("Failed to parse %s: %w", cfgPath, err)
			}

			svcRegistry := newDefaultRegistry()
			eng := engine.New(svcRegistry)

			if err := eng.Init(cmd.Context(), cfg); err != nil {
				return err
			}

			upCtx, cancel := context.WithCancel(cmd.Context())
			defer cancel()
			sigCh := make(chan os.Signal, 1)
			signal.Notify(sigCh, os.Interrupt, syscall.SIGTERM)
			go func() {
				<-sigCh
				fmt.Println("\nReceived interrupt, stopping...")
				cancel()
				eng.Cleanup(context.Background())
				os.Exit(1)
			}()

			fmt.Println()
			opts := engine.UpOptions{
				Detach:    detach,
				Build:     build,
				Services:  svcNames,
				Fresh:     fresh,
				Fork:      fork,
				ForkBlock: forkBlock,
				Snapshot:  snapshot,
				Profile:   profile,
				Timeout:   timeout,
			}

			if err := eng.Up(upCtx, opts); err != nil {
				return err
			}

			state := &ProjectState{
				Project:    cfg.Name,
				StartedAt:  time.Now(),
				Containers: make(map[string]ContainerState),
			}

			for _, c := range eng.Chains() {
				accounts := c.Accounts()
				if len(accounts) > 0 {
					fmt.Println()
					out.Info("Accounts:")
					for i, a := range accounts {
						if i >= 3 {
							break
						}
						balETH := "10000"
						if a.Balance != nil {
							weiPerEth := new(big.Int).Exp(big.NewInt(10), big.NewInt(18), nil)
							ethBal := new(big.Int).Div(a.Balance, weiPerEth)
							balETH = ethBal.String()
						}
						fmt.Printf("  [%d] %s (%s ETH)\n", i, a.Address, balETH)
					}
				}

				state.Containers[c.Name()] = ContainerState{
					ContainerID:   eng.ChainContainerID(c.Name()),
					ContainerName: fmt.Sprintf("dokrypt-%s-%s", cfg.Name, c.Name()),
					Image:         "ghcr.io/foundry-rs/foundry:latest",
					Ports:         map[string]int{"8545": hostPortFromURL(c.RPCURL())},
					Status:        "running",
				}
			}

			svcCount := 0
			for _, svc := range eng.Services() {
				ports := make(map[string]int)
				for k, v := range svc.Ports() {
					ports[k] = v
				}
				state.Containers[svc.Name()] = ContainerState{
					ContainerID:   eng.ServiceContainerID(svc.Name()),
					ContainerName: fmt.Sprintf("dokrypt-%s-%s", cfg.Name, svc.Name()),
					Image:         svc.Type(),
					Ports:         ports,
					Status:        "running",
				}
				svcCount++
			}

			fmt.Println()
			headers := []string{"Service", "Status", "Port", "URL"}
			var rows [][]string
			totalRunning := 0
			for _, c := range eng.Chains() {
				port := hostPortFromURL(c.RPCURL())
				rows = append(rows, []string{
					fmt.Sprintf("%s (%s)", c.Name(), c.Engine()),
					"● Ready",
					fmt.Sprintf("%d", port),
					c.RPCURL(),
				})
				totalRunning++
			}
			for _, svc := range eng.Services() {
				status := "● Ready"
				port := "—"
				url := "—"
				if svc.Health(cmd.Context()) != nil {
					status = "○ Failed"
				} else {
					totalRunning++
				}
				urls := svc.URLs()
				if u, ok := urls["http"]; ok {
					url = u
					port = fmt.Sprintf("%d", hostPortFromURL(u))
				} else if u, ok := urls["api"]; ok {
					url = u
					port = fmt.Sprintf("%d", hostPortFromURL(u))
				}
				rows = append(rows, []string{
					fmt.Sprintf("%s (%s)", svc.Name(), svc.Type()),
					status, port, url,
				})
			}
			out.Table(headers, rows)

			elapsed := time.Since(startTime)
			fmt.Println()
			out.Success("Stack ready in %.1fs — %d services running", elapsed.Seconds(), totalRunning)
			fmt.Println()

			if err := saveState(state); err != nil {
				out.Warning("Failed to save state: %v", err)
			}

			return nil
		},
	}

	cmd.Flags().BoolVarP(&detach, "detach", "d", false, "run in background")
	cmd.Flags().BoolVar(&build, "build", false, "rebuild images")
	cmd.Flags().StringSliceVar(&svcNames, "service", nil, "start specific service(s)")
	cmd.Flags().BoolVar(&fresh, "fresh", false, "destroy existing state, start clean")
	cmd.Flags().StringVar(&fork, "fork", "", "fork a live network")
	cmd.Flags().Uint64Var(&forkBlock, "fork-block", 0, "fork at specific block")
	cmd.Flags().StringVar(&snapshot, "snapshot", "", "start from snapshot")
	cmd.Flags().StringVar(&profile, "profile", "", "config profile (dev, test, staging)")
	cmd.Flags().DurationVar(&timeout, "timeout", 5*time.Minute, "startup timeout")

	return cmd
}

func hostPortFromURL(url string) int {
	for i := len(url) - 1; i >= 0; i-- {
		if url[i] == ':' {
			port := 0
			fmt.Sscanf(url[i+1:], "%d", &port)
			return port
		}
	}
	return 0
}
