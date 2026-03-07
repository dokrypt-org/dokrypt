package network

import (
	"context"
	"fmt"
	"log/slog"

	"github.com/dokrypt/dokrypt/internal/container"
)

type Manager struct {
	runtime     container.Runtime
	projectName string
	networks    map[string]string // name -> network ID
}

func NewManager(runtime container.Runtime, projectName string) *Manager {
	return &Manager{
		runtime:     runtime,
		projectName: projectName,
		networks:    make(map[string]string),
	}
}

func (m *Manager) CreateEnvironmentNetwork(ctx context.Context) (string, error) {
	name := fmt.Sprintf("dokrypt-%s", m.projectName)
	slog.Info("creating environment network", "name", name)

	existingID, err := m.findExistingNetwork(ctx, name)
	if err != nil {
		return "", fmt.Errorf("failed to check for existing network %s: %w", name, err)
	}
	if existingID != "" {
		slog.Info("reusing existing network", "name", name)
		m.networks[name] = existingID
		return existingID, nil
	}

	id, err := m.runtime.CreateNetwork(ctx, name, container.NetworkOptions{
		Driver: "bridge",
		Labels: map[string]string{
			"dokrypt.project": m.projectName,
			"dokrypt.type":    "environment",
		},
	})
	if err != nil {
		return "", fmt.Errorf("failed to create environment network: %w", err)
	}

	m.networks[name] = id
	return id, nil
}

func (m *Manager) CreateChainNetwork(ctx context.Context, chainName string) (string, error) {
	name := fmt.Sprintf("dokrypt-%s-%s", m.projectName, chainName)
	slog.Info("creating chain network", "chain", chainName, "name", name)

	id, err := m.runtime.CreateNetwork(ctx, name, container.NetworkOptions{
		Driver: "bridge",
		Labels: map[string]string{
			"dokrypt.project": m.projectName,
			"dokrypt.chain":   chainName,
			"dokrypt.type":    "chain",
		},
	})
	if err != nil {
		return "", fmt.Errorf("failed to create chain network for %s: %w", chainName, err)
	}

	m.networks[name] = id
	return id, nil
}

func (m *Manager) CreateInterconnectNetwork(ctx context.Context) (string, error) {
	name := fmt.Sprintf("dokrypt-%s-interconnect", m.projectName)
	slog.Info("creating interconnect network", "name", name)

	id, err := m.runtime.CreateNetwork(ctx, name, container.NetworkOptions{
		Driver: "bridge",
		Labels: map[string]string{
			"dokrypt.project": m.projectName,
			"dokrypt.type":    "interconnect",
		},
	})
	if err != nil {
		return "", fmt.Errorf("failed to create interconnect network: %w", err)
	}

	m.networks[name] = id
	return id, nil
}

func (m *Manager) EnvironmentNetworkName() string {
	return fmt.Sprintf("dokrypt-%s", m.projectName)
}

func (m *Manager) ChainNetworkName(chainName string) string {
	return fmt.Sprintf("dokrypt-%s-%s", m.projectName, chainName)
}

func (m *Manager) RemoveAll(ctx context.Context) error {
	for name, id := range m.networks {
		slog.Info("removing network", "name", name)
		if err := m.runtime.RemoveNetwork(ctx, id); err != nil {
			slog.Warn("failed to remove network", "name", name, "error", err)
		}
		delete(m.networks, name)
	}
	return nil
}

func (m *Manager) NetworkID(name string) (string, bool) {
	id, ok := m.networks[name]
	return id, ok
}

func (m *Manager) findExistingNetwork(ctx context.Context, name string) (string, error) {
	networks, err := m.runtime.ListNetworks(ctx)
	if err != nil {
		slog.Error("failed to list Docker networks", "error", err)
		return "", fmt.Errorf("failed to list networks: %w", err)
	}
	for _, n := range networks {
		if n.Name == name {
			return n.ID, nil
		}
	}
	return "", nil
}
