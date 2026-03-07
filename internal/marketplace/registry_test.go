package marketplace

import (
	"encoding/json"
	"io/fs"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gopkg.in/yaml.v3"
)

func TestNewLocalRegistry(t *testing.T) {
	reg := NewLocalRegistry("/some/path")
	require.NotNil(t, reg)
	assert.Equal(t, "/some/path", reg.Dir())
}

func TestLocalRegistry_Dir(t *testing.T) {
	reg := NewLocalRegistry("/custom/dir")
	assert.Equal(t, "/custom/dir", reg.Dir())
}

func TestDefaultLocalRegistry(t *testing.T) {
	reg, err := DefaultLocalRegistry()
	require.NoError(t, err)
	require.NotNil(t, reg)

	home, _ := os.UserHomeDir()
	expected := filepath.Join(home, ".dokrypt", "marketplace")
	assert.Equal(t, expected, reg.Dir())
}

func TestLocalRegistry_Install_WithExplicitName(t *testing.T) {
	tmpDir := t.TempDir()
	reg := NewLocalRegistry(tmpDir)

	srcDir := t.TempDir()
	require.NoError(t, os.WriteFile(filepath.Join(srcDir, "template.yaml"), []byte("name: test"), 0o644))
	require.NoError(t, os.WriteFile(filepath.Join(srcDir, "main.sol"), []byte("// solidity"), 0o644))

	meta := PackageMeta{
		Name:    "my-template",
		Version: "1.0.0",
		Author:  "tester",
	}

	err := reg.Install("custom-name", meta, srcDir)
	require.NoError(t, err)

	destDir := filepath.Join(tmpDir, "custom-name")
	assert.DirExists(t, destDir)

	assert.FileExists(t, filepath.Join(destDir, "template.yaml"))
	assert.FileExists(t, filepath.Join(destDir, "main.sol"))

	metaPath := filepath.Join(destDir, ".dokrypt-install.json")
	assert.FileExists(t, metaPath)

	data, err := os.ReadFile(metaPath)
	require.NoError(t, err)

	var installed InstalledPackage
	err = json.Unmarshal(data, &installed)
	require.NoError(t, err)
	assert.Equal(t, "my-template", installed.Name)
	assert.Equal(t, "1.0.0", installed.Version)
	assert.Equal(t, destDir, installed.Path)
	assert.False(t, installed.InstalledAt.IsZero())
}

func TestLocalRegistry_Install_WithEmptyName(t *testing.T) {
	tmpDir := t.TempDir()
	reg := NewLocalRegistry(tmpDir)

	srcDir := t.TempDir()
	require.NoError(t, os.WriteFile(filepath.Join(srcDir, "README.md"), []byte("# hello"), 0o644))

	meta := PackageMeta{
		Name:    "org/my-template",
		Version: "2.0.0",
	}

	err := reg.Install("", meta, srcDir)
	require.NoError(t, err)

	destDir := filepath.Join(tmpDir, "my-template")
	assert.DirExists(t, destDir)
	assert.FileExists(t, filepath.Join(destDir, "README.md"))
	assert.FileExists(t, filepath.Join(destDir, ".dokrypt-install.json"))
}

func TestLocalRegistry_Install_WithSubdirectories(t *testing.T) {
	tmpDir := t.TempDir()
	reg := NewLocalRegistry(tmpDir)

	srcDir := t.TempDir()
	subDir := filepath.Join(srcDir, "contracts")
	require.NoError(t, os.MkdirAll(subDir, 0o755))
	require.NoError(t, os.WriteFile(filepath.Join(subDir, "Token.sol"), []byte("// token"), 0o644))
	require.NoError(t, os.WriteFile(filepath.Join(srcDir, "hardhat.config.js"), []byte("module.exports = {}"), 0o644))

	meta := PackageMeta{Name: "erc20-template", Version: "1.0.0"}
	err := reg.Install("erc20", meta, srcDir)
	require.NoError(t, err)

	assert.FileExists(t, filepath.Join(tmpDir, "erc20", "contracts", "Token.sol"))
	assert.FileExists(t, filepath.Join(tmpDir, "erc20", "hardhat.config.js"))
}

func TestLocalRegistry_Uninstall_Success(t *testing.T) {
	tmpDir := t.TempDir()
	reg := NewLocalRegistry(tmpDir)

	srcDir := t.TempDir()
	require.NoError(t, os.WriteFile(filepath.Join(srcDir, "file.txt"), []byte("data"), 0o644))
	meta := PackageMeta{Name: "to-remove", Version: "1.0.0"}
	require.NoError(t, reg.Install("to-remove", meta, srcDir))

	assert.DirExists(t, filepath.Join(tmpDir, "to-remove"))

	err := reg.Uninstall("to-remove")
	require.NoError(t, err)

	assert.NoDirExists(t, filepath.Join(tmpDir, "to-remove"))
}

func TestLocalRegistry_Uninstall_NotInstalled(t *testing.T) {
	tmpDir := t.TempDir()
	reg := NewLocalRegistry(tmpDir)

	err := reg.Uninstall("nonexistent")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "not installed")
}

func TestLocalRegistry_Get_FromInstallJSON(t *testing.T) {
	tmpDir := t.TempDir()
	reg := NewLocalRegistry(tmpDir)

	srcDir := t.TempDir()
	require.NoError(t, os.WriteFile(filepath.Join(srcDir, "main.go"), []byte("package main"), 0o644))
	meta := PackageMeta{
		Name:        "get-test",
		Version:     "3.0.0",
		Description: "A test package",
		Author:      "author",
		Category:    "defi",
	}
	require.NoError(t, reg.Install("get-test", meta, srcDir))

	pkg, err := reg.Get("get-test")
	require.NoError(t, err)
	require.NotNil(t, pkg)

	assert.Equal(t, "get-test", pkg.Name)
	assert.Equal(t, "3.0.0", pkg.Version)
	assert.Equal(t, "A test package", pkg.Description)
	assert.Equal(t, "author", pkg.Author)
	assert.Equal(t, "defi", pkg.Category)
	assert.False(t, pkg.InstalledAt.IsZero())
}

func TestLocalRegistry_Get_FallbackToTemplateYAML(t *testing.T) {
	tmpDir := t.TempDir()
	reg := NewLocalRegistry(tmpDir)

	pkgDir := filepath.Join(tmpDir, "yaml-only")
	require.NoError(t, os.MkdirAll(pkgDir, 0o755))

	meta := PackageMeta{
		Name:        "yaml-only",
		Version:     "1.0.0",
		Description: "From YAML",
		Author:      "yaml-author",
		Category:    "nft",
	}
	yamlData, err := yaml.Marshal(meta)
	require.NoError(t, err)
	require.NoError(t, os.WriteFile(filepath.Join(pkgDir, "template.yaml"), yamlData, 0o644))

	pkg, err := reg.Get("yaml-only")
	require.NoError(t, err)
	require.NotNil(t, pkg)

	assert.Equal(t, "yaml-only", pkg.Name)
	assert.Equal(t, "1.0.0", pkg.Version)
	assert.Equal(t, "From YAML", pkg.Description)
	assert.Equal(t, "yaml-author", pkg.Author)
	assert.Equal(t, pkgDir, pkg.Path)
	assert.False(t, pkg.InstalledAt.IsZero())
}

func TestLocalRegistry_Get_NotInstalled(t *testing.T) {
	tmpDir := t.TempDir()
	reg := NewLocalRegistry(tmpDir)

	pkg, err := reg.Get("missing")
	require.Error(t, err)
	assert.Nil(t, pkg)
	assert.Contains(t, err.Error(), "not installed")
}

func TestLocalRegistry_Get_InvalidJSON(t *testing.T) {
	tmpDir := t.TempDir()
	reg := NewLocalRegistry(tmpDir)

	pkgDir := filepath.Join(tmpDir, "bad-json")
	require.NoError(t, os.MkdirAll(pkgDir, 0o755))
	require.NoError(t, os.WriteFile(
		filepath.Join(pkgDir, ".dokrypt-install.json"),
		[]byte("{invalid json"),
		0o644,
	))

	pkg, err := reg.Get("bad-json")
	require.Error(t, err)
	assert.Nil(t, pkg)
	assert.Contains(t, err.Error(), "invalid install metadata")
}

func TestLocalRegistry_Get_InvalidYAMLFallback(t *testing.T) {
	tmpDir := t.TempDir()
	reg := NewLocalRegistry(tmpDir)

	pkgDir := filepath.Join(tmpDir, "bad-yaml")
	require.NoError(t, os.MkdirAll(pkgDir, 0o755))
	require.NoError(t, os.WriteFile(
		filepath.Join(pkgDir, "template.yaml"),
		[]byte(":::invalid yaml:::"),
		0o644,
	))

	pkg, err := reg.Get("bad-yaml")
	if err != nil {
		assert.Nil(t, pkg)
		assert.Contains(t, err.Error(), "invalid template.yaml")
	}
}

func TestLocalRegistry_List_Empty(t *testing.T) {
	tmpDir := t.TempDir()
	reg := NewLocalRegistry(tmpDir)

	list := reg.List()
	assert.Empty(t, list)
}

func TestLocalRegistry_List_NonexistentDir(t *testing.T) {
	reg := NewLocalRegistry(filepath.Join(t.TempDir(), "does-not-exist"))
	list := reg.List()
	assert.Empty(t, list)
}

func TestLocalRegistry_List_MultiplePackages(t *testing.T) {
	tmpDir := t.TempDir()
	reg := NewLocalRegistry(tmpDir)

	srcDir := t.TempDir()
	require.NoError(t, os.WriteFile(filepath.Join(srcDir, "file.txt"), []byte("data"), 0o644))

	require.NoError(t, reg.Install("pkg-a", PackageMeta{Name: "pkg-a", Version: "1.0.0", Category: "defi"}, srcDir))
	require.NoError(t, reg.Install("pkg-b", PackageMeta{Name: "pkg-b", Version: "2.0.0", Category: "nft"}, srcDir))
	require.NoError(t, reg.Install("pkg-c", PackageMeta{Name: "pkg-c", Version: "3.0.0", Category: "defi"}, srcDir))

	list := reg.List()
	assert.Len(t, list, 3)

	names := make(map[string]bool)
	for _, pkg := range list {
		names[pkg.Name] = true
	}
	assert.True(t, names["pkg-a"])
	assert.True(t, names["pkg-b"])
	assert.True(t, names["pkg-c"])
}

func TestLocalRegistry_List_SkipsFiles(t *testing.T) {
	tmpDir := t.TempDir()
	reg := NewLocalRegistry(tmpDir)

	require.NoError(t, os.WriteFile(filepath.Join(tmpDir, "not-a-dir.txt"), []byte("data"), 0o644))

	srcDir := t.TempDir()
	require.NoError(t, os.WriteFile(filepath.Join(srcDir, "file.txt"), []byte("data"), 0o644))
	require.NoError(t, reg.Install("real-pkg", PackageMeta{Name: "real-pkg", Version: "1.0.0"}, srcDir))

	list := reg.List()
	assert.Len(t, list, 1)
	assert.Equal(t, "real-pkg", list[0].Name)
}

func TestLocalRegistry_Search_ByName(t *testing.T) {
	tmpDir := t.TempDir()
	reg := NewLocalRegistry(tmpDir)
	srcDir := t.TempDir()
	require.NoError(t, os.WriteFile(filepath.Join(srcDir, "f.txt"), []byte("x"), 0o644))

	require.NoError(t, reg.Install("erc20-token", PackageMeta{
		Name: "erc20-token", Version: "1.0.0", Description: "ERC20 template",
	}, srcDir))
	require.NoError(t, reg.Install("nft-market", PackageMeta{
		Name: "nft-market", Version: "1.0.0", Description: "NFT marketplace",
	}, srcDir))

	results := reg.Search("erc20")
	assert.Len(t, results, 1)
	assert.Equal(t, "erc20-token", results[0].Name)
}

func TestLocalRegistry_Search_ByDescription(t *testing.T) {
	tmpDir := t.TempDir()
	reg := NewLocalRegistry(tmpDir)
	srcDir := t.TempDir()
	require.NoError(t, os.WriteFile(filepath.Join(srcDir, "f.txt"), []byte("x"), 0o644))

	require.NoError(t, reg.Install("pkg1", PackageMeta{
		Name: "pkg1", Version: "1.0.0", Description: "Decentralized exchange",
	}, srcDir))
	require.NoError(t, reg.Install("pkg2", PackageMeta{
		Name: "pkg2", Version: "1.0.0", Description: "Token bridge",
	}, srcDir))

	results := reg.Search("exchange")
	assert.Len(t, results, 1)
	assert.Equal(t, "pkg1", results[0].Name)
}

func TestLocalRegistry_Search_ByAuthor(t *testing.T) {
	tmpDir := t.TempDir()
	reg := NewLocalRegistry(tmpDir)
	srcDir := t.TempDir()
	require.NoError(t, os.WriteFile(filepath.Join(srcDir, "f.txt"), []byte("x"), 0o644))

	require.NoError(t, reg.Install("pkg1", PackageMeta{
		Name: "pkg1", Version: "1.0.0", Author: "alice",
	}, srcDir))
	require.NoError(t, reg.Install("pkg2", PackageMeta{
		Name: "pkg2", Version: "1.0.0", Author: "bob",
	}, srcDir))

	results := reg.Search("alice")
	assert.Len(t, results, 1)
	assert.Equal(t, "pkg1", results[0].Name)
}

func TestLocalRegistry_Search_ByTag(t *testing.T) {
	tmpDir := t.TempDir()
	reg := NewLocalRegistry(tmpDir)
	srcDir := t.TempDir()
	require.NoError(t, os.WriteFile(filepath.Join(srcDir, "f.txt"), []byte("x"), 0o644))

	require.NoError(t, reg.Install("pkg1", PackageMeta{
		Name: "pkg1", Version: "1.0.0", Tags: []string{"solidity", "evm"},
	}, srcDir))
	require.NoError(t, reg.Install("pkg2", PackageMeta{
		Name: "pkg2", Version: "1.0.0", Tags: []string{"rust", "solana"},
	}, srcDir))

	results := reg.Search("solana")
	assert.Len(t, results, 1)
	assert.Equal(t, "pkg2", results[0].Name)
}

func TestLocalRegistry_Search_ByChain(t *testing.T) {
	tmpDir := t.TempDir()
	reg := NewLocalRegistry(tmpDir)
	srcDir := t.TempDir()
	require.NoError(t, os.WriteFile(filepath.Join(srcDir, "f.txt"), []byte("x"), 0o644))

	require.NoError(t, reg.Install("pkg1", PackageMeta{
		Name: "pkg1", Version: "1.0.0", Chains: []string{"ethereum"},
	}, srcDir))
	require.NoError(t, reg.Install("pkg2", PackageMeta{
		Name: "pkg2", Version: "1.0.0", Chains: []string{"polygon"},
	}, srcDir))

	results := reg.Search("polygon")
	assert.Len(t, results, 1)
	assert.Equal(t, "pkg2", results[0].Name)
}

func TestLocalRegistry_Search_CaseInsensitive(t *testing.T) {
	tmpDir := t.TempDir()
	reg := NewLocalRegistry(tmpDir)
	srcDir := t.TempDir()
	require.NoError(t, os.WriteFile(filepath.Join(srcDir, "f.txt"), []byte("x"), 0o644))

	require.NoError(t, reg.Install("pkg1", PackageMeta{
		Name: "ERC20-Token", Version: "1.0.0",
	}, srcDir))

	results := reg.Search("erc20")
	assert.Len(t, results, 1)

	results = reg.Search("ERC20")
	assert.Len(t, results, 1)
}

func TestLocalRegistry_Search_NoMatch(t *testing.T) {
	tmpDir := t.TempDir()
	reg := NewLocalRegistry(tmpDir)
	srcDir := t.TempDir()
	require.NoError(t, os.WriteFile(filepath.Join(srcDir, "f.txt"), []byte("x"), 0o644))

	require.NoError(t, reg.Install("pkg1", PackageMeta{
		Name: "pkg1", Version: "1.0.0",
	}, srcDir))

	results := reg.Search("zzzzz")
	assert.Empty(t, results)
}

func TestLocalRegistry_Browse_AllCategories(t *testing.T) {
	tmpDir := t.TempDir()
	reg := NewLocalRegistry(tmpDir)
	srcDir := t.TempDir()
	require.NoError(t, os.WriteFile(filepath.Join(srcDir, "f.txt"), []byte("x"), 0o644))

	require.NoError(t, reg.Install("pkg1", PackageMeta{Name: "pkg1", Version: "1.0.0", Category: "defi"}, srcDir))
	require.NoError(t, reg.Install("pkg2", PackageMeta{Name: "pkg2", Version: "1.0.0", Category: "nft"}, srcDir))

	results := reg.Browse("")
	assert.Len(t, results, 2)
}

func TestLocalRegistry_Browse_FilterByCategory(t *testing.T) {
	tmpDir := t.TempDir()
	reg := NewLocalRegistry(tmpDir)
	srcDir := t.TempDir()
	require.NoError(t, os.WriteFile(filepath.Join(srcDir, "f.txt"), []byte("x"), 0o644))

	require.NoError(t, reg.Install("pkg1", PackageMeta{Name: "pkg1", Version: "1.0.0", Category: "defi"}, srcDir))
	require.NoError(t, reg.Install("pkg2", PackageMeta{Name: "pkg2", Version: "1.0.0", Category: "nft"}, srcDir))
	require.NoError(t, reg.Install("pkg3", PackageMeta{Name: "pkg3", Version: "1.0.0", Category: "defi"}, srcDir))

	results := reg.Browse("defi")
	assert.Len(t, results, 2)

	for _, pkg := range results {
		assert.Equal(t, "defi", pkg.Category)
	}
}

func TestLocalRegistry_Browse_CaseInsensitive(t *testing.T) {
	tmpDir := t.TempDir()
	reg := NewLocalRegistry(tmpDir)
	srcDir := t.TempDir()
	require.NoError(t, os.WriteFile(filepath.Join(srcDir, "f.txt"), []byte("x"), 0o644))

	require.NoError(t, reg.Install("pkg1", PackageMeta{Name: "pkg1", Version: "1.0.0", Category: "DeFi"}, srcDir))

	results := reg.Browse("defi")
	assert.Len(t, results, 1)

	results = reg.Browse("DEFI")
	assert.Len(t, results, 1)
}

func TestLocalRegistry_Browse_NoMatch(t *testing.T) {
	tmpDir := t.TempDir()
	reg := NewLocalRegistry(tmpDir)
	srcDir := t.TempDir()
	require.NoError(t, os.WriteFile(filepath.Join(srcDir, "f.txt"), []byte("x"), 0o644))

	require.NoError(t, reg.Install("pkg1", PackageMeta{Name: "pkg1", Version: "1.0.0", Category: "defi"}, srcDir))

	results := reg.Browse("gaming")
	assert.Empty(t, results)
}

func TestLocalRegistry_FS_Success(t *testing.T) {
	tmpDir := t.TempDir()
	reg := NewLocalRegistry(tmpDir)
	srcDir := t.TempDir()
	require.NoError(t, os.WriteFile(filepath.Join(srcDir, "hello.txt"), []byte("world"), 0o644))

	require.NoError(t, reg.Install("fs-test", PackageMeta{Name: "fs-test", Version: "1.0.0"}, srcDir))

	fsys, err := reg.FS("fs-test")
	require.NoError(t, err)
	require.NotNil(t, fsys)

	data, err := fs.ReadFile(fsys, "hello.txt")
	require.NoError(t, err)
	assert.Equal(t, "world", string(data))
}

func TestLocalRegistry_FS_NotInstalled(t *testing.T) {
	tmpDir := t.TempDir()
	reg := NewLocalRegistry(tmpDir)

	fsys, err := reg.FS("missing")
	require.Error(t, err)
	assert.Nil(t, fsys)
	assert.Contains(t, err.Error(), "not installed")
}

func TestLocalRegistry_Install_Overwrite(t *testing.T) {
	tmpDir := t.TempDir()
	reg := NewLocalRegistry(tmpDir)

	srcDir1 := t.TempDir()
	require.NoError(t, os.WriteFile(filepath.Join(srcDir1, "v1.txt"), []byte("version 1"), 0o644))
	require.NoError(t, reg.Install("overwrite-test", PackageMeta{Name: "overwrite-test", Version: "1.0.0"}, srcDir1))

	srcDir2 := t.TempDir()
	require.NoError(t, os.WriteFile(filepath.Join(srcDir2, "v2.txt"), []byte("version 2"), 0o644))
	require.NoError(t, reg.Install("overwrite-test", PackageMeta{Name: "overwrite-test", Version: "2.0.0"}, srcDir2))

	assert.FileExists(t, filepath.Join(tmpDir, "overwrite-test", "v2.txt"))

	pkg, err := reg.Get("overwrite-test")
	require.NoError(t, err)
	assert.Equal(t, "2.0.0", pkg.Version)
}

func TestLocalRegistry_Install_EmptySourceDir(t *testing.T) {
	tmpDir := t.TempDir()
	reg := NewLocalRegistry(tmpDir)
	srcDir := t.TempDir() // empty directory

	err := reg.Install("empty-src", PackageMeta{Name: "empty-src", Version: "1.0.0"}, srcDir)
	require.NoError(t, err)

	assert.FileExists(t, filepath.Join(tmpDir, "empty-src", ".dokrypt-install.json"))
}

func TestLocalRegistry_Install_InvalidSourceDir(t *testing.T) {
	tmpDir := t.TempDir()
	reg := NewLocalRegistry(tmpDir)

	err := reg.Install("bad-src", PackageMeta{Name: "bad-src", Version: "1.0.0"}, "/nonexistent/source/dir")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "copy template files")
}

func TestLocalRegistry_Search_MultipleMatches(t *testing.T) {
	tmpDir := t.TempDir()
	reg := NewLocalRegistry(tmpDir)
	srcDir := t.TempDir()
	require.NoError(t, os.WriteFile(filepath.Join(srcDir, "f.txt"), []byte("x"), 0o644))

	require.NoError(t, reg.Install("defi-swap", PackageMeta{
		Name: "defi-swap", Version: "1.0.0", Description: "DEX swap",
		Tags: []string{"defi"},
	}, srcDir))
	require.NoError(t, reg.Install("defi-lending", PackageMeta{
		Name: "defi-lending", Version: "1.0.0", Description: "Lending protocol",
		Tags: []string{"defi"},
	}, srcDir))
	require.NoError(t, reg.Install("nft-gallery", PackageMeta{
		Name: "nft-gallery", Version: "1.0.0", Description: "NFT gallery",
		Tags: []string{"nft"},
	}, srcDir))

	results := reg.Search("defi")
	assert.Len(t, results, 2)
}
