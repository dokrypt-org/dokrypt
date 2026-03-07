package ipfs

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/dokrypt/dokrypt/internal/config"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNew(t *testing.T) {
	t.Run("basic creation with defaults", func(t *testing.T) {
		cfg := config.ServiceConfig{}
		svc, err := New("my-ipfs", cfg, nil, "testproj")
		require.NoError(t, err)
		require.NotNil(t, svc)

		assert.Equal(t, "my-ipfs", svc.Name())
		assert.Equal(t, "ipfs", svc.Type())
		assert.Equal(t, "testproj", svc.ProjectName)
		assert.NotNil(t, svc.HostPorts)
		assert.NotNil(t, svc.ServiceURLs)
	})

	t.Run("custom ports", func(t *testing.T) {
		cfg := config.ServiceConfig{
			APIPort:     6001,
			GatewayPort: 9090,
		}
		svc, err := New("ipfs-custom", cfg, nil, "proj")
		require.NoError(t, err)
		require.NotNil(t, svc)

		assert.Equal(t, "ipfs-custom", svc.Name())
	})

	t.Run("dependencies from config", func(t *testing.T) {
		cfg := config.ServiceConfig{
			DependsOn: []string{"dep1", "dep2"},
		}
		svc, err := New("ipfs-deps", cfg, nil, "proj")
		require.NoError(t, err)

		deps := svc.DependsOn()
		assert.Equal(t, []string{"dep1", "dep2"}, deps)
	})

	t.Run("empty config has no deps", func(t *testing.T) {
		cfg := config.ServiceConfig{}
		svc, err := New("ipfs-nodeps", cfg, nil, "proj")
		require.NoError(t, err)
		assert.Empty(t, svc.DependsOn())
	})
}

func TestIPFSFactory(t *testing.T) {
	cfg := config.ServiceConfig{}
	svc, err := Factory("factory-ipfs", cfg, nil, "proj")
	require.NoError(t, err)
	require.NotNil(t, svc)
	assert.Equal(t, "factory-ipfs", svc.Name())
	assert.Equal(t, "ipfs", svc.Type())
}

func TestIPFSHealthNoURL(t *testing.T) {
	cfg := config.ServiceConfig{}
	svc, err := New("ipfs-health", cfg, nil, "proj")
	require.NoError(t, err)

	err = svc.Health(context.Background())
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "IPFS API URL not available")
}

func TestIPFSHealthWithServer(t *testing.T) {
	t.Run("healthy server returns ok", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			assert.Equal(t, http.MethodPost, r.Method)
			assert.Equal(t, "/api/v0/id", r.URL.Path)
			w.WriteHeader(http.StatusOK)
			w.Write([]byte(`{"ID":"test"}`))
		}))
		defer server.Close()

		cfg := config.ServiceConfig{}
		svc, err := New("ipfs-healthy", cfg, nil, "proj")
		require.NoError(t, err)

		svc.ServiceURLs["api"] = server.URL

		err = svc.Health(context.Background())
		assert.NoError(t, err)
	})

	t.Run("unhealthy server returns error", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusInternalServerError)
		}))
		defer server.Close()

		cfg := config.ServiceConfig{}
		svc, err := New("ipfs-unhealthy", cfg, nil, "proj")
		require.NoError(t, err)

		svc.ServiceURLs["api"] = server.URL

		err = svc.Health(context.Background())
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "IPFS health check returned status 500")
	})
}

func TestIPFSPortsAndURLs(t *testing.T) {
	cfg := config.ServiceConfig{}
	svc, err := New("ipfs-ports", cfg, nil, "proj")
	require.NoError(t, err)

	assert.NotNil(t, svc.Ports())
	assert.Empty(t, svc.Ports())
	assert.NotNil(t, svc.URLs())
	assert.Empty(t, svc.URLs())
}

func TestIPFSCfgStored(t *testing.T) {
	cfg := config.ServiceConfig{
		Type:        "ipfs",
		APIPort:     6001,
		GatewayPort: 9090,
		Environment: map[string]string{
			"IPFS_PROFILE": "test",
		},
	}
	svc, err := New("ipfs-cfg", cfg, nil, "proj")
	require.NoError(t, err)
	assert.Equal(t, cfg, svc.cfg)
}

func TestIPFSDefaultConstants(t *testing.T) {
	assert.Equal(t, "ipfs/kubo:latest", defaultImage)
	assert.Equal(t, 5001, defaultAPIPort)
	assert.Equal(t, 8080, defaultGatewayPort)
}
