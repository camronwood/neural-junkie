package maps

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"strings"

	mapssideecar "github.com/camronwood/neural-junkie/internal/maps"
	mcp "github.com/camronwood/neural-junkie/internal/mcp"
	mcpgo "github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server"
)

// AttachGeocodeRouteTools registers maps_geocode and maps_route on any MCP server
// (MapsExpert, Assistant, or Composition-granted custom experts).
func AttachGeocodeRouteTools(mcpServer *server.MCPServer) {
	AttachGeocodeRouteToolsWithClient(mcpServer, nil)
}

// AttachGeocodeRouteToolsWithClient is like AttachGeocodeRouteTools but uses an explicit sidecar client.
func AttachGeocodeRouteToolsWithClient(mcpServer *server.MCPServer, client *mapssideecar.Client) {
	if mcpServer == nil {
		return
	}
	t := &geocodeRouteTools{client: client}
	t.register(mcpServer)
	log.Printf("Registered maps geocode/route MCP tools")
}

type geocodeRouteTools struct {
	client *mapssideecar.Client
}

func (t *geocodeRouteTools) sidecar() *mapssideecar.Client {
	if t.client != nil {
		return t.client
	}
	return mapssideecar.DefaultSidecarClient
}

func (t *geocodeRouteTools) register(mcpServer *server.MCPServer) {
	mcpServer.AddTool(mcp.CreateTool(
		"maps_geocode",
		"Geocode a place name or address via Nominatim (OSM). Returns lat/lon and display_name.",
		mcp.CreateObjectInputSchema(map[string]interface{}{
			"query": map[string]interface{}{"type": "string", "description": "Place name or address to geocode"},
			"limit": map[string]interface{}{"type": "integer", "description": "Max results (default 5, max 10)"},
		}, []string{"query"}),
		nil,
	), t.handleGeocode)

	mcpServer.AddTool(mcp.CreateTool(
		"maps_route",
		"Compute a walking or driving route between waypoints via OSRM. Returns distance, duration, and GeoJSON geometry.",
		mcp.CreateObjectInputSchema(map[string]interface{}{
			"mode": map[string]interface{}{"type": "string", "description": "walking or driving (default walking)"},
			"waypoints": map[string]interface{}{
				"type":        "array",
				"description": "Ordered waypoints as {lat, lon} objects (min 2)",
				"items":       map[string]interface{}{"type": "object"},
			},
		}, []string{"waypoints"}),
		nil,
	), t.handleRoute)
}

func (t *geocodeRouteTools) handleGeocode(ctx context.Context, request mcpgo.CallToolRequest) (*mcpgo.CallToolResult, error) {
	args := mcp.GetArgumentsAsMap(request)
	query, _ := args["query"].(string)
	query = strings.TrimSpace(query)
	if query == "" {
		return mcp.HandleToolError(fmt.Errorf("query is required"), "maps_geocode"), nil
	}
	client := t.sidecar()
	if client == nil {
		return mcp.HandleToolError(fmt.Errorf("maps sidecar client not configured"), "maps_geocode"), nil
	}
	out, err := client.Geocode(ctx, args)
	if err != nil {
		return mcp.HandleToolError(err, "maps_geocode"), nil
	}
	raw, err := json.MarshalIndent(out, "", "  ")
	if err != nil {
		return mcp.HandleToolError(err, "maps_geocode"), nil
	}
	return mcp.HandleToolSuccess(string(raw)), nil
}

func (t *geocodeRouteTools) handleRoute(ctx context.Context, request mcpgo.CallToolRequest) (*mcpgo.CallToolResult, error) {
	args := mcp.GetArgumentsAsMap(request)
	waypoints := asObjectSlice(args["waypoints"])
	if len(waypoints) < 2 {
		return mcp.HandleToolError(fmt.Errorf("waypoints must include at least 2 points"), "maps_route"), nil
	}
	client := t.sidecar()
	if client == nil {
		return mcp.HandleToolError(fmt.Errorf("maps sidecar client not configured"), "maps_route"), nil
	}
	out, err := client.Route(ctx, args)
	if err != nil {
		return mcp.HandleToolError(err, "maps_route"), nil
	}
	raw, err := json.MarshalIndent(out, "", "  ")
	if err != nil {
		return mcp.HandleToolError(err, "maps_route"), nil
	}
	return mcp.HandleToolSuccess(string(raw)), nil
}
