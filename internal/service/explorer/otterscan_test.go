package explorer

import (
	"context"
	"testing"

	"github.com/dokrypt/dokrypt/internal/config"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewOtterscan(t *testing.T) {
	t.Run("basic creation", func(t *testing.T) {
		cfg := config.ServiceConfig{
			Type:  "explorer",
			Chain: "ethereum",
		}
		svc, err := NewOtterscan("my-otterscan", cfg, nil, "testproj")
		require.NoError(t, err)
		require.NotNil(t, svc)

		assert.Equal(t, "my-otterscan", svc.Name())
		assert.Equal(t, "otterscan", svc.Type())
		assert.Equal(t, "testproj", svc.ProjectName)
		assert.NotNil(t, svc.HostPorts)
		assert.NotNil(t, svc.ServiceURLs)
	})

	t.Run("dependencies include chain", func(t *testing.T) {
		cfg := config.ServiceConfig{
			Chain:     "arbitrum",
			DependsOn: []string{"dep1"},
		}
		svc, err := NewOtterscan("ot1", cfg, nil, "proj")
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
		svc, err := NewOtterscan("ot2", cfg, nil, "proj")
		require.NoError(t, err)
		assert.Equal(t, []string{"dep1"}, svc.DependsOn())
	})

	t.Run("empty config", func(t *testing.T) {
		cfg := config.ServiceConfig{}
		svc, err := NewOtterscan("ot3", cfg, nil, "proj")
		require.NoError(t, err)
		assert.Empty(t, svc.DependsOn())
	})
}

func TestOtterscanFactory(t *testing.T) {
	cfg := config.ServiceConfig{Chain: "eth"}
	svc, err := OtterscanFactory("factory-ot", cfg, nil, "proj")
	require.NoError(t, err)
	require.NotNil(t, svc)
	assert.Equal(t, "factory-ot", svc.Name())
	assert.Equal(t, "otterscan", svc.Type())
}

func TestOtterscanHealthNoURL(t *testing.T) {
	cfg := config.ServiceConfig{}
	svc, err := NewOtterscan("ot-health", cfg, nil, "proj")
	require.NoError(t, err)

	err = svc.Health(context.Background())
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "otterscan URL not available")
}

func TestOtterscanPortsAndURLs(t *testing.T) {
	cfg := config.ServiceConfig{}
	svc, err := NewOtterscan("ot-ports", cfg, nil, "proj")
	require.NoError(t, err)

	assert.NotNil(t, svc.Ports())
	assert.Empty(t, svc.Ports())
	assert.NotNil(t, svc.URLs())
	assert.Empty(t, svc.URLs())
}

func TestOtterscanCfgStored(t *testing.T) {
	cfg := config.ServiceConfig{
		Type:  "explorer",
		Chain: "optimism",
		Environment: map[string]string{
			"VITE_CONFIG_JSON": `{"custom":true}`,
		},
	}
	svc, err := NewOtterscan("ot-cfg", cfg, nil, "proj")
	require.NoError(t, err)
	assert.Equal(t, cfg, svc.cfg)
}
