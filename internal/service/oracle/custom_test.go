package oracle

import (
	"context"
	"testing"

	"github.com/dokrypt/dokrypt/internal/config"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewCustomOracle(t *testing.T) {
	t.Run("basic creation", func(t *testing.T) {
		cfg := config.ServiceConfig{
			Type:  "oracle",
			Image: "my-oracle:latest",
		}
		svc, err := NewCustomOracle("my-oracle", cfg, nil, "testproj")
		require.NoError(t, err)
		require.NotNil(t, svc)

		assert.Equal(t, "my-oracle", svc.Name())
		assert.Equal(t, "custom-oracle", svc.Type())
		assert.Equal(t, "testproj", svc.ProjectName)
		assert.NotNil(t, svc.HostPorts)
		assert.NotNil(t, svc.ServiceURLs)
	})

	t.Run("dependencies from config", func(t *testing.T) {
		cfg := config.ServiceConfig{
			DependsOn: []string{"chain1", "chain2"},
		}
		svc, err := NewCustomOracle("co1", cfg, nil, "proj")
		require.NoError(t, err)

		deps := svc.DependsOn()
		assert.Equal(t, []string{"chain1", "chain2"}, deps)
	})

	t.Run("empty config", func(t *testing.T) {
		cfg := config.ServiceConfig{}
		svc, err := NewCustomOracle("co2", cfg, nil, "proj")
		require.NoError(t, err)
		assert.Empty(t, svc.DependsOn())
	})
}

func TestCustomOracleFactory(t *testing.T) {
	cfg := config.ServiceConfig{Image: "img"}
	svc, err := CustomOracleFactory("factory-co", cfg, nil, "proj")
	require.NoError(t, err)
	require.NotNil(t, svc)
	assert.Equal(t, "factory-co", svc.Name())
	assert.Equal(t, "custom-oracle", svc.Type())
}

func TestCustomOracleHealthAlwaysOK(t *testing.T) {
	cfg := config.ServiceConfig{}
	svc, err := NewCustomOracle("co-health", cfg, nil, "proj")
	require.NoError(t, err)

	err = svc.Health(context.Background())
	assert.NoError(t, err)
}

func TestCustomOraclePortsAndURLs(t *testing.T) {
	cfg := config.ServiceConfig{}
	svc, err := NewCustomOracle("co-ports", cfg, nil, "proj")
	require.NoError(t, err)

	assert.NotNil(t, svc.Ports())
	assert.Empty(t, svc.Ports())
	assert.NotNil(t, svc.URLs())
	assert.Empty(t, svc.URLs())
}

func TestCustomOracleCfgStored(t *testing.T) {
	cfg := config.ServiceConfig{
		Type:  "oracle",
		Image: "custom-oracle:v2",
		Port:  9090,
		Environment: map[string]string{
			"API_KEY": "test-key",
		},
	}
	svc, err := NewCustomOracle("co-cfg", cfg, nil, "proj")
	require.NoError(t, err)
	assert.Equal(t, cfg, svc.cfg)
}
