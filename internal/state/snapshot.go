package state

import (
	"context"
	"time"
)

type Snapshot struct {
	Name        string            `json:"name"`
	Description string            `json:"description,omitempty"`
	Tags        []string          `json:"tags,omitempty"`
	CreatedAt   time.Time         `json:"created_at"`
	Project     string            `json:"project"`
	Chains      map[string]ChainSnapshot `json:"chains"`
	Volumes     []VolumeSnapshot  `json:"volumes,omitempty"`
	ConfigHash  string            `json:"config_hash"`
}

type ChainSnapshot struct {
	Name        string `json:"name"`
	Engine      string `json:"engine"`
	ChainID     uint64 `json:"chain_id"`
	BlockNumber uint64 `json:"block_number"`
	StateFile   string `json:"state_file"` // Relative path to state dump
}

type VolumeSnapshot struct {
	Name     string `json:"name"`
	Service  string `json:"service"`
	ArchiveFile string `json:"archive_file"` // Relative path to tar.gz
	Size     int64  `json:"size"`
}

type SaveOptions struct {
	Description string
	Tags        []string
	Hot         bool // Don't stop services (use EVM snapshot)
	SkipVolumes bool // Skip volume backup
}

type RestoreOptions struct {
	SkipVolumes bool
}

type DiffResult struct {
	Snapshot1    string           `json:"snapshot1"`
	Snapshot2    string           `json:"snapshot2"`
	ChainDiffs   []ChainDiff     `json:"chain_diffs,omitempty"`
	VolumeDiffs  []VolumeDiff    `json:"volume_diffs,omitempty"`
}

type ChainDiff struct {
	Chain        string `json:"chain"`
	BlockDelta   int64  `json:"block_delta"`
	Block1       uint64 `json:"block1"`
	Block2       uint64 `json:"block2"`
}

type VolumeDiff struct {
	Volume string `json:"volume"`
	Size1  int64  `json:"size1"`
	Size2  int64  `json:"size2"`
}

type Manager interface {
	Save(ctx context.Context, name string, opts SaveOptions) (*Snapshot, error)
	Restore(ctx context.Context, name string, opts RestoreOptions) error
	List(ctx context.Context) ([]*Snapshot, error)
	Get(ctx context.Context, name string) (*Snapshot, error)
	Delete(ctx context.Context, name string) error
	Export(ctx context.Context, name string, outputPath string) error
	Import(ctx context.Context, inputPath string) (*Snapshot, error)
	Diff(ctx context.Context, name1, name2 string) (*DiffResult, error)
}
