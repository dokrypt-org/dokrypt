package plugin

import (
	"archive/tar"
	"bytes"
	"compress/gzip"
	"context"
	"fmt"
	"io"
	"log/slog"
	"os"
	"path/filepath"
	"strings"

	"gopkg.in/yaml.v3"
)

type Manager struct {
	globalDir  string // ~/.dokrypt/plugins
	localDir   string // ./plugins (project-local)
	plugins    map[string]*Info
	loaded     map[string]Plugin
	registry   *RegistryClient
}

func NewManager(globalDir, localDir string, registry *RegistryClient) *Manager {
	if registry == nil {
		registry = DefaultRegistryClient()
	}
	return &Manager{
		globalDir: globalDir,
		localDir:  localDir,
		plugins:   make(map[string]*Info),
		loaded:    make(map[string]Plugin),
		registry:  registry,
	}
}

func DefaultManager(projectDir string) (*Manager, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return nil, fmt.Errorf("cannot determine home directory: %w", err)
	}
	globalDir := filepath.Join(home, ".dokrypt", "plugins")
	localDir := filepath.Join(projectDir, "plugins")
	return NewManager(globalDir, localDir, nil), nil
}

func (m *Manager) Discover() error {
	if err := m.scanDir(m.localDir, false); err != nil {
		slog.Debug("no local plugins", "error", err)
	}

	if err := m.scanDir(m.globalDir, true); err != nil {
		slog.Debug("no global plugins", "error", err)
	}

	slog.Info("discovered plugins", "count", len(m.plugins))
	return nil
}

func (m *Manager) scanDir(dir string, global bool) error {
	entries, err := os.ReadDir(dir)
	if err != nil {
		return err
	}

	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}

		manifestPath := filepath.Join(dir, entry.Name(), "plugin.yaml")
		data, err := os.ReadFile(manifestPath)
		if err != nil {
			continue
		}

		var manifest Manifest
		if err := yaml.Unmarshal(data, &manifest); err != nil {
			slog.Warn("invalid plugin manifest", "path", manifestPath, "error", err)
			continue
		}

		if _, exists := m.plugins[manifest.Name]; exists && global {
			continue
		}

		m.plugins[manifest.Name] = &Info{
			Manifest:  manifest,
			Installed: true,
			Path:      filepath.Join(dir, entry.Name()),
			Global:    global,
		}
	}
	return nil
}

func (m *Manager) List() []*Info {
	result := make([]*Info, 0, len(m.plugins))
	for _, p := range m.plugins {
		result = append(result, p)
	}
	return result
}

func (m *Manager) Get(name string) (*Info, error) {
	info, ok := m.plugins[name]
	if !ok {
		return nil, fmt.Errorf("plugin %q not found", name)
	}
	return info, nil
}

func (m *Manager) Install(ctx context.Context, name string, version string, global bool) error {
	targetDir := m.localDir
	if global {
		targetDir = m.globalDir
	}

	pluginDir := filepath.Join(targetDir, name)
	if err := os.MkdirAll(pluginDir, 0o755); err != nil {
		return fmt.Errorf("failed to create plugin directory: %w", err)
	}

	slog.Info("downloading plugin from registry", "name", name, "version", version)
	archive, err := m.registry.Download(ctx, name, version)
	if err != nil {
		return fmt.Errorf("failed to download plugin %s@%s: %w", name, version, err)
	}

	if err := extractTarGz(archive, pluginDir); err != nil {
		return fmt.Errorf("failed to extract plugin archive: %w", err)
	}

	manifestPath := filepath.Join(pluginDir, "plugin.yaml")
	manifestData, err := os.ReadFile(manifestPath)
	if err != nil {
		return fmt.Errorf("plugin archive missing plugin.yaml: %w", err)
	}

	var manifest Manifest
	if err := yaml.Unmarshal(manifestData, &manifest); err != nil {
		return fmt.Errorf("invalid plugin manifest: %w", err)
	}

	m.plugins[manifest.Name] = &Info{
		Manifest:  manifest,
		Installed: true,
		Path:      pluginDir,
		Global:    global,
	}

	slog.Info("installed plugin", "name", name, "version", version, "global", global)
	return nil
}

func extractTarGz(data []byte, destDir string) error {
	gz, err := gzip.NewReader(bytes.NewReader(data))
	if err != nil {
		return fmt.Errorf("failed to create gzip reader: %w", err)
	}
	defer gz.Close()

	tr := tar.NewReader(gz)
	for {
		header, err := tr.Next()
		if err == io.EOF {
			break
		}
		if err != nil {
			return fmt.Errorf("failed to read tar entry: %w", err)
		}

		target := filepath.Join(destDir, filepath.Clean(header.Name))

		rel, err := filepath.Rel(destDir, target)
		if err != nil || strings.HasPrefix(rel, "..") {
			return fmt.Errorf("archive entry %q attempts path traversal", header.Name)
		}

		switch header.Typeflag {
		case tar.TypeDir:
			if err := os.MkdirAll(target, 0o755); err != nil {
				return fmt.Errorf("failed to create directory %s: %w", target, err)
			}
		case tar.TypeReg:
			if err := os.MkdirAll(filepath.Dir(target), 0o755); err != nil {
				return fmt.Errorf("failed to create parent directory for %s: %w", target, err)
			}
			f, err := os.OpenFile(target, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, os.FileMode(header.Mode))
			if err != nil {
				return fmt.Errorf("failed to create file %s: %w", target, err)
			}
			if _, err := io.Copy(f, tr); err != nil {
				f.Close()
				return fmt.Errorf("failed to write file %s: %w", target, err)
			}
			f.Close()
		}
	}
	return nil
}

func (m *Manager) Uninstall(ctx context.Context, name string) error {
	info, err := m.Get(name)
	if err != nil {
		return err
	}

	if err := os.RemoveAll(info.Path); err != nil {
		return fmt.Errorf("failed to remove plugin: %w", err)
	}

	delete(m.plugins, name)
	delete(m.loaded, name)
	slog.Info("uninstalled plugin", "name", name)
	return nil
}

func (m *Manager) InitAll(ctx context.Context, env Environment) error {
	for name, p := range m.loaded {
		if err := p.OnInit(ctx, env); err != nil {
			slog.Warn("plugin init failed", "plugin", name, "error", err)
		}
	}
	return nil
}
