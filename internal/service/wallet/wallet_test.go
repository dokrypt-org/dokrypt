package wallet

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewWalletServer(t *testing.T) {
	ws := NewWalletServer("http://localhost:8545")
	require.NotNil(t, ws)
	assert.Equal(t, "http://localhost:8545", ws.chainRPCURL)
	assert.NotNil(t, ws.cooldowns)
	assert.Equal(t, 10*time.Second, ws.cooldown)
}

func TestHandler(t *testing.T) {
	ws := NewWalletServer("http://localhost:8545")
	handler := ws.Handler()
	require.NotNil(t, handler)
}

func TestHandleHealth(t *testing.T) {
	ws := NewWalletServer("http://test-chain:8545")
	handler := ws.Handler()

	req := httptest.NewRequest(http.MethodGet, "/health", nil)
	rr := httptest.NewRecorder()
	handler.ServeHTTP(rr, req)

	assert.Equal(t, http.StatusOK, rr.Code)

	var resp map[string]string
	err := json.Unmarshal(rr.Body.Bytes(), &resp)
	require.NoError(t, err)
	assert.Equal(t, "ok", resp["status"])
	assert.Equal(t, "http://test-chain:8545", resp["chain_rpc"])
}

func TestHandleFundMissingAddress(t *testing.T) {
	ws := NewWalletServer("http://localhost:8545")
	handler := ws.Handler()

	req := httptest.NewRequest(http.MethodPost, "/fund", strings.NewReader(`{}`))
	req.Header.Set("Content-Type", "application/json")
	rr := httptest.NewRecorder()
	handler.ServeHTTP(rr, req)

	assert.Equal(t, http.StatusBadRequest, rr.Code)
	assert.Contains(t, rr.Body.String(), "address required")
}

func TestHandleFundInvalidJSON(t *testing.T) {
	ws := NewWalletServer("http://localhost:8545")
	handler := ws.Handler()

	req := httptest.NewRequest(http.MethodPost, "/fund", strings.NewReader(`not json`))
	req.Header.Set("Content-Type", "application/json")
	rr := httptest.NewRecorder()
	handler.ServeHTTP(rr, req)

	assert.Equal(t, http.StatusBadRequest, rr.Code)
	assert.Contains(t, rr.Body.String(), "invalid request")
}

func mockRPCServer(t *testing.T) *httptest.Server {
	t.Helper()
	return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var req jsonRPCRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			w.WriteHeader(http.StatusBadRequest)
			return
		}

		w.Header().Set("Content-Type", "application/json")

		switch req.Method {
		case "eth_accounts":
			json.NewEncoder(w).Encode(jsonRPCResponse{
				JSONRPC: "2.0",
				ID:      req.ID,
				Result:  json.RawMessage(`["0x1234567890abcdef1234567890abcdef12345678"]`),
			})
		case "eth_sendTransaction":
			json.NewEncoder(w).Encode(jsonRPCResponse{
				JSONRPC: "2.0",
				ID:      req.ID,
				Result:  json.RawMessage(`"0xabcdef1234567890abcdef1234567890abcdef1234567890abcdef1234567890"`),
			})
		case "eth_getBalance":
			json.NewEncoder(w).Encode(jsonRPCResponse{
				JSONRPC: "2.0",
				ID:      req.ID,
				Result:  json.RawMessage(`"0xde0b6b3a7640000"`), // 1 ETH in wei
			})
		default:
			json.NewEncoder(w).Encode(jsonRPCResponse{
				JSONRPC: "2.0",
				ID:      req.ID,
				Error:   &jsonRPCError{Code: -32601, Message: "method not found"},
			})
		}
	}))
}

func TestHandleFundSuccess(t *testing.T) {
	rpcServer := mockRPCServer(t)
	defer rpcServer.Close()

	ws := NewWalletServer(rpcServer.URL)
	handler := ws.Handler()

	body := `{"address":"0xaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa","amount":"1.5"}`
	req := httptest.NewRequest(http.MethodPost, "/fund", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	rr := httptest.NewRecorder()
	handler.ServeHTTP(rr, req)

	assert.Equal(t, http.StatusOK, rr.Code)

	var resp map[string]string
	err := json.Unmarshal(rr.Body.Bytes(), &resp)
	require.NoError(t, err)
	assert.Equal(t, "funded", resp["status"])
	assert.Equal(t, "0xaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa", resp["address"])
	assert.Equal(t, "1.5 ETH", resp["amount"])
	assert.NotEmpty(t, resp["tx_hash"])
	assert.NotEmpty(t, resp["wei"])
}

func TestHandleFundDefaultAmount(t *testing.T) {
	rpcServer := mockRPCServer(t)
	defer rpcServer.Close()

	ws := NewWalletServer(rpcServer.URL)
	handler := ws.Handler()

	body := `{"address":"0xbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbb"}`
	req := httptest.NewRequest(http.MethodPost, "/fund", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	rr := httptest.NewRecorder()
	handler.ServeHTTP(rr, req)

	assert.Equal(t, http.StatusOK, rr.Code)

	var resp map[string]string
	err := json.Unmarshal(rr.Body.Bytes(), &resp)
	require.NoError(t, err)
	assert.Equal(t, "1 ETH", resp["amount"])
}

func TestHandleFundInvalidAmount(t *testing.T) {
	rpcServer := mockRPCServer(t)
	defer rpcServer.Close()

	ws := NewWalletServer(rpcServer.URL)
	handler := ws.Handler()

	body := `{"address":"0xcccccccccccccccccccccccccccccccccccccccc","amount":"not-a-number"}`
	req := httptest.NewRequest(http.MethodPost, "/fund", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	rr := httptest.NewRecorder()
	handler.ServeHTTP(rr, req)

	assert.Equal(t, http.StatusBadRequest, rr.Code)
	assert.Contains(t, rr.Body.String(), "invalid amount")
}

func TestHandleFundRateLimiting(t *testing.T) {
	rpcServer := mockRPCServer(t)
	defer rpcServer.Close()

	ws := NewWalletServer(rpcServer.URL)
	ws.cooldown = 1 * time.Hour // Long cooldown to ensure rate limiting triggers
	handler := ws.Handler()

	address := "0xdddddddddddddddddddddddddddddddddddddd"
	body := fmt.Sprintf(`{"address":"%s"}`, address)

	req := httptest.NewRequest(http.MethodPost, "/fund", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	rr := httptest.NewRecorder()
	handler.ServeHTTP(rr, req)
	assert.Equal(t, http.StatusOK, rr.Code)

	req2 := httptest.NewRequest(http.MethodPost, "/fund", strings.NewReader(body))
	req2.Header.Set("Content-Type", "application/json")
	rr2 := httptest.NewRecorder()
	handler.ServeHTTP(rr2, req2)
	assert.Equal(t, http.StatusTooManyRequests, rr2.Code)
	assert.Contains(t, rr2.Body.String(), "rate limited")
}

func TestHandleBalance(t *testing.T) {
	rpcServer := mockRPCServer(t)
	defer rpcServer.Close()

	ws := NewWalletServer(rpcServer.URL)
	handler := ws.Handler()

	req := httptest.NewRequest(http.MethodGet, "/balance/0xeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeee", nil)
	rr := httptest.NewRecorder()
	handler.ServeHTTP(rr, req)

	assert.Equal(t, http.StatusOK, rr.Code)

	var resp map[string]string
	err := json.Unmarshal(rr.Body.Bytes(), &resp)
	require.NoError(t, err)
	assert.Equal(t, "0xeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeee", resp["address"])
	assert.Equal(t, "1000000000000000000", resp["balance"]) // 1 ETH in wei
}

func TestHandleFundRPCAccountsError(t *testing.T) {
	rpcServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var req jsonRPCRequest
		json.NewDecoder(r.Body).Decode(&req)
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(jsonRPCResponse{
			JSONRPC: "2.0",
			ID:      req.ID,
			Error:   &jsonRPCError{Code: -32000, Message: "internal error"},
		})
	}))
	defer rpcServer.Close()

	ws := NewWalletServer(rpcServer.URL)
	handler := ws.Handler()

	body := `{"address":"0xaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa"}`
	req := httptest.NewRequest(http.MethodPost, "/fund", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	rr := httptest.NewRecorder()
	handler.ServeHTTP(rr, req)

	assert.Equal(t, http.StatusInternalServerError, rr.Code)
}

func TestHandleFundNoAccountsAvailable(t *testing.T) {
	rpcServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var req jsonRPCRequest
		json.NewDecoder(r.Body).Decode(&req)
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(jsonRPCResponse{
			JSONRPC: "2.0",
			ID:      req.ID,
			Result:  json.RawMessage(`[]`),
		})
	}))
	defer rpcServer.Close()

	ws := NewWalletServer(rpcServer.URL)
	handler := ws.Handler()

	body := `{"address":"0xaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa"}`
	req := httptest.NewRequest(http.MethodPost, "/fund", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	rr := httptest.NewRecorder()
	handler.ServeHTTP(rr, req)

	assert.Equal(t, http.StatusInternalServerError, rr.Code)
	assert.Contains(t, rr.Body.String(), "no accounts available")
}

func TestHandleBalanceRPCError(t *testing.T) {
	rpcServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var req jsonRPCRequest
		json.NewDecoder(r.Body).Decode(&req)
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(jsonRPCResponse{
			JSONRPC: "2.0",
			ID:      req.ID,
			Error:   &jsonRPCError{Code: -32000, Message: "some rpc error"},
		})
	}))
	defer rpcServer.Close()

	ws := NewWalletServer(rpcServer.URL)
	handler := ws.Handler()

	req := httptest.NewRequest(http.MethodGet, "/balance/0xaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa", nil)
	rr := httptest.NewRecorder()
	handler.ServeHTTP(rr, req)

	assert.Equal(t, http.StatusInternalServerError, rr.Code)
}

func TestHandleFundRPCConnectionError(t *testing.T) {
	ws := NewWalletServer("http://127.0.0.1:1")
	handler := ws.Handler()

	body := `{"address":"0xaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa"}`
	req := httptest.NewRequest(http.MethodPost, "/fund", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	rr := httptest.NewRecorder()
	handler.ServeHTTP(rr, req)

	assert.Equal(t, http.StatusInternalServerError, rr.Code)
	assert.Contains(t, rr.Body.String(), "failed to get accounts from chain")
}

func TestHandleBalanceRPCConnectionError(t *testing.T) {
	ws := NewWalletServer("http://127.0.0.1:1")
	handler := ws.Handler()

	req := httptest.NewRequest(http.MethodGet, "/balance/0xaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa", nil)
	rr := httptest.NewRecorder()
	handler.ServeHTTP(rr, req)

	assert.Equal(t, http.StatusInternalServerError, rr.Code)
	assert.Contains(t, rr.Body.String(), "failed to get balance from chain")
}

func TestRPCCall(t *testing.T) {
	rpcServer := mockRPCServer(t)
	defer rpcServer.Close()

	ws := NewWalletServer(rpcServer.URL)

	t.Run("successful call", func(t *testing.T) {
		resp, err := ws.rpcCall("eth_accounts", []interface{}{})
		require.NoError(t, err)
		require.NotNil(t, resp)
		assert.Nil(t, resp.Error)
		assert.NotEmpty(t, resp.Result)
	})

	t.Run("unknown method", func(t *testing.T) {
		resp, err := ws.rpcCall("unknown_method", []interface{}{})
		require.NoError(t, err)
		require.NotNil(t, resp)
		assert.NotNil(t, resp.Error)
		assert.Equal(t, -32601, resp.Error.Code)
	})
}

func TestRPCCallConnectionError(t *testing.T) {
	ws := NewWalletServer("http://127.0.0.1:1")
	resp, err := ws.rpcCall("eth_accounts", []interface{}{})
	assert.Error(t, err)
	assert.Nil(t, resp)
}

func TestHandleFundTxError(t *testing.T) {
	rpcServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var req jsonRPCRequest
		json.NewDecoder(r.Body).Decode(&req)
		w.Header().Set("Content-Type", "application/json")

		switch req.Method {
		case "eth_accounts":
			json.NewEncoder(w).Encode(jsonRPCResponse{
				JSONRPC: "2.0",
				ID:      req.ID,
				Result:  json.RawMessage(`["0x1234567890abcdef1234567890abcdef12345678"]`),
			})
		case "eth_sendTransaction":
			json.NewEncoder(w).Encode(jsonRPCResponse{
				JSONRPC: "2.0",
				ID:      req.ID,
				Error:   &jsonRPCError{Code: -32000, Message: "insufficient funds"},
			})
		}
	}))
	defer rpcServer.Close()

	ws := NewWalletServer(rpcServer.URL)
	handler := ws.Handler()

	body := `{"address":"0xaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa","amount":"1"}`
	req := httptest.NewRequest(http.MethodPost, "/fund", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	rr := httptest.NewRecorder()
	handler.ServeHTTP(rr, req)

	assert.Equal(t, http.StatusInternalServerError, rr.Code)
}
