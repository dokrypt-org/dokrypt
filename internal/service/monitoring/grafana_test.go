package monitoring

import (
	"context"
	"testing"

	"github.com/dokrypt/dokrypt/internal/config"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewGrafana(t *testing.T) {
	t.Run("basic creation", func(t *testing.T) {
		cfg := config.ServiceConfig{
			Type: "monitoring",
		}
		svc, err := NewGrafana("my-grafana", cfg, nil, "testproj")
		require.NoError(t, err)
		require.NotNil(t, svc)

		assert.Equal(t, "my-grafana", svc.Name())
		assert.Equal(t, "grafana", svc.Type())
		assert.Equal(t, "testproj", svc.ProjectName)
		assert.NotNil(t, svc.HostPorts)
		assert.NotNil(t, svc.ServiceURLs)
	})

	t.Run("dependencies from config", func(t *testing.T) {
		cfg := config.ServiceConfig{
			DependsOn: []string{"prometheus"},
		}
		svc, err := NewGrafana("g1", cfg, nil, "proj")
		require.NoError(t, err)

		deps := svc.DependsOn()
		assert.Equal(t, []string{"prometheus"}, deps)
	})

	t.Run("empty config", func(t *testing.T) {
		cfg := config.ServiceConfig{}
		svc, err := NewGrafana("g2", cfg, nil, "proj")
		require.NoError(t, err)
		assert.Empty(t, svc.DependsOn())
	})
}

func TestGrafanaFactory(t *testing.T) {
	cfg := config.ServiceConfig{}
	svc, err := GrafanaFactory("factory-g", cfg, nil, "proj")
	require.NoError(t, err)
	require.NotNil(t, svc)
	assert.Equal(t, "factory-g", svc.Name())
	assert.Equal(t, "grafana", svc.Type())
}

func TestGrafanaHealthNoURL(t *testing.T) {
	cfg := config.ServiceConfig{}
	svc, err := NewGrafana("g-health", cfg, nil, "proj")
	require.NoError(t, err)

	err = svc.Health(context.Background())
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "grafana URL not available")
}

func TestGrafanaPortsAndURLs(t *testing.T) {
	cfg := config.ServiceConfig{}
	svc, err := NewGrafana("g-ports", cfg, nil, "proj")
	require.NoError(t, err)

	assert.NotNil(t, svc.Ports())
	assert.Empty(t, svc.Ports())
	assert.NotNil(t, svc.URLs())
	assert.Empty(t, svc.URLs())
}

func TestGrafanaCfgStored(t *testing.T) {
	cfg := config.ServiceConfig{
		Type: "monitoring",
		Port: 4000,
		Environment: map[string]string{
			"GF_SECURITY_ADMIN_USER":     "custom-admin",
			"GF_SECURITY_ADMIN_PASSWORD": "custom-pass",
		},
	}
	svc, err := NewGrafana("g-cfg", cfg, nil, "proj")
	require.NoError(t, err)
	assert.Equal(t, cfg, svc.cfg)
}

func TestGrafanaDefaultCredentials(t *testing.T) {
	assert.Equal(t, "admin", defaultGrafanaAdminUser)
	assert.Equal(t, "dokrypt", defaultGrafanaAdminPassword)
}
