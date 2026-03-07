package types

import (
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestServiceStatus_JSONRoundTrip(t *testing.T) {
	original := ServiceStatus{
		Name:   "blockscout",
		Type:   "explorer",
		Status: "running",
		Ports: map[string]int{
			"http": 4000,
			"ws":   4001,
		},
		URLs: map[string]string{
			"web": "http://localhost:4000",
			"api": "http://localhost:4000/api",
		},
		Healthy: true,
	}

	data, err := json.Marshal(original)
	require.NoError(t, err)

	var decoded ServiceStatus
	err = json.Unmarshal(data, &decoded)
	require.NoError(t, err)

	assert.Equal(t, original, decoded)
}

func TestServiceStatus_JSONFieldNames(t *testing.T) {
	ss := ServiceStatus{
		Name:    "test-service",
		Type:    "indexer",
		Status:  "stopped",
		Ports:   map[string]int{"http": 8080},
		URLs:    map[string]string{"main": "http://localhost"},
		Healthy: false,
	}

	data, err := json.Marshal(ss)
	require.NoError(t, err)

	var raw map[string]interface{}
	err = json.Unmarshal(data, &raw)
	require.NoError(t, err)

	assert.Contains(t, raw, "name")
	assert.Contains(t, raw, "type")
	assert.Contains(t, raw, "status")
	assert.Contains(t, raw, "ports")
	assert.Contains(t, raw, "urls")
	assert.Contains(t, raw, "healthy")
}

func TestServiceStatus_PortsOmitEmpty(t *testing.T) {
	ss := ServiceStatus{
		Name:   "minimal",
		Type:   "service",
		Status: "running",
	}

	data, err := json.Marshal(ss)
	require.NoError(t, err)

	var raw map[string]interface{}
	err = json.Unmarshal(data, &raw)
	require.NoError(t, err)

	assert.NotContains(t, raw, "ports")
	assert.NotContains(t, raw, "urls")
}

func TestServiceStatus_ZeroValue(t *testing.T) {
	var ss ServiceStatus
	assert.Empty(t, ss.Name)
	assert.Empty(t, ss.Type)
	assert.Empty(t, ss.Status)
	assert.Nil(t, ss.Ports)
	assert.Nil(t, ss.URLs)
	assert.False(t, ss.Healthy)
}

func TestServiceStatus_JSONUnmarshal(t *testing.T) {
	jsonStr := `{
		"name": "ipfs",
		"type": "storage",
		"status": "running",
		"ports": {"http": 5001, "gateway": 8080},
		"urls": {"api": "http://localhost:5001"},
		"healthy": true
	}`

	var ss ServiceStatus
	err := json.Unmarshal([]byte(jsonStr), &ss)
	require.NoError(t, err)

	assert.Equal(t, "ipfs", ss.Name)
	assert.Equal(t, "storage", ss.Type)
	assert.Equal(t, "running", ss.Status)
	assert.Equal(t, 5001, ss.Ports["http"])
	assert.Equal(t, 8080, ss.Ports["gateway"])
	assert.Equal(t, "http://localhost:5001", ss.URLs["api"])
	assert.True(t, ss.Healthy)
}

func TestServiceStatus_MultiplePorts(t *testing.T) {
	ss := ServiceStatus{
		Ports: map[string]int{
			"http":    80,
			"https":   443,
			"grpc":    50051,
			"metrics": 9090,
		},
	}

	data, err := json.Marshal(ss)
	require.NoError(t, err)

	var decoded ServiceStatus
	err = json.Unmarshal(data, &decoded)
	require.NoError(t, err)

	assert.Len(t, decoded.Ports, 4)
	assert.Equal(t, 80, decoded.Ports["http"])
	assert.Equal(t, 443, decoded.Ports["https"])
	assert.Equal(t, 50051, decoded.Ports["grpc"])
	assert.Equal(t, 9090, decoded.Ports["metrics"])
}

func TestServiceStatus_HealthyStates(t *testing.T) {
	tests := []struct {
		name    string
		healthy bool
	}{
		{"healthy", true},
		{"unhealthy", false},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			ss := ServiceStatus{Healthy: tc.healthy}
			data, err := json.Marshal(ss)
			require.NoError(t, err)

			var decoded ServiceStatus
			err = json.Unmarshal(data, &decoded)
			require.NoError(t, err)
			assert.Equal(t, tc.healthy, decoded.Healthy)
		})
	}
}
