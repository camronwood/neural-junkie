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
		"Retrieve original uncompressed context cached by Neural Junkie compression (ref from compressed marker)",
		mcp.CreateMultiStringInputSchema(map[string]string{
			"ref":   "Cache ref from compression marker (e.g. ctx-abc123)",
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
