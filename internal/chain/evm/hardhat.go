package evm

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"math/big"
	"time"

	"github.com/dokrypt/dokrypt/internal/chain"
	"github.com/dokrypt/dokrypt/internal/common"
	"github.com/dokrypt/dokrypt/internal/config"
	"github.com/dokrypt/dokrypt/internal/container"
)

const (
	hardhatImage       = "node:20-alpine"
	hardhatDefaultPort = 8545
)

type HardhatChain struct {
	name        string
	cfg         config.ChainConfig
	runtime     container.Runtime
	projectName string
	containerID string
	rpcClient   *RPCClient
	accounts    []chain.Account
	forkInfo    *chain.ForkInfo
	hostPort    int
}

func NewHardhatChain(name string, cfg config.ChainConfig, runtime container.Runtime, projectName string) (*HardhatChain, error) {
	return &HardhatChain{
		name:        name,
		cfg:         cfg,
		runtime:     runtime,
		projectName: projectName,
		hostPort:    8545,
	}, nil
}

func (h *HardhatChain) Start(ctx context.Context) error {
	slog.Info("starting hardhat chain", "chain", h.name)

	if err := h.runtime.PullImage(ctx, hardhatImage); err != nil {
		slog.Warn("failed to pull hardhat image", "error", err)
	}

	containerName := fmt.Sprintf("dokrypt-%s-%s", h.projectName, h.name)
	_ = h.runtime.StopContainer(ctx, containerName, 5*time.Second)
	_ = h.runtime.RemoveContainer(ctx, containerName, true)

	cmd := []string{
		"sh", "-c",
		"npx --yes hardhat node --hostname 0.0.0.0 --port 8545",
	}

	networkName := fmt.Sprintf("dokrypt-%s", h.projectName)
	id, err := h.runtime.CreateContainer(ctx, &container.ContainerConfig{
		Name:    containerName,
		Image:   hardhatImage,
		Command: cmd,
		Ports:   map[int]int{hardhatDefaultPort: 0},
		Networks: []string{networkName},
		NetworkAliases: map[string][]string{
			networkName: {h.name, containerName},
		},
		Labels: map[string]string{
			"dokrypt.project": h.projectName,
			"dokrypt.chain":   h.name,
			"dokrypt.engine":  "hardhat",
		},
	})
	if err != nil {
		return fmt.Errorf("failed to create hardhat container: %w", err)
	}
	h.containerID = id

	if err := h.runtime.StartContainer(ctx, id); err != nil {
		return fmt.Errorf("failed to start hardhat container: %w", err)
	}

	info, err := h.runtime.InspectContainer(ctx, id)
	if err != nil {
		return err
	}
	if hp, ok := info.Ports[hardhatDefaultPort]; ok {
		h.hostPort = hp
	}

	h.rpcClient = NewRPCClient(h.RPCURL())

	if err := h.waitForReady(ctx); err != nil {
		return fmt.Errorf("hardhat failed to become ready: %w", err)
	}

	accounts, err := h.generateAccounts(ctx)
	if err != nil {
		slog.Warn("failed to generate accounts", "error", err)
	} else {
		h.accounts = accounts
	}

	slog.Info("hardhat chain started", "chain", h.name, "rpc", h.RPCURL())
	return nil
}

func (h *HardhatChain) Stop(ctx context.Context) error {
	if h.containerID == "" {
		return nil
	}
	if err := h.runtime.StopContainer(ctx, h.containerID, 10*time.Second); err != nil {
		slog.Warn("failed to stop hardhat container", "error", err)
	}
	if err := h.runtime.RemoveContainer(ctx, h.containerID, true); err != nil {
		return err
	}
	h.containerID = ""
	return nil
}

func (h *HardhatChain) IsRunning(ctx context.Context) bool {
	if h.containerID == "" {
		return false
	}
	info, err := h.runtime.InspectContainer(ctx, h.containerID)
	if err != nil {
		return false
	}
	return info.State == "running"
}

func (h *HardhatChain) Health(ctx context.Context) error {
	if h.rpcClient == nil {
		return fmt.Errorf("chain not started")
	}
	_, err := h.rpcClient.Call(ctx, "eth_blockNumber")
	return err
}

func (h *HardhatChain) Name() string                  { return h.name }
func (h *HardhatChain) ChainID() uint64               { return h.cfg.ChainID }
func (h *HardhatChain) RPCURL() string                { return fmt.Sprintf("http://localhost:%d", h.hostPort) }
func (h *HardhatChain) WSURL() string                 { return fmt.Sprintf("ws://localhost:%d", h.hostPort) }
func (h *HardhatChain) Engine() string                { return "hardhat" }
func (h *HardhatChain) ContainerID() string           { return h.containerID }
func (h *HardhatChain) Accounts() []chain.Account     { return h.accounts }
func (h *HardhatChain) ForkInfo() *chain.ForkInfo  { return h.forkInfo }

func (h *HardhatChain) FundAccount(ctx context.Context, address string, amountWei *big.Int) error {
	hexAmount := fmt.Sprintf("0x%x", amountWei)
	_, err := h.rpcClient.Call(ctx, "hardhat_setBalance", address, hexAmount)
	return err
}

func (h *HardhatChain) ImpersonateAccount(ctx context.Context, address string) error {
	_, err := h.rpcClient.Call(ctx, "hardhat_impersonateAccount", address)
	return err
}

func (h *HardhatChain) GenerateAccounts(ctx context.Context, count int) ([]chain.Account, error) {
	return h.generateAccounts(ctx)
}

func (h *HardhatChain) MineBlocks(ctx context.Context, count uint64) error {
	hexCount := fmt.Sprintf("0x%x", count)
	_, err := h.rpcClient.Call(ctx, "hardhat_mine", hexCount)
	return err
}

func (h *HardhatChain) SetBlockTime(ctx context.Context, seconds uint64) error {
	_, err := h.rpcClient.Call(ctx, "hardhat_setIntervalMining", seconds)
	return err
}

func (h *HardhatChain) SetGasPrice(ctx context.Context, gweiPrice uint64) error {
	weiPrice := gweiPrice * 1e9
	hexPrice := fmt.Sprintf("0x%x", weiPrice)
	_, err := h.rpcClient.Call(ctx, "hardhat_setNextBlockBaseFeePerGas", hexPrice)
	return err
}

func (h *HardhatChain) TimeTravel(ctx context.Context, seconds int64) error {
	hexSeconds := fmt.Sprintf("0x%x", seconds)
	_, err := h.rpcClient.Call(ctx, "evm_increaseTime", hexSeconds)
	if err != nil {
		return err
	}
	return h.MineBlocks(ctx, 1)
}

func (h *HardhatChain) SetBalance(ctx context.Context, address string, amountWei *big.Int) error {
	return h.FundAccount(ctx, address, amountWei)
}

func (h *HardhatChain) SetStorageAt(ctx context.Context, address, slot, value string) error {
	_, err := h.rpcClient.Call(ctx, "hardhat_setStorageAt", address, slot, value)
	return err
}

func (h *HardhatChain) TakeSnapshot(ctx context.Context) (string, error) {
	var result string
	err := h.rpcClient.CallResult(ctx, &result, "evm_snapshot")
	return result, err
}

func (h *HardhatChain) RevertSnapshot(ctx context.Context, id string) error {
	_, err := h.rpcClient.Call(ctx, "evm_revert", id)
	return err
}

func (h *HardhatChain) ExportState(ctx context.Context, path string) error {
	result, err := h.rpcClient.Call(ctx, "hardhat_dumpState")
	if err != nil {
		return fmt.Errorf("hardhat_dumpState failed (requires Hardhat 2.12+): %w", err)
	}
	return writeFile(path, result)
}

func (h *HardhatChain) ImportState(ctx context.Context, path string) error {
	data, err := readFile(path)
	if err != nil {
		return err
	}
	var stateStr string
	if jsonErr := json.Unmarshal(data, &stateStr); jsonErr != nil {
		stateStr = string(data)
	}
	_, err = h.rpcClient.Call(ctx, "hardhat_loadState", stateStr)
	if err != nil {
		return fmt.Errorf("hardhat_loadState failed (requires Hardhat 2.12+): %w", err)
	}
	return nil
}

func (h *HardhatChain) Fork(ctx context.Context, opts chain.ForkOptions) error {
	rpcURL := chain.ResolveNetworkRPC(opts.Network)
	params := map[string]any{
		"forking": map[string]any{
			"jsonRpcUrl": rpcURL,
		},
	}
	if opts.BlockNumber > 0 {
		params["forking"].(map[string]any)["blockNumber"] = fmt.Sprintf("0x%x", opts.BlockNumber)
	}
	_, err := h.rpcClient.Call(ctx, "hardhat_reset", params)
	if err != nil {
		return err
	}
	h.forkInfo = &chain.ForkInfo{Network: opts.Network, BlockNumber: opts.BlockNumber, RPCURL: rpcURL}
	return nil
}

func (h *HardhatChain) RPC(ctx context.Context, method string, params ...any) (json.RawMessage, error) {
	return h.rpcClient.Call(ctx, method, params...)
}

func (h *HardhatChain) Logs(ctx context.Context, follow bool) (io.ReadCloser, error) {
	if h.containerID == "" {
		return nil, fmt.Errorf("chain not started")
	}
	return h.runtime.ContainerLogs(ctx, h.containerID, container.LogOptions{Follow: follow, Stdout: true, Stderr: true})
}

func (h *HardhatChain) waitForReady(ctx context.Context) error {
	return common.Retry(ctx, common.RetryConfig{
		MaxAttempts: 60, InitialDelay: 500 * time.Millisecond, MaxDelay: 3 * time.Second, Multiplier: 1.5, Jitter: 0.1,
	}, func(ctx context.Context) error { return h.Health(ctx) })
}

func (h *HardhatChain) generateAccounts(ctx context.Context) ([]chain.Account, error) {
	var addresses []string
	if err := h.rpcClient.CallResult(ctx, &addresses, "eth_accounts"); err != nil {
		return nil, fmt.Errorf("failed to fetch accounts: %w", err)
	}

	keyInfos, _ := common.GenerateAccounts(common.DefaultSeed, len(addresses))
	keyMap := make(map[string]string) // address → private key
	for _, info := range keyInfos {
		keyMap[info.Address] = info.PrivateKey
	}

	balWei := new(big.Int)
	if h.cfg.Balance != "" {
		balWei.SetString(h.cfg.Balance, 10)
	} else {
		balanceEth := h.cfg.AccountBalance
		if balanceEth == "" {
			balanceEth = "10000"
		}
		if balFloat, ok := new(big.Float).SetString(balanceEth); ok {
			balFloat.Mul(balFloat, new(big.Float).SetFloat64(1e18))
			balFloat.Int(balWei)
		}
	}

	labels := []string{"deployer"}
	for i := 1; i < len(addresses); i++ {
		labels = append(labels, fmt.Sprintf("user%d", i))
	}

	accounts := make([]chain.Account, 0, len(addresses))
	for i, addr := range addresses {
		label := ""
		if i < len(labels) {
			label = labels[i]
		}
		accounts = append(accounts, chain.Account{
			Address:    addr,
			PrivateKey: keyMap[addr],
			Balance:    new(big.Int).Set(balWei),
			Label:      label,
		})
	}
	return accounts, nil
}
