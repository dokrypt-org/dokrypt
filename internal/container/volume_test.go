package container

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestVolumeCreateOptions_Defaults(t *testing.T) {
	opts := VolumeCreateOptions{}
	assert.Empty(t, opts.Driver)
	assert.Nil(t, opts.Labels)
	assert.Empty(t, opts.Project)
	assert.Empty(t, opts.Service)
}

func TestVolumeCreateOptions_WithValues(t *testing.T) {
	opts := VolumeCreateOptions{
		VolumeOptions: VolumeOptions{
			Driver: "local",
			Labels: map[string]string{"env": "test"},
		},
		Project: "myproject",
		Service: "myservice",
	}
	assert.Equal(t, "local", opts.Driver)
	assert.Equal(t, "test", opts.Labels["env"])
	assert.Equal(t, "myproject", opts.Project)
	assert.Equal(t, "myservice", opts.Service)
}

func TestNewVolumeManager(t *testing.T) {
	mock := newMockRuntime()
	mgr := NewVolumeManager(mock)
	require.NotNil(t, mgr)
	assert.Equal(t, mock, mgr.runtime)
}

func TestVolumeManager_Create_Success(t *testing.T) {
	mock := newMockRuntime()
	var capturedName string
	var capturedOpts VolumeOptions
	mock.createVolumeFn = func(ctx context.Context, name string, opts VolumeOptions) (string, error) {
		capturedName = name
		capturedOpts = opts
		return name, nil
	}

	mgr := NewVolumeManager(mock)
	name, err := mgr.Create(context.Background(), "my-volume", VolumeCreateOptions{
		VolumeOptions: VolumeOptions{
			Driver: "local",
		},
		Project: "dokrypt-proj",
		Service: "node",
	})
	require.NoError(t, err)
	assert.Equal(t, "my-volume", name)
	assert.Equal(t, "my-volume", capturedName)
	assert.Equal(t, "local", capturedOpts.Driver)
	assert.Equal(t, "true", capturedOpts.Labels["dokrypt.volume"])
	assert.Equal(t, "dokrypt-proj", capturedOpts.Labels["dokrypt.project"])
	assert.Equal(t, "node", capturedOpts.Labels["dokrypt.service"])
}

func TestVolumeManager_Create_NilLabelsInitialized(t *testing.T) {
	mock := newMockRuntime()
	var capturedOpts VolumeOptions
	mock.createVolumeFn = func(ctx context.Context, name string, opts VolumeOptions) (string, error) {
		capturedOpts = opts
		return name, nil
	}

	mgr := NewVolumeManager(mock)
	_, err := mgr.Create(context.Background(), "vol1", VolumeCreateOptions{})
	require.NoError(t, err)
	require.NotNil(t, capturedOpts.Labels)
	assert.Equal(t, "true", capturedOpts.Labels["dokrypt.volume"])
}

func TestVolumeManager_Create_NoProjectOrService(t *testing.T) {
	mock := newMockRuntime()
	var capturedOpts VolumeOptions
	mock.createVolumeFn = func(ctx context.Context, name string, opts VolumeOptions) (string, error) {
		capturedOpts = opts
		return name, nil
	}

	mgr := NewVolumeManager(mock)
	_, err := mgr.Create(context.Background(), "vol1", VolumeCreateOptions{})
	require.NoError(t, err)
	assert.Equal(t, "true", capturedOpts.Labels["dokrypt.volume"])
	_, hasProject := capturedOpts.Labels["dokrypt.project"]
	assert.False(t, hasProject, "should not have project label when empty")
	_, hasService := capturedOpts.Labels["dokrypt.service"]
	assert.False(t, hasService, "should not have service label when empty")
}

func TestVolumeManager_Create_ExistingLabelsPreserved(t *testing.T) {
	mock := newMockRuntime()
	var capturedOpts VolumeOptions
	mock.createVolumeFn = func(ctx context.Context, name string, opts VolumeOptions) (string, error) {
		capturedOpts = opts
		return name, nil
	}

	mgr := NewVolumeManager(mock)
	_, err := mgr.Create(context.Background(), "vol1", VolumeCreateOptions{
		VolumeOptions: VolumeOptions{
			Labels: map[string]string{"custom": "label"},
		},
	})
	require.NoError(t, err)
	assert.Equal(t, "label", capturedOpts.Labels["custom"])
	assert.Equal(t, "true", capturedOpts.Labels["dokrypt.volume"])
}

func TestVolumeManager_Create_Error(t *testing.T) {
	mock := newMockRuntime()
	mock.createVolumeFn = func(ctx context.Context, name string, opts VolumeOptions) (string, error) {
		return "", errMock
	}

	mgr := NewVolumeManager(mock)
	name, err := mgr.Create(context.Background(), "vol1", VolumeCreateOptions{})
	require.Error(t, err)
	assert.Empty(t, name)
}

func TestVolumeManager_Remove_Success(t *testing.T) {
	mock := newMockRuntime()
	mgr := NewVolumeManager(mock)
	err := mgr.Remove(context.Background(), "vol1", false)
	require.NoError(t, err)
	assert.Equal(t, 1, mock.calls["RemoveVolume"])
}

func TestVolumeManager_Remove_Force(t *testing.T) {
	mock := newMockRuntime()
	var capturedForce bool
	mock.removeVolumeFn = func(ctx context.Context, name string, force bool) error {
		capturedForce = force
		return nil
	}

	mgr := NewVolumeManager(mock)
	err := mgr.Remove(context.Background(), "vol1", true)
	require.NoError(t, err)
	assert.True(t, capturedForce)
}

func TestVolumeManager_Remove_Error(t *testing.T) {
	mock := newMockRuntime()
	mock.removeVolumeFn = func(ctx context.Context, name string, force bool) error {
		return errMock
	}

	mgr := NewVolumeManager(mock)
	err := mgr.Remove(context.Background(), "vol1", false)
	require.Error(t, err)
}

func TestVolumeManager_List_FiltersDokryptVolumes(t *testing.T) {
	now := time.Now()
	mock := newMockRuntime()
	mock.listVolumesFn = func(ctx context.Context) ([]VolumeInfo, error) {
		return []VolumeInfo{
			{Name: "vol1", Labels: map[string]string{"dokrypt.volume": "true", "dokrypt.project": "proj1"}, CreatedAt: now},
			{Name: "vol2", Labels: map[string]string{"dokrypt.volume": "true", "dokrypt.project": "proj2"}, CreatedAt: now},
			{Name: "other-vol", Labels: map[string]string{}, CreatedAt: now},
			{Name: "no-labels", Labels: nil, CreatedAt: now},
		}, nil
	}

	mgr := NewVolumeManager(mock)
	volumes, err := mgr.List(context.Background(), "")
	require.NoError(t, err)
	assert.Len(t, volumes, 2)
	assert.Equal(t, "vol1", volumes[0].Name)
	assert.Equal(t, "vol2", volumes[1].Name)
}

func TestVolumeManager_List_FilterByProject(t *testing.T) {
	mock := newMockRuntime()
	mock.listVolumesFn = func(ctx context.Context) ([]VolumeInfo, error) {
		return []VolumeInfo{
			{Name: "vol1", Labels: map[string]string{"dokrypt.volume": "true", "dokrypt.project": "proj1"}},
			{Name: "vol2", Labels: map[string]string{"dokrypt.volume": "true", "dokrypt.project": "proj2"}},
			{Name: "vol3", Labels: map[string]string{"dokrypt.volume": "true", "dokrypt.project": "proj1"}},
		}, nil
	}

	mgr := NewVolumeManager(mock)
	volumes, err := mgr.List(context.Background(), "proj1")
	require.NoError(t, err)
	assert.Len(t, volumes, 2)
	assert.Equal(t, "vol1", volumes[0].Name)
	assert.Equal(t, "vol3", volumes[1].Name)
}

func TestVolumeManager_List_NoMatch(t *testing.T) {
	mock := newMockRuntime()
	mock.listVolumesFn = func(ctx context.Context) ([]VolumeInfo, error) {
		return []VolumeInfo{
			{Name: "vol1", Labels: map[string]string{"dokrypt.volume": "true", "dokrypt.project": "proj1"}},
		}, nil
	}

	mgr := NewVolumeManager(mock)
	volumes, err := mgr.List(context.Background(), "nonexistent")
	require.NoError(t, err)
	assert.Empty(t, volumes)
}

func TestVolumeManager_List_Error(t *testing.T) {
	mock := newMockRuntime()
	mock.listVolumesFn = func(ctx context.Context) ([]VolumeInfo, error) {
		return nil, errMock
	}

	mgr := NewVolumeManager(mock)
	volumes, err := mgr.List(context.Background(), "")
	require.Error(t, err)
	assert.Nil(t, volumes)
}

func TestVolumeManager_List_EmptyFromRuntime(t *testing.T) {
	mock := newMockRuntime()
	mock.listVolumesFn = func(ctx context.Context) ([]VolumeInfo, error) {
		return []VolumeInfo{}, nil
	}

	mgr := NewVolumeManager(mock)
	volumes, err := mgr.List(context.Background(), "")
	require.NoError(t, err)
	assert.Empty(t, volumes)
}

func TestVolumeManager_Inspect_Success(t *testing.T) {
	mock := newMockRuntime()
	now := time.Now()
	mock.inspectVolumeFn = func(ctx context.Context, name string) (*VolumeInfo, error) {
		return &VolumeInfo{
			Name:       name,
			Driver:     "local",
			Mountpoint: "/var/lib/docker/volumes/" + name + "/_data",
			Labels:     map[string]string{"dokrypt.volume": "true"},
			CreatedAt:  now,
		}, nil
	}

	mgr := NewVolumeManager(mock)
	info, err := mgr.Inspect(context.Background(), "my-vol")
	require.NoError(t, err)
	require.NotNil(t, info)
	assert.Equal(t, "my-vol", info.Name)
	assert.Equal(t, "local", info.Driver)
	assert.Contains(t, info.Mountpoint, "my-vol")
	assert.Equal(t, "true", info.Labels["dokrypt.volume"])
	assert.Equal(t, now, info.CreatedAt)
}

func TestVolumeManager_Inspect_Error(t *testing.T) {
	mock := newMockRuntime()
	mock.inspectVolumeFn = func(ctx context.Context, name string) (*VolumeInfo, error) {
		return nil, fmt.Errorf("volume not found")
	}

	mgr := NewVolumeManager(mock)
	info, err := mgr.Inspect(context.Background(), "nonexistent")
	require.Error(t, err)
	assert.Nil(t, info)
	assert.Contains(t, err.Error(), "volume not found")
}

func TestVolumeManager_Export_CreateContainerError(t *testing.T) {
	mock := newMockRuntime()
	mock.createContainerFn = func(ctx context.Context, cfg *ContainerConfig) (string, error) {
		return "", fmt.Errorf("cannot create container")
	}

	mgr := NewVolumeManager(mock)
	err := mgr.Export(context.Background(), "vol1", "/tmp/export.tar")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "failed to create export container")
}

func TestVolumeManager_Export_StartContainerError(t *testing.T) {
	mock := newMockRuntime()
	mock.createContainerFn = func(ctx context.Context, cfg *ContainerConfig) (string, error) {
		return "container-id", nil
	}
	mock.startContainerFn = func(ctx context.Context, id string) error {
		return fmt.Errorf("start failed")
	}

	mgr := NewVolumeManager(mock)
	err := mgr.Export(context.Background(), "vol1", "/tmp/export.tar")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "failed to start export container")
	assert.Equal(t, 1, mock.calls["RemoveContainer"])
}

func TestVolumeManager_Export_ContainerConfigCorrect(t *testing.T) {
	mock := newMockRuntime()
	var capturedCfg *ContainerConfig
	mock.createContainerFn = func(ctx context.Context, cfg *ContainerConfig) (string, error) {
		capturedCfg = cfg
		return "", fmt.Errorf("stop here")
	}

	mgr := NewVolumeManager(mock)
	_ = mgr.Export(context.Background(), "test-vol", "/tmp/out.tar")

	require.NotNil(t, capturedCfg)
	assert.Equal(t, "dokrypt-vol-export-test-vol", capturedCfg.Name)
	assert.Equal(t, "alpine:latest", capturedCfg.Image)
	assert.Equal(t, []string{"sleep", "3600"}, capturedCfg.Command)
	require.Len(t, capturedCfg.Volumes, 1)
	assert.Equal(t, "test-vol", capturedCfg.Volumes[0].Source)
	assert.Equal(t, "/data", capturedCfg.Volumes[0].Target)
	assert.True(t, capturedCfg.Volumes[0].ReadOnly)
}

func TestVolumeManager_Import_CreateContainerError(t *testing.T) {
	mock := newMockRuntime()
	mock.createContainerFn = func(ctx context.Context, cfg *ContainerConfig) (string, error) {
		return "", fmt.Errorf("cannot create container")
	}

	mgr := NewVolumeManager(mock)
	err := mgr.Import(context.Background(), "vol1", "/tmp/import.tar")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "failed to open source archive")
}

func TestVolumeManager_Import_ContainerConfigCorrect(t *testing.T) {
	mock := newMockRuntime()
	var capturedCfg *ContainerConfig
	mock.createContainerFn = func(ctx context.Context, cfg *ContainerConfig) (string, error) {
		capturedCfg = cfg
		return "", fmt.Errorf("stop here")
	}

	mgr := NewVolumeManager(mock)
	err := mgr.Import(context.Background(), "test-vol", "/nonexistent/path.tar")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "failed to open source archive")
	assert.Nil(t, capturedCfg)
}

func TestVolumeManager_Size_CreateContainerError(t *testing.T) {
	mock := newMockRuntime()
	mock.createContainerFn = func(ctx context.Context, cfg *ContainerConfig) (string, error) {
		return "", fmt.Errorf("cannot create")
	}

	mgr := NewVolumeManager(mock)
	size, err := mgr.Size(context.Background(), "vol1")
	require.Error(t, err)
	assert.Equal(t, int64(0), size)
	assert.Contains(t, err.Error(), "failed to create size check container")
}

func TestVolumeManager_Size_StartContainerError(t *testing.T) {
	mock := newMockRuntime()
	mock.startContainerFn = func(ctx context.Context, id string) error {
		return fmt.Errorf("start failed")
	}

	mgr := NewVolumeManager(mock)
	size, err := mgr.Size(context.Background(), "vol1")
	require.Error(t, err)
	assert.Equal(t, int64(0), size)
	assert.Contains(t, err.Error(), "failed to start size check container")
}

func TestVolumeManager_Size_ExecError(t *testing.T) {
	mock := newMockRuntime()
	mock.execInContainerFn = func(ctx context.Context, id string, cmd []string, opts ExecOptions) (*ExecResult, error) {
		return nil, fmt.Errorf("exec failed")
	}

	mgr := NewVolumeManager(mock)
	size, err := mgr.Size(context.Background(), "vol1")
	require.Error(t, err)
	assert.Equal(t, int64(0), size)
	assert.Contains(t, err.Error(), "failed to get volume size")
}

func TestVolumeManager_Size_NonZeroExitCode(t *testing.T) {
	mock := newMockRuntime()
	mock.execInContainerFn = func(ctx context.Context, id string, cmd []string, opts ExecOptions) (*ExecResult, error) {
		return &ExecResult{ExitCode: 1, Stderr: "permission denied"}, nil
	}

	mgr := NewVolumeManager(mock)
	size, err := mgr.Size(context.Background(), "vol1")
	require.Error(t, err)
	assert.Equal(t, int64(0), size)
	assert.Contains(t, err.Error(), "du failed")
	assert.Contains(t, err.Error(), "permission denied")
}

func TestVolumeManager_Size_ParseError(t *testing.T) {
	mock := newMockRuntime()
	mock.execInContainerFn = func(ctx context.Context, id string, cmd []string, opts ExecOptions) (*ExecResult, error) {
		return &ExecResult{ExitCode: 0, Stdout: "not-a-number\t/data"}, nil
	}

	mgr := NewVolumeManager(mock)
	size, err := mgr.Size(context.Background(), "vol1")
	require.Error(t, err)
	assert.Equal(t, int64(0), size)
	assert.Contains(t, err.Error(), "failed to parse size")
}

func TestVolumeManager_Size_Success(t *testing.T) {
	mock := newMockRuntime()
	mock.execInContainerFn = func(ctx context.Context, id string, cmd []string, opts ExecOptions) (*ExecResult, error) {
		assert.Equal(t, []string{"du", "-sb", "/data"}, cmd)
		return &ExecResult{ExitCode: 0, Stdout: "1048576\t/data"}, nil
	}

	mgr := NewVolumeManager(mock)
	size, err := mgr.Size(context.Background(), "vol1")
	require.NoError(t, err)
	assert.Equal(t, int64(1048576), size)
}

func TestVolumeManager_Size_ContainerConfigCorrect(t *testing.T) {
	mock := newMockRuntime()
	var capturedCfg *ContainerConfig
	mock.createContainerFn = func(ctx context.Context, cfg *ContainerConfig) (string, error) {
		capturedCfg = cfg
		return "cid", nil
	}
	mock.execInContainerFn = func(ctx context.Context, id string, cmd []string, opts ExecOptions) (*ExecResult, error) {
		return &ExecResult{ExitCode: 0, Stdout: "0\t/data"}, nil
	}

	mgr := NewVolumeManager(mock)
	_, err := mgr.Size(context.Background(), "test-vol")
	require.NoError(t, err)

	require.NotNil(t, capturedCfg)
	assert.Equal(t, "dokrypt-vol-size-test-vol", capturedCfg.Name)
	assert.Equal(t, "alpine:latest", capturedCfg.Image)
	assert.Equal(t, []string{"sleep", "3600"}, capturedCfg.Command)
	require.Len(t, capturedCfg.Volumes, 1)
	assert.Equal(t, "test-vol", capturedCfg.Volumes[0].Source)
	assert.Equal(t, "/data", capturedCfg.Volumes[0].Target)
	assert.True(t, capturedCfg.Volumes[0].ReadOnly)
}

func TestVolumeManager_Size_CleanupOnSuccess(t *testing.T) {
	mock := newMockRuntime()
	mock.execInContainerFn = func(ctx context.Context, id string, cmd []string, opts ExecOptions) (*ExecResult, error) {
		return &ExecResult{ExitCode: 0, Stdout: "100\t/data"}, nil
	}

	mgr := NewVolumeManager(mock)
	_, err := mgr.Size(context.Background(), "vol1")
	require.NoError(t, err)
	assert.Equal(t, 1, mock.calls["RemoveContainer"])
}
