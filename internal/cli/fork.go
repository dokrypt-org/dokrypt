package cli

import (
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

func rpcCallLongTimeout(url, method string, params ...any) (json.RawMessage, error) {
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

	client := &http.Client{Timeout: 120 * time.Second}
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

var knownNetworks = map[string]string{
	"mainnet":  "https://eth.llamarpc.com",
	"ethereum": "https://eth.llamarpc.com",
	"sepolia":  "https://rpc.sepolia.org",
	"goerli":   "https://rpc.ankr.com/eth_goerli",
	"polygon":  "https://polygon-rpc.com",
	"arbitrum": "https://arb1.arbitrum.io/rpc",
	"optimism": "https://mainnet.optimism.io",
	"base":     "https://mainnet.base.org",
	"bsc":      "https://bsc-dataseed.binance.org",
	"avalanche": "https://api.avax.network/ext/bc/C/rpc",
}

func newForkCmd() *cobra.Command {
	var (
		url       string
		block     uint64
		chainName string
		accounts  int
	)

	cmd := &cobra.Command{
		Use:   "fork [network]",
		Short: "Fork a live network",
		Long: `Fork a live blockchain network. Supported networks:
  mainnet, sepolia, polygon, arbitrum, optimism, base, bsc, avalanche
Or provide a custom RPC URL with --url.`,
		Args: cobra.MaximumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			out := getOutput()

			network := "mainnet"
			if len(args) > 0 {
				network = args[0]
			}

			forkURL := url
			if forkURL == "" {
				if resolved, ok := knownNetworks[strings.ToLower(network)]; ok {
					forkURL = resolved
				} else {
					if strings.HasPrefix(network, "http") {
						forkURL = network
					} else {
						return fmt.Errorf("Unknown network %q. Use --url for custom RPC endpoints.\nSupported: mainnet, sepolia, polygon, arbitrum, optimism, base, bsc, avalanche", network)
					}
				}
			}

			rpcURL, err := getChainRPC(chainName)
			if err != nil {
				return err
			}

			out.Info("Forking %s...", network)
			params := map[string]any{
				"forking": map[string]any{
					"jsonRpcUrl": forkURL,
				},
			}
			if block > 0 {
				params["forking"].(map[string]any)["blockNumber"] = fmt.Sprintf("0x%x", block)
				out.Info("  at block #%d", block)
			}

			if _, err := rpcCallLongTimeout(rpcURL, "anvil_reset", params); err != nil {
				if _, err2 := rpcCallLongTimeout(rpcURL, "hardhat_reset", params); err2 != nil {
					return fmt.Errorf("failed to fork: %w\nMake sure the RPC URL is accessible: %s", err, forkURL)
				}
			}

			blockNum, _ := getCurrentBlock(rpcURL)
			chainID, _ := getChainID(rpcURL)

			fmt.Println()
			out.Success("Forked %s successfully!", network)
			out.Info("  Chain ID:  %d", chainID)
			out.Info("  Block:     #%d", blockNum)
			out.Info("  Fork URL:  %s", forkURL)

			if accounts > 0 {
				out.Info("  Accounts:  %d (funded with 10000 ETH each)", accounts)

				result, err := rpcCall(rpcURL, "eth_accounts")
				if err == nil {
					var addrs []string
					json.Unmarshal(result, &addrs)
					weiAmount := new(big.Int).Mul(big.NewInt(10000), new(big.Int).Exp(big.NewInt(10), big.NewInt(18), nil))
					hexWei := fmt.Sprintf("0x%x", weiAmount)
					for i, addr := range addrs {
						if i >= accounts {
							break
						}
						rpcCall(rpcURL, "anvil_setBalance", addr, hexWei)
					}
				}
			}

			fmt.Println()
			return nil
		},
	}

	cmd.Flags().StringVar(&url, "url", "", "custom RPC URL to fork from")
	cmd.Flags().Uint64Var(&block, "block", 0, "fork at specific block number")
	cmd.Flags().StringVar(&chainName, "chain", "", "target chain (for multi-chain setups)")
	cmd.Flags().IntVar(&accounts, "accounts", 10, "number of funded accounts")

	return cmd
}

func newAccountsCmd() *cobra.Command {
	var chainName string

	cmd := &cobra.Command{
		Use:   "accounts",
		Short: "Account management",
	}

	cmd.PersistentFlags().StringVar(&chainName, "chain", "", "target chain")

	cmd.AddCommand(
		newAccountsListCmd(&chainName),
		newAccountsFundCmd(&chainName),
		newAccountsImpersonateCmd(&chainName),
		newAccountsGenerateCmd(&chainName),
	)

	return cmd
}

func newAccountsListCmd(chainName *string) *cobra.Command {
	return &cobra.Command{
		Use:   "list",
		Short: "List accounts with balances",
		RunE: func(cmd *cobra.Command, args []string) error {
			out := getOutput()

			rpcURL, err := getChainRPC(*chainName)
			if err != nil {
				return err
			}

			result, err := rpcCall(rpcURL, "eth_accounts")
			if err != nil {
				return fmt.Errorf("failed to get accounts: %w", err)
			}

			var accounts []string
			json.Unmarshal(result, &accounts)

			if len(accounts) == 0 {
				out.Info("No accounts found.")
				return nil
			}

			fmt.Println()
			headers := []string{"#", "Address", "Balance (ETH)"}
			var rows [][]string

			for i, addr := range accounts {
				balResult, err := rpcCall(rpcURL, "eth_getBalance", addr, "latest")
				balETH := "?"
				if err == nil {
					var hexBal string
					json.Unmarshal(balResult, &hexBal)
					hexBal = strings.TrimPrefix(hexBal, "0x")
					wei, ok := new(big.Int).SetString(hexBal, 16)
					if ok {
						ethFloat := new(big.Float).Quo(
							new(big.Float).SetInt(wei),
							new(big.Float).SetFloat64(1e18),
						)
						balETH = ethFloat.Text('f', 4)
					}
				}

				rows = append(rows, []string{
					fmt.Sprintf("[%d]", i),
					addr,
					balETH,
				})
			}

			out.Table(headers, rows)
			fmt.Println()
			return nil
		},
	}
}

func newAccountsFundCmd(chainName *string) *cobra.Command {
	return &cobra.Command{
		Use:   "fund <address> <amount-eth>",
		Short: "Fund an account with ETH",
		Args:  requireArgs(2, "dokrypt accounts fund <address> <amount-eth>"),
		RunE: func(cmd *cobra.Command, args []string) error {
			out := getOutput()

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
					return fmt.Errorf("failed to fund account: %w", err)
				}
			}

			out.Success("Funded %s with %s ETH", address, amountStr)
			return nil
		},
	}
}

func newAccountsImpersonateCmd(chainName *string) *cobra.Command {
	return &cobra.Command{
		Use:   "impersonate <address>",
		Short: "Impersonate an address (for forked chains)",
		Args:  requireArgs(1, "dokrypt accounts impersonate <address>"),
		RunE: func(cmd *cobra.Command, args []string) error {
			out := getOutput()

			rpcURL, err := getChainRPC(*chainName)
			if err != nil {
				return err
			}

			if _, err := rpcCall(rpcURL, "anvil_impersonateAccount", args[0]); err != nil {
				if _, err2 := rpcCall(rpcURL, "hardhat_impersonateAccount", args[0]); err2 != nil {
					return fmt.Errorf("failed to impersonate: %w", err)
				}
			}

			out.Success("Now impersonating %s", args[0])
			out.Info("Transactions from this address will succeed without a private key.")
			return nil
		},
	}
}

func newAccountsGenerateCmd(chainName *string) *cobra.Command {
	return &cobra.Command{
		Use:   "generate <count>",
		Short: "Generate and fund new accounts",
		Args:  requireArgs(1, "dokrypt accounts generate <count>"),
		RunE: func(cmd *cobra.Command, args []string) error {
			out := getOutput()

			count, err := strconv.Atoi(args[0])
			if err != nil || count < 1 {
				return fmt.Errorf("invalid count: %s", args[0])
			}

			rpcURL, err := getChainRPC(*chainName)
			if err != nil {
				return err
			}

			out.Info("Generating %d accounts...", count)

			cfg, _ := config.Parse(getConfigPath())
			balance := "10000000000000000000000" // 10000 ETH default
			if cfg != nil {
				for _, chain := range cfg.Chains {
					if chain.Balance != "" {
						balance = chain.Balance
						break
					}
				}
			}

			wei, _ := new(big.Int).SetString(balance, 10)
			hexWei := fmt.Sprintf("0x%x", wei)

			for i := 0; i < count; i++ {
				addr := fmt.Sprintf("0x%040x", i+1000)
				rpcCall(rpcURL, "anvil_setBalance", addr, hexWei)
				fmt.Printf("  [%d] %s (10000 ETH)\n", i, addr)
			}

			fmt.Println()
			out.Success("Generated %d accounts", count)
			return nil
		},
	}
}
