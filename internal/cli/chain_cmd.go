package cli

import (
	"context"
	"encoding/json"
	"fmt"
	"math/big"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/spf13/cobra"

	"github.com/dokrypt/dokrypt/internal/config"
)

func newChainCmd() *cobra.Command {
	var chainName string

	cmd := &cobra.Command{
		Use:   "chain",
		Short: "Chain management",
	}

	cmd.PersistentFlags().StringVar(&chainName, "chain", "", "target chain (for multi-chain setups)")

	cmd.AddCommand(
		newChainMineCmd(&chainName),
		newChainSetBalanceCmd(&chainName),
		newChainTimeTravelCmd(&chainName),
		newChainSetGasPriceCmd(&chainName),
		newChainImpersonateCmd(&chainName),
		newChainStopImpersonatingCmd(&chainName),
		newChainResetCmd(&chainName),
		newChainInfoCmd(&chainName),
	)

	return cmd
}

func newChainMineCmd(chainName *string) *cobra.Command {
	return &cobra.Command{
		Use: "mine [n]", Short: "Mine N blocks (default 1)", Args: cobra.MaximumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			rpcURL, err := getChainRPC(*chainName)
			if err != nil {
				return err
			}

			count := uint64(1)
			if len(args) > 0 {
				n, err := strconv.ParseUint(args[0], 10, 64)
				if err != nil {
					return fmt.Errorf("invalid block count: %s", args[0])
				}
				count = n
			}

			hexCount := fmt.Sprintf("0x%x", count)
			if _, err := rpcCall(rpcURL, "anvil_mine", hexCount); err != nil {
				for i := uint64(0); i < count; i++ {
					if _, err := rpcCall(rpcURL, "evm_mine"); err != nil {
						return fmt.Errorf("failed to mine: %w", err)
					}
				}
			}

			blockNum, err := getCurrentBlock(rpcURL)
			if err != nil {
				fmt.Printf("Mined %d blocks.\n", count)
			} else {
				fmt.Printf("Mined %d blocks. Current block: #%d\n", count, blockNum)
			}
			return nil
		},
	}
}

func newChainSetBalanceCmd(chainName *string) *cobra.Command {
	return &cobra.Command{
		Use: "set-balance <address> <amount-eth>", Short: "Set account balance (in ETH)", Args: requireArgs(2, "dokrypt chain set-balance <address> <amount-eth>"),
		RunE: func(cmd *cobra.Command, args []string) error {
			rpcURL, err := getChainRPC(*chainName)
			if err != nil {
				return err
			}

			address := args[0]
			amountStr := args[1]

			ethFloat, ok := new(big.Float).SetString(amountStr)
			if !ok {
				return fmt.Errorf("invalid amount: %s", amountStr)
			}
			weiFloat := new(big.Float).Mul(ethFloat, new(big.Float).SetFloat64(1e18))
			weiInt, _ := weiFloat.Int(nil)
			hexWei := fmt.Sprintf("0x%x", weiInt)

			if _, err := rpcCall(rpcURL, "anvil_setBalance", address, hexWei); err != nil {
				if _, err2 := rpcCall(rpcURL, "hardhat_setBalance", address, hexWei); err2 != nil {
					return fmt.Errorf("failed to set balance: %w", err)
				}
			}

			fmt.Printf("Balance of %s set to %s ETH\n", address, amountStr)
			return nil
		},
	}
}

func newChainTimeTravelCmd(chainName *string) *cobra.Command {
	return &cobra.Command{
		Use: "time-travel <duration>", Short: "Advance chain time (e.g. 1h, 7d, 3600)", Args: requireArgs(1, "dokrypt chain time-travel <duration>"),
		RunE: func(cmd *cobra.Command, args []string) error {
			rpcURL, err := getChainRPC(*chainName)
			if err != nil {
				return err
			}

			seconds, err := parseDurationStr(args[0])
			if err != nil {
				return err
			}

			hexSeconds := fmt.Sprintf("0x%x", seconds)
			if _, err := rpcCall(rpcURL, "evm_increaseTime", hexSeconds); err != nil {
				return fmt.Errorf("failed to advance time: %w", err)
			}
			rpcCall(rpcURL, "evm_mine")

			ts, err := getCurrentTimestamp(rpcURL)
			if err != nil {
				fmt.Printf("Advanced %s.\n", args[0])
			} else {
				t := time.Unix(ts, 0).UTC()
				fmt.Printf("Advanced %s. Current block time: %s\n", args[0], t.Format("2006-01-02 15:04:05 UTC"))
			}
			return nil
		},
	}
}

func newChainSetGasPriceCmd(chainName *string) *cobra.Command {
	return &cobra.Command{
		Use: "set-gas-price <gwei>", Short: "Set base gas price", Args: requireArgs(1, "dokrypt chain set-gas-price <gwei>"),
		RunE: func(cmd *cobra.Command, args []string) error {
			rpcURL, err := getChainRPC(*chainName)
			if err != nil {
				return err
			}

			gwei, err := strconv.ParseUint(args[0], 10, 64)
			if err != nil {
				return fmt.Errorf("invalid gwei value: %s", args[0])
			}

			weiPrice := gwei * 1e9
			hexPrice := fmt.Sprintf("0x%x", weiPrice)
			if _, err := rpcCall(rpcURL, "anvil_setMinGasPrice", hexPrice); err != nil {
				if _, err2 := rpcCall(rpcURL, "hardhat_setNextBlockBaseFeePerGas", hexPrice); err2 != nil {
					return fmt.Errorf("failed to set gas price: %w", err)
				}
			}

			fmt.Printf("Gas price set to %d gwei\n", gwei)
			return nil
		},
	}
}

func newChainImpersonateCmd(chainName *string) *cobra.Command {
	return &cobra.Command{
		Use: "impersonate <address>", Short: "Impersonate an account", Args: requireArgs(1, "dokrypt chain impersonate <address>"),
		RunE: func(cmd *cobra.Command, args []string) error {
			rpcURL, err := getChainRPC(*chainName)
			if err != nil {
				return err
			}

			if _, err := rpcCall(rpcURL, "anvil_impersonateAccount", args[0]); err != nil {
				if _, err2 := rpcCall(rpcURL, "hardhat_impersonateAccount", args[0]); err2 != nil {
					return fmt.Errorf("failed to impersonate: %w", err)
				}
			}

			fmt.Printf("Now impersonating %s — transactions from this address will succeed without a private key\n", args[0])
			return nil
		},
	}
}

func newChainStopImpersonatingCmd(chainName *string) *cobra.Command {
	return &cobra.Command{
		Use: "stop-impersonating <address>", Short: "Stop impersonating an account", Args: requireArgs(1, "dokrypt chain stop-impersonating <address>"),
		RunE: func(cmd *cobra.Command, args []string) error {
			rpcURL, err := getChainRPC(*chainName)
			if err != nil {
				return err
			}

			if _, err := rpcCall(rpcURL, "anvil_stopImpersonatingAccount", args[0]); err != nil {
				if _, err2 := rpcCall(rpcURL, "hardhat_stopImpersonatingAccount", args[0]); err2 != nil {
					return fmt.Errorf("failed to stop impersonating: %w", err)
				}
			}

			fmt.Printf("Stopped impersonating %s\n", args[0])
			return nil
		},
	}
}

func newChainResetCmd(chainName *string) *cobra.Command {
	var (
		forkURL   string
		forkBlock uint64
	)

	cmd := &cobra.Command{
		Use: "reset", Short: "Reset chain to genesis or fork",
		RunE: func(cmd *cobra.Command, args []string) error {
			rpcURL, err := getChainRPC(*chainName)
			if err != nil {
				return err
			}

			params := map[string]any{}
			if forkURL != "" {
				forking := map[string]any{"jsonRpcUrl": forkURL}
				if forkBlock > 0 {
					forking["blockNumber"] = fmt.Sprintf("0x%x", forkBlock)
				}
				params["forking"] = forking
			}

			if _, err := rpcCall(rpcURL, "anvil_reset", params); err != nil {
				if _, err2 := rpcCall(rpcURL, "hardhat_reset", params); err2 != nil {
					return fmt.Errorf("failed to reset: %w", err)
				}
			}

			if forkURL != "" {
				fmt.Printf("Chain reset. Forked from %s", forkURL)
				if forkBlock > 0 {
					fmt.Printf(" at block #%d", forkBlock)
				}
				fmt.Println()
			} else {
				fmt.Println("Chain reset to genesis. Block number: 0")
			}
			return nil
		},
	}

	cmd.Flags().StringVar(&forkURL, "fork", "", "fork URL")
	cmd.Flags().Uint64Var(&forkBlock, "block", 0, "fork at block number")
	return cmd
}

func newChainInfoCmd(chainName *string) *cobra.Command {
	return &cobra.Command{
		Use: "info", Short: "Show chain info",
		RunE: func(cmd *cobra.Command, args []string) error {
			rpcURL, err := getChainRPC(*chainName)
			if err != nil {
				return err
			}

			blockNum, _ := getCurrentBlock(rpcURL)
			ts, _ := getCurrentTimestamp(rpcURL)
			chainID, _ := getChainID(rpcURL)

			fmt.Println()
			fmt.Printf("  Chain ID:     %d\n", chainID)
			fmt.Printf("  RPC URL:      %s\n", rpcURL)
			fmt.Printf("  Block:        #%d\n", blockNum)
			if ts > 0 {
				t := time.Unix(ts, 0).UTC()
				fmt.Printf("  Block time:   %s\n", t.Format("2006-01-02 15:04:05 UTC"))
			}
			fmt.Println()
			return nil
		},
	}
}

func getChainRPC(chainName string) (string, error) {
	cfg, err := config.Parse(getConfigPath())
	if err != nil {
		return "", fmt.Errorf("No dokrypt.yaml found. Run 'dokrypt init' first.")
	}

	state, err := loadState(cfg.Name)
	if err != nil {
		return "", fmt.Errorf("No Dokrypt stack running. Run 'dokrypt up' first.")
	}

	if chainName == "" {
		for name := range cfg.Chains {
			chainName = name
			break
		}
	}

	cs, ok := state.Containers[chainName]
	if !ok {
		return "", fmt.Errorf("Chain '%s' not found in running stack.", chainName)
	}

	for _, port := range cs.Ports {
		if port > 0 {
			return fmt.Sprintf("http://localhost:%d", port), nil
		}
	}
	return "", fmt.Errorf("Chain '%s' has no exposed ports.", chainName)
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

func rpcCallCtx(ctx context.Context, url, method string, params ...any) (json.RawMessage, error) {
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

	req, err := http.NewRequestWithContext(ctx, "POST", url, strings.NewReader(string(data)))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", "application/json")

	client := &http.Client{Timeout: 10 * time.Second}
	resp, err := client.Do(req)
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

func getCurrentBlock(rpcURL string) (uint64, error) {
	result, err := rpcCall(rpcURL, "eth_blockNumber")
	if err != nil {
		return 0, err
	}
	var hexStr string
	json.Unmarshal(result, &hexStr)
	hexStr = strings.TrimPrefix(hexStr, "0x")
	n, err := strconv.ParseUint(hexStr, 16, 64)
	return n, err
}

func getCurrentTimestamp(rpcURL string) (int64, error) {
	result, err := rpcCall(rpcURL, "eth_getBlockByNumber", "latest", false)
	if err != nil {
		return 0, err
	}
	var block struct {
		Timestamp string `json:"timestamp"`
	}
	json.Unmarshal(result, &block)
	hexStr := strings.TrimPrefix(block.Timestamp, "0x")
	n, err := strconv.ParseInt(hexStr, 16, 64)
	return n, err
}

func getChainID(rpcURL string) (uint64, error) {
	result, err := rpcCall(rpcURL, "eth_chainId")
	if err != nil {
		return 0, err
	}
	var hexStr string
	json.Unmarshal(result, &hexStr)
	hexStr = strings.TrimPrefix(hexStr, "0x")
	n, err := strconv.ParseUint(hexStr, 16, 64)
	return n, err
}

func parseDurationStr(s string) (int64, error) {
	s = strings.TrimSpace(s)
	if s == "" {
		return 0, fmt.Errorf("empty duration")
	}

	if n, err := strconv.ParseInt(s, 10, 64); err == nil {
		return n, nil
	}

	if d, err := time.ParseDuration(s); err == nil {
		return int64(d.Seconds()), nil
	}

	last := s[len(s)-1]
	numStr := s[:len(s)-1]
	n, err := strconv.ParseInt(numStr, 10, 64)
	if err != nil {
		return 0, fmt.Errorf("invalid duration: %s", s)
	}

	switch last {
	case 's':
		return n, nil
	case 'm':
		return n * 60, nil
	case 'h':
		return n * 3600, nil
	case 'd':
		return n * 86400, nil
	default:
		return 0, fmt.Errorf("unknown duration suffix: %c", last)
	}
}
