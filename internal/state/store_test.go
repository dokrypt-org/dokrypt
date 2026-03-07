package state

import (
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewStore_CreatesWithCorrectPath(t *testing.T) {
	dir := t.TempDir()
	store := NewStore(dir)

	require.NotNil(t, store)
	assert.Equal(t, dir, store.Dir())
}

func TestStore_SnapshotDir_ReturnsCorrectPath(t *testing.T) {
	dir := t.TempDir()
	store := NewStore(dir)

	got := store.SnapshotDir("my-snap")
	expected := filepath.Join(dir, "my-snap")
	assert.Equal(t, expected, got)
}

func TestStore_ChainsDir_ReturnsCorrectPath(t *testing.T) {
	dir := t.TempDir()
	store := NewStore(dir)

	got := store.ChainsDir("my-snap")
	expected := filepath.Join(dir, "my-snap", "chains")
	assert.Equal(t, expected, got)
}

func TestStore_VolumesDir_ReturnsCorrectPath(t *testing.T) {
	dir := t.TempDir()
	store := NewStore(dir)

	got := store.VolumesDir("my-snap")
	expected := filepath.Join(dir, "my-snap", "volumes")
	assert.Equal(t, expected, got)
}

func TestStore_EnsureDirs_CreatesAllDirectories(t *testing.T) {
	dir := t.TempDir()
	store := NewStore(dir)

	err := store.EnsureDirs("test-snap")
	require.NoError(t, err)

	assertDirExists(t, store.SnapshotDir("test-snap"))
	assertDirExists(t, store.ChainsDir("test-snap"))
	assertDirExists(t, store.VolumesDir("test-snap"))
}

func TestStore_EnsureDirs_IsIdempotent(t *testing.T) {
	dir := t.TempDir()
	store := NewStore(dir)

	err := store.EnsureDirs("test-snap")
	require.NoError(t, err)

	err = store.EnsureDirs("test-snap")
	require.NoError(t, err)
}

func TestStore_SaveMetadata_AndLoadMetadata_Roundtrip(t *testing.T) {
	dir := t.TempDir()
	store := NewStore(dir)

	snap := &Snapshot{
		Name:        "roundtrip-snap",
		Description: "a test snapshot",
		Tags:        []string{"alpha", "beta"},
		CreatedAt:   time.Date(2025, 6, 15, 12, 0, 0, 0, time.UTC),
		Project:     "my-project",
		Chains:      map[string]ChainSnapshot{},
		ConfigHash:  "deadbeef",
	}

	require.NoError(t, store.EnsureDirs(snap.Name))
	require.NoError(t, store.SaveMetadata(snap))

	loaded, err := store.LoadMetadata(snap.Name)
	require.NoError(t, err)
	require.NotNil(t, loaded)

	assert.Equal(t, snap.Name, loaded.Name)
	assert.Equal(t, snap.Description, loaded.Description)
	assert.Equal(t, snap.Tags, loaded.Tags)
	assert.Equal(t, snap.CreatedAt.UTC(), loaded.CreatedAt.UTC())
	assert.Equal(t, snap.Project, loaded.Project)
	assert.Equal(t, snap.ConfigHash, loaded.ConfigHash)
}

func TestStore_SaveMetadata_WithChainData(t *testing.T) {
	dir := t.TempDir()
	store := NewStore(dir)

	snap := &Snapshot{
		Name:      "chain-snap",
		CreatedAt: time.Now().UTC(),
		Project:   "proj",
		Chains: map[string]ChainSnapshot{
			"mainnet": {
				Name:        "mainnet",
				Engine:      "anvil",
				ChainID:     1,
				BlockNumber: 19_000_000,
				StateFile:   "chains/mainnet/state.json",
			},
		},
	}

	require.NoError(t, store.EnsureDirs(snap.Name))
	require.NoError(t, store.SaveMetadata(snap))

	loaded, err := store.LoadMetadata(snap.Name)
	require.NoError(t, err)

	cs, ok := loaded.Chains["mainnet"]
	require.True(t, ok)
	assert.Equal(t, "mainnet", cs.Name)
	assert.Equal(t, "anvil", cs.Engine)
	assert.Equal(t, uint64(1), cs.ChainID)
	assert.Equal(t, uint64(19_000_000), cs.BlockNumber)
}

func TestStore_LoadMetadata_ErrorWhenMissing(t *testing.T) {
	dir := t.TempDir()
	store := NewStore(dir)

	_, err := store.LoadMetadata("nonexistent")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "nonexistent")
}

func TestStore_ListSnapshots_ReturnsSortedByTimeNewestFirst(t *testing.T) {
	dir := t.TempDir()
	store := NewStore(dir)

	snaps := []*Snapshot{
		{
			Name:      "oldest",
			Project:   "proj",
			CreatedAt: time.Date(2025, 1, 1, 0, 0, 0, 0, time.UTC),
			Chains:    map[string]ChainSnapshot{},
		},
		{
			Name:      "newest",
			Project:   "proj",
			CreatedAt: time.Date(2025, 3, 1, 0, 0, 0, 0, time.UTC),
			Chains:    map[string]ChainSnapshot{},
		},
		{
			Name:      "middle",
			Project:   "proj",
			CreatedAt: time.Date(2025, 2, 1, 0, 0, 0, 0, time.UTC),
			Chains:    map[string]ChainSnapshot{},
		},
	}

	for _, s := range snaps {
		require.NoError(t, store.EnsureDirs(s.Name))
		require.NoError(t, store.SaveMetadata(s))
	}

	list, err := store.ListSnapshots()
	require.NoError(t, err)
	require.Len(t, list, 3)

	assert.Equal(t, "newest", list[0].Name)
	assert.Equal(t, "middle", list[1].Name)
	assert.Equal(t, "oldest", list[2].Name)
}

func TestStore_ListSnapshots_EmptyDirReturnsNil(t *testing.T) {
	dir := t.TempDir()
	store := NewStore(dir)

	list, err := store.ListSnapshots()
	require.NoError(t, err)
	assert.Nil(t, list)
}

func TestStore_ListSnapshots_NonexistentDirReturnsNil(t *testing.T) {
	dir := t.TempDir()
	store := NewStore(filepath.Join(dir, "does-not-exist"))

	list, err := store.ListSnapshots()
	require.NoError(t, err)
	assert.Nil(t, list)
}

func TestStore_ListSnapshots_SkipsCorruptedEntries(t *testing.T) {
	dir := t.TempDir()
	store := NewStore(dir)

	snap := &Snapshot{
		Name:      "valid",
		Project:   "proj",
		CreatedAt: time.Now().UTC(),
		Chains:    map[string]ChainSnapshot{},
	}
	require.NoError(t, store.EnsureDirs(snap.Name))
	require.NoError(t, store.SaveMetadata(snap))

	require.NoError(t, store.EnsureDirs("corrupted"))

	list, err := store.ListSnapshots()
	require.NoError(t, err)
	require.Len(t, list, 1)
	assert.Equal(t, "valid", list[0].Name)
}

func TestStore_Exists_TrueForExistingSnapshot(t *testing.T) {
	dir := t.TempDir()
	store := NewStore(dir)

	snap := &Snapshot{
		Name:      "exists-snap",
		Project:   "proj",
		CreatedAt: time.Now().UTC(),
		Chains:    map[string]ChainSnapshot{},
	}
	require.NoError(t, store.EnsureDirs(snap.Name))
	require.NoError(t, store.SaveMetadata(snap))

	assert.True(t, store.Exists("exists-snap"))
}

func TestStore_Exists_FalseForMissingSnapshot(t *testing.T) {
	dir := t.TempDir()
	store := NewStore(dir)

	assert.False(t, store.Exists("no-such-snap"))
}

func TestStore_Exists_FalseWhenDirExistsButNoMetadata(t *testing.T) {
	dir := t.TempDir()
	store := NewStore(dir)

	require.NoError(t, store.EnsureDirs("dir-only"))

	assert.False(t, store.Exists("dir-only"))
}

func TestStore_Delete_RemovesSnapshotDirectory(t *testing.T) {
	dir := t.TempDir()
	store := NewStore(dir)

	snap := &Snapshot{
		Name:      "to-delete",
		Project:   "proj",
		CreatedAt: time.Now().UTC(),
		Chains:    map[string]ChainSnapshot{},
	}
	require.NoError(t, store.EnsureDirs(snap.Name))
	require.NoError(t, store.SaveMetadata(snap))
	require.True(t, store.Exists(snap.Name))

	err := store.Delete(snap.Name)
	require.NoError(t, err)

	assert.False(t, store.Exists(snap.Name))
}

func TestStore_Delete_ErrorForNonexistentSnapshot(t *testing.T) {
	dir := t.TempDir()
	store := NewStore(dir)

	err := store.Delete("not-here")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "not-here")
}

func TestStore_SaveConfig_AndLoadConfig_Roundtrip(t *testing.T) {
	dir := t.TempDir()
	store := NewStore(dir)

	require.NoError(t, store.EnsureDirs("cfg-snap"))

	configData := []byte("version: 1\nproject: test\n")
	require.NoError(t, store.SaveConfig("cfg-snap", configData))

	loaded, err := store.LoadConfig("cfg-snap")
	require.NoError(t, err)
	assert.Equal(t, configData, loaded)
}

func TestStore_LoadConfig_ErrorWhenMissing(t *testing.T) {
	dir := t.TempDir()
	store := NewStore(dir)

	require.NoError(t, store.EnsureDirs("snap"))

	_, err := store.LoadConfig("snap")
	require.Error(t, err)
}

func assertDirExists(t *testing.T, path string) {
	t.Helper()
	info, err := os.Stat(path)
	require.NoError(t, err, "directory should exist: %s", path)
	assert.True(t, info.IsDir(), "path should be a directory: %s", path)
}
