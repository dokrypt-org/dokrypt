package explorer

import (
	"context"
	"testing"

	"github.com/dokrypt/dokrypt/internal/config"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewBlockscout(t *testing.T) {
	t.Run("basic creation", func(t *testing.T) {
		cfg := config.ServiceConfig{
			Type:  "explorer",
			Chain: "ethereum",
		}
		svc, err := NewBlockscout("my-blockscout", cfg, nil, "testproj")
		require.NoError(t, err)
		require.NotNil(t, svc)

		assert.Equal(t, "my-blockscout", svc.Name())
		assert.Equal(t, "blockscout", svc.Type())
		assert.Equal(t, "testproj", svc.ProjectName)
		assert.NotNil(t, svc.HostPorts)
		assert.NotNil(t, svc.ServiceURLs)
	})

	t.Run("dependencies include chain", func(t *testing.T) {
		cfg := config.ServiceConfig{
			Chain:     "polygon",
			DependsOn: []string{"some-dep"},
		}
		svc, err := NewBlockscout("bs1", cfg, nil, "proj")
		require.NoError(t, err)

		deps := svc.DependsOn()
		assert.Contains(t, deps, "some-dep")
		assert.Contains(t, deps, "polygon")
		assert.Len(t, deps, 2)
	})

	t.Run("no chain does not add dep", func(t *testing.T) {
		cfg := config.ServiceConfig{
			DependsOn: []string{"dep1"},
		}
		svc, err := NewBlockscout("bs2", cfg, nil, "proj")
		require.NoError(t, err)

		deps := svc.DependsOn()
		assert.Equal(t, []string{"dep1"}, deps)
	})

	t.Run("empty config", func(t *testing.T) {
		cfg := config.ServiceConfig{}
		svc, err := NewBlockscout("bs3", cfg, nil, "proj")
		require.NoError(t, err)
		assert.Empty(t, svc.DependsOn())
	})
}

func TestBlockscoutFactory(t *testing.T) {
	cfg := config.ServiceConfig{Chain: "eth"}
	svc, err := BlockscoutFactory("factory-bs", cfg, nil, "proj")
	require.NoError(t, err)
	require.NotNil(t, svc)
	assert.Equal(t, "factory-bs", svc.Name())
	assert.Equal(t, "blockscout", svc.Type())
}

func TestBlockscoutHealthNoURL(t *testing.T) {
	cfg := config.ServiceConfig{}
	svc, err := NewBlockscout("bs-health", cfg, nil, "proj")
	require.NoError(t, err)

	err = svc.Health(context.Background())
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "blockscout URL not available")
}

func TestBlockscoutPortsAndURLs(t *testing.T) {
	cfg := config.ServiceConfig{}
	svc, err := NewBlockscout("bs-ports", cfg, nil, "proj")
	require.NoError(t, err)

	assert.NotNil(t, svc.Ports())
	assert.Empty(t, svc.Ports())
	assert.NotNil(t, svc.URLs())
	assert.Empty(t, svc.URLs())
}

func TestBlockscoutCfgStored(t *testing.T) {
	cfg := config.ServiceConfig{
		Type:  "explorer",
		Chain: "ethereum",
		Port:  5000,
		Environment: map[string]string{
			"DATABASE_URL": "custom-db-url",
		},
	}
	svc, err := NewBlockscout("bs-cfg", cfg, nil, "proj")
	require.NoError(t, err)
	assert.Equal(t, cfg, svc.cfg)
}
