package maps

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

// SidecarBaseURL resolves the maps pack sidecar base URL when wired by the hub server.
var SidecarBaseURL func() string

// Client calls the maps pack sidecar (/api/maps/*).
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
		HTTP:    &http.Client{Timeout: 60 * time.Second},
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
	return nil, fmt.Errorf("maps sidecar not configured")
}

func (c *Client) base() (string, error) {
	client, err := c.resolve()
	if err != nil {
		return "", err
	}
	base := strings.TrimRight(strings.TrimSpace(client.BaseURL()), "/")
	if base == "" {
		return "", fmt.Errorf("maps sidecar not running (enable Maps pack)")
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
		return nil, fmt.Errorf("maps sidecar: %s", msg)
	}
	if resp.StatusCode >= 400 {
		if errMsg, _ := out["error"].(string); strings.TrimSpace(errMsg) != "" {
			return nil, fmt.Errorf("%s", strings.TrimSpace(errMsg))
		}
		return nil, fmt.Errorf("maps sidecar: %s", resp.Status)
	}
	return out, nil
}

// Geocode resolves a place query via Nominatim through the maps sidecar.
// Optional keys: limit, near {lat,lon}, viewbox.
func (c *Client) Geocode(ctx context.Context, req map[string]any) (map[string]any, error) {
	return c.do(ctx, http.MethodPost, "/api/maps/geocode", req)
}

// Reverse resolves lat/lon to a display name via Nominatim through the maps sidecar.
func (c *Client) Reverse(ctx context.Context, req map[string]any) (map[string]any, error) {
	return c.do(ctx, http.MethodPost, "/api/maps/reverse", req)
}

// Route computes a walking/driving route via OSRM through the maps sidecar.
func (c *Client) Route(ctx context.Context, req map[string]any) (map[string]any, error) {
	return c.do(ctx, http.MethodPost, "/api/maps/route", req)
}
