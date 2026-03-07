package service

import (
	"context"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/dokrypt/dokrypt/internal/container"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type mockRuntime struct {
	pullErr    error
	createErr  error
	createID   string
	startErr   error
	stopErr    error
	removeErr  error
	inspectErr error
	inspectInfo *container.ContainerInfo
	logsErr    error
	logsReader io.ReadCloser

	pulledImages   []string
	startedIDs     []string
	stoppedIDs     []string
	removedIDs     []string
	inspectedIDs   []string
	createdConfigs []*container.ContainerConfig
}

func newMockRuntime() *mockRuntime {
	return &mockRuntime{
		createID: "mock-container-123",
		inspectInfo: &container.ContainerInfo{
			ID:    "mock-container-123",
			State: "running",
			Ports: map[int]int{8080: 32000, 9090: 32001},
		},
	}
}

func (m *mockRuntime) CreateContainer(_ context.Context, cfg *container.ContainerConfig) (string, error) {
	m.createdConfigs = append(m.createdConfigs, cfg)
	return m.createID, m.createErr
}

func (m *mockRuntime) StartContainer(_ context.Context, id string) error {
	m.startedIDs = append(m.startedIDs, id)
	return m.startErr
}

func (m *mockRuntime) StopContainer(_ context.Context, id string, _ time.Duration) error {
	m.stoppedIDs = append(m.stoppedIDs, id)
	return m.stopErr
}

func (m *mockRuntime) RemoveContainer(_ context.Context, id string, _ bool) error {
	m.removedIDs = append(m.removedIDs, id)
	return m.removeErr
}

func (m *mockRuntime) ListContainers(_ context.Context, _ container.ListOptions) ([]container.ContainerInfo, error) {
	return nil, nil
}

func (m *mockRuntime) InspectContainer(_ context.Context, id string) (*container.ContainerInfo, error) {
	m.inspectedIDs = append(m.inspectedIDs, id)
	if m.inspectErr != nil {
		return nil, m.inspectErr
	}
	return m.inspectInfo, nil
}

func (m *mockRuntime) WaitContainer(_ context.Context, _ string) (int64, error) {
	return 0, nil
}

func (m *mockRuntime) PullImage(_ context.Context, image string) error {
	m.pulledImages = append(m.pulledImages, image)
	return m.pullErr
}

func (m *mockRuntime) BuildImage(_ context.Context, _ string, _ container.BuildOptions) (string, error) {
	return "", nil
}

func (m *mockRuntime) ListImages(_ context.Context) ([]container.ImageInfo, error) {
	return nil, nil
}

func (m *mockRuntime) RemoveImage(_ context.Context, _ string, _ bool) error {
	return nil
}

func (m *mockRuntime) ContainerLogs(_ context.Context, _ string, _ container.LogOptions) (io.ReadCloser, error) {
	if m.logsErr != nil {
		return nil, m.logsErr
	}
	if m.logsReader != nil {
		return m.logsReader, nil
	}
	return io.NopCloser(strings.NewReader("mock logs")), nil
}

func (m *mockRuntime) ExecInContainer(_ context.Context, _ string, _ []string, _ container.ExecOptions) (*container.ExecResult, error) {
	return &container.ExecResult{ExitCode: 0}, nil
}

func (m *mockRuntime) CreateNetwork(_ context.Context, _ string, _ container.NetworkOptions) (string, error) {
	return "net-123", nil
}

func (m *mockRuntime) RemoveNetwork(_ context.Context, _ string) error { return nil }
func (m *mockRuntime) ConnectNetwork(_ context.Context, _, _ string) error { return nil }
func (m *mockRuntime) DisconnectNetwork(_ context.Context, _, _ string) error { return nil }
func (m *mockRuntime) ListNetworks(_ context.Context) ([]container.NetworkInfo, error) {
	return nil, nil
}

func (m *mockRuntime) CreateVolume(_ context.Context, _ string, _ container.VolumeOptions) (string, error) {
	return "vol-123", nil
}

func (m *mockRuntime) RemoveVolume(_ context.Context, _ string, _ bool) error { return nil }
func (m *mockRuntime) ListVolumes(_ context.Context) ([]container.VolumeInfo, error) {
	return nil, nil
}
func (m *mockRuntime) InspectVolume(_ context.Context, _ string) (*container.VolumeInfo, error) {
	return nil, nil
}

func (m *mockRuntime) Ping(_ context.Context) error           { return nil }
func (m *mockRuntime) Info(_ context.Context) (*container.RuntimeInfo, error) {
	return &container.RuntimeInfo{Name: "mock"}, nil
}

func newTestBaseService(rt container.Runtime) *BaseService {
	return &BaseService{
		ServiceName:  "testservice",
		ServiceType:  "custom",
		Runtime:      rt,
		ProjectName:  "testproject",
		HostPorts:    make(map[string]int),
		ServiceURLs:  make(map[string]string),
		Dependencies: []string{"dep1", "dep2"},
	}
}

func TestBaseService_Name(t *testing.T) {
	bs := &BaseService{ServiceName: "myservice"}
	assert.Equal(t, "myservice", bs.Name())
}

func TestBaseService_Type(t *testing.T) {
	bs := &BaseService{ServiceType: "ipfs"}
	assert.Equal(t, "ipfs", bs.Type())
}

func TestBaseService_Ports(t *testing.T) {
	ports := map[string]int{"http": 8080, "rpc": 9090}
	bs := &BaseService{HostPorts: ports}
	assert.Equal(t, ports, bs.Ports())
}

func TestBaseService_Ports_Nil(t *testing.T) {
	bs := &BaseService{}
	assert.Nil(t, bs.Ports())
}

func TestBaseService_URLs(t *testing.T) {
	urls := map[string]string{"api": "http://localhost:8080"}
	bs := &BaseService{ServiceURLs: urls}
	assert.Equal(t, urls, bs.URLs())
}

func TestBaseService_URLs_Nil(t *testing.T) {
	bs := &BaseService{}
	assert.Nil(t, bs.URLs())
}

func TestBaseService_DependsOn(t *testing.T) {
	deps := []string{"db", "cache"}
	bs := &BaseService{Dependencies: deps}
	assert.Equal(t, deps, bs.DependsOn())
}

func TestBaseService_DependsOn_Empty(t *testing.T) {
	bs := &BaseService{}
	assert.Nil(t, bs.DependsOn())
}

func TestBaseService_GetContainerID(t *testing.T) {
	bs := &BaseService{ContainerID: "abc123"}
	assert.Equal(t, "abc123", bs.GetContainerID())
}

func TestBaseService_GetContainerID_Empty(t *testing.T) {
	bs := &BaseService{}
	assert.Equal(t, "", bs.GetContainerID())
}

func TestBaseService_StartContainer_Success(t *testing.T) {
	rt := newMockRuntime()
	bs := newTestBaseService(rt)
	ctx := context.Background()

	cfg := &container.ContainerConfig{
		Image: "nginx:latest",
		Ports: map[int]int{8080: 0},
	}

	err := bs.StartContainer(ctx, cfg)
	require.NoError(t, err)

	assert.Equal(t, "mock-container-123", bs.ContainerID)

	assert.Equal(t, "dokrypt-testproject-testservice", cfg.Name)
	assert.Equal(t, "testproject", cfg.Labels["dokrypt.project"])
	assert.Equal(t, "testservice", cfg.Labels["dokrypt.service"])
	assert.Equal(t, "custom", cfg.Labels["dokrypt.type"])

	assert.Contains(t, cfg.Networks, "dokrypt-testproject")
	assert.Contains(t, cfg.NetworkAliases["dokrypt-testproject"], "testservice")

	assert.Equal(t, 32000, bs.HostPorts["8080"])
	assert.Equal(t, 32001, bs.HostPorts["9090"])
}

func TestBaseService_StartContainer_PreservesExistingNetworks(t *testing.T) {
	rt := newMockRuntime()
	bs := newTestBaseService(rt)
	ctx := context.Background()

	cfg := &container.ContainerConfig{
		Image:    "nginx:latest",
		Networks: []string{"custom-network"},
	}

	err := bs.StartContainer(ctx, cfg)
	require.NoError(t, err)

	assert.Equal(t, []string{"custom-network"}, cfg.Networks)
}

func TestBaseService_StartContainer_SetsLabelsWhenNil(t *testing.T) {
	rt := newMockRuntime()
	bs := newTestBaseService(rt)
	ctx := context.Background()

	cfg := &container.ContainerConfig{
		Image:  "nginx:latest",
		Labels: nil,
	}

	err := bs.StartContainer(ctx, cfg)
	require.NoError(t, err)

	assert.NotNil(t, cfg.Labels)
	assert.Equal(t, "testproject", cfg.Labels["dokrypt.project"])
}

func TestBaseService_StartContainer_CreateFails(t *testing.T) {
	rt := newMockRuntime()
	rt.createErr = errors.New("create failed")
	bs := newTestBaseService(rt)
	ctx := context.Background()

	cfg := &container.ContainerConfig{Image: "nginx:latest"}
	err := bs.StartContainer(ctx, cfg)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "failed to create container")
	assert.Contains(t, err.Error(), "testservice")
}

func TestBaseService_StartContainer_StartFails(t *testing.T) {
	rt := newMockRuntime()
	rt.startErr = errors.New("start failed")
	bs := newTestBaseService(rt)
	ctx := context.Background()

	cfg := &container.ContainerConfig{Image: "nginx:latest"}
	err := bs.StartContainer(ctx, cfg)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "failed to start container")
}

func TestBaseService_StartContainer_InspectFails(t *testing.T) {
	rt := newMockRuntime()
	rt.inspectErr = errors.New("inspect failed")
	bs := newTestBaseService(rt)
	ctx := context.Background()

	cfg := &container.ContainerConfig{Image: "nginx:latest"}
	err := bs.StartContainer(ctx, cfg)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "failed to inspect container")
}

func TestBaseService_StartContainer_PullImageFailsContinues(t *testing.T) {
	rt := newMockRuntime()
	rt.pullErr = errors.New("pull failed")
	bs := newTestBaseService(rt)
	ctx := context.Background()

	cfg := &container.ContainerConfig{Image: "nginx:latest"}
	err := bs.StartContainer(ctx, cfg)
	require.NoError(t, err)
	assert.Contains(t, rt.pulledImages, "nginx:latest")
}

func TestBaseService_StartContainer_InitializesHostPortsIfNil(t *testing.T) {
	rt := newMockRuntime()
	bs := newTestBaseService(rt)
	bs.HostPorts = nil
	ctx := context.Background()

	cfg := &container.ContainerConfig{Image: "nginx:latest"}
	err := bs.StartContainer(ctx, cfg)
	require.NoError(t, err)
	assert.NotNil(t, bs.HostPorts)
}

func TestBaseService_StopContainer_Success(t *testing.T) {
	rt := newMockRuntime()
	bs := newTestBaseService(rt)
	bs.ContainerID = "abc123"
	ctx := context.Background()

	err := bs.StopContainer(ctx)
	require.NoError(t, err)
	assert.Equal(t, "", bs.ContainerID)
	assert.Contains(t, rt.stoppedIDs, "abc123")
	assert.Contains(t, rt.removedIDs, "abc123")
}

func TestBaseService_StopContainer_NoContainer(t *testing.T) {
	rt := newMockRuntime()
	bs := newTestBaseService(rt)
	bs.ContainerID = ""
	ctx := context.Background()

	err := bs.StopContainer(ctx)
	require.NoError(t, err)
	assert.Empty(t, rt.stoppedIDs)
}

func TestBaseService_StopContainer_RemoveFails(t *testing.T) {
	rt := newMockRuntime()
	rt.removeErr = errors.New("remove failed")
	bs := newTestBaseService(rt)
	bs.ContainerID = "abc123"
	ctx := context.Background()

	err := bs.StopContainer(ctx)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "failed to remove container")
}

func TestBaseService_StopContainer_StopFailsContinuesToRemove(t *testing.T) {
	rt := newMockRuntime()
	rt.stopErr = errors.New("stop failed")
	bs := newTestBaseService(rt)
	bs.ContainerID = "abc123"
	ctx := context.Background()

	err := bs.StopContainer(ctx)
	require.NoError(t, err)
	assert.Contains(t, rt.removedIDs, "abc123")
	assert.Equal(t, "", bs.ContainerID)
}

func TestBaseService_IsContainerRunning_True(t *testing.T) {
	rt := newMockRuntime()
	rt.inspectInfo = &container.ContainerInfo{State: "running"}
	bs := newTestBaseService(rt)
	bs.ContainerID = "abc123"

	assert.True(t, bs.IsContainerRunning(context.Background()))
}

func TestBaseService_IsContainerRunning_False_NotRunning(t *testing.T) {
	rt := newMockRuntime()
	rt.inspectInfo = &container.ContainerInfo{State: "exited"}
	bs := newTestBaseService(rt)
	bs.ContainerID = "abc123"

	assert.False(t, bs.IsContainerRunning(context.Background()))
}

func TestBaseService_IsContainerRunning_False_NoContainerID(t *testing.T) {
	rt := newMockRuntime()
	bs := newTestBaseService(rt)
	bs.ContainerID = ""

	assert.False(t, bs.IsContainerRunning(context.Background()))
}

func TestBaseService_IsContainerRunning_False_InspectError(t *testing.T) {
	rt := newMockRuntime()
	rt.inspectErr = errors.New("inspect error")
	bs := newTestBaseService(rt)
	bs.ContainerID = "abc123"

	assert.False(t, bs.IsContainerRunning(context.Background()))
}

func TestBaseService_HTTPHealthCheck_Success(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
		fmt.Fprint(w, "ok")
	}))
	defer srv.Close()

	bs := &BaseService{}
	err := bs.HTTPHealthCheck(context.Background(), srv.URL)
	require.NoError(t, err)
}

func TestBaseService_HTTPHealthCheck_ServerError(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	}))
	defer srv.Close()

	bs := &BaseService{}
	err := bs.HTTPHealthCheck(context.Background(), srv.URL)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "health check returned status 500")
}

func TestBaseService_HTTPHealthCheck_Status399_OK(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(399)
	}))
	defer srv.Close()

	bs := &BaseService{}
	err := bs.HTTPHealthCheck(context.Background(), srv.URL)
	require.NoError(t, err)
}

func TestBaseService_HTTPHealthCheck_Status400_Error(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(400)
	}))
	defer srv.Close()

	bs := &BaseService{}
	err := bs.HTTPHealthCheck(context.Background(), srv.URL)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "health check returned status 400")
}

func TestBaseService_HTTPHealthCheck_ConnectionRefused(t *testing.T) {
	bs := &BaseService{}
	err := bs.HTTPHealthCheck(context.Background(), "http://127.0.0.1:1")
	require.Error(t, err)
}

func TestBaseService_HTTPHealthCheck_InvalidURL(t *testing.T) {
	bs := &BaseService{}
	err := bs.HTTPHealthCheck(context.Background(), "://bad-url")
	require.Error(t, err)
}

func TestBaseService_WaitForHealthy_ImmediateSuccess(t *testing.T) {
	bs := &BaseService{}
	called := 0
	err := bs.WaitForHealthy(context.Background(), func(_ context.Context) error {
		called++
		return nil
	})
	require.NoError(t, err)
	assert.Equal(t, 1, called)
}

func TestBaseService_WaitForHealthy_EventualSuccess(t *testing.T) {
	bs := &BaseService{}
	calls := 0
	err := bs.WaitForHealthy(context.Background(), func(_ context.Context) error {
		calls++
		if calls < 3 {
			return errors.New("not ready yet")
		}
		return nil
	})
	require.NoError(t, err)
	assert.Equal(t, 3, calls)
}

func TestBaseService_WaitForHealthy_ContextCancelled(t *testing.T) {
	bs := &BaseService{}
	ctx, cancel := context.WithCancel(context.Background())
	cancel() // cancel immediately

	err := bs.WaitForHealthy(ctx, func(_ context.Context) error {
		return errors.New("unhealthy")
	})
	require.Error(t, err)
}

func TestBaseService_ContainerLogs_Success(t *testing.T) {
	rt := newMockRuntime()
	rt.logsReader = io.NopCloser(strings.NewReader("line 1\nline 2\n"))
	bs := newTestBaseService(rt)
	bs.ContainerID = "abc123"

	reader, err := bs.ContainerLogs(context.Background(), LogOptions{
		Follow:     true,
		Tail:       "100",
		Since:      "1h",
		Timestamps: true,
	})
	require.NoError(t, err)
	defer reader.Close()

	data, err := io.ReadAll(reader)
	require.NoError(t, err)
	assert.Equal(t, "line 1\nline 2\n", string(data))
}

func TestBaseService_ContainerLogs_NoContainerID(t *testing.T) {
	rt := newMockRuntime()
	bs := newTestBaseService(rt)
	bs.ContainerID = ""

	_, err := bs.ContainerLogs(context.Background(), LogOptions{})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "not started")
}

func TestBaseService_ContainerLogs_RuntimeError(t *testing.T) {
	rt := newMockRuntime()
	rt.logsErr = errors.New("log stream failed")
	bs := newTestBaseService(rt)
	bs.ContainerID = "abc123"

	_, err := bs.ContainerLogs(context.Background(), LogOptions{})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "log stream failed")
}
