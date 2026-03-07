package evm

import (
	"context"
	"encoding/json"
	"math/big"
	"net/http/httptest"
	"testing"

	"github.com/dokrypt/dokrypt/internal/chain"
	"github.com/dokrypt/dokrypt/internal/config"
	"github.com/dokrypt/dokrypt/internal/rpc"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewGethChain(t *testing.T) {
	cfg := config.ChainConfig{
		Engine:  "geth",
		ChainID: 1337,
	}

	gc, err := NewGethChain("test-geth", cfg, nil, "my-project")
	require.NoError(t, err)
	require.NotNil(t, gc)

	assert.Equal(t, "test-geth", gc.name)
	assert.Equal(t, uint64(1337), gc.cfg.ChainID)
	assert.Equal(t, "my-project", gc.projectName)
	assert.Equal(t, 8545, gc.hostPort)
	assert.Nil(t, gc.runtime)
	assert.Empty(t, gc.containerID)
}

func TestGethChain_Name(t *testing.T) {
	gc, err := NewGethChain("geth-node", config.ChainConfig{ChainID: 1337}, nil, "proj")
	require.NoError(t, err)
	assert.Equal(t, "geth-node", gc.Name())
}

func TestGethChain_ChainID(t *testing.T) {
	gc, err := NewGethChain("test", config.ChainConfig{ChainID: 12345}, nil, "proj")
	require.NoError(t, err)
	assert.Equal(t, uint64(12345), gc.ChainID())
}

func TestGethChain_Engine(t *testing.T) {
	gc, err := NewGethChain("test", config.ChainConfig{}, nil, "proj")
	require.NoError(t, err)
	assert.Equal(t, "geth", gc.Engine())
}

func TestGethChain_RPCURL(t *testing.T) {
	gc, err := NewGethChain("test", config.ChainConfig{}, nil, "proj")
	require.NoError(t, err)
	assert.Equal(t, "http://localhost:8545", gc.RPCURL())

	gc.hostPort = 7777
	assert.Equal(t, "http://localhost:7777", gc.RPCURL())
}

func TestGethChain_WSURL(t *testing.T) {
	gc, err := NewGethChain("test", config.ChainConfig{}, nil, "proj")
	require.NoError(t, err)
	assert.Equal(t, "ws://localhost:0", gc.WSURL())

	gc.wsHostPort = 8546
	assert.Equal(t, "ws://localhost:8546", gc.WSURL())
}

func TestGethChain_ContainerID(t *testing.T) {
	gc, err := NewGethChain("test", config.ChainConfig{}, nil, "proj")
	require.NoError(t, err)
	assert.Empty(t, gc.ContainerID())

	gc.containerID = "container123"
	assert.Equal(t, "container123", gc.ContainerID())
}

func TestGethChain_Accounts_Empty(t *testing.T) {
	gc, err := NewGethChain("test", config.ChainConfig{}, nil, "proj")
	require.NoError(t, err)
	assert.Nil(t, gc.Accounts())
}

func TestGethChain_ForkInfo_Nil(t *testing.T) {
	gc, err := NewGethChain("test", config.ChainConfig{}, nil, "proj")
	require.NoError(t, err)
	assert.Nil(t, gc.ForkInfo())
}

func TestGethChain_Health_NotStarted(t *testing.T) {
	gc, err := NewGethChain("test", config.ChainConfig{}, nil, "proj")
	require.NoError(t, err)

	ctx := context.Background()
	err = gc.Health(ctx)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "chain not started")
}

func TestGethChain_Health_WithRPCClient(t *testing.T) {
	handler := newRPCHandler()
	handler.handle("eth_blockNumber", func(params json.RawMessage) (any, *rpc.RPCError) {
		return "0x5", nil
	})
	srv := httptest.NewServer(handler)
	defer srv.Close()

	gc, err := NewGethChain("test", config.ChainConfig{}, nil, "proj")
	require.NoError(t, err)
	gc.rpcClient = NewRPCClient(srv.URL)

	ctx := context.Background()
	err = gc.Health(ctx)
	require.NoError(t, err)
}

func TestGethChain_IsRunning_NoContainer(t *testing.T) {
	gc, err := NewGethChain("test", config.ChainConfig{}, nil, "proj")
	require.NoError(t, err)
	assert.False(t, gc.IsRunning(context.Background()))
}

func TestGethChain_Stop_NoContainer(t *testing.T) {
	gc, err := NewGethChain("test", config.ChainConfig{}, nil, "proj")
	require.NoError(t, err)
	err = gc.Stop(context.Background())
	require.NoError(t, err)
}

func TestGethChain_Logs_NotStarted(t *testing.T) {
	gc, err := NewGethChain("test", config.ChainConfig{}, nil, "proj")
	require.NoError(t, err)

	_, err = gc.Logs(context.Background(), false)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "chain not started")
}

func TestGethChain_FundAccount_NotSupported(t *testing.T) {
	gc, err := NewGethChain("test", config.ChainConfig{}, nil, "proj")
	require.NoError(t, err)

	ctx := context.Background()
	err = gc.FundAccount(ctx, "0x1234", big.NewInt(1000))
	require.Error(t, err)
	assert.Contains(t, err.Error(), "not supported in geth dev mode")
}

func TestGethChain_ImpersonateAccount_NotSupported(t *testing.T) {
	gc, err := NewGethChain("test", config.ChainConfig{}, nil, "proj")
	require.NoError(t, err)

	err = gc.ImpersonateAccount(context.Background(), "0x1234")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "not supported in geth dev mode")
}

func TestGethChain_SetBlockTime_NotSupported(t *testing.T) {
	gc, err := NewGethChain("test", config.ChainConfig{}, nil, "proj")
	require.NoError(t, err)

	err = gc.SetBlockTime(context.Background(), 5)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "not supported in geth dev mode")
}

func TestGethChain_SetGasPrice_NotSupported(t *testing.T) {
	gc, err := NewGethChain("test", config.ChainConfig{}, nil, "proj")
	require.NoError(t, err)

	err = gc.SetGasPrice(context.Background(), 20)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "not supported in geth dev mode")
}

func TestGethChain_SetBalance_NotSupported(t *testing.T) {
	gc, err := NewGethChain("test", config.ChainConfig{}, nil, "proj")
	require.NoError(t, err)

	err = gc.SetBalance(context.Background(), "0x1234", big.NewInt(1000))
	require.Error(t, err)
	assert.Contains(t, err.Error(), "not supported in geth dev mode")
}

func TestGethChain_SetStorageAt_NotSupported(t *testing.T) {
	gc, err := NewGethChain("test", config.ChainConfig{}, nil, "proj")
	require.NoError(t, err)

	err = gc.SetStorageAt(context.Background(), "0xaddr", "0x0", "0x1")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "not supported in geth dev mode")
}

func TestGethChain_TakeSnapshot_NotSupported(t *testing.T) {
	gc, err := NewGethChain("test", config.ChainConfig{}, nil, "proj")
	require.NoError(t, err)

	_, err = gc.TakeSnapshot(context.Background())
	require.Error(t, err)
	assert.Contains(t, err.Error(), "not supported in geth dev mode")
}

func TestGethChain_RevertSnapshot_NotSupported(t *testing.T) {
	gc, err := NewGethChain("test", config.ChainConfig{}, nil, "proj")
	require.NoError(t, err)

	err = gc.RevertSnapshot(context.Background(), "0x1")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "not supported in geth dev mode")
}

func TestGethChain_ExportState_NotSupported(t *testing.T) {
	gc, err := NewGethChain("test", config.ChainConfig{}, nil, "proj")
	require.NoError(t, err)

	err = gc.ExportState(context.Background(), "/tmp/state.json")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "not supported in geth dev mode")
}

func TestGethChain_ImportState_NotSupported(t *testing.T) {
	gc, err := NewGethChain("test", config.ChainConfig{}, nil, "proj")
	require.NoError(t, err)

	err = gc.ImportState(context.Background(), "/tmp/state.json")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "not supported in geth dev mode")
}

func TestGethChain_Fork_NotSupported(t *testing.T) {
	gc, err := NewGethChain("test", config.ChainConfig{}, nil, "proj")
	require.NoError(t, err)

	err = gc.Fork(context.Background(), chain.ForkOptions{Network: "mainnet"})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "not supported in geth dev mode")
}

func TestGethChain_MineBlocks(t *testing.T) {
	callCount := 0
	handler := newRPCHandler()
	handler.handle("evm_mine", func(params json.RawMessage) (any, *rpc.RPCError) {
		callCount++
		return true, nil
	})
	srv := httptest.NewServer(handler)
	defer srv.Close()

	gc, err := NewGethChain("test", config.ChainConfig{}, nil, "proj")
	require.NoError(t, err)
	gc.rpcClient = NewRPCClient(srv.URL)

	ctx := context.Background()
	err = gc.MineBlocks(ctx, 5)
	require.NoError(t, err)

	assert.Equal(t, 5, callCount)
}

func TestGethChain_MineBlocks_Error(t *testing.T) {
	handler := newRPCHandler()
	handler.handle("evm_mine", func(params json.RawMessage) (any, *rpc.RPCError) {
		return nil, &rpc.RPCError{Code: -32000, Message: "mine failed"}
	})
	srv := httptest.NewServer(handler)
	defer srv.Close()

	gc, err := NewGethChain("test", config.ChainConfig{}, nil, "proj")
	require.NoError(t, err)
	gc.rpcClient = NewRPCClient(srv.URL)

	ctx := context.Background()
	err = gc.MineBlocks(ctx, 5)
	require.Error(t, err)
}

func TestGethChain_TimeTravel(t *testing.T) {
	handler := newRPCHandler()
	handler.handle("evm_increaseTime", func(params json.RawMessage) (any, *rpc.RPCError) {
		return true, nil
	})
	handler.handle("evm_mine", func(params json.RawMessage) (any, *rpc.RPCError) {
		return true, nil
	})
	srv := httptest.NewServer(handler)
	defer srv.Close()

	gc, err := NewGethChain("test", config.ChainConfig{}, nil, "proj")
	require.NoError(t, err)
	gc.rpcClient = NewRPCClient(srv.URL)

	ctx := context.Background()
	err = gc.TimeTravel(ctx, 7200) // 2 hours
	require.NoError(t, err)
}

func TestGethChain_TimeTravel_IncreaseTimeError(t *testing.T) {
	handler := newRPCHandler()
	handler.handle("evm_increaseTime", func(params json.RawMessage) (any, *rpc.RPCError) {
		return nil, &rpc.RPCError{Code: -32000, Message: "error"}
	})
	srv := httptest.NewServer(handler)
	defer srv.Close()

	gc, err := NewGethChain("test", config.ChainConfig{}, nil, "proj")
	require.NoError(t, err)
	gc.rpcClient = NewRPCClient(srv.URL)

	ctx := context.Background()
	err = gc.TimeTravel(ctx, 3600)
	require.Error(t, err)
}

func TestGethChain_RPC(t *testing.T) {
	handler := newRPCHandler()
	handler.handle("eth_chainId", func(params json.RawMessage) (any, *rpc.RPCError) {
		return "0x539", nil // 1337
	})
	srv := httptest.NewServer(handler)
	defer srv.Close()

	gc, err := NewGethChain("test", config.ChainConfig{}, nil, "proj")
	require.NoError(t, err)
	gc.rpcClient = NewRPCClient(srv.URL)

	ctx := context.Background()
	result, err := gc.RPC(ctx, "eth_chainId")
	require.NoError(t, err)

	var chainID string
	err = json.Unmarshal(result, &chainID)
	require.NoError(t, err)
	assert.Equal(t, "0x539", chainID)
}

func TestGethChain_ImplementsChainInterface(t *testing.T) {
	gc, err := NewGethChain("test", config.ChainConfig{ChainID: 1337}, nil, "proj")
	require.NoError(t, err)
	var _ chain.Chain = gc
}

func TestGethChain_GenerateAccounts_DefaultCount(t *testing.T) {
	gc, err := NewGethChain("test", config.ChainConfig{
		ChainID:  1337,
		Accounts: 0, // Should default to 10
	}, nil, "proj")
	require.NoError(t, err)

	ctx := context.Background()
	accounts, err := gc.generateAccounts(ctx)
	require.NoError(t, err)
	assert.Len(t, accounts, 10)
}

func TestGethChain_GenerateAccounts_CustomCount(t *testing.T) {
	gc, err := NewGethChain("test", config.ChainConfig{
		ChainID:  1337,
		Accounts: 5,
	}, nil, "proj")
	require.NoError(t, err)

	ctx := context.Background()
	accounts, err := gc.generateAccounts(ctx)
	require.NoError(t, err)
	assert.Len(t, accounts, 5)
}

func TestGethChain_GenerateAccounts_WithBalance(t *testing.T) {
	gc, err := NewGethChain("test", config.ChainConfig{
		ChainID:        1337,
		Accounts:       2,
		AccountBalance: "100",
	}, nil, "proj")
	require.NoError(t, err)

	ctx := context.Background()
	accounts, err := gc.generateAccounts(ctx)
	require.NoError(t, err)
	assert.Len(t, accounts, 2)

	expected := new(big.Int)
	expectedFloat, _ := new(big.Float).SetString("100")
	expectedFloat.Mul(expectedFloat, new(big.Float).SetFloat64(1e18))
	expectedFloat.Int(expected)

	for _, acct := range accounts {
		assert.Equal(t, expected, acct.Balance)
	}
}

func TestGethChain_GenerateAccounts_Labels(t *testing.T) {
	gc, err := NewGethChain("test", config.ChainConfig{
		ChainID:  1337,
		Accounts: 3,
	}, nil, "proj")
	require.NoError(t, err)

	ctx := context.Background()
	accounts, err := gc.generateAccounts(ctx)
	require.NoError(t, err)

	assert.Equal(t, "deployer", accounts[0].Label)
	assert.Equal(t, "user1", accounts[1].Label)
	assert.Equal(t, "user2", accounts[2].Label)
}

func TestGethChain_GenerateAccounts_DeterministicKeys(t *testing.T) {
	gc, err := NewGethChain("test", config.ChainConfig{
		ChainID:  1337,
		Accounts: 3,
	}, nil, "proj")
	require.NoError(t, err)

	ctx := context.Background()
	accounts1, err := gc.generateAccounts(ctx)
	require.NoError(t, err)

	accounts2, err := gc.generateAccounts(ctx)
	require.NoError(t, err)

	for i := range accounts1 {
		assert.Equal(t, accounts1[i].Address, accounts2[i].Address)
		assert.Equal(t, accounts1[i].PrivateKey, accounts2[i].PrivateKey)
	}
}
