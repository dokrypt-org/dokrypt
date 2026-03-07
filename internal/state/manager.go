package state

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"os"
	"path/filepath"

	"github.com/dokrypt/dokrypt/internal/chain"
	"github.com/dokrypt/dokrypt/internal/container"
)

type DefaultManager struct {
	store    *Store
	chainMgr *ChainStateManager
	volMgr   *VolumeStateManager
	chains   func() []chain.Chain // Accessor for active chains
	project  string
}

func NewDefaultManager(store *Store, runtime container.Runtime, chains func() []chain.Chain, project string) *DefaultManager {
	return &DefaultManager{
		store:    store,
		chainMgr: NewChainStateManager(store),
		volMgr:   NewVolumeStateManager(store, runtime),
		chains:   chains,
		project:  project,
	}
}

func (m *DefaultManager) Save(ctx context.Context, name string, opts SaveOptions) (*Snapshot, error) {
	if m.store.Exists(name) {
		return nil, fmt.Errorf("snapshot %q already exists", name)
	}

	slog.Info("saving snapshot", "name", name)

	if err := m.store.EnsureDirs(name); err != nil {
		return nil, err
	}

	snap := NewSnapshot(name, m.project, opts)

	for _, c := range m.chains() {
		slog.Info("exporting chain state", "chain", c.Name())
		cs, err := m.chainMgr.ExportChainState(ctx, name, c)
		if err != nil {
			slog.Warn("failed to export chain state", "chain", c.Name(), "error", err)
			continue
		}
		snap.Chains[c.Name()] = *cs
	}

	if !opts.SkipVolumes {
		slog.Debug("volume snapshots not yet implemented, skipping volume backup")
	}

	if err := m.store.SaveMetadata(snap); err != nil {
		return nil, err
	}

	slog.Info("snapshot saved", "name", name, "chains", len(snap.Chains))
	return snap, nil
}

func (m *DefaultManager) Restore(ctx context.Context, name string, opts RestoreOptions) error {
	snap, err := m.store.LoadMetadata(name)
	if err != nil {
		return err
	}

	slog.Info("restoring snapshot", "name", name)

	for _, c := range m.chains() {
		cs, ok := snap.Chains[c.Name()]
		if !ok {
			slog.Warn("no state for chain in snapshot", "chain", c.Name())
			continue
		}
		slog.Info("importing chain state", "chain", c.Name())
		if err := m.chainMgr.ImportChainState(ctx, name, c, cs); err != nil {
			return fmt.Errorf("failed to restore chain %s: %w", c.Name(), err)
		}
	}

	if !opts.SkipVolumes && len(snap.Volumes) > 0 {
		for _, vs := range snap.Volumes {
			slog.Info("restoring volume", "volume", vs.Name, "service", vs.Service)
			if err := m.volMgr.RestoreVolume(ctx, name, vs); err != nil {
				return fmt.Errorf("failed to restore volume %s: %w", vs.Name, err)
			}
		}
	}

	slog.Info("snapshot restored", "name", name)
	return nil
}

func (m *DefaultManager) List(ctx context.Context) ([]*Snapshot, error) {
	return m.store.ListSnapshots()
}

func (m *DefaultManager) Get(ctx context.Context, name string) (*Snapshot, error) {
	return m.store.LoadMetadata(name)
}

func (m *DefaultManager) Delete(ctx context.Context, name string) error {
	slog.Info("deleting snapshot", "name", name)
	return m.store.Delete(name)
}

func (m *DefaultManager) Export(ctx context.Context, name string, outputPath string) error {
	if !m.store.Exists(name) {
		return fmt.Errorf("snapshot %q not found", name)
	}

	slog.Info("exporting snapshot", "name", name, "output", outputPath)
	sourceDir := m.store.SnapshotDir(name)
	return createTarGz(outputPath, sourceDir)
}

func (m *DefaultManager) Import(ctx context.Context, inputPath string) (*Snapshot, error) {
	slog.Info("importing snapshot", "input", inputPath)

	tempDir, err := os.MkdirTemp("", "dokrypt-import-*")
	if err != nil {
		return nil, fmt.Errorf("failed to create temp dir: %w", err)
	}
	defer os.RemoveAll(tempDir)

	if err := extractTarGz(inputPath, tempDir); err != nil {
		return nil, fmt.Errorf("failed to extract snapshot: %w", err)
	}

	tempStore := NewStore(tempDir)
	entries, err := os.ReadDir(tempDir)
	if err != nil {
		return nil, err
	}

	flatMeta := filepath.Join(tempDir, "metadata.json")
	if _, statErr := os.Stat(flatMeta); statErr == nil {
		data, readErr := os.ReadFile(flatMeta)
		if readErr != nil {
			return nil, fmt.Errorf("failed to read metadata: %w", readErr)
		}
		var snap Snapshot
		if jsonErr := json.Unmarshal(data, &snap); jsonErr != nil {
			return nil, fmt.Errorf("failed to parse metadata: %w", jsonErr)
		}
		destDir := m.store.SnapshotDir(snap.Name)
		if mkErr := os.MkdirAll(destDir, 0o755); mkErr != nil {
			return nil, mkErr
		}
		return &snap, copyDir(tempDir, destDir)
	}

	var snapshotName string
	for _, entry := range entries {
		if entry.IsDir() {
			if tempStore.Exists(entry.Name()) {
				snapshotName = entry.Name()
				break
			}
		}
	}

	if snapshotName == "" {
		return nil, fmt.Errorf("no valid snapshot found in archive")
	}

	snap, err := tempStore.LoadMetadata(snapshotName)
	if err != nil {
		return nil, err
	}

	destDir := m.store.SnapshotDir(snap.Name)
	if err := os.MkdirAll(destDir, 0o755); err != nil {
		return nil, err
	}

	srcDir := tempStore.SnapshotDir(snapshotName)
	return snap, copyDir(srcDir, destDir)
}

func (m *DefaultManager) Diff(ctx context.Context, name1, name2 string) (*DiffResult, error) {
	snap1, err := m.store.LoadMetadata(name1)
	if err != nil {
		return nil, fmt.Errorf("snapshot %q: %w", name1, err)
	}

	snap2, err := m.store.LoadMetadata(name2)
	if err != nil {
		return nil, fmt.Errorf("snapshot %q: %w", name2, err)
	}

	result := &DiffResult{
		Snapshot1: name1,
		Snapshot2: name2,
	}

	allChains := make(map[string]bool)
	for name := range snap1.Chains {
		allChains[name] = true
	}
	for name := range snap2.Chains {
		allChains[name] = true
	}

	for chainName := range allChains {
		cs1, ok1 := snap1.Chains[chainName]
		cs2, ok2 := snap2.Chains[chainName]

		if ok1 && ok2 {
			result.ChainDiffs = append(result.ChainDiffs, ChainDiff{
				Chain:      chainName,
				Block1:     cs1.BlockNumber,
				Block2:     cs2.BlockNumber,
				BlockDelta: int64(cs2.BlockNumber) - int64(cs1.BlockNumber),
			})
		}
	}

	vols1 := make(map[string]VolumeSnapshot)
	for _, v := range snap1.Volumes {
		vols1[v.Name] = v
	}
	for _, v2 := range snap2.Volumes {
		v1, ok := vols1[v2.Name]
		if ok {
			result.VolumeDiffs = append(result.VolumeDiffs, VolumeDiff{
				Volume: v2.Name,
				Size1:  v1.Size,
				Size2:  v2.Size,
			})
		}
	}

	return result, nil
}

func copyDir(src, dst string) error {
	return filepath.Walk(src, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		rel, err := filepath.Rel(src, path)
		if err != nil {
			return err
		}
		target := filepath.Join(dst, rel)

		if info.IsDir() {
			return os.MkdirAll(target, info.Mode())
		}

		data, err := os.ReadFile(path)
		if err != nil {
			return err
		}
		return os.WriteFile(target, data, info.Mode())
	})
}
