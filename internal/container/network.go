package container

import (
	"context"
	"fmt"
	"log/slog"
)

type NetworkDetail struct {
	NetworkInfo
	Containers []string // Container IDs connected to this network
	Subnet     string
	Gateway    string
}

type NetworkCreateOptions struct {
	NetworkOptions
	Aliases []string
}

type NetworkManager struct {
	runtime Runtime
}

func NewNetworkManager(rt Runtime) *NetworkManager {
	return &NetworkManager{runtime: rt}
}

func (m *NetworkManager) Create(ctx context.Context, name string, opts NetworkCreateOptions) (string, error) {
	networks, err := m.runtime.ListNetworks(ctx)
	if err == nil {
		for _, n := range networks {
			if n.Name == name {
				slog.Debug("network already exists", "name", name)
				return n.ID, nil
			}
		}
	}

	if opts.Labels == nil {
		opts.Labels = make(map[string]string)
	}
	opts.Labels["dokrypt.network"] = "true"

	slog.Info("creating network", "name", name, "driver", opts.Driver)
	id, err := m.runtime.CreateNetwork(ctx, name, opts.NetworkOptions)
	if err != nil {
		return "", fmt.Errorf("failed to create network %s: %w", name, err)
	}
	return id, nil
}

func (m *NetworkManager) Connect(ctx context.Context, networkID, containerID string) error {
	return m.runtime.ConnectNetwork(ctx, networkID, containerID)
}

func (m *NetworkManager) Disconnect(ctx context.Context, networkID, containerID string) error {
	return m.runtime.DisconnectNetwork(ctx, networkID, containerID)
}

func (m *NetworkManager) Remove(ctx context.Context, networkID string, force bool) error {
	return m.runtime.RemoveNetwork(ctx, networkID)
}

func (m *NetworkManager) List(ctx context.Context) ([]NetworkInfo, error) {
	networks, err := m.runtime.ListNetworks(ctx)
	if err != nil {
		return nil, err
	}
	var result []NetworkInfo
	for _, n := range networks {
		if n.Labels["dokrypt.network"] == "true" {
			result = append(result, n)
		}
	}
	return result, nil
}

func (m *NetworkManager) CreateInterconnect(ctx context.Context, projectName string) (string, error) {
	name := fmt.Sprintf("dokrypt-%s-interconnect", projectName)
	return m.Create(ctx, name, NetworkCreateOptions{
		NetworkOptions: NetworkOptions{
			Driver: "bridge",
			Labels: map[string]string{
				"dokrypt.project": projectName,
				"dokrypt.type":    "interconnect",
				"dokrypt.network": "true",
			},
		},
	})
}
