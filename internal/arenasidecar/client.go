package arenasidecar

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

// SidecarBaseURL resolves the Model Arena pack sidecar base URL when wired by the hub server.
var SidecarBaseURL func() string

// DefaultSidecarClient is wired by the hub at startup.
var DefaultSidecarClient *Client

// Client calls the Model Arena pack sidecar.
type Client struct {
	BaseURL func() string
	HTTP    *http.Client
}

// NewSidecarClient returns a client that posts to a dynamic sidecar base URL.
func NewSidecarClient(baseURL func() string) *Client {
	return &Client{
		BaseURL: baseURL,
		HTTP:    &http.Client{Timeout: 120 * time.Second},
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
	return nil, fmt.Errorf("model arena sidecar not configured")
}

func (c *Client) base() (string, error) {
	client, err := c.resolve()
	if err != nil {
		return "", err
	}
	base := strings.TrimRight(strings.TrimSpace(client.BaseURL()), "/")
	if base == "" {
		return "", fmt.Errorf("model arena sidecar not running (enable Model Arena pack)")
	}
	return base, nil
}

func (c *Client) Get(ctx context.Context, path string) (map[string]any, error) {
	base, err := c.base()
	if err != nil {
		return nil, err
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, base+path, nil)
	if err != nil {
		return nil, err
	}
	client, _ := c.resolve()
	return doJSON(client.HTTP, req)
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
	return doJSON(client.HTTP, req)
}

func doJSON(client *http.Client, req *http.Request) (map[string]any, error) {
	if client == nil {
		client = http.DefaultClient
	}
	resp, err := client.Do(req)
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
		return nil, fmt.Errorf("model arena sidecar: %s", msg)
	}
	if resp.StatusCode >= 400 {
		if errMsg, _ := out["error"].(string); strings.TrimSpace(errMsg) != "" {
			return nil, fmt.Errorf("%s", strings.TrimSpace(errMsg))
		}
		return nil, fmt.Errorf("model arena sidecar: %s", resp.Status)
	}
	return out, nil
}
