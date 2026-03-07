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

func newMarketplaceCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:     "marketplace",
		Aliases: []string{"market", "hub"},
		Short:   "Template marketplace — discover, install, and publish templates",
	}

	cmd.AddCommand(
		newMarketplaceSearchCmd(),
		newMarketplaceBrowseCmd(),
		newMarketplaceInstallCmd(),
		newMarketplaceUninstallCmd(),
		newMarketplacePublishCmd(),
		newMarketplaceInfoCmd(),
		newMarketplaceListCmd(),
	)

	return cmd
}

func newMarketplaceSearchCmd() *cobra.Command {
	var remote bool

	cmd := &cobra.Command{
		Use:   "search <query>",
		Short: "Search for templates",
		Args:  requireArgs(1, "dokrypt marketplace search <query>"),
		RunE: func(cmd *cobra.Command, args []string) error {
			out := getOutput()
			query := args[0]

			if remote {
				client := marketplace.NewClient("")
				result, err := client.Search(query)
				if err != nil {
					return err
				}

				if result.Total == 0 {
					out.Info("No templates found for %q", query)
					return nil
				}

				out.Info("Found %d template(s) for %q:", result.Total, query)
				fmt.Println()
				printPackageTable(out, result.Packages)
				return nil
			}

			reg, err := marketplace.DefaultLocalRegistry()
			if err != nil {
				return err
			}

			matches := reg.Search(query)
			if len(matches) == 0 {
				out.Info("No installed templates match %q", query)
				out.Info("Try: dokrypt marketplace search --remote %s", query)
				return nil
			}

			out.Info("Found %d installed template(s) for %q:", len(matches), query)
			fmt.Println()
			printInstalledTable(out, matches)
			return nil
		},
	}

	cmd.Flags().BoolVar(&remote, "remote", false, "search the remote marketplace hub")
	return cmd
}

func newMarketplaceBrowseCmd() *cobra.Command {
	var category string
	var remote bool

	cmd := &cobra.Command{
		Use:   "browse",
		Short: "Browse templates by category",
		RunE: func(cmd *cobra.Command, args []string) error {
			out := getOutput()

			if remote {
				client := marketplace.NewClient("")
				result, err := client.Browse(category)
				if err != nil {
					return err
				}

				if result.Total == 0 {
					out.Info("No templates found")
					return nil
				}

				if category != "" {
					out.Info("Templates in category %q:", category)
				} else {
					out.Info("All marketplace templates:")
				}
				fmt.Println()
				printPackageTable(out, result.Packages)
				return nil
			}

			reg, err := marketplace.DefaultLocalRegistry()
			if err != nil {
				return err
			}

			matches := reg.Browse(category)
			if len(matches) == 0 {
				out.Info("No installed marketplace templates found")
				out.Info("Try: dokrypt marketplace browse --remote")
				return nil
			}

			out.Info("Installed marketplace templates:")
			fmt.Println()
			printInstalledTable(out, matches)
			return nil
		},
	}

	cmd.Flags().StringVar(&category, "category", "", "filter by category (defi, nft, dao, token, basic)")
	cmd.Flags().BoolVar(&remote, "remote", false, "browse the remote marketplace hub")
	return cmd
}

func newMarketplaceInstallCmd() *cobra.Command {
	var from string

	cmd := &cobra.Command{
		Use:   "install <name>",
		Short: "Install a template from the marketplace or local directory",
		Args:  requireArgs(1, "dokrypt marketplace install <name>"),
		RunE: func(cmd *cobra.Command, args []string) error {
			out := getOutput()
			name := args[0]

			reg, err := marketplace.DefaultLocalRegistry()
			if err != nil {
				return err
			}

			if existing, err := reg.Get(name); err == nil {
				out.Warning("Template %q is already installed (v%s) at %s", name, existing.Version, existing.Path)
				out.Info("To reinstall, first run: dokrypt marketplace uninstall %s", name)
				return nil
			}

			if from != "" {
				return installFromLocal(out, reg, name, from)
			}

			client := marketplace.NewClient("")
			out.Step(1, 3, fmt.Sprintf("Fetching template %q from marketplace...", name))

			meta, err := client.GetInfo(name)
			if err != nil {
				return err
			}

			out.Step(2, 3, "Downloading...")
			data, err := client.Download(name)
			if err != nil {
				return err
			}

			tmpDir, err := os.MkdirTemp("", "dokrypt-install-*")
			if err != nil {
				return fmt.Errorf("create temp dir: %w", err)
			}
			defer os.RemoveAll(tmpDir)

			archivePath := filepath.Join(tmpDir, name+".tar.gz")
			if err := os.WriteFile(archivePath, data, 0o644); err != nil {
				return fmt.Errorf("write archive: %w", err)
			}

			out.Step(3, 3, "Installing...")
			if err := reg.Install(name, *meta, tmpDir); err != nil {
				return fmt.Errorf("install failed: %w", err)
			}

			out.Success("Template %q installed!", name)
			out.Info("Use it with: dokrypt init my-project --template %s", name)
			return nil
		},
	}

	cmd.Flags().StringVar(&from, "from", "", "install from a local directory")
	return cmd
}

func installFromLocal(out interface{ Info(string, ...any); Step(int, int, string); Success(string, ...any) }, reg *marketplace.LocalRegistry, name, fromDir string) error {
	metaPath := filepath.Join(fromDir, "template.yaml")
	data, err := os.ReadFile(metaPath)
	if err != nil {
		return fmt.Errorf("no template.yaml found in %s: %w", fromDir, err)
	}

	var t template.Template
	if err := yaml.Unmarshal(data, &t); err != nil {
		return fmt.Errorf("invalid template.yaml: %w", err)
	}

	if t.Name == "" {
		t.Name = name
	}

	meta := marketplace.PackageMeta{
		Name:        t.Name,
		Version:     t.Version,
		Description: t.Description,
		Author:      t.Author,
		Category:    t.Category,
		Difficulty:  t.Difficulty,
		Tags:        t.Tags,
		Chains:      t.Chains,
		Services:    t.Services,
		License:     t.License,
		Premium:     t.Premium,
		Price:       t.Price,
	}

	out.Step(1, 2, fmt.Sprintf("Installing from %s...", fromDir))
	if err := reg.Install(name, meta, fromDir); err != nil {
		return fmt.Errorf("install failed: %w", err)
	}

	out.Step(2, 2, "Done!")
	out.Success("Template %q installed from local directory!", name)
	out.Info("Use it with: dokrypt init my-project --template %s", name)
	return nil
}

func newMarketplaceUninstallCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "uninstall <name>",
		Short: "Remove an installed marketplace template",
		Args:  requireArgs(1, "dokrypt marketplace uninstall <name>"),
		RunE: func(cmd *cobra.Command, args []string) error {
			out := getOutput()
			name := args[0]

			reg, err := marketplace.DefaultLocalRegistry()
			if err != nil {
				return err
			}

			if err := reg.Uninstall(name); err != nil {
				return err
			}

			out.Success("Template %q uninstalled", name)
			return nil
		},
	}
}

func newMarketplacePublishCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "publish",
		Short: "Publish a template to the marketplace hub",
		RunE: func(cmd *cobra.Command, args []string) error {
			out := getOutput()

			cwd, err := os.Getwd()
			if err != nil {
				return fmt.Errorf("get cwd: %w", err)
			}

			metaPath := filepath.Join(cwd, "template.yaml")
			data, err := os.ReadFile(metaPath)
			if err != nil {
				return fmt.Errorf("no template.yaml in current directory: %w", err)
			}

			var t template.Template
			if err := yaml.Unmarshal(data, &t); err != nil {
				return fmt.Errorf("invalid template.yaml: %w", err)
			}

			var missing []string
			if t.Name == "" {
				missing = append(missing, "name")
			}
			if t.Version == "" {
				missing = append(missing, "version")
			}
			if t.Description == "" {
				missing = append(missing, "description")
			}
			if len(missing) > 0 {
				return fmt.Errorf("template.yaml missing required fields: %s", strings.Join(missing, ", "))
			}

			out.Info("Publishing template to marketplace:")
			fmt.Println()
			out.Info("  Name:        %s", t.Name)
			out.Info("  Version:     %s", t.Version)
			out.Info("  Description: %s", t.Description)
			if t.Author != "" {
				out.Info("  Author:      %s", t.Author)
			}
			if t.Category != "" {
				out.Info("  Category:    %s", t.Category)
			}
			if len(t.Tags) > 0 {
				out.Info("  Tags:        %s", strings.Join(t.Tags, ", "))
			}
			if len(t.Chains) > 0 {
				out.Info("  Chains:      %s", strings.Join(t.Chains, ", "))
			}

			fmt.Println()

			client := marketplace.NewClient("")
			meta := marketplace.PackageMeta{
				Name:        t.Name,
				Version:     t.Version,
				Description: t.Description,
				Author:      t.Author,
				Category:    t.Category,
				Tags:        t.Tags,
				Chains:      t.Chains,
				Services:    t.Services,
				License:     t.License,
			}

			if err := client.Publish(meta, cwd, ""); err != nil {
				out.Warning("%v", err)
				return nil
			}

			out.Success("Template %q published!", t.Name)
			return nil
		},
	}
}

func newMarketplaceInfoCmd() *cobra.Command {
	var remote bool

	cmd := &cobra.Command{
		Use:   "info <name>",
		Short: "Show details for a marketplace template",
		Args:  requireArgs(1, "dokrypt marketplace info <name>"),
		RunE: func(cmd *cobra.Command, args []string) error {
			out := getOutput()
			name := args[0]

			if remote {
				client := marketplace.NewClient("")
				meta, err := client.GetInfo(name)
				if err != nil {
					return err
				}
				printPackageInfo(out, *meta)
				return nil
			}

			reg, err := marketplace.DefaultLocalRegistry()
			if err != nil {
				return err
			}

			pkg, err := reg.Get(name)
			if err != nil {
				out.Warning("Template %q is not installed locally", name)
				out.Info("Try: dokrypt marketplace info --remote %s", name)
				return nil
			}

			printPackageInfo(out, pkg.PackageMeta)
			out.Info("Installed: %s", pkg.InstalledAt.Format("2006-01-02 15:04:05"))
			out.Info("Path:      %s", pkg.Path)
			return nil
		},
	}

	cmd.Flags().BoolVar(&remote, "remote", false, "fetch info from the remote marketplace hub")
	return cmd
}

func newMarketplaceListCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "list",
		Short: "List installed marketplace templates",
		RunE: func(cmd *cobra.Command, args []string) error {
			out := getOutput()

			reg, err := marketplace.DefaultLocalRegistry()
			if err != nil {
				return err
			}

			installed := reg.List()
			if len(installed) == 0 {
				out.Info("No marketplace templates installed")
				fmt.Println()
				out.Info("Install templates with:")
				out.Info("  dokrypt marketplace install <name>")
				out.Info("  dokrypt marketplace install <name> --from ./local-dir")
				return nil
			}

			sort.Slice(installed, func(i, j int) bool {
				return installed[i].Name < installed[j].Name
			})

			out.Info("Installed marketplace templates:")
			fmt.Println()
			printInstalledTable(out, installed)
			return nil
		},
	}
}

type outputPrinter interface {
	Table(headers []string, rows [][]string)
	Info(format string, args ...any)
}

func printPackageTable(out outputPrinter, packages []marketplace.PackageMeta) {
	headers := []string{"Name", "Version", "Category", "Author", "Downloads", "Description"}
	rows := make([][]string, 0, len(packages))
	for _, p := range packages {
		cat := p.Category
		if cat == "" {
			cat = "-"
		}
		rows = append(rows, []string{
			p.Name, p.Version, cat, p.Author,
			fmt.Sprintf("%d", p.Downloads), p.Description,
		})
	}
	out.Table(headers, rows)
}

func printInstalledTable(out outputPrinter, packages []marketplace.InstalledPackage) {
	headers := []string{"Name", "Version", "Category", "Author", "Description"}
	rows := make([][]string, 0, len(packages))
	for _, p := range packages {
		cat := p.Category
		if cat == "" {
			cat = "-"
		}
		rows = append(rows, []string{
			p.Name, p.Version, cat, p.Author, p.Description,
		})
	}
	out.Table(headers, rows)
}

func printPackageInfo(out outputPrinter, meta marketplace.PackageMeta) {
	out.Info("Name:        %s", meta.Name)
	out.Info("Version:     %s", meta.Version)
	out.Info("Description: %s", meta.Description)
	if meta.Author != "" {
		out.Info("Author:      %s", meta.Author)
	}
	if meta.Category != "" {
		out.Info("Category:    %s", meta.Category)
	}
	if meta.Difficulty != "" {
		out.Info("Difficulty:  %s", meta.Difficulty)
	}
	if meta.License != "" {
		out.Info("License:     %s", meta.License)
	}
	if meta.Premium {
		out.Info("Premium:     yes (%s)", meta.Price)
	} else {
		out.Info("Premium:     no (free)")
	}
	if len(meta.Tags) > 0 {
		out.Info("Tags:        %s", strings.Join(meta.Tags, ", "))
	}
	if len(meta.Chains) > 0 {
		out.Info("Chains:      %s", strings.Join(meta.Chains, ", "))
	}
	if len(meta.Services) > 0 {
		out.Info("Services:    %s", strings.Join(meta.Services, ", "))
	}
	if meta.Homepage != "" {
		out.Info("Homepage:    %s", meta.Homepage)
	}
	if meta.Repository != "" {
		out.Info("Repository:  %s", meta.Repository)
	}
	if meta.Downloads > 0 {
		out.Info("Downloads:   %d", meta.Downloads)
	}
	if meta.Stars > 0 {
		out.Info("Stars:       %d", meta.Stars)
	}
}
