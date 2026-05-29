package confluencemcp

import (
	"context"
	"fmt"
	"log"
	"strings"
	"sync"
	"time"

	mcp "github.com/camronwood/neural-junkie/internal/mcp"
	"github.com/camronwood/neural-junkie/internal/confluence"
	mcpgo "github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server"
)

// ConfluenceMCP provides in-process MCP tools scoped to a Confluence space index.
type ConfluenceMCP struct {
	mcpServer *server.MCPServer
	spaceKey  string
	client    *confluence.Client
	mu        sync.RWMutex
	index     *confluence.ConfluenceIndex
}

// NewConfluenceMCP creates an in-process MCP server for a Confluence agent.
func NewConfluenceMCP(spaceKey string, client *confluence.Client) (*ConfluenceMCP, error) {
	mcpServer, err := mcp.NewInProcessMCPServer("confluence-agent-mcp", "1.0.0")
	if err != nil {
		return nil, fmt.Errorf("create confluence MCP: %w", err)
	}
	c := &ConfluenceMCP{mcpServer: mcpServer, spaceKey: spaceKey, client: client}
	c.registerTools()
	return c, nil
}

func (c *ConfluenceMCP) GetMCPServer() *server.MCPServer {
	return c.mcpServer
}

func (c *ConfluenceMCP) Start() error {
	return nil
}

func (c *ConfluenceMCP) SetIndex(index *confluence.ConfluenceIndex) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.index = index
}

func (c *ConfluenceMCP) getIndex() *confluence.ConfluenceIndex {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.index
}

func (c *ConfluenceMCP) registerTools() {
	c.mcpServer.AddTool(mcp.CreateTool(
		"search_space",
		"Full-text search across the indexed Confluence space",
		mcp.CreateStringInputSchema("query", "Search query"),
		nil,
	), c.handleSearchSpace)

	c.mcpServer.AddTool(mcp.CreateTool(
		"get_page",
		"Get page content by page ID from the index",
		mcp.CreateStringInputSchema("page_id", "Confluence page ID"),
		nil,
	), c.handleGetPage)

	c.mcpServer.AddTool(mcp.CreateTool(
		"search_by_label",
		"Find pages with a specific label",
		mcp.CreateStringInputSchema("label", "Label name"),
		nil,
	), c.handleSearchByLabel)

	c.mcpServer.AddTool(mcp.CreateTool(
		"list_recent_pages",
		"List pages updated in the last 30 days",
		mcp.CreateStringInputSchema("unused", "Leave empty"),
		nil,
	), c.handleListRecentPages)

	log.Printf("Registered %d Confluence MCP tools", len(c.mcpServer.ListTools()))
}

func (c *ConfluenceMCP) handleSearchSpace(ctx context.Context, request mcpgo.CallToolRequest) (*mcpgo.CallToolResult, error) {
	query := request.GetString("query", "")
	index := c.getIndex()
	if index == nil {
		return mcp.HandleToolError(fmt.Errorf("space index not ready"), "search_space"), nil
	}
	searcher := confluence.NewSearcher(index)
	results := searcher.Search(query, 10)
	if len(results) == 0 {
		return mcp.HandleToolSuccess("No pages matched."), nil
	}
	var b strings.Builder
	for _, r := range results {
		fmt.Fprintf(&b, "- [%s] %s (score %.2f)\n  %s\n", r.Page.ID, r.Page.Title, r.Score, r.Snippet)
	}
	return mcp.HandleToolSuccess(b.String()), nil
}

func (c *ConfluenceMCP) handleGetPage(ctx context.Context, request mcpgo.CallToolRequest) (*mcpgo.CallToolResult, error) {
	pageID := request.GetString("page_id", "")
	index := c.getIndex()
	if index == nil {
		return mcp.HandleToolError(fmt.Errorf("index not ready"), "get_page"), nil
	}
	page, ok := index.GetPage(pageID)
	if !ok {
		return mcp.HandleToolError(fmt.Errorf("page not found: %s", pageID), "get_page"), nil
	}
	content := page.Content
	if c.client != nil {
		if live, err := c.client.GetPageContent(pageID); err == nil && live != nil {
			content = live.Body.Storage.Value
		}
	}
	return mcp.HandleToolSuccess(fmt.Sprintf("# %s\n\n%s", page.Title, content)), nil
}

func (c *ConfluenceMCP) handleSearchByLabel(ctx context.Context, request mcpgo.CallToolRequest) (*mcpgo.CallToolResult, error) {
	label := request.GetString("label", "")
	index := c.getIndex()
	if index == nil {
		return mcp.HandleToolError(fmt.Errorf("index not ready"), "search_by_label"), nil
	}
	searcher := confluence.NewSearcher(index)
	results := searcher.SearchByLabel(label)
	if len(results) == 0 {
		return mcp.HandleToolSuccess("No pages with that label."), nil
	}
	var titles []string
	for _, r := range results {
		titles = append(titles, r.Page.Title)
	}
	return mcp.HandleToolSuccess(strings.Join(titles, "\n")), nil
}

func (c *ConfluenceMCP) handleListRecentPages(ctx context.Context, request mcpgo.CallToolRequest) (*mcpgo.CallToolResult, error) {
	index := c.getIndex()
	if index == nil {
		return mcp.HandleToolError(fmt.Errorf("index not ready"), "list_recent_pages"), nil
	}
	end := time.Now()
	start := end.AddDate(0, 0, -30)
	searcher := confluence.NewSearcher(index)
	results := searcher.SearchByDateRange(start, end)
	if len(results) == 0 {
		return mcp.HandleToolSuccess("No recently updated pages in the last 30 days."), nil
	}
	var b strings.Builder
	for _, r := range results {
		fmt.Fprintf(&b, "- %s (updated %s)\n", r.Page.Title, r.Page.LastUpdated.Format("2006-01-02"))
	}
	return mcp.HandleToolSuccess(b.String()), nil
}
