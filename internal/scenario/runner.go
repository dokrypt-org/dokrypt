package scenario

import (
	"encoding/json"
	"fmt"
	"math/big"
	"net/http"
	"strings"
	"time"
)

type Scenario struct {
	Name        string
	Description string
	Flags       []Flag
	Run         func(rpcURL string, opts map[string]string) error
}

type Flag struct {
	Name         string
	Description  string
	DefaultValue string
}

type Registry struct {
	scenarios map[string]*Scenario
	order     []string
}

func NewRegistry() *Registry {
	r := &Registry{
		scenarios: make(map[string]*Scenario),
	}
	r.registerAll()
	return r
}

func (r *Registry) Register(s *Scenario) {
	r.scenarios[s.Name] = s
	r.order = append(r.order, s.Name)
}

func (r *Registry) Get(name string) (*Scenario, error) {
	s, ok := r.scenarios[name]
	if !ok {
		return nil, fmt.Errorf("scenario %q not found. Run 'dokrypt scenario list' to see available scenarios", name)
	}
	return s, nil
}

func (r *Registry) List() []*Scenario {
	result := make([]*Scenario, 0, len(r.order))
	for _, name := range r.order {
		result = append(result, r.scenarios[name])
	}
	return result
}

func rpcCall(url, method string, params ...any) (json.RawMessage, error) {
	if params == nil {
		params = []any{}
	}
	reqBody := map[string]any{
		"jsonrpc": "2.0",
		"method":  method,
		"params":  params,
		"id":      1,
	}
	data, _ := json.Marshal(reqBody)

	client := &http.Client{Timeout: 10 * time.Second}
	resp, err := client.Post(url, "application/json", strings.NewReader(string(data)))
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	var result struct {
		Result json.RawMessage `json:"result"`
		Error  *struct {
			Code    int    `json:"code"`
			Message string `json:"message"`
		} `json:"error"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, err
	}
	if result.Error != nil {
		return nil, fmt.Errorf("RPC error %d: %s", result.Error.Code, result.Error.Message)
	}
	return result.Result, nil
}

func mine(rpcURL string, blocks uint64) error {
	hexCount := fmt.Sprintf("0x%x", blocks)
	if _, err := rpcCall(rpcURL, "anvil_mine", hexCount); err != nil {
		for i := uint64(0); i < blocks; i++ {
			if _, err := rpcCall(rpcURL, "evm_mine"); err != nil {
				return fmt.Errorf("failed to mine: %w", err)
			}
		}
	}
	return nil
}

func timeTravel(rpcURL string, seconds int64) error {
	hexSeconds := fmt.Sprintf("0x%x", seconds)
	if _, err := rpcCall(rpcURL, "evm_increaseTime", hexSeconds); err != nil {
		return fmt.Errorf("failed to advance time: %w", err)
	}
	_, _ = rpcCall(rpcURL, "evm_mine")
	return nil
}

func setGasPrice(rpcURL string, gwei uint64) error {
	weiPrice := gwei * 1e9
	hexPrice := fmt.Sprintf("0x%x", weiPrice)
	if _, err := rpcCall(rpcURL, "anvil_setMinGasPrice", hexPrice); err != nil {
		if _, err2 := rpcCall(rpcURL, "hardhat_setNextBlockBaseFeePerGas", hexPrice); err2 != nil {
			return fmt.Errorf("failed to set gas price: %w", err)
		}
	}
	return nil
}

func setBalance(rpcURL string, address string, ethAmount float64) error {
	ethFloat := new(big.Float).SetFloat64(ethAmount)
	weiFloat := new(big.Float).Mul(ethFloat, new(big.Float).SetFloat64(1e18))
	weiInt, _ := weiFloat.Int(nil)
	hexWei := fmt.Sprintf("0x%x", weiInt)

	if _, err := rpcCall(rpcURL, "anvil_setBalance", address, hexWei); err != nil {
		if _, err2 := rpcCall(rpcURL, "hardhat_setBalance", address, hexWei); err2 != nil {
			return fmt.Errorf("failed to set balance: %w", err)
		}
	}
	return nil
}

func impersonate(rpcURL string, address string) error {
	if _, err := rpcCall(rpcURL, "anvil_impersonateAccount", address); err != nil {
		if _, err2 := rpcCall(rpcURL, "hardhat_impersonateAccount", address); err2 != nil {
			return fmt.Errorf("failed to impersonate: %w", err)
		}
	}
	return nil
}

func stopImpersonating(rpcURL string, address string) error {
	if _, err := rpcCall(rpcURL, "anvil_stopImpersonatingAccount", address); err != nil {
		if _, err2 := rpcCall(rpcURL, "hardhat_stopImpersonatingAccount", address); err2 != nil {
			return fmt.Errorf("failed to stop impersonating: %w", err)
		}
	}
	return nil
}

func snapshot(rpcURL string) (string, error) {
	result, err := rpcCall(rpcURL, "evm_snapshot")
	if err != nil {
		return "", fmt.Errorf("failed to take snapshot: %w", err)
	}
	var snapID string
	json.Unmarshal(result, &snapID)
	return snapID, nil
}

func revertSnapshot(rpcURL string, snapID string) error {
	result, err := rpcCall(rpcURL, "evm_revert", snapID)
	if err != nil {
		return fmt.Errorf("failed to revert snapshot: %w", err)
	}
	var success bool
	json.Unmarshal(result, &success)
	if !success {
		return fmt.Errorf("snapshot revert returned false — snapshot may have already been consumed")
	}
	return nil
}
