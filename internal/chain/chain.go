package chain

import (
	"context"
	"encoding/json"
	"io"
	"math/big"
)

type Chain interface {
	Start(ctx context.Context) error
	Stop(ctx context.Context) error
	IsRunning(ctx context.Context) bool
	Health(ctx context.Context) error

	Name() string
	ChainID() uint64
	RPCURL() string
	WSURL() string
	Engine() string // "anvil", "hardhat", "geth"

	Accounts() []Account
	FundAccount(ctx context.Context, address string, amountWei *big.Int) error
	ImpersonateAccount(ctx context.Context, address string) error
	GenerateAccounts(ctx context.Context, count int) ([]Account, error)

	MineBlocks(ctx context.Context, count uint64) error
	SetBlockTime(ctx context.Context, seconds uint64) error
	SetGasPrice(ctx context.Context, gweiPrice uint64) error
	TimeTravel(ctx context.Context, seconds int64) error
	SetBalance(ctx context.Context, address string, amountWei *big.Int) error
	SetStorageAt(ctx context.Context, address string, slot string, value string) error

	TakeSnapshot(ctx context.Context) (string, error)
	RevertSnapshot(ctx context.Context, id string) error
	ExportState(ctx context.Context, path string) error
	ImportState(ctx context.Context, path string) error

	Fork(ctx context.Context, opts ForkOptions) error
	ForkInfo() *ForkInfo

	RPC(ctx context.Context, method string, params ...any) (json.RawMessage, error)

	Logs(ctx context.Context, follow bool) (io.ReadCloser, error)
}

type Account struct {
	Address    string   `json:"address"`
	PrivateKey string   `json:"private_key"`
	Balance    *big.Int `json:"balance"`
	Label      string   `json:"label"`
}

type ForkOptions struct {
	Network     string // "mainnet", "sepolia", or RPC URL
	BlockNumber uint64 // 0 = latest
	ChainID     uint64 // Override chain ID
	Accounts    int    // Number of funded accounts
}

type ForkInfo struct {
	Network     string `json:"network"`
	BlockNumber uint64 `json:"block_number"`
	RPCURL      string `json:"rpc_url"`
}

var DefaultNetworks = map[string]NetworkConfig{
	"mainnet":   {ChainID: 1, RPC: "https://eth.llamarpc.com"},
	"sepolia":   {ChainID: 11155111, RPC: "https://rpc.sepolia.org"},
	"holesky":   {ChainID: 17000, RPC: "https://ethereum-holesky-rpc.publicnode.com"},
	"polygon":   {ChainID: 137, RPC: "https://polygon-rpc.com"},
	"arbitrum":  {ChainID: 42161, RPC: "https://arb1.arbitrum.io/rpc"},
	"optimism":  {ChainID: 10, RPC: "https://mainnet.optimism.io"},
	"base":      {ChainID: 8453, RPC: "https://mainnet.base.org"},
	"bsc":       {ChainID: 56, RPC: "https://bsc-dataseed.binance.org"},
	"avalanche": {ChainID: 43114, RPC: "https://api.avax.network/ext/bc/C/rpc"},
}

type NetworkConfig struct {
	ChainID uint64
	RPC     string
}

func ResolveNetworkRPC(network string) string {
	if cfg, ok := DefaultNetworks[network]; ok {
		return cfg.RPC
	}
	return network
}
