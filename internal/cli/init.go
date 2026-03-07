package cli

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"

	"github.com/dokrypt/dokrypt/internal/template"
	"github.com/spf13/cobra"
)

func newInitCmd() *cobra.Command {
	var (
		templateName string
		chain        string
		engine       string
		noGit        bool
		dir          string
	)

	cmd := &cobra.Command{
		Use:   "init <project-name>",
		Short: "Scaffold a new Dokrypt project",
		Long:  "Creates a new directory with contracts, tests, and configuration from a template.",
		Args:  requireArgs(1, "dokrypt init <project-name>"),
		RunE: func(cmd *cobra.Command, args []string) error {
			out := getOutput()
			targetDir := args[0]

			if dir != "" {
				targetDir = dir
			}

			projectName := filepath.Base(targetDir)

			if info, err := os.Stat(targetDir); err == nil && info.IsDir() {
				entries, _ := os.ReadDir(targetDir)
				if len(entries) > 0 {
					return fmt.Errorf("directory %s already exists and is not empty", targetDir)
				}
			}

			mgr, err := template.DefaultManager()
			if err != nil {
				return fmt.Errorf("failed to initialize template manager: %w", err)
			}

			if _, err := mgr.Get(templateName); err != nil {
				return err
			}

			tmplFS, err := mgr.GetFS(templateName)
			if err != nil {
				return fmt.Errorf("failed to load template: %w", err)
			}

			chainID := uint64(31337)

			out.Step(1, 3, fmt.Sprintf("Scaffolding from template %q...", templateName))
			opts := template.ScaffoldOptions{
				Name:     targetDir,
				Template: templateName,
				Dir:      ".",
				Chain:    chain,
				Engine:   engine,
				ChainID:  chainID,
				NoGit:    noGit,
				Vars: template.Vars{
					ProjectName: projectName,
					ChainName:   chain,
					ChainID:     chainID,
					Engine:      engine,
				},
			}

			if err := template.Scaffold(opts, tmplFS); err != nil {
				return fmt.Errorf("scaffolding failed: %w", err)
			}

			if !noGit {
				out.Step(2, 3, "Initializing git repository...")
				gitCmd := exec.Command("git", "init", targetDir)
				gitCmd.Stdout = nil
				gitCmd.Stderr = nil
				if gitErr := gitCmd.Run(); gitErr != nil {
					out.Warning("Could not initialize git repository: %v", gitErr)
				}
			} else {
				out.Step(2, 3, "Skipping git init...")
			}

			out.Step(3, 3, "Done!")
			fmt.Println()
			out.Success("Project %s created from template %s!", projectName, templateName)
			fmt.Println()
			out.Info("Next steps:")
			out.Info("  cd %s", targetDir)
			out.Info("  dokrypt up")
			fmt.Println()
			return nil
		},
	}

	cmd.Flags().StringVarP(&templateName, "template", "t", "evm-basic", "template to use")
	cmd.Flags().StringVar(&chain, "chain", "ethereum", "chain type")
	cmd.Flags().StringVar(&engine, "engine", "anvil", "node engine")
	cmd.Flags().BoolVar(&noGit, "no-git", false, "skip git init")
	cmd.Flags().StringVar(&dir, "dir", "", "target directory")

	return cmd
}
