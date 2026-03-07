package evm

import (
	"context"
	"encoding/json"
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

func TestNewHardhatChain(t *testing.T) {
	cfg := config.ChainConfig{
		Engine:  "hardhat",
		ChainID: 31337,
	}

	hc, err := NewHardhatChain("test-hardhat", cfg, nil, "my-project")
	require.NoError(t, err)
	require.NotNil(t, hc)

	assert.Equal(t, "test-hardhat", hc.name)
	assert.Equal(t, uint64(31337), hc.cfg.ChainID)
	assert.Equal(t, "my-project", hc.projectName)
	assert.Equal(t, 8545, hc.hostPort)
	assert.Nil(t, hc.runtime)
	assert.Empty(t, hc.containerID)
}

func TestHardhatChain_Name(t *testing.T) {
	hc, err := NewHardhatChain("hardhat-node", config.ChainConfig{ChainID: 31337}, nil, "proj")
	require.NoError(t, err)
	assert.Equal(t, "hardhat-node", hc.Name())
}

func TestHardhatChain_ChainID(t *testing.T) {
	hc, err := NewHardhatChain("test", config.ChainConfig{ChainID: 99999}, nil, "proj")
	require.NoError(t, err)
	assert.Equal(t, uint64(99999), hc.ChainID())
}

func TestHardhatChain_Engine(t *testing.T) {
	hc, err := NewHardhatChain("test", config.ChainConfig{}, nil, "proj")
	require.NoError(t, err)
	assert.Equal(t, "hardhat", hc.Engine())
}

func TestHardhatChain_RPCURL(t *testing.T) {
	hc, err := NewHardhatChain("test", config.ChainConfig{}, nil, "proj")
	require.NoError(t, err)
	assert.Equal(t, "http://localhost:8545", hc.RPCURL())

	hc.hostPort = 6666
	assert.Equal(t, "http://localhost:6666", hc.RPCURL())
}

func TestHardhatChain_WSURL(t *testing.T) {
	hc, err := NewHardhatChain("test", config.ChainConfig{}, nil, "proj")
	require.NoError(t, err)
	assert.Equal(t, "ws://localhost:8545", hc.WSURL())

	hc.hostPort = 4444
	assert.Equal(t, "ws://localhost:4444", hc.WSURL())
}

func TestHardhatChain_ContainerID(t *testing.T) {
	hc, err := NewHardhatChain("test", config.ChainConfig{}, nil, "proj")
	require.NoError(t, err)
	assert.Empty(t, hc.ContainerID())

	hc.containerID = "hh-container-1"
	assert.Equal(t, "hh-container-1", hc.ContainerID())
}

func TestHardhatChain_Accounts_Empty(t *testing.T) {
	hc, err := NewHardhatChain("test", config.ChainConfig{}, nil, "proj")
	require.NoError(t, err)
	assert.Nil(t, hc.Accounts())
}

func TestHardhatChain_Accounts_WithData(t *testing.T) {
	hc, err := NewHardhatChain("test", config.ChainConfig{}, nil, "proj")
	require.NoError(t, err)

	hc.accounts = []chain.Account{
		{Address: "0x1111", Label: "deployer"},
		{Address: "0x2222", Label: "user1"},
	}
	accounts := hc.Accounts()
	assert.Len(t, accounts, 2)
}

func TestHardhatChain_ForkInfo_Nil(t *testing.T) {
	hc, err := NewHardhatChain("test", config.ChainConfig{}, nil, "proj")
	require.NoError(t, err)
	assert.Nil(t, hc.ForkInfo())
}

func TestHardhatChain_Health_NotStarted(t *testing.T) {
	hc, err := NewHardhatChain("test", config.ChainConfig{}, nil, "proj")
	require.NoError(t, err)

	ctx := context.Background()
	err = hc.Health(ctx)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "chain not started")
}

func TestHardhatChain_Health_WithRPCClient(t *testing.T) {
	handler := newRPCHandler()
	handler.handle("eth_blockNumber", func(params json.RawMessage) (any, *rpc.RPCError) {
		return "0x0", nil
	})
	srv := httptest.NewServer(handler)
	defer srv.Close()

	hc, err := NewHardhatChain("test", config.ChainConfig{}, nil, "proj")
	require.NoError(t, err)
	hc.rpcClient = NewRPCClient(srv.URL)

	ctx := context.Background()
	err = hc.Health(ctx)
	require.NoError(t, err)
}

func TestHardhatChain_IsRunning_NoContainer(t *testing.T) {
	hc, err := NewHardhatChain("test", config.ChainConfig{}, nil, "proj")
	require.NoError(t, err)
	assert.False(t, hc.IsRunning(context.Background()))
}

func TestHardhatChain_Stop_NoContainer(t *testing.T) {
	hc, err := NewHardhatChain("test", config.ChainConfig{}, nil, "proj")
	require.NoError(t, err)
	err = hc.Stop(context.Background())
	require.NoError(t, err)
}

func TestHardhatChain_Logs_NotStarted(t *testing.T) {
	hc, err := NewHardhatChain("test", config.ChainConfig{}, nil, "proj")
	require.NoError(t, err)

	_, err = hc.Logs(context.Background(), false)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "chain not started")
}

func TestHardhatChain_FundAccount(t *testing.T) {
	handler := newRPCHandler()
	handler.handle("hardhat_setBalance", func(params json.RawMessage) (any, *rpc.RPCError) {
		return true, nil
	})
	srv := httptest.NewServer(handler)
	defer srv.Close()

	hc, err := NewHardhatChain("test", config.ChainConfig{}, nil, "proj")
	require.NoError(t, err)
	hc.rpcClient = NewRPCClient(srv.URL)

	ctx := context.Background()
	amount := big.NewInt(2000000000000000000) // 2 ETH
	err = hc.FundAccount(ctx, "0xaddr", amount)
	require.NoError(t, err)
}

func TestHardhatChain_ImpersonateAccount(t *testing.T) {
	handler := newRPCHandler()
	handler.handle("hardhat_impersonateAccount", func(params json.RawMessage) (any, *rpc.RPCError) {
		return true, nil
	})
	srv := httptest.NewServer(handler)
	defer srv.Close()

	hc, err := NewHardhatChain("test", config.ChainConfig{}, nil, "proj")
	require.NoError(t, err)
	hc.rpcClient = NewRPCClient(srv.URL)

	ctx := context.Background()
	err = hc.ImpersonateAccount(ctx, "0xdeadbeef")
	require.NoError(t, err)
}

func TestHardhatChain_MineBlocks(t *testing.T) {
	handler := newRPCHandler()
	handler.handle("hardhat_mine", func(params json.RawMessage) (any, *rpc.RPCError) {
		return true, nil
	})
	srv := httptest.NewServer(handler)
	defer srv.Close()

	hc, err := NewHardhatChain("test", config.ChainConfig{}, nil, "proj")
	require.NoError(t, err)
	hc.rpcClient = NewRPCClient(srv.URL)

	ctx := context.Background()
	err = hc.MineBlocks(ctx, 100)
	require.NoError(t, err)
}

func TestHardhatChain_SetBlockTime(t *testing.T) {
	handler := newRPCHandler()
	handler.handle("hardhat_setIntervalMining", func(params json.RawMessage) (any, *rpc.RPCError) {
		return true, nil
	})
	srv := httptest.NewServer(handler)
	defer srv.Close()

	hc, err := NewHardhatChain("test", config.ChainConfig{}, nil, "proj")
	require.NoError(t, err)
	hc.rpcClient = NewRPCClient(srv.URL)

	ctx := context.Background()
	err = hc.SetBlockTime(ctx, 10)
	require.NoError(t, err)
}

func TestHardhatChain_SetGasPrice(t *testing.T) {
	handler := newRPCHandler()
	handler.handle("hardhat_setNextBlockBaseFeePerGas", func(params json.RawMessage) (any, *rpc.RPCError) {
		return true, nil
	})
	srv := httptest.NewServer(handler)
	defer srv.Close()

	hc, err := NewHardhatChain("test", config.ChainConfig{}, nil, "proj")
	require.NoError(t, err)
	hc.rpcClient = NewRPCClient(srv.URL)

	ctx := context.Background()
	err = hc.SetGasPrice(ctx, 50) // 50 gwei
	require.NoError(t, err)
}

func TestHardhatChain_TimeTravel(t *testing.T) {
	handler := newRPCHandler()
	handler.handle("evm_increaseTime", func(params json.RawMessage) (any, *rpc.RPCError) {
		return true, nil
	})
	handler.handle("hardhat_mine", func(params json.RawMessage) (any, *rpc.RPCError) {
		return true, nil
	})
	srv := httptest.NewServer(handler)
	defer srv.Close()

	hc, err := NewHardhatChain("test", config.ChainConfig{}, nil, "proj")
	require.NoError(t, err)
	hc.rpcClient = NewRPCClient(srv.URL)

	ctx := context.Background()
	err = hc.TimeTravel(ctx, 86400) // 1 day
	require.NoError(t, err)
}

func TestHardhatChain_TimeTravel_IncreaseTimeError(t *testing.T) {
	handler := newRPCHandler()
	handler.handle("evm_increaseTime", func(params json.RawMessage) (any, *rpc.RPCError) {
		return nil, &rpc.RPCError{Code: -32000, Message: "error"}
	})
	srv := httptest.NewServer(handler)
	defer srv.Close()

	hc, err := NewHardhatChain("test", config.ChainConfig{}, nil, "proj")
	require.NoError(t, err)
	hc.rpcClient = NewRPCClient(srv.URL)

	ctx := context.Background()
	err = hc.TimeTravel(ctx, 3600)
	require.Error(t, err)
}

func TestHardhatChain_SetBalance(t *testing.T) {
	handler := newRPCHandler()
	handler.handle("hardhat_setBalance", func(params json.RawMessage) (any, *rpc.RPCError) {
		return true, nil
	})
	srv := httptest.NewServer(handler)
	defer srv.Close()

	hc, err := NewHardhatChain("test", config.ChainConfig{}, nil, "proj")
	require.NoError(t, err)
	hc.rpcClient = NewRPCClient(srv.URL)

	ctx := context.Background()
	amount := big.NewInt(3000000000000000000) // 3 ETH
	err = hc.SetBalance(ctx, "0x1234", amount)
	require.NoError(t, err)
}

func TestHardhatChain_SetStorageAt(t *testing.T) {
	handler := newRPCHandler()
	handler.handle("hardhat_setStorageAt", func(params json.RawMessage) (any, *rpc.RPCError) {
		return true, nil
	})
	srv := httptest.NewServer(handler)
	defer srv.Close()

	hc, err := NewHardhatChain("test", config.ChainConfig{}, nil, "proj")
	require.NoError(t, err)
	hc.rpcClient = NewRPCClient(srv.URL)

	ctx := context.Background()
	err = hc.SetStorageAt(ctx, "0xcontract", "0x0", "0x1")
	require.NoError(t, err)
}

func TestHardhatChain_TakeSnapshot(t *testing.T) {
	handler := newRPCHandler()
	handler.handle("evm_snapshot", func(params json.RawMessage) (any, *rpc.RPCError) {
		return "0x2", nil
	})
	srv := httptest.NewServer(handler)
	defer srv.Close()

	hc, err := NewHardhatChain("test", config.ChainConfig{}, nil, "proj")
	require.NoError(t, err)
	hc.rpcClient = NewRPCClient(srv.URL)

	ctx := context.Background()
	id, err := hc.TakeSnapshot(ctx)
	require.NoError(t, err)
	assert.Equal(t, "0x2", id)
}

func TestHardhatChain_RevertSnapshot(t *testing.T) {
	handler := newRPCHandler()
	handler.handle("evm_revert", func(params json.RawMessage) (any, *rpc.RPCError) {
		return true, nil
	})
	srv := httptest.NewServer(handler)
	defer srv.Close()

	hc, err := NewHardhatChain("test", config.ChainConfig{}, nil, "proj")
	require.NoError(t, err)
	hc.rpcClient = NewRPCClient(srv.URL)

	ctx := context.Background()
	err = hc.RevertSnapshot(ctx, "0x2")
	require.NoError(t, err)
}

func TestHardhatChain_ExportState(t *testing.T) {
	handler := newRPCHandler()
	handler.handle("hardhat_dumpState", func(params json.RawMessage) (any, *rpc.RPCError) {
		return "base64hardhatstate", nil
	})
	srv := httptest.NewServer(handler)
	defer srv.Close()

	hc, err := NewHardhatChain("test", config.ChainConfig{}, nil, "proj")
	require.NoError(t, err)
	hc.rpcClient = NewRPCClient(srv.URL)

	dir := t.TempDir()
	path := filepath.Join(dir, "hardhat-state.json")

	ctx := context.Background()
	err = hc.ExportState(ctx, path)
	require.NoError(t, err)

	data, err := os.ReadFile(path)
	require.NoError(t, err)
	assert.NotEmpty(t, data)
}

func TestHardhatChain_ExportState_RPCError(t *testing.T) {
	handler := newRPCHandler()
	handler.handle("hardhat_dumpState", func(params json.RawMessage) (any, *rpc.RPCError) {
		return nil, &rpc.RPCError{Code: -32000, Message: "not supported"}
	})
	srv := httptest.NewServer(handler)
	defer srv.Close()

	hc, err := NewHardhatChain("test", config.ChainConfig{}, nil, "proj")
	require.NoError(t, err)
	hc.rpcClient = NewRPCClient(srv.URL)

	ctx := context.Background()
	err = hc.ExportState(ctx, "/tmp/state.json")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "hardhat_dumpState failed")
}

func TestHardhatChain_ImportState(t *testing.T) {
	var receivedState string
	handler := newRPCHandler()
	handler.handle("hardhat_loadState", func(params json.RawMessage) (any, *rpc.RPCError) {
		var args []string
		json.Unmarshal(params, &args)
		if len(args) > 0 {
			receivedState = args[0]
		}
		return true, nil
	})
	srv := httptest.NewServer(handler)
	defer srv.Close()

	hc, err := NewHardhatChain("test", config.ChainConfig{}, nil, "proj")
	require.NoError(t, err)
	hc.rpcClient = NewRPCClient(srv.URL)

	dir := t.TempDir()
	path := filepath.Join(dir, "state.json")
	stateData := `"hardhatstatedata"`
	err = os.WriteFile(path, []byte(stateData), 0644)
	require.NoError(t, err)

	ctx := context.Background()
	err = hc.ImportState(ctx, path)
	require.NoError(t, err)
	assert.Equal(t, "hardhatstatedata", receivedState)
}

func TestHardhatChain_ImportState_NonExistentFile(t *testing.T) {
	hc, err := NewHardhatChain("test", config.ChainConfig{}, nil, "proj")
	require.NoError(t, err)
	hc.rpcClient = NewRPCClient("http://localhost:8545")

	ctx := context.Background()
	err = hc.ImportState(ctx, "/nonexistent/state.json")
	require.Error(t, err)
}

func TestHardhatChain_ImportState_RPCError(t *testing.T) {
	handler := newRPCHandler()
	handler.handle("hardhat_loadState", func(params json.RawMessage) (any, *rpc.RPCError) {
		return nil, &rpc.RPCError{Code: -32000, Message: "load failed"}
	})
	srv := httptest.NewServer(handler)
	defer srv.Close()

	hc, err := NewHardhatChain("test", config.ChainConfig{}, nil, "proj")
	require.NoError(t, err)
	hc.rpcClient = NewRPCClient(srv.URL)

	dir := t.TempDir()
	path := filepath.Join(dir, "state.json")
	err = os.WriteFile(path, []byte(`"somestate"`), 0644)
	require.NoError(t, err)

	ctx := context.Background()
	err = hc.ImportState(ctx, path)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "hardhat_loadState failed")
}

func TestHardhatChain_ImportState_RawData(t *testing.T) {
	handler := newRPCHandler()
	handler.handle("hardhat_loadState", func(params json.RawMessage) (any, *rpc.RPCError) {
		return true, nil
	})
	srv := httptest.NewServer(handler)
	defer srv.Close()

	hc, err := NewHardhatChain("test", config.ChainConfig{}, nil, "proj")
	require.NoError(t, err)
	hc.rpcClient = NewRPCClient(srv.URL)

	dir := t.TempDir()
	path := filepath.Join(dir, "state.raw")
	err = os.WriteFile(path, []byte("rawdata"), 0644)
	require.NoError(t, err)

	ctx := context.Background()
	err = hc.ImportState(ctx, path)
	require.NoError(t, err)
}

func TestHardhatChain_Fork(t *testing.T) {
	handler := newRPCHandler()
	handler.handle("hardhat_reset", func(params json.RawMessage) (any, *rpc.RPCError) {
		return true, nil
	})
	srv := httptest.NewServer(handler)
	defer srv.Close()

	hc, err := NewHardhatChain("test", config.ChainConfig{}, nil, "proj")
	require.NoError(t, err)
	hc.rpcClient = NewRPCClient(srv.URL)

	ctx := context.Background()
	opts := chain.ForkOptions{
		Network:     "sepolia",
		BlockNumber: 5000000,
	}
	err = hc.Fork(ctx, opts)
	require.NoError(t, err)

	info := hc.ForkInfo()
	require.NotNil(t, info)
	assert.Equal(t, "sepolia", info.Network)
	assert.Equal(t, uint64(5000000), info.BlockNumber)
	assert.Equal(t, "https://rpc.sepolia.org", info.RPCURL)
}

func TestHardhatChain_Fork_CustomURL(t *testing.T) {
	handler := newRPCHandler()
	handler.handle("hardhat_reset", func(params json.RawMessage) (any, *rpc.RPCError) {
		return true, nil
	})
	srv := httptest.NewServer(handler)
	defer srv.Close()

	hc, err := NewHardhatChain("test", config.ChainConfig{}, nil, "proj")
	require.NoError(t, err)
	hc.rpcClient = NewRPCClient(srv.URL)

	ctx := context.Background()
	customURL := "https://my-node.example.com"
	err = hc.Fork(ctx, chain.ForkOptions{Network: customURL})
	require.NoError(t, err)

	info := hc.ForkInfo()
	require.NotNil(t, info)
	assert.Equal(t, customURL, info.RPCURL)
}

func TestHardhatChain_Fork_RPCError(t *testing.T) {
	handler := newRPCHandler()
	handler.handle("hardhat_reset", func(params json.RawMessage) (any, *rpc.RPCError) {
		return nil, &rpc.RPCError{Code: -32000, Message: "reset failed"}
	})
	srv := httptest.NewServer(handler)
	defer srv.Close()

	hc, err := NewHardhatChain("test", config.ChainConfig{}, nil, "proj")
	require.NoError(t, err)
	hc.rpcClient = NewRPCClient(srv.URL)

	ctx := context.Background()
	err = hc.Fork(ctx, chain.ForkOptions{Network: "mainnet"})
	require.Error(t, err)
	assert.Nil(t, hc.ForkInfo())
}

func TestHardhatChain_RPC(t *testing.T) {
	handler := newRPCHandler()
	handler.handle("net_version", func(params json.RawMessage) (any, *rpc.RPCError) {
		return "31337", nil
	})
	srv := httptest.NewServer(handler)
	defer srv.Close()

	hc, err := NewHardhatChain("test", config.ChainConfig{}, nil, "proj")
	require.NoError(t, err)
	hc.rpcClient = NewRPCClient(srv.URL)

	ctx := context.Background()
	result, err := hc.RPC(ctx, "net_version")
	require.NoError(t, err)

	var version string
	err = json.Unmarshal(result, &version)
	require.NoError(t, err)
	assert.Equal(t, "31337", version)
}

func TestHardhatChain_ImplementsChainInterface(t *testing.T) {
	hc, err := NewHardhatChain("test", config.ChainConfig{ChainID: 31337}, nil, "proj")
	require.NoError(t, err)
	var _ chain.Chain = hc
}

func TestHardhatChain_Fork_WithBlockNumberZero(t *testing.T) {
	handler := newRPCHandler()
	handler.handle("hardhat_reset", func(params json.RawMessage) (any, *rpc.RPCError) {
		return true, nil
	})
	srv := httptest.NewServer(handler)
	defer srv.Close()

	hc, err := NewHardhatChain("test", config.ChainConfig{}, nil, "proj")
	require.NoError(t, err)
	hc.rpcClient = NewRPCClient(srv.URL)

	ctx := context.Background()
	opts := chain.ForkOptions{
		Network:     "mainnet",
		BlockNumber: 0,
	}
	err = hc.Fork(ctx, opts)
	require.NoError(t, err)

	info := hc.ForkInfo()
	require.NotNil(t, info)
	assert.Equal(t, uint64(0), info.BlockNumber)
}

func TestHardhatChain_FundAccount_Error(t *testing.T) {
	handler := newRPCHandler()
	handler.handle("hardhat_setBalance", func(params json.RawMessage) (any, *rpc.RPCError) {
		return nil, &rpc.RPCError{Code: -32000, Message: "failed"}
	})
	srv := httptest.NewServer(handler)
	defer srv.Close()

	hc, err := NewHardhatChain("test", config.ChainConfig{}, nil, "proj")
	require.NoError(t, err)
	hc.rpcClient = NewRPCClient(srv.URL)

	ctx := context.Background()
	err = hc.FundAccount(ctx, "0x1234", big.NewInt(1000))
	require.Error(t, err)
}

func TestHardhatChain_MineBlocks_Error(t *testing.T) {
	handler := newRPCHandler()
	handler.handle("hardhat_mine", func(params json.RawMessage) (any, *rpc.RPCError) {
		return nil, &rpc.RPCError{Code: -32000, Message: "mine failed"}
	})
	srv := httptest.NewServer(handler)
	defer srv.Close()

	hc, err := NewHardhatChain("test", config.ChainConfig{}, nil, "proj")
	require.NoError(t, err)
	hc.rpcClient = NewRPCClient(srv.URL)

	ctx := context.Background()
	err = hc.MineBlocks(ctx, 10)
	require.Error(t, err)
}
