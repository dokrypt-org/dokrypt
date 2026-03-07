package indexer

import (
	"context"
	"testing"

	"github.com/dokrypt/dokrypt/internal/config"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewSubgraph(t *testing.T) {
	t.Run("basic creation", func(t *testing.T) {
		cfg := config.ServiceConfig{
			Type:  "indexer",
			Chain: "ethereum",
		}
		svc, err := NewSubgraph("my-subgraph", cfg, nil, "testproj")
		require.NoError(t, err)
		require.NotNil(t, svc)

		assert.Equal(t, "my-subgraph", svc.Name())
		assert.Equal(t, "subgraph", svc.Type())
		assert.Equal(t, "testproj", svc.ProjectName)
		assert.NotNil(t, svc.HostPorts)
		assert.NotNil(t, svc.ServiceURLs)
	})

	t.Run("dependencies include chain", func(t *testing.T) {
		cfg := config.ServiceConfig{
			Chain:     "arbitrum",
			DependsOn: []string{"dep1"},
		}
		svc, err := NewSubgraph("sg1", cfg, nil, "proj")
		require.NoError(t, err)

		deps := svc.DependsOn()
		assert.Contains(t, deps, "dep1")
		assert.Contains(t, deps, "arbitrum")
		assert.Len(t, deps, 2)
	})

	t.Run("no chain does not add dep", func(t *testing.T) {
		cfg := config.ServiceConfig{
			DependsOn: []string{"dep1"},
		}
		svc, err := NewSubgraph("sg2", cfg, nil, "proj")
		require.NoError(t, err)
		assert.Equal(t, []string{"dep1"}, svc.DependsOn())
	})

	t.Run("empty config", func(t *testing.T) {
		cfg := config.ServiceConfig{}
		svc, err := NewSubgraph("sg3", cfg, nil, "proj")
		require.NoError(t, err)
		assert.Empty(t, svc.DependsOn())
	})
}

func TestSubgraphFactory(t *testing.T) {
	cfg := config.ServiceConfig{Chain: "eth"}
	svc, err := SubgraphFactory("factory-sg", cfg, nil, "proj")
	require.NoError(t, err)
	require.NotNil(t, svc)
	assert.Equal(t, "factory-sg", svc.Name())
	assert.Equal(t, "subgraph", svc.Type())
}

func TestSubgraphHealthNoURL(t *testing.T) {
	cfg := config.ServiceConfig{}
	svc, err := NewSubgraph("sg-health", cfg, nil, "proj")
	require.NoError(t, err)

	err = svc.Health(context.Background())
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "subgraph GraphQL URL not available")
}

func TestSubgraphPortsAndURLs(t *testing.T) {
	cfg := config.ServiceConfig{}
	svc, err := NewSubgraph("sg-ports", cfg, nil, "proj")
	require.NoError(t, err)

	assert.NotNil(t, svc.Ports())
	assert.Empty(t, svc.Ports())
	assert.NotNil(t, svc.URLs())
	assert.Empty(t, svc.URLs())
}

func TestSubgraphCfgStored(t *testing.T) {
	cfg := config.ServiceConfig{
		Type:        "indexer",
		Chain:       "ethereum",
		GraphQLPort: 9000,
		AdminPort:   9020,
		IPFS:        "custom-ipfs:5001",
		Environment: map[string]string{
			"postgres_host": "custom-postgres",
		},
	}
	svc, err := NewSubgraph("sg-cfg", cfg, nil, "proj")
	require.NoError(t, err)
	assert.Equal(t, cfg, svc.cfg)
}

func TestSubgraphDefaultConstants(t *testing.T) {
	assert.Equal(t, "graphprotocol/graph-node:latest", subgraphImage)
	assert.Equal(t, 8000, defaultGraphQLPort)
	assert.Equal(t, 8020, defaultAdminPort)
}
