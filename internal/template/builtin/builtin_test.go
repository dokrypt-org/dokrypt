package builtin

import (
	"io/fs"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestFS(t *testing.T) {
	fsys := FS()
	require.NotNil(t, fsys)

	f, err := fsys.Open(".")
	require.NoError(t, err)
	f.Close()
}

func TestFSContainsAllTemplates(t *testing.T) {
	fsys := FS()
	names := Names()

	for _, name := range names {
		t.Run(name, func(t *testing.T) {
			f, err := fsys.Open(name)
			require.NoError(t, err, "template directory %q should exist", name)
			defer f.Close()

			info, err := f.Stat()
			require.NoError(t, err)
			assert.True(t, info.IsDir(), "%q should be a directory", name)
		})
	}
}

func TestFSTemplatesHaveTemplateYaml(t *testing.T) {
	fsys := FS()
	names := Names()

	for _, name := range names {
		t.Run(name, func(t *testing.T) {
			data, err := fs.ReadFile(fsys, name+"/template.yaml")
			require.NoError(t, err, "%s should have template.yaml", name)
			assert.NotEmpty(t, data)
		})
	}
}

func TestTemplateFSSuccess(t *testing.T) {
	names := Names()

	for _, name := range names {
		t.Run(name, func(t *testing.T) {
			tmplFS, err := TemplateFS(name)
			require.NoError(t, err)
			require.NotNil(t, tmplFS)

			f, err := tmplFS.Open(".")
			require.NoError(t, err)
			f.Close()

			data, err := fs.ReadFile(tmplFS, "template.yaml")
			require.NoError(t, err)
			assert.NotEmpty(t, data)
		})
	}
}

func TestTemplateFSNotFound(t *testing.T) {
	_, err := TemplateFS("nonexistent-template")
	_ = err
}

func TestTemplateFSEvmBasicContents(t *testing.T) {
	tmplFS, err := TemplateFS("evm-basic")
	require.NoError(t, err)

	expectedFiles := []string{
		"template.yaml",
		"README.md.tmpl",
		"contracts/Counter.sol",
		"contracts/SimpleToken.sol",
	}

	for _, path := range expectedFiles {
		t.Run(path, func(t *testing.T) {
			data, err := fs.ReadFile(tmplFS, path)
			require.NoError(t, err, "file %q should exist in evm-basic", path)
			assert.NotEmpty(t, data)
		})
	}
}

func TestNames(t *testing.T) {
	names := Names()

	assert.Len(t, names, 6)

	expected := []string{"evm-basic", "evm-defi", "evm-nft", "evm-dao", "evm-token", "evm-arbitrum"}
	assert.Equal(t, expected, names)
}

func TestNamesAreUnique(t *testing.T) {
	names := Names()
	seen := make(map[string]bool)

	for _, name := range names {
		assert.False(t, seen[name], "duplicate name: %s", name)
		seen[name] = true
	}
}

func TestNamesNotEmpty(t *testing.T) {
	names := Names()
	for _, name := range names {
		assert.NotEmpty(t, name, "template name should not be empty")
	}
}

func TestFSRootEntries(t *testing.T) {
	fsys := FS()

	entries, err := fs.ReadDir(fsys, ".")
	require.NoError(t, err)

	assert.GreaterOrEqual(t, len(entries), 5)

	entryNames := make(map[string]bool)
	for _, e := range entries {
		entryNames[e.Name()] = true
	}

	for _, name := range Names() {
		assert.True(t, entryNames[name], "template %q should be in root entries", name)
	}
}

func TestTemplateFSIsolation(t *testing.T) {
	basicFS, err := TemplateFS("evm-basic")
	require.NoError(t, err)

	_, err = fs.ReadFile(basicFS, "template.yaml")
	assert.NoError(t, err)

	_, err = fs.ReadFile(basicFS, "../evm-defi/template.yaml")
	assert.Error(t, err, "should not be able to access sibling template via path traversal")
}

func TestTemplateFSReadDirContracts(t *testing.T) {
	basicFS, err := TemplateFS("evm-basic")
	require.NoError(t, err)

	entries, err := fs.ReadDir(basicFS, "contracts")
	require.NoError(t, err)
	assert.GreaterOrEqual(t, len(entries), 2, "should have at least Counter.sol and SimpleToken.sol")

	fileNames := make(map[string]bool)
	for _, e := range entries {
		fileNames[e.Name()] = true
	}
	assert.True(t, fileNames["Counter.sol"])
	assert.True(t, fileNames["SimpleToken.sol"])
}
