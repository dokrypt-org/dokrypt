package cli

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	"github.com/spf13/cobra"

	"github.com/dokrypt/dokrypt/internal/config"
	"github.com/dokrypt/dokrypt/internal/testrunner"
)

func newTestCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "test",
		Short: "Built-in test runner",
	}

	cmd.AddCommand(
		newTestRunCmd(),
		newTestListCmd(),
		newTestReportCmd(),
	)

	return cmd
}

func newTestRunCmd() *cobra.Command {
	var (
		suiteName  string
		filter     string
		parallel   int
		gasReport  bool
		coverage   bool
		snapshot   bool
		timeout    time.Duration
		jsonFlag   bool
	)

	cmd := &cobra.Command{
		Use:   "run",
		Short: "Run all tests",
		RunE: func(cmd *cobra.Command, args []string) error {
			out := getOutput()

			cfg, cfgErr := config.Parse(getConfigPath())

			runCfg := testrunner.Config{
				Filter:   filter,
				Parallel: parallel,
				Timeout:  timeout,
			}

			if cfgErr == nil {
				tc := cfg.Tests
				if !cmd.Flags().Changed("parallel") && tc.Parallel > 0 {
					runCfg.Parallel = tc.Parallel
				}
				if !cmd.Flags().Changed("timeout") && tc.Timeout != "" {
					if d, err := config.ParseDuration(tc.Timeout); err == nil {
						runCfg.Timeout = d
					}
				}
				if !cmd.Flags().Changed("gas-report") {
					gasReport = tc.GasReport
				}
				if !cmd.Flags().Changed("coverage") {
					coverage = tc.Coverage
				}
				if !cmd.Flags().Changed("snapshot") {
					snapshot = tc.SnapshotIsolation
				}
			}

			runCfg.GasReport = gasReport
			runCfg.Coverage = coverage
			runCfg.Snapshot = snapshot

			if runCfg.Parallel <= 0 {
				runCfg.Parallel = 4
			}

			runner := testrunner.NewRunner(runCfg)

			suites := discoverTestSuites(cfg)

			if suiteName != "" {
				var matched []*testrunner.Suite
				for _, s := range suites {
					if s.Name == suiteName {
						matched = append(matched, s)
					}
				}
				suites = matched
			}

			if len(suites) == 0 {
				builtin, err := builtinTestSuite()
				if err != nil {
					return fmt.Errorf("failed to build built-in test suite: %w", err)
				}
				suites = append(suites, builtin)
			}

			for _, s := range suites {
				runner.AddSuite(s)
			}

			out.Info("Running tests...")

			ctx := cmd.Context()
			if ctx == nil {
				ctx = context.Background()
			}

			result, err := runner.Run(ctx)
			if err != nil {
				return fmt.Errorf("test run failed: %w", err)
			}

			if jsonFlag {
				enc := json.NewEncoder(os.Stdout)
				enc.SetIndent("", "  ")
				if err := enc.Encode(result); err != nil {
					return fmt.Errorf("failed to encode result as JSON: %w", err)
				}
			} else {
				reporter := &testrunner.TableReporter{}
				reporter.Report(result, os.Stdout)

				if result.GasReport != nil && len(result.GasReport.Entries) > 0 {
					testrunner.PrintReport(result.GasReport, os.Stdout)
				}
			}

			if cfgErr == nil {
				if err := saveTestReport(cfg.Name, result); err != nil {
					out.Warning("Could not save test report: %v", err)
				}
			}

			if result.Failed > 0 {
				return fmt.Errorf("%d test(s) failed", result.Failed)
			}

			out.Success("All tests passed")
			return nil
		},
	}

	cmd.Flags().StringVar(&suiteName, "suite", "", "run a specific suite by name")
	cmd.Flags().StringVar(&filter, "filter", "", "filter tests by name substring")
	cmd.Flags().IntVar(&parallel, "parallel", 4, "maximum parallel test count")
	cmd.Flags().BoolVar(&gasReport, "gas-report", false, "generate gas usage report")
	cmd.Flags().BoolVar(&coverage, "coverage", false, "generate coverage report")
	cmd.Flags().BoolVar(&snapshot, "snapshot", false, "enable snapshot isolation per test")
	cmd.Flags().DurationVar(&timeout, "timeout", 0, "per-test timeout (e.g. 30s, 2m)")
	cmd.Flags().BoolVar(&jsonFlag, "json", false, "output results as JSON")

	return cmd
}

func newTestListCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "list",
		Short: "List test suites",
		RunE: func(cmd *cobra.Command, args []string) error {
			out := getOutput()

			cfg, _ := config.Parse(getConfigPath())
			suites := discoverTestSuites(cfg)

			if len(suites) == 0 {
				builtin, err := builtinTestSuite()
				if err == nil {
					suites = append(suites, builtin)
				}
			}

			if len(suites) == 0 {
				out.Info("No test suites found.")
				return nil
			}

			headers := []string{"Suite", "Tests", "Description"}
			var rows [][]string
			for _, s := range suites {
				rows = append(rows, []string{
					s.Name,
					fmt.Sprintf("%d", len(s.Tests)),
					s.Description,
				})
			}

			fmt.Println()
			out.Table(headers, rows)
			fmt.Println()
			return nil
		},
	}
}

func newTestReportCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "report",
		Short: "Show last test report",
		RunE: func(cmd *cobra.Command, args []string) error {
			out := getOutput()

			cfg, err := config.Parse(getConfigPath())
			if err != nil {
				return fmt.Errorf("No dokrypt.yaml found. Run 'dokrypt init' first.")
			}

			result, err := loadTestReport(cfg.Name)
			if err != nil {
				out.Info("No test reports available. Run 'dokrypt test run' first.")
				return nil
			}

			reporter := &testrunner.TableReporter{}
			reporter.Report(result, os.Stdout)

			if result.GasReport != nil && len(result.GasReport.Entries) > 0 {
				testrunner.PrintReport(result.GasReport, os.Stdout)
			}

			return nil
		},
	}
}

func discoverTestSuites(cfg *config.Config) []*testrunner.Suite {
	var suites []*testrunner.Suite

	if cfg != nil && len(cfg.Tests.Suites) > 0 {
		baseDir := cfg.Tests.Dir
		if baseDir == "" {
			baseDir = "."
		}
		for name, sc := range cfg.Tests.Suites {
			dir := filepath.Join(baseDir, name)
			files := scanTestDir(dir)
			if len(files) == 0 && sc.Pattern != "" {
				files = scanTestDir(sc.Pattern)
			}
			if len(files) == 0 {
				continue
			}
			s := testrunner.NewSuite(name)
			s.Description = fmt.Sprintf("%d test file(s) matching %q", len(files), sc.Pattern)
			for _, f := range files {
				fname := f // capture
				s.AddTest(filepath.Base(fname), func(ctx context.Context) error {
					return executeTestFile(ctx, fname)
				})
			}
			suites = append(suites, s)
		}
		return suites
	}

	dirs := []string{"test", "tests", filepath.Join("contracts", "test")}
	if cfg != nil && cfg.Tests.Dir != "" {
		dirs = append([]string{cfg.Tests.Dir}, dirs...)
	}

	for _, dir := range dirs {
		files := scanTestDir(dir)
		if len(files) == 0 {
			continue
		}
		s := testrunner.NewSuite(dir)
		s.Description = fmt.Sprintf("%d test file(s) in %s/", len(files), dir)
		for _, f := range files {
			fname := f
			s.AddTest(filepath.Base(fname), func(ctx context.Context) error {
				return executeTestFile(ctx, fname)
			})
		}
		suites = append(suites, s)
	}

	return suites
}

func executeTestFile(ctx context.Context, filePath string) error {
	absPath, err := filepath.Abs(filePath)
	if err != nil {
		return fmt.Errorf("failed to resolve path %q: %w", filePath, err)
	}

	workDir := detectProjectRoot(filepath.Dir(absPath))

	name, args := testCommandForFile(absPath)
	if name == "" {
		return fmt.Errorf("unsupported test file type: %s", filepath.Base(absPath))
	}

	cmd := exec.CommandContext(ctx, name, args...)
	cmd.Dir = workDir

	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	if err := cmd.Run(); err != nil {
		combined := strings.TrimSpace(stdout.String() + "\n" + stderr.String())
		if combined != "" {
			return fmt.Errorf("%s\n%s", err, combined)
		}
		return err
	}

	return nil
}

func testCommandForFile(absPath string) (string, []string) {
	lower := strings.ToLower(absPath)

	switch {
	case strings.HasSuffix(lower, ".t.sol"),
		strings.HasSuffix(lower, ".test.sol"),
		strings.HasSuffix(lower, "_test.sol"):
		return "forge", []string{"test", "--match-path", absPath}

	case strings.HasSuffix(lower, ".test.js"), strings.HasSuffix(lower, ".js"):
		if hasHardhatConfig(filepath.Dir(absPath)) {
			return "npx", []string{"hardhat", "test", absPath}
		}
		return "node", []string{absPath}

	case strings.HasSuffix(lower, ".test.ts"), strings.HasSuffix(lower, ".ts"):
		return "npx", []string{"hardhat", "test", absPath}
	}

	return "", nil
}

func detectProjectRoot(dir string) string {
	markers := []string{
		"foundry.toml",
		"package.json",
		"hardhat.config.js",
		"hardhat.config.ts",
	}

	cur := dir
	for {
		for _, m := range markers {
			if _, err := os.Stat(filepath.Join(cur, m)); err == nil {
				return cur
			}
		}
		parent := filepath.Dir(cur)
		if parent == cur {
			break // reached filesystem root
		}
		cur = parent
	}
	return dir
}

func hasHardhatConfig(dir string) bool {
	root := detectProjectRoot(dir)
	for _, name := range []string{"hardhat.config.js", "hardhat.config.ts"} {
		if _, err := os.Stat(filepath.Join(root, name)); err == nil {
			return true
		}
	}
	return false
}

func scanTestDir(dir string) []string {
	info, err := os.Stat(dir)
	if err != nil || !info.IsDir() {
		return nil
	}

	var files []string
	filepath.Walk(dir, func(path string, fi os.FileInfo, err error) error {
		if err != nil || fi.IsDir() {
			return nil
		}
		name := fi.Name()
		if isTestFile(name) {
			files = append(files, path)
		}
		return nil
	})
	return files
}

func isTestFile(name string) bool {
	lower := strings.ToLower(name)
	switch {
	case strings.HasSuffix(lower, ".t.sol"):
		return true
	case strings.HasSuffix(lower, ".test.sol"):
		return true
	case strings.HasSuffix(lower, "_test.sol"):
		return true
	case strings.HasSuffix(lower, ".test.js"):
		return true
	case strings.HasSuffix(lower, ".test.ts"):
		return true
	}
	return false
}

func builtinTestSuite() (*testrunner.Suite, error) {
	rpcURL, err := getChainRPC("")
	if err != nil {
		return nil, err
	}

	s := testrunner.NewSuite("builtin")
	s.Description = "Built-in chain validation tests"

	s.AddTest("chain_is_running", func(ctx context.Context) error {
		_, err := rpcCallCtx(ctx, rpcURL, "eth_blockNumber")
		if err != nil {
			return fmt.Errorf("chain is not responding: %w", err)
		}
		return nil
	})

	s.AddTest("accounts_available", func(ctx context.Context) error {
		result, err := rpcCallCtx(ctx, rpcURL, "eth_accounts")
		if err != nil {
			return fmt.Errorf("failed to query accounts: %w", err)
		}
		var accounts []string
		if err := json.Unmarshal(result, &accounts); err != nil {
			return fmt.Errorf("failed to parse accounts response: %w", err)
		}
		if len(accounts) == 0 {
			return fmt.Errorf("no accounts available on the chain")
		}
		return nil
	})

	s.AddTest("chain_id_valid", func(ctx context.Context) error {
		id, err := getChainID(rpcURL)
		if err != nil {
			return fmt.Errorf("failed to get chain ID: %w", err)
		}
		if id == 0 {
			return fmt.Errorf("chain ID is 0, expected a non-zero value")
		}
		return nil
	})

	s.AddTest("mining_works", func(ctx context.Context) error {
		beforeBlock, err := getCurrentBlock(rpcURL)
		if err != nil {
			return fmt.Errorf("failed to get block number before mining: %w", err)
		}

		if _, err := rpcCallCtx(ctx, rpcURL, "evm_mine"); err != nil {
			return fmt.Errorf("mining failed: %w", err)
		}

		afterBlock, err := getCurrentBlock(rpcURL)
		if err != nil {
			return fmt.Errorf("failed to get block number after mining: %w", err)
		}
		if afterBlock <= beforeBlock {
			return fmt.Errorf("block number did not advance (before=%d, after=%d)", beforeBlock, afterBlock)
		}
		return nil
	})

	s.AddTest("snapshot_restore", func(ctx context.Context) error {
		snapResult, err := rpcCallCtx(ctx, rpcURL, "evm_snapshot")
		if err != nil {
			return fmt.Errorf("evm_snapshot failed: %w", err)
		}
		var snapID string
		if err := json.Unmarshal(snapResult, &snapID); err != nil {
			return fmt.Errorf("failed to parse snapshot ID: %w", err)
		}

		blockBefore, err := getCurrentBlock(rpcURL)
		if err != nil {
			return fmt.Errorf("failed to get block before mining: %w", err)
		}

		if _, err := rpcCallCtx(ctx, rpcURL, "evm_mine"); err != nil {
			return fmt.Errorf("mining after snapshot failed: %w", err)
		}

		revertResult, err := rpcCallCtx(ctx, rpcURL, "evm_revert", snapID)
		if err != nil {
			return fmt.Errorf("evm_revert failed: %w", err)
		}
		var success bool
		if err := json.Unmarshal(revertResult, &success); err != nil {
			return fmt.Errorf("failed to parse revert result: %w", err)
		}
		if !success {
			return fmt.Errorf("evm_revert returned false")
		}

		blockAfter, err := getCurrentBlock(rpcURL)
		if err != nil {
			return fmt.Errorf("failed to get block after revert: %w", err)
		}
		if blockAfter > blockBefore {
			return fmt.Errorf("snapshot restore failed: block number did not revert (before=%d, after=%d)", blockBefore, blockAfter)
		}
		return nil
	})

	return s, nil
}

func testReportDir(projectName string) string {
	home, _ := os.UserHomeDir()
	return filepath.Join(home, ".dokrypt", "reports", projectName)
}

func saveTestReport(projectName string, result *testrunner.Result) error {
	dir := testReportDir(projectName)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return fmt.Errorf("failed to create report directory: %w", err)
	}

	data, err := json.MarshalIndent(result, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal test result: %w", err)
	}

	path := filepath.Join(dir, "latest.json")
	if err := os.WriteFile(path, data, 0644); err != nil {
		return fmt.Errorf("failed to write report file: %w", err)
	}

	return nil
}

func loadTestReport(projectName string) (*testrunner.Result, error) {
	path := filepath.Join(testReportDir(projectName), "latest.json")
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("no saved report found: %w", err)
	}

	var result testrunner.Result
	if err := json.Unmarshal(data, &result); err != nil {
		return nil, fmt.Errorf("failed to parse saved report: %w", err)
	}

	return &result, nil
}
