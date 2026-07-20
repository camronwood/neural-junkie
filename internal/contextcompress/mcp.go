package contextcompress

import (
	"context"
	"fmt"

	mcp "github.com/camronwood/neural-junkie/internal/mcp"
	mcpgo "github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server"
)

const retrieveToolName = "nj_retrieve_context"

// AttachRetrieveTool registers the CCR retrieval tool on an MCP server.
func AttachRetrieveTool(mcpServer *server.MCPServer, store *Store) {
	if mcpServer == nil {
		return
	}
	if store == nil {
		store = DefaultStore()
	}
	s := store
	mcpServer.AddTool(mcp.CreateTool(
		retrieveToolName,
		"Expand a compressed tool result. Only call when this turn already includes a compression marker with a real ctx-… ref. Never invent or guess a ref; if no marker is present, do not call this tool.",
		mcp.CreateMultiStringInputSchema(map[string]string{
			"ref":   "Exact ctx-… id copied from a compression marker in this turn (12 hex chars after ctx-). Never use invented or example values.",
			"query": "Optional: filter to lines containing this substring",
		}),
		nil,
	), func(ctx context.Context, req mcpgo.CallToolRequest) (*mcpgo.CallToolResult, error) {
		if !TryConsumeRetrieve(ctx) {
			return mcp.HandleToolError(
				fmt.Errorf("retrieve budget exceeded for this turn (max %d)", maxRetrievePerTurn),
				retrieveToolName,
			), nil
		}
		ref := req.GetString("ref", "")
		query := req.GetString("query", "")
		text, err := Retrieve(s, ref, query)
		if err != nil {
			return mcp.HandleToolError(err, retrieveToolName), nil
		}
		return mcp.HandleToolSuccess(text), nil
	})
}
