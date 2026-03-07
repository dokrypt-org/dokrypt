package chain

import (
	"math/big"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestResolveNetworkRPC_KnownNetworks(t *testing.T) {
	tests := []struct {
		network  string
		expected string
	}{
		{"mainnet", "https://eth.llamarpc.com"},
		{"sepolia", "https://rpc.sepolia.org"},
		{"holesky", "https://ethereum-holesky-rpc.publicnode.com"},
		{"polygon", "https://polygon-rpc.com"},
		{"arbitrum", "https://arb1.arbitrum.io/rpc"},
		{"optimism", "https://mainnet.optimism.io"},
		{"base", "https://mainnet.base.org"},
		{"bsc", "https://bsc-dataseed.binance.org"},
		{"avalanche", "https://api.avax.network/ext/bc/C/rpc"},
	}

	for _, tt := range tests {
		t.Run(tt.network, func(t *testing.T) {
			result := ResolveNetworkRPC(tt.network)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestResolveNetworkRPC_UnknownNetwork(t *testing.T) {
	rawURL := "https://my-custom-node.example.com/rpc"
	result := ResolveNetworkRPC(rawURL)
	assert.Equal(t, rawURL, result)
}

func TestResolveNetworkRPC_EmptyString(t *testing.T) {
	result := ResolveNetworkRPC("")
	assert.Equal(t, "", result)
}

func TestDefaultNetworks_ContainsExpectedEntries(t *testing.T) {
	expectedNetworks := []string{"mainnet", "sepolia", "holesky", "polygon", "arbitrum", "optimism", "base", "bsc", "avalanche"}
	for _, name := range expectedNetworks {
		_, ok := DefaultNetworks[name]
		assert.True(t, ok, "DefaultNetworks should contain %q", name)
	}
}

func TestDefaultNetworks_ChainIDs(t *testing.T) {
	tests := []struct {
		network string
		chainID uint64
	}{
		{"mainnet", 1},
		{"sepolia", 11155111},
		{"holesky", 17000},
		{"polygon", 137},
		{"arbitrum", 42161},
		{"optimism", 10},
		{"base", 8453},
		{"bsc", 56},
		{"avalanche", 43114},
	}

	for _, tt := range tests {
		t.Run(tt.network, func(t *testing.T) {
			cfg := DefaultNetworks[tt.network]
			assert.Equal(t, tt.chainID, cfg.ChainID)
		})
	}
}

func TestNetworkConfig_Fields(t *testing.T) {
	cfg := NetworkConfig{
		ChainID: 42,
		RPC:     "https://example.com/rpc",
	}
	assert.Equal(t, uint64(42), cfg.ChainID)
	assert.Equal(t, "https://example.com/rpc", cfg.RPC)
}

func TestAccount_Fields(t *testing.T) {
	balance := big.NewInt(1000000000000000000) // 1 ETH in wei
	acct := Account{
		Address:    "0x1234567890abcdef1234567890abcdef12345678",
		PrivateKey: "0xabcdef1234567890",
		Balance:    balance,
		Label:      "deployer",
	}

	assert.Equal(t, "0x1234567890abcdef1234567890abcdef12345678", acct.Address)
	assert.Equal(t, "0xabcdef1234567890", acct.PrivateKey)
	assert.Equal(t, balance, acct.Balance)
	assert.Equal(t, "deployer", acct.Label)
}

func TestForkOptions_Fields(t *testing.T) {
	opts := ForkOptions{
		Network:     "mainnet",
		BlockNumber: 18000000,
		ChainID:     1,
		Accounts:    10,
	}

	assert.Equal(t, "mainnet", opts.Network)
	assert.Equal(t, uint64(18000000), opts.BlockNumber)
	assert.Equal(t, uint64(1), opts.ChainID)
	assert.Equal(t, 10, opts.Accounts)
}

func TestForkInfo_Fields(t *testing.T) {
	info := ForkInfo{
		Network:     "sepolia",
		BlockNumber: 5000000,
		RPCURL:      "https://rpc.sepolia.org",
	}

	assert.Equal(t, "sepolia", info.Network)
	assert.Equal(t, uint64(5000000), info.BlockNumber)
	assert.Equal(t, "https://rpc.sepolia.org", info.RPCURL)
}
