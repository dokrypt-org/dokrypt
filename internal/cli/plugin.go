package cli

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/spf13/cobra"
	"gopkg.in/yaml.v3"

	"github.com/dokrypt/dokrypt/internal/plugin"
)

func newPluginCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "plugin",
		Short: "Plugin management",
	}

	cmd.AddCommand(
		newPluginInstallCmd(),
		newPluginUninstallCmd(),
		newPluginListCmd(),
		newPluginSearchCmd(),
		newPluginUpdateCmd(),
		newPluginCreateCmd(),
		newPluginPublishCmd(),
	)

	return cmd
}

func newPluginInstallCmd() *cobra.Command {
	var (
		version string
		global  bool
	)

	cmd := &cobra.Command{
		Use:   "install <name>",
		Short: "Install a plugin",
		Args:  requireArgs(1, "dokrypt plugin install <name>"),
		RunE: func(cmd *cobra.Command, args []string) error {
			out := getOutput()
			name := args[0]

			mgr, err := plugin.DefaultManager(".")
			if err != nil {
				return fmt.Errorf("failed to create plugin manager: %w", err)
			}

			if err := mgr.Discover(); err != nil {
				return fmt.Errorf("failed to discover plugins: %w", err)
			}

			if info, _ := mgr.Get(name); info != nil {
				out.Warning("Plugin %q is already installed (version %s)", name, info.Manifest.Version)
				return nil
			}

			if version == "" {
				version = "latest"
			}

			scope := "locally"
			if global {
				scope = "globally"
			}

			out.Info("Installing plugin %q version %s %s...", name, version, scope)

			ctx := cmd.Context()
			if err := mgr.Install(ctx, name, version, global); err != nil {
				return fmt.Errorf("failed to install plugin %q: %w", name, err)
			}

			out.Success("Plugin %q installed successfully", name)
			return nil
		},
	}

	cmd.Flags().StringVar(&version, "version", "", "plugin version (default: latest)")
	cmd.Flags().BoolVar(&global, "global", false, "install globally (~/.dokrypt/plugins)")

	return cmd
}

func newPluginUninstallCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "uninstall <name>",
		Short: "Remove a plugin",
		Args:  requireArgs(1, "dokrypt plugin uninstall <name>"),
		RunE: func(cmd *cobra.Command, args []string) error {
			out := getOutput()
			name := args[0]

			mgr, err := plugin.DefaultManager(".")
			if err != nil {
				return fmt.Errorf("failed to create plugin manager: %w", err)
			}

			if err := mgr.Discover(); err != nil {
				return fmt.Errorf("failed to discover plugins: %w", err)
			}

			if _, err := mgr.Get(name); err != nil {
				return fmt.Errorf("plugin %q is not installed", name)
			}

			out.Info("Uninstalling plugin %q...", name)

			ctx := cmd.Context()
			if err := mgr.Uninstall(ctx, name); err != nil {
				return fmt.Errorf("failed to uninstall plugin %q: %w", name, err)
			}

			out.Success("Plugin %q removed successfully", name)
			return nil
		},
	}
}

func newPluginListCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "list",
		Short: "List installed plugins",
		RunE: func(cmd *cobra.Command, args []string) error {
			out := getOutput()

			mgr, err := plugin.DefaultManager(".")
			if err != nil {
				return fmt.Errorf("failed to create plugin manager: %w", err)
			}

			if err := mgr.Discover(); err != nil {
				return fmt.Errorf("failed to discover plugins: %w", err)
			}

			plugins := mgr.List()
			if len(plugins) == 0 {
				out.Info("No plugins installed. Use 'dokrypt plugin install <name>' to add one.")
				return nil
			}

			headers := []string{"Name", "Version", "Type", "Description", "Scope"}
			rows := make([][]string, 0, len(plugins))

			for _, p := range plugins {
				scope := "local"
				if p.Global {
					scope = "global"
				}

				pluginType := p.Manifest.Type
				if pluginType == "" {
					pluginType = "unknown"
				}

				rows = append(rows, []string{
					p.Manifest.Name,
					p.Manifest.Version,
					pluginType,
					p.Manifest.Description,
					scope,
				})
			}

			out.Table(headers, rows)
			return nil
		},
	}
}

func newPluginSearchCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "search <query>",
		Short: "Search for plugins",
		Args:  requireArgs(1, "dokrypt plugin search <query>"),
		RunE: func(cmd *cobra.Command, args []string) error {
			out := getOutput()
			query := strings.ToLower(args[0])

			headers := []string{"Name", "Version", "Type", "Description", "Source"}
			var rows [][]string

			registry := plugin.DefaultRegistryClient()
			remoteResults, err := registry.Search(cmd.Context(), args[0])
			if err != nil {
				out.Warning("Registry search failed: %v", err)
				out.Info("Falling back to local search...")
			} else {
				for _, m := range remoteResults {
					pluginType := m.Type
					if pluginType == "" {
						pluginType = "unknown"
					}
					rows = append(rows, []string{
						m.Name,
						m.Version,
						pluginType,
						m.Description,
						"registry",
					})
				}
			}

			mgr, err := plugin.DefaultManager(".")
			if err != nil {
				return fmt.Errorf("failed to create plugin manager: %w", err)
			}

			if err := mgr.Discover(); err != nil {
				return fmt.Errorf("failed to discover plugins: %w", err)
			}

			plugins := mgr.List()
			for _, p := range plugins {
				nameMatch := strings.Contains(strings.ToLower(p.Manifest.Name), query)
				descMatch := strings.Contains(strings.ToLower(p.Manifest.Description), query)
				authorMatch := strings.Contains(strings.ToLower(p.Manifest.Author), query)

				if nameMatch || descMatch || authorMatch {
					scope := "local"
					if p.Global {
						scope = "global"
					}

					pluginType := p.Manifest.Type
					if pluginType == "" {
						pluginType = "unknown"
					}

					rows = append(rows, []string{
						p.Manifest.Name,
						p.Manifest.Version,
						pluginType,
						p.Manifest.Description,
						scope,
					})
				}
			}

			if len(rows) == 0 {
				out.Info("No plugins matching %q found.", args[0])
				return nil
			}

			out.Success("Found %d plugin(s) matching %q:", len(rows), args[0])
			out.Table(headers, rows)
			return nil
		},
	}
}

func newPluginUpdateCmd() *cobra.Command {
	var version string

	cmd := &cobra.Command{
		Use:   "update <name>",
		Short: "Update a plugin",
		Args:  requireArgs(1, "dokrypt plugin update <name>"),
		RunE: func(cmd *cobra.Command, args []string) error {
			out := getOutput()
			name := args[0]

			mgr, err := plugin.DefaultManager(".")
			if err != nil {
				return fmt.Errorf("failed to create plugin manager: %w", err)
			}

			if err := mgr.Discover(); err != nil {
				return fmt.Errorf("failed to discover plugins: %w", err)
			}

			info, err := mgr.Get(name)
			if err != nil {
				return fmt.Errorf("plugin %q is not installed; install it first with 'dokrypt plugin install %s'", name, name)
			}

			currentVersion := info.Manifest.Version
			global := info.Global

			if version == "" {
				version = "latest"
			}

			out.Info("Updating plugin %q from version %s to %s...", name, currentVersion, version)

			ctx := cmd.Context()

			if err := mgr.Uninstall(ctx, name); err != nil {
				return fmt.Errorf("failed to remove old version of plugin %q: %w", name, err)
			}

			if err := mgr.Install(ctx, name, version, global); err != nil {
				return fmt.Errorf("failed to install new version of plugin %q: %w", name, err)
			}

			out.Success("Plugin %q updated successfully", name)
			return nil
		},
	}

	cmd.Flags().StringVar(&version, "version", "", "target version (default: latest)")

	return cmd
}

func newPluginCreateCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "create <name>",
		Short: "Scaffold a new plugin",
		Args:  requireArgs(1, "dokrypt plugin create <name>"),
		RunE: func(cmd *cobra.Command, args []string) error {
			out := getOutput()
			name := args[0]

			if strings.ContainsAny(name, " /\\:*?\"<>|") {
				return fmt.Errorf("invalid plugin name %q: must not contain spaces or special characters", name)
			}

			if _, err := os.Stat(name); err == nil {
				return fmt.Errorf("directory %q already exists", name)
			}

			out.Info("Scaffolding plugin %q...", name)

			if err := os.MkdirAll(name, 0o755); err != nil {
				return fmt.Errorf("failed to create plugin directory: %w", err)
			}

			manifest := plugin.Manifest{
				Name:        name,
				Version:     "0.1.0",
				Description: fmt.Sprintf("A Dokrypt plugin: %s", name),
				Author:      "",
				License:     "MIT",
				Type:        "container",
				Commands: []plugin.CommandDef{
					{
						Name:        name,
						Description: fmt.Sprintf("Run the %s plugin", name),
					},
				},
			}

			manifestData, err := yaml.Marshal(&manifest)
			if err != nil {
				return fmt.Errorf("failed to serialize manifest: %w", err)
			}

			manifestPath := filepath.Join(name, "plugin.yaml")
			if err := os.WriteFile(manifestPath, manifestData, 0o644); err != nil {
				return fmt.Errorf("failed to write %s: %w", manifestPath, err)
			}

			dockerfile := fmt.Sprintf(`FROM alpine:3.19

LABEL maintainer="your-name"
LABEL description="Dokrypt plugin: %s"

RUN apk add --no-cache ca-certificates

WORKDIR /plugin

COPY . .

ENTRYPOINT ["/bin/sh"]
`, name)

			dockerfilePath := filepath.Join(name, "Dockerfile")
			if err := os.WriteFile(dockerfilePath, []byte(dockerfile), 0o644); err != nil {
				return fmt.Errorf("failed to write %s: %w", dockerfilePath, err)
			}

			readme := fmt.Sprintf(`# %s

A Dokrypt plugin.

## Installation

`+"```bash"+`
dokrypt plugin install %s
`+"```"+`

## Usage

`+"```bash"+`
dokrypt %s
`+"```"+`

## Development

1. Edit `+"`plugin.yaml`"+` to configure your plugin.
2. Build the container: `+"`docker build -t %s .`"+`
3. Test locally: `+"`dokrypt plugin install . --version dev`"+`

## License

MIT
`, name, name, name, name)

			readmePath := filepath.Join(name, "README.md")
			if err := os.WriteFile(readmePath, []byte(readme), 0o644); err != nil {
				return fmt.Errorf("failed to write %s: %w", readmePath, err)
			}

			out.Success("Plugin %q scaffolded successfully", name)
			out.Info("Created files:")
			out.Info("  %s", manifestPath)
			out.Info("  %s", dockerfilePath)
			out.Info("  %s", readmePath)
			out.Info("")
			out.Info("Next steps:")
			out.Info("  cd %s", name)
			out.Info("  # Edit plugin.yaml and Dockerfile")
			out.Info("  dokrypt plugin publish")
			return nil
		},
	}
}

func newPluginPublishCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "publish",
		Short: "Publish plugin to registry",
		RunE: func(cmd *cobra.Command, args []string) error {
			out := getOutput()

			manifestPath := "plugin.yaml"
			data, err := os.ReadFile(manifestPath)
			if err != nil {
				return fmt.Errorf("no plugin.yaml found in current directory; run this command from a plugin directory")
			}

			var manifest plugin.Manifest
			if err := yaml.Unmarshal(data, &manifest); err != nil {
				return fmt.Errorf("invalid plugin.yaml: %w", err)
			}

			var missing []string
			if manifest.Name == "" {
				missing = append(missing, "name")
			}
			if manifest.Version == "" {
				missing = append(missing, "version")
			}
			if manifest.Description == "" {
				missing = append(missing, "description")
			}
			if manifest.Type == "" {
				missing = append(missing, "type")
			}

			if len(missing) > 0 {
				return fmt.Errorf("plugin.yaml is missing required fields: %s", strings.Join(missing, ", "))
			}

			if manifest.Type != "container" && manifest.Type != "binary" {
				out.Warning("Unrecognized plugin type %q (expected \"container\" or \"binary\")", manifest.Type)
			}

			out.Info("Plugin publish preview:")
			out.Info("")

			details := [][]string{
				{"Name", manifest.Name},
				{"Version", manifest.Version},
				{"Description", manifest.Description},
				{"Author", valueOrDefault(manifest.Author, "(not set)")},
				{"License", valueOrDefault(manifest.License, "(not set)")},
				{"Type", manifest.Type},
			}

			if len(manifest.Commands) > 0 {
				cmdNames := make([]string, 0, len(manifest.Commands))
				for _, c := range manifest.Commands {
					cmdNames = append(cmdNames, c.Name)
				}
				details = append(details, []string{"Commands", strings.Join(cmdNames, ", ")})
			}

			out.Table([]string{"Field", "Value"}, details)
			out.Info("")

			registry := plugin.DefaultRegistryClient()
			token := os.Getenv("DOKRYPT_REGISTRY_TOKEN")
			if err := registry.Publish(cmd.Context(), manifest, ".", token); err != nil {
				return fmt.Errorf("publish failed: %w", err)
			}

			out.Success("Plugin %q v%s published successfully", manifest.Name, manifest.Version)
			return nil
		},
	}
}

func valueOrDefault(value, fallback string) string {
	if value == "" {
		return fallback
	}
	return value
}
