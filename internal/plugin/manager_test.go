package plugin

import (
	"archive/tar"
	"bytes"
	"compress/gzip"
	"context"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gopkg.in/yaml.v3"
)

func TestNewManager(t *testing.T) {
	m := NewManager("/global", "/local", nil)
	assert.Equal(t, "/global", m.globalDir)
	assert.Equal(t, "/local", m.localDir)
	assert.NotNil(t, m.plugins)
	assert.NotNil(t, m.loaded)
	assert.NotNil(t, m.registry)
}

func TestNewManager_WithRegistry(t *testing.T) {
	r := NewRegistryClient("https://example.com")
	m := NewManager("/global", "/local", r)
	assert.Equal(t, r, m.registry)
}

func TestNewManager_NilRegistry_UsesDefault(t *testing.T) {
	m := NewManager("/global", "/local", nil)
	assert.NotNil(t, m.registry)
	assert.Equal(t, DefaultRegistryURL, m.registry.baseURL)
}

func TestDefaultManager(t *testing.T) {
	projectDir := t.TempDir()
	m, err := DefaultManager(projectDir)
	require.NoError(t, err)
	assert.NotNil(t, m)
	assert.Contains(t, m.globalDir, ".dokrypt")
	assert.Contains(t, m.globalDir, "plugins")
	assert.Equal(t, filepath.Join(projectDir, "plugins"), m.localDir)
}

func TestManager_Discover_EmptyDirs(t *testing.T) {
	globalDir := t.TempDir()
	localDir := t.TempDir()
	m := NewManager(globalDir, localDir, nil)

	err := m.Discover()
	assert.NoError(t, err)
	assert.Empty(t, m.plugins)
}

func TestManager_Discover_NonExistentDirs(t *testing.T) {
	m := NewManager("/nonexistent/global", "/nonexistent/local", nil)
	err := m.Discover()
	assert.NoError(t, err) // logs debug, never returns error
	assert.Empty(t, m.plugins)
}

func writeTestManifest(t *testing.T, dir, name, pluginType string) {
	t.Helper()
	pluginDir := filepath.Join(dir, name)
	require.NoError(t, os.MkdirAll(pluginDir, 0o755))

	m := Manifest{
		Name:    name,
		Version: "1.0.0",
		Type:    pluginType,
	}
	data, err := yaml.Marshal(m)
	require.NoError(t, err)
	require.NoError(t, os.WriteFile(filepath.Join(pluginDir, "plugin.yaml"), data, 0o644))
}

func TestManager_Discover_LocalPlugin(t *testing.T) {
	localDir := t.TempDir()
	globalDir := t.TempDir()

	writeTestManifest(t, localDir, "my-plugin", "binary")

	m := NewManager(globalDir, localDir, nil)
	err := m.Discover()
	require.NoError(t, err)
	assert.Len(t, m.plugins, 1)
	assert.Equal(t, "my-plugin", m.plugins["my-plugin"].Manifest.Name)
	assert.False(t, m.plugins["my-plugin"].Global)
}

func TestManager_Discover_GlobalPlugin(t *testing.T) {
	localDir := t.TempDir()
	globalDir := t.TempDir()

	writeTestManifest(t, globalDir, "global-plug", "container")

	m := NewManager(globalDir, localDir, nil)
	err := m.Discover()
	require.NoError(t, err)
	assert.Len(t, m.plugins, 1)
	assert.True(t, m.plugins["global-plug"].Global)
}

func TestManager_Discover_LocalOverridesGlobal(t *testing.T) {
	localDir := t.TempDir()
	globalDir := t.TempDir()

	writeTestManifest(t, localDir, "shared", "binary")
	writeTestManifest(t, globalDir, "shared", "container")

	m := NewManager(globalDir, localDir, nil)
	err := m.Discover()
	require.NoError(t, err)
	assert.Len(t, m.plugins, 1)
	assert.Equal(t, "binary", m.plugins["shared"].Manifest.Type)
	assert.False(t, m.plugins["shared"].Global)
}

func TestManager_Discover_SkipsFiles(t *testing.T) {
	localDir := t.TempDir()
	globalDir := t.TempDir()

	require.NoError(t, os.WriteFile(filepath.Join(localDir, "not-a-dir.txt"), []byte("hello"), 0o644))

	m := NewManager(globalDir, localDir, nil)
	err := m.Discover()
	require.NoError(t, err)
	assert.Empty(t, m.plugins)
}

func TestManager_Discover_SkipsDirsWithoutManifest(t *testing.T) {
	localDir := t.TempDir()
	globalDir := t.TempDir()

	require.NoError(t, os.MkdirAll(filepath.Join(localDir, "empty-plugin"), 0o755))

	m := NewManager(globalDir, localDir, nil)
	err := m.Discover()
	require.NoError(t, err)
	assert.Empty(t, m.plugins)
}

func TestManager_Discover_SkipsInvalidYAML(t *testing.T) {
	localDir := t.TempDir()
	globalDir := t.TempDir()

	pluginDir := filepath.Join(localDir, "bad-yaml")
	require.NoError(t, os.MkdirAll(pluginDir, 0o755))
	require.NoError(t, os.WriteFile(filepath.Join(pluginDir, "plugin.yaml"), []byte(":::invalid"), 0o644))

	m := NewManager(globalDir, localDir, nil)
	err := m.Discover()
	require.NoError(t, err)
	assert.Empty(t, m.plugins)
}

func TestManager_List_Empty(t *testing.T) {
	m := NewManager(t.TempDir(), t.TempDir(), nil)
	assert.Empty(t, m.List())
}

func TestManager_List_Multiple(t *testing.T) {
	m := NewManager(t.TempDir(), t.TempDir(), nil)
	m.plugins["a"] = &Info{Manifest: Manifest{Name: "a"}}
	m.plugins["b"] = &Info{Manifest: Manifest{Name: "b"}}

	list := m.List()
	assert.Len(t, list, 2)
}

func TestManager_Get_Found(t *testing.T) {
	m := NewManager(t.TempDir(), t.TempDir(), nil)
	m.plugins["test"] = &Info{Manifest: Manifest{Name: "test", Version: "1.0.0"}}

	info, err := m.Get("test")
	require.NoError(t, err)
	assert.Equal(t, "test", info.Manifest.Name)
}

func TestManager_Get_NotFound(t *testing.T) {
	m := NewManager(t.TempDir(), t.TempDir(), nil)

	info, err := m.Get("nonexistent")
	assert.Error(t, err)
	assert.Nil(t, info)
	assert.Contains(t, err.Error(), "not found")
}

func TestManager_Uninstall_Success(t *testing.T) {
	localDir := t.TempDir()
	pluginDir := filepath.Join(localDir, "test-plugin")
	require.NoError(t, os.MkdirAll(pluginDir, 0o755))
	require.NoError(t, os.WriteFile(filepath.Join(pluginDir, "data.txt"), []byte("hello"), 0o644))

	m := NewManager(t.TempDir(), localDir, nil)
	m.plugins["test-plugin"] = &Info{
		Manifest: Manifest{Name: "test-plugin"},
		Path:     pluginDir,
	}
	m.loaded["test-plugin"] = &binaryPlugin{info: m.plugins["test-plugin"]}

	err := m.Uninstall(context.Background(), "test-plugin")
	require.NoError(t, err)

	_, statErr := os.Stat(pluginDir)
	assert.True(t, os.IsNotExist(statErr))

	assert.Empty(t, m.plugins)
	assert.Empty(t, m.loaded)
}

func TestManager_Uninstall_NotFound(t *testing.T) {
	m := NewManager(t.TempDir(), t.TempDir(), nil)
	err := m.Uninstall(context.Background(), "ghost")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "not found")
}

func TestManager_InitAll_Empty(t *testing.T) {
	m := NewManager(t.TempDir(), t.TempDir(), nil)
	err := m.InitAll(context.Background(), newTestEnv())
	assert.NoError(t, err)
}

func TestManager_InitAll_WithContainerNoImage(t *testing.T) {
	m := NewManager(t.TempDir(), t.TempDir(), nil)
	cp := &containerPlugin{
		info: &Info{
			Manifest: Manifest{Name: "ctr", Type: "container"},
		},
	}
	m.loaded["ctr"] = cp

	err := m.InitAll(context.Background(), newTestEnv())
	assert.NoError(t, err)
}

func buildTarGz(t *testing.T, files map[string]string) []byte {
	t.Helper()
	var buf bytes.Buffer
	gz := gzip.NewWriter(&buf)
	tw := tar.NewWriter(gz)

	for name, content := range files {
		hdr := &tar.Header{
			Name: name,
			Mode: 0o644,
			Size: int64(len(content)),
		}
		require.NoError(t, tw.WriteHeader(hdr))
		_, err := tw.Write([]byte(content))
		require.NoError(t, err)
	}

	require.NoError(t, tw.Close())
	require.NoError(t, gz.Close())
	return buf.Bytes()
}

func buildTarGzWithDir(t *testing.T, dirs []string, files map[string]string) []byte {
	t.Helper()
	var buf bytes.Buffer
	gz := gzip.NewWriter(&buf)
	tw := tar.NewWriter(gz)

	for _, d := range dirs {
		hdr := &tar.Header{
			Name:     d + "/",
			Typeflag: tar.TypeDir,
			Mode:     0o755,
		}
		require.NoError(t, tw.WriteHeader(hdr))
	}

	for name, content := range files {
		hdr := &tar.Header{
			Name: name,
			Mode: 0o644,
			Size: int64(len(content)),
		}
		require.NoError(t, tw.WriteHeader(hdr))
		_, err := tw.Write([]byte(content))
		require.NoError(t, err)
	}

	require.NoError(t, tw.Close())
	require.NoError(t, gz.Close())
	return buf.Bytes()
}

func TestExtractTarGz_SingleFile(t *testing.T) {
	destDir := t.TempDir()
	archive := buildTarGz(t, map[string]string{
		"plugin.yaml": "name: test\n",
	})

	err := extractTarGz(archive, destDir)
	require.NoError(t, err)

	data, err := os.ReadFile(filepath.Join(destDir, "plugin.yaml"))
	require.NoError(t, err)
	assert.Equal(t, "name: test\n", string(data))
}

func TestExtractTarGz_MultipleFiles(t *testing.T) {
	destDir := t.TempDir()
	archive := buildTarGz(t, map[string]string{
		"plugin.yaml": "name: multi\n",
		"README.md":   "# Plugin\n",
	})

	err := extractTarGz(archive, destDir)
	require.NoError(t, err)

	_, err = os.Stat(filepath.Join(destDir, "plugin.yaml"))
	assert.NoError(t, err)
	_, err = os.Stat(filepath.Join(destDir, "README.md"))
	assert.NoError(t, err)
}

func TestExtractTarGz_WithDirectory(t *testing.T) {
	destDir := t.TempDir()
	archive := buildTarGzWithDir(t, []string{"subdir"}, map[string]string{
		"subdir/file.txt": "hello\n",
	})

	err := extractTarGz(archive, destDir)
	require.NoError(t, err)

	data, err := os.ReadFile(filepath.Join(destDir, "subdir", "file.txt"))
	require.NoError(t, err)
	assert.Equal(t, "hello\n", string(data))
}

func TestExtractTarGz_PathTraversal(t *testing.T) {
	var buf bytes.Buffer
	gz := gzip.NewWriter(&buf)
	tw := tar.NewWriter(gz)

	hdr := &tar.Header{
		Name: "../../etc/passwd",
		Mode: 0o644,
		Size: 5,
	}
	require.NoError(t, tw.WriteHeader(hdr))
	_, err := tw.Write([]byte("evil\n"))
	require.NoError(t, err)
	require.NoError(t, tw.Close())
	require.NoError(t, gz.Close())

	destDir := t.TempDir()
	err = extractTarGz(buf.Bytes(), destDir)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "path traversal")
}

func TestExtractTarGz_InvalidGzip(t *testing.T) {
	destDir := t.TempDir()
	err := extractTarGz([]byte("not gzip data"), destDir)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "gzip")
}

func TestExtractTarGz_EmptyArchive(t *testing.T) {
	var buf bytes.Buffer
	gz := gzip.NewWriter(&buf)
	tw := tar.NewWriter(gz)
	require.NoError(t, tw.Close())
	require.NoError(t, gz.Close())

	destDir := t.TempDir()
	err := extractTarGz(buf.Bytes(), destDir)
	assert.NoError(t, err)
}

func TestManager_Install_DownloadFailure(t *testing.T) {
	reg := NewRegistryClient("http://127.0.0.1:1") // connection refused
	m := NewManager(t.TempDir(), t.TempDir(), reg)

	err := m.Install(context.Background(), "test-plugin", "1.0.0", false)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "failed to download")
}
