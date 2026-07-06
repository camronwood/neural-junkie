package incidentsidecar

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"
)

// SidecarBaseURL resolves the incident-management pack sidecar base URL.
var SidecarBaseURL func() string

// DefaultClient calls incident pack sidecar routes.
var DefaultClient = &Client{BaseURL: func() string {
	if SidecarBaseURL != nil {
		return SidecarBaseURL()
	}
	return ""
}}

type Client struct {
	BaseURL func() string
}

func (c *Client) base() (string, error) {
	url := ""
	if c != nil && c.BaseURL != nil {
		url = c.BaseURL()
	}
	if url == "" {
		return "", fmt.Errorf("incident sidecar not running — enable incident-management pack")
	}
	return url, nil
}

func (c *Client) PostJSON(path string, body map[string]interface{}) (string, error) {
	base, err := c.base()
	if err != nil {
		return "", err
	}
	raw, err := json.Marshal(body)
	if err != nil {
		return "", err
	}
	req, err := http.NewRequest(http.MethodPost, base+path, bytes.NewReader(raw))
	if err != nil {
		return "", err
	}
	req.Header.Set("Content-Type", "application/json")
	client := &http.Client{Timeout: 60 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()
	data, err := io.ReadAll(io.LimitReader(resp.Body, 2<<20))
	if err != nil {
		return "", err
	}
	if resp.StatusCode >= 400 {
		return string(data), fmt.Errorf("incident sidecar %s: %s", path, string(data))
	}
	return string(data), nil
}

func (c *Client) Get(path string) (string, error) {
	base, err := c.base()
	if err != nil {
		return "", err
	}
	req, err := http.NewRequest(http.MethodGet, base+path, nil)
	if err != nil {
		return "", err
	}
	client := &http.Client{Timeout: 30 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()
	data, err := io.ReadAll(io.LimitReader(resp.Body, 2<<20))
	if err != nil {
		return "", err
	}
	if resp.StatusCode >= 400 {
		return string(data), fmt.Errorf("incident sidecar GET %s: %s", path, string(data))
	}
	return string(data), nil
}
