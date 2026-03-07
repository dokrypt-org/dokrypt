package config

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func emptyConfig() *Config {
	return &Config{}
}

func configWithChain(name string, chain ChainConfig) *Config {
	cfg := emptyConfig()
	cfg.Chains = map[string]ChainConfig{name: chain}
	return cfg
}

func TestApplyDefaults_SetsVersion(t *testing.T) {
	cfg := emptyConfig()
	ApplyDefaults(cfg)
	assert.Equal(t, "1.0", cfg.Version)
}

func TestApplyDefaults_DoesNotOverwriteVersion(t *testing.T) {
	cfg := emptyConfig()
	cfg.Version = "2.0"
	ApplyDefaults(cfg)
	assert.Equal(t, "2.0", cfg.Version)
}

func TestApplyDefaults_Settings_Runtime(t *testing.T) {
	cfg := emptyConfig()
	ApplyDefaults(cfg)
	assert.Equal(t, "docker", cfg.Settings.Runtime)
}

func TestApplyDefaults_Settings_DoesNotOverwriteRuntime(t *testing.T) {
	cfg := emptyConfig()
	cfg.Settings.Runtime = "podman"
	ApplyDefaults(cfg)
	assert.Equal(t, "podman", cfg.Settings.Runtime)
}

func TestApplyDefaults_Settings_LogLevel(t *testing.T) {
	cfg := emptyConfig()
	ApplyDefaults(cfg)
	assert.Equal(t, "info", cfg.Settings.LogLevel)
}

func TestApplyDefaults_Settings_DoesNotOverwriteLogLevel(t *testing.T) {
	cfg := emptyConfig()
	cfg.Settings.LogLevel = "debug"
	ApplyDefaults(cfg)
	assert.Equal(t, "debug", cfg.Settings.LogLevel)
}

func TestApplyDefaults_Settings_BlockTime(t *testing.T) {
	cfg := emptyConfig()
	ApplyDefaults(cfg)
	assert.Equal(t, "2s", cfg.Settings.BlockTime)
}

func TestApplyDefaults_Settings_DoesNotOverwriteBlockTime(t *testing.T) {
	cfg := emptyConfig()
	cfg.Settings.BlockTime = "5s"
	ApplyDefaults(cfg)
	assert.Equal(t, "5s", cfg.Settings.BlockTime)
}

func TestApplyDefaults_Settings_Accounts(t *testing.T) {
	cfg := emptyConfig()
	ApplyDefaults(cfg)
	assert.Equal(t, 10, cfg.Settings.Accounts)
}

func TestApplyDefaults_Settings_DoesNotOverwriteAccounts(t *testing.T) {
	cfg := emptyConfig()
	cfg.Settings.Accounts = 5
	ApplyDefaults(cfg)
	assert.Equal(t, 5, cfg.Settings.Accounts)
}

func TestApplyDefaults_Settings_AccountBalance(t *testing.T) {
	cfg := emptyConfig()
	ApplyDefaults(cfg)
	assert.Equal(t, "10000", cfg.Settings.AccountBalance)
}

func TestApplyDefaults_Settings_DoesNotOverwriteAccountBalance(t *testing.T) {
	cfg := emptyConfig()
	cfg.Settings.AccountBalance = "9999"
	ApplyDefaults(cfg)
	assert.Equal(t, "9999", cfg.Settings.AccountBalance)
}

func TestApplyDefaults_Chain_Engine(t *testing.T) {
	cfg := configWithChain("mychain", ChainConfig{})
	ApplyDefaults(cfg)
	assert.Equal(t, "anvil", cfg.Chains["mychain"].Engine)
}

func TestApplyDefaults_Chain_DoesNotOverwriteEngine(t *testing.T) {
	cfg := configWithChain("mychain", ChainConfig{Engine: "geth"})
	ApplyDefaults(cfg)
	assert.Equal(t, "geth", cfg.Chains["mychain"].Engine)
}

func TestApplyDefaults_Chain_ChainID(t *testing.T) {
	cfg := configWithChain("mychain", ChainConfig{})
	ApplyDefaults(cfg)
	assert.Equal(t, uint64(31337), cfg.Chains["mychain"].ChainID)
}

func TestApplyDefaults_Chain_DoesNotOverwriteChainID(t *testing.T) {
	cfg := configWithChain("mychain", ChainConfig{ChainID: 1337})
	ApplyDefaults(cfg)
	assert.Equal(t, uint64(1337), cfg.Chains["mychain"].ChainID)
}

func TestApplyDefaults_Chain_BlockTimeInheritedFromSettings(t *testing.T) {
	cfg := configWithChain("mychain", ChainConfig{})
	cfg.Settings.BlockTime = "3s"
	ApplyDefaults(cfg)
	assert.Equal(t, "3s", cfg.Chains["mychain"].BlockTime)
}

func TestApplyDefaults_Chain_BlockTimeDefault(t *testing.T) {
	cfg := configWithChain("mychain", ChainConfig{})
	ApplyDefaults(cfg)
	assert.Equal(t, "2s", cfg.Chains["mychain"].BlockTime)
}

func TestApplyDefaults_Chain_DoesNotOverwriteBlockTime(t *testing.T) {
	cfg := configWithChain("mychain", ChainConfig{BlockTime: "10s"})
	ApplyDefaults(cfg)
	assert.Equal(t, "10s", cfg.Chains["mychain"].BlockTime)
}

func TestApplyDefaults_Chain_AccountsInheritedFromSettings(t *testing.T) {
	cfg := configWithChain("mychain", ChainConfig{})
	cfg.Settings.Accounts = 20
	ApplyDefaults(cfg)
	assert.Equal(t, 20, cfg.Chains["mychain"].Accounts)
}

func TestApplyDefaults_Chain_AccountsDefault(t *testing.T) {
	cfg := configWithChain("mychain", ChainConfig{})
	ApplyDefaults(cfg)
	assert.Equal(t, 10, cfg.Chains["mychain"].Accounts)
}

func TestApplyDefaults_Chain_AccountBalanceInheritedFromSettings(t *testing.T) {
	cfg := configWithChain("mychain", ChainConfig{})
	cfg.Settings.AccountBalance = "5000"
	ApplyDefaults(cfg)
	assert.Equal(t, "5000", cfg.Chains["mychain"].AccountBalance)
}

func TestApplyDefaults_Chain_GasLimit(t *testing.T) {
	cfg := configWithChain("mychain", ChainConfig{})
	ApplyDefaults(cfg)
	assert.Equal(t, uint64(30000000), cfg.Chains["mychain"].GasLimit)
}

func TestApplyDefaults_Chain_DoesNotOverwriteGasLimit(t *testing.T) {
	cfg := configWithChain("mychain", ChainConfig{GasLimit: 12345678})
	ApplyDefaults(cfg)
	assert.Equal(t, uint64(12345678), cfg.Chains["mychain"].GasLimit)
}

func TestApplyDefaults_Chain_BaseFee(t *testing.T) {
	cfg := configWithChain("mychain", ChainConfig{})
	ApplyDefaults(cfg)
	assert.Equal(t, uint64(1), cfg.Chains["mychain"].BaseFee)
}

func TestApplyDefaults_Chain_DoesNotOverwriteBaseFee(t *testing.T) {
	cfg := configWithChain("mychain", ChainConfig{BaseFee: 7})
	ApplyDefaults(cfg)
	assert.Equal(t, uint64(7), cfg.Chains["mychain"].BaseFee)
}

func TestApplyDefaults_Chain_Hardfork(t *testing.T) {
	cfg := configWithChain("mychain", ChainConfig{})
	ApplyDefaults(cfg)
	assert.Equal(t, "cancun", cfg.Chains["mychain"].Hardfork)
}

func TestApplyDefaults_Chain_DoesNotOverwriteHardfork(t *testing.T) {
	cfg := configWithChain("mychain", ChainConfig{Hardfork: "london"})
	ApplyDefaults(cfg)
	assert.Equal(t, "london", cfg.Chains["mychain"].Hardfork)
}

func TestApplyDefaults_Chain_MiningMode(t *testing.T) {
	cfg := configWithChain("mychain", ChainConfig{})
	ApplyDefaults(cfg)
	assert.Equal(t, "auto", cfg.Chains["mychain"].Mining.Mode)
}

func TestApplyDefaults_Chain_DoesNotOverwriteMiningMode(t *testing.T) {
	cfg := configWithChain("mychain", ChainConfig{Mining: MiningConfig{Mode: "interval"}})
	ApplyDefaults(cfg)
	assert.Equal(t, "interval", cfg.Chains["mychain"].Mining.Mode)
}

func TestApplyDefaults_MultipleChains(t *testing.T) {
	cfg := &Config{
		Chains: map[string]ChainConfig{
			"alpha": {},
			"beta":  {Engine: "hardhat"},
		},
	}
	ApplyDefaults(cfg)

	alpha := cfg.Chains["alpha"]
	beta := cfg.Chains["beta"]

	assert.Equal(t, "anvil", alpha.Engine)
	assert.Equal(t, "hardhat", beta.Engine)

	assert.Equal(t, uint64(30000000), alpha.GasLimit)
	assert.Equal(t, uint64(30000000), beta.GasLimit)
	assert.Equal(t, "cancun", alpha.Hardfork)
	assert.Equal(t, "cancun", beta.Hardfork)
}

func TestApplyDefaults_Tests_Dir(t *testing.T) {
	cfg := emptyConfig()
	ApplyDefaults(cfg)
	assert.Equal(t, "./test", cfg.Tests.Dir)
}

func TestApplyDefaults_Tests_DoesNotOverwriteDir(t *testing.T) {
	cfg := emptyConfig()
	cfg.Tests.Dir = "./custom-tests"
	ApplyDefaults(cfg)
	assert.Equal(t, "./custom-tests", cfg.Tests.Dir)
}

func TestApplyDefaults_Tests_Timeout(t *testing.T) {
	cfg := emptyConfig()
	ApplyDefaults(cfg)
	assert.Equal(t, "60s", cfg.Tests.Timeout)
}

func TestApplyDefaults_Tests_DoesNotOverwriteTimeout(t *testing.T) {
	cfg := emptyConfig()
	cfg.Tests.Timeout = "120s"
	ApplyDefaults(cfg)
	assert.Equal(t, "120s", cfg.Tests.Timeout)
}

func TestApplyDefaults_Tests_Parallel(t *testing.T) {
	cfg := emptyConfig()
	ApplyDefaults(cfg)
	assert.Equal(t, 4, cfg.Tests.Parallel)
}

func TestApplyDefaults_Tests_DoesNotOverwriteParallel(t *testing.T) {
	cfg := emptyConfig()
	cfg.Tests.Parallel = 8
	ApplyDefaults(cfg)
	assert.Equal(t, 8, cfg.Tests.Parallel)
}

func TestApplyDefaults_Idempotent(t *testing.T) {
	cfg := emptyConfig()
	ApplyDefaults(cfg)

	runtime := cfg.Settings.Runtime
	logLevel := cfg.Settings.LogLevel
	version := cfg.Version

	ApplyDefaults(cfg)

	assert.Equal(t, runtime, cfg.Settings.Runtime)
	assert.Equal(t, logLevel, cfg.Settings.LogLevel)
	assert.Equal(t, version, cfg.Version)
}

func TestApplyDefaults_NoChainsDoesNotPanic(t *testing.T) {
	cfg := emptyConfig()
	require.NotPanics(t, func() {
		ApplyDefaults(cfg)
	})
}

func TestApplyDefaults_NilChainMapDoesNotPanic(t *testing.T) {
	cfg := emptyConfig()
	cfg.Chains = nil
	require.NotPanics(t, func() {
		ApplyDefaults(cfg)
	})
}

func TestApplyDefaults_Chain_AccountBalanceSkippedWhenBalanceSet(t *testing.T) {
	cfg := configWithChain("mychain", ChainConfig{Balance: "9999"})
	ApplyDefaults(cfg)
	chain := cfg.Chains["mychain"]
	assert.Equal(t, "9999", chain.Balance)
	assert.Equal(t, "", chain.AccountBalance, "AccountBalance should remain empty when Balance is already set")
}

func TestApplyDefaults_Chain_AccountBalanceSetWhenBothEmpty(t *testing.T) {
	cfg := configWithChain("mychain", ChainConfig{})
	ApplyDefaults(cfg)
	chain := cfg.Chains["mychain"]
	assert.Equal(t, "10000", chain.AccountBalance)
	assert.Equal(t, "", chain.Balance)
}

func TestApplyDefaults_Chain_DoesNotOverwriteAccountBalance(t *testing.T) {
	cfg := configWithChain("mychain", ChainConfig{AccountBalance: "5000"})
	ApplyDefaults(cfg)
	assert.Equal(t, "5000", cfg.Chains["mychain"].AccountBalance)
}

func TestApplyDefaults_EmptyChainMap_DoesNotPanic(t *testing.T) {
	cfg := emptyConfig()
	cfg.Chains = make(map[string]ChainConfig)
	require.NotPanics(t, func() {
		ApplyDefaults(cfg)
	})
	assert.Empty(t, cfg.Chains)
}

func TestApplyDefaults_Chain_AccountsInheritsSettingsCustomValue(t *testing.T) {
	cfg := configWithChain("mychain", ChainConfig{})
	cfg.Settings.Accounts = 42
	ApplyDefaults(cfg)
	assert.Equal(t, 42, cfg.Chains["mychain"].Accounts)
}

func TestApplyDefaults_Tests_BooleanFieldsDefaultToFalse(t *testing.T) {
	cfg := emptyConfig()
	ApplyDefaults(cfg)
	assert.False(t, cfg.Tests.SnapshotIsolation)
	assert.False(t, cfg.Tests.GasReport)
	assert.False(t, cfg.Tests.Coverage)
}

func TestApplyDefaults_Settings_TelemetryDefaultsFalse(t *testing.T) {
	cfg := emptyConfig()
	ApplyDefaults(cfg)
	assert.False(t, cfg.Settings.Telemetry)
}

func TestApplyDefaults_MultipleChains_IndependentDefaults(t *testing.T) {
	cfg := &Config{
		Chains: map[string]ChainConfig{
			"a": {ChainID: 100},
			"b": {ChainID: 200, Engine: "hardhat", Hardfork: "london"},
			"c": {},
		},
	}
	ApplyDefaults(cfg)

	a := cfg.Chains["a"]
	assert.Equal(t, uint64(100), a.ChainID) // preserved
	assert.Equal(t, "anvil", a.Engine)       // defaulted
	assert.Equal(t, "cancun", a.Hardfork)    // defaulted

	b := cfg.Chains["b"]
	assert.Equal(t, uint64(200), b.ChainID) // preserved
	assert.Equal(t, "hardhat", b.Engine)     // preserved
	assert.Equal(t, "london", b.Hardfork)    // preserved

	c := cfg.Chains["c"]
	assert.Equal(t, uint64(31337), c.ChainID) // defaulted
	assert.Equal(t, "anvil", c.Engine)         // defaulted
}
