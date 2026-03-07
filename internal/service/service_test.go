package service

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestLogOptions_ZeroValue(t *testing.T) {
	var opts LogOptions
	assert.False(t, opts.Follow)
	assert.Empty(t, opts.Tail)
	assert.Empty(t, opts.Since)
	assert.False(t, opts.Timestamps)
}

func TestLogOptions_AllFieldsSet(t *testing.T) {
	opts := LogOptions{
		Follow:     true,
		Tail:       "100",
		Since:      "2h",
		Timestamps: true,
	}
	assert.True(t, opts.Follow)
	assert.Equal(t, "100", opts.Tail)
	assert.Equal(t, "2h", opts.Since)
	assert.True(t, opts.Timestamps)
}

func TestServiceStatus_ZeroValue(t *testing.T) {
	var s ServiceStatus
	assert.Empty(t, s.Name)
	assert.Empty(t, s.Type)
	assert.Empty(t, s.Status)
	assert.Nil(t, s.Ports)
	assert.Nil(t, s.URLs)
	assert.False(t, s.Healthy)
}

func TestServiceStatus_AllFieldsSet(t *testing.T) {
	s := ServiceStatus{
		Name:    "api",
		Type:    "custom",
		Status:  "running",
		Ports:   map[string]int{"http": 8080},
		URLs:    map[string]string{"api": "http://localhost:8080"},
		Healthy: true,
	}
	assert.Equal(t, "api", s.Name)
	assert.Equal(t, "custom", s.Type)
	assert.Equal(t, "running", s.Status)
	assert.Equal(t, 8080, s.Ports["http"])
	assert.Equal(t, "http://localhost:8080", s.URLs["api"])
	assert.True(t, s.Healthy)
}

func TestServiceInterface_MockImplementation(t *testing.T) {
	var _ Service = (*mockService)(nil)
}

func TestServiceStatus_RunningStatus(t *testing.T) {
	s := ServiceStatus{
		Name:    "db",
		Type:    "postgres",
		Status:  "running",
		Healthy: true,
		Ports:   map[string]int{"pg": 5432},
		URLs:    map[string]string{"pg": "postgresql://localhost:5432"},
	}
	assert.Equal(t, "running", s.Status)
	assert.True(t, s.Healthy)
}

func TestServiceStatus_StoppedStatus(t *testing.T) {
	s := ServiceStatus{
		Name:    "worker",
		Type:    "custom",
		Status:  "stopped",
		Healthy: false,
	}
	assert.Equal(t, "stopped", s.Status)
	assert.False(t, s.Healthy)
}

func TestServiceStatus_UnhealthyStatus(t *testing.T) {
	s := ServiceStatus{
		Name:    "api",
		Type:    "custom",
		Status:  "unhealthy",
		Healthy: false,
	}
	assert.Equal(t, "unhealthy", s.Status)
	assert.False(t, s.Healthy)
}

func TestServiceStatus_StartingStatus(t *testing.T) {
	s := ServiceStatus{
		Name:   "indexer",
		Type:   "indexer",
		Status: "starting",
	}
	assert.Equal(t, "starting", s.Status)
}
