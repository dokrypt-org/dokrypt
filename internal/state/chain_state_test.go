package state

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"math/big"
	"os"
	"path/filepath"
	"testing"

	"github.com/dokrypt/dokrypt/internal/chain"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type mockChain struct {
	name        string
	engine      string
	chainID     uint64
	exportErr   error
	importErr   error
	rpcResult   json.RawMessage
	rpcErr      error
	exportedTo  string // records path passed to ExportState
	importedFrom string // records path passed to ImportState
}

func (m *mockChain) Name() string    { return m.name }
func (m *mockChain) Engine() string  { return m.engine }
func (m *mockChain) ChainID() uint64 { return m.chainID }

func (m *mockChain) ExportState(_ context.Context, path string) error {
	m.exportedTo = path
	if m.exportErr != nil {
		return m.exportErr
	}
	return os.WriteFile(path, []byte(`{"state":"exported"}`), 0o644)
}

func (m *mockChain) ImportState(_ context.Context, path string) error {
	m.importedFrom = path
	return m.importErr
}

func (m *mockChain) RPC(_ context.Context, method string, _ ...any) (json.RawMessage, error) {
	if m.rpcErr != nil {
		return nil, m.rpcErr
	}
	return m.rpcResult, nil
}

func (m *mockChain) Start(context.Context) error                          { return nil }
func (m *mockChain) Stop(context.Context) error                           { return nil }
func (m *mockChain) IsRunning(context.Context) bool                       { return false }
func (m *mockChain) Health(context.Context) error                         { return nil }
func (m *mockChain) RPCURL() string                                       { return "" }
func (m *mockChain) WSURL() string                                        { return "" }
func (m *mockChain) Accounts() []chain.Account                            { return nil }
func (m *mockChain) FundAccount(context.Context, string, *big.Int) error  { return nil }
func (m *mockChain) ImpersonateAccount(context.Context, string) error     { return nil }
func (m *mockChain) GenerateAccounts(context.Context, int) ([]chain.Account, error) {
	return nil, nil
}
func (m *mockChain) MineBlocks(context.Context, uint64) error             { return nil }
func (m *mockChain) SetBlockTime(context.Context, uint64) error           { return nil }
func (m *mockChain) SetGasPrice(context.Context, uint64) error            { return nil }
func (m *mockChain) TimeTravel(context.Context, int64) error              { return nil }
func (m *mockChain) SetBalance(context.Context, string, *big.Int) error   { return nil }
func (m *mockChain) SetStorageAt(context.Context, string, string, string) error {
	return nil
}
func (m *mockChain) TakeSnapshot(context.Context) (string, error) { return "", nil }
func (m *mockChain) RevertSnapshot(context.Context, string) error { return nil }
func (m *mockChain) Fork(context.Context, chain.ForkOptions) error { return nil }
func (m *mockChain) ForkInfo() *chain.ForkInfo                    { return nil }
func (m *mockChain) Logs(context.Context, bool) (io.ReadCloser, error) {
	return nil, nil
}

var _ chain.Chain = (*mockChain)(nil)

func TestNewChainStateManager_ReturnsNonNil(t *testing.T) {
	store := NewStore(t.TempDir())
	mgr := NewChainStateManager(store)
	require.NotNil(t, mgr)
}

func TestExportChainState_Success(t *testing.T) {
	dir := t.TempDir()
	store := NewStore(dir)
	require.NoError(t, store.EnsureDirs("snap1"))

	mgr := NewChainStateManager(store)

	mc := &mockChain{
		name:    "mainnet",
		engine:  "anvil",
		chainID: 1,
		rpcResult: json.RawMessage(`"0xa"` ), // 10 in hex
	}

	ctx := context.Background()
	cs, err := mgr.ExportChainState(ctx, "snap1", mc)
	require.NoError(t, err)
	require.NotNil(t, cs)

	assert.Equal(t, "mainnet", cs.Name)
	assert.Equal(t, "anvil", cs.Engine)
	assert.Equal(t, uint64(1), cs.ChainID)
	assert.Equal(t, uint64(10), cs.BlockNumber)
	assert.Equal(t, filepath.Join("chains", "mainnet", "state.json"), cs.StateFile)

	stateFile := filepath.Join(store.ChainsDir("snap1"), "mainnet", "state.json")
	assert.FileExists(t, stateFile)
}

func TestExportChainState_CreatesChainDirectory(t *testing.T) {
	dir := t.TempDir()
	store := NewStore(dir)
	require.NoError(t, os.MkdirAll(store.ChainsDir("snap2"), 0o755))

	mgr := NewChainStateManager(store)
	mc := &mockChain{
		name:      "mychain",
		engine:    "geth",
		chainID:   42,
		rpcResult: json.RawMessage(`"0x0"`),
	}

	cs, err := mgr.ExportChainState(context.Background(), "snap2", mc)
	require.NoError(t, err)
	require.NotNil(t, cs)

	chainDir := filepath.Join(store.ChainsDir("snap2"), "mychain")
	info, err := os.Stat(chainDir)
	require.NoError(t, err)
	assert.True(t, info.IsDir())
}

func TestExportChainState_ExportError(t *testing.T) {
	dir := t.TempDir()
	store := NewStore(dir)
	require.NoError(t, store.EnsureDirs("snap3"))

	mgr := NewChainStateManager(store)
	mc := &mockChain{
		name:      "badchain",
		engine:    "anvil",
		chainID:   1,
		exportErr: errors.New("export blew up"),
	}

	cs, err := mgr.ExportChainState(context.Background(), "snap3", mc)
	require.Error(t, err)
	assert.Nil(t, cs)
	assert.Contains(t, err.Error(), "failed to export state for chain badchain")
	assert.Contains(t, err.Error(), "export blew up")
}

func TestExportChainState_RPCErrorReturnsZeroBlockNumber(t *testing.T) {
	dir := t.TempDir()
	store := NewStore(dir)
	require.NoError(t, store.EnsureDirs("snap-rpc-err"))

	mgr := NewChainStateManager(store)
	mc := &mockChain{
		name:    "chain-rpc-err",
		engine:  "anvil",
		chainID: 1,
		rpcErr:  errors.New("rpc failed"),
	}

	cs, err := mgr.ExportChainState(context.Background(), "snap-rpc-err", mc)
	require.NoError(t, err)
	assert.Equal(t, uint64(0), cs.BlockNumber)
}

func TestExportChainState_InvalidHexBlockNumberReturnsZero(t *testing.T) {
	dir := t.TempDir()
	store := NewStore(dir)
	require.NoError(t, store.EnsureDirs("snap-bad-hex"))

	mgr := NewChainStateManager(store)
	mc := &mockChain{
		name:      "chain-bad-hex",
		engine:    "anvil",
		chainID:   1,
		rpcResult: json.RawMessage(`"not-a-hex"`),
	}

	cs, err := mgr.ExportChainState(context.Background(), "snap-bad-hex", mc)
	require.NoError(t, err)
	assert.Equal(t, uint64(0), cs.BlockNumber)
}

func TestExportChainState_UnmarshalErrorReturnsZero(t *testing.T) {
	dir := t.TempDir()
	store := NewStore(dir)
	require.NoError(t, store.EnsureDirs("snap-bad-json"))

	mgr := NewChainStateManager(store)
	mc := &mockChain{
		name:      "chain-bad-json",
		engine:    "anvil",
		chainID:   1,
		rpcResult: json.RawMessage(`12345`), // not a string
	}

	cs, err := mgr.ExportChainState(context.Background(), "snap-bad-json", mc)
	require.NoError(t, err)
	assert.Equal(t, uint64(0), cs.BlockNumber)
}

func TestImportChainState_Success(t *testing.T) {
	dir := t.TempDir()
	store := NewStore(dir)
	require.NoError(t, store.EnsureDirs("snap-import"))

	chainsDir := filepath.Join(store.SnapshotDir("snap-import"), "chains", "mychain")
	require.NoError(t, os.MkdirAll(chainsDir, 0o755))
	stateFile := filepath.Join(chainsDir, "state.json")
	require.NoError(t, os.WriteFile(stateFile, []byte(`{"state":"data"}`), 0o644))

	mgr := NewChainStateManager(store)
	mc := &mockChain{name: "mychain"}

	cs := ChainSnapshot{
		Name:      "mychain",
		StateFile: filepath.Join("chains", "mychain", "state.json"),
	}

	err := mgr.ImportChainState(context.Background(), "snap-import", mc, cs)
	require.NoError(t, err)

	expectedPath := filepath.Join(store.SnapshotDir("snap-import"), cs.StateFile)
	assert.Equal(t, expectedPath, mc.importedFrom)
}

func TestImportChainState_StateFileNotFound(t *testing.T) {
	dir := t.TempDir()
	store := NewStore(dir)
	require.NoError(t, store.EnsureDirs("snap-missing"))

	mgr := NewChainStateManager(store)
	mc := &mockChain{name: "mychain"}

	cs := ChainSnapshot{
		Name:      "mychain",
		StateFile: filepath.Join("chains", "mychain", "state.json"), // does not exist
	}

	err := mgr.ImportChainState(context.Background(), "snap-missing", mc, cs)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "state file not found for chain mychain")
}

func TestImportChainState_ImportError(t *testing.T) {
	dir := t.TempDir()
	store := NewStore(dir)
	require.NoError(t, store.EnsureDirs("snap-imp-err"))

	chainsDir := filepath.Join(store.SnapshotDir("snap-imp-err"), "chains", "mychain")
	require.NoError(t, os.MkdirAll(chainsDir, 0o755))
	stateFile := filepath.Join(chainsDir, "state.json")
	require.NoError(t, os.WriteFile(stateFile, []byte(`{}`), 0o644))

	mgr := NewChainStateManager(store)
	mc := &mockChain{
		name:      "mychain",
		importErr: errors.New("import failed"),
	}

	cs := ChainSnapshot{
		Name:      "mychain",
		StateFile: filepath.Join("chains", "mychain", "state.json"),
	}

	err := mgr.ImportChainState(context.Background(), "snap-imp-err", mc, cs)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "failed to import state for chain mychain")
	assert.Contains(t, err.Error(), "import failed")
}

func TestGetBlockNumber_VariousHexValues(t *testing.T) {
	tests := []struct {
		name     string
		rpcJSON  string
		expected uint64
	}{
		{"zero", `"0x0"`, 0},
		{"small", `"0xa"`, 10},
		{"large", `"0x1234567"`, 0x1234567},
		{"ff", `"0xff"`, 255},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			dir := t.TempDir()
			store := NewStore(dir)
			snapName := fmt.Sprintf("snap-%s", tt.name)
			require.NoError(t, store.EnsureDirs(snapName))

			mgr := NewChainStateManager(store)
			mc := &mockChain{
				name:      "chain",
				engine:    "anvil",
				chainID:   1,
				rpcResult: json.RawMessage(tt.rpcJSON),
			}

			cs, err := mgr.ExportChainState(context.Background(), snapName, mc)
			require.NoError(t, err)
			assert.Equal(t, tt.expected, cs.BlockNumber)
		})
	}
}
