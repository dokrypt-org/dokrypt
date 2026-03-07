package plugin

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"time"
)

const (
	DefaultRegistryURL = "https://hub.dokrypt.dev/api/v1"
)

type RegistryClient struct {
	baseURL    string
	httpClient *http.Client
}

func NewRegistryClient(baseURL string) *RegistryClient {
	if baseURL == "" {
		baseURL = DefaultRegistryURL
	}
	return &RegistryClient{
		baseURL: baseURL,
		httpClient: &http.Client{
			Timeout: 30 * time.Second,
		},
	}
}

func DefaultRegistryClient() *RegistryClient {
	return NewRegistryClient(DefaultRegistryURL)
}

func (c *RegistryClient) Search(ctx context.Context, query string) ([]Manifest, error) {
	u := fmt.Sprintf("%s/plugins/search?q=%s", c.baseURL, url.QueryEscape(query))

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, u, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("registry request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, c.handleErrorStatus("search", resp)
	}

	var results []Manifest
	if err := json.NewDecoder(resp.Body).Decode(&results); err != nil {
		return nil, fmt.Errorf("invalid response from registry: %w", err)
	}
	return results, nil
}

func (c *RegistryClient) Get(ctx context.Context, name string) (*Manifest, error) {
	u := fmt.Sprintf("%s/plugins/%s", c.baseURL, url.PathEscape(name))

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, u, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("registry request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusNotFound {
		return nil, fmt.Errorf("plugin %q not found in registry", name)
	}
	if resp.StatusCode != http.StatusOK {
		return nil, c.handleErrorStatus(name, resp)
	}

	var manifest Manifest
	if err := json.NewDecoder(resp.Body).Decode(&manifest); err != nil {
		return nil, fmt.Errorf("invalid response from registry: %w", err)
	}
	return &manifest, nil
}

func (c *RegistryClient) Download(ctx context.Context, name string, version string) ([]byte, error) {
	u := fmt.Sprintf("%s/plugins/%s/download?version=%s", c.baseURL, url.PathEscape(name), url.QueryEscape(version))

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, u, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("registry request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusNotFound {
		return nil, fmt.Errorf("plugin %q (version %s) not found in registry", name, version)
	}
	if resp.StatusCode == http.StatusPaymentRequired {
		return nil, fmt.Errorf("plugin %q requires a license — visit https://dokrypt.dev/pricing", name)
	}
	if resp.StatusCode != http.StatusOK {
		return nil, c.handleErrorStatus(name, resp)
	}

	data, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("download failed: %w", err)
	}
	return data, nil
}

func (c *RegistryClient) Publish(ctx context.Context, manifest Manifest, archivePath string, token string) error {
	_ = archivePath
	_ = token

	if manifest.Name == "" {
		return fmt.Errorf("plugin name is required")
	}
	if manifest.Version == "" {
		return fmt.Errorf("plugin version is required")
	}

	return fmt.Errorf("plugin registry publishing is coming soon — visit https://hub.dokrypt.dev for updates")
}

func (c *RegistryClient) handleErrorStatus(label string, resp *http.Response) error {
	body, _ := io.ReadAll(io.LimitReader(resp.Body, 1024))
	if len(body) > 0 {
		return fmt.Errorf("registry returned status %d for %q: %s", resp.StatusCode, label, string(body))
	}
	return fmt.Errorf("registry returned status %d for %q", resp.StatusCode, label)
}
