package template

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewRegistryClient(t *testing.T) {
	c := NewRegistryClient("https://example.com/api")
	require.NotNil(t, c)
	assert.Equal(t, "https://example.com/api", c.baseURL)
	assert.NotNil(t, c.httpClient)
}

func TestNewRegistryClientEmptyURL(t *testing.T) {
	c := NewRegistryClient("")
	require.NotNil(t, c)
	assert.Equal(t, DefaultRegistryURL, c.baseURL)
}

func TestNewRegistryClientDefaultURL(t *testing.T) {
	assert.Equal(t, "https://hub.dokrypt.dev/api/v1", DefaultRegistryURL)
}

func TestDefaultRegistryClient(t *testing.T) {
	c := DefaultRegistryClient()
	require.NotNil(t, c)
	assert.Equal(t, DefaultRegistryURL, c.baseURL)
	assert.NotNil(t, c.httpClient)
}

func TestRegistryClientSearchSuccess(t *testing.T) {
	templates := []Template{
		{Name: "test-1", Version: "1.0.0", Description: "Test one"},
		{Name: "test-2", Version: "2.0.0", Description: "Test two"},
	}

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "/templates/search", r.URL.Path)
		assert.Equal(t, "evm", r.URL.Query().Get("q"))
		assert.Equal(t, http.MethodGet, r.Method)

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(templates)
	}))
	defer server.Close()

	c := NewRegistryClient(server.URL)
	results, err := c.Search(context.Background(), "evm")
	require.NoError(t, err)
	assert.Len(t, results, 2)
	assert.Equal(t, "test-1", results[0].Name)
	assert.Equal(t, "test-2", results[1].Name)
}

func TestRegistryClientSearchEmptyQuery(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "", r.URL.Query().Get("q"))
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode([]Template{})
	}))
	defer server.Close()

	c := NewRegistryClient(server.URL)
	results, err := c.Search(context.Background(), "")
	require.NoError(t, err)
	assert.Empty(t, results)
}

func TestRegistryClientSearchQueryEscaping(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "hello world", r.URL.Query().Get("q"))
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode([]Template{})
	}))
	defer server.Close()

	c := NewRegistryClient(server.URL)
	_, err := c.Search(context.Background(), "hello world")
	require.NoError(t, err)
}

func TestRegistryClientSearchServerError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
		w.Write([]byte("internal server error"))
	}))
	defer server.Close()

	c := NewRegistryClient(server.URL)
	results, err := c.Search(context.Background(), "evm")
	assert.Error(t, err)
	assert.Nil(t, results)
	assert.Contains(t, err.Error(), "500")
	assert.Contains(t, err.Error(), "internal server error")
}

func TestRegistryClientSearchServerErrorNoBody(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusServiceUnavailable)
	}))
	defer server.Close()

	c := NewRegistryClient(server.URL)
	results, err := c.Search(context.Background(), "evm")
	assert.Error(t, err)
	assert.Nil(t, results)
	assert.Contains(t, err.Error(), "503")
}

func TestRegistryClientSearchInvalidJSON(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte("not json"))
	}))
	defer server.Close()

	c := NewRegistryClient(server.URL)
	results, err := c.Search(context.Background(), "test")
	assert.Error(t, err)
	assert.Nil(t, results)
	assert.Contains(t, err.Error(), "invalid response")
}

func TestRegistryClientSearchCanceledContext(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode([]Template{})
	}))
	defer server.Close()

	ctx, cancel := context.WithCancel(context.Background())
	cancel() // Cancel immediately

	c := NewRegistryClient(server.URL)
	results, err := c.Search(ctx, "evm")
	assert.Error(t, err)
	assert.Nil(t, results)
}

func TestRegistryClientPullSuccess(t *testing.T) {
	expected := []byte("archive-data-bytes")

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "/templates/my-template/download", r.URL.Path)
		assert.Equal(t, http.MethodGet, r.Method)
		w.Write(expected)
	}))
	defer server.Close()

	c := NewRegistryClient(server.URL)
	data, err := c.Pull(context.Background(), "my-template")
	require.NoError(t, err)
	assert.Equal(t, expected, data)
}

func TestRegistryClientPullNotFound(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNotFound)
	}))
	defer server.Close()

	c := NewRegistryClient(server.URL)
	data, err := c.Pull(context.Background(), "nonexistent")
	assert.Error(t, err)
	assert.Nil(t, data)
	assert.Contains(t, err.Error(), "not found in registry")
}

func TestRegistryClientPullPaymentRequired(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusPaymentRequired)
	}))
	defer server.Close()

	c := NewRegistryClient(server.URL)
	data, err := c.Pull(context.Background(), "premium-template")
	assert.Error(t, err)
	assert.Nil(t, data)
	assert.Contains(t, err.Error(), "requires a license")
	assert.Contains(t, err.Error(), "dokrypt.dev/pricing")
}

func TestRegistryClientPullServerError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
		w.Write([]byte("server broke"))
	}))
	defer server.Close()

	c := NewRegistryClient(server.URL)
	data, err := c.Pull(context.Background(), "broken")
	assert.Error(t, err)
	assert.Nil(t, data)
	assert.Contains(t, err.Error(), "500")
}

func TestRegistryClientPullCanceledContext(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte("data"))
	}))
	defer server.Close()

	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	c := NewRegistryClient(server.URL)
	data, err := c.Pull(ctx, "test")
	assert.Error(t, err)
	assert.Nil(t, data)
}

func TestRegistryClientPullPathEscaping(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.RawPath != "" {
			assert.Contains(t, r.URL.RawPath, "%2F")
		}
		w.Write([]byte("ok"))
	}))
	defer server.Close()

	c := NewRegistryClient(server.URL)
	_, err := c.Pull(context.Background(), "my/template")
	require.NoError(t, err)
}

func TestRegistryClientPushMissingName(t *testing.T) {
	c := NewRegistryClient("https://example.com")
	err := c.Push(context.Background(), Template{Version: "1.0.0"}, "/path/to/archive", "token123")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "name is required")
}

func TestRegistryClientPushMissingVersion(t *testing.T) {
	c := NewRegistryClient("https://example.com")
	err := c.Push(context.Background(), Template{Name: "test"}, "/path/to/archive", "token123")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "version is required")
}

func TestRegistryClientPushComingSoon(t *testing.T) {
	c := NewRegistryClient("https://example.com")
	meta := Template{Name: "test", Version: "1.0.0"}
	err := c.Push(context.Background(), meta, "/path/to/archive", "token123")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "coming soon")
}

func TestRegistryClientPushValidationOrder(t *testing.T) {
	c := NewRegistryClient("https://example.com")

	err := c.Push(context.Background(), Template{}, "/path", "token")
	assert.Contains(t, err.Error(), "name is required")

	err = c.Push(context.Background(), Template{Name: "test"}, "/path", "token")
	assert.Contains(t, err.Error(), "version is required")
}

func TestHandleErrorStatusWithBody(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusBadRequest)
		w.Write([]byte("bad request details"))
	}))
	defer server.Close()

	c := NewRegistryClient(server.URL)
	_, err := c.Search(context.Background(), "test")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "400")
	assert.Contains(t, err.Error(), "bad request details")
}

func TestHandleErrorStatusWithoutBody(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Length", "0")
		w.WriteHeader(http.StatusBadGateway)
	}))
	defer server.Close()

	c := NewRegistryClient(server.URL)
	_, err := c.Search(context.Background(), "test")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "502")
}

func TestRegistryClientSearchConnectionRefused(t *testing.T) {
	c := NewRegistryClient("http://127.0.0.1:1")
	c.httpClient.Timeout = 1 * time.Second

	results, err := c.Search(context.Background(), "test")
	assert.Error(t, err)
	assert.Nil(t, results)
	assert.Contains(t, err.Error(), "registry request failed")
}

func TestRegistryClientPullConnectionRefused(t *testing.T) {
	c := NewRegistryClient("http://127.0.0.1:1")
	c.httpClient.Timeout = 1 * time.Second

	data, err := c.Pull(context.Background(), "test")
	assert.Error(t, err)
	assert.Nil(t, data)
	assert.Contains(t, err.Error(), "registry request failed")
}
