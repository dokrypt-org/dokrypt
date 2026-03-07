package container

import (
	"context"
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNetworkDetail_Struct(t *testing.T) {
	detail := NetworkDetail{
		NetworkInfo: NetworkInfo{
			ID:     "net-1",
			Name:   "mynet",
			Driver: "bridge",
			Subnet: "10.0.0.0/24",
			Labels: map[string]string{"env": "test"},
		},
		Containers: []string{"c1", "c2"},
		Subnet:     "10.0.0.0/24",
		Gateway:    "10.0.0.1",
	}
	assert.Equal(t, "net-1", detail.ID)
	assert.Equal(t, "mynet", detail.Name)
	assert.Equal(t, "bridge", detail.Driver)
	assert.Len(t, detail.Containers, 2)
	assert.Equal(t, "10.0.0.1", detail.Gateway)
}

func TestNetworkCreateOptions_Struct(t *testing.T) {
	opts := NetworkCreateOptions{
		NetworkOptions: NetworkOptions{
			Driver:   "bridge",
			Internal: true,
			Labels:   map[string]string{"a": "b"},
			Subnet:   "172.20.0.0/16",
			Gateway:  "172.20.0.1",
		},
		Aliases: []string{"alias1", "alias2"},
	}
	assert.Equal(t, "bridge", opts.Driver)
	assert.True(t, opts.Internal)
	assert.Equal(t, "b", opts.Labels["a"])
	assert.Equal(t, "172.20.0.0/16", opts.Subnet)
	assert.Equal(t, "172.20.0.1", opts.Gateway)
	assert.Equal(t, []string{"alias1", "alias2"}, opts.Aliases)
}

func TestNewNetworkManager(t *testing.T) {
	mock := newMockRuntime()
	mgr := NewNetworkManager(mock)
	require.NotNil(t, mgr)
	assert.Equal(t, mock, mgr.runtime)
}

func TestNetworkManager_Create_NewNetwork(t *testing.T) {
	mock := newMockRuntime()
	mock.listNetworksFn = func(ctx context.Context) ([]NetworkInfo, error) {
		return []NetworkInfo{}, nil
	}
	var capturedOpts NetworkOptions
	mock.createNetworkFn = func(ctx context.Context, name string, opts NetworkOptions) (string, error) {
		capturedOpts = opts
		return "new-net-id", nil
	}

	mgr := NewNetworkManager(mock)
	id, err := mgr.Create(context.Background(), "mynet", NetworkCreateOptions{
		NetworkOptions: NetworkOptions{
			Driver: "bridge",
		},
	})
	require.NoError(t, err)
	assert.Equal(t, "new-net-id", id)
	assert.Equal(t, 1, mock.calls["CreateNetwork"])
	assert.Equal(t, "true", capturedOpts.Labels["dokrypt.network"])
}

func TestNetworkManager_Create_AlreadyExists(t *testing.T) {
	mock := newMockRuntime()
	mock.listNetworksFn = func(ctx context.Context) ([]NetworkInfo, error) {
		return []NetworkInfo{
			{ID: "existing-id", Name: "mynet", Driver: "bridge"},
		}, nil
	}

	mgr := NewNetworkManager(mock)
	id, err := mgr.Create(context.Background(), "mynet", NetworkCreateOptions{})
	require.NoError(t, err)
	assert.Equal(t, "existing-id", id)
	assert.Equal(t, 0, mock.calls["CreateNetwork"])
}

func TestNetworkManager_Create_ListError(t *testing.T) {
	mock := newMockRuntime()
	mock.listNetworksFn = func(ctx context.Context) ([]NetworkInfo, error) {
		return nil, errMock
	}
	mock.createNetworkFn = func(ctx context.Context, name string, opts NetworkOptions) (string, error) {
		return "net-id", nil
	}

	mgr := NewNetworkManager(mock)
	id, err := mgr.Create(context.Background(), "mynet", NetworkCreateOptions{})
	require.NoError(t, err)
	assert.Equal(t, "net-id", id)
	assert.Equal(t, 1, mock.calls["CreateNetwork"])
}

func TestNetworkManager_Create_CreateError(t *testing.T) {
	mock := newMockRuntime()
	mock.listNetworksFn = func(ctx context.Context) ([]NetworkInfo, error) {
		return []NetworkInfo{}, nil
	}
	mock.createNetworkFn = func(ctx context.Context, name string, opts NetworkOptions) (string, error) {
		return "", fmt.Errorf("permission denied")
	}

	mgr := NewNetworkManager(mock)
	id, err := mgr.Create(context.Background(), "mynet", NetworkCreateOptions{})
	require.Error(t, err)
	assert.Empty(t, id)
	assert.Contains(t, err.Error(), "failed to create network mynet")
	assert.Contains(t, err.Error(), "permission denied")
}

func TestNetworkManager_Create_NilLabelsInitialized(t *testing.T) {
	mock := newMockRuntime()
	mock.listNetworksFn = func(ctx context.Context) ([]NetworkInfo, error) {
		return []NetworkInfo{}, nil
	}
	var capturedOpts NetworkOptions
	mock.createNetworkFn = func(ctx context.Context, name string, opts NetworkOptions) (string, error) {
		capturedOpts = opts
		return "net-id", nil
	}

	mgr := NewNetworkManager(mock)
	id, err := mgr.Create(context.Background(), "mynet", NetworkCreateOptions{})
	require.NoError(t, err)
	assert.NotEmpty(t, id)
	require.NotNil(t, capturedOpts.Labels)
	assert.Equal(t, "true", capturedOpts.Labels["dokrypt.network"])
}

func TestNetworkManager_Create_ExistingLabelsPreserved(t *testing.T) {
	mock := newMockRuntime()
	mock.listNetworksFn = func(ctx context.Context) ([]NetworkInfo, error) {
		return []NetworkInfo{}, nil
	}
	var capturedOpts NetworkOptions
	mock.createNetworkFn = func(ctx context.Context, name string, opts NetworkOptions) (string, error) {
		capturedOpts = opts
		return "net-id", nil
	}

	mgr := NewNetworkManager(mock)
	id, err := mgr.Create(context.Background(), "mynet", NetworkCreateOptions{
		NetworkOptions: NetworkOptions{
			Labels: map[string]string{"existing": "label"},
		},
	})
	require.NoError(t, err)
	assert.NotEmpty(t, id)
	assert.Equal(t, "label", capturedOpts.Labels["existing"])
	assert.Equal(t, "true", capturedOpts.Labels["dokrypt.network"])
}

func TestNetworkManager_Connect_Success(t *testing.T) {
	mock := newMockRuntime()
	mgr := NewNetworkManager(mock)
	err := mgr.Connect(context.Background(), "net-1", "container-1")
	require.NoError(t, err)
	assert.Equal(t, 1, mock.calls["ConnectNetwork"])
}

func TestNetworkManager_Connect_Error(t *testing.T) {
	mock := newMockRuntime()
	mock.connectNetworkFn = func(ctx context.Context, networkID, containerID string) error {
		return errMock
	}
	mgr := NewNetworkManager(mock)
	err := mgr.Connect(context.Background(), "net-1", "container-1")
	require.Error(t, err)
}

func TestNetworkManager_Disconnect_Success(t *testing.T) {
	mock := newMockRuntime()
	mgr := NewNetworkManager(mock)
	err := mgr.Disconnect(context.Background(), "net-1", "container-1")
	require.NoError(t, err)
	assert.Equal(t, 1, mock.calls["DisconnectNetwork"])
}

func TestNetworkManager_Disconnect_Error(t *testing.T) {
	mock := newMockRuntime()
	mock.disconnectNetworkFn = func(ctx context.Context, networkID, containerID string) error {
		return errMock
	}
	mgr := NewNetworkManager(mock)
	err := mgr.Disconnect(context.Background(), "net-1", "container-1")
	require.Error(t, err)
}

func TestNetworkManager_Remove_Success(t *testing.T) {
	mock := newMockRuntime()
	mgr := NewNetworkManager(mock)
	err := mgr.Remove(context.Background(), "net-1", false)
	require.NoError(t, err)
	assert.Equal(t, 1, mock.calls["RemoveNetwork"])
}

func TestNetworkManager_Remove_Error(t *testing.T) {
	mock := newMockRuntime()
	mock.removeNetworkFn = func(ctx context.Context, id string) error {
		return errMock
	}
	mgr := NewNetworkManager(mock)
	err := mgr.Remove(context.Background(), "net-1", true)
	require.Error(t, err)
}

func TestNetworkManager_List_FiltersDokryptNetworks(t *testing.T) {
	mock := newMockRuntime()
	mock.listNetworksFn = func(ctx context.Context) ([]NetworkInfo, error) {
		return []NetworkInfo{
			{ID: "1", Name: "bridge", Labels: nil},
			{ID: "2", Name: "dokrypt-net", Labels: map[string]string{"dokrypt.network": "true"}},
			{ID: "3", Name: "host", Labels: map[string]string{}},
			{ID: "4", Name: "dokrypt-other", Labels: map[string]string{"dokrypt.network": "true", "custom": "val"}},
		}, nil
	}

	mgr := NewNetworkManager(mock)
	networks, err := mgr.List(context.Background())
	require.NoError(t, err)
	assert.Len(t, networks, 2)
	assert.Equal(t, "dokrypt-net", networks[0].Name)
	assert.Equal(t, "dokrypt-other", networks[1].Name)
}

func TestNetworkManager_List_NoDokryptNetworks(t *testing.T) {
	mock := newMockRuntime()
	mock.listNetworksFn = func(ctx context.Context) ([]NetworkInfo, error) {
		return []NetworkInfo{
			{ID: "1", Name: "bridge", Labels: nil},
			{ID: "2", Name: "host", Labels: map[string]string{}},
		}, nil
	}

	mgr := NewNetworkManager(mock)
	networks, err := mgr.List(context.Background())
	require.NoError(t, err)
	assert.Empty(t, networks)
}

func TestNetworkManager_List_Error(t *testing.T) {
	mock := newMockRuntime()
	mock.listNetworksFn = func(ctx context.Context) ([]NetworkInfo, error) {
		return nil, errMock
	}

	mgr := NewNetworkManager(mock)
	networks, err := mgr.List(context.Background())
	require.Error(t, err)
	assert.Nil(t, networks)
}

func TestNetworkManager_List_EmptyFromRuntime(t *testing.T) {
	mock := newMockRuntime()
	mock.listNetworksFn = func(ctx context.Context) ([]NetworkInfo, error) {
		return []NetworkInfo{}, nil
	}

	mgr := NewNetworkManager(mock)
	networks, err := mgr.List(context.Background())
	require.NoError(t, err)
	assert.Empty(t, networks)
}

func TestNetworkManager_CreateInterconnect_Success(t *testing.T) {
	mock := newMockRuntime()
	mock.listNetworksFn = func(ctx context.Context) ([]NetworkInfo, error) {
		return []NetworkInfo{}, nil
	}
	var capturedName string
	var capturedOpts NetworkOptions
	mock.createNetworkFn = func(ctx context.Context, name string, opts NetworkOptions) (string, error) {
		capturedName = name
		capturedOpts = opts
		return "interconnect-id", nil
	}

	mgr := NewNetworkManager(mock)
	id, err := mgr.CreateInterconnect(context.Background(), "myproject")
	require.NoError(t, err)
	assert.Equal(t, "interconnect-id", id)
	assert.Equal(t, "dokrypt-myproject-interconnect", capturedName)
	assert.Equal(t, "bridge", capturedOpts.Driver)
	assert.Equal(t, "myproject", capturedOpts.Labels["dokrypt.project"])
	assert.Equal(t, "interconnect", capturedOpts.Labels["dokrypt.type"])
	assert.Equal(t, "true", capturedOpts.Labels["dokrypt.network"])
}

func TestNetworkManager_CreateInterconnect_AlreadyExists(t *testing.T) {
	mock := newMockRuntime()
	mock.listNetworksFn = func(ctx context.Context) ([]NetworkInfo, error) {
		return []NetworkInfo{
			{ID: "existing-interconnect", Name: "dokrypt-myproject-interconnect"},
		}, nil
	}

	mgr := NewNetworkManager(mock)
	id, err := mgr.CreateInterconnect(context.Background(), "myproject")
	require.NoError(t, err)
	assert.Equal(t, "existing-interconnect", id)
	assert.Equal(t, 0, mock.calls["CreateNetwork"])
}

func TestNetworkManager_CreateInterconnect_NameFormat(t *testing.T) {
	tests := []struct {
		project  string
		expected string
	}{
		{"myapp", "dokrypt-myapp-interconnect"},
		{"test-chain", "dokrypt-test-chain-interconnect"},
		{"prod", "dokrypt-prod-interconnect"},
	}

	for _, tc := range tests {
		t.Run(tc.project, func(t *testing.T) {
			mock := newMockRuntime()
			mock.listNetworksFn = func(ctx context.Context) ([]NetworkInfo, error) {
				return []NetworkInfo{}, nil
			}
			var capturedName string
			mock.createNetworkFn = func(ctx context.Context, name string, opts NetworkOptions) (string, error) {
				capturedName = name
				return "id", nil
			}

			mgr := NewNetworkManager(mock)
			_, err := mgr.CreateInterconnect(context.Background(), tc.project)
			require.NoError(t, err)
			assert.Equal(t, tc.expected, capturedName)
		})
	}
}
