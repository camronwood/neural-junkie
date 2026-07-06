package cadcsidecar

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"
)

// SidecarBaseURL resolves the CAD pack sidecar base URL when wired by the hub server.
var SidecarBaseURL func() string

// Client calls the CAD pack sidecar.
type Client struct {
	BaseURL func() string
	HTTP    *http.Client
}

// DefaultSidecarClient is wired by the hub at startup.
var DefaultSidecarClient *Client

// NewSidecarClient returns a client that posts to a dynamic sidecar base URL.
func NewSidecarClient(baseURL func() string) *Client {
	return &Client{
		BaseURL: baseURL,
		HTTP:    &http.Client{Timeout: 180 * time.Second},
	}
}

func (c *Client) resolve() (*Client, error) {
	if c != nil && c.BaseURL != nil {
		return c, nil
	}
	if DefaultSidecarClient != nil {
		return DefaultSidecarClient, nil
	}
	if SidecarBaseURL != nil {
		return NewSidecarClient(SidecarBaseURL), nil
	}
	return nil, fmt.Errorf("cad sidecar not configured")
}

func (c *Client) base() (string, error) {
	client, err := c.resolve()
	if err != nil {
		return "", err
	}
	base := strings.TrimRight(strings.TrimSpace(client.BaseURL()), "/")
	if base == "" {
		return "", fmt.Errorf("cad sidecar not running (enable CAD pack and run setup-cad-sidecar.sh)")
	}
	return base, nil
}

func (c *Client) Post(ctx context.Context, path string, body map[string]any) (map[string]any, error) {
	base, err := c.base()
	if err != nil {
		return nil, err
	}
	var reader io.Reader
	if body != nil {
		raw, err := json.Marshal(body)
		if err != nil {
			return nil, err
		}
		reader = bytes.NewReader(raw)
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, base+path, reader)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", "application/json")
	client, _ := c.resolve()
	httpClient := client.HTTP
	if httpClient == nil {
		httpClient = http.DefaultClient
	}
	resp, err := httpClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	raw, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}
	var out map[string]any
	if err := json.Unmarshal(raw, &out); err != nil {
		msg := strings.TrimSpace(string(raw))
		if msg == "" {
			msg = resp.Status
		}
		return nil, fmt.Errorf("cad sidecar: %s", msg)
	}
	if resp.StatusCode >= 400 {
		if errMsg, _ := out["error"].(string); strings.TrimSpace(errMsg) != "" {
			return nil, fmt.Errorf("%s", strings.TrimSpace(errMsg))
		}
		return nil, fmt.Errorf("cad sidecar: %s", resp.Status)
	}
	return out, nil
}

// PostJSON marshals the sidecar response as indented JSON for MCP tool output.
func (c *Client) PostJSON(ctx context.Context, path string, body map[string]any) (string, error) {
	out, err := c.Post(ctx, path, body)
	if err != nil {
		return "", err
	}
	raw, err := json.MarshalIndent(out, "", "  ")
	if err != nil {
		return "", err
	}
	return string(raw), nil
}

// Status returns sidecar readiness.
func (c *Client) Status(ctx context.Context) (map[string]any, error) {
	base, err := c.base()
	if err != nil {
		return nil, err
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, base+"/api/cad/status", nil)
	if err != nil {
		return nil, err
	}
	client, _ := c.resolve()
	httpClient := client.HTTP
	if httpClient == nil {
		httpClient = http.DefaultClient
	}
	resp, err := httpClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	raw, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}
	var out map[string]any
	if err := json.Unmarshal(raw, &out); err != nil {
		return nil, err
	}
	if resp.StatusCode >= 400 {
		return nil, fmt.Errorf("cad sidecar status: %s", resp.Status)
	}
	return out, nil
}
