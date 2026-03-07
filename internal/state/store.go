package state

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
)

type Store struct {
	baseDir string // ~/.dokrypt/snapshots
}

func NewStore(baseDir string) *Store {
	return &Store{baseDir: baseDir}
}

func DefaultStore() (*Store, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return nil, fmt.Errorf("cannot determine home directory: %w", err)
	}
	dir := filepath.Join(home, ".dokrypt", "snapshots")
	return NewStore(dir), nil
}

func (s *Store) Dir() string {
	return s.baseDir
}

func (s *Store) SnapshotDir(name string) string {
	return filepath.Join(s.baseDir, name)
}

func (s *Store) ChainsDir(name string) string {
	return filepath.Join(s.SnapshotDir(name), "chains")
}

func (s *Store) VolumesDir(name string) string {
	return filepath.Join(s.SnapshotDir(name), "volumes")
}

func (s *Store) EnsureDirs(name string) error {
	dirs := []string{
		s.SnapshotDir(name),
		s.ChainsDir(name),
		s.VolumesDir(name),
	}
	for _, d := range dirs {
		if err := os.MkdirAll(d, 0o755); err != nil {
			return fmt.Errorf("failed to create directory %s: %w", d, err)
		}
	}
	return nil
}

func (s *Store) SaveMetadata(snap *Snapshot) error {
	path := filepath.Join(s.SnapshotDir(snap.Name), "metadata.json")
	data, err := json.MarshalIndent(snap, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal metadata: %w", err)
	}
	return os.WriteFile(path, data, 0o644)
}

func (s *Store) LoadMetadata(name string) (*Snapshot, error) {
	path := filepath.Join(s.SnapshotDir(name), "metadata.json")
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("failed to read metadata for snapshot %q: %w", name, err)
	}
	var snap Snapshot
	if err := json.Unmarshal(data, &snap); err != nil {
		return nil, fmt.Errorf("failed to parse metadata for snapshot %q: %w", name, err)
	}
	return &snap, nil
}

func (s *Store) ListSnapshots() ([]*Snapshot, error) {
	entries, err := os.ReadDir(s.baseDir)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, fmt.Errorf("failed to list snapshots: %w", err)
	}

	var snapshots []*Snapshot
	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}
		snap, err := s.LoadMetadata(entry.Name())
		if err != nil {
			continue // Skip corrupted snapshots
		}
		snapshots = append(snapshots, snap)
	}

	sort.Slice(snapshots, func(i, j int) bool {
		return snapshots[i].CreatedAt.After(snapshots[j].CreatedAt)
	})

	return snapshots, nil
}

func (s *Store) Exists(name string) bool {
	_, err := os.Stat(filepath.Join(s.SnapshotDir(name), "metadata.json"))
	return err == nil
}

func (s *Store) Delete(name string) error {
	dir := s.SnapshotDir(name)
	if _, err := os.Stat(dir); os.IsNotExist(err) {
		return fmt.Errorf("snapshot %q not found", name)
	}
	return os.RemoveAll(dir)
}

func (s *Store) SaveConfig(name string, configData []byte) error {
	path := filepath.Join(s.SnapshotDir(name), "config.yaml")
	return os.WriteFile(path, configData, 0o644)
}

func (s *Store) LoadConfig(name string) ([]byte, error) {
	path := filepath.Join(s.SnapshotDir(name), "config.yaml")
	return os.ReadFile(path)
}
