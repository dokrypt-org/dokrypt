package service

import (
	"errors"
	"sort"
	"testing"

	"github.com/dokrypt/dokrypt/internal/config"
	"github.com/dokrypt/dokrypt/internal/container"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func dummyFactory(svcType string) Factory {
	return func(name string, _ config.ServiceConfig, _ container.Runtime, _ string) (Service, error) {
		return &mockService{name: name, svcType: svcType}, nil
	}
}

func failingFactory() Factory {
	return func(_ string, _ config.ServiceConfig, _ container.Runtime, _ string) (Service, error) {
		return nil, errors.New("factory error")
	}
}

func TestNewRegistry(t *testing.T) {
	r := NewRegistry()
	require.NotNil(t, r)
	assert.Empty(t, r.Types())
}

func TestRegistry_Register_SingleType(t *testing.T) {
	r := NewRegistry()
	r.Register("custom", dummyFactory("custom"))

	assert.True(t, r.HasType("custom"))
	assert.Len(t, r.Types(), 1)
}

func TestRegistry_Register_MultipleTypes(t *testing.T) {
	r := NewRegistry()
	r.Register("custom", dummyFactory("custom"))
	r.Register("ipfs", dummyFactory("ipfs"))
	r.Register("indexer", dummyFactory("indexer"))

	assert.True(t, r.HasType("custom"))
	assert.True(t, r.HasType("ipfs"))
	assert.True(t, r.HasType("indexer"))
	assert.Len(t, r.Types(), 3)
}

func TestRegistry_Register_OverwritesSameType(t *testing.T) {
	r := NewRegistry()

	r.Register("custom", dummyFactory("custom-v1"))
	r.Register("custom", dummyFactory("custom-v2"))

	assert.True(t, r.HasType("custom"))
	assert.Len(t, r.Types(), 1)

	svc, err := r.Create("test", config.ServiceConfig{Type: "custom"}, nil, "proj")
	require.NoError(t, err)
	assert.Equal(t, "custom-v2", svc.Type())
}

func TestRegistry_Create_Success(t *testing.T) {
	r := NewRegistry()
	r.Register("custom", dummyFactory("custom"))

	svc, err := r.Create("myservice", config.ServiceConfig{Type: "custom"}, nil, "myproject")
	require.NoError(t, err)
	require.NotNil(t, svc)
	assert.Equal(t, "myservice", svc.Name())
	assert.Equal(t, "custom", svc.Type())
}

func TestRegistry_Create_UnknownType(t *testing.T) {
	r := NewRegistry()
	r.Register("custom", dummyFactory("custom"))

	_, err := r.Create("svc", config.ServiceConfig{Type: "unknown"}, nil, "proj")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "unknown service type")
	assert.Contains(t, err.Error(), "unknown")
	assert.Contains(t, err.Error(), "svc")
}

func TestRegistry_Create_FactoryError(t *testing.T) {
	r := NewRegistry()
	r.Register("broken", failingFactory())

	_, err := r.Create("svc", config.ServiceConfig{Type: "broken"}, nil, "proj")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "factory error")
}

func TestRegistry_Create_EmptyRegistry(t *testing.T) {
	r := NewRegistry()

	_, err := r.Create("svc", config.ServiceConfig{Type: "anything"}, nil, "proj")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "unknown service type")
}

func TestRegistry_Create_PassesArguments(t *testing.T) {
	r := NewRegistry()

	var capturedName, capturedProject string
	var capturedCfg config.ServiceConfig
	var capturedRuntime container.Runtime

	r.Register("custom", func(name string, cfg config.ServiceConfig, rt container.Runtime, project string) (Service, error) {
		capturedName = name
		capturedCfg = cfg
		capturedRuntime = rt
		capturedProject = project
		return &mockService{name: name, svcType: "custom"}, nil
	})

	rt := newMockRuntime()
	cfg := config.ServiceConfig{
		Type:  "custom",
		Image: "test:latest",
	}

	_, err := r.Create("my-svc", cfg, rt, "my-project")
	require.NoError(t, err)

	assert.Equal(t, "my-svc", capturedName)
	assert.Equal(t, "custom", capturedCfg.Type)
	assert.Equal(t, "test:latest", capturedCfg.Image)
	assert.Equal(t, rt, capturedRuntime)
	assert.Equal(t, "my-project", capturedProject)
}

func TestRegistry_HasType_True(t *testing.T) {
	r := NewRegistry()
	r.Register("ipfs", dummyFactory("ipfs"))
	assert.True(t, r.HasType("ipfs"))
}

func TestRegistry_HasType_False(t *testing.T) {
	r := NewRegistry()
	assert.False(t, r.HasType("nonexistent"))
}

func TestRegistry_HasType_AfterRegister(t *testing.T) {
	r := NewRegistry()
	assert.False(t, r.HasType("custom"))

	r.Register("custom", dummyFactory("custom"))
	assert.True(t, r.HasType("custom"))
}

func TestRegistry_Types_Empty(t *testing.T) {
	r := NewRegistry()
	assert.Empty(t, r.Types())
}

func TestRegistry_Types_ReturnsList(t *testing.T) {
	r := NewRegistry()
	r.Register("custom", dummyFactory("custom"))
	r.Register("ipfs", dummyFactory("ipfs"))
	r.Register("indexer", dummyFactory("indexer"))

	types := r.Types()
	sort.Strings(types)
	assert.Equal(t, []string{"custom", "indexer", "ipfs"}, types)
}

func TestRegistry_Types_NoDuplicates(t *testing.T) {
	r := NewRegistry()
	r.Register("custom", dummyFactory("custom"))
	r.Register("custom", dummyFactory("custom")) // re-register

	assert.Len(t, r.Types(), 1)
}
