package network

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"testing"

	"github.com/dokrypt/dokrypt/internal/container"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func newTestMultiChain(t *testing.T, rt *mockRuntime) *MultiChainNetwork {
	t.Helper()
	mcn := &MultiChainNetwork{
		runtime:            rt,
		projectName:        "testproj",
		ChainSubnetBase:    defaultChainSubnetBase,
		ChainSubnetMask:    defaultChainSubnetMask,
		InterconnectSubnet: defaultInterconnectSubnet,
	}
	tmpFile := filepath.Join(t.TempDir(), "subnet-allocations.json")
	mcn.allocFilePath = tmpFile
	mcn.loadAllocations()
	return mcn
}

func TestNewMultiChainNetwork(t *testing.T) {
	rt := newMockRuntime()
	mcn := NewMultiChainNetwork(rt, "proj1")
	assert.NotNil(t, mcn)
	assert.Equal(t, "proj1", mcn.projectName)
	assert.Equal(t, defaultChainSubnetBase, mcn.ChainSubnetBase)
	assert.Equal(t, defaultChainSubnetMask, mcn.ChainSubnetMask)
	assert.Equal(t, defaultInterconnectSubnet, mcn.InterconnectSubnet)
	assert.NotNil(t, mcn.allocations)
	assert.Nil(t, mcn.Topology())
}

func TestMultiChainNetwork_SetAllocFilePath(t *testing.T) {
	rt := newMockRuntime()
	mcn := newTestMultiChain(t, rt)

	tmpFile := filepath.Join(t.TempDir(), "alloc.json")
	alloc := subnetAllocations{Counter: 42, Chains: map[string]string{"eth": "10.100.0.1/24"}}
	data, _ := json.Marshal(alloc)
	require.NoError(t, os.WriteFile(tmpFile, data, 0o644))

	mcn.SetAllocFilePath(tmpFile)
	assert.Equal(t, 42, mcn.allocations.Counter)
	assert.Equal(t, "10.100.0.1/24", mcn.allocations.Chains["eth"])
}

func TestMultiChainNetwork_SetAllocFilePath_InvalidJSON(t *testing.T) {
	rt := newMockRuntime()
	mcn := newTestMultiChain(t, rt)

	tmpFile := filepath.Join(t.TempDir(), "bad.json")
	require.NoError(t, os.WriteFile(tmpFile, []byte("{invalid}"), 0o644))

	mcn.SetAllocFilePath(tmpFile)
	assert.Equal(t, 0, mcn.allocations.Counter)
	assert.Empty(t, mcn.allocations.Chains)
}

func TestMultiChainNetwork_SetAllocFilePath_Nonexistent(t *testing.T) {
	rt := newMockRuntime()
	mcn := newTestMultiChain(t, rt)

	mcn.SetAllocFilePath(filepath.Join(t.TempDir(), "nope.json"))
	assert.Equal(t, 0, mcn.allocations.Counter)
}

func TestMultiChainNetwork_AllocateSubnet_FirstAllocation(t *testing.T) {
	mcn := newTestMultiChain(t, newMockRuntime())

	subnet, err := mcn.AllocateSubnet("ethereum", nil)
	require.NoError(t, err)
	assert.Equal(t, "10.100.1.0/24", subnet)
	assert.Equal(t, 1, mcn.allocations.Counter)
	assert.Equal(t, subnet, mcn.allocations.Chains["ethereum"])
}

func TestMultiChainNetwork_AllocateSubnet_MultipleChains(t *testing.T) {
	mcn := newTestMultiChain(t, newMockRuntime())

	s1, err := mcn.AllocateSubnet("eth", nil)
	require.NoError(t, err)
	s2, err := mcn.AllocateSubnet("sol", nil)
	require.NoError(t, err)

	assert.NotEqual(t, s1, s2)
	assert.Equal(t, 2, mcn.allocations.Counter)
}

func TestMultiChainNetwork_AllocateSubnet_ReusesPersisted(t *testing.T) {
	mcn := newTestMultiChain(t, newMockRuntime())
	mcn.allocations.Counter = 5
	mcn.allocations.Chains["eth"] = "10.100.0.3/24"

	subnet, err := mcn.AllocateSubnet("eth", nil)
	require.NoError(t, err)
	assert.Equal(t, "10.100.0.3/24", subnet)
	assert.Equal(t, 5, mcn.allocations.Counter)
}

func TestMultiChainNetwork_AllocateSubnet_PersistedConflict(t *testing.T) {
	mcn := newTestMultiChain(t, newMockRuntime())
	mcn.allocations.Counter = 0
	mcn.allocations.Chains["eth"] = "10.100.0.1/24"

	subnet, err := mcn.AllocateSubnet("eth", []string{"10.100.0.0/24"})
	require.NoError(t, err)
	assert.NotEqual(t, "10.100.0.1/24", subnet)
}

func TestMultiChainNetwork_AllocateSubnet_SkipsConflicts(t *testing.T) {
	mcn := newTestMultiChain(t, newMockRuntime())

	used := []string{"10.100.0.1/24"}
	subnet, err := mcn.AllocateSubnet("eth", used)
	require.NoError(t, err)
	assert.NotEqual(t, "10.100.0.1/24", subnet)
}

func TestMultiChainNetwork_AllocateSubnet_InvalidBase(t *testing.T) {
	mcn := newTestMultiChain(t, newMockRuntime())
	mcn.ChainSubnetBase = "not-an-ip"

	_, err := mcn.AllocateSubnet("eth", nil)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "invalid chain subnet base")
}

func TestMultiChainNetwork_Setup_SingleChain(t *testing.T) {
	rt := newMockRuntime()
	rt.listNetworksFn = func(_ context.Context) ([]container.NetworkInfo, error) {
		return nil, nil
	}
	mcn := newTestMultiChain(t, rt)

	topo, err := mcn.Setup(context.Background(), []string{"ethereum"})
	require.NoError(t, err)
	assert.Len(t, topo.ChainNetworks, 1)
	assert.Contains(t, topo.ChainNetworks, "ethereum")
	assert.Empty(t, topo.InterconnectID) // No interconnect for single chain.
}

func TestMultiChainNetwork_Setup_MultipleChains(t *testing.T) {
	rt := newMockRuntime()
	rt.listNetworksFn = func(_ context.Context) ([]container.NetworkInfo, error) {
		return nil, nil
	}
	mcn := newTestMultiChain(t, rt)

	topo, err := mcn.Setup(context.Background(), []string{"ethereum", "solana"})
	require.NoError(t, err)
	assert.Len(t, topo.ChainNetworks, 2)
	assert.NotEmpty(t, topo.InterconnectID)
}

func TestMultiChainNetwork_Setup_CreateNetworkError(t *testing.T) {
	rt := newMockRuntime()
	rt.listNetworksFn = func(_ context.Context) ([]container.NetworkInfo, error) {
		return nil, nil
	}
	rt.createNetworkFn = func(_ context.Context, _ string, _ container.NetworkOptions) (string, error) {
		return "", errMock
	}
	mcn := newTestMultiChain(t, rt)

	_, err := mcn.Setup(context.Background(), []string{"eth"})
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "failed to create network for chain eth")
}

func TestMultiChainNetwork_Setup_InterconnectConflict(t *testing.T) {
	rt := newMockRuntime()
	callCount := 0
	rt.listNetworksFn = func(_ context.Context) ([]container.NetworkInfo, error) {
		return []container.NetworkInfo{
			{ID: "existing", Name: "some-net", Subnet: "10.200.0.0/16"},
		}, nil
	}
	rt.createNetworkFn = func(_ context.Context, name string, opts container.NetworkOptions) (string, error) {
		callCount++
		return fmt.Sprintf("net-%d", callCount), nil
	}
	mcn := newTestMultiChain(t, rt)

	_, err := mcn.Setup(context.Background(), []string{"eth", "sol"})
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "interconnect subnet")
}

func TestMultiChainNetwork_Setup_ListNetworksError(t *testing.T) {
	rt := newMockRuntime()
	rt.listNetworksFn = func(_ context.Context) ([]container.NetworkInfo, error) {
		return nil, errMock
	}
	mcn := newTestMultiChain(t, rt)

	topo, err := mcn.Setup(context.Background(), []string{"eth"})
	require.NoError(t, err)
	assert.NotNil(t, topo)
}

func TestMultiChainNetwork_Setup_PersistsAllocations(t *testing.T) {
	rt := newMockRuntime()
	rt.listNetworksFn = func(_ context.Context) ([]container.NetworkInfo, error) {
		return nil, nil
	}
	mcn := newTestMultiChain(t, rt)

	_, err := mcn.Setup(context.Background(), []string{"eth"})
	require.NoError(t, err)

	data, err := os.ReadFile(mcn.allocFilePath)
	require.NoError(t, err)
	var alloc subnetAllocations
	require.NoError(t, json.Unmarshal(data, &alloc))
	assert.Equal(t, 1, alloc.Counter)
	assert.Contains(t, alloc.Chains, "eth")
}

func TestMultiChainNetwork_Setup_InterconnectCreateError(t *testing.T) {
	rt := newMockRuntime()
	rt.listNetworksFn = func(_ context.Context) ([]container.NetworkInfo, error) {
		return nil, nil
	}
	callCount := 0
	rt.createNetworkFn = func(_ context.Context, name string, _ container.NetworkOptions) (string, error) {
		callCount++
		if callCount == 3 {
			return "", errMock
		}
		return fmt.Sprintf("net-%d", callCount), nil
	}
	mcn := newTestMultiChain(t, rt)

	_, err := mcn.Setup(context.Background(), []string{"eth", "sol"})
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "failed to create interconnect network")
}

func TestMultiChainNetwork_Teardown(t *testing.T) {
	rt := newMockRuntime()
	rt.listNetworksFn = func(_ context.Context) ([]container.NetworkInfo, error) {
		return nil, nil
	}
	mcn := newTestMultiChain(t, rt)

	_, err := mcn.Setup(context.Background(), []string{"eth", "sol"})
	require.NoError(t, err)

	err = mcn.Teardown(context.Background())
	assert.NoError(t, err)
	assert.Nil(t, mcn.Topology())
	assert.Equal(t, 3, rt.calls["RemoveNetwork"])
}

func TestMultiChainNetwork_Teardown_NilTopology(t *testing.T) {
	mcn := newTestMultiChain(t, newMockRuntime())
	err := mcn.Teardown(context.Background())
	assert.NoError(t, err)
}

func TestMultiChainNetwork_Teardown_RemoveError(t *testing.T) {
	rt := newMockRuntime()
	rt.listNetworksFn = func(_ context.Context) ([]container.NetworkInfo, error) {
		return nil, nil
	}
	rt.removeNetworkFn = func(_ context.Context, _ string) error {
		return errMock
	}
	mcn := newTestMultiChain(t, rt)

	_, err := mcn.Setup(context.Background(), []string{"eth"})
	require.NoError(t, err)

	err = mcn.Teardown(context.Background())
	assert.NoError(t, err)
	assert.Nil(t, mcn.Topology())
}

func TestMultiChainNetwork_ConnectToInterconnect_Success(t *testing.T) {
	rt := newMockRuntime()
	rt.listNetworksFn = func(_ context.Context) ([]container.NetworkInfo, error) {
		return nil, nil
	}
	mcn := newTestMultiChain(t, rt)

	_, err := mcn.Setup(context.Background(), []string{"eth", "sol"})
	require.NoError(t, err)

	var capturedNetworkID, capturedContainerID string
	rt.connectNetworkFn = func(_ context.Context, netID, ctrID string) error {
		capturedNetworkID = netID
		capturedContainerID = ctrID
		return nil
	}

	err = mcn.ConnectToInterconnect(context.Background(), "container-123")
	assert.NoError(t, err)
	assert.Equal(t, mcn.topology.InterconnectID, capturedNetworkID)
	assert.Equal(t, "container-123", capturedContainerID)
}

func TestMultiChainNetwork_ConnectToInterconnect_NoTopology(t *testing.T) {
	mcn := newTestMultiChain(t, newMockRuntime())
	err := mcn.ConnectToInterconnect(context.Background(), "ctr")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "no interconnect network configured")
}

func TestMultiChainNetwork_ConnectToInterconnect_NoInterconnect(t *testing.T) {
	mcn := newTestMultiChain(t, newMockRuntime())
	mcn.topology = &NetworkTopology{InterconnectID: ""}
	err := mcn.ConnectToInterconnect(context.Background(), "ctr")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "no interconnect network configured")
}

func TestMultiChainNetwork_ConnectToInterconnect_RuntimeError(t *testing.T) {
	rt := newMockRuntime()
	rt.listNetworksFn = func(_ context.Context) ([]container.NetworkInfo, error) {
		return nil, nil
	}
	rt.connectNetworkFn = func(_ context.Context, _, _ string) error {
		return errMock
	}
	mcn := newTestMultiChain(t, rt)

	_, err := mcn.Setup(context.Background(), []string{"eth", "sol"})
	require.NoError(t, err)

	err = mcn.ConnectToInterconnect(context.Background(), "ctr")
	assert.Error(t, err)
}

func TestMultiChainNetwork_Topology(t *testing.T) {
	rt := newMockRuntime()
	rt.listNetworksFn = func(_ context.Context) ([]container.NetworkInfo, error) {
		return nil, nil
	}
	mcn := newTestMultiChain(t, rt)
	assert.Nil(t, mcn.Topology())

	_, err := mcn.Setup(context.Background(), []string{"eth"})
	require.NoError(t, err)
	assert.NotNil(t, mcn.Topology())
}

func TestSubnetsOverlap_NoOverlap(t *testing.T) {
	assert.False(t, subnetsOverlap("10.100.0.0/24", []string{"10.200.0.0/24"}))
}

func TestSubnetsOverlap_ExactOverlap(t *testing.T) {
	assert.True(t, subnetsOverlap("10.100.0.0/24", []string{"10.100.0.0/24"}))
}

func TestSubnetsOverlap_ContainsCandidate(t *testing.T) {
	assert.True(t, subnetsOverlap("10.100.0.0/24", []string{"10.100.0.0/16"}))
}

func TestSubnetsOverlap_CandidateContainsExisting(t *testing.T) {
	assert.True(t, subnetsOverlap("10.100.0.0/16", []string{"10.100.5.0/24"}))
}

func TestSubnetsOverlap_EmptyExisting(t *testing.T) {
	assert.False(t, subnetsOverlap("10.100.0.0/24", nil))
}

func TestSubnetsOverlap_InvalidCandidate(t *testing.T) {
	assert.False(t, subnetsOverlap("not-a-cidr", []string{"10.0.0.0/8"}))
}

func TestSubnetsOverlap_InvalidExisting(t *testing.T) {
	assert.False(t, subnetsOverlap("10.100.0.0/24", []string{"not-a-cidr"}))
}

func TestMultiChainNetwork_SaveLoadAllocations(t *testing.T) {
	mcn := newTestMultiChain(t, newMockRuntime())
	mcn.allocations.Counter = 10
	mcn.allocations.Chains["eth"] = "10.100.0.1/24"

	err := mcn.saveAllocations()
	require.NoError(t, err)

	mcn2 := newTestMultiChain(t, newMockRuntime())
	mcn2.allocFilePath = mcn.allocFilePath
	mcn2.loadAllocations()

	assert.Equal(t, 10, mcn2.allocations.Counter)
	assert.Equal(t, "10.100.0.1/24", mcn2.allocations.Chains["eth"])
}

func TestMultiChainNetwork_LoadAllocations_NilChains(t *testing.T) {
	mcn := newTestMultiChain(t, newMockRuntime())
	data := []byte(`{"counter": 5}`)
	require.NoError(t, os.WriteFile(mcn.allocFilePath, data, 0o644))

	mcn.loadAllocations()
	assert.Equal(t, 5, mcn.allocations.Counter)
	assert.NotNil(t, mcn.allocations.Chains)
}
