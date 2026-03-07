package types

import (
	"encoding/json"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gopkg.in/yaml.v3"
)

func TestEnvironmentConfig_JSONRoundTrip(t *testing.T) {
	original := EnvironmentConfig{
		Name: "dev",
		Settings: SettingsConfig{
			Runtime:        "docker",
			LogLevel:       "debug",
			AccountBalance: "10000",
		},
		Chains: map[string]ChainConfig{
			"mainnet": {
				Engine:    "anvil",
				ChainID:   1,
				BlockTime: "12s",
				Accounts:  10,
			},
		},
		Services: map[string]ServiceConfig{
			"explorer": {
				Type:    "blockscout",
				Enabled: true,
				Image:   "blockscout/blockscout:latest",
			},
		},
	}

	data, err := json.Marshal(original)
	require.NoError(t, err)

	var decoded EnvironmentConfig
	err = json.Unmarshal(data, &decoded)
	require.NoError(t, err)

	assert.Equal(t, original, decoded)
}

func TestEnvironmentConfig_YAMLRoundTrip(t *testing.T) {
	original := EnvironmentConfig{
		Name: "staging",
		Settings: SettingsConfig{
			Runtime:  "docker",
			LogLevel: "info",
		},
		Chains: map[string]ChainConfig{
			"l1": {
				Engine:  "anvil",
				ChainID: 31337,
			},
		},
	}

	data, err := yaml.Marshal(original)
	require.NoError(t, err)

	var decoded EnvironmentConfig
	err = yaml.Unmarshal(data, &decoded)
	require.NoError(t, err)

	assert.Equal(t, original, decoded)
}

func TestEnvironmentConfig_JSONFieldNames(t *testing.T) {
	ec := EnvironmentConfig{
		Name:     "test",
		Settings: SettingsConfig{Runtime: "docker", LogLevel: "warn"},
		Chains:   map[string]ChainConfig{"c": {Engine: "anvil"}},
		Services: map[string]ServiceConfig{"s": {Type: "explorer"}},
	}

	data, err := json.Marshal(ec)
	require.NoError(t, err)

	var raw map[string]interface{}
	err = json.Unmarshal(data, &raw)
	require.NoError(t, err)

	assert.Contains(t, raw, "name")
	assert.Contains(t, raw, "settings")
	assert.Contains(t, raw, "chains")
	assert.Contains(t, raw, "services")
}

func TestEnvironmentConfig_ServicesOmitEmpty(t *testing.T) {
	ec := EnvironmentConfig{
		Name:     "test",
		Settings: SettingsConfig{Runtime: "docker"},
		Chains:   map[string]ChainConfig{},
	}

	data, err := json.Marshal(ec)
	require.NoError(t, err)

	var raw map[string]interface{}
	err = json.Unmarshal(data, &raw)
	require.NoError(t, err)

	assert.NotContains(t, raw, "services")
}

func TestEnvironmentConfig_MultipleChains(t *testing.T) {
	ec := EnvironmentConfig{
		Name: "multi",
		Chains: map[string]ChainConfig{
			"l1":  {Engine: "anvil", ChainID: 1},
			"l2":  {Engine: "anvil", ChainID: 10},
			"arb": {Engine: "anvil", ChainID: 42161},
		},
	}

	data, err := json.Marshal(ec)
	require.NoError(t, err)

	var decoded EnvironmentConfig
	err = json.Unmarshal(data, &decoded)
	require.NoError(t, err)

	assert.Len(t, decoded.Chains, 3)
	assert.Equal(t, uint64(1), decoded.Chains["l1"].ChainID)
	assert.Equal(t, uint64(10), decoded.Chains["l2"].ChainID)
	assert.Equal(t, uint64(42161), decoded.Chains["arb"].ChainID)
}

func TestSettingsConfig_JSONRoundTrip(t *testing.T) {
	original := SettingsConfig{
		Runtime:        "docker",
		LogLevel:       "debug",
		AccountBalance: "1000000",
	}

	data, err := json.Marshal(original)
	require.NoError(t, err)

	var decoded SettingsConfig
	err = json.Unmarshal(data, &decoded)
	require.NoError(t, err)

	assert.Equal(t, original, decoded)
}

func TestSettingsConfig_YAMLFieldNames(t *testing.T) {
	sc := SettingsConfig{
		Runtime:        "docker",
		LogLevel:       "info",
		AccountBalance: "100",
	}

	data, err := yaml.Marshal(sc)
	require.NoError(t, err)
	yamlStr := string(data)

	assert.Contains(t, yamlStr, "runtime:")
	assert.Contains(t, yamlStr, "log_level:")
	assert.Contains(t, yamlStr, "balance:")
}

func TestSettingsConfig_AccountBalanceOmitEmpty(t *testing.T) {
	sc := SettingsConfig{
		Runtime:  "docker",
		LogLevel: "info",
	}

	data, err := json.Marshal(sc)
	require.NoError(t, err)

	var raw map[string]interface{}
	err = json.Unmarshal(data, &raw)
	require.NoError(t, err)

	assert.NotContains(t, raw, "account_balance")
}

func TestSettingsConfig_ZeroValue(t *testing.T) {
	var sc SettingsConfig
	assert.Empty(t, sc.Runtime)
	assert.Empty(t, sc.LogLevel)
	assert.Empty(t, sc.AccountBalance)
}

func TestChainConfig_JSONRoundTrip(t *testing.T) {
	original := ChainConfig{
		Engine:    "anvil",
		ChainID:   31337,
		BlockTime: "2s",
		Accounts:  20,
	}

	data, err := json.Marshal(original)
	require.NoError(t, err)

	var decoded ChainConfig
	err = json.Unmarshal(data, &decoded)
	require.NoError(t, err)

	assert.Equal(t, original, decoded)
}

func TestChainConfig_YAMLRoundTrip(t *testing.T) {
	original := ChainConfig{
		Engine:    "hardhat",
		ChainID:   1337,
		BlockTime: "1s",
		Accounts:  5,
	}

	data, err := yaml.Marshal(original)
	require.NoError(t, err)

	var decoded ChainConfig
	err = yaml.Unmarshal(data, &decoded)
	require.NoError(t, err)

	assert.Equal(t, original, decoded)
}

func TestChainConfig_OptionalFieldsOmitEmpty(t *testing.T) {
	cc := ChainConfig{
		Engine:  "anvil",
		ChainID: 1,
	}

	data, err := json.Marshal(cc)
	require.NoError(t, err)

	var raw map[string]interface{}
	err = json.Unmarshal(data, &raw)
	require.NoError(t, err)

	assert.Contains(t, raw, "engine")
	assert.Contains(t, raw, "chain_id")
	assert.NotContains(t, raw, "block_time")
	assert.NotContains(t, raw, "accounts")
}

func TestChainConfig_ZeroValue(t *testing.T) {
	var cc ChainConfig
	assert.Empty(t, cc.Engine)
	assert.Equal(t, uint64(0), cc.ChainID)
	assert.Empty(t, cc.BlockTime)
	assert.Equal(t, 0, cc.Accounts)
}

func TestServiceConfig_JSONRoundTrip(t *testing.T) {
	original := ServiceConfig{
		Type:    "blockscout",
		Enabled: true,
		Image:   "blockscout/blockscout:v5",
	}

	data, err := json.Marshal(original)
	require.NoError(t, err)

	var decoded ServiceConfig
	err = json.Unmarshal(data, &decoded)
	require.NoError(t, err)

	assert.Equal(t, original, decoded)
}

func TestServiceConfig_YAMLRoundTrip(t *testing.T) {
	original := ServiceConfig{
		Type:    "ipfs",
		Enabled: false,
		Image:   "ipfs/go-ipfs:latest",
	}

	data, err := yaml.Marshal(original)
	require.NoError(t, err)

	var decoded ServiceConfig
	err = yaml.Unmarshal(data, &decoded)
	require.NoError(t, err)

	assert.Equal(t, original, decoded)
}

func TestServiceConfig_ImageOmitEmpty(t *testing.T) {
	sc := ServiceConfig{
		Type:    "test",
		Enabled: true,
	}

	data, err := json.Marshal(sc)
	require.NoError(t, err)

	var raw map[string]interface{}
	err = json.Unmarshal(data, &raw)
	require.NoError(t, err)

	assert.NotContains(t, raw, "image")
}

func TestServiceConfig_ZeroValue(t *testing.T) {
	var sc ServiceConfig
	assert.Empty(t, sc.Type)
	assert.False(t, sc.Enabled)
	assert.Empty(t, sc.Image)
}

func TestEnvironment_JSONRoundTrip(t *testing.T) {
	now := time.Now().Truncate(time.Second)
	original := Environment{
		ID:     "env-123",
		Name:   "production",
		Status: "running",
		Region: "us-east-1",
		RPCEndpoints: map[string]string{
			"l1": "https://rpc.example.com/l1",
			"l2": "https://rpc.example.com/l2",
		},
		CreatedAt: now,
		UpdatedAt: now.Add(time.Hour),
	}

	data, err := json.Marshal(original)
	require.NoError(t, err)

	var decoded Environment
	err = json.Unmarshal(data, &decoded)
	require.NoError(t, err)

	assert.Equal(t, original.ID, decoded.ID)
	assert.Equal(t, original.Name, decoded.Name)
	assert.Equal(t, original.Status, decoded.Status)
	assert.Equal(t, original.Region, decoded.Region)
	assert.Equal(t, original.RPCEndpoints, decoded.RPCEndpoints)
	assert.True(t, original.CreatedAt.Equal(decoded.CreatedAt))
	assert.True(t, original.UpdatedAt.Equal(decoded.UpdatedAt))
}

func TestEnvironment_JSONFieldNames(t *testing.T) {
	env := Environment{
		ID:           "1",
		Name:         "test",
		Status:       "stopped",
		Region:       "eu-west-1",
		RPCEndpoints: map[string]string{"a": "b"},
		CreatedAt:    time.Now(),
		UpdatedAt:    time.Now(),
	}

	data, err := json.Marshal(env)
	require.NoError(t, err)

	var raw map[string]interface{}
	err = json.Unmarshal(data, &raw)
	require.NoError(t, err)

	assert.Contains(t, raw, "id")
	assert.Contains(t, raw, "name")
	assert.Contains(t, raw, "status")
	assert.Contains(t, raw, "region")
	assert.Contains(t, raw, "rpc_endpoints")
	assert.Contains(t, raw, "created_at")
	assert.Contains(t, raw, "updated_at")
}

func TestEnvironment_RPCEndpointsOmitEmpty(t *testing.T) {
	env := Environment{
		ID:     "1",
		Name:   "test",
		Status: "stopped",
	}

	data, err := json.Marshal(env)
	require.NoError(t, err)

	var raw map[string]interface{}
	err = json.Unmarshal(data, &raw)
	require.NoError(t, err)

	assert.NotContains(t, raw, "rpc_endpoints")
}

func TestEnvironment_ZeroValue(t *testing.T) {
	var env Environment
	assert.Empty(t, env.ID)
	assert.Empty(t, env.Name)
	assert.Empty(t, env.Status)
	assert.Empty(t, env.Region)
	assert.Nil(t, env.RPCEndpoints)
	assert.True(t, env.CreatedAt.IsZero())
	assert.True(t, env.UpdatedAt.IsZero())
}

func TestSnapshot_JSONRoundTrip(t *testing.T) {
	now := time.Now().Truncate(time.Second)
	original := Snapshot{
		ID:          "snap-abc",
		Name:        "before-deploy",
		Description: "Snapshot before contract deployment",
		Tags:        []string{"deploy", "v1.0"},
		SizeBytes:   1024 * 1024,
		CreatedAt:   now,
	}

	data, err := json.Marshal(original)
	require.NoError(t, err)

	var decoded Snapshot
	err = json.Unmarshal(data, &decoded)
	require.NoError(t, err)

	assert.Equal(t, original.ID, decoded.ID)
	assert.Equal(t, original.Name, decoded.Name)
	assert.Equal(t, original.Description, decoded.Description)
	assert.Equal(t, original.Tags, decoded.Tags)
	assert.Equal(t, original.SizeBytes, decoded.SizeBytes)
	assert.True(t, original.CreatedAt.Equal(decoded.CreatedAt))
}

func TestSnapshot_JSONFieldNames(t *testing.T) {
	snap := Snapshot{
		ID:          "1",
		Name:        "test",
		Description: "desc",
		Tags:        []string{"a"},
		SizeBytes:   100,
		CreatedAt:   time.Now(),
	}

	data, err := json.Marshal(snap)
	require.NoError(t, err)

	var raw map[string]interface{}
	err = json.Unmarshal(data, &raw)
	require.NoError(t, err)

	assert.Contains(t, raw, "id")
	assert.Contains(t, raw, "name")
	assert.Contains(t, raw, "description")
	assert.Contains(t, raw, "tags")
	assert.Contains(t, raw, "size_bytes")
	assert.Contains(t, raw, "created_at")
}

func TestSnapshot_OptionalFieldsOmitEmpty(t *testing.T) {
	snap := Snapshot{
		ID:        "1",
		Name:      "minimal",
		SizeBytes: 0,
	}

	data, err := json.Marshal(snap)
	require.NoError(t, err)

	var raw map[string]interface{}
	err = json.Unmarshal(data, &raw)
	require.NoError(t, err)

	assert.NotContains(t, raw, "description")
	assert.NotContains(t, raw, "tags")
}

func TestSnapshot_ZeroValue(t *testing.T) {
	var snap Snapshot
	assert.Empty(t, snap.ID)
	assert.Empty(t, snap.Name)
	assert.Empty(t, snap.Description)
	assert.Nil(t, snap.Tags)
	assert.Equal(t, int64(0), snap.SizeBytes)
	assert.True(t, snap.CreatedAt.IsZero())
}

func TestSnapshot_EmptyTags(t *testing.T) {
	snap := Snapshot{
		ID:   "1",
		Name: "test",
		Tags: []string{},
	}

	data, err := json.Marshal(snap)
	require.NoError(t, err)

	var decoded Snapshot
	err = json.Unmarshal(data, &decoded)
	require.NoError(t, err)

	assert.Nil(t, decoded.Tags)
}
