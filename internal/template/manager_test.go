package template

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewManager(t *testing.T) {
	m := NewManager("/tmp/test-templates")
	require.NotNil(t, m)
	assert.Equal(t, "/tmp/test-templates", m.globalDir)
	assert.NotNil(t, m.builtins)
}

func TestNewManagerRegistersBuiltins(t *testing.T) {
	m := NewManager(t.TempDir())
	expectedNames := []string{"evm-basic", "evm-defi", "evm-nft", "evm-dao", "evm-token", "evm-arbitrum"}

	for _, name := range expectedNames {
		info, ok := m.builtins[name]
		assert.True(t, ok, "builtin %q should be registered", name)
		assert.NotNil(t, info)
		assert.Equal(t, name, info.Template.Name)
		assert.True(t, info.BuiltIn)
	}
	assert.Len(t, m.builtins, len(expectedNames))
}

func TestNewManagerBuiltinMetadata(t *testing.T) {
	m := NewManager(t.TempDir())

	tests := []struct {
		name       string
		version    string
		category   string
		difficulty string
		premium    bool
		price      string
	}{
		{"evm-basic", "2.0.0", "basic", "beginner", false, "free"},
		{"evm-defi", "2.0.0", "defi", "advanced", false, "free"},
		{"evm-nft", "2.0.0", "nft", "intermediate", false, "free"},
		{"evm-dao", "2.0.0", "dao", "advanced", false, "free"},
		{"evm-token", "2.0.0", "token", "intermediate", false, "free"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			info := m.builtins[tt.name]
			require.NotNil(t, info)
			assert.Equal(t, tt.version, info.Template.Version)
			assert.Equal(t, tt.category, info.Template.Category)
			assert.Equal(t, tt.difficulty, info.Template.Difficulty)
			assert.Equal(t, tt.premium, info.Template.Premium)
			assert.Equal(t, tt.price, info.Template.Price)
			assert.Equal(t, "dokrypt", info.Template.Author)
		})
	}
}

func TestDefaultManager(t *testing.T) {
	m, err := DefaultManager()
	require.NoError(t, err)
	require.NotNil(t, m)

	home, _ := os.UserHomeDir()
	expected := filepath.Join(home, ".dokrypt", "templates")
	assert.Equal(t, expected, m.globalDir)
	assert.NotEmpty(t, m.builtins)
}

func TestManagerListBuiltinsOnly(t *testing.T) {
	m := NewManager(filepath.Join(t.TempDir(), "nonexistent"))
	list := m.List()

	assert.Len(t, list, 6)

	names := make(map[string]bool)
	for _, info := range list {
		names[info.Template.Name] = true
		assert.True(t, info.BuiltIn)
	}

	assert.True(t, names["evm-basic"])
	assert.True(t, names["evm-defi"])
	assert.True(t, names["evm-nft"])
	assert.True(t, names["evm-dao"])
	assert.True(t, names["evm-token"])
	assert.True(t, names["evm-arbitrum"])
}

func TestManagerListWithUserTemplates(t *testing.T) {
	globalDir := t.TempDir()

	userTmplDir := filepath.Join(globalDir, "my-custom")
	require.NoError(t, os.MkdirAll(userTmplDir, 0o755))
	yamlContent := `name: my-custom
version: "1.0.0"
description: "Custom template"
author: user
category: custom
`
	require.NoError(t, os.WriteFile(filepath.Join(userTmplDir, "template.yaml"), []byte(yamlContent), 0o644))

	m := NewManager(globalDir)
	list := m.List()

	assert.Len(t, list, 7)

	found := false
	for _, info := range list {
		if info.Template.Name == "my-custom" {
			found = true
			assert.False(t, info.BuiltIn)
			assert.Equal(t, userTmplDir, info.Path)
			assert.Equal(t, "1.0.0", info.Template.Version)
			assert.Equal(t, "Custom template", info.Template.Description)
			assert.Equal(t, "user", info.Template.Author)
		}
	}
	assert.True(t, found, "user template should be in list")
}

func TestManagerListSkipsFiles(t *testing.T) {
	globalDir := t.TempDir()

	require.NoError(t, os.WriteFile(filepath.Join(globalDir, "not-a-dir.txt"), []byte("hi"), 0o644))

	m := NewManager(globalDir)
	list := m.List()

	assert.Len(t, list, 6)
}

func TestManagerListSkipsDirsWithoutTemplateYaml(t *testing.T) {
	globalDir := t.TempDir()

	require.NoError(t, os.MkdirAll(filepath.Join(globalDir, "no-meta"), 0o755))

	m := NewManager(globalDir)
	list := m.List()

	assert.Len(t, list, 6)
}

func TestManagerListSkipsInvalidYaml(t *testing.T) {
	globalDir := t.TempDir()

	badDir := filepath.Join(globalDir, "bad-yaml")
	require.NoError(t, os.MkdirAll(badDir, 0o755))
	require.NoError(t, os.WriteFile(filepath.Join(badDir, "template.yaml"), []byte("{{invalid yaml"), 0o644))

	m := NewManager(globalDir)
	list := m.List()

	assert.Len(t, list, 6)
}

func TestManagerGetBuiltin(t *testing.T) {
	m := NewManager(t.TempDir())

	info, err := m.Get("evm-basic")
	require.NoError(t, err)
	require.NotNil(t, info)
	assert.Equal(t, "evm-basic", info.Template.Name)
	assert.True(t, info.BuiltIn)
}

func TestManagerGetAllBuiltins(t *testing.T) {
	m := NewManager(t.TempDir())
	names := []string{"evm-basic", "evm-defi", "evm-nft", "evm-dao", "evm-token"}

	for _, name := range names {
		t.Run(name, func(t *testing.T) {
			info, err := m.Get(name)
			require.NoError(t, err)
			require.NotNil(t, info)
			assert.Equal(t, name, info.Template.Name)
			assert.True(t, info.BuiltIn)
		})
	}
}

func TestManagerGetUserTemplate(t *testing.T) {
	globalDir := t.TempDir()

	userDir := filepath.Join(globalDir, "my-template")
	require.NoError(t, os.MkdirAll(userDir, 0o755))
	yamlContent := `name: my-template
version: "1.0.0"
description: "My template"
author: me
`
	require.NoError(t, os.WriteFile(filepath.Join(userDir, "template.yaml"), []byte(yamlContent), 0o644))

	m := NewManager(globalDir)
	info, err := m.Get("my-template")
	require.NoError(t, err)
	require.NotNil(t, info)
	assert.Equal(t, "my-template", info.Template.Name)
	assert.Equal(t, "1.0.0", info.Template.Version)
	assert.Equal(t, userDir, info.Path)
	assert.False(t, info.BuiltIn)
}

func TestManagerGetNotFound(t *testing.T) {
	m := NewManager(t.TempDir())

	info, err := m.Get("nonexistent")
	assert.Error(t, err)
	assert.Nil(t, info)
	assert.Contains(t, err.Error(), "not found")
}

func TestManagerGetInvalidYaml(t *testing.T) {
	globalDir := t.TempDir()

	badDir := filepath.Join(globalDir, "bad-template")
	require.NoError(t, os.MkdirAll(badDir, 0o755))
	require.NoError(t, os.WriteFile(filepath.Join(badDir, "template.yaml"), []byte("{{invalid yaml"), 0o644))

	m := NewManager(globalDir)
	info, err := m.Get("bad-template")
	assert.Error(t, err)
	assert.Nil(t, info)
	assert.Contains(t, err.Error(), "invalid template")
}

func TestManagerGetFSBuiltin(t *testing.T) {
	m := NewManager(t.TempDir())

	fsys, err := m.GetFS("evm-basic")
	require.NoError(t, err)
	require.NotNil(t, fsys)

	_, err = fsys.Open(".")
	assert.NoError(t, err)
}

func TestManagerGetFSAllBuiltins(t *testing.T) {
	m := NewManager(t.TempDir())
	names := []string{"evm-basic", "evm-defi", "evm-nft", "evm-dao", "evm-token"}

	for _, name := range names {
		t.Run(name, func(t *testing.T) {
			fsys, err := m.GetFS(name)
			require.NoError(t, err)
			require.NotNil(t, fsys)
		})
	}
}

func TestManagerGetFSUserTemplate(t *testing.T) {
	globalDir := t.TempDir()

	userDir := filepath.Join(globalDir, "my-template")
	require.NoError(t, os.MkdirAll(userDir, 0o755))
	require.NoError(t, os.WriteFile(filepath.Join(userDir, "template.yaml"), []byte("name: my-template\n"), 0o644))
	require.NoError(t, os.WriteFile(filepath.Join(userDir, "README.md"), []byte("# Hello"), 0o644))

	m := NewManager(globalDir)
	fsys, err := m.GetFS("my-template")
	require.NoError(t, err)
	require.NotNil(t, fsys)
}

func TestManagerGetFSNotFound(t *testing.T) {
	m := NewManager(t.TempDir())

	fsys, err := m.GetFS("nonexistent")
	assert.Error(t, err)
	assert.Nil(t, fsys)
	assert.Contains(t, err.Error(), "not found")
}

func TestManagerBuiltinPrecedence(t *testing.T) {
	globalDir := t.TempDir()

	userDir := filepath.Join(globalDir, "evm-basic")
	require.NoError(t, os.MkdirAll(userDir, 0o755))
	require.NoError(t, os.WriteFile(filepath.Join(userDir, "template.yaml"), []byte("name: evm-basic\nversion: \"99.0.0\"\n"), 0o644))

	m := NewManager(globalDir)

	info, err := m.Get("evm-basic")
	require.NoError(t, err)
	assert.True(t, info.BuiltIn)
	assert.Equal(t, "2.0.0", info.Template.Version) // builtin version, not 99.0.0
}
