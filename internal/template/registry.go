package template

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

func (c *RegistryClient) Search(ctx context.Context, query string) ([]Template, error) {
	u := fmt.Sprintf("%s/templates/search?q=%s", c.baseURL, url.QueryEscape(query))

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

	var results []Template
	if err := json.NewDecoder(resp.Body).Decode(&results); err != nil {
		return nil, fmt.Errorf("invalid response from registry: %w", err)
	}
	return results, nil
}

func (c *RegistryClient) Pull(ctx context.Context, name string) ([]byte, error) {
	u := fmt.Sprintf("%s/templates/%s/download", c.baseURL, url.PathEscape(name))

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
		return nil, fmt.Errorf("template %q not found in registry", name)
	}
	if resp.StatusCode == http.StatusPaymentRequired {
		return nil, fmt.Errorf("template %q requires a license — visit https://dokrypt.dev/pricing", name)
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

func (c *RegistryClient) Push(ctx context.Context, meta Template, archivePath string, token string) error {
	_ = archivePath
	_ = token

	if meta.Name == "" {
		return fmt.Errorf("template name is required")
	}
	if meta.Version == "" {
		return fmt.Errorf("template version is required")
	}

	return fmt.Errorf("template registry publishing is coming soon — visit https://hub.dokrypt.dev for updates")
}

func (c *RegistryClient) handleErrorStatus(label string, resp *http.Response) error {
	body, _ := io.ReadAll(io.LimitReader(resp.Body, 1024))
	if len(body) > 0 {
		return fmt.Errorf("registry returned status %d for %q: %s", resp.StatusCode, label, string(body))
	}
	return fmt.Errorf("registry returned status %d for %q", resp.StatusCode, label)
}
