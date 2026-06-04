package cad

import (
	"context"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"strings"
	"time"

	cadlib "github.com/camronwood/neural-junkie/internal/cad"
	mcp "github.com/camronwood/neural-junkie/internal/mcp"
	mcpgo "github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server"
)

// CADMCP provides MCP tools for OpenSCAD CAD workflows.
type CADMCP struct {
	mcpServer  *server.MCPServer
	httpServer *server.StreamableHTTPServer
	config     *mcp.MCPServerConfig
}

// NewCADMCP creates a new CAD MCP server.
func NewCADMCP() (*CADMCP, error) {
	config := mcp.GetMCPServerConfig("CAD")

	mcpServer, httpServer, err := mcp.NewMCPServer(config)
	if err != nil {
		return nil, fmt.Errorf("failed to create MCP server: %w", err)
	}

	c := &CADMCP{
		mcpServer:  mcpServer,
		httpServer: httpServer,
		config:     config,
	}

	c.registerTools()
	return c, nil
}

// Start starts the CAD MCP server.
func (c *CADMCP) Start() error {
	if c.httpServer == nil {
		return fmt.Errorf("MCP server not configured")
	}
	return mcp.StartMCPServer(c.httpServer, c.config.Port)
}

// GetMCPServer returns the underlying MCP server.
func (c *CADMCP) GetMCPServer() *server.MCPServer {
	return c.mcpServer
}

func (c *CADMCP) registerTools() {
	c.mcpServer.AddTool(mcp.CreateTool(
		"write_openscad",
		"Write or update an OpenSCAD (.scad) file. Provide path (workspace or artifacts) and content. Returns paths for the CAD workbench.",
		mcp.CreateObjectInputSchema(map[string]interface{}{
			"path": map[string]interface{}{
				"type":        "string",
				"description": "Absolute or workspace path ending in .scad, or project id under artifacts",
			},
			"content": map[string]interface{}{
				"type":        "string",
				"description": "OpenSCAD source code",
			},
			"project_id": map[string]interface{}{
				"type":        "string",
				"description": "Optional artifacts project id (uses ~/.neural-junkie/cad/<id>/model.scad)",
			},
		}, []string{"content"}),
		nil,
	), c.handleWriteOpenSCAD)

	c.mcpServer.AddTool(mcp.CreateTool(
		"render_openscad",
		"Render an OpenSCAD file to STL using the local OpenSCAD CLI. Optional params map applies -D overrides.",
		mcp.CreateObjectInputSchema(map[string]interface{}{
			"path": map[string]interface{}{
				"type":        "string",
				"description": "Path to .scad file",
			},
			"params": map[string]interface{}{
				"type":        "object",
				"description": "Optional parameter overrides (name -> value string)",
			},
			"output_path": map[string]interface{}{
				"type":        "string",
				"description": "Optional STL output path (default: preview.stl beside scad or project dir)",
			},
		}, []string{"path"}),
		nil,
	), c.handleRenderOpenSCAD)

	c.mcpServer.AddTool(mcp.CreateTool(
		"list_openscad_params",
		"Parse OpenSCAD Customizer-style top-level variables from a .scad file.",
		mcp.CreateStringInputSchema("path", "Path to .scad file"),
		nil,
	), c.handleListOpenSCADParams)

	c.mcpServer.AddTool(mcp.CreateTool(
		"export_cad",
		"Copy SCAD and/or STL artifacts to a destination path in the workspace.",
		mcp.CreateObjectInputSchema(map[string]interface{}{
			"source_scad": map[string]interface{}{"type": "string"},
			"source_stl":  map[string]interface{}{"type": "string"},
			"dest_dir":    map[string]interface{}{"type": "string", "description": "Destination directory in workspace"},
		}, []string{"dest_dir"}),
		nil,
	), c.handleExportCAD)

	log.Printf("Registered %d CAD MCP tools", len(c.mcpServer.ListTools()))
}

func (c *CADMCP) handleWriteOpenSCAD(ctx context.Context, request mcpgo.CallToolRequest) (*mcpgo.CallToolResult, error) {
	content := request.GetString("content", "")
	if strings.TrimSpace(content) == "" {
		return mcp.HandleToolError(fmt.Errorf("content required"), "write_openscad"), nil
	}
	path, projectID, err := resolveSCADPath(request)
	if err != nil {
		return mcp.HandleToolError(err, "write_openscad"), nil
	}
	if err := cadlib.WriteSCADFile(path, content); err != nil {
		return mcp.HandleToolError(err, "write_openscad"), nil
	}
	stlPath := strings.TrimSuffix(path, filepath.Ext(path)) + ".stl"
	if strings.HasSuffix(strings.ToLower(path), ".scad") {
		dir := filepath.Dir(path)
		stlPath = filepath.Join(dir, "preview.stl")
	}
	return mcp.HandleToolSuccess(fmt.Sprintf("Wrote OpenSCAD to %s\nOpen the CAD workbench on this file. Preview STL target: %s\nProject: %s",
		path, stlPath, projectID)), nil
}

func (c *CADMCP) handleRenderOpenSCAD(ctx context.Context, request mcpgo.CallToolRequest) (*mcpgo.CallToolResult, error) {
	path := strings.TrimSpace(request.GetString("path", ""))
	if path == "" {
		return mcp.HandleToolError(fmt.Errorf("path required"), "render_openscad"), nil
	}
	params := mapStringParams(request.GetArguments())
	outPath := strings.TrimSpace(request.GetString("output_path", ""))
	if outPath == "" {
		outPath = filepath.Join(filepath.Dir(path), "preview.stl")
	}
	settings := cadSettings()
	timeout := time.Duration(settings.RenderTimeoutOrDefault()) * time.Second
	if err := cadlib.RenderSCADToSTL(ctx, path, outPath, cadlib.RenderOptions{
		OpenSCADPath: settings.OpenSCADPathOrDefault(),
		Timeout:      timeout,
		Params:       params,
	}); err != nil {
		return mcp.HandleToolError(err, "render_openscad"), nil
	}
	return mcp.HandleToolSuccess(fmt.Sprintf("Rendered STL: %s\nOpen CAD workbench to preview.", outPath)), nil
}

func (c *CADMCP) handleListOpenSCADParams(ctx context.Context, request mcpgo.CallToolRequest) (*mcpgo.CallToolResult, error) {
	if err := mcp.ValidateToolInput(request, []string{"path"}); err != nil {
		return mcp.HandleToolError(err, "list_openscad_params"), nil
	}
	path := request.GetString("path", "")
	content, err := cadlib.ReadSCADFile(path)
	if err != nil {
		return mcp.HandleToolError(err, "list_openscad_params"), nil
	}
	params := cadlib.ParseParams(content)
	var b strings.Builder
	for _, p := range params {
		fmt.Fprintf(&b, "- %s = %s", p.Name, p.Value)
		if p.Section != "" {
			fmt.Fprintf(&b, " [%s]", p.Section)
		}
		if p.Comment != "" {
			fmt.Fprintf(&b, " // %s", p.Comment)
		}
		b.WriteString("\n")
	}
	if b.Len() == 0 {
		b.WriteString("(no top-level parameters found)\n")
	}
	return mcp.HandleToolSuccess(b.String()), nil
}

func (c *CADMCP) handleExportCAD(ctx context.Context, request mcpgo.CallToolRequest) (*mcpgo.CallToolResult, error) {
	dest := strings.TrimSpace(request.GetString("dest_dir", ""))
	if dest == "" {
		return mcp.HandleToolError(fmt.Errorf("dest_dir required"), "export_cad"), nil
	}
	if err := os.MkdirAll(dest, 0755); err != nil {
		return mcp.HandleToolError(err, "export_cad"), nil
	}
	var copied []string
	if src := strings.TrimSpace(request.GetString("source_scad", "")); src != "" {
		dst := filepath.Join(dest, filepath.Base(src))
		if err := cadlib.CopyFile(src, dst); err != nil {
			return mcp.HandleToolError(err, "export_cad"), nil
		}
		copied = append(copied, dst)
	}
	if src := strings.TrimSpace(request.GetString("source_stl", "")); src != "" {
		dst := filepath.Join(dest, filepath.Base(src))
		if err := cadlib.CopyFile(src, dst); err != nil {
			return mcp.HandleToolError(err, "export_cad"), nil
		}
		copied = append(copied, dst)
	}
	if len(copied) == 0 {
		return mcp.HandleToolError(fmt.Errorf("provide source_scad and/or source_stl"), "export_cad"), nil
	}
	return mcp.HandleToolSuccess(fmt.Sprintf("Exported:\n%s", strings.Join(copied, "\n"))), nil
}

func resolveSCADPath(request mcpgo.CallToolRequest) (path, projectID string, err error) {
	path = strings.TrimSpace(request.GetString("path", ""))
	projectID = strings.TrimSpace(request.GetString("project_id", ""))
	if path != "" {
		return path, projectID, nil
	}
	if projectID == "" {
		projectID = "default"
	}
	settings := cadSettings()
	paths, err := cadlib.ProjectDir(settings.ArtifactsDirOrDefault(), projectID)
	if err != nil {
		return "", "", err
	}
	return paths.SCADPath, projectID, nil
}

func mapStringParams(args map[string]interface{}) map[string]string {
	raw, _ := args["params"].(map[string]interface{})
	if raw == nil {
		return nil
	}
	out := make(map[string]string, len(raw))
	for k, v := range raw {
		out[k] = fmt.Sprint(v)
	}
	return out
}
