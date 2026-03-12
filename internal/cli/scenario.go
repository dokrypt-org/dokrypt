package cli

import (
	"encoding/json"
	"fmt"
	"sort"
	"strings"
	"time"

	"github.com/dokrypt/dokrypt/internal/config"
	"github.com/dokrypt/dokrypt/internal/scenario"
	"github.com/spf13/cobra"
)

func newScenarioCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "scenario",
		Short: "Run stress-test scenarios against your local chain",
	}

	cmd.AddCommand(
		newScenarioListCmd(),
		newScenarioRunCmd(),
		newScenarioResetCmd(),
	)

	return cmd
}

func newScenarioListCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "list",
		Short: "List all available scenarios",
		RunE: func(cmd *cobra.Command, args []string) error {
			out := getOutput()
			registry := scenario.NewRegistry()

			fmt.Println()
			headers := []string{"Scenario", "Description", "Flags"}
			var rows [][]string
			for _, s := range registry.List() {
				flags := ""
				if len(s.Flags) > 0 {
					var parts []string
					for _, f := range s.Flags {
						parts = append(parts, fmt.Sprintf("--%s (default: %s)", f.Name, f.DefaultValue))
					}
					flags = strings.Join(parts, ", ")
				}
				rows = append(rows, []string{s.Name, s.Description, flags})
			}
			out.Table(headers, rows)
			fmt.Println()
			out.Info("Usage: dokrypt scenario run <name> [flags]")
			fmt.Println()
			return nil
		},
	}
}

func newScenarioRunCmd() *cobra.Command {
	var (
		severity int
		hours    int
		percent  int
		gwei     int
	)

	cmd := &cobra.Command{
		Use:   "run <scenario>",
		Short: "Run a scenario",
		Args:  requireArgs(1, "dokrypt scenario run <scenario>"),
		RunE: func(cmd *cobra.Command, args []string) error {
			out := getOutput()
			name := args[0]

			registry := scenario.NewRegistry()
			s, err := registry.Get(name)
			if err != nil {
				return err
			}

			rpcURL, err := getChainRPC("")
			if err != nil {
				return err
			}

			cfg, err := config.Parse(getConfigPath())
			if err != nil {
				return fmt.Errorf("No dokrypt.yaml found.")
			}

			snapName := fmt.Sprintf("pre-scenario-%s-%d", name, time.Now().Unix())
			out.Info("Saving snapshot %q before scenario...", snapName)

			result, err := rpcCall(rpcURL, "evm_snapshot")
			if err != nil {
				return fmt.Errorf("failed to take pre-scenario snapshot: %w", err)
			}

			var snapID string
			json.Unmarshal(result, &snapID)

			blockNum, _ := getCurrentBlock(rpcURL)
			chainID, _ := getChainID(rpcURL)

			meta := SnapshotMetadata{
				Name:        snapName,
				Project:     cfg.Name,
				Description: fmt.Sprintf("Auto-saved before scenario: %s", name),
				Tags:        []string{"scenario", name},
				CreatedAt:   time.Now(),
				ChainID:     chainID,
				BlockNumber: blockNum,
				SnapshotID:  snapID,
			}
			if err := saveSnapshotMeta(cfg.Name, meta); err != nil {
				return fmt.Errorf("failed to save snapshot metadata: %w", err)
			}

			out.Success("Snapshot saved: %s (block #%d)", snapName, blockNum)
			fmt.Println()
			out.Info("Running scenario: %s", s.Name)
			out.Info("  %s", s.Description)
			fmt.Println()

			opts := map[string]string{}
			if cmd.Flags().Changed("severity") {
				opts["severity"] = fmt.Sprintf("%d", severity)
			}
			if cmd.Flags().Changed("hours") {
				opts["hours"] = fmt.Sprintf("%d", hours)
			}
			if cmd.Flags().Changed("percent") {
				opts["percent"] = fmt.Sprintf("%d", percent)
			}
			if cmd.Flags().Changed("gwei") {
				opts["gwei"] = fmt.Sprintf("%d", gwei)
			}

			if err := s.Run(rpcURL, opts); err != nil {
				return fmt.Errorf("scenario %q failed: %w", name, err)
			}

			fmt.Println()
			out.Success("Scenario %q complete!", name)
			out.Info("To restore: dokrypt scenario reset")
			return nil
		},
	}

	cmd.Flags().IntVar(&severity, "severity", 50, "price drop percentage (market-crash)")
	cmd.Flags().IntVar(&hours, "hours", 6, "hours of staleness (oracle-failure)")
	cmd.Flags().IntVar(&percent, "percent", 80, "percentage of liquidity removed (liquidity-drain)")
	cmd.Flags().IntVar(&gwei, "gwei", 500, "target gas price in gwei (gas-spike)")

	return cmd
}

func newScenarioResetCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "reset",
		Short: "Restore to the most recent pre-scenario snapshot",
		RunE: func(cmd *cobra.Command, args []string) error {
			out := getOutput()

			cfg, err := config.Parse(getConfigPath())
			if err != nil {
				return fmt.Errorf("No dokrypt.yaml found.")
			}

			snapshots, err := listSnapshotMetas(cfg.Name)
			if err != nil || len(snapshots) == 0 {
				return fmt.Errorf("No snapshots found. Nothing to reset.")
			}

			var scenarioSnaps []SnapshotMetadata
			for _, s := range snapshots {
				if strings.HasPrefix(s.Name, "pre-scenario-") {
					scenarioSnaps = append(scenarioSnaps, s)
				}
			}

			if len(scenarioSnaps) == 0 {
				return fmt.Errorf("No pre-scenario snapshots found. Run a scenario first.")
			}

			sort.Slice(scenarioSnaps, func(i, j int) bool {
				return scenarioSnaps[i].CreatedAt.After(scenarioSnaps[j].CreatedAt)
			})
			latest := scenarioSnaps[0]

			rpcURL, err := getChainRPC("")
			if err != nil {
				return err
			}

			out.Info("Restoring snapshot %q (block #%d)...", latest.Name, latest.BlockNumber)
			if err := scenarioRevert(rpcURL, latest.SnapshotID); err != nil {
				return err
			}

			newResult, err := rpcCall(rpcURL, "evm_snapshot")
			if err == nil {
				var newSnapID string
				json.Unmarshal(newResult, &newSnapID)
				latest.SnapshotID = newSnapID
				saveSnapshotMeta(cfg.Name, latest)
			}

			blockNum, _ := getCurrentBlock(rpcURL)
			out.Success("Reset complete! Now at block #%d", blockNum)

			for _, s := range scenarioSnaps {
				deleteSnapshotMeta(cfg.Name, s.Name)
			}
			out.Info("Cleaned up %d scenario snapshot(s)", len(scenarioSnaps))
			return nil
		},
	}
}

func scenarioRevert(rpcURL, snapID string) error {
	result, err := rpcCall(rpcURL, "evm_revert", snapID)
	if err != nil {
		return fmt.Errorf("failed to revert snapshot: %w", err)
	}
	var success bool
	json.Unmarshal(result, &success)
	if !success {
		return fmt.Errorf("snapshot revert returned false — snapshot may have already been consumed")
	}
	return nil
}
