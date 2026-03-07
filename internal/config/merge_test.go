package config

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestMergeConfigs_OverrideVersion(t *testing.T) {
	base := &Config{Version: "1.0", Name: "base"}
	override := &Config{Version: "2.0"}

	result := MergeConfigs(base, override)
	assert.Equal(t, "2.0", result.Version)
}

func TestMergeConfigs_EmptyOverrideKeepsBase(t *testing.T) {
	base := &Config{Version: "1.0", Name: "base-name"}
	override := &Config{}

	result := MergeConfigs(base, override)
	assert.Equal(t, "1.0", result.Version)
	assert.Equal(t, "base-name", result.Name)
}

func TestMergeConfigs_OverrideName(t *testing.T) {
	base := &Config{Name: "old-name"}
	override := &Config{Name: "new-name"}

	result := MergeConfigs(base, override)
	assert.Equal(t, "new-name", result.Name)
}

func TestMergeConfigs_OverrideNameEmpty_KeepsBase(t *testing.T) {
	base := &Config{Name: "keep-me"}
	override := &Config{Name: ""}

	result := MergeConfigs(base, override)
	assert.Equal(t, "keep-me", result.Name)
}

func TestMergeConfigs_OverrideSettings_Runtime(t *testing.T) {
	base := &Config{Settings: Settings{Runtime: "docker"}}
	override := &Config{Settings: Settings{Runtime: "podman"}}

	result := MergeConfigs(base, override)
	assert.Equal(t, "podman", result.Settings.Runtime)
}

func TestMergeConfigs_OverrideSettings_PartialOverride(t *testing.T) {
	base := &Config{Settings: Settings{
		Runtime:  "docker",
		LogLevel: "info",
		Accounts: 10,
	}}
	override := &Config{Settings: Settings{
		LogLevel: "debug",
	}}

	result := MergeConfigs(base, override)
	assert.Equal(t, "docker", result.Settings.Runtime) // Kept from base.
	assert.Equal(t, "debug", result.Settings.LogLevel)  // Overridden.
	assert.Equal(t, 10, result.Settings.Accounts)       // Kept from base.
}

func TestMergeConfigs_OverrideSettings_BlockTime(t *testing.T) {
	base := &Config{Settings: Settings{BlockTime: "2s"}}
	override := &Config{Settings: Settings{BlockTime: "5s"}}

	result := MergeConfigs(base, override)
	assert.Equal(t, "5s", result.Settings.BlockTime)
}

func TestMergeConfigs_OverrideSettings_Accounts(t *testing.T) {
	base := &Config{Settings: Settings{Accounts: 10}}
	override := &Config{Settings: Settings{Accounts: 20}}

	result := MergeConfigs(base, override)
	assert.Equal(t, 20, result.Settings.Accounts)
}

func TestMergeConfigs_OverrideSettings_AccountBalance(t *testing.T) {
	base := &Config{Settings: Settings{AccountBalance: "10000"}}
	override := &Config{Settings: Settings{AccountBalance: "99999"}}

	result := MergeConfigs(base, override)
	assert.Equal(t, "99999", result.Settings.AccountBalance)
}

func TestMergeConfigs_OverrideSettings_Telemetry(t *testing.T) {
	base := &Config{Settings: Settings{Telemetry: false}}
	override := &Config{Settings: Settings{Telemetry: true}}

	result := MergeConfigs(base, override)
	assert.True(t, result.Settings.Telemetry)
}

func TestMergeConfigs_Chains_NewChainAdded(t *testing.T) {
	base := &Config{
		Chains: map[string]ChainConfig{
			"eth": {Engine: "anvil", ChainID: 31337},
		},
	}
	override := &Config{
		Chains: map[string]ChainConfig{
			"polygon": {Engine: "hardhat", ChainID: 137},
		},
	}

	result := MergeConfigs(base, override)
	require.Len(t, result.Chains, 2)
	assert.Equal(t, "anvil", result.Chains["eth"].Engine)
	assert.Equal(t, "hardhat", result.Chains["polygon"].Engine)
}

func TestMergeConfigs_Chains_ExistingChainMerged(t *testing.T) {
	base := &Config{
		Chains: map[string]ChainConfig{
			"eth": {Engine: "anvil", ChainID: 31337, GasLimit: 30000000},
		},
	}
	override := &Config{
		Chains: map[string]ChainConfig{
			"eth": {ChainID: 1337, GasLimit: 50000000},
		},
	}

	result := MergeConfigs(base, override)
	chain := result.Chains["eth"]
	assert.Equal(t, "anvil", chain.Engine)          // Kept from base.
	assert.Equal(t, uint64(1337), chain.ChainID)    // Overridden.
	assert.Equal(t, uint64(50000000), chain.GasLimit) // Overridden.
}

func TestMergeConfigs_Chains_OverrideFork(t *testing.T) {
	base := &Config{
		Chains: map[string]ChainConfig{
			"eth": {Engine: "anvil", Fork: nil},
		},
	}
	fork := &ForkConfig{Network: "mainnet", BlockNumber: 12345}
	override := &Config{
		Chains: map[string]ChainConfig{
			"eth": {Fork: fork},
		},
	}

	result := MergeConfigs(base, override)
	require.NotNil(t, result.Chains["eth"].Fork)
	assert.Equal(t, "mainnet", result.Chains["eth"].Fork.Network)
	assert.Equal(t, uint64(12345), result.Chains["eth"].Fork.BlockNumber)
}

func TestMergeConfigs_Chains_OverrideBlockTime(t *testing.T) {
	base := &Config{
		Chains: map[string]ChainConfig{
			"eth": {Engine: "anvil", BlockTime: "2s"},
		},
	}
	override := &Config{
		Chains: map[string]ChainConfig{
			"eth": {BlockTime: "10s"},
		},
	}

	result := MergeConfigs(base, override)
	assert.Equal(t, "10s", result.Chains["eth"].BlockTime)
}

func TestMergeConfigs_Chains_OverrideHardfork(t *testing.T) {
	base := &Config{
		Chains: map[string]ChainConfig{
			"eth": {Engine: "anvil", Hardfork: "london"},
		},
	}
	override := &Config{
		Chains: map[string]ChainConfig{
			"eth": {Hardfork: "cancun"},
		},
	}

	result := MergeConfigs(base, override)
	assert.Equal(t, "cancun", result.Chains["eth"].Hardfork)
}

func TestMergeConfigs_Chains_NilBaseMap(t *testing.T) {
	base := &Config{Chains: nil}
	override := &Config{
		Chains: map[string]ChainConfig{
			"eth": {Engine: "anvil"},
		},
	}

	result := MergeConfigs(base, override)
	require.Len(t, result.Chains, 1)
	assert.Equal(t, "anvil", result.Chains["eth"].Engine)
}

func TestMergeConfigs_Chains_NilOverride_KeepsBase(t *testing.T) {
	base := &Config{
		Chains: map[string]ChainConfig{
			"eth": {Engine: "anvil"},
		},
	}
	override := &Config{Chains: nil}

	result := MergeConfigs(base, override)
	require.Len(t, result.Chains, 1)
	assert.Equal(t, "anvil", result.Chains["eth"].Engine)
}

func TestMergeConfigs_Services_NewServiceAdded(t *testing.T) {
	base := &Config{
		Services: map[string]ServiceConfig{
			"ipfs": {Type: "ipfs", Port: 5001},
		},
	}
	override := &Config{
		Services: map[string]ServiceConfig{
			"grafana": {Type: "grafana", Port: 3000},
		},
	}

	result := MergeConfigs(base, override)
	require.Len(t, result.Services, 2)
	assert.Equal(t, "ipfs", result.Services["ipfs"].Type)
	assert.Equal(t, "grafana", result.Services["grafana"].Type)
}

func TestMergeConfigs_Services_ExistingServiceReplaced(t *testing.T) {
	base := &Config{
		Services: map[string]ServiceConfig{
			"ipfs": {Type: "ipfs", Port: 5001},
		},
	}
	override := &Config{
		Services: map[string]ServiceConfig{
			"ipfs": {Type: "ipfs", Port: 5002},
		},
	}

	result := MergeConfigs(base, override)
	assert.Equal(t, 5002, result.Services["ipfs"].Port)
}

func TestMergeConfigs_Services_NilOverride_KeepsBase(t *testing.T) {
	base := &Config{
		Services: map[string]ServiceConfig{
			"ipfs": {Type: "ipfs"},
		},
	}
	override := &Config{Services: nil}

	result := MergeConfigs(base, override)
	require.Len(t, result.Services, 1)
}

func TestMergeConfigs_Plugins_NewPluginAdded(t *testing.T) {
	base := &Config{
		Plugins: map[string]PluginConfig{
			"alpha": {Version: "^1.0.0"},
		},
	}
	override := &Config{
		Plugins: map[string]PluginConfig{
			"beta": {Version: "~2.0.0"},
		},
	}

	result := MergeConfigs(base, override)
	require.Len(t, result.Plugins, 2)
	assert.Equal(t, "^1.0.0", result.Plugins["alpha"].Version)
	assert.Equal(t, "~2.0.0", result.Plugins["beta"].Version)
}

func TestMergeConfigs_Plugins_NilOverride_KeepsBase(t *testing.T) {
	base := &Config{
		Plugins: map[string]PluginConfig{
			"alpha": {Version: "^1.0.0"},
		},
	}
	override := &Config{Plugins: nil}

	result := MergeConfigs(base, override)
	require.Len(t, result.Plugins, 1)
}

func TestMergeConfigs_Profiles_Merged(t *testing.T) {
	base := &Config{
		Profiles: map[string]Profile{
			"dev": {Settings: Settings{LogLevel: "debug"}},
		},
	}
	override := &Config{
		Profiles: map[string]Profile{
			"staging": {Settings: Settings{LogLevel: "warn"}},
		},
	}

	result := MergeConfigs(base, override)
	require.Len(t, result.Profiles, 2)
	assert.Equal(t, "debug", result.Profiles["dev"].Settings.LogLevel)
	assert.Equal(t, "warn", result.Profiles["staging"].Settings.LogLevel)
}

func TestMergeConfigs_Profiles_NilBaseInitialized(t *testing.T) {
	base := &Config{Profiles: nil}
	override := &Config{
		Profiles: map[string]Profile{
			"dev": {Settings: Settings{LogLevel: "debug"}},
		},
	}

	result := MergeConfigs(base, override)
	require.Len(t, result.Profiles, 1)
}

func TestMergeConfigs_NilOverrideAllMaps(t *testing.T) {
	base := &Config{
		Version: "1.0",
		Name:    "base",
		Settings: Settings{
			Runtime:  "docker",
			LogLevel: "info",
		},
		Chains: map[string]ChainConfig{
			"eth": {Engine: "anvil"},
		},
		Services: map[string]ServiceConfig{
			"ipfs": {Type: "ipfs"},
		},
		Plugins: map[string]PluginConfig{
			"p": {Version: "^1.0.0"},
		},
	}
	override := &Config{} // All maps nil.

	result := MergeConfigs(base, override)
	assert.Equal(t, "1.0", result.Version)
	assert.Equal(t, "base", result.Name)
	assert.Equal(t, "docker", result.Settings.Runtime)
	require.Len(t, result.Chains, 1)
	require.Len(t, result.Services, 1)
	require.Len(t, result.Plugins, 1)
}

func TestMergeConfigs_DoesNotMutateBaseScalars(t *testing.T) {
	base := &Config{
		Version: "1.0",
		Name:    "original",
	}
	override := &Config{
		Name: "overridden",
	}

	result := MergeConfigs(base, override)

	assert.Equal(t, "original", base.Name)
	assert.Equal(t, "overridden", result.Name)
}

func TestMergeConfigs_ResultIsNewStruct(t *testing.T) {
	base := &Config{
		Version: "1.0",
		Name:    "base",
	}
	override := &Config{
		Name: "result",
	}

	result := MergeConfigs(base, override)
	assert.NotSame(t, base, result)
	assert.Equal(t, "result", result.Name)
	assert.Equal(t, "base", base.Name)
}

func TestMergeConfigs_Chains_OverrideAccountBalance(t *testing.T) {
	base := &Config{
		Chains: map[string]ChainConfig{
			"eth": {Engine: "anvil", AccountBalance: "10000"},
		},
	}
	override := &Config{
		Chains: map[string]ChainConfig{
			"eth": {AccountBalance: "99999"},
		},
	}

	result := MergeConfigs(base, override)
	assert.Equal(t, "99999", result.Chains["eth"].AccountBalance)
	assert.Equal(t, "anvil", result.Chains["eth"].Engine) // kept from base
}

func TestMergeConfigs_Chains_EmptyAccountBalance_KeepsBase(t *testing.T) {
	base := &Config{
		Chains: map[string]ChainConfig{
			"eth": {AccountBalance: "10000"},
		},
	}
	override := &Config{
		Chains: map[string]ChainConfig{
			"eth": {},
		},
	}

	result := MergeConfigs(base, override)
	assert.Equal(t, "10000", result.Chains["eth"].AccountBalance)
}

func TestMergeConfigs_Chains_OverrideBalance(t *testing.T) {
	base := &Config{
		Chains: map[string]ChainConfig{
			"eth": {Engine: "anvil", Balance: "5000"},
		},
	}
	override := &Config{
		Chains: map[string]ChainConfig{
			"eth": {Balance: "7777"},
		},
	}

	result := MergeConfigs(base, override)
	assert.Equal(t, "7777", result.Chains["eth"].Balance)
}

func TestMergeConfigs_Chains_EmptyBalance_KeepsBase(t *testing.T) {
	base := &Config{
		Chains: map[string]ChainConfig{
			"eth": {Balance: "5000"},
		},
	}
	override := &Config{
		Chains: map[string]ChainConfig{
			"eth": {},
		},
	}

	result := MergeConfigs(base, override)
	assert.Equal(t, "5000", result.Chains["eth"].Balance)
}

func TestMergeConfigs_Chains_OverrideCodeSizeLimit(t *testing.T) {
	base := &Config{
		Chains: map[string]ChainConfig{
			"eth": {Engine: "anvil", CodeSizeLimit: 24576},
		},
	}
	override := &Config{
		Chains: map[string]ChainConfig{
			"eth": {CodeSizeLimit: 49152},
		},
	}

	result := MergeConfigs(base, override)
	assert.Equal(t, uint64(49152), result.Chains["eth"].CodeSizeLimit)
}

func TestMergeConfigs_Chains_ZeroCodeSizeLimit_KeepsBase(t *testing.T) {
	base := &Config{
		Chains: map[string]ChainConfig{
			"eth": {CodeSizeLimit: 24576},
		},
	}
	override := &Config{
		Chains: map[string]ChainConfig{
			"eth": {},
		},
	}

	result := MergeConfigs(base, override)
	assert.Equal(t, uint64(24576), result.Chains["eth"].CodeSizeLimit)
}

func TestMergeConfigs_Chains_OverrideAutoImpersonate_TrueOverFalse(t *testing.T) {
	base := &Config{
		Chains: map[string]ChainConfig{
			"eth": {Engine: "anvil", AutoImpersonate: false},
		},
	}
	override := &Config{
		Chains: map[string]ChainConfig{
			"eth": {AutoImpersonate: true},
		},
	}

	result := MergeConfigs(base, override)
	assert.True(t, result.Chains["eth"].AutoImpersonate)
}

func TestMergeConfigs_Chains_OverrideAutoImpersonate_FalseDoesNotFlip(t *testing.T) {
	base := &Config{
		Chains: map[string]ChainConfig{
			"eth": {Engine: "anvil", AutoImpersonate: true},
		},
	}
	override := &Config{
		Chains: map[string]ChainConfig{
			"eth": {AutoImpersonate: false},
		},
	}

	result := MergeConfigs(base, override)
	assert.True(t, result.Chains["eth"].AutoImpersonate, "false override should not flip true base")
}

func TestMergeConfigs_Chains_OverrideMining(t *testing.T) {
	base := &Config{
		Chains: map[string]ChainConfig{
			"eth": {Engine: "anvil", Mining: MiningConfig{Mode: "auto"}},
		},
	}
	override := &Config{
		Chains: map[string]ChainConfig{
			"eth": {Mining: MiningConfig{Mode: "interval", Interval: "5s"}},
		},
	}

	result := MergeConfigs(base, override)
	assert.Equal(t, "interval", result.Chains["eth"].Mining.Mode)
	assert.Equal(t, "5s", result.Chains["eth"].Mining.Interval)
}

func TestMergeConfigs_Chains_EmptyMiningMode_KeepsBase(t *testing.T) {
	base := &Config{
		Chains: map[string]ChainConfig{
			"eth": {Mining: MiningConfig{Mode: "auto", Interval: "2s"}},
		},
	}
	override := &Config{
		Chains: map[string]ChainConfig{
			"eth": {Mining: MiningConfig{Mode: ""}},
		},
	}

	result := MergeConfigs(base, override)
	assert.Equal(t, "auto", result.Chains["eth"].Mining.Mode)
	assert.Equal(t, "2s", result.Chains["eth"].Mining.Interval)
}

func TestMergeConfigs_Chains_OverrideGenesisAccounts(t *testing.T) {
	base := &Config{
		Chains: map[string]ChainConfig{
			"eth": {
				Engine: "anvil",
				GenesisAccounts: []GenesisAccount{
					{Address: "0xaaa", Balance: "1000"},
				},
			},
		},
	}
	override := &Config{
		Chains: map[string]ChainConfig{
			"eth": {
				GenesisAccounts: []GenesisAccount{
					{Address: "0xbbb", Balance: "2000"},
					{Address: "0xccc", Balance: "3000"},
				},
			},
		},
	}

	result := MergeConfigs(base, override)
	require.Len(t, result.Chains["eth"].GenesisAccounts, 2)
	assert.Equal(t, "0xbbb", result.Chains["eth"].GenesisAccounts[0].Address)
	assert.Equal(t, "0xccc", result.Chains["eth"].GenesisAccounts[1].Address)
}

func TestMergeConfigs_Chains_EmptyGenesisAccounts_KeepsBase(t *testing.T) {
	base := &Config{
		Chains: map[string]ChainConfig{
			"eth": {
				GenesisAccounts: []GenesisAccount{
					{Address: "0xaaa", Balance: "1000"},
				},
			},
		},
	}
	override := &Config{
		Chains: map[string]ChainConfig{
			"eth": {},
		},
	}

	result := MergeConfigs(base, override)
	require.Len(t, result.Chains["eth"].GenesisAccounts, 1)
	assert.Equal(t, "0xaaa", result.Chains["eth"].GenesisAccounts[0].Address)
}

func TestMergeConfigs_Chains_OverrideDeploy(t *testing.T) {
	base := &Config{
		Chains: map[string]ChainConfig{
			"eth": {
				Engine: "anvil",
				Deploy: []DeployConfig{
					{Artifact: "Token.sol", Label: "token"},
				},
			},
		},
	}
	override := &Config{
		Chains: map[string]ChainConfig{
			"eth": {
				Deploy: []DeployConfig{
					{Artifact: "NFT.sol", Label: "nft"},
				},
			},
		},
	}

	result := MergeConfigs(base, override)
	require.Len(t, result.Chains["eth"].Deploy, 1)
	assert.Equal(t, "NFT.sol", result.Chains["eth"].Deploy[0].Artifact)
	assert.Equal(t, "nft", result.Chains["eth"].Deploy[0].Label)
}

func TestMergeConfigs_Chains_EmptyDeploy_KeepsBase(t *testing.T) {
	base := &Config{
		Chains: map[string]ChainConfig{
			"eth": {
				Deploy: []DeployConfig{
					{Artifact: "Token.sol", Label: "token"},
				},
			},
		},
	}
	override := &Config{
		Chains: map[string]ChainConfig{
			"eth": {},
		},
	}

	result := MergeConfigs(base, override)
	require.Len(t, result.Chains["eth"].Deploy, 1)
	assert.Equal(t, "Token.sol", result.Chains["eth"].Deploy[0].Artifact)
}

func TestMergeConfigs_Chains_OverrideBaseFee(t *testing.T) {
	base := &Config{
		Chains: map[string]ChainConfig{
			"eth": {Engine: "anvil", BaseFee: 1},
		},
	}
	override := &Config{
		Chains: map[string]ChainConfig{
			"eth": {BaseFee: 100},
		},
	}

	result := MergeConfigs(base, override)
	assert.Equal(t, uint64(100), result.Chains["eth"].BaseFee)
}

func TestMergeConfigs_Chains_ZeroBaseFee_KeepsBase(t *testing.T) {
	base := &Config{
		Chains: map[string]ChainConfig{
			"eth": {BaseFee: 1},
		},
	}
	override := &Config{
		Chains: map[string]ChainConfig{
			"eth": {},
		},
	}

	result := MergeConfigs(base, override)
	assert.Equal(t, uint64(1), result.Chains["eth"].BaseFee)
}

func TestMergeConfigs_Chains_OverrideAccounts(t *testing.T) {
	base := &Config{
		Chains: map[string]ChainConfig{
			"eth": {Accounts: 10},
		},
	}
	override := &Config{
		Chains: map[string]ChainConfig{
			"eth": {Accounts: 50},
		},
	}

	result := MergeConfigs(base, override)
	assert.Equal(t, 50, result.Chains["eth"].Accounts)
}

func TestMergeSettings_TelemetryFalseDoesNotFlipTrue(t *testing.T) {
	base := &Config{Settings: Settings{Telemetry: true}}
	override := &Config{Settings: Settings{Telemetry: false}}

	result := MergeConfigs(base, override)
	assert.True(t, result.Settings.Telemetry, "false override should not flip true base")
}

func TestMergeConfigs_Services_NilBaseInitialized(t *testing.T) {
	base := &Config{Services: nil}
	override := &Config{
		Services: map[string]ServiceConfig{
			"ipfs": {Type: "ipfs", Port: 5001},
		},
	}

	result := MergeConfigs(base, override)
	require.Len(t, result.Services, 1)
	assert.Equal(t, 5001, result.Services["ipfs"].Port)
}

func TestMergeConfigs_Plugins_NilBaseInitialized(t *testing.T) {
	base := &Config{Plugins: nil}
	override := &Config{
		Plugins: map[string]PluginConfig{
			"myplugin": {Version: "^2.0.0"},
		},
	}

	result := MergeConfigs(base, override)
	require.Len(t, result.Plugins, 1)
	assert.Equal(t, "^2.0.0", result.Plugins["myplugin"].Version)
}

func TestMergeConfigs_Profiles_ExistingProfileReplaced(t *testing.T) {
	base := &Config{
		Profiles: map[string]Profile{
			"dev": {Settings: Settings{LogLevel: "debug", Runtime: "docker"}},
		},
	}
	override := &Config{
		Profiles: map[string]Profile{
			"dev": {Settings: Settings{LogLevel: "error"}},
		},
	}

	result := MergeConfigs(base, override)
	assert.Equal(t, "error", result.Profiles["dev"].Settings.LogLevel)
}

func TestMergeConfigs_Chains_FullMerge_PreservesUnsetFields(t *testing.T) {
	base := &Config{
		Chains: map[string]ChainConfig{
			"eth": {
				Engine:          "anvil",
				ChainID:         31337,
				GasLimit:        30000000,
				BaseFee:         1,
				Hardfork:        "cancun",
				BlockTime:       "2s",
				Accounts:        10,
				AccountBalance:  "10000",
				CodeSizeLimit:   24576,
				AutoImpersonate: true,
				Mining:          MiningConfig{Mode: "auto"},
				GenesisAccounts: []GenesisAccount{{Address: "0xaaa", Balance: "100"}},
				Deploy:          []DeployConfig{{Artifact: "Token.sol", Label: "token"}},
			},
		},
	}
	override := &Config{
		Chains: map[string]ChainConfig{
			"eth": {ChainID: 1},
		},
	}

	result := MergeConfigs(base, override)
	chain := result.Chains["eth"]
	assert.Equal(t, uint64(1), chain.ChainID)        // overridden
	assert.Equal(t, "anvil", chain.Engine)             // base
	assert.Equal(t, uint64(30000000), chain.GasLimit)  // base
	assert.Equal(t, uint64(1), chain.BaseFee)          // base
	assert.Equal(t, "cancun", chain.Hardfork)          // base
	assert.Equal(t, "2s", chain.BlockTime)             // base
	assert.Equal(t, 10, chain.Accounts)                // base
	assert.Equal(t, "10000", chain.AccountBalance)     // base
	assert.Equal(t, uint64(24576), chain.CodeSizeLimit) // base
	assert.True(t, chain.AutoImpersonate)               // base
	assert.Equal(t, "auto", chain.Mining.Mode)          // base
	require.Len(t, chain.GenesisAccounts, 1)            // base
	require.Len(t, chain.Deploy, 1)                     // base
}
