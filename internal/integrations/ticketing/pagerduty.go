package ticketing

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/camronwood/neural-junkie/internal/config"
)

// PagerDutyClient provides read-only PagerDuty incident access.
type PagerDutyClient struct {
	settings config.PagerDutyConfig
}

func NewPagerDutyClient(settings config.PagerDutyConfig) *PagerDutyClient {
	return &PagerDutyClient{settings: settings}
}

func (c *PagerDutyClient) do(ctx context.Context, path string) ([]byte, error) {
	key := strings.TrimSpace(c.settings.APIKey)
	if key == "" {
		return nil, fmt.Errorf("pagerduty api_key not configured")
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, "https://api.pagerduty.com"+path, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Authorization", "Token token="+key)
	req.Header.Set("Accept", "application/vnd.pagerduty+json;version=2")
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
		return data, fmt.Errorf("pagerduty GET %s: %s", path, strings.TrimSpace(string(data)))
	}
	return data, nil
}

func (c *PagerDutyClient) GetIncident(ctx context.Context, id string) (string, error) {
	path := "/incidents/" + urlPathEscape(strings.TrimSpace(id))
	data, err := c.do(ctx, path)
	return string(data), err
}

func (c *PagerDutyClient) ListIncidents(ctx context.Context, statuses []string) (string, error) {
	q := urlQuery("statuses[]", statuses)
	if svc := strings.TrimSpace(c.settings.DefaultServiceID); svc != "" {
		q += "&service_ids[]=" + urlPathEscape(svc)
	}
	path := "/incidents?" + q
	data, err := c.do(ctx, path)
	return string(data), err
}

func urlPathEscape(s string) string {
	return strings.ReplaceAll(s, " ", "%20")
}

func urlQuery(key string, vals []string) string {
	var parts []string
	for _, v := range vals {
		if v = strings.TrimSpace(v); v != "" {
			parts = append(parts, key+"="+urlPathEscape(v))
		}
	}
	return strings.Join(parts, "&")
}

// Stub Provider methods — PagerDuty is alert-source only in v2.
type pagerDutyProvider struct {
	client *PagerDutyClient
}

func NewPagerDutyProvider(settings config.PagerDutyConfig) Provider {
	return &pagerDutyProvider{client: NewPagerDutyClient(settings)}
}

func (p *pagerDutyProvider) Name() string { return "pagerduty" }

func (p *pagerDutyProvider) Get(ctx context.Context, id string) (string, error) {
	return p.client.GetIncident(ctx, id)
}

func (p *pagerDutyProvider) Search(ctx context.Context, query string, max int) (string, error) {
	return p.client.ListIncidents(ctx, []string{"triggered", "acknowledged"})
}

func (p *pagerDutyProvider) Comment(ctx context.Context, id, body string) (string, error) {
	return "", fmt.Errorf("pagerduty: add notes via PagerDuty UI or API v2 note endpoint (not yet implemented)")
}

func (p *pagerDutyProvider) Create(ctx context.Context, req CreateRequest) (string, error) {
	return "", fmt.Errorf("pagerduty: use trigger incident API (write mode not implemented in v2.2)")
}

func (p *pagerDutyProvider) Transition(ctx context.Context, id, status string) (string, error) {
	return "", fmt.Errorf("pagerduty: resolve/ack via PagerDuty UI (write not implemented)")
}

func (p *pagerDutyProvider) Assign(ctx context.Context, id, assignee string) (string, error) {
	return "", fmt.Errorf("pagerduty: reassign via PagerDuty UI")
}

func (p *pagerDutyProvider) SetPriority(ctx context.Context, id, priority string) (string, error) {
	return "", fmt.Errorf("pagerduty: priority is incident urgency field")
}

// Ensure json import used in future extensions
var _ = json.Marshal
