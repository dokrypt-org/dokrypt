package engine

import (
	"context"
	"testing"
	"time"

	"github.com/dokrypt/dokrypt/internal/chain"
	"github.com/dokrypt/dokrypt/internal/config"
	"github.com/dokrypt/dokrypt/internal/network"
	"github.com/dokrypt/dokrypt/internal/service"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestState_Constants(t *testing.T) {
	assert.Equal(t, State("uninitialized"), StateUninitialized)
	assert.Equal(t, State("initialized"), StateInitialized)
	assert.Equal(t, State("starting"), StateStarting)
	assert.Equal(t, State("running"), StateRunning)
	assert.Equal(t, State("stopping"), StateStopping)
	assert.Equal(t, State("stopped"), StateStopped)
}

func TestNew_ReturnsEngineWithCorrectDefaults(t *testing.T) {
	reg := service.NewRegistry()
	eng := New(reg)

	require.NotNil(t, eng)
	assert.Equal(t, StateUninitialized, eng.state)
	assert.NotNil(t, eng.eventBus)
	assert.Same(t, reg, eng.svcRegistry)
	assert.Nil(t, eng.cfg)
	assert.Nil(t, eng.runtime)
	assert.Nil(t, eng.chainManager)
	assert.Nil(t, eng.orchestrator)
	assert.Nil(t, eng.networkMgr)
	assert.Nil(t, eng.pluginMgr)
	assert.Nil(t, eng.hooks)
}

func TestNew_NilRegistry(t *testing.T) {
	eng := New(nil)
	require.NotNil(t, eng)
	assert.Nil(t, eng.svcRegistry)
	assert.Equal(t, StateUninitialized, eng.state)
}

func TestNew_EventBusIsOperational(t *testing.T) {
	eng := New(service.NewRegistry())
	ch := eng.eventBus.Subscribe(EventChainStarted)

	eng.eventBus.Publish(Event{Type: EventChainStarted, Data: map[string]any{"ok": true}})

	select {
	case evt := <-ch:
		assert.Equal(t, EventChainStarted, evt.Type)
	case <-time.After(time.Second):
		t.Fatal("event not received")
	}
}

func TestEngine_Subscribe(t *testing.T) {
	eng := New(service.NewRegistry())
	ch := eng.Subscribe(EventEnvironmentUp)
	require.NotNil(t, ch)

	eng.eventBus.Publish(Event{Type: EventEnvironmentUp, Data: map[string]any{"project": "test"}})

	select {
	case evt := <-ch:
		assert.Equal(t, EventEnvironmentUp, evt.Type)
		assert.Equal(t, "test", evt.Data["project"])
	case <-time.After(time.Second):
		t.Fatal("event not received")
	}
}

func TestEngine_Config_Nil(t *testing.T) {
	eng := New(service.NewRegistry())
	assert.Nil(t, eng.Config())
}

func TestEngine_Config_ReturnsSet(t *testing.T) {
	eng := New(service.NewRegistry())
	cfg := &config.Config{Name: "myproject"}
	eng.cfg = cfg
	assert.Same(t, cfg, eng.Config())
}

func TestEngine_Runtime_Nil(t *testing.T) {
	eng := New(service.NewRegistry())
	assert.Nil(t, eng.Runtime())
}

func TestEngine_Up_WrongState_Uninitialized(t *testing.T) {
	eng := New(service.NewRegistry())
	err := eng.Up(context.Background(), UpOptions{})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "uninitialized")
}

func TestEngine_Up_WrongState_Running(t *testing.T) {
	eng := New(service.NewRegistry())
	eng.state = StateRunning
	err := eng.Up(context.Background(), UpOptions{})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "running")
}

func TestEngine_Up_WrongState_Starting(t *testing.T) {
	eng := New(service.NewRegistry())
	eng.state = StateStarting
	err := eng.Up(context.Background(), UpOptions{})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "starting")
}

func TestEngine_Up_WrongState_Stopping(t *testing.T) {
	eng := New(service.NewRegistry())
	eng.state = StateStopping
	err := eng.Up(context.Background(), UpOptions{})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "stopping")
}

func setupEngineForDown(t *testing.T) *Engine {
	t.Helper()
	eng := New(service.NewRegistry())
	eng.cfg = &config.Config{Name: "test-project"}
	eng.state = StateRunning
	eng.orchestrator = service.NewOrchestrator()
	eng.chainManager = chain.NewManager(nil)
	eng.networkMgr = network.NewManager(nil, "test-project")
	return eng
}

func TestEngine_Down_SetsStateStopped(t *testing.T) {
	eng := setupEngineForDown(t)
	ch := eng.Subscribe(EventEnvironmentDown)

	err := eng.Down(context.Background(), DownOptions{})
	require.NoError(t, err)
	assert.Equal(t, StateStopped, eng.state)

	select {
	case evt := <-ch:
		assert.Equal(t, EventEnvironmentDown, evt.Type)
		assert.Equal(t, "test-project", evt.Data["project"])
	case <-time.After(time.Second):
		t.Fatal("environment down event not received")
	}
}

func TestEngine_Down_FromNonRunningState(t *testing.T) {
	eng := New(service.NewRegistry())
	eng.cfg = &config.Config{Name: "test"}
	eng.state = StateInitialized
	eng.orchestrator = service.NewOrchestrator()
	eng.chainManager = chain.NewManager(nil)
	eng.networkMgr = network.NewManager(nil, "test")

	err := eng.Down(context.Background(), DownOptions{})
	require.NoError(t, err)
	assert.Equal(t, StateStopped, eng.state)
}

func TestEngine_Cleanup_NotRunning(t *testing.T) {
	eng := New(service.NewRegistry())
	eng.state = StateStopped

	err := eng.Cleanup(context.Background())
	require.NoError(t, err)
}

func TestEngine_Cleanup_RunningCallsDown(t *testing.T) {
	eng := setupEngineForDown(t)
	eng.cfg.Name = "cleanup-test"

	err := eng.Cleanup(context.Background())
	require.NoError(t, err)
	assert.Equal(t, StateStopped, eng.state)
}

func TestEngine_Cleanup_ClosesEventBus(t *testing.T) {
	eng := New(service.NewRegistry())
	ch := eng.Subscribe(EventChainStarted)

	eng.state = StateStopped
	err := eng.Cleanup(context.Background())
	require.NoError(t, err)

	_, ok := <-ch
	assert.False(t, ok, "event channel should be closed after Cleanup")
}

func TestEngine_Chain_NotFound(t *testing.T) {
	eng := New(service.NewRegistry())
	eng.chainManager = chain.NewManager(nil)

	_, err := eng.Chain("nonexistent")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "not found")
}

func TestEngine_Chains_Empty(t *testing.T) {
	eng := New(service.NewRegistry())
	eng.chainManager = chain.NewManager(nil)

	chains := eng.Chains()
	assert.Empty(t, chains)
}

func TestEngine_Service_NotFound(t *testing.T) {
	eng := New(service.NewRegistry())
	eng.orchestrator = service.NewOrchestrator()

	_, err := eng.Service("nonexistent")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "not found")
}

func TestEngine_Services_Empty(t *testing.T) {
	eng := New(service.NewRegistry())
	eng.orchestrator = service.NewOrchestrator()

	svcs := eng.Services()
	assert.Empty(t, svcs)
}

func TestEngine_ChainContainerID_NotFound(t *testing.T) {
	eng := New(service.NewRegistry())
	eng.chainManager = chain.NewManager(nil)

	id := eng.ChainContainerID("nonexistent")
	assert.Empty(t, id)
}

func TestEngine_ServiceContainerID_NotFound(t *testing.T) {
	eng := New(service.NewRegistry())
	eng.orchestrator = service.NewOrchestrator()

	id := eng.ServiceContainerID("nonexistent")
	assert.Empty(t, id)
}

func TestEngine_DispatchPluginHook_NilHooks(t *testing.T) {
	eng := New(service.NewRegistry())
	assert.Nil(t, eng.hooks)

	assert.NotPanics(t, func() {
		eng.DispatchPluginHook(context.Background(), "test_hook", map[string]any{"key": "val"})
	})
}

func TestEngine_StartPluginEventBridge_NilHooks(t *testing.T) {
	eng := New(service.NewRegistry())
	assert.Nil(t, eng.hooks)

	assert.NotPanics(t, func() {
		eng.startPluginEventBridge(context.Background())
	})
}

func TestUpOptions_Defaults(t *testing.T) {
	opts := UpOptions{}
	assert.False(t, opts.Detach)
	assert.False(t, opts.Build)
	assert.Empty(t, opts.Services)
	assert.False(t, opts.Fresh)
	assert.Empty(t, opts.Fork)
	assert.Zero(t, opts.ForkBlock)
	assert.Empty(t, opts.Snapshot)
	assert.Empty(t, opts.Profile)
	assert.Zero(t, opts.Timeout)
}

func TestUpOptions_WithValues(t *testing.T) {
	opts := UpOptions{
		Detach:    true,
		Build:     true,
		Services:  []string{"ipfs", "indexer"},
		Fresh:     true,
		Fork:      "mainnet",
		ForkBlock: 12345678,
		Snapshot:  "snap-01",
		Profile:   "test",
		Timeout:   30 * time.Second,
	}
	assert.True(t, opts.Detach)
	assert.True(t, opts.Build)
	assert.Equal(t, []string{"ipfs", "indexer"}, opts.Services)
	assert.True(t, opts.Fresh)
	assert.Equal(t, "mainnet", opts.Fork)
	assert.Equal(t, uint64(12345678), opts.ForkBlock)
	assert.Equal(t, "snap-01", opts.Snapshot)
	assert.Equal(t, "test", opts.Profile)
	assert.Equal(t, 30*time.Second, opts.Timeout)
}

func TestDownOptions_Defaults(t *testing.T) {
	opts := DownOptions{}
	assert.False(t, opts.RemoveVolumes)
	assert.Empty(t, opts.Services)
	assert.Zero(t, opts.Timeout)
}

func TestDownOptions_WithValues(t *testing.T) {
	opts := DownOptions{
		RemoveVolumes: true,
		Services:      []string{"ipfs"},
		Timeout:       10 * time.Second,
	}
	assert.True(t, opts.RemoveVolumes)
	assert.Equal(t, []string{"ipfs"}, opts.Services)
	assert.Equal(t, 10*time.Second, opts.Timeout)
}

func TestEnvironmentStatus_Fields(t *testing.T) {
	status := EnvironmentStatus{
		Name:  "myproject",
		State: StateRunning,
		Chains: []ChainStatus{
			{
				Name:    "local",
				Engine:  "anvil",
				ChainID: 31337,
				RPCURL:  "http://localhost:8545",
				WSURL:   "ws://localhost:8546",
				Running: true,
				Healthy: true,
			},
		},
		Services: []service.ServiceStatus{
			{
				Name:    "ipfs",
				Type:    "ipfs",
				Status:  "running",
				Healthy: true,
			},
		},
	}

	assert.Equal(t, "myproject", status.Name)
	assert.Equal(t, StateRunning, status.State)
	require.Len(t, status.Chains, 1)
	assert.Equal(t, "local", status.Chains[0].Name)
	assert.Equal(t, "anvil", status.Chains[0].Engine)
	assert.Equal(t, uint64(31337), status.Chains[0].ChainID)
	assert.Equal(t, "http://localhost:8545", status.Chains[0].RPCURL)
	assert.Equal(t, "ws://localhost:8546", status.Chains[0].WSURL)
	assert.True(t, status.Chains[0].Running)
	assert.True(t, status.Chains[0].Healthy)
	require.Len(t, status.Services, 1)
	assert.Equal(t, "ipfs", status.Services[0].Name)
}

func TestChainStatus_ZeroValue(t *testing.T) {
	cs := ChainStatus{}
	assert.Empty(t, cs.Name)
	assert.Empty(t, cs.Engine)
	assert.Zero(t, cs.ChainID)
	assert.Empty(t, cs.RPCURL)
	assert.Empty(t, cs.WSURL)
	assert.False(t, cs.Running)
	assert.False(t, cs.Healthy)
}

func TestEngineEnv_ProjectName_NilConfig(t *testing.T) {
	eng := New(service.NewRegistry())
	env := &engineEnv{engine: eng}
	assert.Empty(t, env.ProjectName())
}

func TestEngineEnv_ProjectName_WithConfig(t *testing.T) {
	eng := New(service.NewRegistry())
	eng.cfg = &config.Config{Name: "my-project"}
	env := &engineEnv{engine: eng}
	assert.Equal(t, "my-project", env.ProjectName())
}

func TestEngineEnv_ChainRPCURL_NotFound(t *testing.T) {
	eng := New(service.NewRegistry())
	eng.chainManager = chain.NewManager(nil)
	env := &engineEnv{engine: eng}
	assert.Empty(t, env.ChainRPCURL("nonexistent"))
}

func TestEngineEnv_ServiceURL_NotFound(t *testing.T) {
	eng := New(service.NewRegistry())
	eng.orchestrator = service.NewOrchestrator()
	env := &engineEnv{engine: eng}
	assert.Empty(t, env.ServiceURL("nonexistent"))
}
