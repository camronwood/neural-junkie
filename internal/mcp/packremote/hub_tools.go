package packremote

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"strings"
	"time"

	mcpgo "github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server"

	"github.com/camronwood/neural-junkie/internal/mcp"
	"github.com/camronwood/neural-junkie/internal/packs"
	"github.com/camronwood/neural-junkie/internal/packsidecar"
)

const defaultToolsRel = "assets/mcp/tools.json"

// HubToolSpec is one pack-owned MCP tool declared in assets/mcp/tools.json.
type HubToolSpec struct {
	Name        string         `json:"name"`
	Description string         `json:"description"`
	InputSchema map[string]any `json:"input_schema"`
}

type hubToolsFile struct {
	Version int           `json:"version"`
	Tools   []HubToolSpec `json:"tools"`
}

// LoadHubToolsCatalog reads pack-owned tool schemas from the installed pack dir.
func LoadHubToolsCatalog(packDir string) ([]HubToolSpec, error) {
	packDir = strings.TrimSpace(packDir)
	if packDir == "" {
		return nil, fmt.Errorf("empty pack dir")
	}
	rel := defaultToolsRel
	if m, err := packs.LoadManifest(packDir); err == nil && m != nil {
		if p := mcpToolsPathFromManifest(m); p != "" {
			rel = p
		}
	}
	path, err := packs.ResolvePackRelativePath(packDir, rel)
	if err != nil {
		return nil, err
	}
	raw, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	var file hubToolsFile
	if err := json.Unmarshal(raw, &file); err != nil {
		return nil, fmt.Errorf("parse %s: %w", path, err)
	}
	var out []HubToolSpec
	for _, t := range file.Tools {
		name := strings.TrimSpace(t.Name)
		if name == "" {
			continue
		}
		out = append(out, HubToolSpec{
			Name:        name,
			Description: strings.TrimSpace(t.Description),
			InputSchema: t.InputSchema,
		})
	}
	if len(out) == 0 {
		return nil, fmt.Errorf("no tools in %s", path)
	}
	return out, nil
}

func mcpToolsPathFromManifest(m *packs.Manifest) string {
	if m == nil {
		return ""
	}
	for _, def := range m.CapabilityDefs {
		if def.Kind == "mcp-tools" {
			if p := strings.TrimSpace(def.MCPToolsPath); p != "" {
				return p
			}
		}
	}
	return ""
}

// PackDirResolver returns the on-disk pack directory for packID.
var PackDirResolver = func(packID string) (string, error) {
	return packs.InstalledPackDir(packID)
}

// PackBaseURLResolver returns the running hub-sidecar base URL for packID.
var PackBaseURLResolver = func(packID string) string {
	mgr := packsidecar.GlobalManager()
	if mgr == nil {
		return ""
	}
	return mgr.BaseURL(packID)
}

// AttachHubPackTools registers pack-owned tools from tools.json onto mcpServer.
// Tool calls POST to the pack hub sidecar /mcp/call. Returns true when at least one tool was attached.
func AttachHubPackTools(mcpServer *server.MCPServer, packID string) bool {
	if mcpServer == nil || strings.TrimSpace(packID) == "" {
		return false
	}
	dir, err := PackDirResolver(packID)
	if err != nil || strings.TrimSpace(dir) == "" {
		log.Printf("packremote: pack %s dir unavailable: %v", packID, err)
		return false
	}
	tools, err := LoadHubToolsCatalog(dir)
	if err != nil {
		log.Printf("packremote: pack %s tools catalog: %v", packID, err)
		return false
	}
	n := 0
	for _, tool := range tools {
		if mcpServer.GetTool(tool.Name) != nil {
			continue
		}
		schema := hubInputSchema(tool.InputSchema)
		mcpServer.AddTool(mcp.CreateTool(tool.Name, tool.Description, schema, nil), hubCallHandler(packID, tool.Name))
		n++
	}
	if n > 0 {
		log.Printf("packremote: attached %d hub tools from pack %s", n, packID)
	}
	return n > 0
}

func hubInputSchema(raw map[string]any) mcpgo.ToolInputSchema {
	if raw == nil {
		return mcp.CreateEmptyInputSchema()
	}
	schema := mcpgo.ToolInputSchema{Type: "object", Properties: map[string]any{}}
	if t, ok := raw["type"].(string); ok && strings.TrimSpace(t) != "" {
		schema.Type = t
	}
	if props, ok := raw["properties"].(map[string]any); ok {
		schema.Properties = props
	}
	switch req := raw["required"].(type) {
	case []any:
		for _, v := range req {
			if s, ok := v.(string); ok && strings.TrimSpace(s) != "" {
				schema.Required = append(schema.Required, s)
			}
		}
	case []string:
		schema.Required = append(schema.Required, req...)
	}
	return schema
}

func hubCallHandler(packID, toolName string) func(context.Context, mcpgo.CallToolRequest) (*mcpgo.CallToolResult, error) {
	return func(ctx context.Context, req mcpgo.CallToolRequest) (*mcpgo.CallToolResult, error) {
		base := strings.TrimRight(strings.TrimSpace(PackBaseURLResolver(packID)), "/")
		if base == "" {
			return mcp.HandleToolError(fmt.Errorf("%s pack sidecar not running", packID), toolName), nil
		}
		args := req.GetArguments()
		if args == nil {
			args = map[string]any{}
		}
		body, _ := json.Marshal(map[string]any{
			"name":      toolName,
			"arguments": args,
		})
		httpReq, err := http.NewRequestWithContext(ctx, http.MethodPost, base+"/mcp/call", bytes.NewReader(body))
		if err != nil {
			return mcp.HandleToolError(err, toolName), nil
		}
		httpReq.Header.Set("Content-Type", "application/json")
		client := &http.Client{Timeout: 120 * time.Second}
		resp, err := client.Do(httpReq)
		if err != nil {
			return mcp.HandleToolError(err, toolName), nil
		}
		defer resp.Body.Close()
		raw, err := io.ReadAll(resp.Body)
		if err != nil {
			return mcp.HandleToolError(err, toolName), nil
		}
		var out struct {
			OK    bool   `json:"ok"`
			Text  string `json:"text"`
			Error string `json:"error"`
		}
		if err := json.Unmarshal(raw, &out); err != nil {
			msg := strings.TrimSpace(string(raw))
			if msg == "" {
				msg = resp.Status
			}
			return mcp.HandleToolError(fmt.Errorf("%s", msg), toolName), nil
		}
		if resp.StatusCode >= 400 || !out.OK {
			errMsg := strings.TrimSpace(out.Error)
			if errMsg == "" {
				errMsg = strings.TrimSpace(out.Text)
			}
			if errMsg == "" {
				errMsg = resp.Status
			}
			return mcp.HandleToolError(fmt.Errorf("%s", errMsg), toolName), nil
		}
		text := strings.TrimSpace(out.Text)
		if text == "" {
			text = strings.TrimSpace(string(raw))
		}
		return mcp.HandleToolSuccess(text), nil
	}
}

// HubPackMCP is an in-process MCP server backed by a pack hub sidecar tools catalog.
type HubPackMCP struct {
	packID    string
	mcpServer *server.MCPServer
}

// NewHubPackMCP builds an MCP host that registers tools from the pack tools.json catalog.
func NewHubPackMCP(packID, label string) (*HubPackMCP, error) {
	packID = strings.TrimSpace(packID)
	if packID == "" {
		return nil, fmt.Errorf("empty pack id")
	}
	name := fmt.Sprintf("%s-hub-mcp", strings.ToLower(strings.ReplaceAll(strings.TrimSpace(label), " ", "-")))
	if name == "-hub-mcp" {
		name = packID + "-hub-mcp"
	}
	srv, err := mcp.NewInProcessMCPServer(name, "1.0.0")
	if err != nil {
		return nil, err
	}
	h := &HubPackMCP{packID: packID, mcpServer: srv}
	if !AttachHubPackTools(srv, packID) {
		return nil, fmt.Errorf("no hub tools attached for pack %s", packID)
	}
	return h, nil
}

// Start is a no-op; tools are in-process and call the hub sidecar on demand.
func (h *HubPackMCP) Start() error { return nil }

// GetMCPServer returns the in-process MCP server.
func (h *HubPackMCP) GetMCPServer() *server.MCPServer {
	if h == nil {
		return nil
	}
	return h.mcpServer
}

// HubPackSidecarHealthy reports whether the pack hub sidecar answers /health.
func HubPackSidecarHealthy(packID string) bool {
	base := strings.TrimRight(strings.TrimSpace(PackBaseURLResolver(packID)), "/")
	if base == "" {
		return false
	}
	return SidecarHealthOK(base)
}
