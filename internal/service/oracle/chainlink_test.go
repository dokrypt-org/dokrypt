package oracle

import (
	"context"
	"testing"

	"github.com/dokrypt/dokrypt/internal/config"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewChainlinkMock(t *testing.T) {
	t.Run("basic creation", func(t *testing.T) {
		cfg := config.ServiceConfig{
			Type:  "oracle",
			Chain: "ethereum",
		}
		svc, err := NewChainlinkMock("my-chainlink", cfg, nil, "testproj")
		require.NoError(t, err)
		require.NotNil(t, svc)

		assert.Equal(t, "my-chainlink", svc.Name())
		assert.Equal(t, "chainlink-mock", svc.Type())
		assert.Equal(t, "testproj", svc.ProjectName)
		assert.NotNil(t, svc.HostPorts)
		assert.NotNil(t, svc.ServiceURLs)
	})

	t.Run("dependencies include chain", func(t *testing.T) {
		cfg := config.ServiceConfig{
			Chain:     "polygon",
			DependsOn: []string{"dep1"},
		}
		svc, err := NewChainlinkMock("cl1", cfg, nil, "proj")
		require.NoError(t, err)

		deps := svc.DependsOn()
		assert.Contains(t, deps, "dep1")
		assert.Contains(t, deps, "polygon")
		assert.Len(t, deps, 2)
	})

	t.Run("no chain does not add dep", func(t *testing.T) {
		cfg := config.ServiceConfig{
			DependsOn: []string{"dep1"},
		}
		svc, err := NewChainlinkMock("cl2", cfg, nil, "proj")
		require.NoError(t, err)
		assert.Equal(t, []string{"dep1"}, svc.DependsOn())
	})

	t.Run("empty config", func(t *testing.T) {
		cfg := config.ServiceConfig{}
		svc, err := NewChainlinkMock("cl3", cfg, nil, "proj")
		require.NoError(t, err)
		assert.Empty(t, svc.DependsOn())
	})
}

func TestChainlinkFactory(t *testing.T) {
	cfg := config.ServiceConfig{Chain: "eth"}
	svc, err := ChainlinkFactory("factory-cl", cfg, nil, "proj")
	require.NoError(t, err)
	require.NotNil(t, svc)
	assert.Equal(t, "factory-cl", svc.Name())
	assert.Equal(t, "chainlink-mock", svc.Type())
}

func TestChainlinkHealthNoURL(t *testing.T) {
	cfg := config.ServiceConfig{}
	svc, err := NewChainlinkMock("cl-health", cfg, nil, "proj")
	require.NoError(t, err)

	err = svc.Health(context.Background())
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "oracle URL not available")
}

func TestChainlinkPortsAndURLs(t *testing.T) {
	cfg := config.ServiceConfig{}
	svc, err := NewChainlinkMock("cl-ports", cfg, nil, "proj")
	require.NoError(t, err)

	assert.NotNil(t, svc.Ports())
	assert.Empty(t, svc.Ports())
	assert.NotNil(t, svc.URLs())
	assert.Empty(t, svc.URLs())
}

func TestChainlinkCfgStored(t *testing.T) {
	cfg := config.ServiceConfig{
		Type:  "oracle",
		Chain: "ethereum",
		Feeds: []config.OracleFeedConfig{
			{Pair: "ETH/USD", Price: 3500.0, Decimals: 8},
		},
	}
	svc, err := NewChainlinkMock("cl-cfg", cfg, nil, "proj")
	require.NoError(t, err)
	assert.Equal(t, cfg, svc.cfg)
}
