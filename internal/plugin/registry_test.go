package plugin

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewRegistryClient_CustomURL(t *testing.T) {
	c := NewRegistryClient("https://custom.registry.dev/api/v1")
	assert.Equal(t, "https://custom.registry.dev/api/v1", c.baseURL)
	assert.NotNil(t, c.httpClient)
}

func TestNewRegistryClient_EmptyFallsToDefault(t *testing.T) {
	c := NewRegistryClient("")
	assert.Equal(t, DefaultRegistryURL, c.baseURL)
}

func TestDefaultRegistryClient(t *testing.T) {
	c := DefaultRegistryClient()
	assert.Equal(t, DefaultRegistryURL, c.baseURL)
}

func TestRegistryClient_Search_Success(t *testing.T) {
	results := []Manifest{
		{Name: "plugin-a", Version: "1.0.0"},
		{Name: "plugin-b", Version: "2.0.0"},
	}

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "/plugins/search", r.URL.Path)
		assert.Equal(t, "test", r.URL.Query().Get("q"))
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(results)
	}))
	defer srv.Close()

	c := NewRegistryClient(srv.URL)
	got, err := c.Search(context.Background(), "test")
	require.NoError(t, err)
	assert.Len(t, got, 2)
	assert.Equal(t, "plugin-a", got[0].Name)
	assert.Equal(t, "plugin-b", got[1].Name)
}

func TestRegistryClient_Search_ServerError(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
		w.Write([]byte("internal error"))
	}))
	defer srv.Close()

	c := NewRegistryClient(srv.URL)
	_, err := c.Search(context.Background(), "test")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "500")
}

func TestRegistryClient_Search_InvalidJSON(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("not json"))
	}))
	defer srv.Close()

	c := NewRegistryClient(srv.URL)
	_, err := c.Search(context.Background(), "test")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "invalid response")
}

func TestRegistryClient_Search_ConnectionRefused(t *testing.T) {
	c := NewRegistryClient("http://127.0.0.1:1")
	_, err := c.Search(context.Background(), "test")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "registry request failed")
}

func TestRegistryClient_Get_Success(t *testing.T) {
	manifest := Manifest{Name: "my-plugin", Version: "1.0.0", Description: "Test plugin"}

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "/plugins/my-plugin", r.URL.Path)
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(manifest)
	}))
	defer srv.Close()

	c := NewRegistryClient(srv.URL)
	got, err := c.Get(context.Background(), "my-plugin")
	require.NoError(t, err)
	assert.Equal(t, "my-plugin", got.Name)
	assert.Equal(t, "1.0.0", got.Version)
}

func TestRegistryClient_Get_NotFound(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNotFound)
	}))
	defer srv.Close()

	c := NewRegistryClient(srv.URL)
	got, err := c.Get(context.Background(), "nonexistent")
	assert.Error(t, err)
	assert.Nil(t, got)
	assert.Contains(t, err.Error(), "not found in registry")
}

func TestRegistryClient_Get_ServerError(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusServiceUnavailable)
	}))
	defer srv.Close()

	c := NewRegistryClient(srv.URL)
	_, err := c.Get(context.Background(), "plugin")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "503")
}

func TestRegistryClient_Get_InvalidJSON(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("{bad json"))
	}))
	defer srv.Close()

	c := NewRegistryClient(srv.URL)
	_, err := c.Get(context.Background(), "plugin")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "invalid response")
}

func TestRegistryClient_Download_Success(t *testing.T) {
	archiveData := []byte("fake-archive-bytes")

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "/plugins/my-plugin/download", r.URL.Path)
		assert.Equal(t, "1.0.0", r.URL.Query().Get("version"))
		w.WriteHeader(http.StatusOK)
		w.Write(archiveData)
	}))
	defer srv.Close()

	c := NewRegistryClient(srv.URL)
	data, err := c.Download(context.Background(), "my-plugin", "1.0.0")
	require.NoError(t, err)
	assert.Equal(t, archiveData, data)
}

func TestRegistryClient_Download_NotFound(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNotFound)
	}))
	defer srv.Close()

	c := NewRegistryClient(srv.URL)
	_, err := c.Download(context.Background(), "ghost", "0.0.1")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "not found in registry")
}

func TestRegistryClient_Download_PaymentRequired(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusPaymentRequired)
	}))
	defer srv.Close()

	c := NewRegistryClient(srv.URL)
	_, err := c.Download(context.Background(), "premium", "1.0.0")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "requires a license")
}

func TestRegistryClient_Download_ServerErrorWithBody(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
		w.Write([]byte("something went wrong"))
	}))
	defer srv.Close()

	c := NewRegistryClient(srv.URL)
	_, err := c.Download(context.Background(), "broken", "1.0.0")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "500")
	assert.Contains(t, err.Error(), "something went wrong")
}

func TestRegistryClient_Download_ConnectionRefused(t *testing.T) {
	c := NewRegistryClient("http://127.0.0.1:1")
	_, err := c.Download(context.Background(), "plugin", "1.0.0")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "registry request failed")
}

func TestRegistryClient_Publish_MissingName(t *testing.T) {
	c := DefaultRegistryClient()
	err := c.Publish(context.Background(), Manifest{Version: "1.0.0"}, "/path", "token")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "name is required")
}

func TestRegistryClient_Publish_MissingVersion(t *testing.T) {
	c := DefaultRegistryClient()
	err := c.Publish(context.Background(), Manifest{Name: "my-plugin"}, "/path", "token")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "version is required")
}

func TestRegistryClient_Publish_NotYetAvailable(t *testing.T) {
	c := DefaultRegistryClient()
	err := c.Publish(context.Background(), Manifest{Name: "x", Version: "1.0.0"}, "/path", "token")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "coming soon")
}

func TestRegistryClient_HandleErrorStatus_WithBody(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusBadRequest)
		w.Write([]byte("bad request body"))
	}))
	defer srv.Close()

	c := NewRegistryClient(srv.URL)
	_, err := c.Search(context.Background(), "test")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "400")
	assert.Contains(t, err.Error(), "bad request body")
}

func TestRegistryClient_HandleErrorStatus_EmptyBody(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Length", "0")
		w.WriteHeader(http.StatusBadGateway)
	}))
	defer srv.Close()

	c := NewRegistryClient(srv.URL)
	_, err := c.Search(context.Background(), "test")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "502")
}

func TestRegistryClient_Search_ContextCancelled(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))
	defer srv.Close()

	c := NewRegistryClient(srv.URL)
	ctx, cancel := context.WithCancel(context.Background())
	cancel() // cancel immediately

	_, err := c.Search(ctx, "test")
	assert.Error(t, err)
}
