package config

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func validBaseConfig() *Config {
	return &Config{
		Version: "1.0",
		Name:    "test-env",
		Settings: Settings{
			Runtime:   "docker",
			LogLevel:  "info",
			BlockTime: "2s",
			Accounts:  10,
		},
	}
}

func TestValidate_ValidMinimalConfig(t *testing.T) {
	cfg := validBaseConfig()
	err := Validate(cfg)
	assert.NoError(t, err)
}

func TestValidate_ValidWithChain(t *testing.T) {
	cfg := validBaseConfig()
	cfg.Chains = map[string]ChainConfig{
		"local": {
			Engine:   "anvil",
			Hardfork: "cancun",
			Mining:   MiningConfig{Mode: "auto"},
		},
	}
	err := Validate(cfg)
	assert.NoError(t, err)
}

func TestValidate_ValidWithAllRuntimes(t *testing.T) {
	for _, rt := range []string{"docker", "podman"} {
		t.Run(rt, func(t *testing.T) {
			cfg := validBaseConfig()
			cfg.Settings.Runtime = rt
			err := Validate(cfg)
			assert.NoError(t, err)
		})
	}
}

func TestValidate_ValidWithAllLogLevels(t *testing.T) {
	for _, lvl := range []string{"debug", "info", "warn", "error"} {
		t.Run(lvl, func(t *testing.T) {
			cfg := validBaseConfig()
			cfg.Settings.LogLevel = lvl
			err := Validate(cfg)
			assert.NoError(t, err)
		})
	}
}

func TestValidate_ValidWithAllEngines(t *testing.T) {
	for _, engine := range []string{"anvil", "hardhat", "geth"} {
		t.Run(engine, func(t *testing.T) {
			cfg := validBaseConfig()
			cfg.Chains = map[string]ChainConfig{
				"c": {Engine: engine},
			}
			err := Validate(cfg)
			assert.NoError(t, err)
		})
	}
}

func TestValidate_MissingVersion_ReturnsError(t *testing.T) {
	cfg := validBaseConfig()
	cfg.Version = ""
	err := Validate(cfg)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "version is required")
}

func TestValidate_MissingName_ReturnsError(t *testing.T) {
	cfg := validBaseConfig()
	cfg.Name = ""
	err := Validate(cfg)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "name is required")
}

func TestValidate_InvalidRuntime_ReturnsError(t *testing.T) {
	cfg := validBaseConfig()
	cfg.Settings.Runtime = "kubernetes"
	err := Validate(cfg)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "invalid runtime")
	assert.Contains(t, err.Error(), "kubernetes")
}

func TestValidate_InvalidLogLevel_ReturnsError(t *testing.T) {
	cfg := validBaseConfig()
	cfg.Settings.LogLevel = "verbose"
	err := Validate(cfg)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "invalid log_level")
}

func TestValidate_InvalidBlockTime_ReturnsError(t *testing.T) {
	cfg := validBaseConfig()
	cfg.Settings.BlockTime = "not-a-duration"
	err := Validate(cfg)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "block_time")
}

func TestValidate_ZeroAccounts_ReturnsError(t *testing.T) {
	cfg := validBaseConfig()
	cfg.Settings.Accounts = 0
	err := Validate(cfg)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "accounts must be at least 1")
}

func TestValidate_NegativeAccounts_ReturnsError(t *testing.T) {
	cfg := validBaseConfig()
	cfg.Settings.Accounts = -1
	err := Validate(cfg)
	require.Error(t, err)
}

func TestValidate_InvalidChainEngine_ReturnsError(t *testing.T) {
	cfg := validBaseConfig()
	cfg.Chains = map[string]ChainConfig{
		"mychain": {Engine: "besu"},
	}
	err := Validate(cfg)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "invalid engine")
	assert.Contains(t, err.Error(), "besu")
}

func TestValidate_InvalidHardfork_ReturnsError(t *testing.T) {
	cfg := validBaseConfig()
	cfg.Chains = map[string]ChainConfig{
		"mychain": {Engine: "anvil", Hardfork: "frontier"},
	}
	err := Validate(cfg)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "invalid hardfork")
}

func TestValidate_ValidHardforks_NoError(t *testing.T) {
	for _, hf := range []string{"london", "shanghai", "cancun"} {
		t.Run(hf, func(t *testing.T) {
			cfg := validBaseConfig()
			cfg.Chains = map[string]ChainConfig{
				"c": {Engine: "anvil", Hardfork: hf},
			}
			err := Validate(cfg)
			assert.NoError(t, err)
		})
	}
}

func TestValidate_InvalidMiningMode_ReturnsError(t *testing.T) {
	cfg := validBaseConfig()
	cfg.Chains = map[string]ChainConfig{
		"mychain": {Engine: "anvil", Mining: MiningConfig{Mode: "burst"}},
	}
	err := Validate(cfg)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "invalid mining mode")
}

func TestValidate_InvalidChainBlockTime_ReturnsError(t *testing.T) {
	cfg := validBaseConfig()
	cfg.Chains = map[string]ChainConfig{
		"mychain": {Engine: "anvil", BlockTime: "bad-time"},
	}
	err := Validate(cfg)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "block_time")
}

func TestValidate_InvalidServiceType_ReturnsError(t *testing.T) {
	cfg := validBaseConfig()
	cfg.Services = map[string]ServiceConfig{
		"mysvc": {Type: "unknown-type"},
	}
	err := Validate(cfg)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "invalid type")
}

func TestValidate_ValidServiceTypes_NoError(t *testing.T) {
	validTypes := []string{
		"ipfs", "subgraph", "ponder",
		"blockscout", "otterscan",
		"chainlink-mock", "pyth-mock",
		"grafana", "prometheus",
		"faucet", "mock-bridge", "custom",
	}
	for _, typ := range validTypes {
		t.Run(typ, func(t *testing.T) {
			cfg := validBaseConfig()
			cfg.Services = map[string]ServiceConfig{
				"svc": {Type: typ},
			}
			err := Validate(cfg)
			assert.NoError(t, err)
		})
	}
}

func TestValidate_ServiceReferencesUndefinedChain_ReturnsError(t *testing.T) {
	cfg := validBaseConfig()
	cfg.Services = map[string]ServiceConfig{
		"svc": {Type: "ipfs", Chain: "nonexistent"},
	}
	err := Validate(cfg)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "undefined chain")
}

func TestValidate_ServiceChainReferenceValid_NoError(t *testing.T) {
	cfg := validBaseConfig()
	cfg.Chains = map[string]ChainConfig{
		"local": {Engine: "anvil"},
	}
	cfg.Services = map[string]ServiceConfig{
		"svc": {Type: "ipfs", Chain: "local"},
	}
	err := Validate(cfg)
	assert.NoError(t, err)
}

func TestValidate_ServiceChainsSliceReferencesUndefinedChain_ReturnsError(t *testing.T) {
	cfg := validBaseConfig()
	cfg.Services = map[string]ServiceConfig{
		"svc": {Type: "mock-bridge", Chains: []string{"missing-chain"}},
	}
	err := Validate(cfg)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "undefined chain")
}

func TestValidate_ServiceDependsOnUndefined_ReturnsError(t *testing.T) {
	cfg := validBaseConfig()
	cfg.Services = map[string]ServiceConfig{
		"svc": {Type: "ipfs", DependsOn: []string{"nonexistent-service"}},
	}
	err := Validate(cfg)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "undefined service/chain")
}

func TestValidate_ServiceDependsOnExistingService_NoError(t *testing.T) {
	cfg := validBaseConfig()
	cfg.Services = map[string]ServiceConfig{
		"alpha": {Type: "ipfs"},
		"beta":  {Type: "subgraph", DependsOn: []string{"alpha"}},
	}
	err := Validate(cfg)
	assert.NoError(t, err)
}

func TestValidate_ServiceDependsOnChain_NoError(t *testing.T) {
	cfg := validBaseConfig()
	cfg.Chains = map[string]ChainConfig{
		"local": {Engine: "anvil"},
	}
	cfg.Services = map[string]ServiceConfig{
		"svc": {Type: "ipfs", DependsOn: []string{"local"}},
	}
	err := Validate(cfg)
	assert.NoError(t, err)
}

func TestValidate_PortConflict_ReturnsError(t *testing.T) {
	cfg := validBaseConfig()
	cfg.Services = map[string]ServiceConfig{
		"svcA": {Type: "ipfs", Port: 8080},
		"svcB": {Type: "prometheus", Port: 8080},
	}
	err := Validate(cfg)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "8080")
}

func TestValidate_NoPortConflict_NoError(t *testing.T) {
	cfg := validBaseConfig()
	cfg.Services = map[string]ServiceConfig{
		"svcA": {Type: "ipfs", Port: 5001},
		"svcB": {Type: "prometheus", Port: 9090},
	}
	err := Validate(cfg)
	assert.NoError(t, err)
}

func TestValidate_APIPortConflict_ReturnsError(t *testing.T) {
	cfg := validBaseConfig()
	cfg.Services = map[string]ServiceConfig{
		"svcA": {Type: "ipfs", APIPort: 5001},
		"svcB": {Type: "subgraph", APIPort: 5001},
	}
	err := Validate(cfg)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "5001")
}

func TestValidate_ZeroPorts_NoConflict(t *testing.T) {
	cfg := validBaseConfig()
	cfg.Services = map[string]ServiceConfig{
		"svcA": {Type: "ipfs"},   // Port = 0
		"svcB": {Type: "custom"}, // Port = 0
	}
	err := Validate(cfg)
	assert.NoError(t, err)
}

func TestValidate_DependencyCycle_ReturnsError(t *testing.T) {
	cfg := validBaseConfig()
	cfg.Services = map[string]ServiceConfig{
		"alpha": {Type: "ipfs", DependsOn: []string{"beta"}},
		"beta":  {Type: "subgraph", DependsOn: []string{"alpha"}},
	}
	err := Validate(cfg)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "cycle")
}

func TestValidate_NoCycle_NoError(t *testing.T) {
	cfg := validBaseConfig()
	cfg.Services = map[string]ServiceConfig{
		"alpha": {Type: "ipfs"},
		"beta":  {Type: "subgraph", DependsOn: []string{"alpha"}},
		"gamma": {Type: "grafana", DependsOn: []string{"beta"}},
	}
	err := Validate(cfg)
	assert.NoError(t, err)
}

func TestValidate_PluginMissingVersion_ReturnsError(t *testing.T) {
	cfg := validBaseConfig()
	cfg.Plugins = map[string]PluginConfig{
		"myplugin": {Version: ""},
	}
	err := Validate(cfg)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "version is required")
}

func TestValidate_PluginInvalidVersionConstraint_ReturnsError(t *testing.T) {
	cfg := validBaseConfig()
	cfg.Plugins = map[string]PluginConfig{
		"myplugin": {Version: "not-a-version"},
	}
	err := Validate(cfg)
	require.Error(t, err)
}

func TestValidate_PluginValidVersionConstraint_NoError(t *testing.T) {
	cfg := validBaseConfig()
	cfg.Plugins = map[string]PluginConfig{
		"myplugin": {Version: "^1.0.0"},
	}
	err := Validate(cfg)
	assert.NoError(t, err)
}

func TestValidate_ErrorIsDokryptError(t *testing.T) {
	cfg := validBaseConfig()
	cfg.Version = ""

	err := Validate(cfg)
	require.Error(t, err)

	assert.Contains(t, err.Error(), "CONFIG_VALIDATION_FAILED")
}

func TestValidate_MultipleErrors_AllReported(t *testing.T) {
	cfg := &Config{
		Settings: Settings{
			Runtime:   "k8s",
			LogLevel:  "trace",
			BlockTime: "2s",
			Accounts:  10,
		},
	}

	err := Validate(cfg)
	require.Error(t, err)

	msg := err.Error()
	assert.Contains(t, msg, "version is required")
	assert.Contains(t, msg, "name is required")
	assert.Contains(t, msg, "invalid runtime")
}

func TestValidate_GatewayPortConflict_ReturnsError(t *testing.T) {
	cfg := validBaseConfig()
	cfg.Services = map[string]ServiceConfig{
		"svcA": {Type: "ipfs", GatewayPort: 8080},
		"svcB": {Type: "custom", GatewayPort: 8080},
	}
	err := Validate(cfg)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "8080")
}

func TestValidate_GraphQLPortConflict_ReturnsError(t *testing.T) {
	cfg := validBaseConfig()
	cfg.Services = map[string]ServiceConfig{
		"svcA": {Type: "subgraph", GraphQLPort: 8000},
		"svcB": {Type: "ponder", GraphQLPort: 8000},
	}
	err := Validate(cfg)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "8000")
}

func TestValidate_AdminPortConflict_ReturnsError(t *testing.T) {
	cfg := validBaseConfig()
	cfg.Services = map[string]ServiceConfig{
		"svcA": {Type: "subgraph", AdminPort: 9090},
		"svcB": {Type: "custom", AdminPort: 9090},
	}
	err := Validate(cfg)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "9090")
}

func TestValidate_CrossTypePortConflict_ReturnsError(t *testing.T) {
	cfg := validBaseConfig()
	cfg.Services = map[string]ServiceConfig{
		"svcA": {Type: "ipfs", Port: 5001},
		"svcB": {Type: "custom", APIPort: 5001},
	}
	err := Validate(cfg)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "5001")
}

func TestValidate_PortsMapConflict_ReturnsError(t *testing.T) {
	cfg := validBaseConfig()
	cfg.Services = map[string]ServiceConfig{
		"svcA": {Type: "custom", Ports: map[string]int{"http": 3000}},
		"svcB": {Type: "grafana", Port: 3000},
	}
	err := Validate(cfg)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "3000")
}

func TestValidate_PortsMapNoConflict_NoError(t *testing.T) {
	cfg := validBaseConfig()
	cfg.Services = map[string]ServiceConfig{
		"svcA": {Type: "custom", Ports: map[string]int{"http": 3000, "grpc": 3001}},
		"svcB": {Type: "grafana", Port: 3002},
	}
	err := Validate(cfg)
	assert.NoError(t, err)
}

func TestValidate_SelfDependencyCycle_ReturnsError(t *testing.T) {
	cfg := validBaseConfig()
	cfg.Services = map[string]ServiceConfig{
		"alpha": {Type: "ipfs", DependsOn: []string{"alpha"}},
	}
	err := Validate(cfg)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "cycle")
}

func TestValidate_ThreeNodeCycle_ReturnsError(t *testing.T) {
	cfg := validBaseConfig()
	cfg.Services = map[string]ServiceConfig{
		"a": {Type: "ipfs", DependsOn: []string{"b"}},
		"b": {Type: "subgraph", DependsOn: []string{"c"}},
		"c": {Type: "grafana", DependsOn: []string{"a"}},
	}
	err := Validate(cfg)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "cycle")
}

func TestValidate_ChainFieldCreatesDependencyEdge(t *testing.T) {
	cfg := validBaseConfig()
	cfg.Chains = map[string]ChainConfig{
		"local": {Engine: "anvil"},
	}
	cfg.Services = map[string]ServiceConfig{
		"indexer":  {Type: "subgraph", Chain: "local", DependsOn: []string{"ipfs"}},
		"ipfs":     {Type: "ipfs"},
	}
	err := Validate(cfg)
	assert.NoError(t, err)
}

func TestValidate_EmptyHardfork_NoError(t *testing.T) {
	cfg := validBaseConfig()
	cfg.Chains = map[string]ChainConfig{
		"c": {Engine: "anvil", Hardfork: ""},
	}
	err := Validate(cfg)
	assert.NoError(t, err)
}

func TestValidate_EmptyMiningMode_NoError(t *testing.T) {
	cfg := validBaseConfig()
	cfg.Chains = map[string]ChainConfig{
		"c": {Engine: "anvil", Mining: MiningConfig{Mode: ""}},
	}
	err := Validate(cfg)
	assert.NoError(t, err)
}

func TestValidate_ValidMiningModes_NoError(t *testing.T) {
	for _, mode := range []string{"auto", "interval", "manual"} {
		t.Run(mode, func(t *testing.T) {
			cfg := validBaseConfig()
			cfg.Chains = map[string]ChainConfig{
				"c": {Engine: "anvil", Mining: MiningConfig{Mode: mode}},
			}
			err := Validate(cfg)
			assert.NoError(t, err)
		})
	}
}

func TestValidate_ChainBlockTime_ValidDuration_NoError(t *testing.T) {
	cfg := validBaseConfig()
	cfg.Chains = map[string]ChainConfig{
		"c": {Engine: "anvil", BlockTime: "500ms"},
	}
	err := Validate(cfg)
	assert.NoError(t, err)
}

func TestValidate_EmptyChainBlockTime_NoError(t *testing.T) {
	cfg := validBaseConfig()
	cfg.Chains = map[string]ChainConfig{
		"c": {Engine: "anvil", BlockTime: ""},
	}
	err := Validate(cfg)
	assert.NoError(t, err)
}

func TestValidate_MultipleChainsWithErrors_AllReported(t *testing.T) {
	cfg := validBaseConfig()
	cfg.Chains = map[string]ChainConfig{
		"a": {Engine: "invalid-engine"},
		"b": {Engine: "anvil", Hardfork: "invalid-hardfork"},
	}
	err := Validate(cfg)
	require.Error(t, err)
	msg := err.Error()
	assert.Contains(t, msg, "invalid engine")
	assert.Contains(t, msg, "invalid hardfork")
}

func TestValidate_ServiceChainsSlice_AllValid_NoError(t *testing.T) {
	cfg := validBaseConfig()
	cfg.Chains = map[string]ChainConfig{
		"eth":     {Engine: "anvil"},
		"polygon": {Engine: "hardhat"},
	}
	cfg.Services = map[string]ServiceConfig{
		"bridge": {Type: "mock-bridge", Chains: []string{"eth", "polygon"}},
	}
	err := Validate(cfg)
	assert.NoError(t, err)
}

func TestValidate_ServiceChainsSlice_PartiallyInvalid_ReturnsError(t *testing.T) {
	cfg := validBaseConfig()
	cfg.Chains = map[string]ChainConfig{
		"eth": {Engine: "anvil"},
	}
	cfg.Services = map[string]ServiceConfig{
		"bridge": {Type: "mock-bridge", Chains: []string{"eth", "missing"}},
	}
	err := Validate(cfg)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "missing")
}

func TestValidate_ServiceMultipleDeps_OneMissing_ReturnsError(t *testing.T) {
	cfg := validBaseConfig()
	cfg.Services = map[string]ServiceConfig{
		"alpha": {Type: "ipfs"},
		"gamma": {Type: "grafana", DependsOn: []string{"alpha", "nonexistent"}},
	}
	err := Validate(cfg)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "nonexistent")
}

func TestValidate_PluginTildeVersion_NoError(t *testing.T) {
	cfg := validBaseConfig()
	cfg.Plugins = map[string]PluginConfig{
		"myplugin": {Version: "~1.2.3"},
	}
	err := Validate(cfg)
	assert.NoError(t, err)
}

func TestValidate_NoChainsNoServicesNoPlugins_NoError(t *testing.T) {
	cfg := validBaseConfig()
	err := Validate(cfg)
	assert.NoError(t, err)
}

func TestValidate_NilMaps_NoError(t *testing.T) {
	cfg := validBaseConfig()
	cfg.Chains = nil
	cfg.Services = nil
	cfg.Plugins = nil
	err := Validate(cfg)
	assert.NoError(t, err)
}

func TestValidate_AccountsExactlyOne_NoError(t *testing.T) {
	cfg := validBaseConfig()
	cfg.Settings.Accounts = 1
	err := Validate(cfg)
	assert.NoError(t, err)
}

func TestValidate_CombinedChainServicePluginErrors(t *testing.T) {
	cfg := validBaseConfig()
	cfg.Chains = map[string]ChainConfig{
		"c": {Engine: "bad-engine"},
	}
	cfg.Services = map[string]ServiceConfig{
		"s": {Type: "bad-type"},
	}
	cfg.Plugins = map[string]PluginConfig{
		"p": {Version: ""},
	}

	err := Validate(cfg)
	require.Error(t, err)
	msg := err.Error()
	assert.Contains(t, msg, "invalid engine")
	assert.Contains(t, msg, "invalid type")
	assert.Contains(t, msg, "version is required")
}

func TestValidate_EmptySettingsBlockTime_NoError(t *testing.T) {
	cfg := validBaseConfig()
	cfg.Settings.BlockTime = ""
	err := Validate(cfg)
	assert.NoError(t, err)
}

func TestValidate_ManyServicesNoPorts_NoError(t *testing.T) {
	cfg := validBaseConfig()
	cfg.Services = map[string]ServiceConfig{
		"a": {Type: "ipfs"},
		"b": {Type: "custom"},
		"c": {Type: "grafana"},
		"d": {Type: "prometheus"},
		"e": {Type: "faucet"},
	}
	err := Validate(cfg)
	assert.NoError(t, err)
}

func TestValidate_DeepDependencyChain_NoCycle_NoError(t *testing.T) {
	cfg := validBaseConfig()
	cfg.Services = map[string]ServiceConfig{
		"a": {Type: "ipfs"},
		"b": {Type: "subgraph", DependsOn: []string{"a"}},
		"c": {Type: "grafana", DependsOn: []string{"b"}},
		"d": {Type: "prometheus", DependsOn: []string{"c"}},
		"e": {Type: "faucet", DependsOn: []string{"d"}},
	}
	err := Validate(cfg)
	assert.NoError(t, err)
}
