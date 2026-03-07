package network

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"net"
	"os"
	"path/filepath"
	"sync"

	"github.com/dokrypt/dokrypt/internal/container"
)

const (
	defaultChainSubnetBase = "10.100.0.0"
	defaultChainSubnetMask = 24

	defaultInterconnectSubnet = "10.200.0.0/16"
)

type subnetAllocations struct {
	Counter int               `json:"counter"`
	Chains  map[string]string `json:"chains"` // chain name -> subnet CIDR
}

type NetworkTopology struct {
	ChainNetworks  map[string]string // chain name -> network ID
	InterconnectID string
	SubnetCounter  int
}

type MultiChainNetwork struct {
	runtime       container.Runtime
	projectName   string
	topology      *NetworkTopology
	mu            sync.Mutex
	allocations   *subnetAllocations
	allocFilePath string

	ChainSubnetBase      string // Base IP for chain subnets (default "10.100.0.0")
	ChainSubnetMask      int    // CIDR mask for each chain subnet (default 24)
	InterconnectSubnet   string // CIDR for the interconnect network (default "10.200.0.0/16")
}

func NewMultiChainNetwork(runtime container.Runtime, projectName string) *MultiChainNetwork {
	m := &MultiChainNetwork{
		runtime:            runtime,
		projectName:        projectName,
		ChainSubnetBase:    defaultChainSubnetBase,
		ChainSubnetMask:    defaultChainSubnetMask,
		InterconnectSubnet: defaultInterconnectSubnet,
	}
	m.allocFilePath = m.defaultAllocFilePath()
	m.loadAllocations()
	return m
}

func (m *MultiChainNetwork) defaultAllocFilePath() string {
	home, err := os.UserHomeDir()
	if err != nil {
		return filepath.Join(os.TempDir(), "dokrypt-subnets-"+m.projectName+".json")
	}
	return filepath.Join(home, ".dokrypt", "state", m.projectName, "subnet-allocations.json")
}

func (m *MultiChainNetwork) SetAllocFilePath(path string) {
	m.allocFilePath = path
	m.loadAllocations()
}

func (m *MultiChainNetwork) loadAllocations() {
	m.allocations = &subnetAllocations{
		Chains: make(map[string]string),
	}
	data, err := os.ReadFile(m.allocFilePath)
	if err != nil {
		return
	}
	var alloc subnetAllocations
	if err := json.Unmarshal(data, &alloc); err != nil {
		slog.Warn("failed to parse subnet allocations file, starting fresh",
			"path", m.allocFilePath, "error", err)
		return
	}
	if alloc.Chains == nil {
		alloc.Chains = make(map[string]string)
	}
	m.allocations = &alloc
	slog.Debug("loaded subnet allocations", "counter", alloc.Counter, "chains", len(alloc.Chains))
}

func (m *MultiChainNetwork) saveAllocations() error {
	dir := filepath.Dir(m.allocFilePath)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return fmt.Errorf("failed to create allocation directory %s: %w", dir, err)
	}
	data, err := json.MarshalIndent(m.allocations, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal subnet allocations: %w", err)
	}
	if err := os.WriteFile(m.allocFilePath, data, 0o644); err != nil {
		return fmt.Errorf("failed to write subnet allocations to %s: %w", m.allocFilePath, err)
	}
	return nil
}

func (m *MultiChainNetwork) Setup(ctx context.Context, chains []string) (*NetworkTopology, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	topology := &NetworkTopology{
		ChainNetworks: make(map[string]string),
	}

	usedSubnets, err := m.listUsedSubnets(ctx)
	if err != nil {
		slog.Warn("could not list existing Docker networks for conflict detection", "error", err)
	}

	for _, chainName := range chains {
		subnet, err := m.AllocateSubnet(chainName, usedSubnets)
		if err != nil {
			return nil, fmt.Errorf("failed to allocate subnet for chain %s: %w", chainName, err)
		}
		name := fmt.Sprintf("dokrypt-%s-%s", m.projectName, chainName)
		slog.Info("creating chain network", "chain", chainName, "subnet", subnet)

		id, err := m.runtime.CreateNetwork(ctx, name, container.NetworkOptions{
			Driver: "bridge",
			Subnet: subnet,
			Labels: map[string]string{
				"dokrypt.project": m.projectName,
				"dokrypt.chain":   chainName,
				"dokrypt.type":    "chain",
				"dokrypt.network": "true",
			},
		})
		if err != nil {
			return nil, fmt.Errorf("failed to create network for chain %s: %w", chainName, err)
		}
		topology.ChainNetworks[chainName] = id
		usedSubnets = append(usedSubnets, subnet)
	}

	if err := m.saveAllocations(); err != nil {
		slog.Warn("failed to persist subnet allocations", "error", err)
	}

	if len(chains) > 1 {
		interconnectSubnet := m.InterconnectSubnet
		if subnetsOverlap(interconnectSubnet, usedSubnets) {
			return nil, fmt.Errorf("interconnect subnet %s conflicts with an existing Docker network; "+
				"set a different InterconnectSubnet on MultiChainNetwork", interconnectSubnet)
		}

		name := fmt.Sprintf("dokrypt-%s-interconnect", m.projectName)
		slog.Info("creating interconnect network", "name", name, "subnet", interconnectSubnet)

		id, err := m.runtime.CreateNetwork(ctx, name, container.NetworkOptions{
			Driver: "bridge",
			Subnet: interconnectSubnet,
			Labels: map[string]string{
				"dokrypt.project": m.projectName,
				"dokrypt.type":    "interconnect",
				"dokrypt.network": "true",
			},
		})
		if err != nil {
			return nil, fmt.Errorf("failed to create interconnect network: %w", err)
		}
		topology.InterconnectID = id
	}

	m.topology = topology
	return topology, nil
}

func (m *MultiChainNetwork) Teardown(ctx context.Context) error {
	if m.topology == nil {
		return nil
	}

	for chainName, id := range m.topology.ChainNetworks {
		slog.Info("removing chain network", "chain", chainName)
		if err := m.runtime.RemoveNetwork(ctx, id); err != nil {
			slog.Warn("failed to remove chain network", "chain", chainName, "error", err)
		}
	}

	if m.topology.InterconnectID != "" {
		slog.Info("removing interconnect network")
		if err := m.runtime.RemoveNetwork(ctx, m.topology.InterconnectID); err != nil {
			slog.Warn("failed to remove interconnect network", "error", err)
		}
	}

	m.topology = nil
	return nil
}

func (m *MultiChainNetwork) AllocateSubnet(chainName string, usedSubnets []string) (string, error) {
	if existing, ok := m.allocations.Chains[chainName]; ok {
		if !subnetsOverlap(existing, usedSubnets) {
			slog.Debug("reusing persisted subnet allocation", "chain", chainName, "subnet", existing)
			return existing, nil
		}
		slog.Warn("persisted subnet conflicts with existing network, reallocating",
			"chain", chainName, "subnet", existing)
		delete(m.allocations.Chains, chainName)
	}

	baseIP := net.ParseIP(m.ChainSubnetBase).To4()
	if baseIP == nil {
		return "", fmt.Errorf("invalid chain subnet base: %s", m.ChainSubnetBase)
	}
	mask := m.ChainSubnetMask

	const maxAttempts = 65536
	for i := 0; i < maxAttempts; i++ {
		m.allocations.Counter++
		counter := m.allocations.Counter

		secondOctet := int(baseIP[1]) + (counter >> 8)
		thirdOctet := int(baseIP[2]) + (counter & 0xFF)

		if secondOctet > 255 {
			return "", fmt.Errorf("subnet allocation exhausted: second octet overflow (counter=%d)", counter)
		}

		candidate := fmt.Sprintf("%d.%d.%d.0/%d", baseIP[0], secondOctet, thirdOctet, mask)

		if !subnetsOverlap(candidate, usedSubnets) {
			m.allocations.Chains[chainName] = candidate
			return candidate, nil
		}

		slog.Debug("subnet candidate conflicts, trying next", "candidate", candidate)
	}

	return "", fmt.Errorf("failed to allocate a non-conflicting subnet after %d attempts", maxAttempts)
}

func (m *MultiChainNetwork) ConnectToInterconnect(ctx context.Context, containerID string) error {
	if m.topology == nil || m.topology.InterconnectID == "" {
		return fmt.Errorf("no interconnect network configured")
	}
	return m.runtime.ConnectNetwork(ctx, m.topology.InterconnectID, containerID)
}

func (m *MultiChainNetwork) Topology() *NetworkTopology {
	return m.topology
}

func (m *MultiChainNetwork) listUsedSubnets(ctx context.Context) ([]string, error) {
	networks, err := m.runtime.ListNetworks(ctx)
	if err != nil {
		return nil, err
	}
	var subnets []string
	for _, n := range networks {
		if n.Subnet != "" {
			subnets = append(subnets, n.Subnet)
		}
	}
	return subnets, nil
}

func subnetsOverlap(candidate string, existing []string) bool {
	_, candidateNet, err := net.ParseCIDR(candidate)
	if err != nil {
		return false
	}
	for _, s := range existing {
		_, existingNet, err := net.ParseCIDR(s)
		if err != nil {
			continue
		}
		if candidateNet.Contains(existingNet.IP) || existingNet.Contains(candidateNet.IP) {
			return true
		}
	}
	return false
}
