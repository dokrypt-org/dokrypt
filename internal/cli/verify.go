package cli

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"mime/multipart"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/spf13/cobra"
)

var verifyAPIs = map[string]string{
	"etherscan":     "https://api.etherscan.io/api",
	"arbiscan":      "https://api.arbiscan.io/api",
	"polygonscan":   "https://api.polygonscan.com/api",
	"basescan":      "https://api.basescan.org/api",
	"optimistic":    "https://api-optimistic.etherscan.io/api",
	"bscscan":       "https://api.bscscan.com/api",
	"sepolia":       "https://api-sepolia.etherscan.io/api",
	"arb-sepolia":   "https://api-sepolia.arbiscan.io/api",
	"sourcify":      "https://sourcify.dev/server",
	"blockscout":    "",
}

func newVerifyCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "verify",
		Short: "Contract verification and source code publishing",
	}

	cmd.AddCommand(
		newVerifyContractCmd(),
		newVerifyStatusCmd(),
		newVerifySourcesCmd(),
	)

	return cmd
}

func newVerifyContractCmd() *cobra.Command {
	var (
		platform      string
		apiKey        string
		compilerVer   string
		optimizations bool
		optRuns       int
		constructorArgs string
		chainID       string
		apiURL        string
		contractPath  string
		contractName  string
		flattenFile   string
	)

	cmd := &cobra.Command{
		Use:   "contract <address>",
		Short: "Verify a deployed contract on a block explorer",
		Long: `Verify contract source code on Etherscan, Arbiscan, Sourcify, or Blockscout.

Examples:
  dokrypt verify contract 0x1234... --platform arbiscan --api-key YOUR_KEY
  dokrypt verify contract 0x1234... --platform sourcify --chain-id 42161
  dokrypt verify contract 0x1234... --platform blockscout --api-url http://localhost:4000/api`,
		Args: requireArgs(1, "dokrypt verify contract <address>"),
		RunE: func(cmd *cobra.Command, args []string) error {
			out := getOutput()
			address := args[0]

			if !strings.HasPrefix(address, "0x") || len(address) != 42 {
				return fmt.Errorf("invalid contract address: %s", address)
			}

			sourcePath := contractPath
			if sourcePath == "" {
				return fmt.Errorf("--source is required: path to the Solidity source file")
			}
			if _, err := os.Stat(sourcePath); os.IsNotExist(err) {
				return fmt.Errorf("source file not found: %s", sourcePath)
			}

			sourceCode, err := os.ReadFile(sourcePath)
			if err != nil {
				return fmt.Errorf("failed to read source file: %w", err)
			}

			if flattenFile != "" {
				flatSource, err := os.ReadFile(flattenFile)
				if err != nil {
					return fmt.Errorf("failed to read flattened file: %w", err)
				}
				sourceCode = flatSource
			}

			if contractName == "" {
				contractName = strings.TrimSuffix(filepath.Base(sourcePath), ".sol")
			}

			platform = strings.ToLower(platform)

			switch platform {
			case "sourcify":
				return verifySourcify(out, address, chainID, sourcePath)
			case "blockscout":
				if apiURL == "" {
					return fmt.Errorf("--api-url is required for Blockscout verification")
				}
				return verifyBlockscout(out, address, string(sourceCode), contractName, compilerVer, optimizations, optRuns, constructorArgs, apiURL)
			default:
				return verifyEtherscan(out, address, string(sourceCode), contractName, compilerVer, optimizations, optRuns, constructorArgs, platform, apiKey, apiURL)
			}
		},
	}

	cmd.Flags().StringVar(&platform, "platform", "etherscan", "verification platform: etherscan, arbiscan, polygonscan, basescan, sourcify, blockscout")
	cmd.Flags().StringVar(&apiKey, "api-key", "", "explorer API key (or set ETHERSCAN_API_KEY / ARBISCAN_API_KEY env var)")
	cmd.Flags().StringVar(&compilerVer, "compiler", "v0.8.20+commit.a1b79de6", "Solidity compiler version")
	cmd.Flags().BoolVar(&optimizations, "optimize", true, "optimizer was enabled during compilation")
	cmd.Flags().IntVar(&optRuns, "opt-runs", 200, "optimizer runs")
	cmd.Flags().StringVar(&constructorArgs, "constructor-args", "", "ABI-encoded constructor arguments (hex)")
	cmd.Flags().StringVar(&chainID, "chain-id", "42161", "chain ID (for Sourcify)")
	cmd.Flags().StringVar(&apiURL, "api-url", "", "custom API endpoint URL")
	cmd.Flags().StringVar(&contractPath, "source", "", "path to Solidity source file")
	cmd.Flags().StringVar(&contractName, "name", "", "contract name (defaults to filename)")
	cmd.Flags().StringVar(&flattenFile, "flatten", "", "path to flattened source file (overrides --source for verification)")

	_ = cmd.MarkFlagRequired("source")

	return cmd
}

func newVerifyStatusCmd() *cobra.Command {
	var (
		platform string
		apiKey   string
		apiURL   string
	)

	cmd := &cobra.Command{
		Use:   "status <guid>",
		Short: "Check verification status",
		Args:  requireArgs(1, "dokrypt verify status <guid>"),
		RunE: func(cmd *cobra.Command, args []string) error {
			out := getOutput()
			guid := args[0]

			resolvedURL := apiURL
			if resolvedURL == "" {
				if u, ok := verifyAPIs[strings.ToLower(platform)]; ok {
					resolvedURL = u
				} else {
					return fmt.Errorf("unknown platform %q, use --api-url", platform)
				}
			}

			key := resolveAPIKey(platform, apiKey)

			params := fmt.Sprintf("?module=contract&action=checkverifystatus&guid=%s&apikey=%s", guid, key)
			resp, err := http.Get(resolvedURL + params)
			if err != nil {
				return fmt.Errorf("failed to check status: %w", err)
			}
			defer resp.Body.Close()

			var result struct {
				Status  string `json:"status"`
				Message string `json:"message"`
				Result  string `json:"result"`
			}
			if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
				return fmt.Errorf("failed to parse response: %w", err)
			}

			if result.Status == "1" {
				out.Success("Verification successful!")
				out.Info("  %s", result.Result)
			} else if strings.Contains(result.Result, "Pending") {
				out.Info("Verification pending... check again in a few seconds.")
				out.Info("  %s", result.Result)
			} else {
				out.Error("Verification failed: %s", result.Result)
			}

			return nil
		},
	}

	cmd.Flags().StringVar(&platform, "platform", "etherscan", "verification platform")
	cmd.Flags().StringVar(&apiKey, "api-key", "", "explorer API key")
	cmd.Flags().StringVar(&apiURL, "api-url", "", "custom API endpoint URL")

	return cmd
}

func newVerifySourcesCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "sources <directory>",
		Short: "List Solidity source files in a directory",
		Args:  cobra.MaximumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			out := getOutput()

			dir := "contracts"
			if len(args) > 0 {
				dir = args[0]
			}

			if _, err := os.Stat(dir); os.IsNotExist(err) {
				return fmt.Errorf("directory not found: %s", dir)
			}

			fmt.Println()
			var count int
			err := filepath.Walk(dir, func(path string, info os.FileInfo, err error) error {
				if err != nil {
					return nil
				}
				if !info.IsDir() && strings.HasSuffix(path, ".sol") && !strings.HasSuffix(path, ".t.sol") {
					rel, _ := filepath.Rel(dir, path)
					name := strings.TrimSuffix(filepath.Base(path), ".sol")
					out.Info("  %s  (%s)", name, rel)
					count++
				}
				return nil
			})
			if err != nil {
				return err
			}

			fmt.Println()
			out.Info("Found %d contract source files in %s/", count, dir)
			return nil
		},
	}
}

func verifyEtherscan(out interface {
	Info(string, ...interface{})
	Success(string, ...interface{})
	Error(string, ...interface{})
}, address, sourceCode, contractName, compilerVer string, optimize bool, optRuns int, constructorArgs, platform, apiKey, apiURL string) error {
	resolvedURL := apiURL
	if resolvedURL == "" {
		if u, ok := verifyAPIs[strings.ToLower(platform)]; ok {
			resolvedURL = u
		} else {
			return fmt.Errorf("unknown platform %q. Supported: etherscan, arbiscan, polygonscan, basescan, optimistic, sepolia, arb-sepolia", platform)
		}
	}

	key := resolveAPIKey(platform, apiKey)
	if key == "" {
		return fmt.Errorf("API key required. Use --api-key or set %s_API_KEY environment variable", strings.ToUpper(platform))
	}

	optEnabled := "0"
	if optimize {
		optEnabled = "1"
	}

	out.Info("Verifying %s on %s...", contractName, platform)
	out.Info("  Address:    %s", address)
	out.Info("  Compiler:   %s", compilerVer)
	out.Info("  Optimizer:  %v (%d runs)", optimize, optRuns)

	formData := map[string]string{
		"apikey":                key,
		"module":               "contract",
		"action":               "verifysourcecode",
		"contractaddress":      address,
		"sourceCode":           sourceCode,
		"codeformat":           "solidity-single-file",
		"contractname":         contractName,
		"compilerversion":      compilerVer,
		"optimizationUsed":     optEnabled,
		"runs":                 fmt.Sprintf("%d", optRuns),
		"constructorArguements": constructorArgs,
	}

	var body bytes.Buffer
	writer := multipart.NewWriter(&body)
	for k, v := range formData {
		writer.WriteField(k, v)
	}
	writer.Close()

	client := &http.Client{Timeout: 30 * time.Second}
	req, err := http.NewRequest("POST", resolvedURL, &body)
	if err != nil {
		return fmt.Errorf("failed to create request: %w", err)
	}
	req.Header.Set("Content-Type", writer.FormDataContentType())

	resp, err := client.Do(req)
	if err != nil {
		return fmt.Errorf("verification request failed: %w", err)
	}
	defer resp.Body.Close()

	var result struct {
		Status  string `json:"status"`
		Message string `json:"message"`
		Result  string `json:"result"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return fmt.Errorf("failed to parse response: %w", err)
	}

	if result.Status == "1" {
		out.Success("Verification submitted!")
		out.Info("  GUID: %s", result.Result)
		out.Info("")
		out.Info("Check status with:")
		out.Info("  dokrypt verify status %s --platform %s --api-key %s", result.Result, platform, key)
	} else {
		out.Error("Verification failed: %s", result.Result)
	}

	return nil
}

func verifySourcify(out interface {
	Info(string, ...interface{})
	Success(string, ...interface{})
	Error(string, ...interface{})
}, address, chainID, sourcePath string) error {
	out.Info("Verifying on Sourcify...")
	out.Info("  Address:  %s", address)
	out.Info("  Chain ID: %s", chainID)

	var body bytes.Buffer
	writer := multipart.NewWriter(&body)

	writer.WriteField("address", address)
	writer.WriteField("chain", chainID)

	file, err := os.Open(sourcePath)
	if err != nil {
		return fmt.Errorf("failed to open source file: %w", err)
	}
	defer file.Close()

	part, err := writer.CreateFormFile("files", filepath.Base(sourcePath))
	if err != nil {
		return fmt.Errorf("failed to create form file: %w", err)
	}
	if _, err := io.Copy(part, file); err != nil {
		return fmt.Errorf("failed to copy file: %w", err)
	}
	writer.Close()

	client := &http.Client{Timeout: 60 * time.Second}
	resp, err := client.Post("https://sourcify.dev/server/verify", writer.FormDataContentType(), &body)
	if err != nil {
		return fmt.Errorf("Sourcify request failed: %w", err)
	}
	defer resp.Body.Close()

	respBody, _ := io.ReadAll(resp.Body)

	if resp.StatusCode == 200 {
		out.Success("Verification successful on Sourcify!")
		out.Info("  View: https://sourcify.dev/#/lookup/%s", address)
	} else {
		var errResp struct {
			Error string `json:"error"`
		}
		json.Unmarshal(respBody, &errResp)
		if errResp.Error != "" {
			out.Error("Sourcify verification failed: %s", errResp.Error)
		} else {
			out.Error("Sourcify verification failed (HTTP %d)", resp.StatusCode)
		}
	}

	return nil
}

func verifyBlockscout(out interface {
	Info(string, ...interface{})
	Success(string, ...interface{})
	Error(string, ...interface{})
}, address, sourceCode, contractName, compilerVer string, optimize bool, optRuns int, constructorArgs, apiURL string) error {
	out.Info("Verifying on Blockscout...")
	out.Info("  Address:   %s", address)
	out.Info("  API:       %s", apiURL)
	out.Info("  Compiler:  %s", compilerVer)

	optEnabled := "false"
	if optimize {
		optEnabled = "true"
	}

	payload := map[string]interface{}{
		"addressHash":                   address,
		"compilerVersion":               compilerVer,
		"contractSourceCode":            sourceCode,
		"name":                          contractName,
		"optimization":                  optEnabled,
		"optimizationRuns":              optRuns,
		"constructorArguments":          constructorArgs,
		"autodetectConstructorArguments": constructorArgs == "",
	}

	data, _ := json.Marshal(payload)
	client := &http.Client{Timeout: 30 * time.Second}

	verifyURL := strings.TrimSuffix(apiURL, "/") + "/api?module=contract&action=verify"
	resp, err := client.Post(verifyURL, "application/json", bytes.NewReader(data))
	if err != nil {
		return fmt.Errorf("Blockscout request failed: %w", err)
	}
	defer resp.Body.Close()

	var result struct {
		Status  string `json:"status"`
		Message string `json:"message"`
		Result  json.RawMessage `json:"result"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return fmt.Errorf("failed to parse response: %w", err)
	}

	if result.Status == "1" {
		out.Success("Contract verified on Blockscout!")
	} else {
		out.Error("Blockscout verification failed: %s", result.Message)
	}

	return nil
}

func resolveAPIKey(platform, provided string) string {
	if provided != "" {
		return provided
	}

	envVars := map[string]string{
		"etherscan":   "ETHERSCAN_API_KEY",
		"arbiscan":    "ARBISCAN_API_KEY",
		"polygonscan": "POLYGONSCAN_API_KEY",
		"basescan":    "BASESCAN_API_KEY",
		"optimistic":  "OPTIMISTIC_API_KEY",
		"bscscan":     "BSCSCAN_API_KEY",
		"sepolia":     "ETHERSCAN_API_KEY",
		"arb-sepolia": "ARBISCAN_API_KEY",
	}

	if envName, ok := envVars[strings.ToLower(platform)]; ok {
		return os.Getenv(envName)
	}

	return os.Getenv("ETHERSCAN_API_KEY")
}
