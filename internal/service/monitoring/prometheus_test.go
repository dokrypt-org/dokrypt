package monitoring

import (
	"context"
	"testing"

	"github.com/dokrypt/dokrypt/internal/config"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewPrometheus(t *testing.T) {
	t.Run("basic creation", func(t *testing.T) {
		cfg := config.ServiceConfig{
			Type: "monitoring",
		}
		svc, err := NewPrometheus("my-prometheus", cfg, nil, "testproj")
		require.NoError(t, err)
		require.NotNil(t, svc)

		assert.Equal(t, "my-prometheus", svc.Name())
		assert.Equal(t, "prometheus", svc.Type())
		assert.Equal(t, "testproj", svc.ProjectName)
		assert.NotNil(t, svc.HostPorts)
		assert.NotNil(t, svc.ServiceURLs)
	})

	t.Run("dependencies from config", func(t *testing.T) {
		cfg := config.ServiceConfig{
			DependsOn: []string{"dep1"},
		}
		svc, err := NewPrometheus("p1", cfg, nil, "proj")
		require.NoError(t, err)

		deps := svc.DependsOn()
		assert.Equal(t, []string{"dep1"}, deps)
	})

	t.Run("empty config", func(t *testing.T) {
		cfg := config.ServiceConfig{}
		svc, err := NewPrometheus("p2", cfg, nil, "proj")
		require.NoError(t, err)
		assert.Empty(t, svc.DependsOn())
	})
}

func TestPrometheusFactory(t *testing.T) {
	cfg := config.ServiceConfig{}
	svc, err := PrometheusFactory("factory-p", cfg, nil, "proj")
	require.NoError(t, err)
	require.NotNil(t, svc)
	assert.Equal(t, "factory-p", svc.Name())
	assert.Equal(t, "prometheus", svc.Type())
}

func TestPrometheusHealthNoURL(t *testing.T) {
	cfg := config.ServiceConfig{}
	svc, err := NewPrometheus("p-health", cfg, nil, "proj")
	require.NoError(t, err)

	err = svc.Health(context.Background())
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "prometheus URL not available")
}

func TestPrometheusPortsAndURLs(t *testing.T) {
	cfg := config.ServiceConfig{}
	svc, err := NewPrometheus("p-ports", cfg, nil, "proj")
	require.NoError(t, err)

	assert.NotNil(t, svc.Ports())
	assert.Empty(t, svc.Ports())
	assert.NotNil(t, svc.URLs())
	assert.Empty(t, svc.URLs())
}

func TestPrometheusCfgStored(t *testing.T) {
	cfg := config.ServiceConfig{
		Type: "monitoring",
		Environment: map[string]string{
			"CUSTOM_FLAG": "true",
		},
	}
	svc, err := NewPrometheus("p-cfg", cfg, nil, "proj")
	require.NoError(t, err)
	assert.Equal(t, cfg, svc.cfg)
}
