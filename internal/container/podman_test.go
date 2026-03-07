package container

import (
	"runtime"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestPodmanRuntime_ImplementsRuntime(t *testing.T) {
	var _ Runtime = (*PodmanRuntime)(nil)
}

func TestPodmanRuntime_EmbedsDockerRuntime(t *testing.T) {
	pr := &PodmanRuntime{
		DockerRuntime: &DockerRuntime{},
	}
	assert.NotNil(t, pr.DockerRuntime)
}

func TestPodmanSocketPath_ReturnsUnixSocket(t *testing.T) {
	path := podmanSocketPath()
	assert.NotEmpty(t, path)

	if runtime.GOOS == "linux" {
		assert.Contains(t, path, "unix://")
		assert.Contains(t, path, "podman")
	} else if runtime.GOOS == "darwin" {
		assert.Contains(t, path, "unix://")
		assert.Contains(t, path, "podman")
	} else {
		assert.Contains(t, path, "podman")
	}
}

func TestPodmanSocketPath_ContainsPodman(t *testing.T) {
	path := podmanSocketPath()
	assert.Contains(t, path, "podman")
}

func TestNewRuntime_Podman(t *testing.T) {
	rt, err := NewRuntime("podman")
	if err != nil {
		assert.Contains(t, err.Error(), "Podman")
	} else {
		require.NotNil(t, rt)
		pr, ok := rt.(*PodmanRuntime)
		assert.True(t, ok, "expected *PodmanRuntime")
		assert.NotNil(t, pr.DockerRuntime)
	}
}

func TestNewPodmanRuntime_CreatesInstance(t *testing.T) {
	rt, err := NewPodmanRuntime()
	if err != nil {
		assert.Contains(t, err.Error(), "Podman")
	} else {
		require.NotNil(t, rt)
		assert.NotNil(t, rt.DockerRuntime)
		assert.NotNil(t, rt.DockerRuntime.client)
	}
}
