package state

import (
	"encoding/json"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestSnapshot_JSONRoundtrip(t *testing.T) {
	original := &Snapshot{
		Name:        "json-snap",
		Description: "a snapshot for json tests",
		Tags:        []string{"prod", "v1.2"},
		CreatedAt:   time.Date(2025, 7, 4, 12, 0, 0, 0, time.UTC),
		Project:     "my-proj",
		Chains: map[string]ChainSnapshot{
			"mainnet": {
				Name:        "mainnet",
				Engine:      "anvil",
				ChainID:     1,
				BlockNumber: 19_500_000,
				StateFile:   "chains/mainnet/state.json",
			},
		},
		Volumes: []VolumeSnapshot{
			{
				Name:        "postgres-data",
				Service:     "postgres",
				ArchiveFile: "volumes/postgres.tar.gz",
				Size:        1024,
			},
		},
		ConfigHash: "abcdef0123456789",
	}

	data, err := json.Marshal(original)
	require.NoError(t, err)

	var restored Snapshot
	err = json.Unmarshal(data, &restored)
	require.NoError(t, err)

	assert.Equal(t, original.Name, restored.Name)
	assert.Equal(t, original.Description, restored.Description)
	assert.Equal(t, original.Tags, restored.Tags)
	assert.Equal(t, original.CreatedAt.UTC(), restored.CreatedAt.UTC())
	assert.Equal(t, original.Project, restored.Project)
	assert.Equal(t, original.ConfigHash, restored.ConfigHash)

	cs, ok := restored.Chains["mainnet"]
	require.True(t, ok)
	assert.Equal(t, "mainnet", cs.Name)
	assert.Equal(t, "anvil", cs.Engine)
	assert.Equal(t, uint64(1), cs.ChainID)
	assert.Equal(t, uint64(19_500_000), cs.BlockNumber)
	assert.Equal(t, "chains/mainnet/state.json", cs.StateFile)

	require.Len(t, restored.Volumes, 1)
	assert.Equal(t, "postgres-data", restored.Volumes[0].Name)
	assert.Equal(t, "postgres", restored.Volumes[0].Service)
	assert.Equal(t, int64(1024), restored.Volumes[0].Size)
}

func TestSnapshot_JSONOmitsEmptyOptionalFields(t *testing.T) {
	snap := &Snapshot{
		Name:      "minimal",
		CreatedAt: time.Date(2025, 1, 1, 0, 0, 0, 0, time.UTC),
		Project:   "proj",
		Chains:    map[string]ChainSnapshot{},
	}

	data, err := json.Marshal(snap)
	require.NoError(t, err)

	jsonStr := string(data)
	assert.NotContains(t, jsonStr, `"description"`)
	assert.NotContains(t, jsonStr, `"tags"`)
	assert.NotContains(t, jsonStr, `"volumes"`)
}

func TestSnapshot_JSONIncludesDescription(t *testing.T) {
	snap := &Snapshot{
		Name:        "with-desc",
		Description: "has a description",
		CreatedAt:   time.Now().UTC(),
		Project:     "proj",
		Chains:      map[string]ChainSnapshot{},
	}

	data, err := json.Marshal(snap)
	require.NoError(t, err)
	assert.Contains(t, string(data), `"description"`)
}

func TestChainSnapshot_JSONRoundtrip(t *testing.T) {
	original := ChainSnapshot{
		Name:        "sepolia",
		Engine:      "geth",
		ChainID:     11155111,
		BlockNumber: 5_000_000,
		StateFile:   "chains/sepolia/state.json",
	}

	data, err := json.Marshal(original)
	require.NoError(t, err)

	var restored ChainSnapshot
	err = json.Unmarshal(data, &restored)
	require.NoError(t, err)

	assert.Equal(t, original, restored)
}

func TestVolumeSnapshot_JSONRoundtrip(t *testing.T) {
	original := VolumeSnapshot{
		Name:        "redis-data",
		Service:     "redis",
		ArchiveFile: "volumes/redis.tar.gz",
		Size:        65536,
	}

	data, err := json.Marshal(original)
	require.NoError(t, err)

	var restored VolumeSnapshot
	err = json.Unmarshal(data, &restored)
	require.NoError(t, err)

	assert.Equal(t, original, restored)
}

func TestSaveOptions_DefaultValues(t *testing.T) {
	opts := SaveOptions{}
	assert.Empty(t, opts.Description)
	assert.Nil(t, opts.Tags)
	assert.False(t, opts.Hot)
	assert.False(t, opts.SkipVolumes)
}

func TestSaveOptions_AllFieldsSet(t *testing.T) {
	opts := SaveOptions{
		Description: "pre-deploy",
		Tags:        []string{"staging"},
		Hot:         true,
		SkipVolumes: true,
	}
	assert.Equal(t, "pre-deploy", opts.Description)
	assert.Equal(t, []string{"staging"}, opts.Tags)
	assert.True(t, opts.Hot)
	assert.True(t, opts.SkipVolumes)
}

func TestRestoreOptions_DefaultValues(t *testing.T) {
	opts := RestoreOptions{}
	assert.False(t, opts.SkipVolumes)
}

func TestRestoreOptions_SkipVolumes(t *testing.T) {
	opts := RestoreOptions{SkipVolumes: true}
	assert.True(t, opts.SkipVolumes)
}

func TestDiffResult_JSONRoundtrip(t *testing.T) {
	original := &DiffResult{
		Snapshot1: "before",
		Snapshot2: "after",
		ChainDiffs: []ChainDiff{
			{Chain: "mainnet", Block1: 100, Block2: 200, BlockDelta: 100},
		},
		VolumeDiffs: []VolumeDiff{
			{Volume: "db-data", Size1: 500, Size2: 1000},
		},
	}

	data, err := json.Marshal(original)
	require.NoError(t, err)

	var restored DiffResult
	err = json.Unmarshal(data, &restored)
	require.NoError(t, err)

	assert.Equal(t, original.Snapshot1, restored.Snapshot1)
	assert.Equal(t, original.Snapshot2, restored.Snapshot2)
	require.Len(t, restored.ChainDiffs, 1)
	assert.Equal(t, int64(100), restored.ChainDiffs[0].BlockDelta)
	require.Len(t, restored.VolumeDiffs, 1)
	assert.Equal(t, int64(1000), restored.VolumeDiffs[0].Size2)
}

func TestDiffResult_OmitsEmptyDiffs(t *testing.T) {
	result := &DiffResult{
		Snapshot1: "s1",
		Snapshot2: "s2",
	}

	data, err := json.Marshal(result)
	require.NoError(t, err)

	jsonStr := string(data)
	assert.NotContains(t, jsonStr, `"chain_diffs"`)
	assert.NotContains(t, jsonStr, `"volume_diffs"`)
}

func TestChainDiff_NegativeDelta(t *testing.T) {
	cd := ChainDiff{
		Chain:      "mainnet",
		Block1:     1000,
		Block2:     500,
		BlockDelta: -500,
	}
	assert.Equal(t, int64(-500), cd.BlockDelta)
}

func TestVolumeDiff_ZeroSizes(t *testing.T) {
	vd := VolumeDiff{
		Volume: "empty-vol",
		Size1:  0,
		Size2:  0,
	}
	assert.Equal(t, int64(0), vd.Size1)
	assert.Equal(t, int64(0), vd.Size2)
}
