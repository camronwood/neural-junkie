package maps

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"strings"

	mapsloc "github.com/camronwood/neural-junkie/internal/maps"
	mcp "github.com/camronwood/neural-junkie/internal/mcp"
	mcpgo "github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server"
)

// AttachLocateTool registers maps_locate (sensitive device location).
// Do not fold this into AttachGeocodeRouteTools — that bundle is exposure:safe.
func AttachLocateTool(mcpServer *server.MCPServer) {
	AttachLocateToolWithStore(mcpServer, nil)
}

// AttachLocateToolWithStore is like AttachLocateTool but uses an explicit store.
func AttachLocateToolWithStore(mcpServer *server.MCPServer, store *mapsloc.LocationStore) {
	if mcpServer == nil {
		return
	}
	t := &locateTool{store: store}
	mcpServer.AddTool(mcp.CreateTool(
		"maps_locate",
		"Read the user's device location. Returns lat/lon, accuracy, and a place name when available. Requires the Device location capability grant and user consent. Use for near-me search, routes from here, or when the granted session location is stale. Never invent coordinates.",
		mcp.CreateObjectInputSchema(map[string]interface{}{
			"reason": map[string]interface{}{
				"type":        "string",
				"description": "Optional short reason shown to the user when a fresh reading is needed",
			},
		}, []string{}),
		nil,
	), t.handleLocate)
	log.Printf("Registered maps_locate MCP tool")
}

type locateTool struct {
	store *mapsloc.LocationStore
}

func (t *locateTool) locationStore() *mapsloc.LocationStore {
	if t.store != nil {
		return t.store
	}
	return mapsloc.DefaultLocationStore
}

func (t *locateTool) handleLocate(ctx context.Context, request mcpgo.CallToolRequest) (*mcpgo.CallToolResult, error) {
	store := t.locationStore()
	if store == nil {
		return mcp.HandleToolError(fmt.Errorf("location store not configured"), "maps_locate"), nil
	}
	if view, ok := store.FreshShared(); ok {
		if strings.TrimSpace(view.DisplayName) == "" {
			view.DisplayName = reverseDisplayName(ctx, view.Lat, view.Lon)
		}
		raw, err := json.MarshalIndent(view, "", "  ")
		if err != nil {
			return mcp.HandleToolError(err, "maps_locate"), nil
		}
		return mcp.HandleToolSuccess(string(raw)), nil
	}

	req := store.RequestLocate("", "Assistant", "")
	done := make(chan struct{})
	var out *mapsloc.LocateRequest
	var waitErr error
	go func() {
		out, waitErr = store.WaitLocate(req.ID, mapsloc.LocateRequestTTL)
		close(done)
	}()
	select {
	case <-ctx.Done():
		_, _ = store.Reject(req.ID, "cancelled")
		return mcp.HandleToolError(fmt.Errorf("maps_locate cancelled"), "maps_locate"), nil
	case <-done:
	}
	if waitErr != nil {
		return mcp.HandleToolError(waitErr, "maps_locate"), nil
	}
	if out == nil || out.Status != "fulfilled" || out.Snapshot == nil {
		reason := "user declined to share location"
		if out != nil && strings.TrimSpace(out.Reason) != "" {
			reason = out.Reason
		}
		return mcp.HandleToolError(fmt.Errorf("%s", reason), "maps_locate"), nil
	}
	view := *out.Snapshot
	if strings.TrimSpace(view.DisplayName) == "" {
		view.DisplayName = reverseDisplayName(ctx, view.Lat, view.Lon)
	}
	raw, err := json.MarshalIndent(view, "", "  ")
	if err != nil {
		return mcp.HandleToolError(err, "maps_locate"), nil
	}
	return mcp.HandleToolSuccess(string(raw)), nil
}

func reverseDisplayName(ctx context.Context, lat, lon float64) string {
	client := mapsloc.DefaultSidecarClient
	if client == nil {
		return ""
	}
	out, err := client.Reverse(ctx, map[string]any{"lat": lat, "lon": lon})
	if err != nil {
		return ""
	}
	name, _ := out["display_name"].(string)
	return strings.TrimSpace(name)
}
