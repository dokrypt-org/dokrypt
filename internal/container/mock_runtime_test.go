package container

import (
	"context"
	"fmt"
	"io"
	"sync"
	"time"
)

type mockRuntime struct {
	mu sync.Mutex
	createContainerFn  func(ctx context.Context, cfg *ContainerConfig) (string, error)
	startContainerFn   func(ctx context.Context, id string) error
	stopContainerFn    func(ctx context.Context, id string, timeout time.Duration) error
	removeContainerFn  func(ctx context.Context, id string, force bool) error
	listContainersFn   func(ctx context.Context, opts ListOptions) ([]ContainerInfo, error)
	inspectContainerFn func(ctx context.Context, id string) (*ContainerInfo, error)
	waitContainerFn    func(ctx context.Context, id string) (int64, error)

	pullImageFn   func(ctx context.Context, image string) error
	buildImageFn  func(ctx context.Context, contextPath string, opts BuildOptions) (string, error)
	listImagesFn  func(ctx context.Context) ([]ImageInfo, error)
	removeImageFn func(ctx context.Context, image string, force bool) error

	containerLogsFn  func(ctx context.Context, id string, opts LogOptions) (io.ReadCloser, error)
	execInContainerFn func(ctx context.Context, id string, cmd []string, opts ExecOptions) (*ExecResult, error)

	createNetworkFn     func(ctx context.Context, name string, opts NetworkOptions) (string, error)
	removeNetworkFn     func(ctx context.Context, id string) error
	connectNetworkFn    func(ctx context.Context, networkID, containerID string) error
	disconnectNetworkFn func(ctx context.Context, networkID, containerID string) error
	listNetworksFn      func(ctx context.Context) ([]NetworkInfo, error)

	createVolumeFn  func(ctx context.Context, name string, opts VolumeOptions) (string, error)
	removeVolumeFn  func(ctx context.Context, name string, force bool) error
	listVolumesFn   func(ctx context.Context) ([]VolumeInfo, error)
	inspectVolumeFn func(ctx context.Context, name string) (*VolumeInfo, error)

	pingFn func(ctx context.Context) error
	infoFn func(ctx context.Context) (*RuntimeInfo, error)

	calls map[string]int
}

func newMockRuntime() *mockRuntime {
	return &mockRuntime{
		calls: make(map[string]int),
	}
}

func (m *mockRuntime) recordCall(name string) {
	m.mu.Lock()
	m.calls[name]++
	m.mu.Unlock()
}

func (m *mockRuntime) CreateContainer(ctx context.Context, cfg *ContainerConfig) (string, error) {
	m.recordCall("CreateContainer")
	if m.createContainerFn != nil {
		return m.createContainerFn(ctx, cfg)
	}
	return "mock-container-id", nil
}

func (m *mockRuntime) StartContainer(ctx context.Context, id string) error {
	m.recordCall("StartContainer")
	if m.startContainerFn != nil {
		return m.startContainerFn(ctx, id)
	}
	return nil
}

func (m *mockRuntime) StopContainer(ctx context.Context, id string, timeout time.Duration) error {
	m.recordCall("StopContainer")
	if m.stopContainerFn != nil {
		return m.stopContainerFn(ctx, id, timeout)
	}
	return nil
}

func (m *mockRuntime) RemoveContainer(ctx context.Context, id string, force bool) error {
	m.recordCall("RemoveContainer")
	if m.removeContainerFn != nil {
		return m.removeContainerFn(ctx, id, force)
	}
	return nil
}

func (m *mockRuntime) ListContainers(ctx context.Context, opts ListOptions) ([]ContainerInfo, error) {
	m.recordCall("ListContainers")
	if m.listContainersFn != nil {
		return m.listContainersFn(ctx, opts)
	}
	return nil, nil
}

func (m *mockRuntime) InspectContainer(ctx context.Context, id string) (*ContainerInfo, error) {
	m.recordCall("InspectContainer")
	if m.inspectContainerFn != nil {
		return m.inspectContainerFn(ctx, id)
	}
	return &ContainerInfo{ID: id}, nil
}

func (m *mockRuntime) WaitContainer(ctx context.Context, id string) (int64, error) {
	m.recordCall("WaitContainer")
	if m.waitContainerFn != nil {
		return m.waitContainerFn(ctx, id)
	}
	return 0, nil
}

func (m *mockRuntime) PullImage(ctx context.Context, image string) error {
	m.recordCall("PullImage")
	if m.pullImageFn != nil {
		return m.pullImageFn(ctx, image)
	}
	return nil
}

func (m *mockRuntime) BuildImage(ctx context.Context, contextPath string, opts BuildOptions) (string, error) {
	m.recordCall("BuildImage")
	if m.buildImageFn != nil {
		return m.buildImageFn(ctx, contextPath, opts)
	}
	return "mock-image-id", nil
}

func (m *mockRuntime) ListImages(ctx context.Context) ([]ImageInfo, error) {
	m.recordCall("ListImages")
	if m.listImagesFn != nil {
		return m.listImagesFn(ctx)
	}
	return nil, nil
}

func (m *mockRuntime) RemoveImage(ctx context.Context, image string, force bool) error {
	m.recordCall("RemoveImage")
	if m.removeImageFn != nil {
		return m.removeImageFn(ctx, image, force)
	}
	return nil
}

func (m *mockRuntime) ContainerLogs(ctx context.Context, id string, opts LogOptions) (io.ReadCloser, error) {
	m.recordCall("ContainerLogs")
	if m.containerLogsFn != nil {
		return m.containerLogsFn(ctx, id, opts)
	}
	return io.NopCloser(nil), nil
}

func (m *mockRuntime) ExecInContainer(ctx context.Context, id string, cmd []string, opts ExecOptions) (*ExecResult, error) {
	m.recordCall("ExecInContainer")
	if m.execInContainerFn != nil {
		return m.execInContainerFn(ctx, id, cmd, opts)
	}
	return &ExecResult{ExitCode: 0}, nil
}

func (m *mockRuntime) CreateNetwork(ctx context.Context, name string, opts NetworkOptions) (string, error) {
	m.recordCall("CreateNetwork")
	if m.createNetworkFn != nil {
		return m.createNetworkFn(ctx, name, opts)
	}
	return "mock-network-id", nil
}

func (m *mockRuntime) RemoveNetwork(ctx context.Context, id string) error {
	m.recordCall("RemoveNetwork")
	if m.removeNetworkFn != nil {
		return m.removeNetworkFn(ctx, id)
	}
	return nil
}

func (m *mockRuntime) ConnectNetwork(ctx context.Context, networkID, containerID string) error {
	m.recordCall("ConnectNetwork")
	if m.connectNetworkFn != nil {
		return m.connectNetworkFn(ctx, networkID, containerID)
	}
	return nil
}

func (m *mockRuntime) DisconnectNetwork(ctx context.Context, networkID, containerID string) error {
	m.recordCall("DisconnectNetwork")
	if m.disconnectNetworkFn != nil {
		return m.disconnectNetworkFn(ctx, networkID, containerID)
	}
	return nil
}

func (m *mockRuntime) ListNetworks(ctx context.Context) ([]NetworkInfo, error) {
	m.recordCall("ListNetworks")
	if m.listNetworksFn != nil {
		return m.listNetworksFn(ctx)
	}
	return nil, nil
}

func (m *mockRuntime) CreateVolume(ctx context.Context, name string, opts VolumeOptions) (string, error) {
	m.recordCall("CreateVolume")
	if m.createVolumeFn != nil {
		return m.createVolumeFn(ctx, name, opts)
	}
	return name, nil
}

func (m *mockRuntime) RemoveVolume(ctx context.Context, name string, force bool) error {
	m.recordCall("RemoveVolume")
	if m.removeVolumeFn != nil {
		return m.removeVolumeFn(ctx, name, force)
	}
	return nil
}

func (m *mockRuntime) ListVolumes(ctx context.Context) ([]VolumeInfo, error) {
	m.recordCall("ListVolumes")
	if m.listVolumesFn != nil {
		return m.listVolumesFn(ctx)
	}
	return nil, nil
}

func (m *mockRuntime) InspectVolume(ctx context.Context, name string) (*VolumeInfo, error) {
	m.recordCall("InspectVolume")
	if m.inspectVolumeFn != nil {
		return m.inspectVolumeFn(ctx, name)
	}
	return &VolumeInfo{Name: name}, nil
}

func (m *mockRuntime) Ping(ctx context.Context) error {
	m.recordCall("Ping")
	if m.pingFn != nil {
		return m.pingFn(ctx)
	}
	return nil
}

func (m *mockRuntime) Info(ctx context.Context) (*RuntimeInfo, error) {
	m.recordCall("Info")
	if m.infoFn != nil {
		return m.infoFn(ctx)
	}
	return &RuntimeInfo{Name: "mock"}, nil
}

var _ Runtime = (*mockRuntime)(nil)

var errMock = fmt.Errorf("mock error")
