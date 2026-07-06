package browser

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

// SidecarBaseURL resolves the web-browser pack sidecar base URL when wired by the hub server.
var SidecarBaseURL func() string

// Client calls the browser automation pack sidecar.
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
	return nil, fmt.Errorf("browser sidecar not configured")
}

func (c *Client) base() (string, error) {
	client, err := c.resolve()
	if err != nil {
		return "", err
	}
	base := strings.TrimRight(strings.TrimSpace(client.BaseURL()), "/")
	if base == "" {
		return "", fmt.Errorf("browser sidecar not running (enable Web browser pack and run setup-playwright.sh)")
	}
	return base, nil
}

func (c *Client) do(ctx context.Context, method, path string, body any) (map[string]any, error) {
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
	req, err := http.NewRequestWithContext(ctx, method, base+path, reader)
	if err != nil {
		return nil, err
	}
	if body != nil {
		req.Header.Set("Content-Type", "application/json")
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
		msg := strings.TrimSpace(string(raw))
		if msg == "" {
			msg = resp.Status
		}
		return nil, fmt.Errorf("browser sidecar: %s", msg)
	}
	if resp.StatusCode >= 400 {
		if errMsg, _ := out["error"].(string); strings.TrimSpace(errMsg) != "" {
			return nil, fmt.Errorf("%s", strings.TrimSpace(errMsg))
		}
		return nil, fmt.Errorf("browser sidecar: %s", resp.Status)
	}
	return out, nil
}

// Status returns sidecar readiness.
func (c *Client) Status(ctx context.Context) (map[string]any, error) {
	return c.do(ctx, http.MethodGet, "/api/browser/status", nil)
}

// Screenshot captures a page PNG (returns metadata map with png_b64).
func (c *Client) Screenshot(ctx context.Context, req map[string]any) (map[string]any, error) {
	return c.do(ctx, http.MethodPost, "/api/browser/screenshot", req)
}

// Navigate opens a URL and returns a session id.
func (c *Client) Navigate(ctx context.Context, req map[string]any) (map[string]any, error) {
	return c.do(ctx, http.MethodPost, "/api/browser/navigate", req)
}

// Click clicks an element in a session.
func (c *Client) Click(ctx context.Context, req map[string]any) (map[string]any, error) {
	return c.do(ctx, http.MethodPost, "/api/browser/click", req)
}

// Fill fills a form field in a session.
func (c *Client) Fill(ctx context.Context, req map[string]any) (map[string]any, error) {
	return c.do(ctx, http.MethodPost, "/api/browser/fill", req)
}

// A11yAudit runs axe-core on a URL.
func (c *Client) A11yAudit(ctx context.Context, req map[string]any) (map[string]any, error) {
	return c.do(ctx, http.MethodPost, "/api/browser/a11y-audit", req)
}

// Metrics collects Lighthouse-lite performance metrics.
func (c *Client) Metrics(ctx context.Context, req map[string]any) (map[string]any, error) {
	return c.do(ctx, http.MethodPost, "/api/browser/metrics", req)
}

// VisualDiff compares a screenshot to a baseline PNG on disk.
func (c *Client) VisualDiff(ctx context.Context, req map[string]any) (map[string]any, error) {
	return c.do(ctx, http.MethodPost, "/api/browser/visual-diff", req)
}

// PickElement returns DOM info for coordinates on a page.
func (c *Client) PickElement(ctx context.Context, req map[string]any) (map[string]any, error) {
	return c.do(ctx, http.MethodPost, "/api/browser/pick-element", req)
}
