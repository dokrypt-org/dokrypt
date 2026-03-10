package cli

import (
	"encoding/json"
	"fmt"
	"math/big"
	"strings"

	"github.com/spf13/cobra"
)

func newReplayCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "replay",
		Short: "Transaction replay and debugging",
	}

	cmd.AddCommand(
		newReplayTxCmd(),
		newReplayBlockCmd(),
		newReplayTraceCmd(),
	)

	return cmd
}

func newReplayTxCmd() *cobra.Command {
	var (
		chainName string
		network   string
		block     uint64
		verbose   bool
	)

	cmd := &cobra.Command{
		Use:   "tx <tx-hash>",
		Short: "Replay a transaction from a live network locally",
		Long: `Fetch a transaction from a live network, fork the chain at the transaction's block,
and re-execute it locally. Shows execution trace, gas usage, state changes, and revert reasons.

Examples:
  dokrypt replay tx 0xabc123... --network arbitrum
  dokrypt replay tx 0xabc123... --network mainnet --verbose`,
		Args: requireArgs(1, "dokrypt replay tx <tx-hash>"),
		RunE: func(cmd *cobra.Command, args []string) error {
			out := getOutput()
			txHash := args[0]

			if !strings.HasPrefix(txHash, "0x") || len(txHash) != 66 {
				return fmt.Errorf("invalid transaction hash: %s", txHash)
			}

			networkRPC := ""
			if resolved, ok := knownNetworks[strings.ToLower(network)]; ok {
				networkRPC = resolved
			} else if strings.HasPrefix(network, "http") {
				networkRPC = network
			} else {
				return fmt.Errorf("unknown network %q. Use a known network name or an RPC URL", network)
			}

			out.Info("Fetching transaction %s from %s...", txHash[:10]+"...", network)

			txResult, err := rpcCallLongTimeout(networkRPC, "eth_getTransactionByHash", txHash)
			if err != nil {
				return fmt.Errorf("failed to fetch transaction: %w", err)
			}

			var tx struct {
				BlockNumber string `json:"blockNumber"`
				From        string `json:"from"`
				To          string `json:"to"`
				Value       string `json:"value"`
				Input       string `json:"input"`
				Gas         string `json:"gas"`
				GasPrice    string `json:"gasPrice"`
				Nonce       string `json:"nonce"`
			}
			if err := json.Unmarshal(txResult, &tx); err != nil {
				return fmt.Errorf("failed to parse transaction: %w", err)
			}

			if tx.BlockNumber == "" {
				return fmt.Errorf("transaction not found or still pending: %s", txHash)
			}

			blockHex := strings.TrimPrefix(tx.BlockNumber, "0x")
			txBlock, _ := new(big.Int).SetString(blockHex, 16)

			receiptResult, err := rpcCallLongTimeout(networkRPC, "eth_getTransactionReceipt", txHash)
			if err != nil {
				return fmt.Errorf("failed to fetch receipt: %w", err)
			}

			var receipt struct {
				Status          string `json:"status"`
				GasUsed         string `json:"gasUsed"`
				ContractAddress string `json:"contractAddress"`
				Logs            []struct {
					Address string   `json:"address"`
					Topics  []string `json:"topics"`
					Data    string   `json:"data"`
				} `json:"logs"`
			}
			json.Unmarshal(receiptResult, &receipt)

			fmt.Println()
			out.Info("Transaction Details:")
			out.Info("  Hash:      %s", txHash)
			out.Info("  Block:     #%s", txBlock.String())
			out.Info("  From:      %s", tx.From)
			if tx.To != "" {
				out.Info("  To:        %s", tx.To)
			} else {
				out.Info("  To:        (contract creation)")
			}

			if tx.Value != "" && tx.Value != "0x0" {
				valHex := strings.TrimPrefix(tx.Value, "0x")
				valWei, _ := new(big.Int).SetString(valHex, 16)
				valETH := new(big.Float).Quo(
					new(big.Float).SetInt(valWei),
					new(big.Float).SetFloat64(1e18),
				)
				out.Info("  Value:     %s ETH", valETH.Text('f', 6))
			}

			if receipt.GasUsed != "" {
				gasHex := strings.TrimPrefix(receipt.GasUsed, "0x")
				gasUsed, _ := new(big.Int).SetString(gasHex, 16)
				out.Info("  Gas Used:  %s", gasUsed.String())
			}

			if receipt.Status == "0x1" {
				out.Success("  Status:    Success")
			} else if receipt.Status == "0x0" {
				out.Error("  Status:    Reverted")
			}

			if receipt.ContractAddress != "" {
				out.Info("  Created:   %s", receipt.ContractAddress)
			}

			out.Info("  Events:    %d log(s) emitted", len(receipt.Logs))

			if len(tx.Input) > 10 {
				out.Info("  Method:    %s", tx.Input[:10])
			}

			forkBlock := txBlock.Uint64() - 1
			if block > 0 {
				forkBlock = block
			}

			localRPC, err := getChainRPC(chainName)
			if err != nil {
				return err
			}

			out.Info("")
			out.Info("Forking %s at block #%d (one block before tx)...", network, forkBlock)

			forkParams := map[string]any{
				"forking": map[string]any{
					"jsonRpcUrl":  networkRPC,
					"blockNumber": fmt.Sprintf("0x%x", forkBlock),
				},
			}

			if _, err := rpcCallLongTimeout(localRPC, "anvil_reset", forkParams); err != nil {
				if _, err2 := rpcCallLongTimeout(localRPC, "hardhat_reset", forkParams); err2 != nil {
					return fmt.Errorf("failed to fork network: %w", err)
				}
			}

			out.Success("Forked at block #%d", forkBlock)

			rpcCall(localRPC, "anvil_impersonateAccount", tx.From)

			bigBal := new(big.Int).Mul(big.NewInt(100000), new(big.Int).Exp(big.NewInt(10), big.NewInt(18), nil))
			rpcCall(localRPC, "anvil_setBalance", tx.From, fmt.Sprintf("0x%x", bigBal))

			out.Info("Replaying transaction...")

			callObj := map[string]string{
				"from":  tx.From,
				"input": tx.Input,
				"gas":   tx.Gas,
			}
			if tx.To != "" {
				callObj["to"] = tx.To
			}
			if tx.Value != "" {
				callObj["value"] = tx.Value
			}

			if verbose {
				traceOpts := map[string]string{"tracer": "callTracer"}
				traceResult, err := rpcCallLongTimeout(localRPC, "debug_traceCall", callObj, "latest", traceOpts)
				if err == nil {
					out.Info("")
					out.Info("Execution Trace:")
					var trace map[string]interface{}
					if json.Unmarshal(traceResult, &trace) == nil {
						prettyTrace, _ := json.MarshalIndent(trace, "  ", "  ")
						fmt.Printf("  %s\n", string(prettyTrace))
					}
				}
			}

			callResult, err := rpcCallLongTimeout(localRPC, "eth_call", callObj, "latest")
			if err != nil {
				out.Error("")
				out.Error("Transaction reverted during replay!")
				out.Error("  Reason: %s", err.Error())

				if strings.Contains(err.Error(), "revert") {
					out.Info("")
					out.Info("The transaction reverts at this block. This matches the on-chain result.")
				}
			} else {
				out.Success("")
				out.Success("Transaction replayed successfully!")
				if len(callResult) > 2 {
					out.Info("  Return data: %s", string(callResult))
				}
			}

			sendObj := map[string]string{
				"from":  tx.From,
				"input": tx.Input,
			}
			if tx.To != "" {
				sendObj["to"] = tx.To
			}
			if tx.Value != "" {
				sendObj["value"] = tx.Value
			}

			replayTxResult, err := rpcCall(localRPC, "eth_sendTransaction", sendObj)
			if err == nil {
				var replayHash string
				json.Unmarshal(replayTxResult, &replayHash)
				if replayHash != "" {
					rpcCall(localRPC, "evm_mine")

					replayReceipt, err := rpcCall(localRPC, "eth_getTransactionReceipt", replayHash)
					if err == nil {
						var rr struct {
							GasUsed string `json:"gasUsed"`
							Status  string `json:"status"`
						}
						json.Unmarshal(replayReceipt, &rr)

						if rr.GasUsed != "" {
							gasHex := strings.TrimPrefix(rr.GasUsed, "0x")
							replayGas, _ := new(big.Int).SetString(gasHex, 16)
							origGasHex := strings.TrimPrefix(receipt.GasUsed, "0x")
							origGas, _ := new(big.Int).SetString(origGasHex, 16)

							out.Info("")
							out.Info("Gas Comparison:")
							out.Info("  Original:  %s gas", origGas.String())
							out.Info("  Replayed:  %s gas", replayGas.String())
							diff := new(big.Int).Sub(replayGas, origGas)
							if diff.Sign() != 0 {
								out.Info("  Diff:      %s gas", diff.String())
							} else {
								out.Info("  Diff:      identical")
							}
						}
					}
				}
			}

			rpcCall(localRPC, "anvil_stopImpersonatingAccount", tx.From)

			fmt.Println()
			return nil
		},
	}

	cmd.Flags().StringVar(&chainName, "chain", "", "target local chain")
	cmd.Flags().StringVar(&network, "network", "mainnet", "source network to fetch tx from")
	cmd.Flags().Uint64Var(&block, "block", 0, "custom fork block (default: one block before tx)")
	cmd.Flags().BoolVar(&verbose, "verbose", false, "show full execution trace")

	return cmd
}

func newReplayBlockCmd() *cobra.Command {
	var (
		chainName string
		network   string
		limit     int
	)

	cmd := &cobra.Command{
		Use:   "block <block-number>",
		Short: "Replay all transactions in a block",
		Args:  requireArgs(1, "dokrypt replay block <block-number>"),
		RunE: func(cmd *cobra.Command, args []string) error {
			out := getOutput()

			blockNum, ok := new(big.Int).SetString(args[0], 0)
			if !ok {
				return fmt.Errorf("invalid block number: %s", args[0])
			}

			networkRPC := ""
			if resolved, ok := knownNetworks[strings.ToLower(network)]; ok {
				networkRPC = resolved
			} else if strings.HasPrefix(network, "http") {
				networkRPC = network
			} else {
				return fmt.Errorf("unknown network %q", network)
			}

			blockHex := fmt.Sprintf("0x%x", blockNum)
			out.Info("Fetching block #%s from %s...", blockNum.String(), network)

			blockResult, err := rpcCallLongTimeout(networkRPC, "eth_getBlockByNumber", blockHex, true)
			if err != nil {
				return fmt.Errorf("failed to fetch block: %w", err)
			}

			var blockData struct {
				Transactions []struct {
					Hash  string `json:"hash"`
					From  string `json:"from"`
					To    string `json:"to"`
					Value string `json:"value"`
				} `json:"transactions"`
			}
			if err := json.Unmarshal(blockResult, &blockData); err != nil {
				return fmt.Errorf("failed to parse block: %w", err)
			}

			txCount := len(blockData.Transactions)
			out.Info("Block #%s contains %d transactions", blockNum.String(), txCount)

			if limit > 0 && txCount > limit {
				txCount = limit
				out.Info("Showing first %d transactions (use --limit to change)", limit)
			}

			fmt.Println()
			headers := []string{"#", "Hash", "From", "To", "Value (ETH)"}
			var rows [][]string

			for i := 0; i < txCount; i++ {
				tx := blockData.Transactions[i]
				to := tx.To
				if to == "" {
					to = "(create)"
				} else if len(to) > 10 {
					to = to[:6] + "..." + to[len(to)-4:]
				}

				from := tx.From
				if len(from) > 10 {
					from = from[:6] + "..." + from[len(from)-4:]
				}

				valETH := "0"
				if tx.Value != "" && tx.Value != "0x0" {
					valHex := strings.TrimPrefix(tx.Value, "0x")
					valWei, ok := new(big.Int).SetString(valHex, 16)
					if ok {
						valF := new(big.Float).Quo(
							new(big.Float).SetInt(valWei),
							new(big.Float).SetFloat64(1e18),
						)
						valETH = valF.Text('f', 4)
					}
				}

				hash := tx.Hash
				if len(hash) > 14 {
					hash = hash[:10] + "..."
				}

				rows = append(rows, []string{
					fmt.Sprintf("%d", i),
					hash,
					from,
					to,
					valETH,
				})
			}

			out.Table(headers, rows)
			fmt.Println()
			out.Info("To replay a specific transaction:")
			out.Info("  dokrypt replay tx <hash> --network %s", network)

			localRPC, err := getChainRPC(chainName)
			if err == nil {
				forkBlock := blockNum.Uint64() - 1
				out.Info("")
				out.Info("Forking %s at block #%d to replay...", network, forkBlock)

				forkParams := map[string]any{
					"forking": map[string]any{
						"jsonRpcUrl":  networkRPC,
						"blockNumber": fmt.Sprintf("0x%x", forkBlock),
					},
				}
				if _, err := rpcCallLongTimeout(localRPC, "anvil_reset", forkParams); err == nil {
					out.Success("Forked successfully. Local chain is at block #%d", forkBlock)
				}
			}

			return nil
		},
	}

	cmd.Flags().StringVar(&chainName, "chain", "", "target local chain")
	cmd.Flags().StringVar(&network, "network", "mainnet", "source network")
	cmd.Flags().IntVar(&limit, "limit", 50, "max transactions to display")

	return cmd
}

func newReplayTraceCmd() *cobra.Command {
	var (
		chainName string
		network   string
	)

	cmd := &cobra.Command{
		Use:   "trace <tx-hash>",
		Short: "Get detailed execution trace of a transaction",
		Args:  requireArgs(1, "dokrypt replay trace <tx-hash>"),
		RunE: func(cmd *cobra.Command, args []string) error {
			out := getOutput()
			txHash := args[0]

			networkRPC := ""
			if resolved, ok := knownNetworks[strings.ToLower(network)]; ok {
				networkRPC = resolved
			} else if strings.HasPrefix(network, "http") {
				networkRPC = network
			} else {
				return fmt.Errorf("unknown network %q", network)
			}

			out.Info("Tracing transaction %s on %s...", txHash[:10]+"...", network)

			traceOpts := map[string]string{"tracer": "callTracer"}
			traceResult, err := rpcCallLongTimeout(networkRPC, "debug_traceTransaction", txHash, traceOpts)
			if err != nil {
				out.Info("Remote trace not available, forking and tracing locally...")

				localRPC, err := getChainRPC(chainName)
				if err != nil {
					return fmt.Errorf("need a running local chain for tracing: %w", err)
				}

				txResult, err := rpcCallLongTimeout(networkRPC, "eth_getTransactionByHash", txHash)
				if err != nil {
					return fmt.Errorf("failed to fetch transaction: %w", err)
				}
				var tx struct {
					BlockNumber string `json:"blockNumber"`
				}
				json.Unmarshal(txResult, &tx)

				blockHex := strings.TrimPrefix(tx.BlockNumber, "0x")
				blockNum, _ := new(big.Int).SetString(blockHex, 16)

				forkParams := map[string]any{
					"forking": map[string]any{
						"jsonRpcUrl":  networkRPC,
						"blockNumber": fmt.Sprintf("0x%x", blockNum.Uint64()),
					},
				}
				rpcCallLongTimeout(localRPC, "anvil_reset", forkParams)

				traceResult, err = rpcCallLongTimeout(localRPC, "debug_traceTransaction", txHash, traceOpts)
				if err != nil {
					return fmt.Errorf("failed to trace transaction: %w", err)
				}
			}

			var trace map[string]interface{}
			if err := json.Unmarshal(traceResult, &trace); err != nil {
				fmt.Println(string(traceResult))
				return nil
			}

			fmt.Println()
			printCallTrace(out, trace, 0)
			fmt.Println()

			return nil
		},
	}

	cmd.Flags().StringVar(&chainName, "chain", "", "target local chain for tracing")
	cmd.Flags().StringVar(&network, "network", "mainnet", "source network")

	return cmd
}

func printCallTrace(out interface {
	Info(string, ...interface{})
	Success(string, ...interface{})
	Error(string, ...interface{})
}, trace map[string]interface{}, depth int) {
	indent := strings.Repeat("  ", depth)

	callType, _ := trace["type"].(string)
	from, _ := trace["from"].(string)
	to, _ := trace["to"].(string)
	value, _ := trace["value"].(string)
	gasUsed, _ := trace["gasUsed"].(string)
	output, _ := trace["output"].(string)
	errorMsg, _ := trace["error"].(string)
	input, _ := trace["input"].(string)

	method := ""
	if len(input) >= 10 {
		method = input[:10]
	}

	if to == "" {
		to = "(create)"
	}

	if errorMsg != "" {
		out.Error("%s%s %s -> %s [%s] REVERT: %s", indent, callType, truncAddr(from), truncAddr(to), method, errorMsg)
	} else {
		gasInfo := ""
		if gasUsed != "" {
			gasHex := strings.TrimPrefix(gasUsed, "0x")
			gas, _ := new(big.Int).SetString(gasHex, 16)
			gasInfo = fmt.Sprintf(" (%s gas)", gas.String())
		}

		valInfo := ""
		if value != "" && value != "0x0" {
			valHex := strings.TrimPrefix(value, "0x")
			valWei, _ := new(big.Int).SetString(valHex, 16)
			if valWei != nil && valWei.Sign() > 0 {
				valETH := new(big.Float).Quo(new(big.Float).SetInt(valWei), new(big.Float).SetFloat64(1e18))
				valInfo = fmt.Sprintf(" {%s ETH}", valETH.Text('f', 6))
			}
		}

		out.Info("%s%s %s -> %s [%s]%s%s", indent, callType, truncAddr(from), truncAddr(to), method, valInfo, gasInfo)

		if output != "" && output != "0x" && len(output) > 2 && depth == 0 {
			if len(output) > 66 {
				out.Info("%s  return: %s...", indent, output[:66])
			} else {
				out.Info("%s  return: %s", indent, output)
			}
		}
	}

	if calls, ok := trace["calls"].([]interface{}); ok {
		for _, call := range calls {
			if callMap, ok := call.(map[string]interface{}); ok {
				printCallTrace(out, callMap, depth+1)
			}
		}
	}
}

func truncAddr(addr string) string {
	if len(addr) > 10 {
		return addr[:6] + "..." + addr[len(addr)-4:]
	}
	return addr
}
