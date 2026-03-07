package oracle

import (
	"context"
	"testing"

	"github.com/dokrypt/dokrypt/internal/config"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewPythMock(t *testing.T) {
	t.Run("basic creation", func(t *testing.T) {
		cfg := config.ServiceConfig{
			Type:  "oracle",
			Chain: "ethereum",
		}
		svc, err := NewPythMock("my-pyth", cfg, nil, "testproj")
		require.NoError(t, err)
		require.NotNil(t, svc)

		assert.Equal(t, "my-pyth", svc.Name())
		assert.Equal(t, "pyth-mock", svc.Type())
		assert.Equal(t, "testproj", svc.ProjectName)
		assert.NotNil(t, svc.HostPorts)
		assert.NotNil(t, svc.ServiceURLs)
	})

	t.Run("dependencies include chain", func(t *testing.T) {
		cfg := config.ServiceConfig{
			Chain:     "solana",
			DependsOn: []string{"dep1"},
		}
		svc, err := NewPythMock("pyth1", cfg, nil, "proj")
		require.NoError(t, err)

		deps := svc.DependsOn()
		assert.Contains(t, deps, "dep1")
		assert.Contains(t, deps, "solana")
		assert.Len(t, deps, 2)
	})

	t.Run("no chain does not add dep", func(t *testing.T) {
		cfg := config.ServiceConfig{
			DependsOn: []string{"dep1"},
		}
		svc, err := NewPythMock("pyth2", cfg, nil, "proj")
		require.NoError(t, err)
		assert.Equal(t, []string{"dep1"}, svc.DependsOn())
	})

	t.Run("empty config", func(t *testing.T) {
		cfg := config.ServiceConfig{}
		svc, err := NewPythMock("pyth3", cfg, nil, "proj")
		require.NoError(t, err)
		assert.Empty(t, svc.DependsOn())
	})
}

func TestPythFactory(t *testing.T) {
	cfg := config.ServiceConfig{Chain: "eth"}
	svc, err := PythFactory("factory-pyth", cfg, nil, "proj")
	require.NoError(t, err)
	require.NotNil(t, svc)
	assert.Equal(t, "factory-pyth", svc.Name())
	assert.Equal(t, "pyth-mock", svc.Type())
}

func TestPythHealthNoURL(t *testing.T) {
	cfg := config.ServiceConfig{}
	svc, err := NewPythMock("pyth-health", cfg, nil, "proj")
	require.NoError(t, err)

	err = svc.Health(context.Background())
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "pyth oracle URL not available")
}

func TestPythPortsAndURLs(t *testing.T) {
	cfg := config.ServiceConfig{}
	svc, err := NewPythMock("pyth-ports", cfg, nil, "proj")
	require.NoError(t, err)

	assert.NotNil(t, svc.Ports())
	assert.Empty(t, svc.Ports())
	assert.NotNil(t, svc.URLs())
	assert.Empty(t, svc.URLs())
}

func TestPythCfgStored(t *testing.T) {
	cfg := config.ServiceConfig{
		Type:  "oracle",
		Chain: "ethereum",
		Feeds: []config.OracleFeedConfig{
			{Pair: "BTC/USD", Price: 68000.0, Decimals: 8},
			{Pair: "ETH/USD", Price: 3500.0, Decimals: 8},
		},
	}
	svc, err := NewPythMock("pyth-cfg", cfg, nil, "proj")
	require.NoError(t, err)
	assert.Equal(t, cfg, svc.cfg)
}
