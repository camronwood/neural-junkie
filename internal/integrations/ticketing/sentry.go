package ticketing

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/camronwood/neural-junkie/internal/config"
)

// SentryClient provides read-only Sentry issue/event access.
type SentryClient struct {
	settings config.SentryConfig
}

func NewSentryClient(settings config.SentryConfig) *SentryClient {
	return &SentryClient{settings: settings}
}

func (c *SentryClient) baseURL() string {
	org := strings.TrimSpace(c.settings.DefaultOrg)
	return fmt.Sprintf("https://sentry.io/api/0/organizations/%s", org)
}

func (c *SentryClient) do(ctx context.Context, path string) ([]byte, error) {
	token := strings.TrimSpace(c.settings.AuthToken)
	if token == "" {
		return nil, fmt.Errorf("sentry auth_token not configured")
	}
	if strings.TrimSpace(c.settings.DefaultOrg) == "" {
		return nil, fmt.Errorf("sentry default_org not configured")
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, c.baseURL()+path, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Authorization", "Bearer "+token)
	client := &http.Client{Timeout: 45 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	data, err := io.ReadAll(io.LimitReader(resp.Body, 2<<20))
	if err != nil {
		return nil, err
	}
	if resp.StatusCode >= 400 {
		return data, fmt.Errorf("sentry GET %s: %s", path, strings.TrimSpace(string(data)))
	}
	return data, nil
}

func (c *SentryClient) GetIssue(ctx context.Context, issueID string) (string, error) {
	proj := strings.TrimSpace(c.settings.DefaultProject)
	if proj == "" {
		return "", fmt.Errorf("sentry default_project required")
	}
	path := fmt.Sprintf("/issues/%s/", strings.TrimSpace(issueID))
	data, err := c.do(ctx, path)
	return string(data), err
}

func (c *SentryClient) GetLatestEvent(ctx context.Context, issueID string) (string, error) {
	path := fmt.Sprintf("/issues/%s/events/latest/", strings.TrimSpace(issueID))
	data, err := c.do(ctx, path)
	return string(data), err
}

func (c *SentryClient) ListIssues(ctx context.Context, query string) (string, error) {
	proj := strings.TrimSpace(c.settings.DefaultProject)
	if proj == "" {
		return "", fmt.Errorf("sentry default_project required")
	}
	path := fmt.Sprintf("/issues/?project=%s&query=%s", proj, strings.ReplaceAll(query, " ", "+"))
	data, err := c.do(ctx, path)
	return string(data), err
}
