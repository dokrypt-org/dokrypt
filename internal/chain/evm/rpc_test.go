package evm

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/dokrypt/dokrypt/internal/rpc"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func newTestRPCServer(t *testing.T, handler func(method string, params json.RawMessage) (any, *rpc.RPCError)) *httptest.Server {
	t.Helper()
	return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
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

		result, rpcErr := handler(req.Method, req.Params)

		resp := map[string]any{
			"jsonrpc": "2.0",
			"id":      req.ID,
		}
		if rpcErr != nil {
			resp["error"] = rpcErr
		} else {
			resultJSON, _ := json.Marshal(result)
			resp["result"] = json.RawMessage(resultJSON)
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(resp)
	}))
}

func newTestBatchRPCServer(t *testing.T, handler func(method string, params json.RawMessage) (any, *rpc.RPCError)) *httptest.Server {
	t.Helper()
	return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var rawBody json.RawMessage
		if err := json.NewDecoder(r.Body).Decode(&rawBody); err != nil {
			http.Error(w, "bad request", http.StatusBadRequest)
			return
		}

		w.Header().Set("Content-Type", "application/json")

		if len(rawBody) > 0 && rawBody[0] == '[' {
			var reqs []struct {
				JSONRPC string          `json:"jsonrpc"`
				Method  string          `json:"method"`
				Params  json.RawMessage `json:"params"`
				ID      int64           `json:"id"`
			}
			if err := json.Unmarshal(rawBody, &reqs); err != nil {
				http.Error(w, "bad batch request", http.StatusBadRequest)
				return
			}

			var responses []map[string]any
			for _, req := range reqs {
				result, rpcErr := handler(req.Method, req.Params)
				resp := map[string]any{
					"jsonrpc": "2.0",
					"id":      req.ID,
				}
				if rpcErr != nil {
					resp["error"] = rpcErr
				} else {
					resultJSON, _ := json.Marshal(result)
					resp["result"] = json.RawMessage(resultJSON)
				}
				responses = append(responses, resp)
			}
			json.NewEncoder(w).Encode(responses)
		} else {
			var req struct {
				JSONRPC string          `json:"jsonrpc"`
				Method  string          `json:"method"`
				Params  json.RawMessage `json:"params"`
				ID      int64           `json:"id"`
			}
			if err := json.Unmarshal(rawBody, &req); err != nil {
				http.Error(w, "bad request", http.StatusBadRequest)
				return
			}
			result, rpcErr := handler(req.Method, req.Params)
			resp := map[string]any{
				"jsonrpc": "2.0",
				"id":      req.ID,
			}
			if rpcErr != nil {
				resp["error"] = rpcErr
			} else {
				resultJSON, _ := json.Marshal(result)
				resp["result"] = json.RawMessage(resultJSON)
			}
			json.NewEncoder(w).Encode(resp)
		}
	}))
}

func TestNewRPCClient(t *testing.T) {
	client := NewRPCClient("http://localhost:8545")
	require.NotNil(t, client)
	assert.Equal(t, "http://localhost:8545", client.URL())
	assert.NotNil(t, client.Inner())
}

func TestRPCClient_Call_Success(t *testing.T) {
	srv := newTestRPCServer(t, func(method string, params json.RawMessage) (any, *rpc.RPCError) {
		if method == "eth_blockNumber" {
			return "0x10", nil
		}
		return nil, &rpc.RPCError{Code: -32601, Message: "method not found"}
	})
	defer srv.Close()

	client := NewRPCClient(srv.URL)
	ctx := context.Background()

	result, err := client.Call(ctx, "eth_blockNumber")
	require.NoError(t, err)

	var blockNum string
	err = json.Unmarshal(result, &blockNum)
	require.NoError(t, err)
	assert.Equal(t, "0x10", blockNum)
}

func TestRPCClient_Call_WithParams(t *testing.T) {
	srv := newTestRPCServer(t, func(method string, params json.RawMessage) (any, *rpc.RPCError) {
		if method == "eth_getBalance" {
			return "0xde0b6b3a7640000", nil
		}
		return nil, &rpc.RPCError{Code: -32601, Message: "method not found"}
	})
	defer srv.Close()

	client := NewRPCClient(srv.URL)
	ctx := context.Background()

	result, err := client.Call(ctx, "eth_getBalance", "0x1234", "latest")
	require.NoError(t, err)

	var balance string
	err = json.Unmarshal(result, &balance)
	require.NoError(t, err)
	assert.Equal(t, "0xde0b6b3a7640000", balance)
}

func TestRPCClient_Call_RPCError(t *testing.T) {
	srv := newTestRPCServer(t, func(method string, params json.RawMessage) (any, *rpc.RPCError) {
		return nil, &rpc.RPCError{Code: -32601, Message: "method not found"}
	})
	defer srv.Close()

	client := NewRPCClient(srv.URL)
	ctx := context.Background()

	_, err := client.Call(ctx, "invalid_method")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "method not found")
}

func TestRPCClient_CallResult_Success(t *testing.T) {
	srv := newTestRPCServer(t, func(method string, params json.RawMessage) (any, *rpc.RPCError) {
		if method == "eth_accounts" {
			return []string{"0xabc123", "0xdef456"}, nil
		}
		return nil, &rpc.RPCError{Code: -32601, Message: "method not found"}
	})
	defer srv.Close()

	client := NewRPCClient(srv.URL)
	ctx := context.Background()

	var accounts []string
	err := client.CallResult(ctx, &accounts, "eth_accounts")
	require.NoError(t, err)
	assert.Len(t, accounts, 2)
	assert.Equal(t, "0xabc123", accounts[0])
	assert.Equal(t, "0xdef456", accounts[1])
}

func TestRPCClient_CallResult_Error(t *testing.T) {
	srv := newTestRPCServer(t, func(method string, params json.RawMessage) (any, *rpc.RPCError) {
		return nil, &rpc.RPCError{Code: -32000, Message: "internal error"}
	})
	defer srv.Close()

	client := NewRPCClient(srv.URL)
	ctx := context.Background()

	var result string
	err := client.CallResult(ctx, &result, "some_method")
	require.Error(t, err)
}

func TestRPCClient_BatchCall_Success(t *testing.T) {
	srv := newTestBatchRPCServer(t, func(method string, params json.RawMessage) (any, *rpc.RPCError) {
		switch method {
		case "eth_blockNumber":
			return "0x10", nil
		case "eth_chainId":
			return "0x7a69", nil
		default:
			return nil, &rpc.RPCError{Code: -32601, Message: "method not found"}
		}
	})
	defer srv.Close()

	client := NewRPCClient(srv.URL)
	ctx := context.Background()

	calls := []rpc.BatchRequest{
		{Method: "eth_blockNumber"},
		{Method: "eth_chainId"},
	}

	responses, err := client.BatchCall(ctx, calls)
	require.NoError(t, err)
	assert.Len(t, responses, 2)

	for _, resp := range responses {
		assert.Nil(t, resp.Error)
		assert.NotNil(t, resp.Result)
	}
}

func TestRPCClient_SetURL(t *testing.T) {
	client := NewRPCClient("http://localhost:8545")
	assert.Equal(t, "http://localhost:8545", client.URL())

	client.SetURL("http://localhost:9545")
	assert.Equal(t, "http://localhost:9545", client.URL())
}

func TestRPCClient_URL(t *testing.T) {
	url := "http://example.com:8545"
	client := NewRPCClient(url)
	assert.Equal(t, url, client.URL())
}

func TestRPCClient_Inner(t *testing.T) {
	client := NewRPCClient("http://localhost:8545")
	inner := client.Inner()
	require.NotNil(t, inner)
	assert.Equal(t, "http://localhost:8545", inner.URL())
}

func TestRPCClient_Call_ConnectionRefused(t *testing.T) {
	client := NewRPCClient("http://127.0.0.1:1")
	ctx := context.Background()

	_, err := client.Call(ctx, "eth_blockNumber")
	require.Error(t, err)
}

func TestRPCClient_Call_ContextCanceled(t *testing.T) {
	srv := newTestRPCServer(t, func(method string, params json.RawMessage) (any, *rpc.RPCError) {
		return "0x1", nil
	})
	defer srv.Close()

	client := NewRPCClient(srv.URL)
	ctx, cancel := context.WithCancel(context.Background())
	cancel() // Cancel immediately

	_, err := client.Call(ctx, "eth_blockNumber")
	require.Error(t, err)
}
