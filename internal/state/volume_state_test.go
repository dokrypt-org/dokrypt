package state

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewVolumeStateManager_ReturnsNonNil(t *testing.T) {
	store := NewStore(t.TempDir())
	mgr := NewVolumeStateManager(store, &mockRuntime{})
	require.NotNil(t, mgr)
}

func TestCreateTarGz_CreatesArchiveFile(t *testing.T) {
	srcDir := t.TempDir()
	require.NoError(t, os.WriteFile(filepath.Join(srcDir, "data.txt"), []byte("hello"), 0o644))

	archivePath := filepath.Join(t.TempDir(), "archive.tar.gz")
	err := createTarGz(archivePath, srcDir)
	require.NoError(t, err)

	assert.FileExists(t, archivePath)
	fi, err := os.Stat(archivePath)
	require.NoError(t, err)
	assert.Greater(t, fi.Size(), int64(0))
}

func TestCreateTarGz_EmptyDirectory(t *testing.T) {
	srcDir := t.TempDir()
	archivePath := filepath.Join(t.TempDir(), "empty.tar.gz")

	err := createTarGz(archivePath, srcDir)
	require.NoError(t, err)
	assert.FileExists(t, archivePath)
}

func TestCreateTarGz_NestedDirectories(t *testing.T) {
	srcDir := t.TempDir()
	subDir := filepath.Join(srcDir, "level1", "level2")
	require.NoError(t, os.MkdirAll(subDir, 0o755))
	require.NoError(t, os.WriteFile(filepath.Join(subDir, "deep.txt"), []byte("nested"), 0o644))

	archivePath := filepath.Join(t.TempDir(), "nested.tar.gz")
	err := createTarGz(archivePath, srcDir)
	require.NoError(t, err)
	assert.FileExists(t, archivePath)
}

func TestCreateTarGz_ErrorForNonexistentSourceDir(t *testing.T) {
	archivePath := filepath.Join(t.TempDir(), "fail.tar.gz")
	err := createTarGz(archivePath, filepath.Join(t.TempDir(), "no-such-dir"))
	require.Error(t, err)
}

func TestExtractTarGz_RestoresFiles(t *testing.T) {
	srcDir := t.TempDir()
	require.NoError(t, os.WriteFile(filepath.Join(srcDir, "file1.txt"), []byte("content1"), 0o644))
	require.NoError(t, os.MkdirAll(filepath.Join(srcDir, "sub"), 0o755))
	require.NoError(t, os.WriteFile(filepath.Join(srcDir, "sub", "file2.txt"), []byte("content2"), 0o644))

	archivePath := filepath.Join(t.TempDir(), "test.tar.gz")
	err := createTarGz(archivePath, srcDir)
	require.NoError(t, err)

	destDir := t.TempDir()
	err = extractTarGz(archivePath, destDir)
	require.NoError(t, err)

	data1, err := os.ReadFile(filepath.Join(destDir, "file1.txt"))
	require.NoError(t, err)
	assert.Equal(t, "content1", string(data1))

	data2, err := os.ReadFile(filepath.Join(destDir, "sub", "file2.txt"))
	require.NoError(t, err)
	assert.Equal(t, "content2", string(data2))
}

func TestExtractTarGz_CreatesSubdirectories(t *testing.T) {
	srcDir := t.TempDir()
	deep := filepath.Join(srcDir, "a", "b", "c")
	require.NoError(t, os.MkdirAll(deep, 0o755))
	require.NoError(t, os.WriteFile(filepath.Join(deep, "leaf.txt"), []byte("leaf"), 0o644))

	archivePath := filepath.Join(t.TempDir(), "deep.tar.gz")
	require.NoError(t, createTarGz(archivePath, srcDir))

	destDir := t.TempDir()
	require.NoError(t, extractTarGz(archivePath, destDir))

	data, err := os.ReadFile(filepath.Join(destDir, "a", "b", "c", "leaf.txt"))
	require.NoError(t, err)
	assert.Equal(t, "leaf", string(data))
}

func TestExtractTarGz_ErrorForNonexistentArchive(t *testing.T) {
	err := extractTarGz(filepath.Join(t.TempDir(), "nope.tar.gz"), t.TempDir())
	require.Error(t, err)
}

func TestExtractTarGz_ErrorForInvalidArchive(t *testing.T) {
	badFile := filepath.Join(t.TempDir(), "bad.tar.gz")
	require.NoError(t, os.WriteFile(badFile, []byte("not a gzip file"), 0o644))

	err := extractTarGz(badFile, t.TempDir())
	require.Error(t, err)
}

func TestCreateAndExtract_Roundtrip_PreservesContent(t *testing.T) {
	srcDir := t.TempDir()

	files := map[string]string{
		"root.txt":            "root content",
		"dir1/file1.txt":      "dir1 content",
		"dir1/dir2/file2.txt": "dir2 content",
	}
	for relPath, content := range files {
		fullPath := filepath.Join(srcDir, relPath)
		require.NoError(t, os.MkdirAll(filepath.Dir(fullPath), 0o755))
		require.NoError(t, os.WriteFile(fullPath, []byte(content), 0o644))
	}

	archivePath := filepath.Join(t.TempDir(), "roundtrip.tar.gz")
	require.NoError(t, createTarGz(archivePath, srcDir))

	destDir := t.TempDir()
	require.NoError(t, extractTarGz(archivePath, destDir))

	for relPath, expectedContent := range files {
		data, err := os.ReadFile(filepath.Join(destDir, relPath))
		require.NoError(t, err, "should be able to read %s", relPath)
		assert.Equal(t, expectedContent, string(data), "content mismatch for %s", relPath)
	}
}

func TestCreateAndExtract_LargerFile(t *testing.T) {
	srcDir := t.TempDir()

	largeContent := make([]byte, 10*1024)
	for i := range largeContent {
		largeContent[i] = byte(i % 256)
	}
	require.NoError(t, os.WriteFile(filepath.Join(srcDir, "large.bin"), largeContent, 0o644))

	archivePath := filepath.Join(t.TempDir(), "large.tar.gz")
	require.NoError(t, createTarGz(archivePath, srcDir))

	destDir := t.TempDir()
	require.NoError(t, extractTarGz(archivePath, destDir))

	data, err := os.ReadFile(filepath.Join(destDir, "large.bin"))
	require.NoError(t, err)
	assert.Equal(t, largeContent, data)
}

func TestCreateTarGz_MultipleFiles(t *testing.T) {
	srcDir := t.TempDir()
	for i := 0; i < 10; i++ {
		name := filepath.Join(srcDir, "file"+string(rune('A'+i))+".txt")
		require.NoError(t, os.WriteFile(name, []byte("data"), 0o644))
	}

	archivePath := filepath.Join(t.TempDir(), "multi.tar.gz")
	require.NoError(t, createTarGz(archivePath, srcDir))

	destDir := t.TempDir()
	require.NoError(t, extractTarGz(archivePath, destDir))

	entries, err := os.ReadDir(destDir)
	require.NoError(t, err)
	fileCount := 0
	for _, e := range entries {
		if !e.IsDir() {
			fileCount++
		}
	}
	assert.Equal(t, 10, fileCount)
}

func TestVolumeSnapshot_FieldValues(t *testing.T) {
	vs := VolumeSnapshot{
		Name:        "test-volume",
		Service:     "test-service",
		ArchiveFile: "volumes/test-service.tar.gz",
		Size:        4096,
	}

	assert.Equal(t, "test-volume", vs.Name)
	assert.Equal(t, "test-service", vs.Service)
	assert.Equal(t, "volumes/test-service.tar.gz", vs.ArchiveFile)
	assert.Equal(t, int64(4096), vs.Size)
}

func TestVolumeSnapshot_ZeroSize(t *testing.T) {
	vs := VolumeSnapshot{
		Name:    "empty-vol",
		Service: "svc",
	}
	assert.Equal(t, int64(0), vs.Size)
	assert.Empty(t, vs.ArchiveFile)
}
