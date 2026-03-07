package service

import (
	"context"
	"errors"
	"io"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type mockService struct {
	name        string
	svcType     string
	deps        []string
	ports       map[string]int
	urls        map[string]string
	running     bool
	healthy     bool
	startErr    error
	stopErr     error
	restartErr  error
	healthErr   error

	startCalls   int
	stopCalls    int
	restartCalls int
}

func newMockService(name string, deps ...string) *mockService {
	return &mockService{
		name:    name,
		svcType: "mock",
		deps:    deps,
		ports:   map[string]int{"http": 8080},
		urls:    map[string]string{"http": "http://localhost:8080"},
		running: false,
		healthy: true,
	}
}

func (m *mockService) Name() string                { return m.name }
func (m *mockService) Type() string                { return m.svcType }
func (m *mockService) Ports() map[string]int       { return m.ports }
func (m *mockService) URLs() map[string]string     { return m.urls }
func (m *mockService) DependsOn() []string         { return m.deps }

func (m *mockService) Start(_ context.Context) error {
	m.startCalls++
	if m.startErr != nil {
		return m.startErr
	}
	m.running = true
	return nil
}

func (m *mockService) Stop(_ context.Context) error {
	m.stopCalls++
	if m.stopErr != nil {
		return m.stopErr
	}
	m.running = false
	return nil
}

func (m *mockService) Restart(_ context.Context) error {
	m.restartCalls++
	if m.restartErr != nil {
		return m.restartErr
	}
	m.running = true
	return nil
}

func (m *mockService) IsRunning(_ context.Context) bool {
	return m.running
}

func (m *mockService) Health(_ context.Context) error {
	if !m.healthy {
		return errors.New("unhealthy")
	}
	return m.healthErr
}

func (m *mockService) Logs(_ context.Context, _ LogOptions) (io.ReadCloser, error) {
	return io.NopCloser(strings.NewReader("mock logs")), nil
}

func TestNewOrchestrator(t *testing.T) {
	o := NewOrchestrator()
	require.NotNil(t, o)
	assert.Empty(t, o.All())
}

func TestOrchestrator_Register(t *testing.T) {
	o := NewOrchestrator()
	svc := newMockService("svc1")

	o.Register(svc)

	all := o.All()
	require.Len(t, all, 1)
	assert.Equal(t, "svc1", all[0].Name())
}

func TestOrchestrator_Register_Multiple(t *testing.T) {
	o := NewOrchestrator()
	o.Register(newMockService("svc1"))
	o.Register(newMockService("svc2"))
	o.Register(newMockService("svc3"))

	assert.Len(t, o.All(), 3)
}

func TestOrchestrator_Register_OverwritesSameName(t *testing.T) {
	o := NewOrchestrator()

	svc1 := newMockService("myname")
	svc1.svcType = "type-a"
	o.Register(svc1)

	svc2 := newMockService("myname")
	svc2.svcType = "type-b"
	o.Register(svc2)

	all := o.All()
	require.Len(t, all, 1)
	assert.Equal(t, "type-b", all[0].Type())
}

func TestOrchestrator_Get_Found(t *testing.T) {
	o := NewOrchestrator()
	o.Register(newMockService("alpha"))
	o.Register(newMockService("beta"))

	svc, err := o.Get("alpha")
	require.NoError(t, err)
	assert.Equal(t, "alpha", svc.Name())
}

func TestOrchestrator_Get_NotFound(t *testing.T) {
	o := NewOrchestrator()
	o.Register(newMockService("alpha"))

	_, err := o.Get("nonexistent")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "not found")
}

func TestOrchestrator_Get_EmptyOrchestrator(t *testing.T) {
	o := NewOrchestrator()
	_, err := o.Get("anything")
	require.Error(t, err)
}

func TestOrchestrator_All_Empty(t *testing.T) {
	o := NewOrchestrator()
	assert.Empty(t, o.All())
}

func TestOrchestrator_All_ReturnsAllServices(t *testing.T) {
	o := NewOrchestrator()
	o.Register(newMockService("a"))
	o.Register(newMockService("b"))
	o.Register(newMockService("c"))

	all := o.All()
	assert.Len(t, all, 3)

	names := make(map[string]bool)
	for _, svc := range all {
		names[svc.Name()] = true
	}
	assert.True(t, names["a"])
	assert.True(t, names["b"])
	assert.True(t, names["c"])
}

func TestOrchestrator_StartAll_NoDependencies(t *testing.T) {
	o := NewOrchestrator()
	svc1 := newMockService("a")
	svc2 := newMockService("b")
	o.Register(svc1)
	o.Register(svc2)

	err := o.StartAll(context.Background())
	require.NoError(t, err)

	assert.Equal(t, 1, svc1.startCalls)
	assert.Equal(t, 1, svc2.startCalls)
	assert.True(t, svc1.running)
	assert.True(t, svc2.running)
}

func TestOrchestrator_StartAll_WithDependencies(t *testing.T) {
	o := NewOrchestrator()
	db := newMockService("db")
	api := newMockService("api", "db")
	web := newMockService("web", "api")
	o.Register(db)
	o.Register(api)
	o.Register(web)

	err := o.StartAll(context.Background())
	require.NoError(t, err)

	assert.True(t, db.running)
	assert.True(t, api.running)
	assert.True(t, web.running)
}

func TestOrchestrator_StartAll_Empty(t *testing.T) {
	o := NewOrchestrator()
	err := o.StartAll(context.Background())
	require.NoError(t, err)
}

func TestOrchestrator_StartAll_ServiceStartFails(t *testing.T) {
	o := NewOrchestrator()
	svc := newMockService("failing")
	svc.startErr = errors.New("boom")
	o.Register(svc)

	err := o.StartAll(context.Background())
	require.Error(t, err)
	assert.Contains(t, err.Error(), "failing")
}

func TestOrchestrator_StartAll_CyclicDependency(t *testing.T) {
	o := NewOrchestrator()
	o.Register(newMockService("a", "b"))
	o.Register(newMockService("b", "a"))

	err := o.StartAll(context.Background())
	require.Error(t, err)
	assert.Contains(t, err.Error(), "cycle")
}

func TestOrchestrator_StopAll(t *testing.T) {
	o := NewOrchestrator()
	db := newMockService("db")
	db.running = true
	api := newMockService("api", "db")
	api.running = true
	o.Register(db)
	o.Register(api)

	err := o.StopAll(context.Background())
	require.NoError(t, err)

	assert.False(t, db.running)
	assert.False(t, api.running)
}

func TestOrchestrator_StopAll_Empty(t *testing.T) {
	o := NewOrchestrator()
	err := o.StopAll(context.Background())
	require.NoError(t, err)
}

func TestOrchestrator_StopAll_StopErrorDoesNotAbort(t *testing.T) {
	o := NewOrchestrator()
	svc := newMockService("flaky")
	svc.running = true
	svc.stopErr = errors.New("stop error")
	o.Register(svc)

	err := o.StopAll(context.Background())
	require.NoError(t, err)
}

func TestOrchestrator_StopAll_CyclicDependencyFallback(t *testing.T) {
	o := NewOrchestrator()
	a := newMockService("a", "b")
	a.running = true
	b := newMockService("b", "a")
	b.running = true
	o.Register(a)
	o.Register(b)

	err := o.StopAll(context.Background())
	require.NoError(t, err)

	assert.False(t, a.running)
	assert.False(t, b.running)
}

func TestOrchestrator_StartService_Found(t *testing.T) {
	o := NewOrchestrator()
	svc := newMockService("alpha")
	o.Register(svc)

	err := o.StartService(context.Background(), "alpha")
	require.NoError(t, err)
	assert.Equal(t, 1, svc.startCalls)
	assert.True(t, svc.running)
}

func TestOrchestrator_StartService_NotFound(t *testing.T) {
	o := NewOrchestrator()
	err := o.StartService(context.Background(), "nonexistent")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "not found")
}

func TestOrchestrator_StartService_Error(t *testing.T) {
	o := NewOrchestrator()
	svc := newMockService("broken")
	svc.startErr = errors.New("cannot start")
	o.Register(svc)

	err := o.StartService(context.Background(), "broken")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "cannot start")
}

func TestOrchestrator_StopService_Found(t *testing.T) {
	o := NewOrchestrator()
	svc := newMockService("alpha")
	svc.running = true
	o.Register(svc)

	err := o.StopService(context.Background(), "alpha")
	require.NoError(t, err)
	assert.Equal(t, 1, svc.stopCalls)
	assert.False(t, svc.running)
}

func TestOrchestrator_StopService_NotFound(t *testing.T) {
	o := NewOrchestrator()
	err := o.StopService(context.Background(), "ghost")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "not found")
}

func TestOrchestrator_StopService_Error(t *testing.T) {
	o := NewOrchestrator()
	svc := newMockService("sticky")
	svc.stopErr = errors.New("won't stop")
	svc.running = true
	o.Register(svc)

	err := o.StopService(context.Background(), "sticky")
	require.Error(t, err)
}

func TestOrchestrator_Status_Empty(t *testing.T) {
	o := NewOrchestrator()
	statuses := o.Status(context.Background())
	assert.Empty(t, statuses)
}

func TestOrchestrator_Status_RunningAndHealthy(t *testing.T) {
	o := NewOrchestrator()
	svc := newMockService("healthy-svc")
	svc.running = true
	svc.healthy = true
	o.Register(svc)

	statuses := o.Status(context.Background())
	require.Len(t, statuses, 1)
	assert.Equal(t, "healthy-svc", statuses[0].Name)
	assert.Equal(t, "mock", statuses[0].Type)
	assert.Equal(t, "running", statuses[0].Status)
	assert.True(t, statuses[0].Healthy)
	assert.Equal(t, svc.ports, statuses[0].Ports)
	assert.Equal(t, svc.urls, statuses[0].URLs)
}

func TestOrchestrator_Status_RunningButUnhealthy(t *testing.T) {
	o := NewOrchestrator()
	svc := newMockService("sick-svc")
	svc.running = true
	svc.healthy = false
	o.Register(svc)

	statuses := o.Status(context.Background())
	require.Len(t, statuses, 1)
	assert.Equal(t, "unhealthy", statuses[0].Status)
	assert.False(t, statuses[0].Healthy)
}

func TestOrchestrator_Status_Stopped(t *testing.T) {
	o := NewOrchestrator()
	svc := newMockService("stopped-svc")
	svc.running = false
	o.Register(svc)

	statuses := o.Status(context.Background())
	require.Len(t, statuses, 1)
	assert.Equal(t, "stopped", statuses[0].Status)
	assert.False(t, statuses[0].Healthy)
}

func TestOrchestrator_Status_MultipleServices(t *testing.T) {
	o := NewOrchestrator()

	running := newMockService("running")
	running.running = true
	running.healthy = true
	o.Register(running)

	stopped := newMockService("stopped")
	stopped.running = false
	o.Register(stopped)

	unhealthy := newMockService("unhealthy")
	unhealthy.running = true
	unhealthy.healthy = false
	o.Register(unhealthy)

	statuses := o.Status(context.Background())
	require.Len(t, statuses, 3)

	statusMap := make(map[string]ServiceStatus)
	for _, s := range statuses {
		statusMap[s.Name] = s
	}

	assert.Equal(t, "running", statusMap["running"].Status)
	assert.True(t, statusMap["running"].Healthy)
	assert.Equal(t, "stopped", statusMap["stopped"].Status)
	assert.False(t, statusMap["stopped"].Healthy)
	assert.Equal(t, "unhealthy", statusMap["unhealthy"].Status)
	assert.False(t, statusMap["unhealthy"].Healthy)
}
