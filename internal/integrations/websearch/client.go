package websearch

import (
	"context"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/camronwood/neural-junkie/internal/config"
)

const defaultTimeout = 30 * time.Second

// Result is one web search hit returned to MCP tools and runbook actions.
type Result struct {
	Title       string `json:"title"`
	URL         string `json:"url"`
	Description string `json:"description"`
}

// Client queries a configured web search provider.
type Client struct {
	cfg  config.WebSearchConfig
	http *http.Client
}

// NewClient builds a search client from hub config.
func NewClient(cfg config.WebSearchConfig) *Client {
	return &Client{
		cfg: cfg,
		http: &http.Client{
			Timeout: defaultTimeout,
		},
	}
}

// Ready reports whether search can run.
func (c *Client) Ready() bool {
	if c == nil {
		return false
	}
	return c.cfg.Ready()
}

// Search runs a web search query.
func (c *Client) Search(ctx context.Context, query string, limit int) ([]Result, error) {
	if c == nil || !c.Ready() {
		return nil, fmt.Errorf("web search is not configured (enable in Settings; for Tavily use an API key or keyless mode, for Brave set an API key)")
	}
	query = strings.TrimSpace(query)
	if query == "" {
		return nil, fmt.Errorf("query is required")
	}
	if limit <= 0 {
		limit = c.cfg.MaxResultsOrDefault()
	}
	if limit > 20 {
		limit = 20
	}
	switch c.cfg.ProviderName() {
	case "tavily":
		return searchTavily(ctx, c, query, limit)
	case "brave":
		return searchBrave(ctx, c, query, limit)
	default:
		return nil, fmt.Errorf("unsupported web search provider %q", c.cfg.ProviderName())
	}
}

// ResultsForRunbook adapts search hits for collaboration action tasks.
func ResultsForRunbook(results []Result) []map[string]interface{} {
	out := make([]map[string]interface{}, 0, len(results))
	for _, r := range results {
		out = append(out, map[string]interface{}{
			"title":       r.Title,
			"url":         r.URL,
			"description": r.Description,
		})
	}
	return out
}
