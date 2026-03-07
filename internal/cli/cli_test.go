package cli

import (
	"testing"
	"time"

	"github.com/spf13/cobra"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewRootCmd(t *testing.T) {
	cmd := NewRootCmd()

	assert.Equal(t, "dokrypt", cmd.Use)
	assert.Equal(t, "Dokrypt — Docker for Web3", cmd.Short)
	assert.Contains(t, cmd.Long, "Web3-native containerization")
	assert.True(t, cmd.SilenceUsage)
	assert.True(t, cmd.SilenceErrors)
}

func TestRootCmd_PersistentFlags(t *testing.T) {
	cmd := NewRootCmd()

	tests := []struct {
		name         string
		shorthand    string
		defaultValue string
		usage        string
	}{
		{"config", "c", "", "path to dokrypt.yaml"},
		{"verbose", "v", "false", "enable verbose output"},
		{"quiet", "q", "false", "suppress non-error output"},
		{"json", "", "false", "output in JSON format"},
		{"runtime", "", "", "container runtime (docker or podman)"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			f := cmd.PersistentFlags().Lookup(tt.name)
			require.NotNil(t, f, "persistent flag %q should exist", tt.name)
			assert.Equal(t, tt.defaultValue, f.DefValue, "default for %q", tt.name)
			if tt.shorthand != "" {
				assert.Equal(t, tt.shorthand, f.Shorthand, "shorthand for %q", tt.name)
			}
		})
	}
}

func TestRootCmd_HasAllSubcommands(t *testing.T) {
	cmd := NewRootCmd()

	expected := []string{
		"init", "up", "down", "restart", "status", "logs", "exec",
		"snapshot", "fork", "accounts", "chain", "bridge",
		"test", "plugin", "template", "config", "doctor",
		"version", "marketplace",
	}

	subNames := make(map[string]bool)
	for _, sub := range cmd.Commands() {
		subNames[sub.Name()] = true
	}

	for _, name := range expected {
		assert.True(t, subNames[name], "root should have subcommand %q", name)
	}
}

func TestNewVersionCmd(t *testing.T) {
	cmd := newVersionCmd()

	assert.Equal(t, "version", cmd.Use)
	assert.Equal(t, "Show version information", cmd.Short)
	assert.NotNil(t, cmd.RunE)
}

func TestNewInitCmd(t *testing.T) {
	cmd := newInitCmd()

	assert.Equal(t, "init <project-name>", cmd.Use)
	assert.Equal(t, "Scaffold a new Dokrypt project", cmd.Short)
	assert.Contains(t, cmd.Long, "Creates a new directory")
	assert.NotNil(t, cmd.Args)

	err := cmd.Args(cmd, []string{})
	assert.Error(t, err, "should reject zero args")
	err = cmd.Args(cmd, []string{"one", "two"})
	assert.Error(t, err, "should reject two args")
	err = cmd.Args(cmd, []string{"my-project"})
	assert.NoError(t, err, "should accept exactly one arg")
}

func TestInitCmd_Flags(t *testing.T) {
	cmd := newInitCmd()

	tests := []struct {
		name         string
		shorthand    string
		defaultValue string
	}{
		{"template", "t", "evm-basic"},
		{"chain", "", "ethereum"},
		{"engine", "", "anvil"},
		{"no-git", "", "false"},
		{"dir", "", ""},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			f := cmd.Flags().Lookup(tt.name)
			require.NotNil(t, f, "flag %q should exist", tt.name)
			assert.Equal(t, tt.defaultValue, f.DefValue, "default for %q", tt.name)
			if tt.shorthand != "" {
				assert.Equal(t, tt.shorthand, f.Shorthand)
			}
		})
	}
}

func TestInitCmd_FlagParsing(t *testing.T) {
	cmd := newInitCmd()
	cmd.SetArgs([]string{"myproj", "--template", "evm-defi", "--chain", "polygon", "--engine", "hardhat", "--no-git", "--dir", "/tmp/mydir"})
	cmd.RunE = func(c *cobra.Command, args []string) error { return nil }
	err := cmd.Execute()
	require.NoError(t, err)

	assert.Equal(t, "evm-defi", cmd.Flags().Lookup("template").Value.String())
	assert.Equal(t, "polygon", cmd.Flags().Lookup("chain").Value.String())
	assert.Equal(t, "hardhat", cmd.Flags().Lookup("engine").Value.String())
	assert.Equal(t, "true", cmd.Flags().Lookup("no-git").Value.String())
	assert.Equal(t, "/tmp/mydir", cmd.Flags().Lookup("dir").Value.String())
}

func TestNewUpCmd(t *testing.T) {
	cmd := newUpCmd()

	assert.Equal(t, "up", cmd.Use)
	assert.Equal(t, "Start all services", cmd.Short)
	assert.Contains(t, cmd.Long, "dokrypt.yaml")
	assert.NotNil(t, cmd.RunE)
}

func TestUpCmd_Flags(t *testing.T) {
	cmd := newUpCmd()

	tests := []struct {
		name         string
		shorthand    string
		defaultValue string
	}{
		{"detach", "d", "false"},
		{"build", "", "false"},
		{"service", "", "[]"},
		{"fresh", "", "false"},
		{"fork", "", ""},
		{"fork-block", "", "0"},
		{"snapshot", "", ""},
		{"profile", "", ""},
		{"timeout", "", "5m0s"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			f := cmd.Flags().Lookup(tt.name)
			require.NotNil(t, f, "flag %q should exist", tt.name)
			assert.Equal(t, tt.defaultValue, f.DefValue, "default for %q", tt.name)
			if tt.shorthand != "" {
				assert.Equal(t, tt.shorthand, f.Shorthand)
			}
		})
	}
}

func TestNewDownCmd(t *testing.T) {
	cmd := newDownCmd()

	assert.Equal(t, "down", cmd.Use)
	assert.Equal(t, "Stop all services", cmd.Short)
	assert.Contains(t, cmd.Long, "Stops all running")
	assert.NotNil(t, cmd.RunE)
}

func TestDownCmd_Flags(t *testing.T) {
	cmd := newDownCmd()

	tests := []struct {
		name         string
		defaultValue string
	}{
		{"volumes", "false"},
		{"service", "[]"},
		{"timeout", "30s"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			f := cmd.Flags().Lookup(tt.name)
			require.NotNil(t, f, "flag %q should exist", tt.name)
			assert.Equal(t, tt.defaultValue, f.DefValue)
		})
	}
}

func TestNewRestartCmd(t *testing.T) {
	cmd := newRestartCmd()

	assert.Equal(t, "restart", cmd.Use)
	assert.Equal(t, "Restart services", cmd.Short)
	assert.Contains(t, cmd.Long, "Restarts individual service")
	assert.NotNil(t, cmd.RunE)

	f := cmd.Flags().Lookup("service")
	require.NotNil(t, f)
	assert.Equal(t, "", f.DefValue)
}

func TestNewStatusCmd(t *testing.T) {
	cmd := newStatusCmd()

	assert.Equal(t, "status", cmd.Use)
	assert.Equal(t, "Show environment status", cmd.Short)
	assert.NotNil(t, cmd.RunE)
}

func TestStatusCmd_Flags(t *testing.T) {
	cmd := newStatusCmd()

	tests := []struct {
		name         string
		shorthand    string
		defaultValue string
	}{
		{"watch", "w", "false"},
		{"service", "", ""},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			f := cmd.Flags().Lookup(tt.name)
			require.NotNil(t, f, "flag %q should exist", tt.name)
			assert.Equal(t, tt.defaultValue, f.DefValue)
			if tt.shorthand != "" {
				assert.Equal(t, tt.shorthand, f.Shorthand)
			}
		})
	}
}

func TestNewLogsCmd(t *testing.T) {
	cmd := newLogsCmd()

	assert.Equal(t, "logs", cmd.Use)
	assert.Equal(t, "Stream service logs", cmd.Short)
	assert.NotNil(t, cmd.RunE)
}

func TestLogsCmd_Flags(t *testing.T) {
	cmd := newLogsCmd()

	tests := []struct {
		name         string
		shorthand    string
		defaultValue string
	}{
		{"service", "s", ""},
		{"follow", "f", "false"},
		{"tail", "", "50"},
		{"since", "", ""},
		{"timestamps", "", "false"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			f := cmd.Flags().Lookup(tt.name)
			require.NotNil(t, f, "flag %q should exist", tt.name)
			assert.Equal(t, tt.defaultValue, f.DefValue, "default for %q", tt.name)
			if tt.shorthand != "" {
				assert.Equal(t, tt.shorthand, f.Shorthand)
			}
		})
	}
}

func TestNewExecCmd(t *testing.T) {
	cmd := newExecCmd()

	assert.Equal(t, "exec <service> <command> [args...]", cmd.Use)
	assert.Equal(t, "Execute command in service container", cmd.Short)
	assert.Contains(t, cmd.Long, "Execute a command inside")
	assert.NotNil(t, cmd.RunE)

	err := cmd.Args(cmd, []string{})
	assert.Error(t, err, "should reject zero args")
	err = cmd.Args(cmd, []string{"svc"})
	assert.Error(t, err, "should reject one arg")
	err = cmd.Args(cmd, []string{"svc", "cmd"})
	assert.NoError(t, err, "should accept two args")
	err = cmd.Args(cmd, []string{"svc", "cmd", "arg1", "arg2"})
	assert.NoError(t, err, "should accept more than two args")
}

func TestExecCmd_Flags(t *testing.T) {
	cmd := newExecCmd()

	tests := []struct {
		name         string
		shorthand    string
		defaultValue string
	}{
		{"interactive", "i", "false"},
		{"tty", "t", "false"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			f := cmd.Flags().Lookup(tt.name)
			require.NotNil(t, f, "flag %q should exist", tt.name)
			assert.Equal(t, tt.defaultValue, f.DefValue)
			assert.Equal(t, tt.shorthand, f.Shorthand)
		})
	}
}

func TestNewBridgeCmd(t *testing.T) {
	cmd := newBridgeCmd()

	assert.Equal(t, "bridge", cmd.Use)
	assert.Equal(t, "Bridge simulation (multi-chain)", cmd.Short)

	subNames := make(map[string]bool)
	for _, sub := range cmd.Commands() {
		subNames[sub.Name()] = true
	}

	assert.True(t, subNames["send"], "should have send subcommand")
	assert.True(t, subNames["status"], "should have status subcommand")
	assert.True(t, subNames["relay"], "should have relay subcommand")
	assert.True(t, subNames["config"], "should have config subcommand")
}

func TestBridgeSendCmd(t *testing.T) {
	cmd := newBridgeSendCmd()

	assert.Equal(t, "send <from-chain> <to-chain> <amount>", cmd.Use)
	assert.Equal(t, "Simulate bridge transfer", cmd.Short)

	err := cmd.Args(cmd, []string{"a", "b"})
	assert.Error(t, err)
	err = cmd.Args(cmd, []string{"a", "b", "c"})
	assert.NoError(t, err)
	err = cmd.Args(cmd, []string{"a", "b", "c", "d"})
	assert.Error(t, err)

	f := cmd.Flags().Lookup("token")
	require.NotNil(t, f)
	assert.Equal(t, "", f.DefValue)

	f = cmd.Flags().Lookup("from")
	require.NotNil(t, f)
	assert.Equal(t, "", f.DefValue)
}

func TestBridgeRelayCmd(t *testing.T) {
	cmd := newBridgeRelayCmd()

	assert.Equal(t, "relay", cmd.Use)
	assert.Equal(t, "Force relay pending messages", cmd.Short)

	f := cmd.Flags().Lookup("blocks")
	require.NotNil(t, f)
	assert.Equal(t, "0", f.DefValue)
}

func TestBridgeStatusCmd(t *testing.T) {
	cmd := newBridgeStatusCmd()
	assert.Equal(t, "status", cmd.Use)
	assert.Equal(t, "Show bridge queue", cmd.Short)
}

func TestBridgeConfigCmd(t *testing.T) {
	cmd := newBridgeConfigCmd()
	assert.Equal(t, "config", cmd.Use)
	assert.Equal(t, "Show bridge configuration", cmd.Short)
}

func TestBridgeAddress(t *testing.T) {
	assert.Equal(t, "0x000000000000000000000000000000000000B12D", bridgeAddress)
}

func TestNewChainCmd(t *testing.T) {
	cmd := newChainCmd()

	assert.Equal(t, "chain", cmd.Use)
	assert.Equal(t, "Chain management", cmd.Short)

	f := cmd.PersistentFlags().Lookup("chain")
	require.NotNil(t, f)
	assert.Equal(t, "", f.DefValue)

	subNames := make(map[string]bool)
	for _, sub := range cmd.Commands() {
		subNames[sub.Name()] = true
	}

	expected := []string{"mine", "set-balance", "time-travel", "set-gas-price", "impersonate", "stop-impersonating", "reset", "info"}
	for _, name := range expected {
		assert.True(t, subNames[name], "chain should have subcommand %q", name)
	}
}

func TestChainMineCmd(t *testing.T) {
	chainName := ""
	cmd := newChainMineCmd(&chainName)

	assert.Equal(t, "mine [n]", cmd.Use)
	assert.Equal(t, "Mine N blocks (default 1)", cmd.Short)

	err := cmd.Args(cmd, []string{})
	assert.NoError(t, err)
	err = cmd.Args(cmd, []string{"5"})
	assert.NoError(t, err)
	err = cmd.Args(cmd, []string{"5", "10"})
	assert.Error(t, err)
}

func TestChainSetBalanceCmd(t *testing.T) {
	chainName := ""
	cmd := newChainSetBalanceCmd(&chainName)

	assert.Equal(t, "set-balance <address> <amount-eth>", cmd.Use)
	assert.Equal(t, "Set account balance (in ETH)", cmd.Short)

	err := cmd.Args(cmd, []string{"addr"})
	assert.Error(t, err)
	err = cmd.Args(cmd, []string{"addr", "100"})
	assert.NoError(t, err)
}

func TestChainTimeTravelCmd(t *testing.T) {
	chainName := ""
	cmd := newChainTimeTravelCmd(&chainName)

	assert.Equal(t, "time-travel <duration>", cmd.Use)
	assert.Equal(t, "Advance chain time (e.g. 1h, 7d, 3600)", cmd.Short)

	err := cmd.Args(cmd, []string{})
	assert.Error(t, err)
	err = cmd.Args(cmd, []string{"1h"})
	assert.NoError(t, err)
}

func TestChainSetGasPriceCmd(t *testing.T) {
	chainName := ""
	cmd := newChainSetGasPriceCmd(&chainName)

	assert.Equal(t, "set-gas-price <gwei>", cmd.Use)
	assert.Equal(t, "Set base gas price", cmd.Short)

	err := cmd.Args(cmd, []string{})
	assert.Error(t, err)
	err = cmd.Args(cmd, []string{"20"})
	assert.NoError(t, err)
}

func TestChainImpersonateCmd(t *testing.T) {
	chainName := ""
	cmd := newChainImpersonateCmd(&chainName)

	assert.Equal(t, "impersonate <address>", cmd.Use)
	assert.Equal(t, "Impersonate an account", cmd.Short)

	err := cmd.Args(cmd, []string{"0xabc"})
	assert.NoError(t, err)
	err = cmd.Args(cmd, []string{})
	assert.Error(t, err)
}

func TestChainStopImpersonatingCmd(t *testing.T) {
	chainName := ""
	cmd := newChainStopImpersonatingCmd(&chainName)

	assert.Equal(t, "stop-impersonating <address>", cmd.Use)
	assert.Equal(t, "Stop impersonating an account", cmd.Short)

	err := cmd.Args(cmd, []string{"0xabc"})
	assert.NoError(t, err)
}

func TestChainResetCmd(t *testing.T) {
	chainName := ""
	cmd := newChainResetCmd(&chainName)

	assert.Equal(t, "reset", cmd.Use)
	assert.Equal(t, "Reset chain to genesis or fork", cmd.Short)

	f := cmd.Flags().Lookup("fork")
	require.NotNil(t, f)
	assert.Equal(t, "", f.DefValue)

	f = cmd.Flags().Lookup("block")
	require.NotNil(t, f)
	assert.Equal(t, "0", f.DefValue)
}

func TestChainInfoCmd(t *testing.T) {
	chainName := ""
	cmd := newChainInfoCmd(&chainName)

	assert.Equal(t, "info", cmd.Use)
	assert.Equal(t, "Show chain info", cmd.Short)
}

func TestNewConfigCmd(t *testing.T) {
	cmd := newConfigCmd()

	assert.Equal(t, "config", cmd.Use)
	assert.Equal(t, "Configuration management", cmd.Short)

	subNames := make(map[string]bool)
	for _, sub := range cmd.Commands() {
		subNames[sub.Name()] = true
	}

	assert.True(t, subNames["validate"])
	assert.True(t, subNames["show"])
	assert.True(t, subNames["init"])
}

func TestConfigValidateCmd(t *testing.T) {
	cmd := newConfigValidateCmd()
	assert.Equal(t, "validate", cmd.Use)
	assert.Equal(t, "Validate dokrypt.yaml", cmd.Short)
}

func TestConfigShowCmd(t *testing.T) {
	cmd := newConfigShowCmd()
	assert.Equal(t, "show", cmd.Use)
	assert.Equal(t, "Show resolved configuration", cmd.Short)

	f := cmd.Flags().Lookup("raw")
	require.NotNil(t, f)
	assert.Equal(t, "false", f.DefValue)
}

func TestConfigInitCmd(t *testing.T) {
	cmd := newConfigInitCmd()
	assert.Equal(t, "init", cmd.Use)
	assert.Equal(t, "Generate default dokrypt.yaml in current directory", cmd.Short)
}

func TestGenerateDefaultYAML(t *testing.T) {
	yaml := generateDefaultYAML()

	assert.Contains(t, yaml, "name: my-project")
	assert.Contains(t, yaml, "version: \"1.0\"")
	assert.Contains(t, yaml, "chains:")
	assert.Contains(t, yaml, "ethereum:")
	assert.Contains(t, yaml, "engine: anvil")
	assert.Contains(t, yaml, "chain_id: 31337")
	assert.Contains(t, yaml, "block_time: 1s")
	assert.Contains(t, yaml, "accounts: 10")
}

func TestNewDoctorCmd(t *testing.T) {
	cmd := newDoctorCmd()

	assert.Equal(t, "doctor", cmd.Use)
	assert.Equal(t, "Check system requirements", cmd.Short)
	assert.NotNil(t, cmd.RunE)
}

func TestIsAPIVersionOK(t *testing.T) {
	tests := []struct {
		version  string
		expected bool
	}{
		{"1.41", true},
		{"1.42", true},
		{"1.45", true},
		{"1.40", false},
		{"1.39", false},
		{"1.0", false},
		{"2.0", true},
		{"2.1", true},
		{"0.1", false},
		{"invalid", false},
		{"", false},
	}

	for _, tt := range tests {
		t.Run(tt.version, func(t *testing.T) {
			assert.Equal(t, tt.expected, isAPIVersionOK(tt.version))
		})
	}
}

func TestNewForkCmd(t *testing.T) {
	cmd := newForkCmd()

	assert.Equal(t, "fork [network]", cmd.Use)
	assert.Equal(t, "Fork a live network", cmd.Short)
	assert.Contains(t, cmd.Long, "Fork a live blockchain network")
	assert.NotNil(t, cmd.RunE)

	err := cmd.Args(cmd, []string{})
	assert.NoError(t, err)
	err = cmd.Args(cmd, []string{"mainnet"})
	assert.NoError(t, err)
	err = cmd.Args(cmd, []string{"mainnet", "extra"})
	assert.Error(t, err)
}

func TestForkCmd_Flags(t *testing.T) {
	cmd := newForkCmd()

	tests := []struct {
		name         string
		defaultValue string
	}{
		{"url", ""},
		{"block", "0"},
		{"chain", ""},
		{"accounts", "10"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			f := cmd.Flags().Lookup(tt.name)
			require.NotNil(t, f, "flag %q should exist", tt.name)
			assert.Equal(t, tt.defaultValue, f.DefValue)
		})
	}
}

func TestKnownNetworks(t *testing.T) {
	expected := []string{
		"mainnet", "ethereum", "sepolia", "goerli",
		"polygon", "arbitrum", "optimism", "base", "bsc", "avalanche",
	}

	for _, name := range expected {
		t.Run(name, func(t *testing.T) {
			url, ok := knownNetworks[name]
			assert.True(t, ok, "network %q should exist in knownNetworks", name)
			assert.NotEmpty(t, url)
			assert.Contains(t, url, "https://")
		})
	}
}

func TestNewAccountsCmd(t *testing.T) {
	cmd := newAccountsCmd()

	assert.Equal(t, "accounts", cmd.Use)
	assert.Equal(t, "Account management", cmd.Short)

	f := cmd.PersistentFlags().Lookup("chain")
	require.NotNil(t, f)
	assert.Equal(t, "", f.DefValue)

	subNames := make(map[string]bool)
	for _, sub := range cmd.Commands() {
		subNames[sub.Name()] = true
	}

	assert.True(t, subNames["list"])
	assert.True(t, subNames["fund"])
	assert.True(t, subNames["impersonate"])
	assert.True(t, subNames["generate"])
}

func TestAccountsListCmd(t *testing.T) {
	chainName := ""
	cmd := newAccountsListCmd(&chainName)
	assert.Equal(t, "list", cmd.Use)
	assert.Equal(t, "List accounts with balances", cmd.Short)
}

func TestAccountsFundCmd(t *testing.T) {
	chainName := ""
	cmd := newAccountsFundCmd(&chainName)

	assert.Equal(t, "fund <address> <amount-eth>", cmd.Use)
	assert.Equal(t, "Fund an account with ETH", cmd.Short)

	err := cmd.Args(cmd, []string{"addr", "100"})
	assert.NoError(t, err)
	err = cmd.Args(cmd, []string{"addr"})
	assert.Error(t, err)
}

func TestAccountsImpersonateCmd(t *testing.T) {
	chainName := ""
	cmd := newAccountsImpersonateCmd(&chainName)

	assert.Equal(t, "impersonate <address>", cmd.Use)

	err := cmd.Args(cmd, []string{"0xabc"})
	assert.NoError(t, err)
	err = cmd.Args(cmd, []string{})
	assert.Error(t, err)
}

func TestAccountsGenerateCmd(t *testing.T) {
	chainName := ""
	cmd := newAccountsGenerateCmd(&chainName)

	assert.Equal(t, "generate <count>", cmd.Use)
	assert.Equal(t, "Generate and fund new accounts", cmd.Short)

	err := cmd.Args(cmd, []string{"5"})
	assert.NoError(t, err)
	err = cmd.Args(cmd, []string{})
	assert.Error(t, err)
}

func TestNewMarketplaceCmd(t *testing.T) {
	cmd := newMarketplaceCmd()

	assert.Equal(t, "marketplace", cmd.Use)
	assert.Equal(t, "Template marketplace — discover, install, and publish templates", cmd.Short)
	assert.Equal(t, []string{"market", "hub"}, cmd.Aliases)

	subNames := make(map[string]bool)
	for _, sub := range cmd.Commands() {
		subNames[sub.Name()] = true
	}

	expected := []string{"search", "browse", "install", "uninstall", "publish", "info", "list"}
	for _, name := range expected {
		assert.True(t, subNames[name], "marketplace should have subcommand %q", name)
	}
}

func TestMarketplaceSearchCmd(t *testing.T) {
	cmd := newMarketplaceSearchCmd()

	assert.Equal(t, "search <query>", cmd.Use)
	assert.Equal(t, "Search for templates", cmd.Short)

	err := cmd.Args(cmd, []string{"defi"})
	assert.NoError(t, err)
	err = cmd.Args(cmd, []string{})
	assert.Error(t, err)

	f := cmd.Flags().Lookup("remote")
	require.NotNil(t, f)
	assert.Equal(t, "false", f.DefValue)
}

func TestMarketplaceBrowseCmd(t *testing.T) {
	cmd := newMarketplaceBrowseCmd()

	assert.Equal(t, "browse", cmd.Use)

	f := cmd.Flags().Lookup("category")
	require.NotNil(t, f)
	assert.Equal(t, "", f.DefValue)

	f = cmd.Flags().Lookup("remote")
	require.NotNil(t, f)
	assert.Equal(t, "false", f.DefValue)
}

func TestMarketplaceInstallCmd(t *testing.T) {
	cmd := newMarketplaceInstallCmd()

	assert.Equal(t, "install <name>", cmd.Use)

	err := cmd.Args(cmd, []string{"my-template"})
	assert.NoError(t, err)
	err = cmd.Args(cmd, []string{})
	assert.Error(t, err)

	f := cmd.Flags().Lookup("from")
	require.NotNil(t, f)
	assert.Equal(t, "", f.DefValue)
}

func TestMarketplaceUninstallCmd(t *testing.T) {
	cmd := newMarketplaceUninstallCmd()

	assert.Equal(t, "uninstall <name>", cmd.Use)

	err := cmd.Args(cmd, []string{"my-template"})
	assert.NoError(t, err)
	err = cmd.Args(cmd, []string{})
	assert.Error(t, err)
}

func TestMarketplacePublishCmd(t *testing.T) {
	cmd := newMarketplacePublishCmd()
	assert.Equal(t, "publish", cmd.Use)
	assert.Equal(t, "Publish a template to the marketplace hub", cmd.Short)
}

func TestMarketplaceInfoCmd(t *testing.T) {
	cmd := newMarketplaceInfoCmd()

	assert.Equal(t, "info <name>", cmd.Use)

	err := cmd.Args(cmd, []string{"my-template"})
	assert.NoError(t, err)
	err = cmd.Args(cmd, []string{})
	assert.Error(t, err)

	f := cmd.Flags().Lookup("remote")
	require.NotNil(t, f)
	assert.Equal(t, "false", f.DefValue)
}

func TestMarketplaceListCmd(t *testing.T) {
	cmd := newMarketplaceListCmd()
	assert.Equal(t, "list", cmd.Use)
	assert.Equal(t, "List installed marketplace templates", cmd.Short)
}

func TestNewPluginCmd(t *testing.T) {
	cmd := newPluginCmd()

	assert.Equal(t, "plugin", cmd.Use)
	assert.Equal(t, "Plugin management", cmd.Short)

	subNames := make(map[string]bool)
	for _, sub := range cmd.Commands() {
		subNames[sub.Name()] = true
	}

	expected := []string{"install", "uninstall", "list", "search", "update", "create", "publish"}
	for _, name := range expected {
		assert.True(t, subNames[name], "plugin should have subcommand %q", name)
	}
}

func TestPluginInstallCmd(t *testing.T) {
	cmd := newPluginInstallCmd()

	assert.Equal(t, "install <name>", cmd.Use)
	assert.Equal(t, "Install a plugin", cmd.Short)

	err := cmd.Args(cmd, []string{"my-plugin"})
	assert.NoError(t, err)
	err = cmd.Args(cmd, []string{})
	assert.Error(t, err)

	f := cmd.Flags().Lookup("version")
	require.NotNil(t, f)
	assert.Equal(t, "", f.DefValue)

	f = cmd.Flags().Lookup("global")
	require.NotNil(t, f)
	assert.Equal(t, "false", f.DefValue)
}

func TestPluginUninstallCmd(t *testing.T) {
	cmd := newPluginUninstallCmd()

	assert.Equal(t, "uninstall <name>", cmd.Use)
	assert.Equal(t, "Remove a plugin", cmd.Short)

	err := cmd.Args(cmd, []string{"my-plugin"})
	assert.NoError(t, err)
}

func TestPluginListCmd(t *testing.T) {
	cmd := newPluginListCmd()
	assert.Equal(t, "list", cmd.Use)
	assert.Equal(t, "List installed plugins", cmd.Short)
}

func TestPluginSearchCmd(t *testing.T) {
	cmd := newPluginSearchCmd()

	assert.Equal(t, "search <query>", cmd.Use)
	assert.Equal(t, "Search for plugins", cmd.Short)

	err := cmd.Args(cmd, []string{"oracle"})
	assert.NoError(t, err)
}

func TestPluginUpdateCmd(t *testing.T) {
	cmd := newPluginUpdateCmd()

	assert.Equal(t, "update <name>", cmd.Use)
	assert.Equal(t, "Update a plugin", cmd.Short)

	err := cmd.Args(cmd, []string{"my-plugin"})
	assert.NoError(t, err)

	f := cmd.Flags().Lookup("version")
	require.NotNil(t, f)
	assert.Equal(t, "", f.DefValue)
}

func TestPluginCreateCmd(t *testing.T) {
	cmd := newPluginCreateCmd()

	assert.Equal(t, "create <name>", cmd.Use)
	assert.Equal(t, "Scaffold a new plugin", cmd.Short)

	err := cmd.Args(cmd, []string{"my-plugin"})
	assert.NoError(t, err)
}

func TestPluginPublishCmd(t *testing.T) {
	cmd := newPluginPublishCmd()
	assert.Equal(t, "publish", cmd.Use)
	assert.Equal(t, "Publish plugin to registry", cmd.Short)
}

func TestNewSnapshotCmd(t *testing.T) {
	cmd := newSnapshotCmd()

	assert.Equal(t, "snapshot", cmd.Use)
	assert.Equal(t, "State snapshot management", cmd.Short)

	subNames := make(map[string]bool)
	for _, sub := range cmd.Commands() {
		subNames[sub.Name()] = true
	}

	expected := []string{"save", "restore", "list", "delete", "export", "import", "diff"}
	for _, name := range expected {
		assert.True(t, subNames[name], "snapshot should have subcommand %q", name)
	}
}

func TestSnapshotSaveCmd(t *testing.T) {
	cmd := newSnapshotSaveCmd()

	assert.Equal(t, "save <name>", cmd.Use)
	assert.Equal(t, "Save current state as snapshot", cmd.Short)

	err := cmd.Args(cmd, []string{"my-snap"})
	assert.NoError(t, err)
	err = cmd.Args(cmd, []string{})
	assert.Error(t, err)

	f := cmd.Flags().Lookup("description")
	require.NotNil(t, f)
	assert.Equal(t, "", f.DefValue)

	f = cmd.Flags().Lookup("tags")
	require.NotNil(t, f)
	assert.Equal(t, "[]", f.DefValue)
}

func TestSnapshotRestoreCmd(t *testing.T) {
	cmd := newSnapshotRestoreCmd()

	assert.Equal(t, "restore <name>", cmd.Use)

	err := cmd.Args(cmd, []string{"my-snap"})
	assert.NoError(t, err)
}

func TestSnapshotListCmd(t *testing.T) {
	cmd := newSnapshotListCmd()
	assert.Equal(t, "list", cmd.Use)
	assert.Equal(t, "List all snapshots", cmd.Short)
}

func TestSnapshotDeleteCmd(t *testing.T) {
	cmd := newSnapshotDeleteCmd()

	assert.Equal(t, "delete <name>", cmd.Use)

	err := cmd.Args(cmd, []string{"my-snap"})
	assert.NoError(t, err)
}

func TestSnapshotExportCmd(t *testing.T) {
	cmd := newSnapshotExportCmd()

	assert.Equal(t, "export <name> <path>", cmd.Use)

	err := cmd.Args(cmd, []string{"snap", "/tmp/out.json"})
	assert.NoError(t, err)
	err = cmd.Args(cmd, []string{"snap"})
	assert.Error(t, err)
}

func TestSnapshotImportCmd(t *testing.T) {
	cmd := newSnapshotImportCmd()

	assert.Equal(t, "import <path>", cmd.Use)

	err := cmd.Args(cmd, []string{"/tmp/snap.json"})
	assert.NoError(t, err)
}

func TestSnapshotDiffCmd(t *testing.T) {
	cmd := newSnapshotDiffCmd()

	assert.Equal(t, "diff <snap1> <snap2>", cmd.Use)

	err := cmd.Args(cmd, []string{"snap1", "snap2"})
	assert.NoError(t, err)
	err = cmd.Args(cmd, []string{"snap1"})
	assert.Error(t, err)
}

func TestNewTemplateCmd(t *testing.T) {
	cmd := newTemplateCmd()

	assert.Equal(t, "template", cmd.Use)
	assert.Equal(t, "Template management", cmd.Short)

	subNames := make(map[string]bool)
	for _, sub := range cmd.Commands() {
		subNames[sub.Name()] = true
	}

	expected := []string{"list", "info", "pull", "push", "create"}
	for _, name := range expected {
		assert.True(t, subNames[name], "template should have subcommand %q", name)
	}
}

func TestTemplateListCmd(t *testing.T) {
	cmd := newTemplateListCmd()
	assert.Equal(t, "list", cmd.Use)
	assert.Equal(t, "List available templates", cmd.Short)
}

func TestTemplateInfoCmd(t *testing.T) {
	cmd := newTemplateInfoCmd()

	assert.Equal(t, "info <name>", cmd.Use)

	err := cmd.Args(cmd, []string{"evm-basic"})
	assert.NoError(t, err)
	err = cmd.Args(cmd, []string{})
	assert.Error(t, err)
}

func TestTemplatePullCmd(t *testing.T) {
	cmd := newTemplatePullCmd()

	assert.Equal(t, "pull <name>", cmd.Use)

	err := cmd.Args(cmd, []string{"my-template"})
	assert.NoError(t, err)
}

func TestTemplatePushCmd(t *testing.T) {
	cmd := newTemplatePushCmd()
	assert.Equal(t, "push", cmd.Use)
	assert.Equal(t, "Publish a template to the marketplace", cmd.Short)
}

func TestTemplateCreateCmd(t *testing.T) {
	cmd := newTemplateCreateCmd()

	assert.Equal(t, "create <name>", cmd.Use)
	assert.Contains(t, cmd.Long, "template.yaml")

	err := cmd.Args(cmd, []string{"my-tmpl"})
	assert.NoError(t, err)
}

func TestNewTestCmd(t *testing.T) {
	cmd := newTestCmd()

	assert.Equal(t, "test", cmd.Use)
	assert.Equal(t, "Built-in test runner", cmd.Short)

	subNames := make(map[string]bool)
	for _, sub := range cmd.Commands() {
		subNames[sub.Name()] = true
	}

	assert.True(t, subNames["run"])
	assert.True(t, subNames["list"])
	assert.True(t, subNames["report"])
}

func TestTestRunCmd(t *testing.T) {
	cmd := newTestRunCmd()

	assert.Equal(t, "run", cmd.Use)
	assert.Equal(t, "Run all tests", cmd.Short)

	tests := []struct {
		name         string
		defaultValue string
	}{
		{"suite", ""},
		{"filter", ""},
		{"parallel", "4"},
		{"gas-report", "false"},
		{"coverage", "false"},
		{"snapshot", "false"},
		{"timeout", "0s"},
		{"json", "false"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			f := cmd.Flags().Lookup(tt.name)
			require.NotNil(t, f, "flag %q should exist", tt.name)
			assert.Equal(t, tt.defaultValue, f.DefValue, "default for %q", tt.name)
		})
	}
}

func TestTestListCmd(t *testing.T) {
	cmd := newTestListCmd()
	assert.Equal(t, "list", cmd.Use)
	assert.Equal(t, "List test suites", cmd.Short)
}

func TestTestReportCmd(t *testing.T) {
	cmd := newTestReportCmd()
	assert.Equal(t, "report", cmd.Use)
	assert.Equal(t, "Show last test report", cmd.Short)
}

func TestHostPortFromURL(t *testing.T) {
	tests := []struct {
		url      string
		expected int
	}{
		{"http://localhost:8545", 8545},
		{"http://localhost:5001", 5001},
		{"http://127.0.0.1:3000", 3000},
		{"https://example.com:443", 443},
		{"http://localhost", 0},
		{"", 0},
		{"noport", 0},
	}

	for _, tt := range tests {
		t.Run(tt.url, func(t *testing.T) {
			assert.Equal(t, tt.expected, hostPortFromURL(tt.url))
		})
	}
}

func TestParseDurationStr(t *testing.T) {
	tests := []struct {
		input    string
		expected int64
		hasError bool
	}{
		{"3600", 3600, false},
		{"60", 60, false},
		{"0", 0, false},

		{"1h", 3600, false},
		{"30m", 1800, false},
		{"10s", 10, false},
		{"1h30m", 5400, false},

		{"5d", 432000, false},
		{"7d", 604800, false},

		{"", 0, true},
		{"abc", 0, true},
		{"5x", 0, true},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			result, err := parseDurationStr(tt.input)
			if tt.hasError {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
				assert.Equal(t, tt.expected, result)
			}
		})
	}
}

func TestContainerNames(t *testing.T) {
	state := &ProjectState{
		Containers: map[string]ContainerState{
			"ethereum": {ContainerID: "abc"},
			"ipfs":     {ContainerID: "def"},
		},
	}

	names := containerNames(state)
	assert.Contains(t, names, "ethereum")
	assert.Contains(t, names, "ipfs")
	assert.Contains(t, names, ", ")
}

func TestContainerNames_Empty(t *testing.T) {
	state := &ProjectState{
		Containers: map[string]ContainerState{},
	}
	assert.Equal(t, "", containerNames(state))
}

func TestContainerNames_Single(t *testing.T) {
	state := &ProjectState{
		Containers: map[string]ContainerState{
			"ethereum": {ContainerID: "abc"},
		},
	}
	assert.Equal(t, "ethereum", containerNames(state))
}

func TestValueOrDefault(t *testing.T) {
	assert.Equal(t, "hello", valueOrDefault("hello", "fallback"))
	assert.Equal(t, "fallback", valueOrDefault("", "fallback"))
}

func TestIsTestFile(t *testing.T) {
	tests := []struct {
		name     string
		expected bool
	}{
		{"Counter.t.sol", true},
		{"Token.test.sol", true},
		{"MyContract_test.sol", true},
		{"transfer.test.js", true},
		{"deploy.test.ts", true},
		{"Counter.sol", false},
		{"deploy.js", false},
		{"config.ts", false},
		{"README.md", false},
		{"test.go", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.expected, isTestFile(tt.name))
		})
	}
}

func TestTestCommandForFile(t *testing.T) {
	tests := []struct {
		path        string
		expectCmd   string
		expectEmpty bool
	}{
		{"/project/test/Counter.t.sol", "forge", false},
		{"/project/test/Token.test.sol", "forge", false},
		{"/project/test/NFT_test.sol", "forge", false},
		{"/project/test/README.md", "", true},
	}

	for _, tt := range tests {
		t.Run(tt.path, func(t *testing.T) {
			cmd, args := testCommandForFile(tt.path)
			if tt.expectEmpty {
				assert.Empty(t, cmd)
				assert.Nil(t, args)
			} else {
				assert.Equal(t, tt.expectCmd, cmd)
				assert.NotEmpty(t, args)
			}
		})
	}
}

func TestProjectStateTypes(t *testing.T) {
	now := time.Now()
	state := &ProjectState{
		Project:   "test-project",
		StartedAt: now,
		Containers: map[string]ContainerState{
			"ethereum": {
				ContainerID:   "abc123",
				ContainerName: "dokrypt-test-ethereum",
				Image:         "ghcr.io/foundry-rs/foundry:latest",
				Ports:         map[string]int{"8545": 8545},
				Status:        "running",
			},
		},
		NetworkID: "net123",
	}

	assert.Equal(t, "test-project", state.Project)
	assert.Equal(t, now, state.StartedAt)
	assert.Equal(t, "abc123", state.Containers["ethereum"].ContainerID)
	assert.Equal(t, "dokrypt-test-ethereum", state.Containers["ethereum"].ContainerName)
	assert.Equal(t, 8545, state.Containers["ethereum"].Ports["8545"])
	assert.Equal(t, "running", state.Containers["ethereum"].Status)
}

func TestSnapshotMetadataTypes(t *testing.T) {
	now := time.Now()
	meta := SnapshotMetadata{
		Name:        "initial",
		Project:     "my-project",
		Description: "Initial state",
		Tags:        []string{"genesis", "clean"},
		CreatedAt:   now,
		ChainID:     31337,
		BlockNumber: 42,
		SnapshotID:  "0x1",
	}

	assert.Equal(t, "initial", meta.Name)
	assert.Equal(t, "my-project", meta.Project)
	assert.Equal(t, "Initial state", meta.Description)
	assert.Equal(t, []string{"genesis", "clean"}, meta.Tags)
	assert.Equal(t, uint64(31337), meta.ChainID)
	assert.Equal(t, uint64(42), meta.BlockNumber)
	assert.Equal(t, "0x1", meta.SnapshotID)
}

func TestBuildVariableDefaults(t *testing.T) {
	assert.Equal(t, "dev", Version)
	assert.Equal(t, "none", Commit)
	assert.Equal(t, "unknown", Date)
}

func TestGetConfigPath_DefaultsToYaml(t *testing.T) {
	oldCfgFile := cfgFile
	cfgFile = ""
	defer func() { cfgFile = oldCfgFile }()

	t.Setenv("DOKRYPT_CONFIG", "")
	assert.Equal(t, "dokrypt.yaml", getConfigPath())
}

func TestGetConfigPath_PrefersFlag(t *testing.T) {
	oldCfgFile := cfgFile
	cfgFile = "/custom/path/config.yaml"
	defer func() { cfgFile = oldCfgFile }()

	assert.Equal(t, "/custom/path/config.yaml", getConfigPath())
}

func TestGetConfigPath_FallsBackToEnv(t *testing.T) {
	oldCfgFile := cfgFile
	cfgFile = ""
	defer func() { cfgFile = oldCfgFile }()

	t.Setenv("DOKRYPT_CONFIG", "/env/dokrypt.yaml")
	assert.Equal(t, "/env/dokrypt.yaml", getConfigPath())
}

func TestGetConfigPath_FlagWinsOverEnv(t *testing.T) {
	oldCfgFile := cfgFile
	cfgFile = "/flag/config.yaml"
	defer func() { cfgFile = oldCfgFile }()

	t.Setenv("DOKRYPT_CONFIG", "/env/config.yaml")
	assert.Equal(t, "/flag/config.yaml", getConfigPath())
}

func TestGetOutput_ReturnsNonNil(t *testing.T) {
	oldOutput := output
	output = nil
	defer func() { output = oldOutput }()

	out := getOutput()
	assert.NotNil(t, out, "getOutput should return a non-nil output even if global is nil")
}

func TestServiceColors(t *testing.T) {
	assert.Len(t, serviceColors, 5, "should have 5 service colors")
	assert.Equal(t, "\033[0m", colorReset)
}

func TestUpCmd_FlagParsing(t *testing.T) {
	cmd := newUpCmd()
	cmd.RunE = func(c *cobra.Command, args []string) error { return nil }

	cmd.SetArgs([]string{
		"--detach",
		"--build",
		"--service", "ethereum",
		"--service", "ipfs",
		"--fresh",
		"--fork", "mainnet",
		"--fork-block", "12345",
		"--snapshot", "my-snap",
		"--profile", "dev",
		"--timeout", "10m",
	})

	err := cmd.Execute()
	require.NoError(t, err)

	assert.Equal(t, "true", cmd.Flags().Lookup("detach").Value.String())
	assert.Equal(t, "true", cmd.Flags().Lookup("build").Value.String())
	assert.Equal(t, "[ethereum,ipfs]", cmd.Flags().Lookup("service").Value.String())
	assert.Equal(t, "true", cmd.Flags().Lookup("fresh").Value.String())
	assert.Equal(t, "mainnet", cmd.Flags().Lookup("fork").Value.String())
	assert.Equal(t, "12345", cmd.Flags().Lookup("fork-block").Value.String())
	assert.Equal(t, "my-snap", cmd.Flags().Lookup("snapshot").Value.String())
	assert.Equal(t, "dev", cmd.Flags().Lookup("profile").Value.String())
	assert.Equal(t, "10m0s", cmd.Flags().Lookup("timeout").Value.String())
}

func TestDownCmd_FlagParsing(t *testing.T) {
	cmd := newDownCmd()
	cmd.RunE = func(c *cobra.Command, args []string) error { return nil }

	cmd.SetArgs([]string{"--volumes", "--service", "ethereum", "--timeout", "1m"})
	err := cmd.Execute()
	require.NoError(t, err)

	assert.Equal(t, "true", cmd.Flags().Lookup("volumes").Value.String())
	assert.Equal(t, "[ethereum]", cmd.Flags().Lookup("service").Value.String())
	assert.Equal(t, "1m0s", cmd.Flags().Lookup("timeout").Value.String())
}

func TestLogsCmd_FlagParsing(t *testing.T) {
	cmd := newLogsCmd()
	cmd.RunE = func(c *cobra.Command, args []string) error { return nil }

	cmd.SetArgs([]string{"--service", "ethereum", "--follow", "--tail", "100", "--since", "5m", "--timestamps"})
	err := cmd.Execute()
	require.NoError(t, err)

	assert.Equal(t, "ethereum", cmd.Flags().Lookup("service").Value.String())
	assert.Equal(t, "true", cmd.Flags().Lookup("follow").Value.String())
	assert.Equal(t, "100", cmd.Flags().Lookup("tail").Value.String())
	assert.Equal(t, "5m", cmd.Flags().Lookup("since").Value.String())
	assert.Equal(t, "true", cmd.Flags().Lookup("timestamps").Value.String())
}

func TestTestRunCmd_FlagParsing(t *testing.T) {
	cmd := newTestRunCmd()
	cmd.RunE = func(c *cobra.Command, args []string) error { return nil }

	cmd.SetArgs([]string{
		"--suite", "unit",
		"--filter", "transfer",
		"--parallel", "8",
		"--gas-report",
		"--coverage",
		"--snapshot",
		"--timeout", "2m",
		"--json",
	})

	err := cmd.Execute()
	require.NoError(t, err)

	assert.Equal(t, "unit", cmd.Flags().Lookup("suite").Value.String())
	assert.Equal(t, "transfer", cmd.Flags().Lookup("filter").Value.String())
	assert.Equal(t, "8", cmd.Flags().Lookup("parallel").Value.String())
	assert.Equal(t, "true", cmd.Flags().Lookup("gas-report").Value.String())
	assert.Equal(t, "true", cmd.Flags().Lookup("coverage").Value.String())
	assert.Equal(t, "true", cmd.Flags().Lookup("snapshot").Value.String())
	assert.Equal(t, "2m0s", cmd.Flags().Lookup("timeout").Value.String())
	assert.Equal(t, "true", cmd.Flags().Lookup("json").Value.String())
}
