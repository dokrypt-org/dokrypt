package state

import (
	"context"
	"encoding/json"
	"io"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/dokrypt/dokrypt/internal/chain"
	"github.com/dokrypt/dokrypt/internal/container"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type mockRuntime struct{}

func (r *mockRuntime) CreateContainer(context.Context, *container.ContainerConfig) (string, error) {
	return "", nil
}
func (r *mockRuntime) StartContainer(context.Context, string) error { return nil }
func (r *mockRuntime) StopContainer(context.Context, string, time.Duration) error {
	return nil
}
func (r *mockRuntime) RemoveContainer(context.Context, string, bool) error { return nil }
func (r *mockRuntime) ListContainers(context.Context, container.ListOptions) ([]container.ContainerInfo, error) {
	return nil, nil
}
func (r *mockRuntime) InspectContainer(context.Context, string) (*container.ContainerInfo, error) {
	return nil, nil
}
func (r *mockRuntime) WaitContainer(context.Context, string) (int64, error) { return 0, nil }
func (r *mockRuntime) PullImage(context.Context, string) error              { return nil }
func (r *mockRuntime) BuildImage(context.Context, string, container.BuildOptions) (string, error) {
	return "", nil
}
func (r *mockRuntime) ListImages(context.Context) ([]container.ImageInfo, error) { return nil, nil }
func (r *mockRuntime) RemoveImage(context.Context, string, bool) error           { return nil }
func (r *mockRuntime) ContainerLogs(ctx context.Context, id string, opts container.LogOptions) (io.ReadCloser, error) {
	return nil, nil
}
func (r *mockRuntime) ExecInContainer(context.Context, string, []string, container.ExecOptions) (*container.ExecResult, error) {
	return nil, nil
}
func (r *mockRuntime) CreateNetwork(context.Context, string, container.NetworkOptions) (string, error) {
	return "", nil
}
func (r *mockRuntime) RemoveNetwork(context.Context, string) error                  { return nil }
func (r *mockRuntime) ConnectNetwork(context.Context, string, string) error         { return nil }
func (r *mockRuntime) DisconnectNetwork(context.Context, string, string) error      { return nil }
func (r *mockRuntime) ListNetworks(context.Context) ([]container.NetworkInfo, error) { return nil, nil }
func (r *mockRuntime) CreateVolume(context.Context, string, container.VolumeOptions) (string, error) {
	return "", nil
}
func (r *mockRuntime) RemoveVolume(context.Context, string, bool) error { return nil }
func (r *mockRuntime) ListVolumes(context.Context) ([]container.VolumeInfo, error) {
	return nil, nil
}
func (r *mockRuntime) InspectVolume(context.Context, string) (*container.VolumeInfo, error) {
	return nil, nil
}
func (r *mockRuntime) Ping(context.Context) error                        { return nil }
func (r *mockRuntime) Info(context.Context) (*container.RuntimeInfo, error) { return nil, nil }

var _ container.Runtime = (*mockRuntime)(nil)

func newTestManager(t *testing.T, chains []chain.Chain) (*DefaultManager, *Store) {
	t.Helper()
	dir := t.TempDir()
	store := NewStore(dir)
	mgr := NewDefaultManager(store, &mockRuntime{}, func() []chain.Chain {
		return chains
	}, "test-project")
	return mgr, store
}

func saveTestSnapshot(t *testing.T, store *Store, name string, snap *Snapshot) {
	t.Helper()
	require.NoError(t, store.EnsureDirs(name))
	require.NoError(t, store.SaveMetadata(snap))
}

func TestNewDefaultManager_ReturnsNonNil(t *testing.T) {
	mgr, _ := newTestManager(t, nil)
	require.NotNil(t, mgr)
}

func TestSave_CreatesSnapshot(t *testing.T) {
	mc := &mockChain{
		name:      "chain1",
		engine:    "anvil",
		chainID:   1,
		rpcResult: json.RawMessage(`"0x5"`),
	}
	mgr, store := newTestManager(t, []chain.Chain{mc})
	ctx := context.Background()

	snap, err := mgr.Save(ctx, "my-snap", SaveOptions{
		Description: "test save",
		Tags:        []string{"v1"},
	})
	require.NoError(t, err)
	require.NotNil(t, snap)

	assert.Equal(t, "my-snap", snap.Name)
	assert.Equal(t, "test-project", snap.Project)
	assert.Equal(t, "test save", snap.Description)
	assert.Equal(t, []string{"v1"}, snap.Tags)

	cs, ok := snap.Chains["chain1"]
	require.True(t, ok)
	assert.Equal(t, "chain1", cs.Name)
	assert.Equal(t, "anvil", cs.Engine)
	assert.Equal(t, uint64(1), cs.ChainID)
	assert.Equal(t, uint64(5), cs.BlockNumber)

	assert.True(t, store.Exists("my-snap"))
}

func TestSave_ErrorWhenSnapshotAlreadyExists(t *testing.T) {
	mgr, store := newTestManager(t, nil)
	ctx := context.Background()

	saveTestSnapshot(t, store, "existing", &Snapshot{
		Name:      "existing",
		Project:   "proj",
		CreatedAt: time.Now().UTC(),
		Chains:    map[string]ChainSnapshot{},
	})

	_, err := mgr.Save(ctx, "existing", SaveOptions{})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "already exists")
}

func TestSave_ContinuesOnChainExportError(t *testing.T) {
	good := &mockChain{
		name:      "good-chain",
		engine:    "anvil",
		chainID:   1,
		rpcResult: json.RawMessage(`"0x1"`),
	}
	bad := &mockChain{
		name:      "bad-chain",
		engine:    "anvil",
		chainID:   2,
		exportErr: assert.AnError,
	}
	mgr, _ := newTestManager(t, []chain.Chain{good, bad})

	snap, err := mgr.Save(context.Background(), "partial-snap", SaveOptions{})
	require.NoError(t, err)

	_, goodOK := snap.Chains["good-chain"]
	assert.True(t, goodOK)

	_, badOK := snap.Chains["bad-chain"]
	assert.False(t, badOK)
}

func TestSave_NoChains(t *testing.T) {
	mgr, _ := newTestManager(t, nil)

	snap, err := mgr.Save(context.Background(), "empty-snap", SaveOptions{})
	require.NoError(t, err)
	assert.Empty(t, snap.Chains)
}

func TestSave_MultipleChains(t *testing.T) {
	chains := []chain.Chain{
		&mockChain{name: "c1", engine: "anvil", chainID: 1, rpcResult: json.RawMessage(`"0x1"`)},
		&mockChain{name: "c2", engine: "hardhat", chainID: 2, rpcResult: json.RawMessage(`"0x2"`)},
		&mockChain{name: "c3", engine: "geth", chainID: 3, rpcResult: json.RawMessage(`"0x3"`)},
	}
	mgr, _ := newTestManager(t, chains)

	snap, err := mgr.Save(context.Background(), "multi-snap", SaveOptions{})
	require.NoError(t, err)
	assert.Len(t, snap.Chains, 3)
}

func TestRestore_Success(t *testing.T) {
	mc := &mockChain{
		name:    "chain1",
		engine:  "anvil",
		chainID: 1,
	}
	mgr, store := newTestManager(t, []chain.Chain{mc})
	ctx := context.Background()

	snapName := "restore-snap"
	require.NoError(t, store.EnsureDirs(snapName))
	chainsDir := filepath.Join(store.SnapshotDir(snapName), "chains", "chain1")
	require.NoError(t, os.MkdirAll(chainsDir, 0o755))
	stateFile := filepath.Join(chainsDir, "state.json")
	require.NoError(t, os.WriteFile(stateFile, []byte(`{}`), 0o644))

	snap := &Snapshot{
		Name:      snapName,
		Project:   "proj",
		CreatedAt: time.Now().UTC(),
		Chains: map[string]ChainSnapshot{
			"chain1": {
				Name:      "chain1",
				StateFile: filepath.Join("chains", "chain1", "state.json"),
			},
		},
	}
	require.NoError(t, store.SaveMetadata(snap))

	err := mgr.Restore(ctx, snapName, RestoreOptions{})
	require.NoError(t, err)

	expectedPath := filepath.Join(store.SnapshotDir(snapName), "chains", "chain1", "state.json")
	assert.Equal(t, expectedPath, mc.importedFrom)
}

func TestRestore_ErrorWhenSnapshotNotFound(t *testing.T) {
	mgr, _ := newTestManager(t, nil)

	err := mgr.Restore(context.Background(), "nonexistent", RestoreOptions{})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "nonexistent")
}

func TestRestore_SkipsChainNotInSnapshot(t *testing.T) {
	mc := &mockChain{name: "chain-not-in-snap"}
	mgr, store := newTestManager(t, []chain.Chain{mc})

	snapName := "no-chain-snap"
	saveTestSnapshot(t, store, snapName, &Snapshot{
		Name:      snapName,
		Project:   "proj",
		CreatedAt: time.Now().UTC(),
		Chains:    map[string]ChainSnapshot{}, // no chains
	})

	err := mgr.Restore(context.Background(), snapName, RestoreOptions{})
	require.NoError(t, err)
	assert.Empty(t, mc.importedFrom)
}

func TestRestore_ErrorOnImportFailure(t *testing.T) {
	mc := &mockChain{
		name:      "fail-chain",
		importErr: assert.AnError,
	}
	mgr, store := newTestManager(t, []chain.Chain{mc})

	snapName := "import-fail-snap"
	require.NoError(t, store.EnsureDirs(snapName))
	chainsDir := filepath.Join(store.SnapshotDir(snapName), "chains", "fail-chain")
	require.NoError(t, os.MkdirAll(chainsDir, 0o755))
	require.NoError(t, os.WriteFile(filepath.Join(chainsDir, "state.json"), []byte(`{}`), 0o644))

	saveTestSnapshot(t, store, snapName, &Snapshot{
		Name:      snapName,
		Project:   "proj",
		CreatedAt: time.Now().UTC(),
		Chains: map[string]ChainSnapshot{
			"fail-chain": {
				Name:      "fail-chain",
				StateFile: filepath.Join("chains", "fail-chain", "state.json"),
			},
		},
	})

	err := mgr.Restore(context.Background(), snapName, RestoreOptions{})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "failed to restore chain fail-chain")
}

func TestList_ReturnsAllSnapshots(t *testing.T) {
	mgr, store := newTestManager(t, nil)

	for _, name := range []string{"a", "b", "c"} {
		saveTestSnapshot(t, store, name, &Snapshot{
			Name:      name,
			Project:   "proj",
			CreatedAt: time.Now().UTC(),
			Chains:    map[string]ChainSnapshot{},
		})
	}

	list, err := mgr.List(context.Background())
	require.NoError(t, err)
	assert.Len(t, list, 3)
}

func TestList_EmptyStore(t *testing.T) {
	mgr, _ := newTestManager(t, nil)

	list, err := mgr.List(context.Background())
	require.NoError(t, err)
	assert.Nil(t, list)
}

func TestGet_ReturnsSnapshot(t *testing.T) {
	mgr, store := newTestManager(t, nil)

	saveTestSnapshot(t, store, "get-snap", &Snapshot{
		Name:        "get-snap",
		Description: "found it",
		Project:     "proj",
		CreatedAt:   time.Now().UTC(),
		Chains:      map[string]ChainSnapshot{},
	})

	snap, err := mgr.Get(context.Background(), "get-snap")
	require.NoError(t, err)
	assert.Equal(t, "get-snap", snap.Name)
	assert.Equal(t, "found it", snap.Description)
}

func TestGet_ErrorWhenNotFound(t *testing.T) {
	mgr, _ := newTestManager(t, nil)

	_, err := mgr.Get(context.Background(), "missing")
	require.Error(t, err)
}

func TestDelete_RemovesSnapshot(t *testing.T) {
	mgr, store := newTestManager(t, nil)

	saveTestSnapshot(t, store, "del-snap", &Snapshot{
		Name:      "del-snap",
		Project:   "proj",
		CreatedAt: time.Now().UTC(),
		Chains:    map[string]ChainSnapshot{},
	})
	require.True(t, store.Exists("del-snap"))

	err := mgr.Delete(context.Background(), "del-snap")
	require.NoError(t, err)
	assert.False(t, store.Exists("del-snap"))
}

func TestDelete_ErrorWhenNotFound(t *testing.T) {
	mgr, _ := newTestManager(t, nil)

	err := mgr.Delete(context.Background(), "no-such-snap")
	require.Error(t, err)
}

func TestExport_ErrorWhenSnapshotNotFound(t *testing.T) {
	mgr, _ := newTestManager(t, nil)

	err := mgr.Export(context.Background(), "nonexistent", filepath.Join(t.TempDir(), "out.tar.gz"))
	require.Error(t, err)
	assert.Contains(t, err.Error(), "not found")
}

func TestExport_CreatesArchive(t *testing.T) {
	mgr, store := newTestManager(t, nil)
	ctx := context.Background()

	saveTestSnapshot(t, store, "export-snap", &Snapshot{
		Name:      "export-snap",
		Project:   "proj",
		CreatedAt: time.Now().UTC(),
		Chains:    map[string]ChainSnapshot{},
	})

	outputPath := filepath.Join(t.TempDir(), "export-snap.tar.gz")
	err := mgr.Export(ctx, "export-snap", outputPath)
	require.NoError(t, err)
	assert.FileExists(t, outputPath)
}

func TestExportAndImport_Roundtrip(t *testing.T) {
	mgr1, store1 := newTestManager(t, nil)
	ctx := context.Background()

	snap := &Snapshot{
		Name:        "roundtrip",
		Description: "test roundtrip",
		Project:     "proj",
		CreatedAt:   time.Now().UTC(),
		Chains: map[string]ChainSnapshot{
			"chain1": {
				Name:    "chain1",
				Engine:  "anvil",
				ChainID: 1,
			},
		},
	}
	saveTestSnapshot(t, store1, "roundtrip", snap)

	archivePath := filepath.Join(t.TempDir(), "roundtrip.tar.gz")
	err := mgr1.Export(ctx, "roundtrip", archivePath)
	require.NoError(t, err)

	mgr2, store2 := newTestManager(t, nil)
	imported, err := mgr2.Import(ctx, archivePath)
	require.NoError(t, err)
	require.NotNil(t, imported)
	assert.Equal(t, "roundtrip", imported.Name)
	assert.Equal(t, "test roundtrip", imported.Description)

	assert.True(t, store2.Exists("roundtrip"))
}

func TestImport_ErrorForInvalidArchive(t *testing.T) {
	mgr, _ := newTestManager(t, nil)

	badFile := filepath.Join(t.TempDir(), "bad.tar.gz")
	require.NoError(t, os.WriteFile(badFile, []byte("not a tar gz"), 0o644))

	_, err := mgr.Import(context.Background(), badFile)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "failed to extract snapshot")
}

func TestImport_ErrorForMissingFile(t *testing.T) {
	mgr, _ := newTestManager(t, nil)

	_, err := mgr.Import(context.Background(), filepath.Join(t.TempDir(), "does-not-exist.tar.gz"))
	require.Error(t, err)
}

func TestDiff_ComparesChainBlockNumbers(t *testing.T) {
	mgr, store := newTestManager(t, nil)
	ctx := context.Background()

	saveTestSnapshot(t, store, "snap-a", &Snapshot{
		Name:      "snap-a",
		Project:   "proj",
		CreatedAt: time.Now().UTC(),
		Chains: map[string]ChainSnapshot{
			"chain1": {Name: "chain1", BlockNumber: 100},
		},
	})
	saveTestSnapshot(t, store, "snap-b", &Snapshot{
		Name:      "snap-b",
		Project:   "proj",
		CreatedAt: time.Now().UTC(),
		Chains: map[string]ChainSnapshot{
			"chain1": {Name: "chain1", BlockNumber: 150},
		},
	})

	result, err := mgr.Diff(ctx, "snap-a", "snap-b")
	require.NoError(t, err)
	require.NotNil(t, result)

	assert.Equal(t, "snap-a", result.Snapshot1)
	assert.Equal(t, "snap-b", result.Snapshot2)
	require.Len(t, result.ChainDiffs, 1)
	assert.Equal(t, "chain1", result.ChainDiffs[0].Chain)
	assert.Equal(t, uint64(100), result.ChainDiffs[0].Block1)
	assert.Equal(t, uint64(150), result.ChainDiffs[0].Block2)
	assert.Equal(t, int64(50), result.ChainDiffs[0].BlockDelta)
}

func TestDiff_NegativeBlockDelta(t *testing.T) {
	mgr, store := newTestManager(t, nil)

	saveTestSnapshot(t, store, "newer", &Snapshot{
		Name:      "newer",
		Project:   "proj",
		CreatedAt: time.Now().UTC(),
		Chains: map[string]ChainSnapshot{
			"chain1": {Name: "chain1", BlockNumber: 200},
		},
	})
	saveTestSnapshot(t, store, "older", &Snapshot{
		Name:      "older",
		Project:   "proj",
		CreatedAt: time.Now().UTC(),
		Chains: map[string]ChainSnapshot{
			"chain1": {Name: "chain1", BlockNumber: 50},
		},
	})

	result, err := mgr.Diff(context.Background(), "newer", "older")
	require.NoError(t, err)
	require.Len(t, result.ChainDiffs, 1)
	assert.Equal(t, int64(-150), result.ChainDiffs[0].BlockDelta)
}

func TestDiff_ChainOnlyInOneSnapshot(t *testing.T) {
	mgr, store := newTestManager(t, nil)

	saveTestSnapshot(t, store, "has-chain", &Snapshot{
		Name:      "has-chain",
		Project:   "proj",
		CreatedAt: time.Now().UTC(),
		Chains: map[string]ChainSnapshot{
			"chain1": {Name: "chain1", BlockNumber: 100},
		},
	})
	saveTestSnapshot(t, store, "no-chain", &Snapshot{
		Name:      "no-chain",
		Project:   "proj",
		CreatedAt: time.Now().UTC(),
		Chains:    map[string]ChainSnapshot{},
	})

	result, err := mgr.Diff(context.Background(), "has-chain", "no-chain")
	require.NoError(t, err)
	assert.Empty(t, result.ChainDiffs)
}

func TestDiff_ComparesVolumes(t *testing.T) {
	mgr, store := newTestManager(t, nil)

	saveTestSnapshot(t, store, "vol-a", &Snapshot{
		Name:      "vol-a",
		Project:   "proj",
		CreatedAt: time.Now().UTC(),
		Chains:    map[string]ChainSnapshot{},
		Volumes: []VolumeSnapshot{
			{Name: "data-vol", Service: "svc1", Size: 1000},
		},
	})
	saveTestSnapshot(t, store, "vol-b", &Snapshot{
		Name:      "vol-b",
		Project:   "proj",
		CreatedAt: time.Now().UTC(),
		Chains:    map[string]ChainSnapshot{},
		Volumes: []VolumeSnapshot{
			{Name: "data-vol", Service: "svc1", Size: 2000},
		},
	})

	result, err := mgr.Diff(context.Background(), "vol-a", "vol-b")
	require.NoError(t, err)
	require.Len(t, result.VolumeDiffs, 1)
	assert.Equal(t, "data-vol", result.VolumeDiffs[0].Volume)
	assert.Equal(t, int64(1000), result.VolumeDiffs[0].Size1)
	assert.Equal(t, int64(2000), result.VolumeDiffs[0].Size2)
}

func TestDiff_ErrorWhenFirstSnapshotMissing(t *testing.T) {
	mgr, store := newTestManager(t, nil)

	saveTestSnapshot(t, store, "exists", &Snapshot{
		Name:      "exists",
		Project:   "proj",
		CreatedAt: time.Now().UTC(),
		Chains:    map[string]ChainSnapshot{},
	})

	_, err := mgr.Diff(context.Background(), "missing", "exists")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "missing")
}

func TestDiff_ErrorWhenSecondSnapshotMissing(t *testing.T) {
	mgr, store := newTestManager(t, nil)

	saveTestSnapshot(t, store, "exists", &Snapshot{
		Name:      "exists",
		Project:   "proj",
		CreatedAt: time.Now().UTC(),
		Chains:    map[string]ChainSnapshot{},
	})

	_, err := mgr.Diff(context.Background(), "exists", "missing")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "missing")
}

func TestDiff_EmptySnapshots(t *testing.T) {
	mgr, store := newTestManager(t, nil)

	saveTestSnapshot(t, store, "empty1", &Snapshot{
		Name:      "empty1",
		Project:   "proj",
		CreatedAt: time.Now().UTC(),
		Chains:    map[string]ChainSnapshot{},
	})
	saveTestSnapshot(t, store, "empty2", &Snapshot{
		Name:      "empty2",
		Project:   "proj",
		CreatedAt: time.Now().UTC(),
		Chains:    map[string]ChainSnapshot{},
	})

	result, err := mgr.Diff(context.Background(), "empty1", "empty2")
	require.NoError(t, err)
	assert.Empty(t, result.ChainDiffs)
	assert.Empty(t, result.VolumeDiffs)
}

func TestCopyDir_CopiesFilesAndDirectories(t *testing.T) {
	src := t.TempDir()
	dst := filepath.Join(t.TempDir(), "dest")

	require.NoError(t, os.MkdirAll(filepath.Join(src, "subdir"), 0o755))
	require.NoError(t, os.WriteFile(filepath.Join(src, "file1.txt"), []byte("hello"), 0o644))
	require.NoError(t, os.WriteFile(filepath.Join(src, "subdir", "file2.txt"), []byte("world"), 0o644))

	require.NoError(t, os.MkdirAll(dst, 0o755))
	err := copyDir(src, dst)
	require.NoError(t, err)

	data1, err := os.ReadFile(filepath.Join(dst, "file1.txt"))
	require.NoError(t, err)
	assert.Equal(t, "hello", string(data1))

	data2, err := os.ReadFile(filepath.Join(dst, "subdir", "file2.txt"))
	require.NoError(t, err)
	assert.Equal(t, "world", string(data2))
}

func TestCopyDir_EmptyDirectory(t *testing.T) {
	src := t.TempDir()
	dst := filepath.Join(t.TempDir(), "dest")
	require.NoError(t, os.MkdirAll(dst, 0o755))

	err := copyDir(src, dst)
	require.NoError(t, err)

	entries, err := os.ReadDir(dst)
	require.NoError(t, err)
	assert.Empty(t, entries)
}
