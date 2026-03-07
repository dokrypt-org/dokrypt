package service

import (
	"context"
	"io"
	"strings"
	"testing"

	"github.com/dokrypt/dokrypt/internal/config"
	"github.com/dokrypt/dokrypt/internal/container"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewCustomService_BasicCreation(t *testing.T) {
	rt := newMockRuntime()
	cfg := config.ServiceConfig{
		Type:      "custom",
		Image:     "myimage:latest",
		DependsOn: []string{"db"},
	}

	svc, err := NewCustomService("mysvc", cfg, rt, "proj1")
	require.NoError(t, err)
	require.NotNil(t, svc)

	assert.Equal(t, "mysvc", svc.Name())
	assert.Equal(t, "custom", svc.Type())
	assert.Equal(t, []string{"db"}, svc.DependsOn())
	assert.NotNil(t, svc.Ports())
	assert.NotNil(t, svc.URLs())
	assert.Empty(t, svc.GetContainerID())
}

func TestNewCustomService_NoDependencies(t *testing.T) {
	rt := newMockRuntime()
	cfg := config.ServiceConfig{
		Type:  "custom",
		Image: "nginx",
	}

	svc, err := NewCustomService("web", cfg, rt, "proj1")
	require.NoError(t, err)
	assert.Nil(t, svc.DependsOn())
}

func TestNewCustomService_PreservesConfig(t *testing.T) {
	rt := newMockRuntime()
	cfg := config.ServiceConfig{
		Type:  "custom",
		Image: "postgres:15",
		Port:  5432,
		Ports: map[string]int{"db": 5432, "admin": 8080},
		Environment: map[string]string{
			"POSTGRES_PASSWORD": "test",
		},
		Volumes:   []string{"/data:/var/lib/postgresql/data"},
		Command:   []string{"postgres", "-c", "log_statement=all"},
		DependsOn: []string{"network"},
	}

	svc, err := NewCustomService("postgres", cfg, rt, "myproj")
	require.NoError(t, err)
	assert.Equal(t, "postgres", svc.Name())
	assert.Equal(t, "custom", svc.Type())
	assert.Equal(t, "myproj", svc.ProjectName)
}

func TestCustomFactory_ReturnsService(t *testing.T) {
	rt := newMockRuntime()
	cfg := config.ServiceConfig{
		Type:  "custom",
		Image: "redis:7",
	}

	svc, err := CustomFactory("redis", cfg, rt, "proj")
	require.NoError(t, err)
	require.NotNil(t, svc)

	var _ Service = svc
	assert.Equal(t, "redis", svc.Name())
	assert.Equal(t, "custom", svc.Type())
}

func TestCustomService_Start_BuildsContainerConfig(t *testing.T) {
	rt := newMockRuntime()
	cfg := config.ServiceConfig{
		Type:    "custom",
		Image:   "myapp:v1",
		Port:    3000,
		Ports:   map[string]int{"http": 8080, "grpc": 9090},
		Command: []string{"./start.sh"},
		Environment: map[string]string{
			"MODE": "production",
		},
		Volumes: []string{
			"/host/data:/container/data",
			"/host/config:/container/config:ro",
			"/shared",
		},
	}

	svc, err := NewCustomService("app", cfg, rt, "proj")
	require.NoError(t, err)

	err = svc.Start(context.Background())
	require.NoError(t, err)

	require.Len(t, rt.createdConfigs, 1)
	cc := rt.createdConfigs[0]

	assert.Equal(t, "myapp:v1", cc.Image)
	assert.Equal(t, []string{"./start.sh"}, cc.Command)

	assert.Contains(t, cc.Ports, 3000)
	assert.Contains(t, cc.Ports, 8080)
	assert.Contains(t, cc.Ports, 9090)

	assert.Equal(t, "production", cc.Env["MODE"])

	require.Len(t, cc.Volumes, 3)

	assert.Equal(t, "/host/data", cc.Volumes[0].Source)
	assert.Equal(t, "/container/data", cc.Volumes[0].Target)
	assert.False(t, cc.Volumes[0].ReadOnly)

	assert.Equal(t, "/host/config", cc.Volumes[1].Source)
	assert.Equal(t, "/container/config", cc.Volumes[1].Target)
	assert.True(t, cc.Volumes[1].ReadOnly)

	assert.Equal(t, "/shared", cc.Volumes[2].Source)
	assert.Equal(t, "/shared", cc.Volumes[2].Target)
	assert.False(t, cc.Volumes[2].ReadOnly)
}

func TestCustomService_Start_NoPorts(t *testing.T) {
	rt := newMockRuntime()
	cfg := config.ServiceConfig{
		Type:  "custom",
		Image: "worker:latest",
	}

	svc, err := NewCustomService("worker", cfg, rt, "proj")
	require.NoError(t, err)

	err = svc.Start(context.Background())
	require.NoError(t, err)

	cc := rt.createdConfigs[0]
	assert.Len(t, cc.Ports, 0) // empty map, no ports mapped
}

func TestCustomService_Start_NoVolumes(t *testing.T) {
	rt := newMockRuntime()
	cfg := config.ServiceConfig{
		Type:  "custom",
		Image: "worker:latest",
	}

	svc, err := NewCustomService("worker", cfg, rt, "proj")
	require.NoError(t, err)

	err = svc.Start(context.Background())
	require.NoError(t, err)

	cc := rt.createdConfigs[0]
	assert.Nil(t, cc.Volumes)
}

func TestCustomService_Start_VolumeThreePartRW(t *testing.T) {
	rt := newMockRuntime()
	cfg := config.ServiceConfig{
		Type:    "custom",
		Image:   "app:latest",
		Volumes: []string{"/src:/dst:rw"},
	}

	svc, err := NewCustomService("app", cfg, rt, "proj")
	require.NoError(t, err)

	err = svc.Start(context.Background())
	require.NoError(t, err)

	cc := rt.createdConfigs[0]
	require.Len(t, cc.Volumes, 1)
	assert.Equal(t, "/src", cc.Volumes[0].Source)
	assert.Equal(t, "/dst", cc.Volumes[0].Target)
	assert.False(t, cc.Volumes[0].ReadOnly) // "rw" != "ro"
}

func TestCustomService_Stop(t *testing.T) {
	rt := newMockRuntime()
	cfg := config.ServiceConfig{Type: "custom", Image: "app:latest"}
	svc, _ := NewCustomService("app", cfg, rt, "proj")
	svc.ContainerID = "c123"

	err := svc.Stop(context.Background())
	require.NoError(t, err)
	assert.Equal(t, "", svc.ContainerID)
}

func TestCustomService_Restart(t *testing.T) {
	rt := newMockRuntime()
	cfg := config.ServiceConfig{Type: "custom", Image: "app:latest"}
	svc, _ := NewCustomService("app", cfg, rt, "proj")

	err := svc.Start(context.Background())
	require.NoError(t, err)

	err = svc.Restart(context.Background())
	require.NoError(t, err)

	assert.Len(t, rt.createdConfigs, 2)
}

func TestCustomService_IsRunning(t *testing.T) {
	rt := newMockRuntime()
	rt.inspectInfo = &container.ContainerInfo{State: "running"}
	cfg := config.ServiceConfig{Type: "custom", Image: "app:latest"}
	svc, _ := NewCustomService("app", cfg, rt, "proj")
	svc.ContainerID = "c123"

	assert.True(t, svc.IsRunning(context.Background()))
}

func TestCustomService_IsRunning_NotStarted(t *testing.T) {
	rt := newMockRuntime()
	cfg := config.ServiceConfig{Type: "custom", Image: "app:latest"}
	svc, _ := NewCustomService("app", cfg, rt, "proj")

	assert.False(t, svc.IsRunning(context.Background()))
}

func TestCustomService_Health_NoHealthcheck(t *testing.T) {
	rt := newMockRuntime()
	cfg := config.ServiceConfig{Type: "custom", Image: "app:latest"}
	svc, _ := NewCustomService("app", cfg, rt, "proj")

	err := svc.Health(context.Background())
	require.NoError(t, err)
}

func TestCustomService_Health_NilHealthcheck(t *testing.T) {
	rt := newMockRuntime()
	cfg := config.ServiceConfig{
		Type:        "custom",
		Image:       "app:latest",
		Healthcheck: nil,
	}
	svc, _ := NewCustomService("app", cfg, rt, "proj")

	err := svc.Health(context.Background())
	require.NoError(t, err)
}

func TestCustomService_Health_EmptyHTTP(t *testing.T) {
	rt := newMockRuntime()
	cfg := config.ServiceConfig{
		Type:        "custom",
		Image:       "app:latest",
		Healthcheck: &config.HealthcheckConfig{HTTP: ""},
	}
	svc, _ := NewCustomService("app", cfg, rt, "proj")

	err := svc.Health(context.Background())
	require.NoError(t, err)
}

func TestCustomService_Health_NoURLs(t *testing.T) {
	rt := newMockRuntime()
	cfg := config.ServiceConfig{
		Type:        "custom",
		Image:       "app:latest",
		Healthcheck: &config.HealthcheckConfig{HTTP: "/health"},
	}
	svc, _ := NewCustomService("app", cfg, rt, "proj")

	err := svc.Health(context.Background())
	require.NoError(t, err)
}

func TestCustomService_Logs(t *testing.T) {
	rt := newMockRuntime()
	rt.logsReader = io.NopCloser(strings.NewReader("service log"))
	cfg := config.ServiceConfig{Type: "custom", Image: "app:latest"}
	svc, _ := NewCustomService("app", cfg, rt, "proj")
	svc.ContainerID = "c123"

	reader, err := svc.Logs(context.Background(), LogOptions{Follow: true})
	require.NoError(t, err)
	defer reader.Close()

	data, err := io.ReadAll(reader)
	require.NoError(t, err)
	assert.Equal(t, "service log", string(data))
}

func TestCustomService_Logs_NotStarted(t *testing.T) {
	rt := newMockRuntime()
	cfg := config.ServiceConfig{Type: "custom", Image: "app:latest"}
	svc, _ := NewCustomService("app", cfg, rt, "proj")

	_, err := svc.Logs(context.Background(), LogOptions{})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "not started")
}
