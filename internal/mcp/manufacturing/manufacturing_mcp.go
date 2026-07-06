package manufacturing

import (
	"context"
	"fmt"
	"log"
	"strings"

	"github.com/camronwood/neural-junkie/internal/cadcsidecar"
	mcp "github.com/camronwood/neural-junkie/internal/mcp"
	mcpgo "github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server"
)

// ManufacturingMCP provides MCP tools for print/export workflows via the CAD sidecar.
type ManufacturingMCP struct {
	mcpServer  *server.MCPServer
	httpServer *server.StreamableHTTPServer
	config     *mcp.MCPServerConfig
	client     *cadcsidecar.Client
}

// NewManufacturingMCP creates a new Manufacturing MCP server.
func NewManufacturingMCP() (*ManufacturingMCP, error) {
	cfg := mcp.GetMCPServerConfig("MANUFACTURING")
	mcpServer, httpServer, err := mcp.NewMCPServer(cfg)
	if err != nil {
		return nil, fmt.Errorf("failed to create MCP server: %w", err)
	}
	m := &ManufacturingMCP{
		mcpServer:  mcpServer,
		httpServer: httpServer,
		config:     cfg,
		client:     cadcsidecar.DefaultSidecarClient,
	}
	m.registerTools()
	return m, nil
}

func (m *ManufacturingMCP) Start() error {
	return mcp.StartMCPServer(m.httpServer, m.config.Port)
}

func (m *ManufacturingMCP) GetMCPServer() *server.MCPServer {
	return m.mcpServer
}

func (m *ManufacturingMCP) sidecar() *cadcsidecar.Client {
	if m.client != nil {
		return m.client
	}
	return cadcsidecar.DefaultSidecarClient
}

func (m *ManufacturingMCP) registerTools() {
	m.mcpServer.AddTool(mcp.CreateTool(
		"check_printability",
		"Analyze STL for FDM printability: overhangs and estimated wall thickness.",
		mcp.CreateObjectInputSchema(map[string]interface{}{
			"stl_path": map[string]interface{}{"type": "string", "description": "Path to STL file"},
			"min_wall_mm": map[string]interface{}{"type": "number", "description": "Minimum wall thickness in mm (default 1.2)"},
		}, []string{"stl_path"}),
		nil,
	), m.handleCheckPrintability)

	m.mcpServer.AddTool(mcp.CreateTool(
		"repair_mesh",
		"Repair STL mesh (non-manifold, degenerate faces) via CAD sidecar.",
		mcp.CreateStringInputSchema("stl_path", "Path to STL file"),
		nil,
	), m.handleRepairMesh)

	m.mcpServer.AddTool(mcp.CreateTool(
		"export_slicer_preset",
		"Export PrusaSlicer or Orca slicer preset JSON for an STL.",
		mcp.CreateObjectInputSchema(map[string]interface{}{
			"stl_path": map[string]interface{}{"type": "string"},
			"slicer":   map[string]interface{}{"type": "string", "description": "prusa or orca"},
			"dest_path": map[string]interface{}{"type": "string"},
		}, []string{"stl_path"}),
		nil,
	), m.handleExportSlicer)

	m.mcpServer.AddTool(mcp.CreateTool(
		"sanity_check_gcode",
		"Sanity-check G-code for layer height jumps, temperatures, and extrusion.",
		mcp.CreateObjectInputSchema(map[string]interface{}{
			"gcode": map[string]interface{}{"type": "string"},
			"path":  map[string]interface{}{"type": "string", "description": "Optional path to .gcode file"},
		}, nil),
		nil,
	), m.handleSanityGcode)

	m.mcpServer.AddTool(mcp.CreateTool(
		"export_step",
		"Export STL mesh to STEP via FreeCAD sidecar.",
		mcp.CreateObjectInputSchema(map[string]interface{}{
			"stl_path":  map[string]interface{}{"type": "string"},
			"dest_path": map[string]interface{}{"type": "string"},
		}, []string{"stl_path"}),
		nil,
	), m.handleExportSTEP)

	m.mcpServer.AddTool(mcp.CreateTool(
		"export_drawing",
		"Export dimensioned 2D drawing (PDF placeholder) from STEP/STL.",
		mcp.CreateObjectInputSchema(map[string]interface{}{
			"stl_path":  map[string]interface{}{"type": "string"},
			"dest_path": map[string]interface{}{"type": "string"},
		}, []string{"stl_path"}),
		nil,
	), m.handleExportDrawing)

	log.Printf("Registered %d Manufacturing MCP tools", len(m.mcpServer.ListTools()))
}

func (m *ManufacturingMCP) postTool(ctx context.Context, path string, body map[string]any) (string, error) {
	client := m.sidecar()
	if client == nil {
		return "", fmt.Errorf("cad sidecar client not configured")
	}
	return client.PostJSON(ctx, path, body)
}

func (m *ManufacturingMCP) handleCheckPrintability(ctx context.Context, request mcpgo.CallToolRequest) (*mcpgo.CallToolResult, error) {
	stlPath := strings.TrimSpace(request.GetString("stl_path", ""))
	if stlPath == "" {
		return mcp.HandleToolError(fmt.Errorf("stl_path required"), "check_printability"), nil
	}
	body := map[string]any{"stl_path": stlPath}
	if s := strings.TrimSpace(request.GetString("min_wall_mm", "")); s != "" {
		body["min_wall_mm"] = s
	}
	out, err := m.postTool(ctx, "/api/cad/printability", body)
	if err != nil {
		return mcp.HandleToolError(err, "check_printability"), nil
	}
	return mcp.HandleToolSuccess(out), nil
}

func (m *ManufacturingMCP) handleRepairMesh(ctx context.Context, request mcpgo.CallToolRequest) (*mcpgo.CallToolResult, error) {
	stlPath := strings.TrimSpace(request.GetString("stl_path", ""))
	if stlPath == "" {
		return mcp.HandleToolError(fmt.Errorf("stl_path required"), "repair_mesh"), nil
	}
	out, err := m.postTool(ctx, "/api/cad/geometry/repair", map[string]any{"stl_path": stlPath})
	if err != nil {
		return mcp.HandleToolError(err, "repair_mesh"), nil
	}
	return mcp.HandleToolSuccess(out), nil
}

func (m *ManufacturingMCP) handleExportSlicer(ctx context.Context, request mcpgo.CallToolRequest) (*mcpgo.CallToolResult, error) {
	stlPath := strings.TrimSpace(request.GetString("stl_path", ""))
	if stlPath == "" {
		return mcp.HandleToolError(fmt.Errorf("stl_path required"), "export_slicer_preset"), nil
	}
	body := map[string]any{"stl_path": stlPath}
	if s := strings.TrimSpace(request.GetString("slicer", "")); s != "" {
		body["slicer"] = s
	}
	if d := strings.TrimSpace(request.GetString("dest_path", "")); d != "" {
		body["dest_path"] = d
	}
	out, err := m.postTool(ctx, "/api/cad/export/slicer", body)
	if err != nil {
		return mcp.HandleToolError(err, "export_slicer_preset"), nil
	}
	return mcp.HandleToolSuccess(out), nil
}

func (m *ManufacturingMCP) handleSanityGcode(ctx context.Context, request mcpgo.CallToolRequest) (*mcpgo.CallToolResult, error) {
	body := map[string]any{}
	if g := strings.TrimSpace(request.GetString("gcode", "")); g != "" {
		body["gcode"] = g
	}
	if p := strings.TrimSpace(request.GetString("path", "")); p != "" {
		body["path"] = p
	}
	if len(body) == 0 {
		return mcp.HandleToolError(fmt.Errorf("gcode or path required"), "sanity_check_gcode"), nil
	}
	out, err := m.postTool(ctx, "/api/cad/gcode/sanity", body)
	if err != nil {
		return mcp.HandleToolError(err, "sanity_check_gcode"), nil
	}
	return mcp.HandleToolSuccess(out), nil
}

func (m *ManufacturingMCP) handleExportSTEP(ctx context.Context, request mcpgo.CallToolRequest) (*mcpgo.CallToolResult, error) {
	stlPath := strings.TrimSpace(request.GetString("stl_path", ""))
	if stlPath == "" {
		return mcp.HandleToolError(fmt.Errorf("stl_path required"), "export_step"), nil
	}
	body := map[string]any{"stl_path": stlPath}
	if d := strings.TrimSpace(request.GetString("dest_path", "")); d != "" {
		body["dest_path"] = d
	}
	out, err := m.postTool(ctx, "/api/cad/export/step", body)
	if err != nil {
		return mcp.HandleToolError(err, "export_step"), nil
	}
	return mcp.HandleToolSuccess(out), nil
}

func (m *ManufacturingMCP) handleExportDrawing(ctx context.Context, request mcpgo.CallToolRequest) (*mcpgo.CallToolResult, error) {
	stlPath := strings.TrimSpace(request.GetString("stl_path", ""))
	if stlPath == "" {
		return mcp.HandleToolError(fmt.Errorf("stl_path required"), "export_drawing"), nil
	}
	body := map[string]any{"stl_path": stlPath}
	if d := strings.TrimSpace(request.GetString("dest_path", "")); d != "" {
		body["dest_path"] = d
	}
	out, err := m.postTool(ctx, "/api/cad/export/drawing", body)
	if err != nil {
		return mcp.HandleToolError(err, "export_drawing"), nil
	}
	return mcp.HandleToolSuccess(out), nil
}
