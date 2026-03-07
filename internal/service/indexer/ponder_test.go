package indexer

import (
	"context"
	"testing"

	"github.com/dokrypt/dokrypt/internal/config"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewPonder(t *testing.T) {
	t.Run("basic creation", func(t *testing.T) {
		cfg := config.ServiceConfig{
			Type:  "indexer",
			Chain: "ethereum",
		}
		svc, err := NewPonder("my-ponder", cfg, nil, "testproj")
		require.NoError(t, err)
		require.NotNil(t, svc)

		assert.Equal(t, "my-ponder", svc.Name())
		assert.Equal(t, "ponder", svc.Type())
		assert.Equal(t, "testproj", svc.ProjectName)
		assert.NotNil(t, svc.HostPorts)
		assert.NotNil(t, svc.ServiceURLs)
	})

	t.Run("dependencies include chain", func(t *testing.T) {
		cfg := config.ServiceConfig{
			Chain:     "polygon",
			DependsOn: []string{"dep1"},
		}
		svc, err := NewPonder("p1", cfg, nil, "proj")
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
		svc, err := NewPonder("p2", cfg, nil, "proj")
		require.NoError(t, err)
		assert.Equal(t, []string{"dep1"}, svc.DependsOn())
	})

	t.Run("empty config", func(t *testing.T) {
		cfg := config.ServiceConfig{}
		svc, err := NewPonder("p3", cfg, nil, "proj")
		require.NoError(t, err)
		assert.Empty(t, svc.DependsOn())
	})
}

func TestPonderFactory(t *testing.T) {
	cfg := config.ServiceConfig{Chain: "eth"}
	svc, err := PonderFactory("factory-p", cfg, nil, "proj")
	require.NoError(t, err)
	require.NotNil(t, svc)
	assert.Equal(t, "factory-p", svc.Name())
	assert.Equal(t, "ponder", svc.Type())
}

func TestPonderHealthNoURL(t *testing.T) {
	cfg := config.ServiceConfig{}
	svc, err := NewPonder("p-health", cfg, nil, "proj")
	require.NoError(t, err)

	err = svc.Health(context.Background())
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "ponder URL not available")
}

func TestPonderPortsAndURLs(t *testing.T) {
	cfg := config.ServiceConfig{}
	svc, err := NewPonder("p-ports", cfg, nil, "proj")
	require.NoError(t, err)

	assert.NotNil(t, svc.Ports())
	assert.Empty(t, svc.Ports())
	assert.NotNil(t, svc.URLs())
	assert.Empty(t, svc.URLs())
}

func TestPonderCfgStored(t *testing.T) {
	cfg := config.ServiceConfig{
		Type:    "indexer",
		Chain:   "optimism",
		Image:   "node:20-alpine",
		Port:    5555,
		Volumes: []string{"/my/project"},
		Command: []string{"npm", "run", "dev"},
		Environment: map[string]string{
			"PONDER_RPC_URL_1": "http://custom:8545",
		},
	}
	svc, err := NewPonder("p-cfg", cfg, nil, "proj")
	require.NoError(t, err)
	assert.Equal(t, cfg, svc.cfg)
}

func TestPonderDefaultConstants(t *testing.T) {
	assert.Equal(t, "node:18-alpine", defaultPonderImage)
	assert.Equal(t, 42069, defaultPonderPort)
}
