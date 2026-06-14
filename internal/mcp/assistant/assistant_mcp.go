// Package assistant provides in-process workspace MCP tools for the personal assistant agent.
package assistant

import (
	"fmt"

	mcp "github.com/camronwood/neural-junkie/internal/mcp"
	"github.com/camronwood/neural-junkie/internal/mcp/workspace"
	webmcp "github.com/camronwood/neural-junkie/internal/mcp/web"
	"github.com/mark3labs/mcp-go/server"
)

// AssistantMCP exposes read-only workspace file tools for the assistant agent.
type AssistantMCP struct {
	mcpServer *server.MCPServer
}

// NewAssistantMCP creates an in-process MCP server with workspace file tools only.
func NewAssistantMCP() (*AssistantMCP, error) {
	mcpServer, err := mcp.NewInProcessMCPServer("assistant-workspace-mcp", "1.0.0")
	if err != nil {
		return nil, fmt.Errorf("create assistant MCP: %w", err)
	}
	return &AssistantMCP{mcpServer: mcpServer}, nil
}

// GetMCPServer returns the underlying MCP server.
func (a *AssistantMCP) GetMCPServer() *server.MCPServer {
	return a.mcpServer
}

// Start is a no-op for in-process MCP servers.
func (a *AssistantMCP) Start() error {
	return nil
}

// AttachWorkspaceTools registers read_file, grep, glob_file_search, list_dir, and semantic_search
// using the agent's current WorkspacePath as root.
func (a *AssistantMCP) AttachWorkspaceTools(root workspace.RootResolver) {
	workspace.AttachTools(a.mcpServer, root)
}

// AttachWebTools registers web_search and fetch_url (requires hub web search config).
func (a *AssistantMCP) AttachWebTools() {
	webmcp.AttachTools(a.mcpServer)
}
