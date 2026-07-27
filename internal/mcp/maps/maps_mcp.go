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

// RendererID is the Neural Canvas renderer for map artifacts.
const RendererID = "nj.map"

// MapsMCP provides MCP tools for geocoding, routing, and map canvas artifacts.
type MapsMCP struct {
	mcpServer  *server.MCPServer
	httpServer *server.StreamableHTTPServer
	config     *mcp.MCPServerConfig
	client     *mapssideecar.Client
}

// NewMapsMCP creates a new Maps MCP server.
func NewMapsMCP() (*MapsMCP, error) {
	cfg := mcp.GetMCPServerConfig("MAPS")
	mcpServer, httpServer, err := mcp.NewMCPServer(cfg)
	if err != nil {
		return nil, fmt.Errorf("failed to create MCP server: %w", err)
	}
	m := &MapsMCP{
		mcpServer:  mcpServer,
		httpServer: httpServer,
		config:     cfg,
		client:     mapssideecar.DefaultSidecarClient,
	}
	m.registerTools()
	return m, nil
}

// Start begins serving the Maps MCP HTTP endpoint when configured.
func (m *MapsMCP) Start() error {
	return mcp.StartMCPServer(m.httpServer, m.config.Port)
}

// GetMCPServer returns the underlying MCP server for in-process registration.
func (m *MapsMCP) GetMCPServer() *server.MCPServer {
	return m.mcpServer
}

func (m *MapsMCP) sidecar() *mapssideecar.Client {
	if m.client != nil {
		return m.client
	}
	return mapssideecar.DefaultSidecarClient
}

func (m *MapsMCP) registerTools() {
	m.mcpServer.AddTool(mcp.CreateTool(
		"maps_geocode",
		"Geocode a place name or address via Nominatim (OSM). Returns lat/lon and display_name.",
		mcp.CreateObjectInputSchema(map[string]interface{}{
			"query": map[string]interface{}{"type": "string", "description": "Place name or address to geocode"},
			"limit": map[string]interface{}{"type": "integer", "description": "Max results (default 5, max 10)"},
		}, []string{"query"}),
		nil,
	), m.handleGeocode)

	m.mcpServer.AddTool(mcp.CreateTool(
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
	), m.handleRoute)

	m.mcpServer.AddTool(mcp.CreateTool(
		"maps_create",
		"Build an interactive map artifact payload (application/vnd.neural-junkie.map+json) for Neural Canvas (nj.map). Optionally routes between waypoints.",
		mcp.CreateObjectInputSchema(map[string]interface{}{
			"title": map[string]interface{}{"type": "string"},
			"center": map[string]interface{}{
				"type":        "object",
				"description": "{lat, lon} map center",
			},
			"zoom": map[string]interface{}{"type": "number"},
			"markers": map[string]interface{}{
				"type":  "array",
				"items": map[string]interface{}{"type": "object"},
			},
			"routes": map[string]interface{}{
				"type":  "array",
				"items": map[string]interface{}{"type": "object"},
			},
			"waypoints": map[string]interface{}{
				"type":        "array",
				"description": "If routes omitted and >=2 waypoints, computes a route via OSRM",
				"items":       map[string]interface{}{"type": "object"},
			},
			"mode":              map[string]interface{}{"type": "string", "description": "walking or driving when computing route from waypoints"},
			"tile_url_template": map[string]interface{}{"type": "string"},
		}, nil),
		nil,
	), m.handleCreate)

	m.mcpServer.AddTool(mcp.CreateTool(
		"maps_update",
		"Prepare an update_artifact hint for an existing map on Neural Canvas. Pass artifact_id, expected_revision, and fields to merge (center, zoom, markers, routes, waypoints, title).",
		mcp.CreateObjectInputSchema(map[string]interface{}{
			"artifact_id":       map[string]interface{}{"type": "string"},
			"expected_revision": map[string]interface{}{"type": "integer"},
			"title":             map[string]interface{}{"type": "string"},
			"center":            map[string]interface{}{"type": "object"},
			"zoom":              map[string]interface{}{"type": "number"},
			"markers": map[string]interface{}{
				"type":  "array",
				"items": map[string]interface{}{"type": "object"},
			},
			"routes": map[string]interface{}{
				"type":  "array",
				"items": map[string]interface{}{"type": "object"},
			},
			"waypoints": map[string]interface{}{
				"type":  "array",
				"items": map[string]interface{}{"type": "object"},
			},
			"mode":              map[string]interface{}{"type": "string"},
			"tile_url_template": map[string]interface{}{"type": "string"},
			"hint":              map[string]interface{}{"type": "string"},
		}, []string{"artifact_id", "expected_revision"}),
		nil,
	), m.handleUpdate)

	log.Printf("Registered %d Maps MCP tools", len(m.mcpServer.ListTools()))
}

func (m *MapsMCP) handleGeocode(ctx context.Context, request mcpgo.CallToolRequest) (*mcpgo.CallToolResult, error) {
	args := mcp.GetArgumentsAsMap(request)
	query, _ := args["query"].(string)
	query = strings.TrimSpace(query)
	if query == "" {
		return mcp.HandleToolError(fmt.Errorf("query is required"), "maps_geocode"), nil
	}
	client := m.sidecar()
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

func (m *MapsMCP) handleRoute(ctx context.Context, request mcpgo.CallToolRequest) (*mcpgo.CallToolResult, error) {
	args := mcp.GetArgumentsAsMap(request)
	waypoints := asObjectSlice(args["waypoints"])
	if len(waypoints) < 2 {
		return mcp.HandleToolError(fmt.Errorf("waypoints must include at least 2 points"), "maps_route"), nil
	}
	client := m.sidecar()
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

func (m *MapsMCP) handleCreate(ctx context.Context, request mcpgo.CallToolRequest) (*mcpgo.CallToolResult, error) {
	args := mcp.GetArgumentsAsMap(request)
	payload, err := BuildMapPayload(ctx, m.sidecar(), args)
	if err != nil {
		return mcp.HandleToolError(err, "maps_create"), nil
	}
	title, _ := payload["title"].(string)
	out := map[string]any{
		"title":       title,
		"renderer_id": RendererID,
		"media_type":  MediaType,
		"data":        payload,
		"hint":        "Pass these fields to create_artifact to open the map on Neural Canvas.",
	}
	raw, err := json.MarshalIndent(out, "", "  ")
	if err != nil {
		return mcp.HandleToolError(err, "maps_create"), nil
	}
	return mcp.HandleToolSuccess(string(raw)), nil
}

func (m *MapsMCP) handleUpdate(ctx context.Context, request mcpgo.CallToolRequest) (*mcpgo.CallToolResult, error) {
	args := mcp.GetArgumentsAsMap(request)
	artifactID, _ := args["artifact_id"].(string)
	artifactID = strings.TrimSpace(artifactID)
	if artifactID == "" {
		return mcp.HandleToolError(fmt.Errorf("artifact_id is required"), "maps_update"), nil
	}
	rev, ok := asFloat(args["expected_revision"])
	if !ok || rev < 1 {
		return mcp.HandleToolError(fmt.Errorf("expected_revision is required (integer >= 1)"), "maps_update"), nil
	}

	mergeArgs := StripMetaArgs(args)
	payload, err := BuildMapPayload(ctx, m.sidecar(), mergeArgs)
	if err != nil {
		return mcp.HandleToolError(err, "maps_update"), nil
	}
	title, _ := payload["title"].(string)
	out := map[string]any{
		"action":            "update_artifact",
		"artifact_id":       artifactID,
		"expected_revision": int(rev),
		"title":             title,
		"renderer_id":       RendererID,
		"media_type":        MediaType,
		"data":              payload,
		"hint":              "Call update_artifact with artifact_id, expected_revision, and data from this response.",
	}
	raw, err := json.MarshalIndent(out, "", "  ")
	if err != nil {
		return mcp.HandleToolError(err, "maps_update"), nil
	}
	return mcp.HandleToolSuccess(string(raw)), nil
}
