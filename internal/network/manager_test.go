package network

import (
	"context"
	"fmt"
	"testing"

	"github.com/dokrypt/dokrypt/internal/container"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewManager(t *testing.T) {
	rt := newMockRuntime()
	m := NewManager(rt, "myproject")

	assert.NotNil(t, m)
	assert.Equal(t, "myproject", m.projectName)
	assert.NotNil(t, m.networks)
}

func TestManager_EnvironmentNetworkName(t *testing.T) {
	m := NewManager(newMockRuntime(), "proj")
	assert.Equal(t, "dokrypt-proj", m.EnvironmentNetworkName())
}

func TestManager_ChainNetworkName(t *testing.T) {
	m := NewManager(newMockRuntime(), "proj")
	assert.Equal(t, "dokrypt-proj-ethereum", m.ChainNetworkName("ethereum"))
}

func TestManager_CreateEnvironmentNetwork_Success(t *testing.T) {
	rt := newMockRuntime()
	var capturedName string
	var capturedOpts container.NetworkOptions
	rt.createNetworkFn = func(_ context.Context, name string, opts container.NetworkOptions) (string, error) {
		capturedName = name
		capturedOpts = opts
		return "net-123", nil
	}
	rt.listNetworksFn = func(_ context.Context) ([]container.NetworkInfo, error) {
		return nil, nil // no existing networks
	}

	m := NewManager(rt, "alpha")
	id, err := m.CreateEnvironmentNetwork(context.Background())
	require.NoError(t, err)
	assert.Equal(t, "net-123", id)
	assert.Equal(t, "dokrypt-alpha", capturedName)
	assert.Equal(t, "bridge", capturedOpts.Driver)
	assert.Equal(t, "alpha", capturedOpts.Labels["dokrypt.project"])
	assert.Equal(t, "environment", capturedOpts.Labels["dokrypt.type"])

	storedID, ok := m.NetworkID("dokrypt-alpha")
	assert.True(t, ok)
	assert.Equal(t, "net-123", storedID)
}

func TestManager_CreateEnvironmentNetwork_ReusesExisting(t *testing.T) {
	rt := newMockRuntime()
	rt.listNetworksFn = func(_ context.Context) ([]container.NetworkInfo, error) {
		return []container.NetworkInfo{
			{ID: "existing-id", Name: "dokrypt-beta"},
		}, nil
	}
	rt.createNetworkFn = func(_ context.Context, _ string, _ container.NetworkOptions) (string, error) {
		t.Fatal("CreateNetwork should not be called when network already exists")
		return "", nil
	}

	m := NewManager(rt, "beta")
	id, err := m.CreateEnvironmentNetwork(context.Background())
	require.NoError(t, err)
	assert.Equal(t, "existing-id", id)
}

func TestManager_CreateEnvironmentNetwork_ListError(t *testing.T) {
	rt := newMockRuntime()
	rt.listNetworksFn = func(_ context.Context) ([]container.NetworkInfo, error) {
		return nil, errMock
	}

	m := NewManager(rt, "proj")
	_, err := m.CreateEnvironmentNetwork(context.Background())
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "failed to check for existing network")
}

func TestManager_CreateEnvironmentNetwork_CreateError(t *testing.T) {
	rt := newMockRuntime()
	rt.listNetworksFn = func(_ context.Context) ([]container.NetworkInfo, error) {
		return nil, nil
	}
	rt.createNetworkFn = func(_ context.Context, _ string, _ container.NetworkOptions) (string, error) {
		return "", fmt.Errorf("docker daemon error")
	}

	m := NewManager(rt, "proj")
	_, err := m.CreateEnvironmentNetwork(context.Background())
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "failed to create environment network")
}

func TestManager_CreateChainNetwork_Success(t *testing.T) {
	rt := newMockRuntime()
	var capturedName string
	rt.createNetworkFn = func(_ context.Context, name string, opts container.NetworkOptions) (string, error) {
		capturedName = name
		assert.Equal(t, "ethereum", opts.Labels["dokrypt.chain"])
		assert.Equal(t, "chain", opts.Labels["dokrypt.type"])
		return "chain-net-id", nil
	}

	m := NewManager(rt, "myproj")
	id, err := m.CreateChainNetwork(context.Background(), "ethereum")
	require.NoError(t, err)
	assert.Equal(t, "chain-net-id", id)
	assert.Equal(t, "dokrypt-myproj-ethereum", capturedName)
}

func TestManager_CreateChainNetwork_Error(t *testing.T) {
	rt := newMockRuntime()
	rt.createNetworkFn = func(_ context.Context, _ string, _ container.NetworkOptions) (string, error) {
		return "", errMock
	}

	m := NewManager(rt, "proj")
	_, err := m.CreateChainNetwork(context.Background(), "solana")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "failed to create chain network for solana")
}

func TestManager_CreateInterconnectNetwork_Success(t *testing.T) {
	rt := newMockRuntime()
	var capturedName string
	rt.createNetworkFn = func(_ context.Context, name string, opts container.NetworkOptions) (string, error) {
		capturedName = name
		assert.Equal(t, "interconnect", opts.Labels["dokrypt.type"])
		return "ic-id", nil
	}

	m := NewManager(rt, "proj")
	id, err := m.CreateInterconnectNetwork(context.Background())
	require.NoError(t, err)
	assert.Equal(t, "ic-id", id)
	assert.Equal(t, "dokrypt-proj-interconnect", capturedName)
}

func TestManager_CreateInterconnectNetwork_Error(t *testing.T) {
	rt := newMockRuntime()
	rt.createNetworkFn = func(_ context.Context, _ string, _ container.NetworkOptions) (string, error) {
		return "", errMock
	}

	m := NewManager(rt, "proj")
	_, err := m.CreateInterconnectNetwork(context.Background())
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "failed to create interconnect network")
}

func TestManager_NetworkID_Found(t *testing.T) {
	rt := newMockRuntime()
	rt.listNetworksFn = func(_ context.Context) ([]container.NetworkInfo, error) {
		return nil, nil
	}
	m := NewManager(rt, "proj")
	m.CreateEnvironmentNetwork(context.Background())

	id, ok := m.NetworkID("dokrypt-proj")
	assert.True(t, ok)
	assert.NotEmpty(t, id)
}

func TestManager_NetworkID_NotFound(t *testing.T) {
	m := NewManager(newMockRuntime(), "proj")
	_, ok := m.NetworkID("nonexistent")
	assert.False(t, ok)
}

func TestManager_RemoveAll(t *testing.T) {
	rt := newMockRuntime()
	rt.listNetworksFn = func(_ context.Context) ([]container.NetworkInfo, error) {
		return nil, nil
	}
	m := NewManager(rt, "proj")
	m.CreateEnvironmentNetwork(context.Background())
	m.CreateChainNetwork(context.Background(), "eth")

	err := m.RemoveAll(context.Background())
	assert.NoError(t, err)
	assert.GreaterOrEqual(t, rt.calls["RemoveNetwork"], 2)

	_, ok := m.NetworkID("dokrypt-proj")
	assert.False(t, ok)
}

func TestManager_RemoveAll_Empty(t *testing.T) {
	m := NewManager(newMockRuntime(), "proj")
	err := m.RemoveAll(context.Background())
	assert.NoError(t, err)
}

func TestManager_RemoveAll_RemoveError(t *testing.T) {
	rt := newMockRuntime()
	rt.listNetworksFn = func(_ context.Context) ([]container.NetworkInfo, error) {
		return nil, nil
	}
	rt.removeNetworkFn = func(_ context.Context, _ string) error {
		return errMock
	}
	m := NewManager(rt, "proj")
	m.CreateEnvironmentNetwork(context.Background())

	err := m.RemoveAll(context.Background())
	assert.NoError(t, err)
}
