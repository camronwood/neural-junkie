package maps

import (
	"context"
	"strings"
	"testing"

	mapsloc "github.com/camronwood/neural-junkie/internal/maps"
	"github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server"
)

func TestLocateToolReturnsFreshShared(t *testing.T) {
	store := mapsloc.NewLocationStore()
	store.Publish(mapsloc.DeviceSnapshot{
		Lat:         41.8826,
		Lon:         -87.6226,
		DisplayName: "Millennium Park",
		Shared:      true,
	})
	srv := server.NewMCPServer("t", "1.0")
	AttachLocateToolWithStore(srv, store)

	tool := &locateTool{store: store}
	res, err := tool.handleLocate(context.Background(), mcp.CallToolRequest{})
	if err != nil {
		t.Fatal(err)
	}
	if res == nil || res.IsError {
		t.Fatalf("unexpected error result: %+v", res)
	}
	text := ""
	for _, c := range res.Content {
		if tc, ok := c.(mcp.TextContent); ok {
			text += tc.Text
		}
	}
	if !strings.Contains(text, "Millennium Park") || !strings.Contains(text, "41.8826") {
		t.Fatalf("missing snapshot in tool output: %s", text)
	}
}
