package cli

import (
	"fmt"
	"os"

	"github.com/dokrypt/dokrypt/internal/common"
	"github.com/spf13/cobra"
)

var (
	Version = "dev"
	Commit  = "none"
	Date    = "unknown"
)

var (
	cfgFile    string
	verbose    bool
	quiet      bool
	jsonOutput bool
	runtime    string
	output     common.Output
)

func NewRootCmd() *cobra.Command {
	rootCmd := &cobra.Command{
		Use:   "dokrypt",
		Short: "Dokrypt — Docker for Web3",
		Long:  "Dokrypt is a Web3-native containerization and orchestration platform for dApp development, testing, and deployment.",
		PersistentPreRun: func(cmd *cobra.Command, args []string) {
			level := "warn"
			if verbose {
				level = "debug"
			}
			common.SetupLogger(level, jsonOutput, os.Stderr)

			if jsonOutput {
				output = common.NewJSONOutput(os.Stdout)
			} else {
				output = common.NewConsoleOutput(os.Stdout, common.NoColor(), quiet)
			}
		},
		SilenceUsage:  true,
		SilenceErrors: true,
	}

	rootCmd.PersistentFlags().StringVarP(&cfgFile, "config", "c", "", "path to dokrypt.yaml")
	rootCmd.PersistentFlags().BoolVarP(&verbose, "verbose", "v", false, "enable verbose output")
	rootCmd.PersistentFlags().BoolVarP(&quiet, "quiet", "q", false, "suppress non-error output")
	rootCmd.PersistentFlags().BoolVar(&jsonOutput, "json", false, "output in JSON format")
	rootCmd.PersistentFlags().StringVar(&runtime, "runtime", "", "container runtime (docker or podman)")

	rootCmd.AddCommand(
		newInitCmd(),
		newUpCmd(),
		newDownCmd(),
		newRestartCmd(),
		newStatusCmd(),
		newLogsCmd(),
		newExecCmd(),
		newSnapshotCmd(),
		newForkCmd(),
		newAccountsCmd(),
		newChainCmd(),
		newBridgeCmd(),
		newTestCmd(),
		newPluginCmd(),
		newTemplateCmd(),
		newConfigCmd(),
		newDoctorCmd(),
		newVersionCmd(),
		newMarketplaceCmd(),
		newCICmd(),
	)

	return rootCmd
}

func Execute() int {
	rootCmd := NewRootCmd()
	if err := rootCmd.Execute(); err != nil {
		if output != nil {
			output.Error("%v", err)
		} else {
			os.Stderr.WriteString("Error: " + err.Error() + "\n")
		}
		return 1
	}
	return 0
}

func getConfigPath() string {
	if cfgFile != "" {
		return cfgFile
	}
	if envPath := os.Getenv("DOKRYPT_CONFIG"); envPath != "" {
		return envPath
	}
	return "dokrypt.yaml"
}

func requireArgs(n int, usage string) cobra.PositionalArgs {
	return func(cmd *cobra.Command, args []string) error {
		if len(args) < n {
			return fmt.Errorf("missing required argument\n\n  Usage: %s\n\n  Run '%s --help' for details", usage, cmd.CommandPath())
		}
		if len(args) > n {
			return fmt.Errorf("too many arguments\n\n  Usage: %s", usage)
		}
		return nil
	}
}

func getOutput() common.Output {
	if output == nil {
		return common.NewConsoleOutput(os.Stdout, false, false)
	}
	return output
}
