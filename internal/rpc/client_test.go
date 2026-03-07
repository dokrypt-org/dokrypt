package rpc

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"sync/atomic"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestRPCError_Error_WithData(t *testing.T) {
	e := &RPCError{Code: -32600, Message: "Invalid Request", Data: "extra info"}
	assert.Equal(t, "RPC error -32600: Invalid Request (extra info)", e.Error())
}

func TestRPCError_Error_WithoutData(t *testing.T) {
	e := &RPCError{Code: -32601, Message: "Method not found"}
	assert.Equal(t, "RPC error -32601: Method not found", e.Error())
}

func TestRPCError_Error_ZeroValues(t *testing.T) {
	e := &RPCError{}
	assert.Equal(t, "RPC error 0: ", e.Error())
}

func TestNewClient_DefaultValues(t *testing.T) {
	c := NewClient("http://localhost:8545")
	require.NotNil(t, c)
	assert.Equal(t, "http://localhost:8545", c.url)
	assert.Equal(t, 3, c.retries)
	assert.Equal(t, 30*time.Second, c.timeout)
	assert.NotNil(t, c.httpClient)
}

func TestNewClient_WithRetries(t *testing.T) {
	c := NewClient("http://localhost:8545", WithRetries(5))
	assert.Equal(t, 5, c.retries)
}

func TestNewClient_WithTimeout(t *testing.T) {
	c := NewClient("http://localhost:8545", WithTimeout(10*time.Second))
	assert.Equal(t, 10*time.Second, c.timeout)
}

func TestNewClient_WithHTTPClient(t *testing.T) {
	hc := &http.Client{Timeout: 5 * time.Second}
	c := NewClient("http://localhost:8545", WithHTTPClient(hc))
	assert.Same(t, hc, c.httpClient)
}

func TestNewClient_MultipleOptions(t *testing.T) {
	hc := &http.Client{}
	c := NewClient("http://localhost:8545",
		WithRetries(1),
		WithTimeout(5*time.Second),
		WithHTTPClient(hc),
	)
	assert.Equal(t, 1, c.retries)
	assert.Equal(t, 5*time.Second, c.timeout)
	assert.Same(t, hc, c.httpClient)
}

func TestClient_URL(t *testing.T) {
	c := NewClient("http://example.com")
	assert.Equal(t, "http://example.com", c.URL())
}

func TestClient_SetURL(t *testing.T) {
	c := NewClient("http://old.com")
	c.SetURL("http://new.com")
	assert.Equal(t, "http://new.com", c.URL())
}

func TestClient_Call_Success(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "application/json", r.Header.Get("Content-Type"))
		assert.Equal(t, http.MethodPost, r.Method)

		body, err := io.ReadAll(r.Body)
		require.NoError(t, err)

		var req Request
		require.NoError(t, json.Unmarshal(body, &req))
		assert.Equal(t, "2.0", req.JSONRPC)
		assert.Equal(t, "eth_blockNumber", req.Method)

		resp := Response{
			JSONRPC: "2.0",
			Result:  json.RawMessage(`"0x10"`),
			ID:      req.ID,
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(resp)
	}))
	defer server.Close()

	c := NewClient(server.URL, WithRetries(0))
	result, err := c.Call(context.Background(), "eth_blockNumber")
	require.NoError(t, err)
	assert.JSONEq(t, `"0x10"`, string(result))
}

func TestClient_Call_WithParams(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		body, _ := io.ReadAll(r.Body)
		var req Request
		json.Unmarshal(body, &req)

		paramsJSON, _ := json.Marshal(req.Params)
		assert.Contains(t, string(paramsJSON), "0xabc")

		resp := Response{
			JSONRPC: "2.0",
			Result:  json.RawMessage(`{"balance":"0x100"}`),
			ID:      req.ID,
		}
		json.NewEncoder(w).Encode(resp)
	}))
	defer server.Close()

	c := NewClient(server.URL, WithRetries(0))
	result, err := c.Call(context.Background(), "eth_getBalance", "0xabc", "latest")
	require.NoError(t, err)
	assert.Contains(t, string(result), "balance")
}

func TestClient_Call_NilParams(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		body, _ := io.ReadAll(r.Body)
		var req Request
		json.Unmarshal(body, &req)

		paramsJSON, _ := json.Marshal(req.Params)
		assert.Equal(t, "[]", string(paramsJSON))

		resp := Response{JSONRPC: "2.0", Result: json.RawMessage(`"ok"`), ID: req.ID}
		json.NewEncoder(w).Encode(resp)
	}))
	defer server.Close()

	c := NewClient(server.URL, WithRetries(0))
	result, err := c.Call(context.Background(), "eth_test")
	require.NoError(t, err)
	assert.Equal(t, `"ok"`, string(result))
}

func TestClient_Call_RPCError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		body, _ := io.ReadAll(r.Body)
		var req Request
		json.Unmarshal(body, &req)

		resp := Response{
			JSONRPC: "2.0",
			Error:   &RPCError{Code: -32601, Message: "Method not found"},
			ID:      req.ID,
		}
		json.NewEncoder(w).Encode(resp)
	}))
	defer server.Close()

	c := NewClient(server.URL, WithRetries(0))
	result, err := c.Call(context.Background(), "invalid_method")
	assert.Nil(t, result)
	require.Error(t, err)

	var rpcErr *RPCError
	assert.ErrorAs(t, err, &rpcErr)
	assert.Equal(t, -32601, rpcErr.Code)
	assert.Equal(t, "Method not found", rpcErr.Message)
}

func TestClient_Call_IncrementingID(t *testing.T) {
	var ids []int64
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		body, _ := io.ReadAll(r.Body)
		var req Request
		json.Unmarshal(body, &req)
		ids = append(ids, req.ID)

		resp := Response{JSONRPC: "2.0", Result: json.RawMessage(`"ok"`), ID: req.ID}
		json.NewEncoder(w).Encode(resp)
	}))
	defer server.Close()

	c := NewClient(server.URL, WithRetries(0))
	c.Call(context.Background(), "method1")
	c.Call(context.Background(), "method2")
	c.Call(context.Background(), "method3")

	require.Len(t, ids, 3)
	assert.Less(t, ids[0], ids[1])
	assert.Less(t, ids[1], ids[2])
}

func TestClient_Call_Retry(t *testing.T) {
	var attempts atomic.Int32

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		n := attempts.Add(1)
		body, _ := io.ReadAll(r.Body)
		var req Request
		json.Unmarshal(body, &req)

		if n < 3 {
			w.WriteHeader(http.StatusInternalServerError)
			w.Write([]byte("internal error"))
			return
		}

		resp := Response{JSONRPC: "2.0", Result: json.RawMessage(`"success"`), ID: req.ID}
		json.NewEncoder(w).Encode(resp)
	}))
	defer server.Close()

	c := NewClient(server.URL, WithRetries(3))
	result, err := c.Call(context.Background(), "retry_method")
	require.NoError(t, err)
	assert.Equal(t, `"success"`, string(result))
	assert.GreaterOrEqual(t, int(attempts.Load()), 3)
}

func TestClient_Call_AllRetriesFail(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
		w.Write([]byte("always fails"))
	}))
	defer server.Close()

	c := NewClient(server.URL, WithRetries(1))
	_, err := c.Call(context.Background(), "fail_method")
	require.Error(t, err)
}

func TestClient_Call_ContextCancelled(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		time.Sleep(5 * time.Second)
	}))
	defer server.Close()

	c := NewClient(server.URL, WithRetries(0))
	ctx, cancel := context.WithTimeout(context.Background(), 50*time.Millisecond)
	defer cancel()

	_, err := c.Call(ctx, "slow_method")
	require.Error(t, err)
}

func TestClient_Call_InvalidURL(t *testing.T) {
	c := NewClient("http://127.0.0.1:1", WithRetries(0))
	_, err := c.Call(context.Background(), "test")
	require.Error(t, err)
}

func TestClient_Call_InvalidJSONResponse(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte("not json"))
	}))
	defer server.Close()

	c := NewClient(server.URL, WithRetries(0))
	_, err := c.Call(context.Background(), "test")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "unmarshal")
}

func TestClient_CallResult_Success(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		body, _ := io.ReadAll(r.Body)
		var req Request
		json.Unmarshal(body, &req)

		resp := Response{
			JSONRPC: "2.0",
			Result:  json.RawMessage(`"0x10d"`),
			ID:      req.ID,
		}
		json.NewEncoder(w).Encode(resp)
	}))
	defer server.Close()

	c := NewClient(server.URL, WithRetries(0))
	var result string
	err := c.CallResult(context.Background(), &result, "eth_blockNumber")
	require.NoError(t, err)
	assert.Equal(t, "0x10d", result)
}

func TestClient_CallResult_StructResult(t *testing.T) {
	type Block struct {
		Number string `json:"number"`
		Hash   string `json:"hash"`
	}

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		body, _ := io.ReadAll(r.Body)
		var req Request
		json.Unmarshal(body, &req)

		result := Block{Number: "0x10", Hash: "0xabc"}
		resultJSON, _ := json.Marshal(result)
		resp := Response{JSONRPC: "2.0", Result: json.RawMessage(resultJSON), ID: req.ID}
		json.NewEncoder(w).Encode(resp)
	}))
	defer server.Close()

	c := NewClient(server.URL, WithRetries(0))
	var block Block
	err := c.CallResult(context.Background(), &block, "eth_getBlockByNumber", "0x10", true)
	require.NoError(t, err)
	assert.Equal(t, "0x10", block.Number)
	assert.Equal(t, "0xabc", block.Hash)
}

func TestClient_CallResult_RPCError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		body, _ := io.ReadAll(r.Body)
		var req Request
		json.Unmarshal(body, &req)

		resp := Response{
			JSONRPC: "2.0",
			Error:   &RPCError{Code: -32600, Message: "Invalid request"},
			ID:      req.ID,
		}
		json.NewEncoder(w).Encode(resp)
	}))
	defer server.Close()

	c := NewClient(server.URL, WithRetries(0))
	var result string
	err := c.CallResult(context.Background(), &result, "bad_method")
	require.Error(t, err)
}

func TestClient_CallResult_UnmarshalError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		body, _ := io.ReadAll(r.Body)
		var req Request
		json.Unmarshal(body, &req)

		resp := Response{JSONRPC: "2.0", Result: json.RawMessage(`"not_a_number"`), ID: req.ID}
		json.NewEncoder(w).Encode(resp)
	}))
	defer server.Close()

	c := NewClient(server.URL, WithRetries(0))
	var result int
	err := c.CallResult(context.Background(), &result, "test")
	require.Error(t, err)
}

func TestClient_BatchCall_Success(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		body, _ := io.ReadAll(r.Body)

		var requests []Request
		require.NoError(t, json.Unmarshal(body, &requests))
		assert.Len(t, requests, 2)

		responses := make([]Response, len(requests))
		for i, req := range requests {
			assert.Equal(t, "2.0", req.JSONRPC)
			responses[i] = Response{
				JSONRPC: "2.0",
				Result:  json.RawMessage(`"result_` + req.Method + `"`),
				ID:      req.ID,
			}
		}
		json.NewEncoder(w).Encode(responses)
	}))
	defer server.Close()

	c := NewClient(server.URL, WithRetries(0))
	calls := []BatchRequest{
		{Method: "eth_blockNumber", Params: nil},
		{Method: "eth_chainId", Params: nil},
	}

	responses, err := c.BatchCall(context.Background(), calls)
	require.NoError(t, err)
	require.Len(t, responses, 2)
	assert.Equal(t, `"result_eth_blockNumber"`, string(responses[0].Result))
	assert.Equal(t, `"result_eth_chainId"`, string(responses[1].Result))
}

func TestClient_BatchCall_EmptyBatch(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		body, _ := io.ReadAll(r.Body)
		var requests []Request
		json.Unmarshal(body, &requests)
		assert.Empty(t, requests)

		json.NewEncoder(w).Encode([]Response{})
	}))
	defer server.Close()

	c := NewClient(server.URL, WithRetries(0))
	responses, err := c.BatchCall(context.Background(), []BatchRequest{})
	require.NoError(t, err)
	assert.Empty(t, responses)
}

func TestClient_BatchCall_WithErrors(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		body, _ := io.ReadAll(r.Body)
		var requests []Request
		json.Unmarshal(body, &requests)

		responses := []Response{
			{JSONRPC: "2.0", Result: json.RawMessage(`"0x1"`), ID: requests[0].ID},
			{JSONRPC: "2.0", Error: &RPCError{Code: -32601, Message: "Method not found"}, ID: requests[1].ID},
		}
		json.NewEncoder(w).Encode(responses)
	}))
	defer server.Close()

	c := NewClient(server.URL, WithRetries(0))
	calls := []BatchRequest{
		{Method: "eth_blockNumber"},
		{Method: "invalid_method"},
	}

	responses, err := c.BatchCall(context.Background(), calls)
	require.NoError(t, err) // BatchCall itself should not error.
	require.Len(t, responses, 2)
	assert.Nil(t, responses[0].Error)
	require.NotNil(t, responses[1].Error)
	assert.Equal(t, -32601, responses[1].Error.Code)
}

func TestClient_BatchCall_ServerError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
		w.Write([]byte("server error"))
	}))
	defer server.Close()

	c := NewClient(server.URL, WithRetries(0))
	calls := []BatchRequest{{Method: "test"}}

	_, err := c.BatchCall(context.Background(), calls)
	require.Error(t, err)
}

func TestClient_BatchCall_InvalidJSON(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte("not valid json"))
	}))
	defer server.Close()

	c := NewClient(server.URL, WithRetries(0))
	calls := []BatchRequest{{Method: "test"}}

	_, err := c.BatchCall(context.Background(), calls)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "unmarshal")
}

func TestClient_BatchCall_ContextCancelled(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		time.Sleep(5 * time.Second)
	}))
	defer server.Close()

	c := NewClient(server.URL, WithRetries(0))
	ctx, cancel := context.WithTimeout(context.Background(), 50*time.Millisecond)
	defer cancel()

	calls := []BatchRequest{{Method: "slow_method"}}
	_, err := c.BatchCall(ctx, calls)
	require.Error(t, err)
}

func TestClient_BatchCall_NilParams(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		body, _ := io.ReadAll(r.Body)
		var requests []Request
		json.Unmarshal(body, &requests)

		for _, req := range requests {
			paramsJSON, _ := json.Marshal(req.Params)
			assert.Equal(t, "[]", string(paramsJSON))
		}

		responses := make([]Response, len(requests))
		for i, req := range requests {
			responses[i] = Response{JSONRPC: "2.0", Result: json.RawMessage(`"ok"`), ID: req.ID}
		}
		json.NewEncoder(w).Encode(responses)
	}))
	defer server.Close()

	c := NewClient(server.URL, WithRetries(0))
	calls := []BatchRequest{
		{Method: "test1", Params: nil},
		{Method: "test2", Params: nil},
	}
	responses, err := c.BatchCall(context.Background(), calls)
	require.NoError(t, err)
	assert.Len(t, responses, 2)
}

func TestRequest_JSONSerialization(t *testing.T) {
	req := Request{
		JSONRPC: "2.0",
		Method:  "eth_call",
		Params:  []any{"0x1", "latest"},
		ID:      42,
	}

	data, err := json.Marshal(req)
	require.NoError(t, err)

	var decoded Request
	require.NoError(t, json.Unmarshal(data, &decoded))
	assert.Equal(t, "2.0", decoded.JSONRPC)
	assert.Equal(t, "eth_call", decoded.Method)
	assert.Equal(t, int64(42), decoded.ID)
}

func TestResponse_JSONSerialization(t *testing.T) {
	resp := Response{
		JSONRPC: "2.0",
		Result:  json.RawMessage(`"0xdeadbeef"`),
		ID:      1,
	}

	data, err := json.Marshal(resp)
	require.NoError(t, err)

	var decoded Response
	require.NoError(t, json.Unmarshal(data, &decoded))
	assert.Equal(t, "2.0", decoded.JSONRPC)
	assert.Equal(t, int64(1), decoded.ID)
	assert.Nil(t, decoded.Error)
	assert.Equal(t, `"0xdeadbeef"`, string(decoded.Result))
}

func TestResponse_JSONSerialization_WithError(t *testing.T) {
	resp := Response{
		JSONRPC: "2.0",
		Error:   &RPCError{Code: -32000, Message: "execution error", Data: "revert"},
		ID:      2,
	}

	data, err := json.Marshal(resp)
	require.NoError(t, err)

	var decoded Response
	require.NoError(t, json.Unmarshal(data, &decoded))
	assert.Equal(t, int64(2), decoded.ID)
	require.NotNil(t, decoded.Error)
	assert.Equal(t, -32000, decoded.Error.Code)
	assert.Equal(t, "revert", decoded.Error.Data)
}

func TestBatchRequest_Struct(t *testing.T) {
	br := BatchRequest{
		Method: "eth_call",
		Params: []any{"0x1"},
	}
	assert.Equal(t, "eth_call", br.Method)
}

func TestWithRetries_SetsRetries(t *testing.T) {
	c := &Client{}
	WithRetries(7)(c)
	assert.Equal(t, 7, c.retries)
}

func TestWithTimeout_SetsTimeout(t *testing.T) {
	c := &Client{}
	WithTimeout(15 * time.Second)(c)
	assert.Equal(t, 15*time.Second, c.timeout)
}

func TestWithHTTPClient_SetsHTTPClient(t *testing.T) {
	hc := &http.Client{}
	c := &Client{}
	WithHTTPClient(hc)(c)
	assert.Same(t, hc, c.httpClient)
}
