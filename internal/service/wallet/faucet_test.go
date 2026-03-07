package wallet

import (
	"context"
	"testing"

	"github.com/dokrypt/dokrypt/internal/config"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewFaucet(t *testing.T) {
	t.Run("basic creation", func(t *testing.T) {
		cfg := config.ServiceConfig{
			Type:  "faucet",
			Chain: "ethereum",
		}
		svc, err := NewFaucet("my-faucet", cfg, nil, "testproj")
		require.NoError(t, err)
		require.NotNil(t, svc)

		assert.Equal(t, "my-faucet", svc.Name())
		assert.Equal(t, "faucet", svc.Type())
		assert.Equal(t, "testproj", svc.ProjectName)
		assert.NotNil(t, svc.HostPorts)
		assert.NotNil(t, svc.ServiceURLs)
	})

	t.Run("dependencies include chain", func(t *testing.T) {
		cfg := config.ServiceConfig{
			Chain:     "polygon",
			DependsOn: []string{"dep1"},
		}
		svc, err := NewFaucet("f1", cfg, nil, "proj")
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
		svc, err := NewFaucet("f2", cfg, nil, "proj")
		require.NoError(t, err)
		assert.Equal(t, []string{"dep1"}, svc.DependsOn())
	})

	t.Run("empty config", func(t *testing.T) {
		cfg := config.ServiceConfig{}
		svc, err := NewFaucet("f3", cfg, nil, "proj")
		require.NoError(t, err)
		assert.Empty(t, svc.DependsOn())
	})
}

func TestFaucetFactory(t *testing.T) {
	cfg := config.ServiceConfig{Chain: "eth"}
	svc, err := FaucetFactory("factory-f", cfg, nil, "proj")
	require.NoError(t, err)
	require.NotNil(t, svc)
	assert.Equal(t, "factory-f", svc.Name())
	assert.Equal(t, "faucet", svc.Type())
}

func TestFaucetHealthNoURL(t *testing.T) {
	cfg := config.ServiceConfig{}
	svc, err := NewFaucet("f-health", cfg, nil, "proj")
	require.NoError(t, err)

	err = svc.Health(context.Background())
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "faucet URL not available")
}

func TestFaucetPortsAndURLs(t *testing.T) {
	cfg := config.ServiceConfig{}
	svc, err := NewFaucet("f-ports", cfg, nil, "proj")
	require.NoError(t, err)

	assert.NotNil(t, svc.Ports())
	assert.Empty(t, svc.Ports())
	assert.NotNil(t, svc.URLs())
	assert.Empty(t, svc.URLs())
}

func TestFaucetCfgStored(t *testing.T) {
	cfg := config.ServiceConfig{
		Type:       "faucet",
		Chain:      "ethereum",
		Port:       4000,
		DripAmount: "5",
		Cooldown:   "30s",
		Environment: map[string]string{
			"CHAIN_RPC_URL": "http://custom:8545",
		},
	}
	svc, err := NewFaucet("f-cfg", cfg, nil, "proj")
	require.NoError(t, err)
	assert.Equal(t, cfg, svc.cfg)
}
