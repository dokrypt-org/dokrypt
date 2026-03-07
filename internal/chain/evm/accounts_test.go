package evm

import (
	"context"
	"encoding/json"
	"fmt"
	"math/big"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/dokrypt/dokrypt/internal/common"
	"github.com/dokrypt/dokrypt/internal/rpc"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type rpcHandler struct {
	handlers map[string]func(params json.RawMessage) (any, *rpc.RPCError)
}

func newRPCHandler() *rpcHandler {
	return &rpcHandler{
		handlers: make(map[string]func(params json.RawMessage) (any, *rpc.RPCError)),
	}
}

func (h *rpcHandler) handle(method string, fn func(params json.RawMessage) (any, *rpc.RPCError)) {
	h.handlers[method] = fn
}

func (h *rpcHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	var req struct {
		JSONRPC string          `json:"jsonrpc"`
		Method  string          `json:"method"`
		Params  json.RawMessage `json:"params"`
		ID      int64           `json:"id"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "bad request", http.StatusBadRequest)
		return
	}

	resp := map[string]any{
		"jsonrpc": "2.0",
		"id":      req.ID,
	}

	fn, ok := h.handlers[req.Method]
	if !ok {
		resp["error"] = &rpc.RPCError{Code: -32601, Message: fmt.Sprintf("method %q not found", req.Method)}
	} else {
		result, rpcErr := fn(req.Params)
		if rpcErr != nil {
			resp["error"] = rpcErr
		} else {
			resultJSON, _ := json.Marshal(result)
			resp["result"] = json.RawMessage(resultJSON)
		}
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(resp)
}

func TestNewAccountManager_DefaultSeed(t *testing.T) {
	handler := newRPCHandler()
	srv := httptest.NewServer(handler)
	defer srv.Close()

	client := NewRPCClient(srv.URL)
	mgr, err := NewAccountManager(client, 5, nil, "anvil")
	require.NoError(t, err)
	require.NotNil(t, mgr)

	accounts := mgr.Accounts()
	assert.Len(t, accounts, 5)

	for _, acct := range accounts {
		assert.NotEmpty(t, acct.Address)
		assert.NotEmpty(t, acct.PrivateKey)
		assert.True(t, len(acct.Address) > 2, "address should start with 0x")
	}
}

func TestNewAccountManager_CustomSeed(t *testing.T) {
	handler := newRPCHandler()
	srv := httptest.NewServer(handler)
	defer srv.Close()

	client := NewRPCClient(srv.URL)
	seed := []byte("custom-test-seed-123")
	mgr, err := NewAccountManager(client, 3, seed, "anvil")
	require.NoError(t, err)
	require.NotNil(t, mgr)

	accounts := mgr.Accounts()
	assert.Len(t, accounts, 3)
}

func TestNewAccountManager_ZeroCount(t *testing.T) {
	handler := newRPCHandler()
	srv := httptest.NewServer(handler)
	defer srv.Close()

	client := NewRPCClient(srv.URL)
	mgr, err := NewAccountManager(client, 0, nil, "anvil")
	require.NoError(t, err)

	accounts := mgr.Accounts()
	assert.Len(t, accounts, 10)
}

func TestNewAccountManager_NegativeCount(t *testing.T) {
	handler := newRPCHandler()
	srv := httptest.NewServer(handler)
	defer srv.Close()

	client := NewRPCClient(srv.URL)
	mgr, err := NewAccountManager(client, -5, nil, "hardhat")
	require.NoError(t, err)

	accounts := mgr.Accounts()
	assert.Len(t, accounts, 10)
}

func TestNewAccountManager_DeterministicAccounts(t *testing.T) {
	handler := newRPCHandler()
	srv := httptest.NewServer(handler)
	defer srv.Close()

	client := NewRPCClient(srv.URL)

	mgr1, err := NewAccountManager(client, 5, []byte("same-seed"), "anvil")
	require.NoError(t, err)

	mgr2, err := NewAccountManager(client, 5, []byte("same-seed"), "anvil")
	require.NoError(t, err)

	accounts1 := mgr1.Accounts()
	accounts2 := mgr2.Accounts()

	for i := range accounts1 {
		assert.Equal(t, accounts1[i].Address, accounts2[i].Address)
		assert.Equal(t, accounts1[i].PrivateKey, accounts2[i].PrivateKey)
	}
}

func TestNewAccountManager_Labels(t *testing.T) {
	handler := newRPCHandler()
	srv := httptest.NewServer(handler)
	defer srv.Close()

	client := NewRPCClient(srv.URL)
	mgr, err := NewAccountManager(client, 3, nil, "anvil")
	require.NoError(t, err)

	accounts := mgr.Accounts()
	assert.Equal(t, "deployer", accounts[0].Label)
	assert.Equal(t, "user1", accounts[1].Label)
	assert.Equal(t, "user2", accounts[2].Label)
}

func TestAccountManager_FundAccounts_Anvil(t *testing.T) {
	var calledMethods []string
	handler := newRPCHandler()
	handler.handle("anvil_setBalance", func(params json.RawMessage) (any, *rpc.RPCError) {
		calledMethods = append(calledMethods, "anvil_setBalance")
		return true, nil
	})

	srv := httptest.NewServer(handler)
	defer srv.Close()

	client := NewRPCClient(srv.URL)
	mgr, err := NewAccountManager(client, 2, nil, "anvil")
	require.NoError(t, err)

	balance := new(big.Int)
	balance.SetString("10000000000000000000000", 10) // 10000 ETH

	ctx := context.Background()
	err = mgr.FundAccounts(ctx, balance)
	require.NoError(t, err)

	assert.Len(t, calledMethods, 2)

	for _, acct := range mgr.Accounts() {
		assert.Equal(t, balance, acct.Balance)
	}
}

func TestAccountManager_FundAccounts_Hardhat(t *testing.T) {
	var calledMethods []string
	handler := newRPCHandler()
	handler.handle("hardhat_setBalance", func(params json.RawMessage) (any, *rpc.RPCError) {
		calledMethods = append(calledMethods, "hardhat_setBalance")
		return true, nil
	})

	srv := httptest.NewServer(handler)
	defer srv.Close()

	client := NewRPCClient(srv.URL)
	mgr, err := NewAccountManager(client, 2, nil, "hardhat")
	require.NoError(t, err)

	balance := big.NewInt(1000000000000000000) // 1 ETH
	ctx := context.Background()
	err = mgr.FundAccounts(ctx, balance)
	require.NoError(t, err)

	assert.Len(t, calledMethods, 2)
}

func TestAccountManager_FundAccounts_Geth(t *testing.T) {
	handler := newRPCHandler()
	srv := httptest.NewServer(handler)
	defer srv.Close()

	client := NewRPCClient(srv.URL)
	mgr, err := NewAccountManager(client, 2, nil, "geth")
	require.NoError(t, err)

	balance := big.NewInt(1000000000000000000)
	ctx := context.Background()
	err = mgr.FundAccounts(ctx, balance)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "not supported in geth dev mode")
}

func TestAccountManager_FundAccounts_RPCError(t *testing.T) {
	handler := newRPCHandler()
	handler.handle("anvil_setBalance", func(params json.RawMessage) (any, *rpc.RPCError) {
		return nil, &rpc.RPCError{Code: -32000, Message: "internal error"}
	})

	srv := httptest.NewServer(handler)
	defer srv.Close()

	client := NewRPCClient(srv.URL)
	mgr, err := NewAccountManager(client, 2, nil, "anvil")
	require.NoError(t, err)

	balance := big.NewInt(1000)
	ctx := context.Background()
	err = mgr.FundAccounts(ctx, balance)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "failed to fund account")
}

func TestAccountManager_GetBalance(t *testing.T) {
	handler := newRPCHandler()
	handler.handle("eth_getBalance", func(params json.RawMessage) (any, *rpc.RPCError) {
		return "0xde0b6b3a7640000", nil // 1 ETH in hex wei
	})

	srv := httptest.NewServer(handler)
	defer srv.Close()

	client := NewRPCClient(srv.URL)
	mgr, err := NewAccountManager(client, 1, nil, "anvil")
	require.NoError(t, err)

	ctx := context.Background()
	balance, err := mgr.GetBalance(ctx, "0x1234567890abcdef1234567890abcdef12345678")
	require.NoError(t, err)

	expectedBalance := new(big.Int)
	expectedBalance.SetString("de0b6b3a7640000", 16) // 1 ETH
	assert.Equal(t, expectedBalance, balance)
}

func TestAccountManager_GetBalance_NoPrefix(t *testing.T) {
	handler := newRPCHandler()
	handler.handle("eth_getBalance", func(params json.RawMessage) (any, *rpc.RPCError) {
		return "de0b6b3a7640000", nil // Without 0x prefix
	})

	srv := httptest.NewServer(handler)
	defer srv.Close()

	client := NewRPCClient(srv.URL)
	mgr, err := NewAccountManager(client, 1, nil, "anvil")
	require.NoError(t, err)

	ctx := context.Background()
	balance, err := mgr.GetBalance(ctx, "0x1234")
	require.NoError(t, err)

	expectedBalance := new(big.Int)
	expectedBalance.SetString("de0b6b3a7640000", 16)
	assert.Equal(t, expectedBalance, balance)
}

func TestAccountManager_GetBalance_RPCError(t *testing.T) {
	handler := newRPCHandler()
	handler.handle("eth_getBalance", func(params json.RawMessage) (any, *rpc.RPCError) {
		return nil, &rpc.RPCError{Code: -32000, Message: "server error"}
	})

	srv := httptest.NewServer(handler)
	defer srv.Close()

	client := NewRPCClient(srv.URL)
	mgr, err := NewAccountManager(client, 1, nil, "anvil")
	require.NoError(t, err)

	ctx := context.Background()
	_, err = mgr.GetBalance(ctx, "0x1234")
	require.Error(t, err)
}

func TestAccountManager_List(t *testing.T) {
	callCount := 0
	handler := newRPCHandler()
	handler.handle("eth_getBalance", func(params json.RawMessage) (any, *rpc.RPCError) {
		callCount++
		return "0x1000", nil
	})

	srv := httptest.NewServer(handler)
	defer srv.Close()

	client := NewRPCClient(srv.URL)
	mgr, err := NewAccountManager(client, 3, nil, "anvil")
	require.NoError(t, err)

	ctx := context.Background()
	accounts, err := mgr.List(ctx)
	require.NoError(t, err)
	assert.Len(t, accounts, 3)

	assert.Equal(t, 3, callCount)

	expectedBal := new(big.Int)
	expectedBal.SetString("1000", 16)
	for _, acct := range accounts {
		assert.Equal(t, expectedBal, acct.Balance)
	}
}

func TestAccountManager_List_PartialFailure(t *testing.T) {
	callCount := 0
	handler := newRPCHandler()
	handler.handle("eth_getBalance", func(params json.RawMessage) (any, *rpc.RPCError) {
		callCount++
		if callCount == 2 {
			return nil, &rpc.RPCError{Code: -32000, Message: "error"}
		}
		return "0x2000", nil
	})

	srv := httptest.NewServer(handler)
	defer srv.Close()

	client := NewRPCClient(srv.URL)
	mgr, err := NewAccountManager(client, 3, nil, "anvil")
	require.NoError(t, err)

	ctx := context.Background()
	accounts, err := mgr.List(ctx)
	require.NoError(t, err)
	assert.Len(t, accounts, 3)
}

func TestAccountManager_Import(t *testing.T) {
	handler := newRPCHandler()
	srv := httptest.NewServer(handler)
	defer srv.Close()

	client := NewRPCClient(srv.URL)
	mgr, err := NewAccountManager(client, 1, nil, "anvil")
	require.NoError(t, err)

	key, err := common.GenerateKeyPair()
	require.NoError(t, err)
	privHex := common.PrivateKeyToHex(key)
	expectedAddr := common.AddressFromPrivateKey(key)

	acct, err := mgr.Import(privHex, "imported-account")
	require.NoError(t, err)
	require.NotNil(t, acct)
	assert.Equal(t, expectedAddr, acct.Address)
	assert.Equal(t, "imported-account", acct.Label)

	accounts := mgr.Accounts()
	assert.Len(t, accounts, 2)
}

func TestAccountManager_Import_With0xPrefix(t *testing.T) {
	handler := newRPCHandler()
	srv := httptest.NewServer(handler)
	defer srv.Close()

	client := NewRPCClient(srv.URL)
	mgr, err := NewAccountManager(client, 0, nil, "anvil")
	require.NoError(t, err)

	key, err := common.GenerateKeyPair()
	require.NoError(t, err)
	privHex := common.PrivateKeyToHex(key) // Has 0x prefix

	acct, err := mgr.Import(privHex, "test")
	require.NoError(t, err)
	assert.NotEmpty(t, acct.Address)
}

func TestAccountManager_Import_InvalidKey(t *testing.T) {
	handler := newRPCHandler()
	srv := httptest.NewServer(handler)
	defer srv.Close()

	client := NewRPCClient(srv.URL)
	mgr, err := NewAccountManager(client, 1, nil, "anvil")
	require.NoError(t, err)

	_, err = mgr.Import("not-a-valid-hex-key", "bad-key")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "failed to parse private key")
}

func TestAccountManager_SetLabel_Success(t *testing.T) {
	handler := newRPCHandler()
	srv := httptest.NewServer(handler)
	defer srv.Close()

	client := NewRPCClient(srv.URL)
	mgr, err := NewAccountManager(client, 2, nil, "anvil")
	require.NoError(t, err)

	accounts := mgr.Accounts()
	addr := accounts[0].Address

	err = mgr.SetLabel(addr, "new-label")
	require.NoError(t, err)

	updated := mgr.Accounts()
	assert.Equal(t, "new-label", updated[0].Label)
}

func TestAccountManager_SetLabel_NotFound(t *testing.T) {
	handler := newRPCHandler()
	srv := httptest.NewServer(handler)
	defer srv.Close()

	client := NewRPCClient(srv.URL)
	mgr, err := NewAccountManager(client, 1, nil, "anvil")
	require.NoError(t, err)

	err = mgr.SetLabel("0xNonExistentAddress", "some-label")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "not found")
}

func TestAccountManager_Accounts_Immutability(t *testing.T) {
	handler := newRPCHandler()
	srv := httptest.NewServer(handler)
	defer srv.Close()

	client := NewRPCClient(srv.URL)
	mgr, err := NewAccountManager(client, 3, nil, "anvil")
	require.NoError(t, err)

	accounts1 := mgr.Accounts()
	accounts2 := mgr.Accounts()

	assert.Equal(t, len(accounts1), len(accounts2))
	for i := range accounts1 {
		assert.Equal(t, accounts1[i].Address, accounts2[i].Address)
	}
}
