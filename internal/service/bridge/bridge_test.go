package bridge

import (
	"context"
	"testing"

	"github.com/dokrypt/dokrypt/internal/config"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNew(t *testing.T) {
	t.Run("basic creation", func(t *testing.T) {
		cfg := config.ServiceConfig{
			Type:   "bridge",
			Chains: []string{"ethereum", "polygon"},
		}
		svc, err := New("my-bridge", cfg, nil, "testproj")
		require.NoError(t, err)
		require.NotNil(t, svc)

		assert.Equal(t, "my-bridge", svc.Name())
		assert.Equal(t, "mock-bridge", svc.Type())
		assert.Equal(t, "testproj", svc.ProjectName)
		assert.NotNil(t, svc.HostPorts)
		assert.NotNil(t, svc.ServiceURLs)
	})

	t.Run("dependencies include chains", func(t *testing.T) {
		cfg := config.ServiceConfig{
			Chains:    []string{"ethereum", "polygon"},
			DependsOn: []string{"some-dep"},
		}
		svc, err := New("bridge1", cfg, nil, "proj")
		require.NoError(t, err)

		deps := svc.DependsOn()
		assert.Contains(t, deps, "some-dep")
		assert.Contains(t, deps, "ethereum")
		assert.Contains(t, deps, "polygon")
		assert.Len(t, deps, 3)
	})

	t.Run("no chains no depends", func(t *testing.T) {
		cfg := config.ServiceConfig{}
		svc, err := New("bridge2", cfg, nil, "proj")
		require.NoError(t, err)
		assert.Empty(t, svc.DependsOn())
	})

	t.Run("host ports and service urls initialized", func(t *testing.T) {
		cfg := config.ServiceConfig{}
		svc, err := New("bridge3", cfg, nil, "proj")
		require.NoError(t, err)
		assert.NotNil(t, svc.Ports())
		assert.Empty(t, svc.Ports())
		assert.NotNil(t, svc.URLs())
		assert.Empty(t, svc.URLs())
	})
}

func TestFactory(t *testing.T) {
	cfg := config.ServiceConfig{
		Chains: []string{"eth"},
	}
	svc, err := Factory("factory-bridge", cfg, nil, "proj")
	require.NoError(t, err)
	require.NotNil(t, svc)
	assert.Equal(t, "factory-bridge", svc.Name())
	assert.Equal(t, "mock-bridge", svc.Type())
}

func TestHealthNoURL(t *testing.T) {
	cfg := config.ServiceConfig{}
	svc, err := New("bridge-health", cfg, nil, "proj")
	require.NoError(t, err)

	err = svc.Health(context.Background())
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "bridge URL not available")
}

func TestCfgStored(t *testing.T) {
	cfg := config.ServiceConfig{
		Type:   "bridge",
		Chains: []string{"a", "b"},
		Environment: map[string]string{
			"FOO": "bar",
		},
	}
	svc, err := New("bridge-cfg", cfg, nil, "proj")
	require.NoError(t, err)
	assert.Equal(t, cfg, svc.cfg)
}
