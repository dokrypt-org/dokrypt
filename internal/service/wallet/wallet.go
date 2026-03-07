package wallet

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"math/big"
	"net/http"
	"sync"
	"time"
)

type WalletServer struct {
	chainRPCURL string
	mu          sync.Mutex
	cooldowns   map[string]time.Time // address -> last funded time
	cooldown    time.Duration
}

func NewWalletServer(chainRPCURL string) *WalletServer {
	return &WalletServer{
		chainRPCURL: chainRPCURL,
		cooldowns:   make(map[string]time.Time),
		cooldown:    10 * time.Second,
	}
}

func (w *WalletServer) Handler() http.Handler {
	mux := http.NewServeMux()
	mux.HandleFunc("GET /health", w.handleHealth)
	mux.HandleFunc("POST /fund", w.handleFund)
	mux.HandleFunc("GET /balance/{address}", w.handleBalance)
	return mux
}

type jsonRPCRequest struct {
	JSONRPC string        `json:"jsonrpc"`
	ID      int           `json:"id"`
	Method  string        `json:"method"`
	Params  []interface{} `json:"params"`
}

type jsonRPCResponse struct {
	JSONRPC string          `json:"jsonrpc"`
	ID      int             `json:"id"`
	Result  json.RawMessage `json:"result,omitempty"`
	Error   *jsonRPCError   `json:"error,omitempty"`
}

type jsonRPCError struct {
	Code    int    `json:"code"`
	Message string `json:"message"`
}

func (w *WalletServer) rpcCall(method string, params []interface{}) (*jsonRPCResponse, error) {
	reqBody := jsonRPCRequest{
		JSONRPC: "2.0",
		ID:      1,
		Method:  method,
		Params:  params,
	}
	body, err := json.Marshal(reqBody)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal RPC request: %w", err)
	}

	client := &http.Client{Timeout: 10 * time.Second}
	resp, err := client.Post(w.chainRPCURL, "application/json", bytes.NewReader(body))
	if err != nil {
		return nil, fmt.Errorf("RPC call to %s failed: %w", w.chainRPCURL, err)
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read RPC response: %w", err)
	}

	var rpcResp jsonRPCResponse
	if err := json.Unmarshal(respBody, &rpcResp); err != nil {
		return nil, fmt.Errorf("failed to parse RPC response: %w", err)
	}
	return &rpcResp, nil
}

func (w *WalletServer) handleHealth(rw http.ResponseWriter, _ *http.Request) {
	rw.Header().Set("Content-Type", "application/json")
	rw.WriteHeader(http.StatusOK)
	json.NewEncoder(rw).Encode(map[string]string{"status": "ok", "chain_rpc": w.chainRPCURL})
}

type fundRequest struct {
	Address string `json:"address"`
	Amount  string `json:"amount"` // ETH amount as string
}

func (w *WalletServer) handleFund(rw http.ResponseWriter, r *http.Request) {
	var req fundRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(rw, `{"error":"invalid request"}`, http.StatusBadRequest)
		return
	}

	if req.Address == "" {
		http.Error(rw, `{"error":"address required"}`, http.StatusBadRequest)
		return
	}

	w.mu.Lock()
	lastFunded, exists := w.cooldowns[req.Address]
	if exists && time.Since(lastFunded) < w.cooldown {
		w.mu.Unlock()
		remaining := w.cooldown - time.Since(lastFunded)
		http.Error(rw, fmt.Sprintf(`{"error":"rate limited","retry_after_seconds":%d}`, int(remaining.Seconds())), http.StatusTooManyRequests)
		return
	}
	w.cooldowns[req.Address] = time.Now()
	w.mu.Unlock()

	amount := req.Amount
	if amount == "" {
		amount = "1" // default 1 ETH
	}

	slog.Info("funding address", "address", req.Address, "amount", amount)

	ethFloat, ok := new(big.Float).SetString(amount)
	if !ok {
		http.Error(rw, `{"error":"invalid amount"}`, http.StatusBadRequest)
		return
	}
	weiPerEth := new(big.Float).SetFloat64(1e18)
	weiFloat := new(big.Float).Mul(ethFloat, weiPerEth)
	weiInt, _ := weiFloat.Int(nil)
	weiHex := fmt.Sprintf("0x%x", weiInt)

	accountsResp, err := w.rpcCall("eth_accounts", []interface{}{})
	if err != nil {
		slog.Error("failed to get accounts from chain", "error", err)
		http.Error(rw, `{"error":"failed to get accounts from chain"}`, http.StatusInternalServerError)
		return
	}
	if accountsResp.Error != nil {
		slog.Error("RPC error getting accounts", "error", accountsResp.Error.Message)
		http.Error(rw, fmt.Sprintf(`{"error":"%s"}`, accountsResp.Error.Message), http.StatusInternalServerError)
		return
	}

	var accounts []string
	if err := json.Unmarshal(accountsResp.Result, &accounts); err != nil || len(accounts) == 0 {
		slog.Error("no accounts available on chain node")
		http.Error(rw, `{"error":"no accounts available on chain node"}`, http.StatusInternalServerError)
		return
	}
	from := accounts[0]

	txParams := map[string]string{
		"from":  from,
		"to":    req.Address,
		"value": weiHex,
	}
	txResp, err := w.rpcCall("eth_sendTransaction", []interface{}{txParams})
	if err != nil {
		slog.Error("failed to send transaction", "error", err)
		http.Error(rw, `{"error":"failed to send transaction"}`, http.StatusInternalServerError)
		return
	}
	if txResp.Error != nil {
		slog.Error("RPC error sending transaction", "error", txResp.Error.Message)
		http.Error(rw, fmt.Sprintf(`{"error":"%s"}`, txResp.Error.Message), http.StatusInternalServerError)
		return
	}

	var txHash string
	if err := json.Unmarshal(txResp.Result, &txHash); err != nil {
		txHash = string(txResp.Result)
	}

	rw.Header().Set("Content-Type", "application/json")
	json.NewEncoder(rw).Encode(map[string]string{
		"status":  "funded",
		"address": req.Address,
		"amount":  amount + " ETH",
		"wei":     weiInt.String(),
		"tx_hash": txHash,
	})
}

func (w *WalletServer) handleBalance(rw http.ResponseWriter, r *http.Request) {
	address := r.PathValue("address")
	if address == "" {
		http.Error(rw, `{"error":"address required"}`, http.StatusBadRequest)
		return
	}

	resp, err := w.rpcCall("eth_getBalance", []interface{}{address, "latest"})
	if err != nil {
		slog.Error("failed to get balance from chain", "address", address, "error", err)
		http.Error(rw, `{"error":"failed to get balance from chain"}`, http.StatusInternalServerError)
		return
	}
	if resp.Error != nil {
		slog.Error("RPC error getting balance", "error", resp.Error.Message)
		http.Error(rw, fmt.Sprintf(`{"error":"%s"}`, resp.Error.Message), http.StatusInternalServerError)
		return
	}

	var hexBalance string
	if err := json.Unmarshal(resp.Result, &hexBalance); err != nil {
		slog.Error("failed to parse balance response", "error", err)
		http.Error(rw, `{"error":"failed to parse balance"}`, http.StatusInternalServerError)
		return
	}

	balance := new(big.Int)
	if len(hexBalance) > 2 && hexBalance[:2] == "0x" {
		balance.SetString(hexBalance[2:], 16)
	} else {
		balance.SetString(hexBalance, 16)
	}

	rw.Header().Set("Content-Type", "application/json")
	json.NewEncoder(rw).Encode(map[string]string{
		"address": address,
		"balance": balance.String(),
	})
}
