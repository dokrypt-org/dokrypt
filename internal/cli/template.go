package cli

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/dokrypt/dokrypt/internal/marketplace"
	"github.com/dokrypt/dokrypt/internal/template"
	"github.com/spf13/cobra"
	"gopkg.in/yaml.v3"
)

func newTemplateCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "template",
		Short: "Template management",
	}

	cmd.AddCommand(
		newTemplateListCmd(),
		newTemplateInfoCmd(),
		newTemplatePullCmd(),
		newTemplatePushCmd(),
		newTemplateCreateCmd(),
	)

	return cmd
}

func newTemplateListCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "list",
		Short: "List available templates",
		RunE: func(cmd *cobra.Command, args []string) error {
			out := getOutput()

			mgr, err := template.DefaultManager()
			if err != nil {
				return fmt.Errorf("failed to initialize template manager: %w", err)
			}

			templates := mgr.List()
			if len(templates) == 0 {
				out.Warning("No templates found")
				return nil
			}

			sort.Slice(templates, func(i, j int) bool {
				return templates[i].Template.Name < templates[j].Template.Name
			})

			headers := []string{"Name", "Version", "Category", "Price", "Description", "Chains"}
			rows := make([][]string, 0, len(templates))
			for _, info := range templates {
				t := info.Template
				price := "free"
				if t.Premium {
					price = t.Price
				}
				category := t.Category
				if category == "" {
					category = "-"
				}
				chains := "-"
				if len(t.Chains) > 0 {
					chains = strings.Join(t.Chains, ", ")
				}
				rows = append(rows, []string{t.Name, t.Version, category, price, t.Description, chains})
			}

			out.Table(headers, rows)
			return nil
		},
	}
}

func newTemplateInfoCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "info <name>",
		Short: "Show template details",
		Args:  requireArgs(1, "dokrypt template info <name>"),
		RunE: func(cmd *cobra.Command, args []string) error {
			out := getOutput()
			name := args[0]

			mgr, err := template.DefaultManager()
			if err != nil {
				return fmt.Errorf("failed to initialize template manager: %w", err)
			}

			info, err := mgr.Get(name)
			if err != nil {
				return fmt.Errorf("template %q not found: %w", name, err)
			}

			t := info.Template

			out.Info("Template: %s", t.Name)
			out.Info("Version:  %s", t.Version)
			out.Info("Description: %s", t.Description)
			out.Info("Author:   %s", t.Author)

			if t.License != "" {
				out.Info("License:  %s", t.License)
			}

			builtIn := "no"
			if info.BuiltIn {
				builtIn = "yes"
			}
			out.Info("Built-in: %s", builtIn)
			if t.Category != "" {
				out.Info("Category: %s", t.Category)
			}
			if t.Difficulty != "" {
				out.Info("Difficulty: %s", t.Difficulty)
			}
			if t.Premium {
				out.Info("Premium:  yes (%s)", t.Price)
			} else {
				out.Info("Premium:  no (free)")
			}

			if info.Path != "" {
				out.Info("Path:     %s", info.Path)
			}

			if len(t.Tags) > 0 {
				out.Info("Tags:     %s", strings.Join(t.Tags, ", "))
			}

			if len(t.Chains) > 0 {
				out.Info("Chains:   %s", strings.Join(t.Chains, ", "))
			}

			if len(t.Services) > 0 {
				out.Info("Services: %s", strings.Join(t.Services, ", "))
			}

			return nil
		},
	}
}

func newTemplatePullCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "pull <name>",
		Short: "Download a template from the marketplace",
		Args:  requireArgs(1, "dokrypt template pull <name>"),
		RunE: func(cmd *cobra.Command, args []string) error {
			out := getOutput()
			name := args[0]

			mgr, err := template.DefaultManager()
			if err != nil {
				return fmt.Errorf("failed to initialize template manager: %w", err)
			}

			info, err := mgr.Get(name)
			if err == nil && info.BuiltIn {
				out.Info("Template %q is a built-in template and does not need to be pulled.", name)
				out.Info("It is already available for use.")
				return nil
			}

			if err == nil {
				out.Info("Template %q is already installed at %s", name, info.Path)
				return nil
			}

			out.Info("Pulling from marketplace...")
			out.Info("Equivalent to: dokrypt marketplace install %s", name)
			fmt.Println()

			reg, regErr := marketplace.DefaultLocalRegistry()
			if regErr != nil {
				return regErr
			}

			client := marketplace.NewClient("")

			out.Step(1, 3, fmt.Sprintf("Fetching %q from marketplace...", name))
			meta, metaErr := client.GetInfo(name)
			if metaErr != nil {
				return metaErr
			}

			out.Step(2, 3, "Downloading...")
			data, dlErr := client.Download(name)
			if dlErr != nil {
				return dlErr
			}

			tmpDir, tmpErr := os.MkdirTemp("", "dokrypt-pull-*")
			if tmpErr != nil {
				return fmt.Errorf("create temp dir: %w", tmpErr)
			}
			defer os.RemoveAll(tmpDir)

			archivePath := filepath.Join(tmpDir, name+".tar.gz")
			if wErr := os.WriteFile(archivePath, data, 0o644); wErr != nil {
				return fmt.Errorf("write archive: %w", wErr)
			}

			out.Step(3, 3, "Installing...")
			if iErr := reg.Install(name, *meta, tmpDir); iErr != nil {
				return fmt.Errorf("install failed: %w", iErr)
			}

			out.Success("Template %q installed from marketplace!", name)
			out.Info("Use it with: dokrypt init my-project --template %s", name)
			return nil
		},
	}
}

func newTemplatePushCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "push",
		Short: "Publish a template to the marketplace",
		Long:  "Publishes the template in the current directory to the Dokrypt marketplace hub.\nEquivalent to: dokrypt marketplace publish",
		RunE: func(cmd *cobra.Command, args []string) error {
			out := getOutput()
			out.Info("Delegating to: dokrypt marketplace publish")
			fmt.Println()

			publishCmd := newMarketplacePublishCmd()
			return publishCmd.RunE(publishCmd, args)
		},
	}
}

func newTemplateCreateCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "create <name>",
		Short: "Scaffold a new template directory",
		Long:  "Creates a new template directory with a template.yaml metadata file and a basic dokrypt.yaml.tmpl template file.",
		Args:  requireArgs(1, "dokrypt template create <name>"),
		RunE: func(cmd *cobra.Command, args []string) error {
			out := getOutput()
			name := args[0]

			if err := os.MkdirAll(name, 0o755); err != nil {
				return fmt.Errorf("failed to create directory %q: %w", name, err)
			}

			meta := template.Template{
				Name:        name,
				Version:     "0.1.0",
				Description: fmt.Sprintf("Custom template: %s", name),
				Author:      "",
				Tags:        []string{},
				Chains:      []string{"ethereum"},
				Services:    []string{},
				License:     "MIT",
			}

			metaData, err := yaml.Marshal(&meta)
			if err != nil {
				return fmt.Errorf("failed to marshal template.yaml: %w", err)
			}

			metaPath := filepath.Join(name, "template.yaml")
			if err := os.WriteFile(metaPath, metaData, 0o644); err != nil {
				return fmt.Errorf("failed to write template.yaml: %w", err)
			}

			tmplContent := `name: {{ .ProjectName }}
version: "1.0"

chains:
  ethereum:
    engine: anvil
    chain_id: 31337
    block_time: 1s
    accounts: 10
    balance: "10000000000000000000000"

# Add services below as needed:
# services:
#   ipfs:
#     type: ipfs
#   explorer:
#     type: blockscout
#     depends_on:
#       - ethereum
`

			tmplPath := filepath.Join(name, "dokrypt.yaml.tmpl")
			if err := os.WriteFile(tmplPath, []byte(tmplContent), 0o644); err != nil {
				return fmt.Errorf("failed to write dokrypt.yaml.tmpl: %w", err)
			}

			out.Success("Template %q created!", name)
			fmt.Println()
			out.Info("Files created:")
			out.Info("  %s", metaPath)
			out.Info("  %s", tmplPath)
			fmt.Println()
			out.Info("Next steps:")
			out.Info("  1. Edit %s to customize template metadata", metaPath)
			out.Info("  2. Edit %s to define the project configuration", tmplPath)
			out.Info("  3. Run 'dokrypt template push' from the %s directory to publish", name)

			return nil
		},
	}
}
