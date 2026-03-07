package container

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestUnsupportedRuntimeError_Error(t *testing.T) {
	err := &UnsupportedRuntimeError{Name: "rkt"}
	assert.Equal(t, "unsupported container runtime: rkt", err.Error())
}

func TestUnsupportedRuntimeError_ErrorEmpty(t *testing.T) {
	err := &UnsupportedRuntimeError{Name: ""}
	assert.Equal(t, "unsupported container runtime: ", err.Error())
}

func TestUnsupportedRuntimeError_ImplementsError(t *testing.T) {
	var err error = &UnsupportedRuntimeError{Name: "test"}
	assert.NotNil(t, err)
	assert.Contains(t, err.Error(), "test")
}

func TestNewRuntime_UnsupportedRuntime(t *testing.T) {
	rt, err := NewRuntime("lxc")
	assert.Nil(t, rt)
	require.Error(t, err)

	var ure *UnsupportedRuntimeError
	require.ErrorAs(t, err, &ure)
	assert.Equal(t, "lxc", ure.Name)
}

func TestNewRuntime_UnsupportedRuntimeContainsMessage(t *testing.T) {
	_, err := NewRuntime("containerd")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "unsupported container runtime: containerd")
}

func TestContainerConfig_Defaults(t *testing.T) {
	cfg := &ContainerConfig{}
	assert.Empty(t, cfg.Name)
	assert.Empty(t, cfg.Image)
	assert.Nil(t, cfg.Command)
	assert.Nil(t, cfg.Entrypoint)
	assert.Nil(t, cfg.Env)
	assert.Nil(t, cfg.Ports)
	assert.Nil(t, cfg.Volumes)
	assert.Nil(t, cfg.Networks)
	assert.Nil(t, cfg.Labels)
	assert.Empty(t, cfg.WorkingDir)
	assert.Empty(t, cfg.User)
	assert.Empty(t, cfg.RestartPolicy)
	assert.Equal(t, int64(0), cfg.MemoryLimit)
	assert.Equal(t, float64(0), cfg.CPULimit)
	assert.False(t, cfg.ReadOnly)
	assert.Nil(t, cfg.CapDrop)
	assert.Empty(t, cfg.Hostname)
	assert.Nil(t, cfg.NetworkAliases)
}

func TestContainerConfig_FullyPopulated(t *testing.T) {
	cfg := &ContainerConfig{
		Name:       "test-container",
		Image:      "alpine:latest",
		Command:    []string{"echo", "hello"},
		Entrypoint: []string{"/bin/sh"},
		Env:        map[string]string{"KEY": "VAL"},
		Ports:      map[int]int{8080: 80, 443: 0},
		Volumes: []VolumeMount{
			{Source: "/host", Target: "/container", ReadOnly: true},
		},
		Networks:      []string{"net1"},
		Labels:        map[string]string{"app": "test"},
		WorkingDir:    "/app",
		User:          "root",
		RestartPolicy: "always",
		MemoryLimit:   1024 * 1024 * 256,
		CPULimit:      2.5,
		ReadOnly:      true,
		CapDrop:       []string{"NET_RAW"},
		Hostname:      "myhost",
		NetworkAliases: map[string][]string{
			"net1": {"alias1", "alias2"},
		},
	}

	assert.Equal(t, "test-container", cfg.Name)
	assert.Equal(t, "alpine:latest", cfg.Image)
	assert.Equal(t, []string{"echo", "hello"}, cfg.Command)
	assert.Equal(t, []string{"/bin/sh"}, cfg.Entrypoint)
	assert.Equal(t, "VAL", cfg.Env["KEY"])
	assert.Equal(t, 80, cfg.Ports[8080])
	assert.Equal(t, 0, cfg.Ports[443])
	assert.Len(t, cfg.Volumes, 1)
	assert.True(t, cfg.Volumes[0].ReadOnly)
	assert.Equal(t, []string{"net1"}, cfg.Networks)
	assert.Equal(t, "test", cfg.Labels["app"])
	assert.Equal(t, "/app", cfg.WorkingDir)
	assert.Equal(t, "root", cfg.User)
	assert.Equal(t, "always", cfg.RestartPolicy)
	assert.Equal(t, int64(1024*1024*256), cfg.MemoryLimit)
	assert.Equal(t, 2.5, cfg.CPULimit)
	assert.True(t, cfg.ReadOnly)
	assert.Equal(t, []string{"NET_RAW"}, cfg.CapDrop)
	assert.Equal(t, "myhost", cfg.Hostname)
	assert.Equal(t, []string{"alias1", "alias2"}, cfg.NetworkAliases["net1"])
}

func TestContainerInfo_Defaults(t *testing.T) {
	info := ContainerInfo{}
	assert.Empty(t, info.ID)
	assert.Empty(t, info.Name)
	assert.Empty(t, info.Image)
	assert.Empty(t, info.Status)
	assert.Empty(t, info.State)
	assert.Nil(t, info.Ports)
	assert.Nil(t, info.Labels)
	assert.Nil(t, info.Networks)
	assert.True(t, info.CreatedAt.IsZero())
	assert.True(t, info.StartedAt.IsZero())
	assert.Empty(t, info.IPAddress)
}

func TestContainerInfo_FullyPopulated(t *testing.T) {
	now := time.Now()
	info := ContainerInfo{
		ID:        "abc123",
		Name:      "my-container",
		Image:     "nginx:latest",
		Status:    "running",
		State:     "running",
		Ports:     map[int]int{80: 8080},
		Labels:    map[string]string{"env": "prod"},
		Networks:  []string{"bridge"},
		CreatedAt: now,
		StartedAt: now,
		IPAddress: "172.17.0.2",
	}

	assert.Equal(t, "abc123", info.ID)
	assert.Equal(t, "my-container", info.Name)
	assert.Equal(t, "nginx:latest", info.Image)
	assert.Equal(t, "running", info.Status)
	assert.Equal(t, "running", info.State)
	assert.Equal(t, 8080, info.Ports[80])
	assert.Equal(t, "prod", info.Labels["env"])
	assert.Equal(t, []string{"bridge"}, info.Networks)
	assert.Equal(t, now, info.CreatedAt)
	assert.Equal(t, now, info.StartedAt)
	assert.Equal(t, "172.17.0.2", info.IPAddress)
}

func TestListOptions_Defaults(t *testing.T) {
	opts := ListOptions{}
	assert.False(t, opts.All)
	assert.Nil(t, opts.Labels)
	assert.Equal(t, 0, opts.Limit)
}

func TestListOptions_WithValues(t *testing.T) {
	opts := ListOptions{
		All:    true,
		Labels: map[string]string{"env": "test"},
		Limit:  10,
	}
	assert.True(t, opts.All)
	assert.Equal(t, "test", opts.Labels["env"])
	assert.Equal(t, 10, opts.Limit)
}

func TestBuildOptions_Defaults(t *testing.T) {
	opts := BuildOptions{}
	assert.Nil(t, opts.Tags)
	assert.Empty(t, opts.Dockerfile)
	assert.Nil(t, opts.BuildArgs)
	assert.False(t, opts.NoCache)
}

func TestBuildOptions_WithValues(t *testing.T) {
	opts := BuildOptions{
		Tags:       []string{"myimage:v1", "myimage:latest"},
		Dockerfile: "Dockerfile.prod",
		BuildArgs:  map[string]string{"VERSION": "1.0"},
		NoCache:    true,
	}
	assert.Len(t, opts.Tags, 2)
	assert.Equal(t, "Dockerfile.prod", opts.Dockerfile)
	assert.Equal(t, "1.0", opts.BuildArgs["VERSION"])
	assert.True(t, opts.NoCache)
}

func TestImageInfo_FullyPopulated(t *testing.T) {
	now := time.Now()
	info := ImageInfo{
		ID:      "sha256:abc",
		Tags:    []string{"alpine:3.18", "alpine:latest"},
		Size:    5242880,
		Created: now,
	}
	assert.Equal(t, "sha256:abc", info.ID)
	assert.Contains(t, info.Tags, "alpine:3.18")
	assert.Equal(t, int64(5242880), info.Size)
	assert.Equal(t, now, info.Created)
}

func TestLogOptions_Defaults(t *testing.T) {
	opts := LogOptions{}
	assert.False(t, opts.Follow)
	assert.Empty(t, opts.Tail)
	assert.Empty(t, opts.Since)
	assert.False(t, opts.Timestamps)
	assert.False(t, opts.Stdout)
	assert.False(t, opts.Stderr)
}

func TestLogOptions_WithValues(t *testing.T) {
	opts := LogOptions{
		Follow:     true,
		Tail:       "100",
		Since:      "2024-01-01",
		Timestamps: true,
		Stdout:     true,
		Stderr:     true,
	}
	assert.True(t, opts.Follow)
	assert.Equal(t, "100", opts.Tail)
	assert.Equal(t, "2024-01-01", opts.Since)
	assert.True(t, opts.Timestamps)
	assert.True(t, opts.Stdout)
	assert.True(t, opts.Stderr)
}

func TestExecOptions_Defaults(t *testing.T) {
	opts := ExecOptions{}
	assert.Nil(t, opts.Stdin)
	assert.Nil(t, opts.Stdout)
	assert.Nil(t, opts.Stderr)
	assert.False(t, opts.Interactive)
	assert.False(t, opts.TTY)
	assert.Nil(t, opts.Env)
	assert.Empty(t, opts.WorkingDir)
}

func TestExecResult_Defaults(t *testing.T) {
	result := ExecResult{}
	assert.Equal(t, 0, result.ExitCode)
	assert.Empty(t, result.Stdout)
	assert.Empty(t, result.Stderr)
}

func TestExecResult_WithValues(t *testing.T) {
	result := ExecResult{
		ExitCode: 1,
		Stdout:   "some output",
		Stderr:   "some error",
	}
	assert.Equal(t, 1, result.ExitCode)
	assert.Equal(t, "some output", result.Stdout)
	assert.Equal(t, "some error", result.Stderr)
}

func TestVolumeMount_Defaults(t *testing.T) {
	vm := VolumeMount{}
	assert.Empty(t, vm.Source)
	assert.Empty(t, vm.Target)
	assert.False(t, vm.ReadOnly)
}

func TestVolumeMount_WithValues(t *testing.T) {
	vm := VolumeMount{
		Source:   "/host/data",
		Target:   "/data",
		ReadOnly: true,
	}
	assert.Equal(t, "/host/data", vm.Source)
	assert.Equal(t, "/data", vm.Target)
	assert.True(t, vm.ReadOnly)
}

func TestNetworkOptions_Defaults(t *testing.T) {
	opts := NetworkOptions{}
	assert.Empty(t, opts.Driver)
	assert.False(t, opts.Internal)
	assert.Nil(t, opts.Labels)
	assert.Empty(t, opts.Subnet)
	assert.Empty(t, opts.Gateway)
}

func TestNetworkOptions_WithValues(t *testing.T) {
	opts := NetworkOptions{
		Driver:   "bridge",
		Internal: true,
		Labels:   map[string]string{"env": "test"},
		Subnet:   "172.20.0.0/16",
		Gateway:  "172.20.0.1",
	}
	assert.Equal(t, "bridge", opts.Driver)
	assert.True(t, opts.Internal)
	assert.Equal(t, "test", opts.Labels["env"])
	assert.Equal(t, "172.20.0.0/16", opts.Subnet)
	assert.Equal(t, "172.20.0.1", opts.Gateway)
}

func TestNetworkInfo_FullyPopulated(t *testing.T) {
	info := NetworkInfo{
		ID:     "net-abc",
		Name:   "my-network",
		Driver: "bridge",
		Subnet: "10.0.0.0/24",
		Labels: map[string]string{"scope": "local"},
	}
	assert.Equal(t, "net-abc", info.ID)
	assert.Equal(t, "my-network", info.Name)
	assert.Equal(t, "bridge", info.Driver)
	assert.Equal(t, "10.0.0.0/24", info.Subnet)
	assert.Equal(t, "local", info.Labels["scope"])
}

func TestVolumeOptions_Defaults(t *testing.T) {
	opts := VolumeOptions{}
	assert.Empty(t, opts.Driver)
	assert.Nil(t, opts.Labels)
}

func TestVolumeOptions_WithValues(t *testing.T) {
	opts := VolumeOptions{
		Driver: "local",
		Labels: map[string]string{"type": "data"},
	}
	assert.Equal(t, "local", opts.Driver)
	assert.Equal(t, "data", opts.Labels["type"])
}

func TestVolumeInfo_FullyPopulated(t *testing.T) {
	now := time.Now()
	info := VolumeInfo{
		Name:       "my-volume",
		Driver:     "local",
		Mountpoint: "/var/lib/docker/volumes/my-volume/_data",
		Labels:     map[string]string{"project": "dokrypt"},
		CreatedAt:  now,
	}
	assert.Equal(t, "my-volume", info.Name)
	assert.Equal(t, "local", info.Driver)
	assert.Contains(t, info.Mountpoint, "my-volume")
	assert.Equal(t, "dokrypt", info.Labels["project"])
	assert.Equal(t, now, info.CreatedAt)
}

func TestRuntimeInfo_FullyPopulated(t *testing.T) {
	info := RuntimeInfo{
		Name:       "docker",
		Version:    "24.0.7",
		APIVersion: "1.43",
		OS:         "linux",
		Arch:       "amd64",
	}
	assert.Equal(t, "docker", info.Name)
	assert.Equal(t, "24.0.7", info.Version)
	assert.Equal(t, "1.43", info.APIVersion)
	assert.Equal(t, "linux", info.OS)
	assert.Equal(t, "amd64", info.Arch)
}

func TestRuntimeInterface_Satisfied(t *testing.T) {
	var rt Runtime = newMockRuntime()
	assert.NotNil(t, rt)
}
