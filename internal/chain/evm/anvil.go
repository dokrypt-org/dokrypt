package evm

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"math/big"
	"strconv"
	"time"

	"github.com/dokrypt/dokrypt/internal/chain"
	"github.com/dokrypt/dokrypt/internal/common"
	"github.com/dokrypt/dokrypt/internal/config"
	"github.com/dokrypt/dokrypt/internal/container"
)

const (
	anvilImage       = "ghcr.io/foundry-rs/foundry:latest"
	anvilDefaultPort = 8545
)

type AnvilChain struct {
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

func NewAnvilChain(name string, cfg config.ChainConfig, runtime container.Runtime, projectName string) (*AnvilChain, error) {
	return &AnvilChain{
		name:        name,
		cfg:         cfg,
		runtime:     runtime,
		projectName: projectName,
		hostPort:    8545,
	}, nil
}

func (a *AnvilChain) Start(ctx context.Context) error {
	slog.Info("starting anvil chain", "chain", a.name, "chain_id", a.cfg.ChainID)

	if err := a.runtime.PullImage(ctx, anvilImage); err != nil {
		slog.Warn("failed to pull anvil image, trying with cached", "error", err)
	}

	cmd := a.buildArgs()

	containerName := fmt.Sprintf("dokrypt-%s-%s", a.projectName, a.name)
	_ = a.runtime.StopContainer(ctx, containerName, 5*time.Second)
	_ = a.runtime.RemoveContainer(ctx, containerName, true)

	networkName := fmt.Sprintf("dokrypt-%s", a.projectName)
	id, err := a.runtime.CreateContainer(ctx, &container.ContainerConfig{
		Name:       containerName,
		Image:      anvilImage,
		Entrypoint: cmd[:1],  // ["anvil"]
		Command:    cmd[1:],  // ["--host", "0.0.0.0", ...]
		Ports:      map[int]int{anvilDefaultPort: 0}, // auto-assign host port
		Networks:   []string{networkName},
		NetworkAliases: map[string][]string{
			networkName: {a.name, containerName},
		},
		Labels: map[string]string{
			"dokrypt.project": a.projectName,
			"dokrypt.chain":   a.name,
			"dokrypt.engine":  "anvil",
		},
	})
	if err != nil {
		return fmt.Errorf("failed to create anvil container: %w", err)
	}
	a.containerID = id

	if err := a.runtime.StartContainer(ctx, id); err != nil {
		return fmt.Errorf("failed to start anvil container: %w", err)
	}

	info, err := a.runtime.InspectContainer(ctx, id)
	if err != nil {
		return fmt.Errorf("failed to inspect anvil container: %w", err)
	}
	if hp, ok := info.Ports[anvilDefaultPort]; ok {
		a.hostPort = hp
	}

	a.rpcClient = NewRPCClient(a.RPCURL())

	if err := a.waitForReady(ctx); err != nil {
		return fmt.Errorf("anvil failed to become ready: %w", err)
	}

	accounts, err := a.generateAccounts(ctx)
	if err != nil {
		slog.Warn("failed to generate accounts", "error", err)
	} else {
		a.accounts = accounts
	}

	slog.Info("anvil chain started", "chain", a.name, "rpc", a.RPCURL())
	return nil
}

func (a *AnvilChain) Stop(ctx context.Context) error {
	if a.containerID == "" {
		return nil
	}
	slog.Info("stopping anvil chain", "chain", a.name)

	if err := a.runtime.StopContainer(ctx, a.containerID, 10*time.Second); err != nil {
		slog.Warn("failed to stop anvil container", "error", err)
	}
	if err := a.runtime.RemoveContainer(ctx, a.containerID, true); err != nil {
		return fmt.Errorf("failed to remove anvil container: %w", err)
	}
	a.containerID = ""
	return nil
}

func (a *AnvilChain) IsRunning(ctx context.Context) bool {
	if a.containerID == "" {
		return false
	}
	info, err := a.runtime.InspectContainer(ctx, a.containerID)
	if err != nil {
		return false
	}
	return info.State == "running"
}

func (a *AnvilChain) Health(ctx context.Context) error {
	if a.rpcClient == nil {
		return fmt.Errorf("chain not started")
	}
	_, err := a.rpcClient.Call(ctx, "eth_blockNumber")
	return err
}

func (a *AnvilChain) Name() string { return a.name }

func (a *AnvilChain) ChainID() uint64 { return a.cfg.ChainID }

func (a *AnvilChain) RPCURL() string {
	return fmt.Sprintf("http://localhost:%d", a.hostPort)
}

func (a *AnvilChain) WSURL() string {
	return fmt.Sprintf("ws://localhost:%d", a.hostPort)
}

func (a *AnvilChain) Engine() string { return "anvil" }

func (a *AnvilChain) ContainerID() string { return a.containerID }

func (a *AnvilChain) Accounts() []chain.Account { return a.accounts }

func (a *AnvilChain) FundAccount(ctx context.Context, address string, amountWei *big.Int) error {
	hexAmount := fmt.Sprintf("0x%x", amountWei)
	_, err := a.rpcClient.Call(ctx, "anvil_setBalance", address, hexAmount)
	return err
}

func (a *AnvilChain) ImpersonateAccount(ctx context.Context, address string) error {
	_, err := a.rpcClient.Call(ctx, "anvil_impersonateAccount", address)
	return err
}

func (a *AnvilChain) GenerateAccounts(ctx context.Context, count int) ([]chain.Account, error) {
	return a.generateAccounts(ctx)
}

func (a *AnvilChain) MineBlocks(ctx context.Context, count uint64) error {
	hexCount := fmt.Sprintf("0x%x", count)
	_, err := a.rpcClient.Call(ctx, "anvil_mine", hexCount)
	return err
}

func (a *AnvilChain) SetBlockTime(ctx context.Context, seconds uint64) error {
	_, err := a.rpcClient.Call(ctx, "evm_setIntervalMining", seconds)
	return err
}

func (a *AnvilChain) SetGasPrice(ctx context.Context, gweiPrice uint64) error {
	weiPrice := gweiPrice * 1e9
	hexPrice := fmt.Sprintf("0x%x", weiPrice)
	_, err := a.rpcClient.Call(ctx, "anvil_setNextBlockBaseFeePerGas", hexPrice)
	return err
}

func (a *AnvilChain) TimeTravel(ctx context.Context, seconds int64) error {
	hexSeconds := fmt.Sprintf("0x%x", seconds)
	_, err := a.rpcClient.Call(ctx, "evm_increaseTime", hexSeconds)
	if err != nil {
		return err
	}
	return a.MineBlocks(ctx, 1)
}

func (a *AnvilChain) SetBalance(ctx context.Context, address string, amountWei *big.Int) error {
	hexAmount := fmt.Sprintf("0x%x", amountWei)
	_, err := a.rpcClient.Call(ctx, "anvil_setBalance", address, hexAmount)
	return err
}

func (a *AnvilChain) SetStorageAt(ctx context.Context, address, slot, value string) error {
	_, err := a.rpcClient.Call(ctx, "anvil_setStorageAt", address, slot, value)
	return err
}

func (a *AnvilChain) TakeSnapshot(ctx context.Context) (string, error) {
	var result string
	err := a.rpcClient.CallResult(ctx, &result, "evm_snapshot")
	return result, err
}

func (a *AnvilChain) RevertSnapshot(ctx context.Context, id string) error {
	_, err := a.rpcClient.Call(ctx, "evm_revert", id)
	return err
}

func (a *AnvilChain) ExportState(ctx context.Context, path string) error {
	result, err := a.rpcClient.Call(ctx, "anvil_dumpState")
	if err != nil {
		return err
	}
	return writeFile(path, result)
}

func (a *AnvilChain) ImportState(ctx context.Context, path string) error {
	data, err := readFile(path)
	if err != nil {
		return err
	}
	var stateStr string
	if jsonErr := json.Unmarshal(data, &stateStr); jsonErr != nil {
		stateStr = string(data)
	}
	_, err = a.rpcClient.Call(ctx, "anvil_loadState", stateStr)
	return err
}

func (a *AnvilChain) Fork(ctx context.Context, opts chain.ForkOptions) error {
	rpcURL := chain.ResolveNetworkRPC(opts.Network)
	params := map[string]any{
		"forking": map[string]any{
			"jsonRpcUrl": rpcURL,
		},
	}
	if opts.BlockNumber > 0 {
		params["forking"].(map[string]any)["blockNumber"] = fmt.Sprintf("0x%x", opts.BlockNumber)
	}
	_, err := a.rpcClient.Call(ctx, "anvil_reset", params)
	if err != nil {
		return err
	}
	a.forkInfo = &chain.ForkInfo{
		Network:     opts.Network,
		BlockNumber: opts.BlockNumber,
		RPCURL:      rpcURL,
	}
	return nil
}

func (a *AnvilChain) ForkInfo() *chain.ForkInfo { return a.forkInfo }

func (a *AnvilChain) RPC(ctx context.Context, method string, params ...any) (json.RawMessage, error) {
	return a.rpcClient.Call(ctx, method, params...)
}

func (a *AnvilChain) Logs(ctx context.Context, follow bool) (io.ReadCloser, error) {
	if a.containerID == "" {
		return nil, fmt.Errorf("chain not started")
	}
	return a.runtime.ContainerLogs(ctx, a.containerID, container.LogOptions{
		Follow: follow,
		Stdout: true,
		Stderr: true,
	})
}

func (a *AnvilChain) buildArgs() []string {
	args := []string{
		"anvil",
		"--host", "0.0.0.0",
		"--port", strconv.Itoa(anvilDefaultPort),
		"--chain-id", strconv.FormatUint(a.cfg.ChainID, 10),
	}

	if a.cfg.Accounts > 0 {
		args = append(args, "--accounts", strconv.Itoa(a.cfg.Accounts))
	}

	balanceETH := a.cfg.AccountBalance
	if a.cfg.Balance != "" {
		balWei := new(big.Int)
		if _, ok := balWei.SetString(a.cfg.Balance, 10); ok {
			weiPerEth := new(big.Int).Exp(big.NewInt(10), big.NewInt(18), nil)
			balEth := new(big.Int).Div(balWei, weiPerEth)
			balanceETH = balEth.String()
		}
	}
	if balanceETH != "" {
		args = append(args, "--balance", balanceETH)
	}

	if a.cfg.GasLimit > 0 {
		args = append(args, "--gas-limit", strconv.FormatUint(a.cfg.GasLimit, 10))
	}
	if a.cfg.BaseFee > 0 {
		args = append(args, "--base-fee", strconv.FormatUint(a.cfg.BaseFee, 10))
	}
	if a.cfg.Hardfork != "" {
		args = append(args, "--hardfork", a.cfg.Hardfork)
	}

	if a.cfg.CodeSizeLimit > 0 {
		args = append(args, "--code-size-limit", strconv.FormatUint(a.cfg.CodeSizeLimit, 10))
	}

	if a.cfg.AutoImpersonate {
		args = append(args, "--auto-impersonate")
	}

	if dur, err := config.ParseDuration(a.cfg.BlockTime); err == nil && dur > 0 {
		args = append(args, "--block-time", strconv.Itoa(int(dur.Seconds())))
	}

	if a.cfg.Fork != nil {
		forkURL := a.cfg.Fork.RPCURL
		if forkURL == "" {
			forkURL = chain.ResolveNetworkRPC(a.cfg.Fork.Network)
		}
		if forkURL != "" {
			args = append(args, "--fork-url", forkURL)
			if a.cfg.Fork.BlockNumber > 0 {
				args = append(args, "--fork-block-number", strconv.FormatUint(a.cfg.Fork.BlockNumber, 10))
			}
			a.forkInfo = &chain.ForkInfo{
				Network:     a.cfg.Fork.Network,
				BlockNumber: a.cfg.Fork.BlockNumber,
				RPCURL:      forkURL,
			}
		}
	}

	return args
}

func (a *AnvilChain) waitForReady(ctx context.Context) error {
	return common.Retry(ctx, common.RetryConfig{
		MaxAttempts:  30,
		InitialDelay: 200 * time.Millisecond,
		MaxDelay:     2 * time.Second,
		Multiplier:   1.5,
		Jitter:       0.1,
	}, func(ctx context.Context) error {
		return a.Health(ctx)
	})
}

func (a *AnvilChain) generateAccounts(ctx context.Context) ([]chain.Account, error) {
	var addresses []string
	if err := a.rpcClient.CallResult(ctx, &addresses, "eth_accounts"); err != nil {
		return nil, fmt.Errorf("failed to fetch accounts: %w", err)
	}

	keyInfos, _ := common.GenerateAccounts(common.DefaultSeed, len(addresses))
	keyMap := make(map[string]string) // lowercase address → private key
	for _, info := range keyInfos {
		keyMap[info.Address] = info.PrivateKey
	}

	balWei := new(big.Int)
	if a.cfg.Balance != "" {
		balWei.SetString(a.cfg.Balance, 10)
	} else {
		balanceEth := a.cfg.AccountBalance
		if balanceEth == "" {
			balanceEth = "10000"
		}
		balFloat, ok := new(big.Float).SetString(balanceEth)
		if ok {
			weiPerEth := new(big.Float).SetFloat64(1e18)
			balFloat.Mul(balFloat, weiPerEth)
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

func writeFile(path string, data json.RawMessage) error {
	return writeFileBytes(path, data)
}

func readFile(path string) ([]byte, error) {
	return readFileBytes(path)
}
