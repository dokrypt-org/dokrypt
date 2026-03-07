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
	gethImage       = "ethereum/client-go:latest"
	gethDefaultPort = 8545
)

type GethChain struct {
	name        string
	cfg         config.ChainConfig
	runtime     container.Runtime
	projectName string
	containerID string
	rpcClient   *RPCClient
	accounts    []chain.Account
	forkInfo    *chain.ForkInfo
	hostPort    int
	wsHostPort  int
}

func NewGethChain(name string, cfg config.ChainConfig, runtime container.Runtime, projectName string) (*GethChain, error) {
	return &GethChain{
		name:        name,
		cfg:         cfg,
		runtime:     runtime,
		projectName: projectName,
		hostPort:    8545,
	}, nil
}

func (g *GethChain) Start(ctx context.Context) error {
	slog.Info("starting geth chain", "chain", g.name)

	if err := g.runtime.PullImage(ctx, gethImage); err != nil {
		slog.Warn("failed to pull geth image", "error", err)
	}

	containerName := fmt.Sprintf("dokrypt-%s-%s", g.projectName, g.name)
	_ = g.runtime.StopContainer(ctx, containerName, 5*time.Second)
	_ = g.runtime.RemoveContainer(ctx, containerName, true)

	cmd := []string{
		"--dev",
		"--http",
		"--http.addr", "0.0.0.0",
		"--http.port", "8545",
		"--http.corsdomain", "*",
		"--http.api", "eth,net,web3,debug,personal,miner,txpool",
		"--ws",
		"--ws.addr", "0.0.0.0",
		"--ws.port", "8546",
		"--ws.origins", "*",
	}

	if dur, err := config.ParseDuration(g.cfg.BlockTime); err == nil && dur > 0 {
		cmd = append(cmd, "--dev.period", fmt.Sprintf("%d", int(dur.Seconds())))
	}

	id, err := g.runtime.CreateContainer(ctx, &container.ContainerConfig{
		Name:    containerName,
		Image:   gethImage,
		Command: cmd,
		Ports:   map[int]int{gethDefaultPort: 0, 8546: 0},
		Labels: map[string]string{
			"dokrypt.project": g.projectName,
			"dokrypt.chain":   g.name,
			"dokrypt.engine":  "geth",
		},
	})
	if err != nil {
		return fmt.Errorf("failed to create geth container: %w", err)
	}
	g.containerID = id

	if err := g.runtime.StartContainer(ctx, id); err != nil {
		return fmt.Errorf("failed to start geth container: %w", err)
	}

	info, err := g.runtime.InspectContainer(ctx, id)
	if err != nil {
		return err
	}
	if hp, ok := info.Ports[gethDefaultPort]; ok {
		g.hostPort = hp
	}
	if wsHP, ok := info.Ports[8546]; ok {
		g.wsHostPort = wsHP
	}

	g.rpcClient = NewRPCClient(g.RPCURL())

	if err := g.waitForReady(ctx); err != nil {
		return fmt.Errorf("geth failed to become ready: %w", err)
	}

	accounts, err := g.generateAccounts(ctx)
	if err != nil {
		slog.Warn("failed to generate accounts", "error", err)
	} else {
		g.accounts = accounts
	}

	slog.Info("geth chain started", "chain", g.name, "rpc", g.RPCURL())
	return nil
}

func (g *GethChain) Stop(ctx context.Context) error {
	if g.containerID == "" {
		return nil
	}
	if err := g.runtime.StopContainer(ctx, g.containerID, 10*time.Second); err != nil {
		slog.Warn("failed to stop geth container", "error", err)
	}
	if err := g.runtime.RemoveContainer(ctx, g.containerID, true); err != nil {
		return err
	}
	g.containerID = ""
	return nil
}

func (g *GethChain) IsRunning(ctx context.Context) bool {
	if g.containerID == "" {
		return false
	}
	info, err := g.runtime.InspectContainer(ctx, g.containerID)
	if err != nil {
		return false
	}
	return info.State == "running"
}

func (g *GethChain) Health(ctx context.Context) error {
	if g.rpcClient == nil {
		return fmt.Errorf("chain not started")
	}
	_, err := g.rpcClient.Call(ctx, "eth_blockNumber")
	return err
}

func (g *GethChain) Name() string                  { return g.name }
func (g *GethChain) ChainID() uint64               { return g.cfg.ChainID }
func (g *GethChain) RPCURL() string                 { return fmt.Sprintf("http://localhost:%d", g.hostPort) }
func (g *GethChain) WSURL() string                  { return fmt.Sprintf("ws://localhost:%d", g.wsHostPort) }
func (g *GethChain) Engine() string                 { return "geth" }
func (g *GethChain) ContainerID() string            { return g.containerID }
func (g *GethChain) Accounts() []chain.Account      { return g.accounts }
func (g *GethChain) ForkInfo() *chain.ForkInfo       { return g.forkInfo }

func (g *GethChain) FundAccount(ctx context.Context, address string, amountWei *big.Int) error {
	return fmt.Errorf("direct balance setting not supported in geth dev mode; use faucet or dev account transfer")
}

func (g *GethChain) ImpersonateAccount(_ context.Context, _ string) error {
	return fmt.Errorf("account impersonation not supported in geth dev mode")
}

func (g *GethChain) GenerateAccounts(ctx context.Context, count int) ([]chain.Account, error) {
	return g.generateAccounts(ctx)
}

func (g *GethChain) MineBlocks(ctx context.Context, count uint64) error {
	for range count {
		if _, err := g.rpcClient.Call(ctx, "evm_mine"); err != nil {
			return err
		}
	}
	return nil
}

func (g *GethChain) SetBlockTime(_ context.Context, _ uint64) error {
	return fmt.Errorf("dynamic block time change not supported in geth dev mode")
}

func (g *GethChain) SetGasPrice(_ context.Context, _ uint64) error {
	return fmt.Errorf("gas price setting not supported in geth dev mode")
}

func (g *GethChain) TimeTravel(ctx context.Context, seconds int64) error {
	hexSeconds := fmt.Sprintf("0x%x", seconds)
	_, err := g.rpcClient.Call(ctx, "evm_increaseTime", hexSeconds)
	if err != nil {
		return err
	}
	return g.MineBlocks(ctx, 1)
}

func (g *GethChain) SetBalance(_ context.Context, _ string, _ *big.Int) error {
	return fmt.Errorf("direct balance setting not supported in geth dev mode")
}

func (g *GethChain) SetStorageAt(_ context.Context, _, _, _ string) error {
	return fmt.Errorf("storage manipulation not supported in geth dev mode")
}

func (g *GethChain) TakeSnapshot(_ context.Context) (string, error) {
	return "", fmt.Errorf("evm_snapshot not supported in geth dev mode; use anvil or hardhat instead")
}

func (g *GethChain) RevertSnapshot(_ context.Context, _ string) error {
	return fmt.Errorf("evm_revert not supported in geth dev mode; use anvil or hardhat instead")
}

func (g *GethChain) ExportState(_ context.Context, _ string) error {
	return fmt.Errorf("state export not supported in geth dev mode")
}

func (g *GethChain) ImportState(_ context.Context, _ string) error {
	return fmt.Errorf("state import not supported in geth dev mode")
}

func (g *GethChain) Fork(_ context.Context, _ chain.ForkOptions) error {
	return fmt.Errorf("chain forking not supported in geth dev mode; use anvil instead")
}

func (g *GethChain) RPC(ctx context.Context, method string, params ...any) (json.RawMessage, error) {
	return g.rpcClient.Call(ctx, method, params...)
}

func (g *GethChain) Logs(ctx context.Context, follow bool) (io.ReadCloser, error) {
	if g.containerID == "" {
		return nil, fmt.Errorf("chain not started")
	}
	return g.runtime.ContainerLogs(ctx, g.containerID, container.LogOptions{Follow: follow, Stdout: true, Stderr: true})
}

func (g *GethChain) waitForReady(ctx context.Context) error {
	return common.Retry(ctx, common.RetryConfig{
		MaxAttempts: 30, InitialDelay: 500 * time.Millisecond, MaxDelay: 3 * time.Second, Multiplier: 1.5, Jitter: 0.1,
	}, func(ctx context.Context) error { return g.Health(ctx) })
}

func (g *GethChain) generateAccounts(_ context.Context) ([]chain.Account, error) {
	count := g.cfg.Accounts
	if count <= 0 {
		count = 10
	}
	infos, err := common.GenerateAccounts(common.DefaultSeed, count)
	if err != nil {
		return nil, err
	}
	accounts := make([]chain.Account, 0, len(infos))
	for _, info := range infos {
		balWei := new(big.Int)
		if balFloat, ok := new(big.Float).SetString(g.cfg.AccountBalance); ok {
			balFloat.Mul(balFloat, new(big.Float).SetFloat64(1e18))
			balFloat.Int(balWei)
		}
		accounts = append(accounts, chain.Account{Address: info.Address, PrivateKey: info.PrivateKey, Balance: balWei, Label: info.Label})
	}
	return accounts, nil
}
