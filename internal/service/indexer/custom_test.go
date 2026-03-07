package indexer

import (
	"context"
	"testing"

	"github.com/dokrypt/dokrypt/internal/config"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewCustomIndexer(t *testing.T) {
	t.Run("basic creation", func(t *testing.T) {
		cfg := config.ServiceConfig{
			Type:  "indexer",
			Image: "my-indexer:latest",
		}
		svc, err := NewCustomIndexer("my-indexer", cfg, nil, "testproj")
		require.NoError(t, err)
		require.NotNil(t, svc)

		assert.Equal(t, "my-indexer", svc.Name())
		assert.Equal(t, "custom-indexer", svc.Type())
		assert.Equal(t, "testproj", svc.ProjectName)
		assert.NotNil(t, svc.HostPorts)
		assert.NotNil(t, svc.ServiceURLs)
	})

	t.Run("dependencies from config", func(t *testing.T) {
		cfg := config.ServiceConfig{
			DependsOn: []string{"dep1", "dep2"},
		}
		svc, err := NewCustomIndexer("ci1", cfg, nil, "proj")
		require.NoError(t, err)

		deps := svc.DependsOn()
		assert.Equal(t, []string{"dep1", "dep2"}, deps)
	})

	t.Run("empty config", func(t *testing.T) {
		cfg := config.ServiceConfig{}
		svc, err := NewCustomIndexer("ci2", cfg, nil, "proj")
		require.NoError(t, err)
		assert.Empty(t, svc.DependsOn())
	})
}

func TestCustomIndexerFactory(t *testing.T) {
	cfg := config.ServiceConfig{Image: "img"}
	svc, err := CustomIndexerFactory("factory-ci", cfg, nil, "proj")
	require.NoError(t, err)
	require.NotNil(t, svc)
	assert.Equal(t, "factory-ci", svc.Name())
	assert.Equal(t, "custom-indexer", svc.Type())
}

func TestCustomIndexerHealthNoURL(t *testing.T) {
	cfg := config.ServiceConfig{}
	svc, err := NewCustomIndexer("ci-health", cfg, nil, "proj")
	require.NoError(t, err)

	err = svc.Health(context.Background())
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "indexer URL not available")
}

func TestCustomIndexerPortsAndURLs(t *testing.T) {
	cfg := config.ServiceConfig{}
	svc, err := NewCustomIndexer("ci-ports", cfg, nil, "proj")
	require.NoError(t, err)

	assert.NotNil(t, svc.Ports())
	assert.Empty(t, svc.Ports())
	assert.NotNil(t, svc.URLs())
	assert.Empty(t, svc.URLs())
}

func TestCustomIndexerCfgStored(t *testing.T) {
	cfg := config.ServiceConfig{
		Type:  "indexer",
		Image: "custom:v1",
		Port:  4000,
		Environment: map[string]string{
			"DB_URL": "postgres://localhost/mydb",
		},
	}
	svc, err := NewCustomIndexer("ci-cfg", cfg, nil, "proj")
	require.NoError(t, err)
	assert.Equal(t, cfg, svc.cfg)
}
