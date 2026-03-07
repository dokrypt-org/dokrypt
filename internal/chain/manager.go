package chain

import (
	"context"
	"fmt"
	"sync"

	"github.com/dokrypt/dokrypt/internal/config"
	"github.com/dokrypt/dokrypt/internal/container"
)

type ChainFactory func(name string, cfg config.ChainConfig, runtime container.Runtime, projectName string) (Chain, error)

type Manager struct {
	runtime   container.Runtime
	chains    map[string]Chain
	factories map[string]ChainFactory
	mu        sync.RWMutex
}

func NewManager(runtime container.Runtime) *Manager {
	return &Manager{
		runtime:   runtime,
		chains:    make(map[string]Chain),
		factories: make(map[string]ChainFactory),
	}
}

func (m *Manager) RegisterFactory(engine string, factory ChainFactory) {
	m.factories[engine] = factory
}

func (m *Manager) CreateChain(name string, cfg config.ChainConfig, projectName string) (Chain, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	if _, exists := m.chains[name]; exists {
		return nil, fmt.Errorf("chain %q already exists", name)
	}

	factory, ok := m.factories[cfg.Engine]
	if !ok {
		return nil, fmt.Errorf("unsupported chain engine: %s", cfg.Engine)
	}

	c, err := factory(name, cfg, m.runtime, projectName)
	if err != nil {
		return nil, err
	}

	m.chains[name] = c
	return c, nil
}

func (m *Manager) GetChain(name string) (Chain, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	c, ok := m.chains[name]
	if !ok {
		return nil, fmt.Errorf("chain %q not found", name)
	}
	return c, nil
}

func (m *Manager) AllChains() []Chain {
	m.mu.RLock()
	defer m.mu.RUnlock()

	chains := make([]Chain, 0, len(m.chains))
	for _, c := range m.chains {
		chains = append(chains, c)
	}
	return chains
}

func (m *Manager) RemoveChain(name string, ctx context.Context) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	c, ok := m.chains[name]
	if !ok {
		return nil
	}
	if err := c.Stop(ctx); err != nil {
		return fmt.Errorf("failed to stop chain %s: %w", name, err)
	}
	delete(m.chains, name)
	return nil
}

func (m *Manager) StopAll(ctx context.Context) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	var lastErr error
	for name, c := range m.chains {
		if err := c.Stop(ctx); err != nil {
			lastErr = fmt.Errorf("failed to stop chain %s: %w", name, err)
		}
	}
	return lastErr
}
