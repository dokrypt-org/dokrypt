package marketplace

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewClient_DefaultURL(t *testing.T) {
	client := NewClient("")
	require.NotNil(t, client)
	assert.Equal(t, DefaultHubURL, client.baseURL)
}

func TestNewClient_CustomURL(t *testing.T) {
	client := NewClient("https://custom.hub.dev/api/v1")
	require.NotNil(t, client)
	assert.Equal(t, "https://custom.hub.dev/api/v1", client.baseURL)
}

func TestNewClient_HasHTTPClient(t *testing.T) {
	client := NewClient("")
	require.NotNil(t, client.httpClient)
	assert.NotZero(t, client.httpClient.Timeout)
}

func TestClient_Search_Success(t *testing.T) {
	expected := SearchResult{
		Query: "defi",
		Total: 2,
		Packages: []PackageMeta{
			{Name: "defi-swap", Version: "1.0.0"},
			{Name: "defi-lending", Version: "2.0.0"},
		},
	}

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "/templates/search", r.URL.Path)
		assert.Equal(t, "defi", r.URL.Query().Get("q"))
		assert.Equal(t, http.MethodGet, r.Method)

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(expected)
	}))
	defer server.Close()

	client := NewClient(server.URL)
	result, err := client.Search("defi")
	require.NoError(t, err)
	require.NotNil(t, result)

	assert.Equal(t, "defi", result.Query)
	assert.Equal(t, 2, result.Total)
	assert.Len(t, result.Packages, 2)
	assert.Equal(t, "defi-swap", result.Packages[0].Name)
	assert.Equal(t, "defi-lending", result.Packages[1].Name)
}

func TestClient_Search_QueryEncoding(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "hello world", r.URL.Query().Get("q"))
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(SearchResult{Query: "hello world"})
	}))
	defer server.Close()

	client := NewClient(server.URL)
	result, err := client.Search("hello world")
	require.NoError(t, err)
	assert.Equal(t, "hello world", result.Query)
}

func TestClient_Search_ServerError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	}))
	defer server.Close()

	client := NewClient(server.URL)
	result, err := client.Search("test")
	require.Error(t, err)
	assert.Nil(t, result)
	assert.Contains(t, err.Error(), "status 500")
}

func TestClient_Search_InvalidJSON(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte("{invalid json"))
	}))
	defer server.Close()

	client := NewClient(server.URL)
	result, err := client.Search("test")
	require.Error(t, err)
	assert.Nil(t, result)
	assert.Contains(t, err.Error(), "invalid response")
}

func TestClient_Search_ConnectionError(t *testing.T) {
	client := NewClient("http://127.0.0.1:1")
	result, err := client.Search("test")
	require.Error(t, err)
	assert.Nil(t, result)
	assert.Contains(t, err.Error(), "marketplace request failed")
}

func TestClient_Browse_Success(t *testing.T) {
	expected := SearchResult{
		Query: "",
		Total: 1,
		Packages: []PackageMeta{
			{Name: "nft-gallery", Version: "1.0.0", Category: "nft"},
		},
	}

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "/templates", r.URL.Path)
		assert.Equal(t, "nft", r.URL.Query().Get("category"))
		assert.Equal(t, http.MethodGet, r.Method)

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(expected)
	}))
	defer server.Close()

	client := NewClient(server.URL)
	result, err := client.Browse("nft")
	require.NoError(t, err)
	require.NotNil(t, result)

	assert.Equal(t, 1, result.Total)
	assert.Len(t, result.Packages, 1)
	assert.Equal(t, "nft-gallery", result.Packages[0].Name)
}

func TestClient_Browse_EmptyCategory(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "", r.URL.Query().Get("category"))
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(SearchResult{Total: 5})
	}))
	defer server.Close()

	client := NewClient(server.URL)
	result, err := client.Browse("")
	require.NoError(t, err)
	assert.Equal(t, 5, result.Total)
}

func TestClient_Browse_ServerError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusServiceUnavailable)
	}))
	defer server.Close()

	client := NewClient(server.URL)
	result, err := client.Browse("defi")
	require.Error(t, err)
	assert.Nil(t, result)
	assert.Contains(t, err.Error(), "status 503")
}

func TestClient_Browse_InvalidJSON(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte("not json"))
	}))
	defer server.Close()

	client := NewClient(server.URL)
	result, err := client.Browse("defi")
	require.Error(t, err)
	assert.Nil(t, result)
	assert.Contains(t, err.Error(), "invalid response")
}

func TestClient_GetInfo_Success(t *testing.T) {
	expected := PackageMeta{
		Name:        "my-template",
		Version:     "3.0.0",
		Description: "A great template",
		Author:      "author",
		Category:    "defi",
		Downloads:   100,
		Stars:       42,
	}

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "/templates/my-template", r.URL.Path)
		assert.Equal(t, http.MethodGet, r.Method)

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(expected)
	}))
	defer server.Close()

	client := NewClient(server.URL)
	meta, err := client.GetInfo("my-template")
	require.NoError(t, err)
	require.NotNil(t, meta)

	assert.Equal(t, "my-template", meta.Name)
	assert.Equal(t, "3.0.0", meta.Version)
	assert.Equal(t, "A great template", meta.Description)
	assert.Equal(t, "author", meta.Author)
	assert.Equal(t, 100, meta.Downloads)
	assert.Equal(t, 42, meta.Stars)
}

func TestClient_GetInfo_NotFound(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNotFound)
	}))
	defer server.Close()

	client := NewClient(server.URL)
	meta, err := client.GetInfo("nonexistent")
	require.Error(t, err)
	assert.Nil(t, meta)
	assert.Contains(t, err.Error(), "not found in marketplace")
}

func TestClient_GetInfo_ServerError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusBadGateway)
	}))
	defer server.Close()

	client := NewClient(server.URL)
	meta, err := client.GetInfo("template")
	require.Error(t, err)
	assert.Nil(t, meta)
	assert.Contains(t, err.Error(), "status 502")
}

func TestClient_GetInfo_InvalidJSON(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte("broken"))
	}))
	defer server.Close()

	client := NewClient(server.URL)
	meta, err := client.GetInfo("template")
	require.Error(t, err)
	assert.Nil(t, meta)
	assert.Contains(t, err.Error(), "invalid response")
}

func TestClient_GetInfo_NameWithSpecialChars(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.RawPath != "" {
			assert.Contains(t, r.URL.RawPath, "org%2Fmy-template")
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(PackageMeta{Name: "org/my-template"})
	}))
	defer server.Close()

	client := NewClient(server.URL)
	meta, err := client.GetInfo("org/my-template")
	require.NoError(t, err)
	assert.Equal(t, "org/my-template", meta.Name)
}

func TestClient_Download_Success(t *testing.T) {
	tarballData := []byte("fake-tarball-data-bytes")

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "/templates/my-pkg/download", r.URL.Path)
		assert.Equal(t, http.MethodGet, r.Method)

		w.Header().Set("Content-Type", "application/octet-stream")
		w.Write(tarballData)
	}))
	defer server.Close()

	client := NewClient(server.URL)
	data, err := client.Download("my-pkg")
	require.NoError(t, err)
	assert.Equal(t, tarballData, data)
}

func TestClient_Download_NotFound(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNotFound)
	}))
	defer server.Close()

	client := NewClient(server.URL)
	data, err := client.Download("missing")
	require.Error(t, err)
	assert.Nil(t, data)
	assert.Contains(t, err.Error(), "not found in marketplace")
}

func TestClient_Download_PaymentRequired(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusPaymentRequired)
	}))
	defer server.Close()

	client := NewClient(server.URL)
	data, err := client.Download("premium-pkg")
	require.Error(t, err)
	assert.Nil(t, data)
	assert.Contains(t, err.Error(), "requires a license")
	assert.Contains(t, err.Error(), "dokrypt.com/pricing")
}

func TestClient_Download_ServerError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	}))
	defer server.Close()

	client := NewClient(server.URL)
	data, err := client.Download("template")
	require.Error(t, err)
	assert.Nil(t, data)
	assert.Contains(t, err.Error(), "status 500")
}

func TestClient_Download_ConnectionError(t *testing.T) {
	client := NewClient("http://127.0.0.1:1")
	data, err := client.Download("template")
	require.Error(t, err)
	assert.Nil(t, data)
	assert.Contains(t, err.Error(), "marketplace request failed")
}

func TestClient_Download_LargePayload(t *testing.T) {
	largeData := make([]byte, 1024*1024) // 1MB
	for i := range largeData {
		largeData[i] = byte(i % 256)
	}

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/octet-stream")
		w.Write(largeData)
	}))
	defer server.Close()

	client := NewClient(server.URL)
	data, err := client.Download("large-pkg")
	require.NoError(t, err)
	assert.Len(t, data, len(largeData))
	assert.Equal(t, largeData, data)
}

func TestClient_Publish_MissingName(t *testing.T) {
	client := NewClient("")
	err := client.Publish(PackageMeta{Version: "1.0.0"}, "/path/to/archive.tar.gz", "token")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "name is required")
}

func TestClient_Publish_MissingVersion(t *testing.T) {
	client := NewClient("")
	err := client.Publish(PackageMeta{Name: "my-pkg"}, "/path/to/archive.tar.gz", "token")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "version is required")
}

func TestClient_Publish_ComingSoon(t *testing.T) {
	client := NewClient("")
	meta := PackageMeta{
		Name:    "my-template",
		Version: "1.0.0",
	}
	err := client.Publish(meta, "/path/to/archive.tar.gz", "my-token")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "coming soon")
	assert.Contains(t, err.Error(), "hub.dokrypt.com")
}

func TestClient_Publish_ValidatesBeforeHubCheck(t *testing.T) {
	client := NewClient("")

	err := client.Publish(PackageMeta{}, "/archive.tar.gz", "token")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "name is required")

	err = client.Publish(PackageMeta{Name: "pkg"}, "/archive.tar.gz", "token")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "version is required")
}

func TestDefaultHubURL_Value(t *testing.T) {
	assert.Equal(t, "https://hub.dokrypt.com/api/v1", DefaultHubURL)
}

func TestClient_Search_EmptyQuery(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "", r.URL.Query().Get("q"))
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(SearchResult{Total: 10})
	}))
	defer server.Close()

	client := NewClient(server.URL)
	result, err := client.Search("")
	require.NoError(t, err)
	assert.Equal(t, 10, result.Total)
}

func TestClient_Browse_CategoryEncoding(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "smart contracts", r.URL.Query().Get("category"))
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(SearchResult{})
	}))
	defer server.Close()

	client := NewClient(server.URL)
	_, err := client.Browse("smart contracts")
	require.NoError(t, err)
}

func TestClient_Download_NameWithSlash(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.RawPath != "" {
			assert.Contains(t, r.URL.RawPath, "org%2Fpkg")
		}
		w.Header().Set("Content-Type", "application/octet-stream")
		w.Write([]byte("data"))
	}))
	defer server.Close()

	client := NewClient(server.URL)
	data, err := client.Download("org/pkg")
	require.NoError(t, err)
	assert.Equal(t, []byte("data"), data)
}
