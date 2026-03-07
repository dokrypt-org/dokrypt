package cli

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"time"

	"github.com/spf13/cobra"

	"github.com/dokrypt/dokrypt/internal/config"
)

type SnapshotMetadata struct {
	Name        string    `json:"name"`
	Project     string    `json:"project"`
	Description string    `json:"description,omitempty"`
	Tags        []string  `json:"tags,omitempty"`
	CreatedAt   time.Time `json:"created_at"`
	ChainID     uint64    `json:"chain_id,omitempty"`
	BlockNumber uint64    `json:"block_number,omitempty"`
	SnapshotID  string    `json:"snapshot_id"` // EVM snapshot ID from anvil
}

func newSnapshotCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "snapshot",
		Short: "State snapshot management",
	}

	cmd.AddCommand(
		newSnapshotSaveCmd(),
		newSnapshotRestoreCmd(),
		newSnapshotListCmd(),
		newSnapshotDeleteCmd(),
		newSnapshotExportCmd(),
		newSnapshotImportCmd(),
		newSnapshotDiffCmd(),
	)

	return cmd
}

func newSnapshotSaveCmd() *cobra.Command {
	var (
		description string
		tags        []string
	)

	cmd := &cobra.Command{
		Use:   "save <name>",
		Short: "Save current state as snapshot",
		Args:  requireArgs(1, "dokrypt snapshot save <name>"),
		RunE: func(cmd *cobra.Command, args []string) error {
			out := getOutput()
			name := args[0]

			rpcURL, err := getChainRPC("")
			if err != nil {
				return err
			}

			out.Info("Taking snapshot %q...", name)
			result, err := rpcCall(rpcURL, "evm_snapshot")
			if err != nil {
				return fmt.Errorf("failed to take snapshot: %w", err)
			}

			var snapID string
			json.Unmarshal(result, &snapID)

			blockNum, _ := getCurrentBlock(rpcURL)
			chainID, _ := getChainID(rpcURL)

			cfg, err := config.Parse(getConfigPath())
			if err != nil {
				return fmt.Errorf("No dokrypt.yaml found.")
			}

			meta := SnapshotMetadata{
				Name:        name,
				Project:     cfg.Name,
				Description: description,
				Tags:        tags,
				CreatedAt:   time.Now(),
				ChainID:     chainID,
				BlockNumber: blockNum,
				SnapshotID:  snapID,
			}

			if err := saveSnapshotMeta(cfg.Name, meta); err != nil {
				return fmt.Errorf("failed to save snapshot metadata: %w", err)
			}

			out.Success("Snapshot %q saved (block #%d, id: %s)", name, blockNum, snapID)
			return nil
		},
	}

	cmd.Flags().StringVar(&description, "description", "", "snapshot description")
	cmd.Flags().StringSliceVar(&tags, "tags", nil, "snapshot tags")
	return cmd
}

func newSnapshotRestoreCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "restore <name>",
		Short: "Restore to a saved snapshot",
		Args:  requireArgs(1, "dokrypt snapshot restore <name>"),
		RunE: func(cmd *cobra.Command, args []string) error {
			out := getOutput()
			name := args[0]

			cfg, err := config.Parse(getConfigPath())
			if err != nil {
				return fmt.Errorf("No dokrypt.yaml found.")
			}

			meta, err := loadSnapshotMeta(cfg.Name, name)
			if err != nil {
				return fmt.Errorf("Snapshot %q not found. Run 'dokrypt snapshot list' to see available snapshots.", name)
			}

			rpcURL, err := getChainRPC("")
			if err != nil {
				return err
			}

			out.Info("Restoring snapshot %q...", name)
			result, err := rpcCall(rpcURL, "evm_revert", meta.SnapshotID)
			if err != nil {
				return fmt.Errorf("failed to restore snapshot: %w", err)
			}

			var success bool
			json.Unmarshal(result, &success)
			if !success {
				return fmt.Errorf("snapshot restore returned false — snapshot may have already been consumed")
			}

			newResult, err := rpcCall(rpcURL, "evm_snapshot")
			if err == nil {
				var newSnapID string
				json.Unmarshal(newResult, &newSnapID)
				meta.SnapshotID = newSnapID
				saveSnapshotMeta(cfg.Name, *meta)
			}

			blockNum, _ := getCurrentBlock(rpcURL)
			out.Success("Restored to snapshot %q (now at block #%d)", name, blockNum)
			return nil
		},
	}
}

func newSnapshotListCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "list",
		Short: "List all snapshots",
		RunE: func(cmd *cobra.Command, args []string) error {
			out := getOutput()

			cfg, err := config.Parse(getConfigPath())
			if err != nil {
				return fmt.Errorf("No dokrypt.yaml found.")
			}

			snapshots, err := listSnapshotMetas(cfg.Name)
			if err != nil || len(snapshots) == 0 {
				out.Info("No snapshots found. Use 'dokrypt snapshot save <name>' to create one.")
				return nil
			}

			fmt.Println()
			headers := []string{"Name", "Block", "Created", "Description"}
			var rows [][]string
			for _, s := range snapshots {
				rows = append(rows, []string{
					s.Name,
					fmt.Sprintf("#%d", s.BlockNumber),
					s.CreatedAt.Format("2006-01-02 15:04:05"),
					s.Description,
				})
			}
			out.Table(headers, rows)
			fmt.Println()
			return nil
		},
	}
}

func newSnapshotDeleteCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "delete <name>",
		Short: "Delete a snapshot",
		Args:  requireArgs(1, "dokrypt snapshot delete <name>"),
		RunE: func(cmd *cobra.Command, args []string) error {
			out := getOutput()

			cfg, err := config.Parse(getConfigPath())
			if err != nil {
				return fmt.Errorf("No dokrypt.yaml found.")
			}

			if err := deleteSnapshotMeta(cfg.Name, args[0]); err != nil {
				return fmt.Errorf("Snapshot %q not found.", args[0])
			}

			out.Success("Snapshot %q deleted", args[0])
			return nil
		},
	}
}

func newSnapshotExportCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "export <name> <path>",
		Short: "Export snapshot to file",
		Args:  requireArgs(2, "dokrypt snapshot export <name> <path>"),
		RunE: func(cmd *cobra.Command, args []string) error {
			out := getOutput()
			name := args[0]
			outPath := args[1]

			cfg, err := config.Parse(getConfigPath())
			if err != nil {
				return fmt.Errorf("No dokrypt.yaml found.")
			}

			meta, err := loadSnapshotMeta(cfg.Name, name)
			if err != nil {
				return fmt.Errorf("Snapshot %q not found.", name)
			}

			rpcURL, err := getChainRPC("")
			if err != nil {
				return err
			}

			out.Info("Exporting chain state...")
			result, err := rpcCall(rpcURL, "anvil_dumpState")
			if err != nil {
				return fmt.Errorf("failed to dump state: %w", err)
			}

			exportData := map[string]any{
				"metadata": meta,
				"state":    result,
			}
			data, _ := json.MarshalIndent(exportData, "", "  ")
			if err := os.WriteFile(outPath, data, 0644); err != nil {
				return fmt.Errorf("failed to write export file: %w", err)
			}

			out.Success("Snapshot %q exported to %s", name, outPath)
			return nil
		},
	}
}

func newSnapshotImportCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "import <path>",
		Short: "Import snapshot from file",
		Args:  requireArgs(1, "dokrypt snapshot import <path>"),
		RunE: func(cmd *cobra.Command, args []string) error {
			out := getOutput()
			importPath := args[0]

			data, err := os.ReadFile(importPath)
			if err != nil {
				return fmt.Errorf("failed to read import file: %w", err)
			}

			var exportData struct {
				Metadata SnapshotMetadata `json:"metadata"`
				State    json.RawMessage  `json:"state"`
			}
			if err := json.Unmarshal(data, &exportData); err != nil {
				return fmt.Errorf("invalid snapshot file: %w", err)
			}

			rpcURL, err := getChainRPC("")
			if err != nil {
				return err
			}

			out.Info("Importing chain state...")
			var stateStr string
			json.Unmarshal(exportData.State, &stateStr)
			if _, err := rpcCall(rpcURL, "anvil_loadState", stateStr); err != nil {
				return fmt.Errorf("failed to load state: %w", err)
			}

			cfg, err := config.Parse(getConfigPath())
			if err == nil {
				exportData.Metadata.Project = cfg.Name
				saveSnapshotMeta(cfg.Name, exportData.Metadata)
			}

			blockNum, _ := getCurrentBlock(rpcURL)
			out.Success("Snapshot %q imported (now at block #%d)", exportData.Metadata.Name, blockNum)
			return nil
		},
	}
}

func newSnapshotDiffCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "diff <snap1> <snap2>",
		Short: "Show diff between snapshots",
		Args:  requireArgs(2, "dokrypt snapshot diff <snap1> <snap2>"),
		RunE: func(cmd *cobra.Command, args []string) error {
			out := getOutput()

			cfg, err := config.Parse(getConfigPath())
			if err != nil {
				return fmt.Errorf("No dokrypt.yaml found.")
			}

			s1, err := loadSnapshotMeta(cfg.Name, args[0])
			if err != nil {
				return fmt.Errorf("Snapshot %q not found.", args[0])
			}
			s2, err := loadSnapshotMeta(cfg.Name, args[1])
			if err != nil {
				return fmt.Errorf("Snapshot %q not found.", args[1])
			}

			fmt.Println()
			out.Info("Snapshot: %s vs %s", s1.Name, s2.Name)
			out.Info("  Block:   #%d → #%d (%+d blocks)", s1.BlockNumber, s2.BlockNumber, int64(s2.BlockNumber)-int64(s1.BlockNumber))
			out.Info("  Created: %s → %s", s1.CreatedAt.Format("15:04:05"), s2.CreatedAt.Format("15:04:05"))
			fmt.Println()
			return nil
		},
	}
}

func snapshotsDir(projectName string) string {
	home, _ := os.UserHomeDir()
	return filepath.Join(home, ".dokrypt", "snapshots", projectName)
}

func saveSnapshotMeta(projectName string, meta SnapshotMetadata) error {
	dir := snapshotsDir(projectName)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return err
	}
	data, err := json.MarshalIndent(meta, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(filepath.Join(dir, meta.Name+".json"), data, 0644)
}

func loadSnapshotMeta(projectName, name string) (*SnapshotMetadata, error) {
	path := filepath.Join(snapshotsDir(projectName), name+".json")
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	var meta SnapshotMetadata
	if err := json.Unmarshal(data, &meta); err != nil {
		return nil, err
	}
	return &meta, nil
}

func listSnapshotMetas(projectName string) ([]SnapshotMetadata, error) {
	dir := snapshotsDir(projectName)
	entries, err := os.ReadDir(dir)
	if err != nil {
		return nil, err
	}

	var snapshots []SnapshotMetadata
	for _, e := range entries {
		if filepath.Ext(e.Name()) != ".json" {
			continue
		}
		data, err := os.ReadFile(filepath.Join(dir, e.Name()))
		if err != nil {
			continue
		}
		var meta SnapshotMetadata
		if json.Unmarshal(data, &meta) == nil {
			snapshots = append(snapshots, meta)
		}
	}

	sort.Slice(snapshots, func(i, j int) bool {
		return snapshots[i].CreatedAt.Before(snapshots[j].CreatedAt)
	})
	return snapshots, nil
}

func deleteSnapshotMeta(projectName, name string) error {
	path := filepath.Join(snapshotsDir(projectName), name+".json")
	return os.Remove(path)
}
