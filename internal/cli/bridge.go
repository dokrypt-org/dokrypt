package cli

import (
	"encoding/json"
	"fmt"
	"math/big"
	"net/http"
	"sort"
	"strings"
	"time"

	"github.com/spf13/cobra"

	"github.com/dokrypt/dokrypt/internal/config"
)

const bridgeAddress = "0x000000000000000000000000000000000000B12D"

func newBridgeCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "bridge",
		Short: "Bridge simulation (multi-chain)",
	}

	cmd.AddCommand(
		newBridgeSendCmd(),
		newBridgeStatusCmd(),
		newBridgeRelayCmd(),
		newBridgeConfigCmd(),
	)

	return cmd
}

func newBridgeSendCmd() *cobra.Command {
	var (
		token string
		from  string
	)

	cmd := &cobra.Command{
		Use:   "send <from-chain> <to-chain> <amount>",
		Short: "Simulate bridge transfer",
		Long: `Simulate a cross-chain bridge transfer.

Locks ETH (or a token) on the source chain by sending it to a bridge address,
then mints the equivalent amount on the destination chain via anvil_setBalance.`,
		Args: requireArgs(3, "dokrypt bridge send <from-chain> <to-chain> <amount>"),
		RunE: func(cmd *cobra.Command, args []string) error {
			out := getOutput()

			srcChain := args[0]
			dstChain := args[1]
			amountStr := args[2]

			ethFloat, ok := new(big.Float).SetString(amountStr)
			if !ok {
				return fmt.Errorf("invalid amount: %s", amountStr)
			}
			weiFloat := new(big.Float).Mul(ethFloat, new(big.Float).SetFloat64(1e18))
			weiInt, _ := weiFloat.Int(nil)
			hexWei := fmt.Sprintf("0x%x", weiInt)

			srcRPC, err := getChainRPC(srcChain)
			if err != nil {
				return fmt.Errorf("source chain %q: %w", srcChain, err)
			}
			dstRPC, err := getChainRPC(dstChain)
			if err != nil {
				return fmt.Errorf("destination chain %q: %w", dstChain, err)
			}

			sender := from
			if sender == "" {
				result, err := rpcCall(srcRPC, "eth_accounts")
				if err != nil {
					return fmt.Errorf("failed to fetch accounts on %s: %w", srcChain, err)
				}
				var accounts []string
				if err := json.Unmarshal(result, &accounts); err != nil || len(accounts) == 0 {
					return fmt.Errorf("no accounts available on %s", srcChain)
				}
				sender = accounts[0]
			}

			asset := "ETH"
			if token != "" {
				asset = token
			}

			out.Info("Initiating bridge transfer...")
			out.Info("  From:   %s (chain: %s)", sender, srcChain)
			out.Info("  To:     %s (chain: %s)", bridgeAddress, dstChain)
			out.Info("  Amount: %s %s", amountStr, asset)

			txObj := map[string]string{
				"from":  sender,
				"to":    bridgeAddress,
				"value": hexWei,
			}
			_, err = rpcCall(srcRPC, "eth_sendTransaction", txObj)
			if err != nil {
				return fmt.Errorf("failed to send lock transaction on %s: %w", srcChain, err)
			}

			rpcCall(srcRPC, "evm_mine")

			out.Info("  Locked %s %s on %s", amountStr, asset, srcChain)

			currentBalance := big.NewInt(0)
			balResult, err := rpcCall(dstRPC, "eth_getBalance", sender, "latest")
			if err == nil {
				var hexBal string
				json.Unmarshal(balResult, &hexBal)
				hexBal = strings.TrimPrefix(hexBal, "0x")
				if parsed, ok := new(big.Int).SetString(hexBal, 16); ok {
					currentBalance = parsed
				}
			}

			newBalance := new(big.Int).Add(currentBalance, weiInt)
			hexNewBalance := fmt.Sprintf("0x%x", newBalance)

			if _, err := rpcCall(dstRPC, "anvil_setBalance", sender, hexNewBalance); err != nil {
				if _, err2 := rpcCall(dstRPC, "hardhat_setBalance", sender, hexNewBalance); err2 != nil {
					return fmt.Errorf("failed to mint on destination chain %s: %w", dstChain, err)
				}
			}

			out.Info("  Minted %s %s on %s for %s", amountStr, asset, dstChain, sender)
			out.Success("Bridge transfer complete: %s -> %s (%s %s)", srcChain, dstChain, amountStr, asset)
			return nil
		},
	}

	cmd.Flags().StringVar(&token, "token", "", "token symbol (defaults to native ETH)")
	cmd.Flags().StringVar(&from, "from", "", "sender address (defaults to account[0] on the source chain)")
	return cmd
}

func newBridgeStatusCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "status",
		Short: "Show bridge queue",
		Long:  `Display configured bridges and their settings from dokrypt.yaml services.`,
		RunE: func(cmd *cobra.Command, args []string) error {
			out := getOutput()

			cfg, err := config.Parse(getConfigPath())
			if err != nil {
				return fmt.Errorf("No dokrypt.yaml found. Run 'dokrypt init' first.")
			}

			type bridgeInfo struct {
				name               string
				chains             []string
				relayDelay         string
				confirmationBlocks int
			}

			var bridges []bridgeInfo
			for name, svc := range cfg.Services {
				if svc.Type != "bridge" {
					continue
				}
				bridges = append(bridges, bridgeInfo{
					name:               name,
					chains:             svc.Chains,
					relayDelay:         svc.RelayDelay,
					confirmationBlocks: svc.ConfirmationBlocks,
				})
			}

			if len(bridges) == 0 {
				out.Info("No bridge services configured in dokrypt.yaml.")
				return nil
			}

			sort.Slice(bridges, func(i, j int) bool {
				return bridges[i].name < bridges[j].name
			})

			fmt.Println()
			out.Info("Bridge Status")
			fmt.Println()

			headers := []string{"Service", "Chains", "Relay Delay", "Confirmations", "Queue"}
			var rows [][]string
			for _, b := range bridges {
				chainsStr := strings.Join(b.chains, " <-> ")
				if chainsStr == "" {
					chainsStr = "-"
				}
				relayDelay := b.relayDelay
				if relayDelay == "" {
					relayDelay = "instant"
				}
				confirmations := fmt.Sprintf("%d", b.confirmationBlocks)
				if b.confirmationBlocks == 0 {
					confirmations = "0 (none)"
				}

				queueStr := bridgeQueueStatus(cfg.Name, b.name)

				rows = append(rows, []string{
					b.name,
					chainsStr,
					relayDelay,
					confirmations,
					queueStr,
				})
			}

			out.Table(headers, rows)
			fmt.Println()
			return nil
		},
	}
}

func newBridgeRelayCmd() *cobra.Command {
	var blocks int

	cmd := &cobra.Command{
		Use:   "relay",
		Short: "Force relay pending messages",
		Long: `Force-relay all pending bridge messages by mining confirmation blocks on
every configured chain. This simulates finality on both sides of the bridge.`,
		RunE: func(cmd *cobra.Command, args []string) error {
			out := getOutput()

			cfg, err := config.Parse(getConfigPath())
			if err != nil {
				return fmt.Errorf("No dokrypt.yaml found. Run 'dokrypt init' first.")
			}

			mineCount := blocks
			if mineCount <= 0 {
				for _, svc := range cfg.Services {
					if svc.Type == "bridge" && svc.ConfirmationBlocks > mineCount {
						mineCount = svc.ConfirmationBlocks
					}
				}
				if mineCount <= 0 {
					mineCount = 1
				}
			}

			chainSet := map[string]struct{}{}
			for _, svc := range cfg.Services {
				if svc.Type != "bridge" {
					continue
				}
				for _, c := range svc.Chains {
					chainSet[c] = struct{}{}
				}
			}

			if len(chainSet) == 0 {
				for name := range cfg.Chains {
					chainSet[name] = struct{}{}
				}
			}

			if len(chainSet) == 0 {
				out.Warning("No chains found to relay on.")
				return nil
			}

			chainNames := make([]string, 0, len(chainSet))
			for name := range chainSet {
				chainNames = append(chainNames, name)
			}
			sort.Strings(chainNames)

			out.Info("Relaying pending messages: mining %d block(s) on %d chain(s)...", mineCount, len(chainNames))

			hexCount := fmt.Sprintf("0x%x", mineCount)

			for _, chainName := range chainNames {
				rpcURL, err := getChainRPC(chainName)
				if err != nil {
					out.Warning("  Skipping %s: %v", chainName, err)
					continue
				}

				if _, err := rpcCall(rpcURL, "anvil_mine", hexCount); err != nil {
					for i := 0; i < mineCount; i++ {
						if _, err2 := rpcCall(rpcURL, "evm_mine"); err2 != nil {
							out.Warning("  Failed to mine on %s: %v", chainName, err2)
							break
						}
					}
				}

				blockNum, err := getCurrentBlock(rpcURL)
				if err != nil {
					out.Info("  %s: mined %d block(s)", chainName, mineCount)
				} else {
					out.Info("  %s: mined %d block(s), now at #%d", chainName, mineCount, blockNum)
				}
			}

			relayed := triggerBridgeRelay(cfg.Name, cfg, out)
			if relayed > 0 {
				out.Success("Relayed %d message(s) via bridge service API", relayed)
			} else {
				out.Info("No pending messages to relay")
			}
			return nil
		},
	}

	cmd.Flags().IntVar(&blocks, "blocks", 0, "number of blocks to mine (defaults to confirmation_blocks from config)")
	return cmd
}

func newBridgeConfigCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "config",
		Short: "Show bridge configuration",
		Long:  `Display bridge service configuration from dokrypt.yaml.`,
		RunE: func(cmd *cobra.Command, args []string) error {
			out := getOutput()

			cfg, err := config.Parse(getConfigPath())
			if err != nil {
				return fmt.Errorf("No dokrypt.yaml found. Run 'dokrypt init' first.")
			}

			type bridgeCfg struct {
				name               string
				chains             []string
				relayDelay         string
				confirmationBlocks int
			}

			var bridges []bridgeCfg
			for name, svc := range cfg.Services {
				if svc.Type != "bridge" {
					continue
				}
				bridges = append(bridges, bridgeCfg{
					name:               name,
					chains:             svc.Chains,
					relayDelay:         svc.RelayDelay,
					confirmationBlocks: svc.ConfirmationBlocks,
				})
			}

			if len(bridges) == 0 {
				out.Info("No bridge services configured in dokrypt.yaml.")
				return nil
			}

			sort.Slice(bridges, func(i, j int) bool {
				return bridges[i].name < bridges[j].name
			})

			fmt.Println()
			out.Info("Bridge Configuration")
			fmt.Println()

			for _, b := range bridges {
				out.Info("  Service: %s", b.name)

				if len(b.chains) > 0 {
					out.Info("    Chains:             %s", strings.Join(b.chains, ", "))
				} else {
					out.Info("    Chains:             (none)")
				}

				if b.relayDelay != "" {
					out.Info("    Relay Delay:        %s", b.relayDelay)
				} else {
					out.Info("    Relay Delay:        instant")
				}

				out.Info("    Confirmation Blocks: %d", b.confirmationBlocks)
				fmt.Println()
			}

			chainSet := map[string]struct{}{}
			for _, b := range bridges {
				for _, c := range b.chains {
					chainSet[c] = struct{}{}
				}
			}

			if len(chainSet) > 0 {
				out.Info("Referenced Chains")
				fmt.Println()

				headers := []string{"Chain", "Engine", "Chain ID", "Block Time"}
				var rows [][]string

				sortedChains := make([]string, 0, len(chainSet))
				for name := range chainSet {
					sortedChains = append(sortedChains, name)
				}
				sort.Strings(sortedChains)

				for _, name := range sortedChains {
					cc, ok := cfg.Chains[name]
					if !ok {
						rows = append(rows, []string{name, "?", "?", "?"})
						continue
					}
					engine := cc.Engine
					if engine == "" {
						engine = "anvil"
					}
					chainID := fmt.Sprintf("%d", cc.ChainID)
					blockTime := cc.BlockTime
					if blockTime == "" {
						blockTime = "auto"
					}
					rows = append(rows, []string{name, engine, chainID, blockTime})
				}

				out.Table(headers, rows)
				fmt.Println()
			}

			return nil
		},
	}
}

func bridgeQueueStatus(projectName, bridgeName string) string {
	state, err := loadState(projectName)
	if err != nil {
		return "unknown"
	}

	cs, ok := state.Containers[bridgeName]
	if !ok {
		return "unknown"
	}

	var port int
	for _, p := range cs.Ports {
		if p > 0 {
			port = p
			break
		}
	}
	if port == 0 {
		return "unknown"
	}

	client := &http.Client{Timeout: 3 * time.Second}
	resp, err := client.Get(fmt.Sprintf("http://localhost:%d/messages", port))
	if err != nil {
		return "unreachable"
	}
	defer resp.Body.Close()

	var result struct {
		Messages []struct {
			Status string `json:"status"`
		} `json:"messages"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return "error"
	}

	if len(result.Messages) == 0 {
		return "empty"
	}

	pending := 0
	for _, m := range result.Messages {
		if m.Status == "pending" {
			pending++
		}
	}
	if pending > 0 {
		return fmt.Sprintf("%d pending", pending)
	}
	return fmt.Sprintf("%d relayed", len(result.Messages))
}

func triggerBridgeRelay(projectName string, cfg *config.Config, out interface {
	Info(string, ...any)
	Warning(string, ...any)
}) int {
	state, err := loadState(projectName)
	if err != nil {
		return 0
	}

	total := 0
	for svcName, svc := range cfg.Services {
		if svc.Type != "bridge" {
			continue
		}

		cs, ok := state.Containers[svcName]
		if !ok {
			continue
		}

		var port int
		for _, p := range cs.Ports {
			if p > 0 {
				port = p
				break
			}
		}
		if port == 0 {
			continue
		}

		relayURL := fmt.Sprintf("http://localhost:%d/relay", port)
		chains := svc.Chains
		payload := "{}"
		if len(chains) >= 2 {
			payload = fmt.Sprintf(`{"from_chain":"%s","to_chain":"%s"}`, chains[0], chains[1])
		}

		client := &http.Client{Timeout: 5 * time.Second}
		resp, err := client.Post(relayURL, "application/json", strings.NewReader(payload))
		if err != nil {
			out.Warning("  Failed to trigger relay on %s: %v", svcName, err)
			continue
		}
		resp.Body.Close()

		if resp.StatusCode < 300 {
			out.Info("  %s: relay triggered successfully", svcName)
			total++
		} else {
			out.Warning("  %s: relay returned status %d", svcName, resp.StatusCode)
		}
	}
	return total
}
