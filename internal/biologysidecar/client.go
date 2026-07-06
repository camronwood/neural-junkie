package biologysidecar

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

// SidecarBaseURL resolves the life-sciences pack sidecar base URL when wired by the hub server.
var SidecarBaseURL func() string

// Client calls the biology pack Python sidecar.
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
		HTTP:    &http.Client{Timeout: 10 * time.Minute},
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
	return nil, fmt.Errorf("biology sidecar not configured")
}

func (c *Client) base() (string, error) {
	client, err := c.resolve()
	if err != nil {
		return "", err
	}
	base := strings.TrimRight(strings.TrimSpace(client.BaseURL()), "/")
	if base == "" {
		return "", fmt.Errorf("biology sidecar not running (enable Life sciences pack and run setup-biology-sidecar.sh)")
	}
	return base, nil
}

// Available reports whether the biology sidecar base URL is configured.
func (c *Client) Available() bool {
	base, err := c.base()
	return err == nil && base != ""
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
		return nil, fmt.Errorf("biology sidecar: %s", msg)
	}
	if resp.StatusCode >= 400 {
		if errMsg, _ := out["error"].(string); strings.TrimSpace(errMsg) != "" {
			return nil, fmt.Errorf("%s", strings.TrimSpace(errMsg))
		}
		return nil, fmt.Errorf("biology sidecar: %s", resp.Status)
	}
	return out, nil
}

// Status returns sidecar readiness.
func (c *Client) Status(ctx context.Context) (map[string]any, error) {
	base, err := c.base()
	if err != nil {
		return nil, err
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, base+"/api/biology/status", nil)
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
		return nil, fmt.Errorf("biology sidecar status: %s", resp.Status)
	}
	return out, nil
}

// FoldProtein delegates folding to the pack sidecar.
func (c *Client) FoldProtein(ctx context.Context, sequence string, maxLen int) (string, error) {
	out, err := c.Post(ctx, "/api/biology/fold", map[string]any{
		"sequence":   sequence,
		"max_length": maxLen,
	})
	if err != nil {
		return "", err
	}
	if summary, _ := out["summary"].(string); strings.TrimSpace(summary) != "" {
		return summary, nil
	}
	path, _ := out["pdb_path"].(string)
	if path == "" {
		return "", fmt.Errorf("biology sidecar fold returned no pdb_path")
	}
	return fmt.Sprintf("Structure prediction complete (in silico)\nPDB file: %s\n", path), nil
}
