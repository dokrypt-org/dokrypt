package cli

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/spf13/cobra"

	"github.com/dokrypt/dokrypt/internal/config"
)

func newCICmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "ci",
		Short: "CI/CD workflow generation",
	}

	cmd.AddCommand(
		newCIGenerateCmd(),
		newCIValidateCmd(),
	)

	return cmd
}

func newCIGenerateCmd() *cobra.Command {
	var (
		provider string
		outDir   string
		withDeploy bool
	)

	cmd := &cobra.Command{
		Use:   "generate",
		Short: "Generate CI/CD workflow files",
		Long:  "Generate CI/CD pipeline configuration for GitHub Actions or GitLab CI that runs dokrypt tests on every push.",
		RunE: func(cmd *cobra.Command, args []string) error {
			out := getOutput()

			cfg, _ := config.Parse(getConfigPath())

			projectName := "my-project"
			if cfg != nil && cfg.Name != "" {
				projectName = cfg.Name
			}

			provider = strings.ToLower(provider)

			switch provider {
			case "github", "github-actions":
				return generateGitHubActions(out, projectName, cfg, outDir, withDeploy)
			case "gitlab", "gitlab-ci":
				return generateGitLabCI(out, projectName, cfg, outDir, withDeploy)
			default:
				return fmt.Errorf("unsupported CI provider %q. Supported: github, gitlab", provider)
			}
		},
	}

	cmd.Flags().StringVar(&provider, "provider", "github", "CI provider: github, gitlab")
	cmd.Flags().StringVar(&outDir, "output", ".", "output directory for generated files")
	cmd.Flags().BoolVar(&withDeploy, "deploy", false, "include deployment step (testnet)")

	return cmd
}

func newCIValidateCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "validate",
		Short: "Validate CI workflow configuration",
		RunE: func(cmd *cobra.Command, args []string) error {
			out := getOutput()

			ghPath := filepath.Join(".github", "workflows", "dokrypt.yml")
			if _, err := os.Stat(ghPath); err == nil {
				out.Success("Found GitHub Actions workflow: %s", ghPath)
				return validateWorkflowFile(out, ghPath)
			}

			glPath := ".gitlab-ci.yml"
			if _, err := os.Stat(glPath); err == nil {
				out.Success("Found GitLab CI config: %s", glPath)
				return nil
			}

			out.Warning("No CI workflow files found. Run 'dokrypt ci generate' to create one.")
			return nil
		},
	}
}

func generateGitHubActions(out interface{ Info(string, ...interface{}); Success(string, ...interface{}) }, projectName string, cfg *config.Config, outDir string, withDeploy bool) error {
	dir := filepath.Join(outDir, ".github", "workflows")
	if err := os.MkdirAll(dir, 0755); err != nil {
		return fmt.Errorf("failed to create directory %s: %w", dir, err)
	}

	testFlags := ""
	if cfg != nil {
		if cfg.Tests.GasReport {
			testFlags += " --gas-report"
		}
		if cfg.Tests.Coverage {
			testFlags += " --coverage"
		}
	}

	workflow := fmt.Sprintf(`name: Dokrypt CI

on:
  push:
    branches: [main, develop]
  pull_request:
    branches: [main]

env:
  DOKRYPT_VERSION: "latest"

jobs:
  test:
    name: Smart Contract Tests
    runs-on: ubuntu-latest

    steps:
      - name: Checkout code
        uses: actions/checkout@v4

      - name: Install Dokrypt
        run: npm install -g dokrypt@${{ env.DOKRYPT_VERSION }}

      - name: Verify installation
        run: dokrypt doctor

      - name: Start environment
        run: dokrypt up --detach --fresh

      - name: Wait for services
        run: dokrypt status --wait --timeout 60s

      - name: Run tests
        run: dokrypt test run --json%s > test-results.json

      - name: Upload test results
        if: always()
        uses: actions/upload-artifact@v4
        with:
          name: test-results
          path: test-results.json

      - name: Stop environment
        if: always()
        run: dokrypt down --volumes
`, testFlags)

	if withDeploy {
		workflow += `
  deploy-testnet:
    name: Deploy to Testnet
    needs: test
    runs-on: ubuntu-latest
    if: github.ref == 'refs/heads/main' && github.event_name == 'push'

    steps:
      - name: Checkout code
        uses: actions/checkout@v4

      - name: Install Dokrypt
        run: npm install -g dokrypt@${{ env.DOKRYPT_VERSION }}

      - name: Install Foundry
        uses: foundry-rs/foundry-toolchain@v1

      - name: Deploy contracts
        env:
          RPC_URL: ${{ secrets.TESTNET_RPC_URL }}
          PRIVATE_KEY: ${{ secrets.DEPLOYER_PRIVATE_KEY }}
        run: forge script scripts/Deploy.s.sol --rpc-url $RPC_URL --private-key $PRIVATE_KEY --broadcast
`
	}

	path := filepath.Join(dir, "dokrypt.yml")
	if err := os.WriteFile(path, []byte(workflow), 0644); err != nil {
		return fmt.Errorf("failed to write workflow file: %w", err)
	}

	out.Success("Generated GitHub Actions workflow: %s", path)
	out.Info("  Triggers: push to main/develop, pull requests to main")
	out.Info("  Steps: install dokrypt, start environment, run tests, upload results")
	if withDeploy {
		out.Info("  Deploy: testnet deployment on push to main (requires secrets)")
	}
	out.Info("")
	out.Info("Required secrets (if using deploy):")
	out.Info("  TESTNET_RPC_URL      — RPC endpoint for testnet")
	out.Info("  DEPLOYER_PRIVATE_KEY — Private key for deployment")

	return nil
}

func generateGitLabCI(out interface{ Info(string, ...interface{}); Success(string, ...interface{}) }, projectName string, cfg *config.Config, outDir string, withDeploy bool) error {
	testFlags := ""
	if cfg != nil {
		if cfg.Tests.GasReport {
			testFlags += " --gas-report"
		}
		if cfg.Tests.Coverage {
			testFlags += " --coverage"
		}
	}

	pipeline := fmt.Sprintf(`stages:
  - test
%s
variables:
  DOKRYPT_VERSION: "latest"

test:
  stage: test
  image: node:20
  services:
    - docker:dind
  variables:
    DOCKER_HOST: tcp://docker:2375
    DOCKER_TLS_CERTDIR: ""
  before_script:
    - npm install -g dokrypt@${DOKRYPT_VERSION}
    - dokrypt doctor
  script:
    - dokrypt up --detach --fresh
    - dokrypt status --wait --timeout 60s
    - dokrypt test run --json%s > test-results.json
    - dokrypt down --volumes
  artifacts:
    when: always
    paths:
      - test-results.json
    reports:
      junit: test-results.json
`, func() string {
		if withDeploy {
			return "  - deploy"
		}
		return ""
	}(), testFlags)

	if withDeploy {
		pipeline += `
deploy-testnet:
  stage: deploy
  image: node:20
  only:
    - main
  before_script:
    - npm install -g dokrypt@${DOKRYPT_VERSION}
    - curl -L https://foundry.paradigm.xyz | bash && foundryup
  script:
    - forge script scripts/Deploy.s.sol --rpc-url $TESTNET_RPC_URL --private-key $DEPLOYER_PRIVATE_KEY --broadcast
`
	}

	path := filepath.Join(outDir, ".gitlab-ci.yml")
	if err := os.WriteFile(path, []byte(pipeline), 0644); err != nil {
		return fmt.Errorf("failed to write GitLab CI file: %w", err)
	}

	out.Success("Generated GitLab CI config: %s", path)
	out.Info("  Triggers: all pushes and merge requests")
	out.Info("  Steps: install dokrypt, start environment, run tests")
	if withDeploy {
		out.Info("  Deploy: testnet deployment on push to main (requires CI variables)")
	}

	return nil
}

func validateWorkflowFile(out interface{ Info(string, ...interface{}); Success(string, ...interface{}); Warning(string, ...interface{}) }, path string) error {
	data, err := os.ReadFile(path)
	if err != nil {
		return fmt.Errorf("failed to read %s: %w", path, err)
	}

	content := string(data)
	checks := []struct {
		search string
		label  string
	}{
		{"dokrypt up", "Environment startup"},
		{"dokrypt test", "Test execution"},
		{"dokrypt down", "Environment cleanup"},
		{"actions/checkout", "Code checkout"},
	}

	allGood := true
	for _, c := range checks {
		if strings.Contains(content, c.search) {
			out.Success("  %s", c.label)
		} else {
			out.Warning("  Missing: %s (%s)", c.label, c.search)
			allGood = false
		}
	}

	if allGood {
		out.Success("Workflow is valid")
	}

	return nil
}
