package cli

import (
	"fmt"
	"os"
	"strings"

	"github.com/spf13/cobra"
	"gopkg.in/yaml.v3"

	"github.com/dokrypt/dokrypt/internal/config"
)

func newConfigCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "config",
		Short: "Configuration management",
	}

	cmd.AddCommand(
		newConfigValidateCmd(),
		newConfigShowCmd(),
		newConfigInitCmd(),
	)

	return cmd
}

func newConfigValidateCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "validate",
		Short: "Validate dokrypt.yaml",
		RunE: func(cmd *cobra.Command, args []string) error {
			out := getOutput()
			cfgPath := getConfigPath()

			if _, err := os.Stat(cfgPath); os.IsNotExist(err) {
				return fmt.Errorf("No dokrypt.yaml found at %s", cfgPath)
			}

			cfg, err := config.Parse(cfgPath)
			if err != nil {
				return fmt.Errorf("Parse error: %w", err)
			}

			if err := config.Validate(cfg); err != nil {
				return fmt.Errorf("Validation failed: %w", err)
			}

			fmt.Println()
			out.Success("Configuration is valid!")
			fmt.Println()
			out.Info("  Project:   %s", cfg.Name)
			out.Info("  Version:   %s", cfg.Version)
			out.Info("  Chains:    %d", len(cfg.Chains))
			for name, chain := range cfg.Chains {
				out.Info("    - %s (engine: %s, chain_id: %d)", name, chain.Engine, chain.ChainID)
			}
			if len(cfg.Services) > 0 {
				out.Info("  Services:  %d", len(cfg.Services))
				for name, svc := range cfg.Services {
					deps := ""
					if len(svc.DependsOn) > 0 {
						deps = fmt.Sprintf(" [depends_on: %s]", strings.Join(svc.DependsOn, ", "))
					}
					out.Info("    - %s (type: %s)%s", name, svc.Type, deps)
				}
			}
			fmt.Println()
			return nil
		},
	}
}

func newConfigShowCmd() *cobra.Command {
	var raw bool

	cmd := &cobra.Command{
		Use:   "show",
		Short: "Show resolved configuration",
		RunE: func(cmd *cobra.Command, args []string) error {
			cfgPath := getConfigPath()

			if raw {
				data, err := os.ReadFile(cfgPath)
				if err != nil {
					return fmt.Errorf("No dokrypt.yaml found at %s", cfgPath)
				}
				fmt.Print(string(data))
				return nil
			}

			cfg, err := config.Parse(cfgPath)
			if err != nil {
				return fmt.Errorf("Parse error: %w", err)
			}

			data, err := yaml.Marshal(cfg)
			if err != nil {
				return fmt.Errorf("failed to serialize config: %w", err)
			}

			fmt.Println(string(data))
			return nil
		},
	}

	cmd.Flags().BoolVar(&raw, "raw", false, "show raw file without resolving defaults")
	return cmd
}

func newConfigInitCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "init",
		Short: "Generate default dokrypt.yaml in current directory",
		RunE: func(cmd *cobra.Command, args []string) error {
			out := getOutput()

			if _, err := os.Stat("dokrypt.yaml"); err == nil {
				return fmt.Errorf("dokrypt.yaml already exists in this directory")
			}

			yamlContent := generateDefaultYAML()
			if err := os.WriteFile("dokrypt.yaml", []byte(yamlContent), 0644); err != nil {
				return fmt.Errorf("failed to write dokrypt.yaml: %w", err)
			}

			out.Success("Generated dokrypt.yaml")
			out.Info("Edit the file, then run 'dokrypt up' to start.")
			return nil
		},
	}
}

func generateDefaultYAML() string {
	var sb strings.Builder
	sb.WriteString("name: my-project\n")
	sb.WriteString("version: \"1.0\"\n")
	sb.WriteString("\n")
	sb.WriteString("chains:\n")
	sb.WriteString("  ethereum:\n")
	sb.WriteString("    engine: anvil\n")
	sb.WriteString("    chain_id: 31337\n")
	sb.WriteString("    block_time: 1s\n")
	sb.WriteString("    accounts: 10\n")
	sb.WriteString("    balance: \"10000000000000000000000\"\n")
	return sb.String()
}
