package web

import (
	"context"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/camronwood/neural-junkie/internal/config"
	mcp "github.com/camronwood/neural-junkie/internal/mcp"
	"github.com/camronwood/neural-junkie/internal/integrations/websearch"
	mcpgo "github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server"
)

const maxFetchBytes = 256 * 1024

// AttachTools registers web_search and fetch_url on an MCP server.
// Config is read from hub app config on each tool call. Idempotent if already attached.
func AttachTools(mcpServer *server.MCPServer) {
	if mcpServer == nil {
		return
	}
	if mcpServer.GetTool("web_search") != nil {
		return
	}
	t := &tools{
		http: &http.Client{Timeout: 60 * time.Second},
	}
	t.register(mcpServer)
}

type tools struct {
	http *http.Client
}

func (t *tools) register(mcpServer *server.MCPServer) {
	mcpServer.AddTool(mcp.CreateTool(
		"web_search",
		"Search the public web for current information (news, docs, releases, facts). Use for time-sensitive or external topics; prefer workspace tools for repo-local questions.",
		mcp.CreateMultiStringInputSchema(map[string]string{
			"query": "Search query",
		}),
		nil,
	), t.handleWebSearch)

	mcpServer.AddTool(mcp.CreateTool(
		"fetch_url",
		"Fetch readable text from a public HTTPS URL (use after web_search when you need page detail). Blocks private/loopback hosts.",
		mcp.CreateMultiStringInputSchema(map[string]string{
			"url": "HTTPS URL to fetch",
		}),
		nil,
	), t.handleFetchURL)
}

func (t *tools) handleWebSearch(ctx context.Context, request mcpgo.CallToolRequest) (*mcpgo.CallToolResult, error) {
	query := strings.TrimSpace(request.GetString("query", ""))
	if query == "" {
		return mcp.HandleToolError(fmt.Errorf("missing required parameter: query"), "web_search"), nil
	}
	cfg := config.WebSearchConfig{}
	if c := mcp.AppConfig(); c != nil {
		cfg = c.WebSearch
	}
	client := websearch.NewClient(cfg)
	results, err := client.Search(ctx, query, cfg.MaxResultsOrDefault())
	if err != nil {
		return mcp.HandleToolError(err, "web_search"), nil
	}
	if len(results) == 0 {
		return mcp.HandleToolSuccess("No web results matched that query."), nil
	}
	var b strings.Builder
	for i, r := range results {
		fmt.Fprintf(&b, "%d. %s\n   %s\n   %s\n", i+1, r.Title, r.URL, r.Description)
	}
	return mcp.HandleToolSuccess(strings.TrimSpace(b.String())), nil
}

func (t *tools) handleFetchURL(ctx context.Context, request mcpgo.CallToolRequest) (*mcpgo.CallToolResult, error) {
	rawURL := strings.TrimSpace(request.GetString("url", ""))
	if rawURL == "" {
		return mcp.HandleToolError(fmt.Errorf("missing required parameter: url"), "fetch_url"), nil
	}
	if err := CheckPublicURL(rawURL); err != nil {
		return mcp.HandleToolError(err, "fetch_url"), nil
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, rawURL, nil)
	if err != nil {
		return mcp.HandleToolError(err, "fetch_url"), nil
	}
	req.Header.Set("Accept", "text/html,application/xhtml+xml,text/plain,application/json;q=0.9,*/*;q=0.8")
	req.Header.Set("User-Agent", "NeuralJunkie-Assistant/1.0")
	resp, err := t.http.Do(req)
	if err != nil {
		return mcp.HandleToolError(err, "fetch_url"), nil
	}
	defer resp.Body.Close()
	body, _ := io.ReadAll(io.LimitReader(resp.Body, maxFetchBytes))
	text := strings.TrimSpace(string(body))
	if len(text) > 12000 {
		text = text[:12000] + "\n...(truncated)"
	}
	summary := fmt.Sprintf("HTTP %d %s\n\n%s", resp.StatusCode, rawURL, text)
	return mcp.HandleToolSuccess(summary), nil
}

// CheckPublicURL rejects non-http(s) schemes and private/loopback/link-local
// hosts. Shared by fetch_url and other outbound-HTTP MCP tools (e.g. the MCP
// Tool Wizard's user-defined HTTP-fetch tools) to keep one SSRF gate.
func CheckPublicURL(rawURL string) error {
	u, err := url.Parse(rawURL)
	if err != nil {
		return err
	}
	if u.Scheme != "http" && u.Scheme != "https" {
		return fmt.Errorf("unsupported URL scheme %q", u.Scheme)
	}
	host := u.Hostname()
	if host == "" {
		return fmt.Errorf("missing host")
	}
	if ip := net.ParseIP(host); ip != nil {
		if ip.IsLoopback() || ip.IsPrivate() {
			return fmt.Errorf("SSRF: private/loopback IP not allowed")
		}
	}
	lower := strings.ToLower(host)
	if lower == "localhost" || strings.HasSuffix(lower, ".local") {
		return fmt.Errorf("SSRF: host %q not allowed", host)
	}
	return nil
}
