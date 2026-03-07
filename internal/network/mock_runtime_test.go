package network

import (
	"context"
	"fmt"
	"io"
	"time"

	"github.com/dokrypt/dokrypt/internal/container"
)

type mockRuntime struct {
	createNetworkFn     func(ctx context.Context, name string, opts container.NetworkOptions) (string, error)
	removeNetworkFn     func(ctx context.Context, id string) error
	connectNetworkFn    func(ctx context.Context, networkID, containerID string) error
	disconnectNetworkFn func(ctx context.Context, networkID, containerID string) error
	listNetworksFn      func(ctx context.Context) ([]container.NetworkInfo, error)

	calls map[string]int
}

func newMockRuntime() *mockRuntime {
	return &mockRuntime{
		calls: make(map[string]int),
	}
}

func (m *mockRuntime) recordCall(name string) {
	m.calls[name]++
}

func (m *mockRuntime) CreateNetwork(ctx context.Context, name string, opts container.NetworkOptions) (string, error) {
	m.recordCall("CreateNetwork")
	if m.createNetworkFn != nil {
		return m.createNetworkFn(ctx, name, opts)
	}
	return "net-" + name, nil
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

func (m *mockRuntime) ListNetworks(ctx context.Context) ([]container.NetworkInfo, error) {
	m.recordCall("ListNetworks")
	if m.listNetworksFn != nil {
		return m.listNetworksFn(ctx)
	}
	return nil, nil
}

func (m *mockRuntime) CreateContainer(ctx context.Context, cfg *container.ContainerConfig) (string, error) {
	return "mock-ctr", nil
}
func (m *mockRuntime) StartContainer(ctx context.Context, id string) error { return nil }
func (m *mockRuntime) StopContainer(ctx context.Context, id string, timeout time.Duration) error {
	return nil
}
func (m *mockRuntime) RemoveContainer(ctx context.Context, id string, force bool) error { return nil }
func (m *mockRuntime) ListContainers(ctx context.Context, opts container.ListOptions) ([]container.ContainerInfo, error) {
	return nil, nil
}
func (m *mockRuntime) InspectContainer(ctx context.Context, id string) (*container.ContainerInfo, error) {
	return &container.ContainerInfo{ID: id}, nil
}
func (m *mockRuntime) WaitContainer(ctx context.Context, id string) (int64, error) { return 0, nil }
func (m *mockRuntime) PullImage(ctx context.Context, image string) error             { return nil }
func (m *mockRuntime) BuildImage(ctx context.Context, contextPath string, opts container.BuildOptions) (string, error) {
	return "img", nil
}
func (m *mockRuntime) ListImages(ctx context.Context) ([]container.ImageInfo, error) { return nil, nil }
func (m *mockRuntime) RemoveImage(ctx context.Context, image string, force bool) error { return nil }
func (m *mockRuntime) ContainerLogs(ctx context.Context, id string, opts container.LogOptions) (io.ReadCloser, error) {
	return io.NopCloser(nil), nil
}
func (m *mockRuntime) ExecInContainer(ctx context.Context, id string, cmd []string, opts container.ExecOptions) (*container.ExecResult, error) {
	return &container.ExecResult{}, nil
}
func (m *mockRuntime) CreateVolume(ctx context.Context, name string, opts container.VolumeOptions) (string, error) {
	return name, nil
}
func (m *mockRuntime) RemoveVolume(ctx context.Context, name string, force bool) error { return nil }
func (m *mockRuntime) ListVolumes(ctx context.Context) ([]container.VolumeInfo, error) {
	return nil, nil
}
func (m *mockRuntime) InspectVolume(ctx context.Context, name string) (*container.VolumeInfo, error) {
	return &container.VolumeInfo{Name: name}, nil
}
func (m *mockRuntime) Ping(ctx context.Context) error { return nil }
func (m *mockRuntime) Info(ctx context.Context) (*container.RuntimeInfo, error) {
	return &container.RuntimeInfo{Name: "mock"}, nil
}

var _ container.Runtime = (*mockRuntime)(nil)

var errMock = fmt.Errorf("mock error")
