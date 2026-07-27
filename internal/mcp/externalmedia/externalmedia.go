// Package externalmedia implements an optional, off-by-default MCP tool set
// (media_submit / media_status / media_fetch) that lets a granted custom
// expert agent submit jobs to a third-party media-generation HTTP API (e.g.
// image/video/audio) and poll for results.
//
// The feature is disabled unless an operator sets
// config.MCPConfig.ExternalMedia.BaseURL — see docs/MCP_INTEGRATION.md and
// docs/FUTURE_ENHANCEMENTS.md. With BaseURL empty (the default), NewForAgent
// returns (nil, nil) so no tools are attached at all.
package externalmedia

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	mcp "github.com/camronwood/neural-junkie/internal/mcp"
	"github.com/camronwood/neural-junkie/internal/mcp/web"
	mcpgo "github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server"
)

const maxFetchBytes = 256 * 1024

// ExternalMediaMCP is an in-process MCP server exposing media_submit,
// media_status, and media_fetch for one granted custom expert agent.
type ExternalMediaMCP struct {
	mcpServer *server.MCPServer
	http      *http.Client
	baseURL   string
	apiKey    string
}

// NewForAgent builds an in-process MCP server wired to the configured
// external media API, if enabled and granted to agentName. Returns
// (nil, nil) when the feature is disabled (empty BaseURL) or agentName has
// no grant, so callers can skip attaching an MCP server entirely.
func NewForAgent(agentName string) (*ExternalMediaMCP, error) {
	cfg := mcp.AppConfig()
	if cfg == nil {
		return nil, nil
	}
	settings := cfg.ExternalMediaSettings()
	if !settings.Enabled() || !settings.GrantedTo(agentName) {
		return nil, nil
	}
	mcpServer, err := mcp.NewInProcessMCPServer("external-media-mcp", "1.0.0")
	if err != nil {
		return nil, fmt.Errorf("create external-media MCP: %w", err)
	}
	return AttachGranted(mcpServer, agentName)
}

// AttachGranted registers media_submit/media_status/media_fetch directly
// onto an existing MCP server when the feature is enabled and granted to
// agentName, so callers combining multiple tool sources on one in-process
// server per agent can do so without each source owning its own server.
// Returns (nil, nil) when disabled or ungranted.
func AttachGranted(mcpServer *server.MCPServer, agentName string) (*ExternalMediaMCP, error) {
	cfg := mcp.AppConfig()
	if cfg == nil {
		return nil, nil
	}
	settings := cfg.ExternalMediaSettings()
	if !settings.Enabled() || !settings.GrantedTo(agentName) {
		return nil, nil
	}
	m := &ExternalMediaMCP{
		mcpServer: mcpServer,
		http:      &http.Client{Timeout: 30 * time.Second},
		baseURL:   strings.TrimRight(strings.TrimSpace(settings.BaseURL), "/"),
		apiKey:    settings.APIKey,
	}
	m.registerTools()
	return m, nil
}

// GetMCPServer implements agent.MCPServerInterface.
func (m *ExternalMediaMCP) GetMCPServer() *server.MCPServer { return m.mcpServer }

// Start implements agent.MCPServerInterface (in-process only, no HTTP listener).
func (m *ExternalMediaMCP) Start() error { return nil }

func (m *ExternalMediaMCP) registerTools() {
	m.mcpServer.AddTool(mcp.CreateTool(
		"media_submit",
		"Submit a media generation job (image/video/audio) to the configured external media API. Returns a job_id to poll with media_status.",
		mcp.CreateObjectInputSchema(map[string]interface{}{
			"kind": map[string]any{
				"type":        "string",
				"description": "Media kind, e.g. \"image\", \"video\", or \"audio\".",
			},
			"prompt": map[string]any{
				"type":        "string",
				"description": "Generation prompt/description.",
			},
			"params_json": map[string]any{
				"type":        "string",
				"description": "Optional extra JSON object of provider-specific parameters.",
			},
		}, []string{"kind", "prompt"}),
		nil,
	), func(ctx context.Context, req mcpgo.CallToolRequest) (*mcpgo.CallToolResult, error) {
		kind := strings.TrimSpace(req.GetString("kind", ""))
		prompt := strings.TrimSpace(req.GetString("prompt", ""))
		paramsJSON := strings.TrimSpace(req.GetString("params_json", ""))
		result, err := m.Submit(ctx, kind, prompt, paramsJSON)
		if err != nil {
			return mcp.HandleToolError(err, "media_submit"), nil
		}
		return mcp.HandleToolSuccess(result), nil
	})

	m.mcpServer.AddTool(mcp.CreateTool(
		"media_status",
		"Check the status of a previously submitted media generation job.",
		mcp.CreateObjectInputSchema(map[string]interface{}{
			"job_id": map[string]any{
				"type":        "string",
				"description": "Job ID returned by media_submit.",
			},
		}, []string{"job_id"}),
		nil,
	), func(ctx context.Context, req mcpgo.CallToolRequest) (*mcpgo.CallToolResult, error) {
		jobID := strings.TrimSpace(req.GetString("job_id", ""))
		result, err := m.Status(ctx, jobID)
		if err != nil {
			return mcp.HandleToolError(err, "media_status"), nil
		}
		return mcp.HandleToolSuccess(result), nil
	})

	m.mcpServer.AddTool(mcp.CreateTool(
		"media_fetch",
		"Fetch the result (URL or content descriptor) of a completed media generation job.",
		mcp.CreateObjectInputSchema(map[string]interface{}{
			"job_id": map[string]any{
				"type":        "string",
				"description": "Job ID returned by media_submit.",
			},
		}, []string{"job_id"}),
		nil,
	), func(ctx context.Context, req mcpgo.CallToolRequest) (*mcpgo.CallToolResult, error) {
		jobID := strings.TrimSpace(req.GetString("job_id", ""))
		result, err := m.Fetch(ctx, jobID)
		if err != nil {
			return mcp.HandleToolError(err, "media_fetch"), nil
		}
		return mcp.HandleToolSuccess(result), nil
	})
}

// Submit posts a new media generation job. Exported for the wizard/test API
// so both the live MCP tool handler and any "test this" API action share
// one SSRF-gated code path.
func (m *ExternalMediaMCP) Submit(ctx context.Context, kind, prompt, paramsJSON string) (string, error) {
	if kind == "" || prompt == "" {
		return "", fmt.Errorf("kind and prompt are required")
	}
	body := map[string]interface{}{"kind": kind, "prompt": prompt}
	if paramsJSON != "" {
		var extra map[string]interface{}
		if err := json.Unmarshal([]byte(paramsJSON), &extra); err == nil {
			for k, v := range extra {
				body[k] = v
			}
		}
	}
	payload, err := json.Marshal(body)
	if err != nil {
		return "", err
	}
	return m.do(ctx, http.MethodPost, "/submit", payload)
}

// Status polls a job's status.
func (m *ExternalMediaMCP) Status(ctx context.Context, jobID string) (string, error) {
	if jobID == "" {
		return "", fmt.Errorf("job_id is required")
	}
	return m.do(ctx, http.MethodGet, "/status/"+jobID, nil)
}

// Fetch retrieves a completed job's result descriptor.
func (m *ExternalMediaMCP) Fetch(ctx context.Context, jobID string) (string, error) {
	if jobID == "" {
		return "", fmt.Errorf("job_id is required")
	}
	return m.do(ctx, http.MethodGet, "/fetch/"+jobID, nil)
}

// do resolves path against the configured (SSRF-gated) base URL and sends
// the request. Callers never pass an agent-controlled base URL — it's
// admin-configured — but the gate is kept as defense in depth.
func (m *ExternalMediaMCP) do(ctx context.Context, method, path string, body []byte) (string, error) {
	target := m.baseURL + path
	if err := web.CheckPublicURL(target); err != nil {
		return "", err
	}
	return m.send(ctx, method, target, body)
}

// send performs the actual HTTP round trip against an already-validated
// target URL. Split out from do() so tests can exercise the request/response
// handling (headers, JSON body, status formatting) against a local mock
// server without tripping the loopback-blocking SSRF gate meant for
// production traffic.
func (m *ExternalMediaMCP) send(ctx context.Context, method, target string, body []byte) (string, error) {
	var reqBody io.Reader
	if body != nil {
		reqBody = strings.NewReader(string(body))
	}
	req, err := http.NewRequestWithContext(ctx, method, target, reqBody)
	if err != nil {
		return "", err
	}
	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}
	if m.apiKey != "" {
		req.Header.Set("Authorization", "Bearer "+m.apiKey)
	}
	req.Header.Set("Accept", "application/json")

	resp, err := m.http.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()
	respBody, _ := io.ReadAll(io.LimitReader(resp.Body, maxFetchBytes))
	return fmt.Sprintf("HTTP %d %s\n\n%s", resp.StatusCode, target, strings.TrimSpace(string(respBody))), nil
}
