package chain

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"math/big"
	"testing"
	"time"

	"github.com/dokrypt/dokrypt/internal/config"
	"github.com/dokrypt/dokrypt/internal/container"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type mockChain struct {
	name    string
	chainID uint64
	stopped bool
	stopErr error
}

func (m *mockChain) Start(_ context.Context) error                          { return nil }
func (m *mockChain) Stop(_ context.Context) error                           { m.stopped = true; return m.stopErr }
func (m *mockChain) IsRunning(_ context.Context) bool                       { return !m.stopped }
func (m *mockChain) Health(_ context.Context) error                         { return nil }
func (m *mockChain) Name() string                                           { return m.name }
func (m *mockChain) ChainID() uint64                                        { return m.chainID }
func (m *mockChain) RPCURL() string                                         { return "http://localhost:8545" }
func (m *mockChain) WSURL() string                                          { return "ws://localhost:8545" }
func (m *mockChain) Engine() string                                         { return "mock" }
func (m *mockChain) Accounts() []Account                                    { return nil }
func (m *mockChain) FundAccount(_ context.Context, _ string, _ *big.Int) error { return nil }
func (m *mockChain) ImpersonateAccount(_ context.Context, _ string) error   { return nil }
func (m *mockChain) GenerateAccounts(_ context.Context, _ int) ([]Account, error) {
	return nil, nil
}
func (m *mockChain) MineBlocks(_ context.Context, _ uint64) error           { return nil }
func (m *mockChain) SetBlockTime(_ context.Context, _ uint64) error         { return nil }
func (m *mockChain) SetGasPrice(_ context.Context, _ uint64) error          { return nil }
func (m *mockChain) TimeTravel(_ context.Context, _ int64) error            { return nil }
func (m *mockChain) SetBalance(_ context.Context, _ string, _ *big.Int) error { return nil }
func (m *mockChain) SetStorageAt(_ context.Context, _, _, _ string) error   { return nil }
func (m *mockChain) TakeSnapshot(_ context.Context) (string, error)         { return "", nil }
func (m *mockChain) RevertSnapshot(_ context.Context, _ string) error       { return nil }
func (m *mockChain) ExportState(_ context.Context, _ string) error          { return nil }
func (m *mockChain) ImportState(_ context.Context, _ string) error          { return nil }
func (m *mockChain) Fork(_ context.Context, _ ForkOptions) error            { return nil }
func (m *mockChain) ForkInfo() *ForkInfo                                    { return nil }
func (m *mockChain) RPC(_ context.Context, _ string, _ ...any) (json.RawMessage, error) {
	return nil, nil
}
func (m *mockChain) Logs(_ context.Context, _ bool) (io.ReadCloser, error) { return nil, nil }

type mockRuntime struct{}

func (mr *mockRuntime) CreateContainer(_ context.Context, _ *container.ContainerConfig) (string, error) {
	return "", nil
}
func (mr *mockRuntime) StartContainer(_ context.Context, _ string) error { return nil }
func (mr *mockRuntime) StopContainer(_ context.Context, _ string, _ time.Duration) error {
	return nil
}
func (mr *mockRuntime) RemoveContainer(_ context.Context, _ string, _ bool) error { return nil }
func (mr *mockRuntime) ListContainers(_ context.Context, _ container.ListOptions) ([]container.ContainerInfo, error) {
	return nil, nil
}
func (mr *mockRuntime) InspectContainer(_ context.Context, _ string) (*container.ContainerInfo, error) {
	return nil, nil
}
func (mr *mockRuntime) WaitContainer(_ context.Context, _ string) (int64, error) { return 0, nil }
func (mr *mockRuntime) PullImage(_ context.Context, _ string) error              { return nil }
func (mr *mockRuntime) BuildImage(_ context.Context, _ string, _ container.BuildOptions) (string, error) {
	return "", nil
}
func (mr *mockRuntime) ListImages(_ context.Context) ([]container.ImageInfo, error) { return nil, nil }
func (mr *mockRuntime) RemoveImage(_ context.Context, _ string, _ bool) error      { return nil }
func (mr *mockRuntime) ContainerLogs(_ context.Context, _ string, _ container.LogOptions) (io.ReadCloser, error) {
	return nil, nil
}
func (mr *mockRuntime) ExecInContainer(_ context.Context, _ string, _ []string, _ container.ExecOptions) (*container.ExecResult, error) {
	return nil, nil
}
func (mr *mockRuntime) CreateNetwork(_ context.Context, _ string, _ container.NetworkOptions) (string, error) {
	return "", nil
}
func (mr *mockRuntime) RemoveNetwork(_ context.Context, _ string) error { return nil }
func (mr *mockRuntime) ConnectNetwork(_ context.Context, _, _ string) error {
	return nil
}
func (mr *mockRuntime) DisconnectNetwork(_ context.Context, _, _ string) error {
	return nil
}
func (mr *mockRuntime) ListNetworks(_ context.Context) ([]container.NetworkInfo, error) {
	return nil, nil
}
func (mr *mockRuntime) CreateVolume(_ context.Context, _ string, _ container.VolumeOptions) (string, error) {
	return "", nil
}
func (mr *mockRuntime) RemoveVolume(_ context.Context, _ string, _ bool) error { return nil }
func (mr *mockRuntime) ListVolumes(_ context.Context) ([]container.VolumeInfo, error) {
	return nil, nil
}
func (mr *mockRuntime) InspectVolume(_ context.Context, _ string) (*container.VolumeInfo, error) {
	return nil, nil
}
func (mr *mockRuntime) Ping(_ context.Context) error                         { return nil }
func (mr *mockRuntime) Info(_ context.Context) (*container.RuntimeInfo, error) { return nil, nil }

func TestNewManager(t *testing.T) {
	rt := &mockRuntime{}
	mgr := NewManager(rt)
	require.NotNil(t, mgr)
	assert.NotNil(t, mgr.chains)
	assert.NotNil(t, mgr.factories)
	assert.Equal(t, 0, len(mgr.chains))
}

func TestManager_RegisterFactory(t *testing.T) {
	mgr := NewManager(&mockRuntime{})

	factory := func(name string, cfg config.ChainConfig, runtime container.Runtime, projectName string) (Chain, error) {
		return &mockChain{name: name, chainID: cfg.ChainID}, nil
	}

	mgr.RegisterFactory("mock", factory)
	assert.Contains(t, mgr.factories, "mock")
}

func TestManager_CreateChain_Success(t *testing.T) {
	mgr := NewManager(&mockRuntime{})

	factory := func(name string, cfg config.ChainConfig, runtime container.Runtime, projectName string) (Chain, error) {
		return &mockChain{name: name, chainID: cfg.ChainID}, nil
	}
	mgr.RegisterFactory("mock", factory)

	cfg := config.ChainConfig{
		Engine:  "mock",
		ChainID: 31337,
	}

	c, err := mgr.CreateChain("test-chain", cfg, "my-project")
	require.NoError(t, err)
	require.NotNil(t, c)
	assert.Equal(t, "test-chain", c.Name())
	assert.Equal(t, uint64(31337), c.ChainID())
}

func TestManager_CreateChain_DuplicateName(t *testing.T) {
	mgr := NewManager(&mockRuntime{})

	factory := func(name string, cfg config.ChainConfig, runtime container.Runtime, projectName string) (Chain, error) {
		return &mockChain{name: name, chainID: cfg.ChainID}, nil
	}
	mgr.RegisterFactory("mock", factory)

	cfg := config.ChainConfig{Engine: "mock", ChainID: 31337}

	_, err := mgr.CreateChain("test-chain", cfg, "my-project")
	require.NoError(t, err)

	_, err = mgr.CreateChain("test-chain", cfg, "my-project")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "already exists")
}

func TestManager_CreateChain_UnsupportedEngine(t *testing.T) {
	mgr := NewManager(&mockRuntime{})

	cfg := config.ChainConfig{Engine: "nonexistent", ChainID: 31337}

	_, err := mgr.CreateChain("test-chain", cfg, "my-project")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "unsupported chain engine")
}

func TestManager_CreateChain_FactoryError(t *testing.T) {
	mgr := NewManager(&mockRuntime{})

	factory := func(name string, cfg config.ChainConfig, runtime container.Runtime, projectName string) (Chain, error) {
		return nil, fmt.Errorf("factory error")
	}
	mgr.RegisterFactory("failing", factory)

	cfg := config.ChainConfig{Engine: "failing", ChainID: 31337}

	_, err := mgr.CreateChain("test-chain", cfg, "my-project")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "factory error")
}

func TestManager_GetChain_Success(t *testing.T) {
	mgr := NewManager(&mockRuntime{})

	factory := func(name string, cfg config.ChainConfig, runtime container.Runtime, projectName string) (Chain, error) {
		return &mockChain{name: name, chainID: cfg.ChainID}, nil
	}
	mgr.RegisterFactory("mock", factory)

	cfg := config.ChainConfig{Engine: "mock", ChainID: 31337}
	_, err := mgr.CreateChain("test-chain", cfg, "my-project")
	require.NoError(t, err)

	c, err := mgr.GetChain("test-chain")
	require.NoError(t, err)
	assert.Equal(t, "test-chain", c.Name())
}

func TestManager_GetChain_NotFound(t *testing.T) {
	mgr := NewManager(&mockRuntime{})

	_, err := mgr.GetChain("nonexistent")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "not found")
}

func TestManager_AllChains(t *testing.T) {
	mgr := NewManager(&mockRuntime{})

	factory := func(name string, cfg config.ChainConfig, runtime container.Runtime, projectName string) (Chain, error) {
		return &mockChain{name: name, chainID: cfg.ChainID}, nil
	}
	mgr.RegisterFactory("mock", factory)

	chains := mgr.AllChains()
	assert.Len(t, chains, 0)

	cfg1 := config.ChainConfig{Engine: "mock", ChainID: 1}
	cfg2 := config.ChainConfig{Engine: "mock", ChainID: 2}

	_, err := mgr.CreateChain("chain1", cfg1, "proj")
	require.NoError(t, err)
	_, err = mgr.CreateChain("chain2", cfg2, "proj")
	require.NoError(t, err)

	chains = mgr.AllChains()
	assert.Len(t, chains, 2)

	names := make(map[string]bool)
	for _, c := range chains {
		names[c.Name()] = true
	}
	assert.True(t, names["chain1"])
	assert.True(t, names["chain2"])
}

func TestManager_RemoveChain_Success(t *testing.T) {
	mgr := NewManager(&mockRuntime{})

	factory := func(name string, cfg config.ChainConfig, runtime container.Runtime, projectName string) (Chain, error) {
		return &mockChain{name: name, chainID: cfg.ChainID}, nil
	}
	mgr.RegisterFactory("mock", factory)

	cfg := config.ChainConfig{Engine: "mock", ChainID: 31337}
	_, err := mgr.CreateChain("test-chain", cfg, "proj")
	require.NoError(t, err)

	ctx := context.Background()
	err = mgr.RemoveChain("test-chain", ctx)
	require.NoError(t, err)

	_, err = mgr.GetChain("test-chain")
	require.Error(t, err)
}

func TestManager_RemoveChain_NotFound(t *testing.T) {
	mgr := NewManager(&mockRuntime{})
	ctx := context.Background()

	err := mgr.RemoveChain("nonexistent", ctx)
	require.NoError(t, err)
}

func TestManager_RemoveChain_StopError(t *testing.T) {
	mgr := NewManager(&mockRuntime{})

	factory := func(name string, cfg config.ChainConfig, runtime container.Runtime, projectName string) (Chain, error) {
		return &mockChain{name: name, chainID: cfg.ChainID, stopErr: fmt.Errorf("stop failed")}, nil
	}
	mgr.RegisterFactory("mock", factory)

	cfg := config.ChainConfig{Engine: "mock", ChainID: 31337}
	_, err := mgr.CreateChain("test-chain", cfg, "proj")
	require.NoError(t, err)

	ctx := context.Background()
	err = mgr.RemoveChain("test-chain", ctx)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "failed to stop chain")
}

func TestManager_StopAll_Success(t *testing.T) {
	mgr := NewManager(&mockRuntime{})

	var createdChains []*mockChain
	factory := func(name string, cfg config.ChainConfig, runtime container.Runtime, projectName string) (Chain, error) {
		mc := &mockChain{name: name, chainID: cfg.ChainID}
		createdChains = append(createdChains, mc)
		return mc, nil
	}
	mgr.RegisterFactory("mock", factory)

	cfg := config.ChainConfig{Engine: "mock", ChainID: 1}
	_, _ = mgr.CreateChain("chain1", cfg, "proj")
	_, _ = mgr.CreateChain("chain2", cfg, "proj2")

	ctx := context.Background()
	err := mgr.StopAll(ctx)
	require.NoError(t, err)

	for _, mc := range createdChains {
		assert.True(t, mc.stopped, "chain %s should be stopped", mc.name)
	}
}

func TestManager_StopAll_WithError(t *testing.T) {
	mgr := NewManager(&mockRuntime{})

	factory := func(name string, cfg config.ChainConfig, runtime container.Runtime, projectName string) (Chain, error) {
		return &mockChain{name: name, chainID: cfg.ChainID, stopErr: fmt.Errorf("stop error for %s", name)}, nil
	}
	mgr.RegisterFactory("mock", factory)

	cfg := config.ChainConfig{Engine: "mock", ChainID: 1}
	_, _ = mgr.CreateChain("chain1", cfg, "proj")

	ctx := context.Background()
	err := mgr.StopAll(ctx)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "failed to stop chain")
}

func TestManager_StopAll_Empty(t *testing.T) {
	mgr := NewManager(&mockRuntime{})
	ctx := context.Background()
	err := mgr.StopAll(ctx)
	require.NoError(t, err)
}
