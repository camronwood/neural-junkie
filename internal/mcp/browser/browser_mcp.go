package browser

import (
	"fmt"
	"log"

	mcp "github.com/camronwood/neural-junkie/internal/mcp"
	webmcp "github.com/camronwood/neural-junkie/internal/mcp/web"
	"github.com/mark3labs/mcp-go/server"
)

// BrowserMCP provides MCP tools for web browsing and HTML preview workflows.
type BrowserMCP struct {
	mcpServer  *server.MCPServer
	httpServer *server.StreamableHTTPServer
	config     *mcp.MCPServerConfig
}

// NewBrowserMCP creates a new web browser MCP server.
func NewBrowserMCP() (*BrowserMCP, error) {
	cfg := mcp.GetMCPServerConfig("BROWSER")
	mcpServer, httpServer, err := mcp.NewMCPServer(cfg)
	if err != nil {
		return nil, fmt.Errorf("failed to create MCP server: %w", err)
	}
	b := &BrowserMCP{mcpServer: mcpServer, httpServer: httpServer, config: cfg}
	b.registerTools()
	return b, nil
}

func (b *BrowserMCP) Start() error {
	return mcp.StartMCPServer(b.httpServer, b.config.Port)
}

func (b *BrowserMCP) GetMCPServer() *server.MCPServer {
	return b.mcpServer
}

func (b *BrowserMCP) registerTools() {
	webmcp.AttachTools(b.mcpServer)
	log.Printf("Registered %d web browser MCP tools", len(b.mcpServer.ListTools()))
}
