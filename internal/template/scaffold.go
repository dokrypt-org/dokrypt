package template

import (
	"bytes"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
	gotemplate "text/template"
)

type ScaffoldOptions struct {
	Name     string
	Template string
	Dir      string
	Chain    string
	Engine   string
	ChainID  uint64
	Services []string
	NoGit    bool
	Vars     Vars
}

func Scaffold(opts ScaffoldOptions, tmplFS fs.FS) error {
	projectDir := opts.Name
	if !filepath.IsAbs(projectDir) && opts.Dir != "" {
		projectDir = filepath.Join(opts.Dir, opts.Name)
	}
	if err := os.MkdirAll(projectDir, 0o755); err != nil {
		return fmt.Errorf("create project dir: %w", err)
	}

	return fs.WalkDir(tmplFS, ".", func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if path == "." || path == "template.yaml" {
			return nil
		}

		target := filepath.Join(projectDir, path)

		if d.IsDir() {
			return os.MkdirAll(target, 0o755)
		}

		data, err := fs.ReadFile(tmplFS, path)
		if err != nil {
			return fmt.Errorf("read %s: %w", path, err)
		}

		if strings.HasSuffix(path, ".tmpl") {
			target = strings.TrimSuffix(target, ".tmpl")
			rendered, renderErr := renderTemplate(path, data, opts.Vars)
			if renderErr != nil {
				return fmt.Errorf("render %s: %w", path, renderErr)
			}
			data = rendered
		}

		return os.WriteFile(target, data, 0o644)
	})
}

func renderTemplate(name string, data []byte, vars Vars) ([]byte, error) {
	tmpl, err := gotemplate.New(name).Parse(string(data))
	if err != nil {
		return nil, err
	}
	var buf bytes.Buffer
	if err := tmpl.Execute(&buf, vars); err != nil {
		return nil, err
	}
	return buf.Bytes(), nil
}
