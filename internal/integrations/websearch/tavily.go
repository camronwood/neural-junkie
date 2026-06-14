package websearch

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
)

var tavilySearchBaseURL = "https://api.tavily.com/search"

func searchTavily(ctx context.Context, c *Client, query string, limit int) ([]Result, error) {
	payload := map[string]interface{}{
		"query":        query,
		"max_results":  limit,
		"search_depth": "basic",
	}
	body, err := json.Marshal(payload)
	if err != nil {
		return nil, err
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, tavilySearchBaseURL, bytes.NewReader(body))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Accept", "application/json")
	req.Header.Set("Content-Type", "application/json")
	if key := strings.TrimSpace(c.cfg.APIKey); key != "" {
		req.Header.Set("Authorization", "Bearer "+key)
	} else if c.cfg.Keyless {
		req.Header.Set("X-Tavily-Access-Mode", "keyless")
	} else {
		return nil, fmt.Errorf("tavily requires an API key or keyless mode")
	}

	resp, err := c.http.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	respBody, _ := io.ReadAll(io.LimitReader(resp.Body, 256*1024))
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return nil, fmt.Errorf("tavily search HTTP %d: %s", resp.StatusCode, truncateErrBody(respBody))
	}

	var parsed struct {
		Results []struct {
			Title   string `json:"title"`
			URL     string `json:"url"`
			Content string `json:"content"`
		} `json:"results"`
	}
	if err := json.Unmarshal(respBody, &parsed); err != nil {
		return nil, fmt.Errorf("decode tavily search response: %w", err)
	}

	out := make([]Result, 0, len(parsed.Results))
	for _, item := range parsed.Results {
		out = append(out, Result{
			Title:       strings.TrimSpace(item.Title),
			URL:         strings.TrimSpace(item.URL),
			Description: strings.TrimSpace(item.Content),
		})
	}
	return out, nil
}
