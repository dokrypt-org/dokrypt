package marketplace

import (
	"encoding/json"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
	"time"

	"gopkg.in/yaml.v3"
)

type LocalRegistry struct {
	dir string // ~/.dokrypt/marketplace
}

func NewLocalRegistry(dir string) *LocalRegistry {
	return &LocalRegistry{dir: dir}
}

func DefaultLocalRegistry() (*LocalRegistry, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return nil, fmt.Errorf("cannot determine home directory: %w", err)
	}
	dir := filepath.Join(home, ".dokrypt", "marketplace")
	return NewLocalRegistry(dir), nil
}

func (r *LocalRegistry) Dir() string {
	return r.dir
}

func (r *LocalRegistry) Install(name string, meta PackageMeta, sourceDir string) error {
	if name == "" {
		name = filepath.Base(meta.Name)
	}
	destDir := filepath.Join(r.dir, name)
	if err := os.MkdirAll(destDir, 0o755); err != nil {
		return fmt.Errorf("create install dir: %w", err)
	}

	if err := copyDir(sourceDir, destDir); err != nil {
		return fmt.Errorf("copy template files: %w", err)
	}

	installed := InstalledPackage{
		PackageMeta: meta,
		InstalledAt: time.Now(),
		Path:        destDir,
	}

	data, err := json.MarshalIndent(installed, "", "  ")
	if err != nil {
		return fmt.Errorf("marshal metadata: %w", err)
	}

	metaPath := filepath.Join(destDir, ".dokrypt-install.json")
	if err := os.WriteFile(metaPath, data, 0o644); err != nil {
		return fmt.Errorf("write install metadata: %w", err)
	}

	return nil
}

func (r *LocalRegistry) Uninstall(name string) error {
	dir := filepath.Join(r.dir, name)
	if _, err := os.Stat(dir); os.IsNotExist(err) {
		return fmt.Errorf("template %q is not installed", name)
	}
	return os.RemoveAll(dir)
}

func (r *LocalRegistry) Get(name string) (*InstalledPackage, error) {
	dir := filepath.Join(r.dir, name)
	metaPath := filepath.Join(dir, ".dokrypt-install.json")

	data, err := os.ReadFile(metaPath)
	if err != nil {
		return r.getFromTemplateYAML(name, dir)
	}

	var pkg InstalledPackage
	if err := json.Unmarshal(data, &pkg); err != nil {
		return nil, fmt.Errorf("invalid install metadata for %q: %w", name, err)
	}
	return &pkg, nil
}

func (r *LocalRegistry) getFromTemplateYAML(name, dir string) (*InstalledPackage, error) {
	yamlPath := filepath.Join(dir, "template.yaml")
	data, err := os.ReadFile(yamlPath)
	if err != nil {
		return nil, fmt.Errorf("template %q is not installed", name)
	}

	var meta PackageMeta
	if err := yaml.Unmarshal(data, &meta); err != nil {
		return nil, fmt.Errorf("invalid template.yaml for %q: %w", name, err)
	}

	info, _ := os.Stat(yamlPath)
	return &InstalledPackage{
		PackageMeta: meta,
		InstalledAt: info.ModTime(),
		Path:        dir,
	}, nil
}

func (r *LocalRegistry) List() []InstalledPackage {
	var result []InstalledPackage

	entries, err := os.ReadDir(r.dir)
	if err != nil {
		return result
	}

	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}
		pkg, err := r.Get(entry.Name())
		if err != nil {
			continue
		}
		result = append(result, *pkg)
	}

	return result
}

func (r *LocalRegistry) Search(query string) []InstalledPackage {
	query = strings.ToLower(query)
	all := r.List()
	var matches []InstalledPackage

	for _, pkg := range all {
		if matchesQuery(pkg.PackageMeta, query) {
			matches = append(matches, pkg)
		}
	}
	return matches
}

func (r *LocalRegistry) Browse(category string) []InstalledPackage {
	category = strings.ToLower(category)
	all := r.List()
	if category == "" {
		return all
	}

	var matches []InstalledPackage
	for _, pkg := range all {
		if strings.EqualFold(pkg.Category, category) {
			matches = append(matches, pkg)
		}
	}
	return matches
}

func (r *LocalRegistry) FS(name string) (fs.FS, error) {
	dir := filepath.Join(r.dir, name)
	if _, err := os.Stat(dir); os.IsNotExist(err) {
		return nil, fmt.Errorf("template %q is not installed", name)
	}
	return os.DirFS(dir), nil
}

func matchesQuery(meta PackageMeta, query string) bool {
	if strings.Contains(strings.ToLower(meta.Name), query) {
		return true
	}
	if strings.Contains(strings.ToLower(meta.Description), query) {
		return true
	}
	if strings.Contains(strings.ToLower(meta.Author), query) {
		return true
	}
	for _, tag := range meta.Tags {
		if strings.Contains(strings.ToLower(tag), query) {
			return true
		}
	}
	for _, chain := range meta.Chains {
		if strings.Contains(strings.ToLower(chain), query) {
			return true
		}
	}
	return false
}

func copyDir(src, dst string) error {
	return filepath.WalkDir(src, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}

		rel, err := filepath.Rel(src, path)
		if err != nil {
			return err
		}

		target := filepath.Join(dst, rel)

		if d.IsDir() {
			return os.MkdirAll(target, 0o755)
		}

		data, err := os.ReadFile(path)
		if err != nil {
			return err
		}

		return os.WriteFile(target, data, 0o644)
	})
}
