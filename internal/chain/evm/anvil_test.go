package evm

import (
	"context"
	"encoding/json"
	"fmt"
	"math/big"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"

	"github.com/dokrypt/dokrypt/internal/chain"
	"github.com/dokrypt/dokrypt/internal/config"
	"github.com/dokrypt/dokrypt/internal/rpc"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewAnvilChain(t *testing.T) {
	cfg := config.ChainConfig{
		Engine:  "anvil",
		ChainID: 31337,
	}

	ac, err := NewAnvilChain("test-anvil", cfg, nil, "my-project")
	require.NoError(t, err)
	require.NotNil(t, ac)

	assert.Equal(t, "test-anvil", ac.name)
	assert.Equal(t, uint64(31337), ac.cfg.ChainID)
	assert.Equal(t, "my-project", ac.projectName)
	assert.Equal(t, 8545, ac.hostPort)
	assert.Nil(t, ac.runtime)
	assert.Empty(t, ac.containerID)
}

func TestAnvilChain_Name(t *testing.T) {
	ac, err := NewAnvilChain("my-chain", config.ChainConfig{ChainID: 1}, nil, "proj")
	require.NoError(t, err)
	assert.Equal(t, "my-chain", ac.Name())
}

func TestAnvilChain_ChainID(t *testing.T) {
	ac, err := NewAnvilChain("test", config.ChainConfig{ChainID: 42161}, nil, "proj")
	require.NoError(t, err)
	assert.Equal(t, uint64(42161), ac.ChainID())
}

func TestAnvilChain_Engine(t *testing.T) {
	ac, err := NewAnvilChain("test", config.ChainConfig{}, nil, "proj")
	require.NoError(t, err)
	assert.Equal(t, "anvil", ac.Engine())
}

func TestAnvilChain_RPCURL(t *testing.T) {
	ac, err := NewAnvilChain("test", config.ChainConfig{}, nil, "proj")
	require.NoError(t, err)
	assert.Equal(t, "http://localhost:8545", ac.RPCURL())

	ac.hostPort = 9999
	assert.Equal(t, "http://localhost:9999", ac.RPCURL())
}

func TestAnvilChain_WSURL(t *testing.T) {
	ac, err := NewAnvilChain("test", config.ChainConfig{}, nil, "proj")
	require.NoError(t, err)
	assert.Equal(t, "ws://localhost:8545", ac.WSURL())

	ac.hostPort = 12345
	assert.Equal(t, "ws://localhost:12345", ac.WSURL())
}

func TestAnvilChain_ContainerID(t *testing.T) {
	ac, err := NewAnvilChain("test", config.ChainConfig{}, nil, "proj")
	require.NoError(t, err)
	assert.Empty(t, ac.ContainerID())

	ac.containerID = "abc123"
	assert.Equal(t, "abc123", ac.ContainerID())
}

func TestAnvilChain_Accounts_Empty(t *testing.T) {
	ac, err := NewAnvilChain("test", config.ChainConfig{}, nil, "proj")
	require.NoError(t, err)
	assert.Nil(t, ac.Accounts())
}

func TestAnvilChain_Accounts_WithData(t *testing.T) {
	ac, err := NewAnvilChain("test", config.ChainConfig{}, nil, "proj")
	require.NoError(t, err)

	ac.accounts = []chain.Account{
		{Address: "0xabc", PrivateKey: "0x123", Label: "deployer"},
	}
	accounts := ac.Accounts()
	assert.Len(t, accounts, 1)
	assert.Equal(t, "0xabc", accounts[0].Address)
}

func TestAnvilChain_ForkInfo_Nil(t *testing.T) {
	ac, err := NewAnvilChain("test", config.ChainConfig{}, nil, "proj")
	require.NoError(t, err)
	assert.Nil(t, ac.ForkInfo())
}

func TestAnvilChain_ForkInfo_Set(t *testing.T) {
	ac, err := NewAnvilChain("test", config.ChainConfig{}, nil, "proj")
	require.NoError(t, err)

	ac.forkInfo = &chain.ForkInfo{
		Network:     "mainnet",
		BlockNumber: 18000000,
		RPCURL:      "https://eth.llamarpc.com",
	}
	info := ac.ForkInfo()
	require.NotNil(t, info)
	assert.Equal(t, "mainnet", info.Network)
	assert.Equal(t, uint64(18000000), info.BlockNumber)
}

func TestAnvilChain_Health_NotStarted(t *testing.T) {
	ac, err := NewAnvilChain("test", config.ChainConfig{}, nil, "proj")
	require.NoError(t, err)

	ctx := context.Background()
	err = ac.Health(ctx)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "chain not started")
}

func TestAnvilChain_Health_WithRPCClient(t *testing.T) {
	handler := newRPCHandler()
	handler.handle("eth_blockNumber", func(params json.RawMessage) (any, *rpc.RPCError) {
		return "0x10", nil
	})
	srv := httptest.NewServer(handler)
	defer srv.Close()

	ac, err := NewAnvilChain("test", config.ChainConfig{}, nil, "proj")
	require.NoError(t, err)
	ac.rpcClient = NewRPCClient(srv.URL)

	ctx := context.Background()
	err = ac.Health(ctx)
	require.NoError(t, err)
}

func TestAnvilChain_IsRunning_NoContainer(t *testing.T) {
	ac, err := NewAnvilChain("test", config.ChainConfig{}, nil, "proj")
	require.NoError(t, err)

	ctx := context.Background()
	assert.False(t, ac.IsRunning(ctx))
}

func TestAnvilChain_Stop_NoContainer(t *testing.T) {
	ac, err := NewAnvilChain("test", config.ChainConfig{}, nil, "proj")
	require.NoError(t, err)

	ctx := context.Background()
	err = ac.Stop(ctx)
	require.NoError(t, err) // Should be nil when no container is running
}

func TestAnvilChain_Logs_NotStarted(t *testing.T) {
	ac, err := NewAnvilChain("test", config.ChainConfig{}, nil, "proj")
	require.NoError(t, err)

	ctx := context.Background()
	_, err = ac.Logs(ctx, false)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "chain not started")
}

func TestAnvilChain_FundAccount(t *testing.T) {
	handler := newRPCHandler()
	var receivedMethod string
	handler.handle("anvil_setBalance", func(params json.RawMessage) (any, *rpc.RPCError) {
		receivedMethod = "anvil_setBalance"
		return true, nil
	})
	srv := httptest.NewServer(handler)
	defer srv.Close()

	ac, err := NewAnvilChain("test", config.ChainConfig{}, nil, "proj")
	require.NoError(t, err)
	ac.rpcClient = NewRPCClient(srv.URL)

	ctx := context.Background()
	amount := big.NewInt(1000000000000000000) // 1 ETH
	err = ac.FundAccount(ctx, "0x1234", amount)
	require.NoError(t, err)
	assert.Equal(t, "anvil_setBalance", receivedMethod)
}

func TestAnvilChain_ImpersonateAccount(t *testing.T) {
	handler := newRPCHandler()
	handler.handle("anvil_impersonateAccount", func(params json.RawMessage) (any, *rpc.RPCError) {
		return true, nil
	})
	srv := httptest.NewServer(handler)
	defer srv.Close()

	ac, err := NewAnvilChain("test", config.ChainConfig{}, nil, "proj")
	require.NoError(t, err)
	ac.rpcClient = NewRPCClient(srv.URL)

	ctx := context.Background()
	err = ac.ImpersonateAccount(ctx, "0xdead")
	require.NoError(t, err)
}

func TestAnvilChain_MineBlocks(t *testing.T) {
	handler := newRPCHandler()
	handler.handle("anvil_mine", func(params json.RawMessage) (any, *rpc.RPCError) {
		return true, nil
	})
	srv := httptest.NewServer(handler)
	defer srv.Close()

	ac, err := NewAnvilChain("test", config.ChainConfig{}, nil, "proj")
	require.NoError(t, err)
	ac.rpcClient = NewRPCClient(srv.URL)

	ctx := context.Background()
	err = ac.MineBlocks(ctx, 10)
	require.NoError(t, err)
}

func TestAnvilChain_SetBlockTime(t *testing.T) {
	handler := newRPCHandler()
	handler.handle("evm_setIntervalMining", func(params json.RawMessage) (any, *rpc.RPCError) {
		return true, nil
	})
	srv := httptest.NewServer(handler)
	defer srv.Close()

	ac, err := NewAnvilChain("test", config.ChainConfig{}, nil, "proj")
	require.NoError(t, err)
	ac.rpcClient = NewRPCClient(srv.URL)

	ctx := context.Background()
	err = ac.SetBlockTime(ctx, 5)
	require.NoError(t, err)
}

func TestAnvilChain_SetGasPrice(t *testing.T) {
	handler := newRPCHandler()
	handler.handle("anvil_setNextBlockBaseFeePerGas", func(params json.RawMessage) (any, *rpc.RPCError) {
		return true, nil
	})
	srv := httptest.NewServer(handler)
	defer srv.Close()

	ac, err := NewAnvilChain("test", config.ChainConfig{}, nil, "proj")
	require.NoError(t, err)
	ac.rpcClient = NewRPCClient(srv.URL)

	ctx := context.Background()
	err = ac.SetGasPrice(ctx, 20) // 20 gwei
	require.NoError(t, err)
}

func TestAnvilChain_TimeTravel(t *testing.T) {
	handler := newRPCHandler()
	handler.handle("evm_increaseTime", func(params json.RawMessage) (any, *rpc.RPCError) {
		return true, nil
	})
	handler.handle("anvil_mine", func(params json.RawMessage) (any, *rpc.RPCError) {
		return true, nil
	})
	srv := httptest.NewServer(handler)
	defer srv.Close()

	ac, err := NewAnvilChain("test", config.ChainConfig{}, nil, "proj")
	require.NoError(t, err)
	ac.rpcClient = NewRPCClient(srv.URL)

	ctx := context.Background()
	err = ac.TimeTravel(ctx, 3600) // 1 hour
	require.NoError(t, err)
}

func TestAnvilChain_SetBalance(t *testing.T) {
	handler := newRPCHandler()
	handler.handle("anvil_setBalance", func(params json.RawMessage) (any, *rpc.RPCError) {
		return true, nil
	})
	srv := httptest.NewServer(handler)
	defer srv.Close()

	ac, err := NewAnvilChain("test", config.ChainConfig{}, nil, "proj")
	require.NoError(t, err)
	ac.rpcClient = NewRPCClient(srv.URL)

	ctx := context.Background()
	amount := big.NewInt(5000000000000000000) // 5 ETH
	err = ac.SetBalance(ctx, "0x1234", amount)
	require.NoError(t, err)
}

func TestAnvilChain_SetStorageAt(t *testing.T) {
	handler := newRPCHandler()
	handler.handle("anvil_setStorageAt", func(params json.RawMessage) (any, *rpc.RPCError) {
		return true, nil
	})
	srv := httptest.NewServer(handler)
	defer srv.Close()

	ac, err := NewAnvilChain("test", config.ChainConfig{}, nil, "proj")
	require.NoError(t, err)
	ac.rpcClient = NewRPCClient(srv.URL)

	ctx := context.Background()
	err = ac.SetStorageAt(ctx, "0xcontract", "0x0", "0x1")
	require.NoError(t, err)
}

func TestAnvilChain_TakeSnapshot(t *testing.T) {
	handler := newRPCHandler()
	handler.handle("evm_snapshot", func(params json.RawMessage) (any, *rpc.RPCError) {
		return "0x1", nil
	})
	srv := httptest.NewServer(handler)
	defer srv.Close()

	ac, err := NewAnvilChain("test", config.ChainConfig{}, nil, "proj")
	require.NoError(t, err)
	ac.rpcClient = NewRPCClient(srv.URL)

	ctx := context.Background()
	id, err := ac.TakeSnapshot(ctx)
	require.NoError(t, err)
	assert.Equal(t, "0x1", id)
}

func TestAnvilChain_RevertSnapshot(t *testing.T) {
	handler := newRPCHandler()
	handler.handle("evm_revert", func(params json.RawMessage) (any, *rpc.RPCError) {
		return true, nil
	})
	srv := httptest.NewServer(handler)
	defer srv.Close()

	ac, err := NewAnvilChain("test", config.ChainConfig{}, nil, "proj")
	require.NoError(t, err)
	ac.rpcClient = NewRPCClient(srv.URL)

	ctx := context.Background()
	err = ac.RevertSnapshot(ctx, "0x1")
	require.NoError(t, err)
}

func TestAnvilChain_ExportState(t *testing.T) {
	handler := newRPCHandler()
	handler.handle("anvil_dumpState", func(params json.RawMessage) (any, *rpc.RPCError) {
		return "base64encodedstate", nil
	})
	srv := httptest.NewServer(handler)
	defer srv.Close()

	ac, err := NewAnvilChain("test", config.ChainConfig{}, nil, "proj")
	require.NoError(t, err)
	ac.rpcClient = NewRPCClient(srv.URL)

	dir := t.TempDir()
	path := filepath.Join(dir, "state.json")

	ctx := context.Background()
	err = ac.ExportState(ctx, path)
	require.NoError(t, err)

	data, err := os.ReadFile(path)
	require.NoError(t, err)
	assert.NotEmpty(t, data)
}

func TestAnvilChain_ImportState(t *testing.T) {
	var receivedState string
	handler := newRPCHandler()
	handler.handle("anvil_loadState", func(params json.RawMessage) (any, *rpc.RPCError) {
		var args []string
		json.Unmarshal(params, &args)
		if len(args) > 0 {
			receivedState = args[0]
		}
		return true, nil
	})
	srv := httptest.NewServer(handler)
	defer srv.Close()

	ac, err := NewAnvilChain("test", config.ChainConfig{}, nil, "proj")
	require.NoError(t, err)
	ac.rpcClient = NewRPCClient(srv.URL)

	dir := t.TempDir()
	path := filepath.Join(dir, "state.json")
	stateData := `"base64statedata"`
	err = os.WriteFile(path, []byte(stateData), 0644)
	require.NoError(t, err)

	ctx := context.Background()
	err = ac.ImportState(ctx, path)
	require.NoError(t, err)
	assert.Equal(t, "base64statedata", receivedState)
}

func TestAnvilChain_ImportState_NonExistentFile(t *testing.T) {
	ac, err := NewAnvilChain("test", config.ChainConfig{}, nil, "proj")
	require.NoError(t, err)
	ac.rpcClient = NewRPCClient("http://localhost:8545")

	ctx := context.Background()
	err = ac.ImportState(ctx, "/nonexistent/state.json")
	require.Error(t, err)
}

func TestAnvilChain_Fork(t *testing.T) {
	handler := newRPCHandler()
	handler.handle("anvil_reset", func(params json.RawMessage) (any, *rpc.RPCError) {
		return true, nil
	})
	srv := httptest.NewServer(handler)
	defer srv.Close()

	ac, err := NewAnvilChain("test", config.ChainConfig{}, nil, "proj")
	require.NoError(t, err)
	ac.rpcClient = NewRPCClient(srv.URL)

	ctx := context.Background()
	opts := chain.ForkOptions{
		Network:     "mainnet",
		BlockNumber: 18000000,
	}
	err = ac.Fork(ctx, opts)
	require.NoError(t, err)

	info := ac.ForkInfo()
	require.NotNil(t, info)
	assert.Equal(t, "mainnet", info.Network)
	assert.Equal(t, uint64(18000000), info.BlockNumber)
	assert.Equal(t, "https://eth.llamarpc.com", info.RPCURL)
}

func TestAnvilChain_Fork_WithCustomURL(t *testing.T) {
	handler := newRPCHandler()
	handler.handle("anvil_reset", func(params json.RawMessage) (any, *rpc.RPCError) {
		return true, nil
	})
	srv := httptest.NewServer(handler)
	defer srv.Close()

	ac, err := NewAnvilChain("test", config.ChainConfig{}, nil, "proj")
	require.NoError(t, err)
	ac.rpcClient = NewRPCClient(srv.URL)

	ctx := context.Background()
	customURL := "https://my-custom-rpc.example.com"
	opts := chain.ForkOptions{
		Network: customURL,
	}
	err = ac.Fork(ctx, opts)
	require.NoError(t, err)

	info := ac.ForkInfo()
	require.NotNil(t, info)
	assert.Equal(t, customURL, info.RPCURL)
}

func TestAnvilChain_RPC(t *testing.T) {
	handler := newRPCHandler()
	handler.handle("custom_method", func(params json.RawMessage) (any, *rpc.RPCError) {
		return "custom_result", nil
	})
	srv := httptest.NewServer(handler)
	defer srv.Close()

	ac, err := NewAnvilChain("test", config.ChainConfig{}, nil, "proj")
	require.NoError(t, err)
	ac.rpcClient = NewRPCClient(srv.URL)

	ctx := context.Background()
	result, err := ac.RPC(ctx, "custom_method", "param1")
	require.NoError(t, err)

	var resultStr string
	err = json.Unmarshal(result, &resultStr)
	require.NoError(t, err)
	assert.Equal(t, "custom_result", resultStr)
}

func TestAnvilChain_BuildArgs_Basic(t *testing.T) {
	ac, err := NewAnvilChain("test", config.ChainConfig{
		ChainID: 31337,
	}, nil, "proj")
	require.NoError(t, err)

	args := ac.buildArgs()
	assert.Contains(t, args, "anvil")
	assert.Contains(t, args, "--host")
	assert.Contains(t, args, "0.0.0.0")
	assert.Contains(t, args, "--port")
	assert.Contains(t, args, "8545")
	assert.Contains(t, args, "--chain-id")
	assert.Contains(t, args, "31337")
}

func TestAnvilChain_BuildArgs_WithAccounts(t *testing.T) {
	ac, err := NewAnvilChain("test", config.ChainConfig{
		ChainID:  31337,
		Accounts: 20,
	}, nil, "proj")
	require.NoError(t, err)

	args := ac.buildArgs()
	assert.Contains(t, args, "--accounts")
	assert.Contains(t, args, "20")
}

func TestAnvilChain_BuildArgs_WithAccountBalance(t *testing.T) {
	ac, err := NewAnvilChain("test", config.ChainConfig{
		ChainID:        31337,
		AccountBalance: "10000",
	}, nil, "proj")
	require.NoError(t, err)

	args := ac.buildArgs()
	assert.Contains(t, args, "--balance")
	assert.Contains(t, args, "10000")
}

func TestAnvilChain_BuildArgs_WithBalance_WeiConversion(t *testing.T) {
	ac, err := NewAnvilChain("test", config.ChainConfig{
		ChainID: 31337,
		Balance: "10000000000000000000000", // 10000 ETH in wei
	}, nil, "proj")
	require.NoError(t, err)

	args := ac.buildArgs()
	assert.Contains(t, args, "--balance")
	assert.Contains(t, args, "10000")
}

func TestAnvilChain_BuildArgs_WithGasLimit(t *testing.T) {
	ac, err := NewAnvilChain("test", config.ChainConfig{
		ChainID:  31337,
		GasLimit: 30000000,
	}, nil, "proj")
	require.NoError(t, err)

	args := ac.buildArgs()
	assert.Contains(t, args, "--gas-limit")
	assert.Contains(t, args, "30000000")
}

func TestAnvilChain_BuildArgs_WithBaseFee(t *testing.T) {
	ac, err := NewAnvilChain("test", config.ChainConfig{
		ChainID: 31337,
		BaseFee: 1000000000,
	}, nil, "proj")
	require.NoError(t, err)

	args := ac.buildArgs()
	assert.Contains(t, args, "--base-fee")
	assert.Contains(t, args, "1000000000")
}

func TestAnvilChain_BuildArgs_WithHardfork(t *testing.T) {
	ac, err := NewAnvilChain("test", config.ChainConfig{
		ChainID:  31337,
		Hardfork: "shanghai",
	}, nil, "proj")
	require.NoError(t, err)

	args := ac.buildArgs()
	assert.Contains(t, args, "--hardfork")
	assert.Contains(t, args, "shanghai")
}

func TestAnvilChain_BuildArgs_WithCodeSizeLimit(t *testing.T) {
	ac, err := NewAnvilChain("test", config.ChainConfig{
		ChainID:       31337,
		CodeSizeLimit: 100000,
	}, nil, "proj")
	require.NoError(t, err)

	args := ac.buildArgs()
	assert.Contains(t, args, "--code-size-limit")
	assert.Contains(t, args, "100000")
}

func TestAnvilChain_BuildArgs_WithAutoImpersonate(t *testing.T) {
	ac, err := NewAnvilChain("test", config.ChainConfig{
		ChainID:         31337,
		AutoImpersonate: true,
	}, nil, "proj")
	require.NoError(t, err)

	args := ac.buildArgs()
	assert.Contains(t, args, "--auto-impersonate")
}

func TestAnvilChain_BuildArgs_WithoutAutoImpersonate(t *testing.T) {
	ac, err := NewAnvilChain("test", config.ChainConfig{
		ChainID:         31337,
		AutoImpersonate: false,
	}, nil, "proj")
	require.NoError(t, err)

	args := ac.buildArgs()
	assert.NotContains(t, args, "--auto-impersonate")
}

func TestAnvilChain_BuildArgs_WithBlockTime(t *testing.T) {
	ac, err := NewAnvilChain("test", config.ChainConfig{
		ChainID:   31337,
		BlockTime: "5s",
	}, nil, "proj")
	require.NoError(t, err)

	args := ac.buildArgs()
	assert.Contains(t, args, "--block-time")
	assert.Contains(t, args, "5")
}

func TestAnvilChain_BuildArgs_WithForkConfig(t *testing.T) {
	ac, err := NewAnvilChain("test", config.ChainConfig{
		ChainID: 31337,
		Fork: &config.ForkConfig{
			Network:     "mainnet",
			BlockNumber: 18000000,
		},
	}, nil, "proj")
	require.NoError(t, err)

	args := ac.buildArgs()
	assert.Contains(t, args, "--fork-url")
	assert.Contains(t, args, "https://eth.llamarpc.com")
	assert.Contains(t, args, "--fork-block-number")
	assert.Contains(t, args, "18000000")

	info := ac.ForkInfo()
	require.NotNil(t, info)
	assert.Equal(t, "mainnet", info.Network)
}

func TestAnvilChain_BuildArgs_WithForkConfig_CustomRPCURL(t *testing.T) {
	ac, err := NewAnvilChain("test", config.ChainConfig{
		ChainID: 31337,
		Fork: &config.ForkConfig{
			RPCURL: "https://custom-rpc.example.com",
		},
	}, nil, "proj")
	require.NoError(t, err)

	args := ac.buildArgs()
	assert.Contains(t, args, "--fork-url")
	assert.Contains(t, args, "https://custom-rpc.example.com")
}

func TestAnvilChain_BuildArgs_WithForkConfig_NoBlockNumber(t *testing.T) {
	ac, err := NewAnvilChain("test", config.ChainConfig{
		ChainID: 31337,
		Fork: &config.ForkConfig{
			Network: "sepolia",
		},
	}, nil, "proj")
	require.NoError(t, err)

	args := ac.buildArgs()
	assert.Contains(t, args, "--fork-url")
	assert.NotContains(t, args, "--fork-block-number")
}

func TestAnvilChain_BuildArgs_AllOptions(t *testing.T) {
	ac, err := NewAnvilChain("test", config.ChainConfig{
		ChainID:         31337,
		Accounts:        20,
		AccountBalance:  "10000",
		GasLimit:        30000000,
		BaseFee:         1000000000,
		Hardfork:        "shanghai",
		CodeSizeLimit:   100000,
		AutoImpersonate: true,
		BlockTime:       "2s",
	}, nil, "proj")
	require.NoError(t, err)

	args := ac.buildArgs()

	expectedFlags := []string{
		"anvil", "--host", "--port", "--chain-id",
		"--accounts", "--balance", "--gas-limit", "--base-fee",
		"--hardfork", "--code-size-limit", "--auto-impersonate", "--block-time",
	}
	for _, flag := range expectedFlags {
		assert.Contains(t, args, flag, "expected flag %q not found", flag)
	}
}

func TestAnvilChain_ImplementsChainInterface(t *testing.T) {
	ac, err := NewAnvilChain("test", config.ChainConfig{ChainID: 31337}, nil, "proj")
	require.NoError(t, err)

	var _ chain.Chain = ac
}

func TestAnvilChain_ExportState_RPCError(t *testing.T) {
	handler := newRPCHandler()
	handler.handle("anvil_dumpState", func(params json.RawMessage) (any, *rpc.RPCError) {
		return nil, &rpc.RPCError{Code: -32000, Message: "dump failed"}
	})
	srv := httptest.NewServer(handler)
	defer srv.Close()

	ac, err := NewAnvilChain("test", config.ChainConfig{}, nil, "proj")
	require.NoError(t, err)
	ac.rpcClient = NewRPCClient(srv.URL)

	ctx := context.Background()
	err = ac.ExportState(ctx, "/tmp/state.json")
	require.Error(t, err)
}

func TestAnvilChain_TimeTravel_IncrementError(t *testing.T) {
	handler := newRPCHandler()
	handler.handle("evm_increaseTime", func(params json.RawMessage) (any, *rpc.RPCError) {
		return nil, &rpc.RPCError{Code: -32000, Message: "increase time failed"}
	})
	srv := httptest.NewServer(handler)
	defer srv.Close()

	ac, err := NewAnvilChain("test", config.ChainConfig{}, nil, "proj")
	require.NoError(t, err)
	ac.rpcClient = NewRPCClient(srv.URL)

	ctx := context.Background()
	err = ac.TimeTravel(ctx, 3600)
	require.Error(t, err)
}

func TestAnvilChain_Fork_RPCError(t *testing.T) {
	handler := newRPCHandler()
	handler.handle("anvil_reset", func(params json.RawMessage) (any, *rpc.RPCError) {
		return nil, &rpc.RPCError{Code: -32000, Message: "reset failed"}
	})
	srv := httptest.NewServer(handler)
	defer srv.Close()

	ac, err := NewAnvilChain("test", config.ChainConfig{}, nil, "proj")
	require.NoError(t, err)
	ac.rpcClient = NewRPCClient(srv.URL)

	ctx := context.Background()
	err = ac.Fork(ctx, chain.ForkOptions{Network: "mainnet"})
	require.Error(t, err)

	assert.Nil(t, ac.ForkInfo())
}

func TestAnvilChain_Fork_WithBlockNumberZero(t *testing.T) {
	var receivedParams json.RawMessage
	handler := newRPCHandler()
	handler.handle("anvil_reset", func(params json.RawMessage) (any, *rpc.RPCError) {
		receivedParams = params
		return true, nil
	})
	srv := httptest.NewServer(handler)
	defer srv.Close()

	ac, err := NewAnvilChain("test", config.ChainConfig{}, nil, "proj")
	require.NoError(t, err)
	ac.rpcClient = NewRPCClient(srv.URL)

	ctx := context.Background()
	opts := chain.ForkOptions{
		Network:     "mainnet",
		BlockNumber: 0, // latest
	}
	err = ac.Fork(ctx, opts)
	require.NoError(t, err)

	assert.NotNil(t, receivedParams)
	var paramArr []map[string]any
	err = json.Unmarshal(receivedParams, &paramArr)
	if err == nil && len(paramArr) > 0 {
		forking, ok := paramArr[0]["forking"].(map[string]any)
		if ok {
			_, hasBlockNumber := forking["blockNumber"]
			assert.False(t, hasBlockNumber, "blockNumber should not be present when 0")
		}
	}
}

func TestAnvilChain_ImportState_RawData(t *testing.T) {
	handler := newRPCHandler()
	handler.handle("anvil_loadState", func(params json.RawMessage) (any, *rpc.RPCError) {
		return true, nil
	})
	srv := httptest.NewServer(handler)
	defer srv.Close()

	ac, err := NewAnvilChain("test", config.ChainConfig{}, nil, "proj")
	require.NoError(t, err)
	ac.rpcClient = NewRPCClient(srv.URL)

	dir := t.TempDir()
	path := filepath.Join(dir, "state.raw")
	err = os.WriteFile(path, []byte("rawstatedata"), 0644)
	require.NoError(t, err)

	ctx := context.Background()
	err = ac.ImportState(ctx, path)
	require.NoError(t, err)
}

func TestAnvilChain_BuildArgs_BalancePrecedence(t *testing.T) {
	ac, err := NewAnvilChain("test", config.ChainConfig{
		ChainID:        31337,
		Balance:        "5000000000000000000000", // 5000 ETH in wei
		AccountBalance: "10000",                  // 10000 ETH
	}, nil, "proj")
	require.NoError(t, err)

	args := ac.buildArgs()
	assert.Contains(t, args, "--balance")
	assert.Contains(t, args, "5000")
}

func TestAnvilChain_BuildArgs_InvalidBlockTime(t *testing.T) {
	ac, err := NewAnvilChain("test", config.ChainConfig{
		ChainID:   31337,
		BlockTime: "invalid",
	}, nil, "proj")
	require.NoError(t, err)

	args := ac.buildArgs()
	assert.NotContains(t, args, "--block-time")
}

func TestAnvilChain_BuildArgs_EmptyBlockTime(t *testing.T) {
	ac, err := NewAnvilChain("test", config.ChainConfig{
		ChainID:   31337,
		BlockTime: "",
	}, nil, "proj")
	require.NoError(t, err)

	args := ac.buildArgs()
	assert.NotContains(t, args, "--block-time")
}

func TestAnvilChain_SetGasPrice_Conversion(t *testing.T) {
	var receivedParams json.RawMessage
	handler := newRPCHandler()
	handler.handle("anvil_setNextBlockBaseFeePerGas", func(params json.RawMessage) (any, *rpc.RPCError) {
		receivedParams = params
		return true, nil
	})
	srv := httptest.NewServer(handler)
	defer srv.Close()

	ac, err := NewAnvilChain("test", config.ChainConfig{}, nil, "proj")
	require.NoError(t, err)
	ac.rpcClient = NewRPCClient(srv.URL)

	ctx := context.Background()
	err = ac.SetGasPrice(ctx, 20)
	require.NoError(t, err)

	var params []string
	err = json.Unmarshal(receivedParams, &params)
	if err == nil && len(params) > 0 {
		expected := fmt.Sprintf("0x%x", uint64(20)*1e9)
		assert.Equal(t, expected, params[0])
	}
}
