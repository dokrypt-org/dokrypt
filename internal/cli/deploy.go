package cli

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/spf13/cobra"
)

type DeploymentRecord struct {
	ContractName string `json:"contract_name"`
	Address      string `json:"address"`
	Network      string `json:"network"`
	ChainID      string `json:"chain_id"`
	TxHash       string `json:"tx_hash"`
	BlockNumber  string `json:"block_number"`
	Deployer     string `json:"deployer"`
	Timestamp    string `json:"timestamp"`
	Compiler     string `json:"compiler"`
	Verified     bool   `json:"verified"`
	Tags         string `json:"tags"`
}

type DeploymentManifest struct {
	ProjectName string             `json:"project_name"`
	Version     string             `json:"version"`
	Deployments []DeploymentRecord `json:"deployments"`
}

func newDeployCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "deploy",
		Short: "Deployment management and tracking",
	}

	cmd.AddCommand(
		newDeployTrackCmd(),
		newDeployListCmd(),
		newDeployExportCmd(),
		newDeployMultiCmd(),
		newDeployDiffCmd(),
	)

	return cmd
}

func newDeployTrackCmd() *cobra.Command {
	var (
		network      string
		chainID      string
		txHash       string
		deployer     string
		compiler     string
		tags         string
	)

	cmd := &cobra.Command{
		Use:   "track <contract-name> <address>",
		Short: "Track a contract deployment",
		Long: `Record a contract deployment for tracking across environments.

Examples:
  dokrypt deploy track MyToken 0x1234... --network arbitrum --tx-hash 0xabc...
  dokrypt deploy track MyToken 0x1234... --network sepolia --tags "staging,v2"`,
		Args: requireArgs(2, "dokrypt deploy track <contract-name> <address>"),
		RunE: func(cmd *cobra.Command, args []string) error {
			out := getOutput()

			contractName := args[0]
			address := args[1]

			if !strings.HasPrefix(address, "0x") || len(address) != 42 {
				return fmt.Errorf("invalid contract address: %s", address)
			}

			manifest, manifestPath := loadManifest()

			blockNumber := ""
			if txHash != "" {
				networkRPC := ""
				if resolved, ok := knownNetworks[strings.ToLower(network)]; ok {
					networkRPC = resolved
				}
				if networkRPC != "" {
					receipt, err := rpcCallLongTimeout(networkRPC, "eth_getTransactionReceipt", txHash)
					if err == nil {
						var r struct {
							BlockNumber string `json:"blockNumber"`
						}
						json.Unmarshal(receipt, &r)
						blockNumber = r.BlockNumber
					}
				}
			}

			record := DeploymentRecord{
				ContractName: contractName,
				Address:      address,
				Network:      network,
				ChainID:      chainID,
				TxHash:       txHash,
				BlockNumber:  blockNumber,
				Deployer:     deployer,
				Timestamp:    time.Now().UTC().Format(time.RFC3339),
				Compiler:     compiler,
				Verified:     false,
				Tags:         tags,
			}

			manifest.Deployments = append(manifest.Deployments, record)
			if err := saveManifest(manifest, manifestPath); err != nil {
				return err
			}

			out.Success("Tracked deployment: %s at %s", contractName, address)
			out.Info("  Network:   %s", network)
			if txHash != "" {
				out.Info("  Tx Hash:   %s", txHash)
			}
			if tags != "" {
				out.Info("  Tags:      %s", tags)
			}
			out.Info("  Saved to:  %s", manifestPath)

			return nil
		},
	}

	cmd.Flags().StringVar(&network, "network", "localhost", "deployment network")
	cmd.Flags().StringVar(&chainID, "chain-id", "", "chain ID")
	cmd.Flags().StringVar(&txHash, "tx-hash", "", "deployment transaction hash")
	cmd.Flags().StringVar(&deployer, "deployer", "", "deployer address")
	cmd.Flags().StringVar(&compiler, "compiler", "solc-0.8.20", "compiler version")
	cmd.Flags().StringVar(&tags, "tags", "", "comma-separated tags (e.g., staging,v2)")

	return cmd
}

func newDeployListCmd() *cobra.Command {
	var (
		network string
		tag     string
	)

	cmd := &cobra.Command{
		Use:   "list",
		Short: "List tracked deployments",
		RunE: func(cmd *cobra.Command, args []string) error {
			out := getOutput()

			manifest, _ := loadManifest()

			if len(manifest.Deployments) == 0 {
				out.Info("No deployments tracked yet.")
				out.Info("Use 'dokrypt deploy track <name> <address>' to record a deployment.")
				return nil
			}

			var filtered []DeploymentRecord
			for _, d := range manifest.Deployments {
				if network != "" && !strings.EqualFold(d.Network, network) {
					continue
				}
				if tag != "" && !strings.Contains(d.Tags, tag) {
					continue
				}
				filtered = append(filtered, d)
			}

			if len(filtered) == 0 {
				out.Info("No deployments match the filters.")
				return nil
			}

			fmt.Println()
			headers := []string{"Contract", "Address", "Network", "Deployed", "Verified", "Tags"}
			var rows [][]string

			for _, d := range filtered {
				addr := d.Address
				if len(addr) > 14 {
					addr = addr[:6] + "..." + addr[len(addr)-4:]
				}

				verified := "no"
				if d.Verified {
					verified = "yes"
				}

				deployed := d.Timestamp
				if len(deployed) > 10 {
					deployed = deployed[:10]
				}

				rows = append(rows, []string{
					d.ContractName,
					addr,
					d.Network,
					deployed,
					verified,
					d.Tags,
				})
			}

			out.Table(headers, rows)
			fmt.Println()
			out.Info("Total: %d deployment(s)", len(filtered))

			return nil
		},
	}

	cmd.Flags().StringVar(&network, "network", "", "filter by network")
	cmd.Flags().StringVar(&tag, "tag", "", "filter by tag")

	return cmd
}

func newDeployExportCmd() *cobra.Command {
	var format string

	cmd := &cobra.Command{
		Use:   "export",
		Short: "Export deployment manifest",
		RunE: func(cmd *cobra.Command, args []string) error {
			out := getOutput()

			manifest, _ := loadManifest()

			if len(manifest.Deployments) == 0 {
				out.Info("No deployments to export.")
				return nil
			}

			switch strings.ToLower(format) {
			case "json":
				data, _ := json.MarshalIndent(manifest, "", "  ")
				fmt.Println(string(data))
			case "env":
				for _, d := range manifest.Deployments {
					envName := strings.ToUpper(strings.ReplaceAll(d.ContractName, "-", "_"))
					networkSuffix := strings.ToUpper(strings.ReplaceAll(d.Network, "-", "_"))
					fmt.Printf("%s_%s_ADDRESS=%s\n", envName, networkSuffix, d.Address)
				}
			case "markdown", "md":
				fmt.Println("# Deployment Addresses")
				fmt.Println()
				fmt.Println("| Contract | Network | Address |")
				fmt.Println("|----------|---------|---------|")
				for _, d := range manifest.Deployments {
					fmt.Printf("| %s | %s | `%s` |\n", d.ContractName, d.Network, d.Address)
				}
			default:
				return fmt.Errorf("unsupported format %q. Use: json, env, markdown", format)
			}

			out.Info("")
			out.Info("Exported %d deployment(s) as %s", len(manifest.Deployments), format)
			return nil
		},
	}

	cmd.Flags().StringVar(&format, "format", "json", "output format: json, env, markdown")

	return cmd
}

func newDeployMultiCmd() *cobra.Command {
	var (
		networks string
		script   string
		dryRun   bool
	)

	cmd := &cobra.Command{
		Use:   "multi",
		Short: "Deploy contracts to multiple chains",
		Long: `Deploy the same contracts to multiple networks in sequence.

Examples:
  dokrypt deploy multi --networks "arbitrum,optimism,base" --script scripts/deploy.js
  dokrypt deploy multi --networks "sepolia,arb-sepolia" --script scripts/deploy.js --dry-run`,
		RunE: func(cmd *cobra.Command, args []string) error {
			out := getOutput()

			if script == "" {
				return fmt.Errorf("--script is required: path to deployment script")
			}
			if _, err := os.Stat(script); os.IsNotExist(err) {
				return fmt.Errorf("script not found: %s", script)
			}

			chainList := strings.Split(networks, ",")
			if len(chainList) == 0 {
				return fmt.Errorf("--networks is required: comma-separated list of networks")
			}

			out.Info("Multi-chain deployment")
			out.Info("  Script:   %s", script)
			out.Info("  Networks: %s", networks)
			if dryRun {
				out.Info("  Mode:     dry-run (no actual deployment)")
			}
			fmt.Println()

			results := make(map[string]string)

			for i, chain := range chainList {
				chain = strings.TrimSpace(chain)

				rpcURL := ""
				if resolved, ok := knownNetworks[strings.ToLower(chain)]; ok {
					rpcURL = resolved
				} else if strings.HasPrefix(chain, "http") {
					rpcURL = chain
				}

				out.Info("[%d/%d] Deploying to %s...", i+1, len(chainList), chain)

				if rpcURL == "" {
					out.Error("  Unknown network %q, skipping", chain)
					results[chain] = "skipped (unknown network)"
					continue
				}

				if dryRun {
					out.Info("  RPC: %s", rpcURL)
					out.Info("  Would run: npx hardhat run %s --network %s", script, chain)
					results[chain] = "dry-run ok"
					continue
				}

				out.Info("  RPC: %s", rpcURL)
				out.Info("  Run: RPC_URL=%s npx hardhat run %s", rpcURL, script)
				out.Info("  Or:  forge script %s --rpc-url %s --broadcast", script, rpcURL)
				results[chain] = "ready"

				fmt.Println()
			}

			fmt.Println()
			out.Info("Deployment Summary:")
			headers := []string{"Network", "Status"}
			var rows [][]string
			for _, chain := range chainList {
				chain = strings.TrimSpace(chain)
				rows = append(rows, []string{chain, results[chain]})
			}
			out.Table(headers, rows)

			fmt.Println()
			out.Info("After deploying, track each deployment with:")
			out.Info("  dokrypt deploy track <ContractName> <address> --network <network> --tx-hash <hash>")

			return nil
		},
	}

	cmd.Flags().StringVar(&networks, "networks", "", "comma-separated networks (e.g., arbitrum,optimism,base)")
	cmd.Flags().StringVar(&script, "script", "", "path to deployment script")
	cmd.Flags().BoolVar(&dryRun, "dry-run", false, "preview deployment without executing")

	_ = cmd.MarkFlagRequired("networks")

	return cmd
}

func newDeployDiffCmd() *cobra.Command {
	var (
		network1 string
		network2 string
	)

	cmd := &cobra.Command{
		Use:   "diff",
		Short: "Compare deployments across networks",
		Long:  "Show which contracts are deployed on each network and highlight differences.",
		RunE: func(cmd *cobra.Command, args []string) error {
			out := getOutput()

			manifest, _ := loadManifest()

			if len(manifest.Deployments) == 0 {
				out.Info("No deployments tracked.")
				return nil
			}

			byNetwork := make(map[string]map[string]DeploymentRecord)
			for _, d := range manifest.Deployments {
				if _, ok := byNetwork[d.Network]; !ok {
					byNetwork[d.Network] = make(map[string]DeploymentRecord)
				}
				byNetwork[d.Network][d.ContractName] = d
			}

			if network1 == "" || network2 == "" {
				fmt.Println()
				out.Info("Tracked networks:")
				for net, contracts := range byNetwork {
					out.Info("  %s: %d contract(s)", net, len(contracts))
				}
				out.Info("")
				out.Info("Use --net1 and --net2 to compare two networks:")
				out.Info("  dokrypt deploy diff --net1 arbitrum --net2 sepolia")
				return nil
			}

			net1Contracts := byNetwork[network1]
			net2Contracts := byNetwork[network2]

			allContracts := make(map[string]bool)
			for name := range net1Contracts {
				allContracts[name] = true
			}
			for name := range net2Contracts {
				allContracts[name] = true
			}

			fmt.Println()
			headers := []string{"Contract", network1, network2, "Status"}
			var rows [][]string

			for name := range allContracts {
				d1, ok1 := net1Contracts[name]
				d2, ok2 := net2Contracts[name]

				addr1 := "-"
				addr2 := "-"
				status := ""

				if ok1 {
					addr1 = d1.Address
					if len(addr1) > 14 {
						addr1 = addr1[:6] + "..." + addr1[len(addr1)-4:]
					}
				}
				if ok2 {
					addr2 = d2.Address
					if len(addr2) > 14 {
						addr2 = addr2[:6] + "..." + addr2[len(addr2)-4:]
					}
				}

				if ok1 && ok2 {
					status = "both"
				} else if ok1 {
					status = fmt.Sprintf("only %s", network1)
				} else {
					status = fmt.Sprintf("only %s", network2)
				}

				rows = append(rows, []string{name, addr1, addr2, status})
			}

			out.Table(headers, rows)
			fmt.Println()

			return nil
		},
	}

	cmd.Flags().StringVar(&network1, "net1", "", "first network to compare")
	cmd.Flags().StringVar(&network2, "net2", "", "second network to compare")

	return cmd
}

func loadManifest() (*DeploymentManifest, string) {
	manifestPath := filepath.Join("deployments.json")

	manifest := &DeploymentManifest{
		Version:     "1.0",
		Deployments: []DeploymentRecord{},
	}

	data, err := os.ReadFile(manifestPath)
	if err == nil {
		json.Unmarshal(data, manifest)
	}

	return manifest, manifestPath
}

func saveManifest(manifest *DeploymentManifest, path string) error {
	data, err := json.MarshalIndent(manifest, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal manifest: %w", err)
	}

	if err := os.WriteFile(path, data, 0644); err != nil {
		return fmt.Errorf("failed to write manifest: %w", err)
	}

	return nil
}
