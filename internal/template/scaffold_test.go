package template

import (
	"io/fs"
	"os"
	"path/filepath"
	"testing"
	"testing/fstest"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestScaffoldCreatesProjectDirectory(t *testing.T) {
	outDir := t.TempDir()

	tmplFS := fstest.MapFS{
		"template.yaml": &fstest.MapFile{Data: []byte("name: test\n")},
		"README.md":     &fstest.MapFile{Data: []byte("# Hello")},
	}

	opts := ScaffoldOptions{
		Name: "my-project",
		Dir:  outDir,
	}

	err := Scaffold(opts, tmplFS)
	require.NoError(t, err)

	projectDir := filepath.Join(outDir, "my-project")
	info, err := os.Stat(projectDir)
	require.NoError(t, err)
	assert.True(t, info.IsDir())
}

func TestScaffoldCopiesPlainFiles(t *testing.T) {
	outDir := t.TempDir()

	tmplFS := fstest.MapFS{
		"template.yaml": &fstest.MapFile{Data: []byte("name: test\n")},
		"README.md":     &fstest.MapFile{Data: []byte("# My Project")},
		"config.json":   &fstest.MapFile{Data: []byte(`{"key": "value"}`)},
	}

	opts := ScaffoldOptions{
		Name: "proj",
		Dir:  outDir,
	}

	err := Scaffold(opts, tmplFS)
	require.NoError(t, err)

	projectDir := filepath.Join(outDir, "proj")

	data, err := os.ReadFile(filepath.Join(projectDir, "README.md"))
	require.NoError(t, err)
	assert.Equal(t, "# My Project", string(data))

	data, err = os.ReadFile(filepath.Join(projectDir, "config.json"))
	require.NoError(t, err)
	assert.Equal(t, `{"key": "value"}`, string(data))
}

func TestScaffoldSkipsTemplateYaml(t *testing.T) {
	outDir := t.TempDir()

	tmplFS := fstest.MapFS{
		"template.yaml": &fstest.MapFile{Data: []byte("name: test\n")},
		"file.txt":      &fstest.MapFile{Data: []byte("hello")},
	}

	opts := ScaffoldOptions{
		Name: "proj",
		Dir:  outDir,
	}

	err := Scaffold(opts, tmplFS)
	require.NoError(t, err)

	projectDir := filepath.Join(outDir, "proj")

	_, err = os.Stat(filepath.Join(projectDir, "template.yaml"))
	assert.True(t, os.IsNotExist(err), "template.yaml should not be copied to output")

	_, err = os.Stat(filepath.Join(projectDir, "file.txt"))
	assert.NoError(t, err)
}

func TestScaffoldCreatesSubdirectories(t *testing.T) {
	outDir := t.TempDir()

	tmplFS := fstest.MapFS{
		"template.yaml":    &fstest.MapFile{Data: []byte("name: test\n")},
		"src":              &fstest.MapFile{Mode: fs.ModeDir},
		"src/main.go":      &fstest.MapFile{Data: []byte("package main")},
		"contracts":        &fstest.MapFile{Mode: fs.ModeDir},
		"contracts/A.sol":  &fstest.MapFile{Data: []byte("// SPDX")},
	}

	opts := ScaffoldOptions{
		Name: "proj",
		Dir:  outDir,
	}

	err := Scaffold(opts, tmplFS)
	require.NoError(t, err)

	projectDir := filepath.Join(outDir, "proj")

	info, err := os.Stat(filepath.Join(projectDir, "src"))
	require.NoError(t, err)
	assert.True(t, info.IsDir())

	info, err = os.Stat(filepath.Join(projectDir, "contracts"))
	require.NoError(t, err)
	assert.True(t, info.IsDir())

	data, err := os.ReadFile(filepath.Join(projectDir, "src", "main.go"))
	require.NoError(t, err)
	assert.Equal(t, "package main", string(data))

	data, err = os.ReadFile(filepath.Join(projectDir, "contracts", "A.sol"))
	require.NoError(t, err)
	assert.Equal(t, "// SPDX", string(data))
}

func TestScaffoldRendersTmplFiles(t *testing.T) {
	outDir := t.TempDir()

	tmplFS := fstest.MapFS{
		"template.yaml": &fstest.MapFile{Data: []byte("name: test\n")},
		"README.md.tmpl": &fstest.MapFile{
			Data: []byte("# {{ .ProjectName }}\nBy {{ .Author }}\nChain: {{ .ChainName }} ({{ .ChainID }})\nEngine: {{ .Engine }}"),
		},
	}

	opts := ScaffoldOptions{
		Name: "proj",
		Dir:  outDir,
		Vars: Vars{
			ProjectName: "My Cool Project",
			Author:      "Developer",
			ChainName:   "ethereum",
			ChainID:     1,
			Engine:      "hardhat",
		},
	}

	err := Scaffold(opts, tmplFS)
	require.NoError(t, err)

	projectDir := filepath.Join(outDir, "proj")

	_, err = os.Stat(filepath.Join(projectDir, "README.md.tmpl"))
	assert.True(t, os.IsNotExist(err), ".tmpl file should not exist in output")

	data, err := os.ReadFile(filepath.Join(projectDir, "README.md"))
	require.NoError(t, err)

	content := string(data)
	assert.Contains(t, content, "# My Cool Project")
	assert.Contains(t, content, "By Developer")
	assert.Contains(t, content, "Chain: ethereum (1)")
	assert.Contains(t, content, "Engine: hardhat")
}

func TestScaffoldTmplWithInvalidTemplate(t *testing.T) {
	outDir := t.TempDir()

	tmplFS := fstest.MapFS{
		"template.yaml":  &fstest.MapFile{Data: []byte("name: test\n")},
		"broken.txt.tmpl": &fstest.MapFile{Data: []byte("{{ .Invalid {{")},
	}

	opts := ScaffoldOptions{
		Name: "proj",
		Dir:  outDir,
		Vars: Vars{ProjectName: "test"},
	}

	err := Scaffold(opts, tmplFS)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "render")
}

func TestScaffoldTmplWithMissingField(t *testing.T) {
	outDir := t.TempDir()

	tmplFS := fstest.MapFS{
		"template.yaml":  &fstest.MapFile{Data: []byte("name: test\n")},
		"config.txt.tmpl": &fstest.MapFile{Data: []byte("project={{ .ProjectName }}")},
	}

	opts := ScaffoldOptions{
		Name: "proj",
		Dir:  outDir,
		Vars: Vars{}, // Empty vars
	}

	err := Scaffold(opts, tmplFS)
	require.NoError(t, err)

	data, err := os.ReadFile(filepath.Join(outDir, "proj", "config.txt"))
	require.NoError(t, err)
	assert.Equal(t, "project=", string(data))
}

func TestScaffoldAbsolutePath(t *testing.T) {
	outDir := t.TempDir()
	projectDir := filepath.Join(outDir, "abs-project")

	tmplFS := fstest.MapFS{
		"template.yaml": &fstest.MapFile{Data: []byte("name: test\n")},
		"file.txt":      &fstest.MapFile{Data: []byte("content")},
	}

	opts := ScaffoldOptions{
		Name: projectDir, // Absolute path as name
	}

	err := Scaffold(opts, tmplFS)
	require.NoError(t, err)

	data, err := os.ReadFile(filepath.Join(projectDir, "file.txt"))
	require.NoError(t, err)
	assert.Equal(t, "content", string(data))
}

func TestScaffoldRelativePathWithDir(t *testing.T) {
	outDir := t.TempDir()

	tmplFS := fstest.MapFS{
		"template.yaml": &fstest.MapFile{Data: []byte("name: test\n")},
		"hello.txt":     &fstest.MapFile{Data: []byte("hello")},
	}

	opts := ScaffoldOptions{
		Name: "relative-proj",
		Dir:  outDir,
	}

	err := Scaffold(opts, tmplFS)
	require.NoError(t, err)

	data, err := os.ReadFile(filepath.Join(outDir, "relative-proj", "hello.txt"))
	require.NoError(t, err)
	assert.Equal(t, "hello", string(data))
}

func TestScaffoldEmptyFS(t *testing.T) {
	outDir := t.TempDir()

	tmplFS := fstest.MapFS{
		"template.yaml": &fstest.MapFile{Data: []byte("name: test\n")},
	}

	opts := ScaffoldOptions{
		Name: "empty-proj",
		Dir:  outDir,
	}

	err := Scaffold(opts, tmplFS)
	require.NoError(t, err)

	projectDir := filepath.Join(outDir, "empty-proj")
	info, err := os.Stat(projectDir)
	require.NoError(t, err)
	assert.True(t, info.IsDir())
}

func TestScaffoldWithBuiltinFS(t *testing.T) {
	outDir := t.TempDir()

	m := NewManager(t.TempDir())
	tmplFS, err := m.GetFS("evm-basic")
	require.NoError(t, err)

	opts := ScaffoldOptions{
		Name: "builtin-proj",
		Dir:  outDir,
		Vars: Vars{
			ProjectName: "builtin-proj",
			ChainName:   "ethereum",
			ChainID:     1,
			Engine:      "foundry",
			Author:      "tester",
		},
	}

	err = Scaffold(opts, tmplFS)
	require.NoError(t, err)

	projectDir := filepath.Join(outDir, "builtin-proj")

	info, err := os.Stat(filepath.Join(projectDir, "contracts"))
	require.NoError(t, err)
	assert.True(t, info.IsDir())

	_, err = os.Stat(filepath.Join(projectDir, "contracts", "Counter.sol"))
	assert.NoError(t, err)

	_, err = os.Stat(filepath.Join(projectDir, "template.yaml"))
	assert.True(t, os.IsNotExist(err))

	data, err := os.ReadFile(filepath.Join(projectDir, "README.md"))
	require.NoError(t, err)
	assert.Contains(t, string(data), "builtin-proj")
}

func TestScaffoldMultipleTmplFiles(t *testing.T) {
	outDir := t.TempDir()

	tmplFS := fstest.MapFS{
		"template.yaml": &fstest.MapFile{Data: []byte("name: test\n")},
		"a.txt.tmpl":    &fstest.MapFile{Data: []byte("A={{ .ProjectName }}")},
		"b.txt.tmpl":    &fstest.MapFile{Data: []byte("B={{ .Author }}")},
		"c.txt":         &fstest.MapFile{Data: []byte("C=plain")},
	}

	opts := ScaffoldOptions{
		Name: "multi",
		Dir:  outDir,
		Vars: Vars{ProjectName: "proj", Author: "dev"},
	}

	err := Scaffold(opts, tmplFS)
	require.NoError(t, err)

	projectDir := filepath.Join(outDir, "multi")

	data, err := os.ReadFile(filepath.Join(projectDir, "a.txt"))
	require.NoError(t, err)
	assert.Equal(t, "A=proj", string(data))

	data, err = os.ReadFile(filepath.Join(projectDir, "b.txt"))
	require.NoError(t, err)
	assert.Equal(t, "B=dev", string(data))

	data, err = os.ReadFile(filepath.Join(projectDir, "c.txt"))
	require.NoError(t, err)
	assert.Equal(t, "C=plain", string(data))
}

func TestScaffoldNestedTmplFile(t *testing.T) {
	outDir := t.TempDir()

	tmplFS := fstest.MapFS{
		"template.yaml":        &fstest.MapFile{Data: []byte("name: test\n")},
		"config":               &fstest.MapFile{Mode: fs.ModeDir},
		"config/app.yaml.tmpl": &fstest.MapFile{Data: []byte("name: {{ .ProjectName }}")},
	}

	opts := ScaffoldOptions{
		Name: "nested",
		Dir:  outDir,
		Vars: Vars{ProjectName: "nested-proj"},
	}

	err := Scaffold(opts, tmplFS)
	require.NoError(t, err)

	data, err := os.ReadFile(filepath.Join(outDir, "nested", "config", "app.yaml"))
	require.NoError(t, err)
	assert.Equal(t, "name: nested-proj", string(data))
}

func TestScaffoldOptionsStruct(t *testing.T) {
	opts := ScaffoldOptions{
		Name:     "my-project",
		Template: "evm-basic",
		Dir:      "/output",
		Chain:    "ethereum",
		Engine:   "hardhat",
		ChainID:  1,
		Services: []string{"ipfs", "subgraph"},
		NoGit:    true,
		Vars: Vars{
			ProjectName: "my-project",
			ChainName:   "ethereum",
		},
	}

	assert.Equal(t, "my-project", opts.Name)
	assert.Equal(t, "evm-basic", opts.Template)
	assert.Equal(t, "/output", opts.Dir)
	assert.Equal(t, "ethereum", opts.Chain)
	assert.Equal(t, "hardhat", opts.Engine)
	assert.Equal(t, uint64(1), opts.ChainID)
	assert.Equal(t, []string{"ipfs", "subgraph"}, opts.Services)
	assert.True(t, opts.NoGit)
	assert.Equal(t, "my-project", opts.Vars.ProjectName)
}

func TestScaffoldOverwritesExistingDir(t *testing.T) {
	outDir := t.TempDir()
	projectDir := filepath.Join(outDir, "existing")

	require.NoError(t, os.MkdirAll(projectDir, 0o755))
	require.NoError(t, os.WriteFile(filepath.Join(projectDir, "old.txt"), []byte("old"), 0o644))

	tmplFS := fstest.MapFS{
		"template.yaml": &fstest.MapFile{Data: []byte("name: test\n")},
		"new.txt":       &fstest.MapFile{Data: []byte("new")},
	}

	opts := ScaffoldOptions{
		Name: "existing",
		Dir:  outDir,
	}

	err := Scaffold(opts, tmplFS)
	require.NoError(t, err)

	data, err := os.ReadFile(filepath.Join(projectDir, "new.txt"))
	require.NoError(t, err)
	assert.Equal(t, "new", string(data))

	data, err = os.ReadFile(filepath.Join(projectDir, "old.txt"))
	require.NoError(t, err)
	assert.Equal(t, "old", string(data))
}

func TestScaffoldDirNotSet(t *testing.T) {
	outDir := t.TempDir()
	projectDir := filepath.Join(outDir, "no-dir-proj")

	tmplFS := fstest.MapFS{
		"template.yaml": &fstest.MapFile{Data: []byte("name: test\n")},
		"file.txt":      &fstest.MapFile{Data: []byte("data")},
	}

	opts := ScaffoldOptions{
		Name: projectDir, // absolute path
	}

	err := Scaffold(opts, tmplFS)
	require.NoError(t, err)

	data, err := os.ReadFile(filepath.Join(projectDir, "file.txt"))
	require.NoError(t, err)
	assert.Equal(t, "data", string(data))
}
