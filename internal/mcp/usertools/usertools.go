// Package usertools implements the MCP Tool Wizard's HTTP-fetch tool
// template: user-defined tools (config.UserMCPTool) that call a public
// HTTP(S) endpoint and are granted to chosen custom expert agents by name.
//
// See docs/MCP_INTEGRATION.md#future-enhancements and
// docs/FUTURE_ENHANCEMENTS.md#mcp-tool-wizard-user-defined-tools--agents.
package usertools

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/camronwood/neural-junkie/internal/config"
	mcp "github.com/camronwood/neural-junkie/internal/mcp"
	"github.com/camronwood/neural-junkie/internal/mcp/web"
	mcpgo "github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server"
)

const maxFetchBytes = 128 * 1024
const maxResultChars = 8000

// UserToolsMCP is an in-process MCP server exposing one custom expert's
// granted user-defined HTTP-fetch tools.
type UserToolsMCP struct {
	mcpServer *server.MCPServer
	http      *http.Client
	toolCount int
}

// NewForAgent builds an in-process MCP server with the tools granted to
// agentName (case-insensitive). Returns (nil, nil) when no tools are
// granted, so callers can skip attaching an MCP server entirely.
func NewForAgent(agentName string) (*UserToolsMCP, error) {
	cfg := mcp.AppConfig()
	if cfg == nil || len(cfg.UserToolsForAgent(agentName)) == 0 {
		return nil, nil
	}
	mcpServer, err := mcp.NewInProcessMCPServer("user-tools-mcp", "1.0.0")
	if err != nil {
		return nil, fmt.Errorf("create user-tools MCP: %w", err)
	}
	count := AttachGranted(mcpServer, agentName)
	return &UserToolsMCP{
		mcpServer: mcpServer,
		http:      &http.Client{Timeout: 30 * time.Second},
		toolCount: count,
	}, nil
}

// AttachGranted registers agentName's granted user tools directly onto an
// existing MCP server, so callers that need to combine several tool sources
// (e.g. user tools + external media) on one in-process server per agent can
// do so without each source owning its own server. Returns the number of
// tools registered (0 when none are granted).
func AttachGranted(mcpServer *server.MCPServer, agentName string) int {
	cfg := mcp.AppConfig()
	if cfg == nil {
		return 0
	}
	granted := cfg.UserToolsForAgent(agentName)
	if len(granted) == 0 {
		return 0
	}
	u := &UserToolsMCP{mcpServer: mcpServer, http: &http.Client{Timeout: 30 * time.Second}}
	for _, t := range granted {
		u.registerTool(t)
	}
	return u.toolCount
}

// GetMCPServer implements agent.MCPServerInterface.
func (u *UserToolsMCP) GetMCPServer() *server.MCPServer { return u.mcpServer }

// Start implements agent.MCPServerInterface (in-process only, no HTTP listener).
func (u *UserToolsMCP) Start() error { return nil }

// ToolCount reports how many user tools were registered.
func (u *UserToolsMCP) ToolCount() int { return u.toolCount }

var toolNameSanitizer = regexp.MustCompile(`[^a-z0-9_]+`)

// sanitizeToolName maps a user-provided tool display name to an MCP-safe
// snake_case identifier, e.g. "Read My Site" -> "user_read_my_site".
func sanitizeToolName(id, name string) string {
	slug := toolNameSanitizer.ReplaceAllString(strings.ToLower(strings.TrimSpace(name)), "_")
	slug = strings.Trim(slug, "_")
	if slug == "" {
		slug = "tool"
	}
	// Disambiguate with a short id suffix so two tools with similar names
	// don't collide inside one agent's tool catalog.
	suffix := id
	if len(suffix) > 8 {
		suffix = suffix[:8]
	}
	return fmt.Sprintf("user_%s_%s", slug, suffix)
}

func (u *UserToolsMCP) registerTool(t config.UserMCPTool) {
	name := sanitizeToolName(t.ID, t.Name)
	desc := t.Description
	if desc == "" {
		desc = fmt.Sprintf("User-defined HTTP tool: fetches %s", t.URL)
	}
	tool := t
	u.mcpServer.AddTool(mcp.CreateTool(
		name,
		desc,
		mcp.CreateObjectInputSchema(map[string]interface{}{
			"query": map[string]any{
				"type":        "string",
				"description": "Optional query string appended to the tool's configured URL (e.g. \"?id=42\" or \"/42\").",
			},
		}, nil),
		nil,
	), func(ctx context.Context, req mcpgo.CallToolRequest) (*mcpgo.CallToolResult, error) {
		query := strings.TrimSpace(req.GetString("query", ""))
		result, err := Execute(ctx, u.http, tool, query)
		if err != nil {
			return mcp.HandleToolError(err, name), nil
		}
		return mcp.HandleToolSuccess(result), nil
	})
	u.toolCount++
}

// Execute runs one user-defined HTTP-fetch tool call and returns readable
// text (optionally JSON-path-extracted). Used by both the live MCP tool
// handler and the wizard's "test this tool" API action, so both paths go
// through the same SSRF gate and truncation rules.
func Execute(ctx context.Context, client *http.Client, t config.UserMCPTool, query string) (string, error) {
	if client == nil {
		client = &http.Client{Timeout: 30 * time.Second}
	}
	target := strings.TrimSpace(t.URL)
	if target == "" {
		return "", fmt.Errorf("tool has no configured URL")
	}
	if query != "" {
		if strings.HasPrefix(query, "/") || !strings.Contains(target, "?") {
			target += query
		} else {
			target += "&" + strings.TrimPrefix(query, "?")
		}
	}
	if err := web.CheckPublicURL(target); err != nil {
		return "", err
	}

	method := t.MethodOrDefault()
	req, err := http.NewRequestWithContext(ctx, method, target, nil)
	if err != nil {
		return "", err
	}
	for k, v := range t.Headers {
		req.Header.Set(k, v)
	}
	if req.Header.Get("Accept") == "" {
		req.Header.Set("Accept", "application/json, text/plain, */*;q=0.8")
	}
	if req.Header.Get("User-Agent") == "" {
		req.Header.Set("User-Agent", "NeuralJunkie-UserTool/1.0")
	}

	resp, err := client.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()
	body, _ := io.ReadAll(io.LimitReader(resp.Body, maxFetchBytes))

	text := string(body)
	if path := strings.TrimSpace(t.JSONPath); path != "" {
		if extracted, ok := extractJSONPath(body, path); ok {
			text = extracted
		}
	}
	text = strings.TrimSpace(text)
	if len(text) > maxResultChars {
		text = text[:maxResultChars] + "\n...(truncated)"
	}
	return fmt.Sprintf("HTTP %d %s\n\n%s", resp.StatusCode, target, text), nil
}

// extractJSONPath walks dot-separated keys / numeric array indices over a
// JSON document, e.g. "data.items.0.title". Returns ok=false when the
// response isn't valid JSON or the path doesn't resolve, so callers can
// fall back to the raw body.
func extractJSONPath(body []byte, path string) (string, bool) {
	var doc interface{}
	if err := json.Unmarshal(body, &doc); err != nil {
		return "", false
	}
	cur := doc
	for _, seg := range strings.Split(path, ".") {
		seg = strings.TrimSpace(seg)
		if seg == "" {
			continue
		}
		switch v := cur.(type) {
		case map[string]interface{}:
			next, ok := v[seg]
			if !ok {
				return "", false
			}
			cur = next
		case []interface{}:
			idx, err := strconv.Atoi(seg)
			if err != nil || idx < 0 || idx >= len(v) {
				return "", false
			}
			cur = v[idx]
		default:
			return "", false
		}
	}
	switch v := cur.(type) {
	case string:
		return v, true
	default:
		out, err := json.MarshalIndent(v, "", "  ")
		if err != nil {
			return "", false
		}
		return string(out), true
	}
}
