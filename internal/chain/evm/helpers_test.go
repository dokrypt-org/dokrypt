package evm

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestWriteFileBytes(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "test-write.json")

	data := []byte(`{"key":"value"}`)
	err := writeFileBytes(path, data)
	require.NoError(t, err)

	content, err := os.ReadFile(path)
	require.NoError(t, err)
	assert.Equal(t, data, content)
}

func TestReadFileBytes(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "test-read.json")

	expected := []byte(`{"hello":"world"}`)
	err := os.WriteFile(path, expected, 0644)
	require.NoError(t, err)

	data, err := readFileBytes(path)
	require.NoError(t, err)
	assert.Equal(t, expected, data)
}

func TestReadFileBytes_NonExistent(t *testing.T) {
	_, err := readFileBytes("/nonexistent/file.json")
	require.Error(t, err)
}

func TestWriteFile(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "state.json")

	data := []byte(`"0xdeadbeef"`)
	err := writeFile(path, data)
	require.NoError(t, err)

	content, err := os.ReadFile(path)
	require.NoError(t, err)
	assert.Equal(t, data, content)
}

func TestReadFile(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "state.json")

	expected := []byte(`"0xbeefdead"`)
	err := os.WriteFile(path, expected, 0644)
	require.NoError(t, err)

	data, err := readFile(path)
	require.NoError(t, err)
	assert.Equal(t, expected, data)
}

func TestReadFile_NonExistent(t *testing.T) {
	_, err := readFile("/nonexistent/path/file.json")
	require.Error(t, err)
}

func TestWriteFileBytes_CreatesFile(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "newfile.bin")

	_, err := os.Stat(path)
	require.True(t, os.IsNotExist(err))

	err = writeFileBytes(path, []byte("binary data"))
	require.NoError(t, err)

	info, err := os.Stat(path)
	require.NoError(t, err)
	assert.False(t, info.IsDir())
}

func TestWriteFileBytes_OverwritesExisting(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "overwrite.txt")

	err := os.WriteFile(path, []byte("old content"), 0644)
	require.NoError(t, err)

	newData := []byte("new content")
	err = writeFileBytes(path, newData)
	require.NoError(t, err)

	content, err := os.ReadFile(path)
	require.NoError(t, err)
	assert.Equal(t, newData, content)
}

func TestReadFileBytes_EmptyFile(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "empty.txt")

	err := os.WriteFile(path, []byte{}, 0644)
	require.NoError(t, err)

	data, err := readFileBytes(path)
	require.NoError(t, err)
	assert.Empty(t, data)
}
