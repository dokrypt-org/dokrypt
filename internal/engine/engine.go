package engine

import (
	"context"
	"fmt"
	"log/slog"
	"os"
	"time"

	"github.com/dokrypt/dokrypt/internal/chain"
	"github.com/dokrypt/dokrypt/internal/chain/evm"
	"github.com/dokrypt/dokrypt/internal/config"
	"github.com/dokrypt/dokrypt/internal/container"
	"github.com/dokrypt/dokrypt/internal/network"
	"github.com/dokrypt/dokrypt/internal/plugin"
	"github.com/dokrypt/dokrypt/internal/service"
)

type State string

const (
	StateUninitialized State = "uninitialized"
	StateInitialized   State = "initialized"
	StateStarting      State = "starting"
	StateRunning       State = "running"
	StateStopping      State = "stopping"
	StateStopped       State = "stopped"
)

type Engine struct {
	cfg           *config.Config
	runtime       container.Runtime
	chainManager  *chain.Manager
	orchestrator  *service.Orchestrator
	networkMgr    *network.Manager
	svcRegistry   *service.Registry
	eventBus      *EventBus
	state         State
	pluginMgr     *plugin.Manager
	hooks         *plugin.HookDispatcher
}

type UpOptions struct {
	Detach     bool
	Build      bool
	Services   []string // Specific services to start; empty = all
	Fresh      bool     // Destroy existing state first
	Fork       string   // Fork network name or URL
	ForkBlock  uint64
	Snapshot   string   // Start from snapshot
	Profile    string
	Timeout    time.Duration
}

type DownOptions struct {
	RemoveVolumes bool
	Services      []string
	Timeout       time.Duration
}

type EnvironmentStatus struct {
	Name     string                  `json:"name"`
	State    State                   `json:"state"`
	Chains   []ChainStatus           `json:"chains"`
	Services []service.ServiceStatus `json:"services"`
}

type ChainStatus struct {
	Name    string `json:"name"`
	Engine  string `json:"engine"`
	ChainID uint64 `json:"chain_id"`
	RPCURL  string `json:"rpc_url"`
	WSURL   string `json:"ws_url"`
	Running bool   `json:"running"`
	Healthy bool   `json:"healthy"`
}

func New(svcRegistry *service.Registry) *Engine {
	return &Engine{
		svcRegistry: svcRegistry,
		eventBus:    NewEventBus(),
		state:       StateUninitialized,
	}
}

func (e *Engine) Init(ctx context.Context, cfg *config.Config) error {
	slog.Info("initializing engine", "project", cfg.Name)

	if err := config.Validate(cfg); err != nil {
		return fmt.Errorf("config validation failed: %w", err)
	}

	runtime, err := container.NewRuntime(cfg.Settings.Runtime)
	if err != nil {
		return fmt.Errorf("failed to create container runtime: %w", err)
	}

	if err := runtime.Ping(ctx); err != nil {
		return fmt.Errorf("container runtime not available: %w", err)
	}

	e.cfg = cfg
	e.runtime = runtime
	e.chainManager = chain.NewManager(runtime)
	e.orchestrator = service.NewOrchestrator()
	e.networkMgr = network.NewManager(runtime, cfg.Name)

	e.chainManager.RegisterFactory("anvil", wrapChainFactory(evm.NewAnvilChain))
	e.chainManager.RegisterFactory("hardhat", wrapChainFactory(evm.NewHardhatChain))
	e.chainManager.RegisterFactory("geth", wrapChainFactory(evm.NewGethChain))

	e.initPlugins()

	e.state = StateInitialized

	return nil
}

func (e *Engine) Up(ctx context.Context, opts UpOptions) error {
	if e.state != StateInitialized && e.state != StateStopped {
		return fmt.Errorf("engine is in state %s, expected initialized or stopped", e.state)
	}

	e.state = StateStarting
	slog.Info("starting environment", "project", e.cfg.Name)

	slog.Info("creating networks")
	if _, err := e.networkMgr.CreateEnvironmentNetwork(ctx); err != nil {
		e.state = StateStopped
		return fmt.Errorf("failed to create environment network: %w", err)
	}

	if len(e.cfg.Chains) > 1 {
		for chainName := range e.cfg.Chains {
			if _, err := e.networkMgr.CreateChainNetwork(ctx, chainName); err != nil {
				slog.Warn("failed to create chain network", "chain", chainName, "error", err)
			}
		}
		if _, err := e.networkMgr.CreateInterconnectNetwork(ctx); err != nil {
			slog.Warn("failed to create interconnect network", "error", err)
		}
	}

	slog.Info("starting chains", "count", len(e.cfg.Chains))
	for name, chainCfg := range e.cfg.Chains {
		c, err := e.chainManager.CreateChain(name, chainCfg, e.cfg.Name)
		if err != nil {
			e.state = StateStopped
			return fmt.Errorf("failed to create chain %s: %w", name, err)
		}
		if err := c.Start(ctx); err != nil {
			e.state = StateStopped
			return fmt.Errorf("failed to start chain %s: %w", name, err)
		}
		e.eventBus.Publish(Event{Type: EventChainStarted, Data: map[string]any{"chain": name}})
	}

	if len(e.cfg.Services) > 0 && e.svcRegistry != nil {
		slog.Info("creating services", "count", len(e.cfg.Services))
		for name, svcCfg := range e.cfg.Services {
			if !e.svcRegistry.HasType(svcCfg.Type) {
				slog.Warn("unknown service type, skipping", "service", name, "type", svcCfg.Type)
				continue
			}
			svc, err := e.svcRegistry.Create(name, svcCfg, e.runtime, e.cfg.Name)
			if err != nil {
				slog.Warn("failed to create service, skipping", "service", name, "error", err)
				continue
			}
			e.orchestrator.Register(svc)
		}

		if err := e.orchestrator.StartAll(ctx); err != nil {
			slog.Warn("some services failed to start", "error", err)
		}
	}

	if e.hooks != nil {
		e.hooks.DispatchHook(ctx, plugin.HookOnUp, &engineEnv{engine: e})
	}

	e.state = StateRunning
	e.eventBus.Publish(Event{Type: EventEnvironmentUp, Data: map[string]any{"project": e.cfg.Name}})

	e.startPluginEventBridge(ctx)

	slog.Info("environment is up", "project", e.cfg.Name)
	return nil
}

func (e *Engine) Down(ctx context.Context, opts DownOptions) error {
	if e.state != StateRunning {
		slog.Warn("stopping environment from state", "state", e.state)
	}

	e.state = StateStopping
	slog.Info("stopping environment", "project", e.cfg.Name)

	if e.hooks != nil {
		e.hooks.DispatchHook(ctx, plugin.HookOnDown, &engineEnv{engine: e})
	}

	if err := e.orchestrator.StopAll(ctx); err != nil {
		slog.Warn("error stopping services", "error", err)
	}

	for _, c := range e.chainManager.AllChains() {
		if err := c.Stop(ctx); err != nil {
			slog.Warn("failed to stop chain", "chain", c.Name(), "error", err)
		}
		e.eventBus.Publish(Event{Type: EventChainStopped, Data: map[string]any{"chain": c.Name()}})
	}

	if err := e.networkMgr.RemoveAll(ctx); err != nil {
		slog.Warn("error removing networks", "error", err)
	}

	e.state = StateStopped
	e.eventBus.Publish(Event{Type: EventEnvironmentDown, Data: map[string]any{"project": e.cfg.Name}})
	slog.Info("environment is down", "project", e.cfg.Name)
	return nil
}

func (e *Engine) Status(ctx context.Context) *EnvironmentStatus {
	status := &EnvironmentStatus{
		Name:  e.cfg.Name,
		State: e.state,
	}

	for _, c := range e.chainManager.AllChains() {
		healthy := c.Health(ctx) == nil
		status.Chains = append(status.Chains, ChainStatus{
			Name:    c.Name(),
			Engine:  c.Engine(),
			ChainID: c.ChainID(),
			RPCURL:  c.RPCURL(),
			WSURL:   c.WSURL(),
			Running: c.IsRunning(ctx),
			Healthy: healthy,
		})
	}

	status.Services = e.orchestrator.Status(ctx)

	return status
}

func (e *Engine) Chain(name string) (chain.Chain, error) {
	return e.chainManager.GetChain(name)
}

func (e *Engine) Chains() []chain.Chain {
	return e.chainManager.AllChains()
}

func (e *Engine) Service(name string) (service.Service, error) {
	return e.orchestrator.Get(name)
}

func (e *Engine) Services() []service.Service {
	return e.orchestrator.All()
}

func (e *Engine) Subscribe(eventType EventType) <-chan Event {
	return e.eventBus.Subscribe(eventType)
}

func (e *Engine) Config() *config.Config {
	return e.cfg
}

func (e *Engine) Runtime() container.Runtime {
	return e.runtime
}

func (e *Engine) Cleanup(ctx context.Context) error {
	e.eventBus.Close()
	if e.state == StateRunning {
		return e.Down(ctx, DownOptions{})
	}
	return nil
}

type containerIDer interface {
	ContainerID() string
}

type baseIDer interface {
	GetContainerID() string
}

func (e *Engine) ChainContainerID(name string) string {
	c, err := e.chainManager.GetChain(name)
	if err != nil {
		return ""
	}
	if cid, ok := c.(containerIDer); ok {
		return cid.ContainerID()
	}
	return ""
}

func (e *Engine) ServiceContainerID(name string) string {
	svc, err := e.orchestrator.Get(name)
	if err != nil {
		return ""
	}
	if cid, ok := svc.(containerIDer); ok {
		return cid.ContainerID()
	}
	if cid, ok := svc.(baseIDer); ok {
		return cid.GetContainerID()
	}
	return ""
}

func (e *Engine) initPlugins() {
	projectDir, err := os.Getwd()
	if err != nil {
		slog.Warn("plugin init: cannot determine working directory", "error", err)
		return
	}

	mgr, err := plugin.DefaultManager(projectDir)
	if err != nil {
		slog.Warn("plugin init: failed to create manager", "error", err)
		return
	}

	if err := mgr.Discover(); err != nil {
		slog.Warn("plugin init: discovery failed", "error", err)
		return
	}

	loader := plugin.NewLoader(mgr)
	if _, err := loader.LoadAll(); err != nil {
		slog.Warn("plugin init: loading failed", "error", err)
		return
	}

	e.pluginMgr = mgr
	e.hooks = plugin.NewHookDispatcher(mgr)

	e.hooks.DispatchHook(context.Background(), plugin.HookOnInit, &engineEnv{engine: e})

	loaded := mgr.List()
	for _, info := range loaded {
		e.eventBus.Publish(Event{Type: EventPluginLoaded, Data: map[string]any{"plugin": info.Manifest.Name}})
	}

	slog.Info("plugins initialized", "count", len(loaded))
}

type engineEnv struct {
	engine *Engine
}

func (env *engineEnv) ProjectName() string {
	if env.engine.cfg != nil {
		return env.engine.cfg.Name
	}
	return ""
}

func (env *engineEnv) ChainRPCURL(name string) string {
	c, err := env.engine.chainManager.GetChain(name)
	if err != nil {
		return ""
	}
	return c.RPCURL()
}

func (env *engineEnv) ServiceURL(name string) string {
	svc, err := env.engine.orchestrator.Get(name)
	if err != nil {
		return ""
	}
	urls := svc.URLs()
	if u, ok := urls["http"]; ok {
		return u
	}
	if u, ok := urls["api"]; ok {
		return u
	}
	return ""
}

func (e *Engine) startPluginEventBridge(ctx context.Context) {
	if e.hooks == nil {
		return
	}

	env := &engineEnv{engine: e}

	eventMap := map[EventType]plugin.HookType{
		EventChainBlockMined: plugin.HookOnBlockMined,
		EventTestPassed:      plugin.HookOnTestEnd,
		EventTestFailed:      plugin.HookOnTestEnd,
	}

	for eventType, hookType := range eventMap {
		ch := e.eventBus.Subscribe(eventType)
		hook := hookType
		go func() {
			for {
				select {
				case <-ctx.Done():
					return
				case evt, ok := <-ch:
					if !ok {
						return
					}
					e.hooks.Dispatch(ctx, plugin.HookEvent{
						Hook: hook,
						Env:  env,
						Data: evt.Data,
					})
				}
			}
		}()
	}
}

func (e *Engine) DispatchPluginHook(ctx context.Context, hook plugin.HookType, data map[string]any) {
	if e.hooks == nil {
		return
	}
	e.hooks.Dispatch(ctx, plugin.HookEvent{
		Hook: hook,
		Env:  &engineEnv{engine: e},
		Data: data,
	})
}

func wrapChainFactory[T chain.Chain](fn func(string, config.ChainConfig, container.Runtime, string) (T, error)) chain.ChainFactory {
	return func(name string, cfg config.ChainConfig, runtime container.Runtime, projectName string) (chain.Chain, error) {
		return fn(name, cfg, runtime, projectName)
	}
}
