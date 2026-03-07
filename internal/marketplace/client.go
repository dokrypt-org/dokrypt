package marketplace

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"time"
)

const (
	DefaultHubURL = "https://hub.dokrypt.com/api/v1"
)

type Client struct {
	baseURL    string
	httpClient *http.Client
}

func NewClient(baseURL string) *Client {
	if baseURL == "" {
		baseURL = DefaultHubURL
	}
	return &Client{
		baseURL: baseURL,
		httpClient: &http.Client{
			Timeout: 30 * time.Second,
		},
	}
}

func (c *Client) Search(query string) (*SearchResult, error) {
	u := fmt.Sprintf("%s/templates/search?q=%s", c.baseURL, url.QueryEscape(query))
	resp, err := c.httpClient.Get(u)
	if err != nil {
		return nil, fmt.Errorf("marketplace request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("marketplace returned status %d", resp.StatusCode)
	}

	var result SearchResult
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, fmt.Errorf("invalid response: %w", err)
	}
	return &result, nil
}

func (c *Client) Browse(category string) (*SearchResult, error) {
	u := fmt.Sprintf("%s/templates?category=%s", c.baseURL, url.QueryEscape(category))
	resp, err := c.httpClient.Get(u)
	if err != nil {
		return nil, fmt.Errorf("marketplace request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("marketplace returned status %d", resp.StatusCode)
	}

	var result SearchResult
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, fmt.Errorf("invalid response: %w", err)
	}
	return &result, nil
}

func (c *Client) GetInfo(name string) (*PackageMeta, error) {
	u := fmt.Sprintf("%s/templates/%s", c.baseURL, url.PathEscape(name))
	resp, err := c.httpClient.Get(u)
	if err != nil {
		return nil, fmt.Errorf("marketplace request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusNotFound {
		return nil, fmt.Errorf("template %q not found in marketplace", name)
	}
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("marketplace returned status %d", resp.StatusCode)
	}

	var meta PackageMeta
	if err := json.NewDecoder(resp.Body).Decode(&meta); err != nil {
		return nil, fmt.Errorf("invalid response: %w", err)
	}
	return &meta, nil
}

func (c *Client) Download(name string) ([]byte, error) {
	u := fmt.Sprintf("%s/templates/%s/download", c.baseURL, url.PathEscape(name))
	resp, err := c.httpClient.Get(u)
	if err != nil {
		return nil, fmt.Errorf("marketplace request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusNotFound {
		return nil, fmt.Errorf("template %q not found in marketplace", name)
	}
	if resp.StatusCode == http.StatusPaymentRequired {
		return nil, fmt.Errorf("template %q requires a license — visit https://dokrypt.com/pricing", name)
	}
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("marketplace returned status %d", resp.StatusCode)
	}

	data, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("download failed: %w", err)
	}
	return data, nil
}

func (c *Client) Publish(meta PackageMeta, archivePath string, token string) error {
	_ = archivePath
	_ = token

	if meta.Name == "" {
		return fmt.Errorf("template name is required")
	}
	if meta.Version == "" {
		return fmt.Errorf("template version is required")
	}

	return fmt.Errorf("marketplace hub publishing is coming soon — visit https://hub.dokrypt.com for updates")
}
