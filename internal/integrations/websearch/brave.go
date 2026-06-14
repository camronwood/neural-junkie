package websearch

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
)

var braveSearchBaseURL = "https://api.search.brave.com/res/v1/web/search"

func searchBrave(ctx context.Context, c *Client, query string, limit int) ([]Result, error) {
	base, err := url.Parse(braveSearchBaseURL)
	if err != nil {
		return nil, err
	}
	q := base.Query()
	q.Set("q", query)
	q.Set("count", fmt.Sprintf("%d", limit))
	base.RawQuery = q.Encode()

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, base.String(), nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Accept", "application/json")
	req.Header.Set("X-Subscription-Token", strings.TrimSpace(c.cfg.APIKey))

	resp, err := c.http.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	body, _ := io.ReadAll(io.LimitReader(resp.Body, 256*1024))
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return nil, fmt.Errorf("brave search HTTP %d: %s", resp.StatusCode, truncateErrBody(body))
	}

	var payload struct {
		Web struct {
			Results []struct {
				Title       string `json:"title"`
				URL         string `json:"url"`
				Description string `json:"description"`
			} `json:"results"`
		} `json:"web"`
	}
	if err := json.Unmarshal(body, &payload); err != nil {
		return nil, fmt.Errorf("decode brave search response: %w", err)
	}

	out := make([]Result, 0, len(payload.Web.Results))
	for _, item := range payload.Web.Results {
		out = append(out, Result{
			Title:       strings.TrimSpace(item.Title),
			URL:         strings.TrimSpace(item.URL),
			Description: strings.TrimSpace(item.Description),
		})
	}
	return out, nil
}

func truncateErrBody(body []byte) string {
	msg := strings.TrimSpace(string(body))
	if len(msg) > 200 {
		return msg[:200] + "..."
	}
	if msg == "" {
		return "request failed"
	}
	return msg
}
