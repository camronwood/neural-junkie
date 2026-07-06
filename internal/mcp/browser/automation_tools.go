package browser

import (
	"context"
	"fmt"
	"strings"

	packsbrowser "github.com/camronwood/neural-junkie/internal/browser"
	mcp "github.com/camronwood/neural-junkie/internal/mcp"
	mcpgo "github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server"
)

// AttachAutomationTools registers Playwright sidecar MCP tools on the browser MCP server.
func AttachAutomationTools(mcpServer *server.MCPServer) {
	if mcpServer == nil {
		return
	}
	t := &automationTools{}
	t.register(mcpServer)
}

type automationTools struct{}

func (t *automationTools) register(mcpServer *server.MCPServer) {
	mcpServer.AddTool(mcp.CreateTool(
		"browser_screenshot",
		"Capture a PNG screenshot of a URL via the Playwright sidecar (localhost, workspace preview, or dev-server URLs). Returns dimensions and a short summary; image is stored for workbench use.",
		mcp.CreateMultiStringInputSchema(map[string]string{
			"url":        "URL to capture (http://localhost, hub workspace-preview URL, or file:// path)",
			"width":      "Optional viewport width (default 1280)",
			"height":     "Optional viewport height (default 800)",
			"full_page":  "Optional: true for full-page capture",
		}),
		nil,
	), t.handleScreenshot)

	mcpServer.AddTool(mcp.CreateTool(
		"browser_navigate",
		"Open a URL in Playwright and return a session_id for follow-up click/fill actions.",
		mcp.CreateMultiStringInputSchema(map[string]string{
			"url":    "URL to open",
			"width":  "Optional viewport width",
			"height": "Optional viewport height",
		}),
		nil,
	), t.handleNavigate)

	mcpServer.AddTool(mcp.CreateTool(
		"browser_click",
		"Click an element in an existing Playwright browser session.",
		mcp.CreateMultiStringInputSchema(map[string]string{
			"session_id": "Session id from browser_navigate",
			"selector":   "CSS selector to click",
		}),
		nil,
	), t.handleClick)

	mcpServer.AddTool(mcp.CreateTool(
		"browser_fill",
		"Fill a form field in an existing Playwright browser session.",
		mcp.CreateMultiStringInputSchema(map[string]string{
			"session_id": "Session id from browser_navigate",
			"selector":   "CSS selector for the input",
			"value":      "Text value to fill",
		}),
		nil,
	), t.handleFill)

	mcpServer.AddTool(mcp.CreateTool(
		"browser_a11y_audit",
		"Run an axe-core accessibility audit on a URL. Returns WCAG violations summary.",
		mcp.CreateMultiStringInputSchema(map[string]string{
			"url": "URL to audit",
		}),
		nil,
	), t.handleA11yAudit)

	mcpServer.AddTool(mcp.CreateTool(
		"browser_metrics",
		"Collect Lighthouse-lite performance metrics (FCP, load time, DOM size) for a URL.",
		mcp.CreateMultiStringInputSchema(map[string]string{
			"url": "URL to measure",
		}),
		nil,
	), t.handleMetrics)
}

func (t *automationTools) client() *packsbrowser.Client {
	return packsbrowser.DefaultSidecarClient
}

func (t *automationTools) viewportFromRequest(request mcpgo.CallToolRequest) map[string]any {
	w := strings.TrimSpace(request.GetString("width", ""))
	h := strings.TrimSpace(request.GetString("height", ""))
	if w == "" && h == "" {
		return nil
	}
	width, height := 1280, 800
	if w != "" {
		fmt.Sscanf(w, "%d", &width)
	}
	if h != "" {
		fmt.Sscanf(h, "%d", &height)
	}
	return map[string]any{"width": width, "height": height}
}

func (t *automationTools) handleScreenshot(ctx context.Context, request mcpgo.CallToolRequest) (*mcpgo.CallToolResult, error) {
	url := strings.TrimSpace(request.GetString("url", ""))
	if url == "" {
		return mcp.HandleToolError(fmt.Errorf("missing required parameter: url"), "browser_screenshot"), nil
	}
	req := map[string]any{"url": url}
	if vp := t.viewportFromRequest(request); vp != nil {
		req["viewport"] = vp
	}
	if strings.EqualFold(strings.TrimSpace(request.GetString("full_page", "")), "true") {
		req["full_page"] = true
	}
	out, err := t.client().Screenshot(ctx, req)
	if err != nil {
		return mcp.HandleToolError(err, "browser_screenshot"), nil
	}
	width, _ := out["width"].(float64)
	height, _ := out["height"].(float64)
	pageURL, _ := out["url"].(string)
	b64, _ := out["png_b64"].(string)
	summary := fmt.Sprintf("Screenshot captured (%dx%d) of %s (%d bytes PNG b64).", int(width), int(height), pageURL, len(b64))
	return mcp.HandleToolSuccess(summary), nil
}

func (t *automationTools) handleNavigate(ctx context.Context, request mcpgo.CallToolRequest) (*mcpgo.CallToolResult, error) {
	url := strings.TrimSpace(request.GetString("url", ""))
	if url == "" {
		return mcp.HandleToolError(fmt.Errorf("missing required parameter: url"), "browser_navigate"), nil
	}
	req := map[string]any{"url": url}
	if vp := t.viewportFromRequest(request); vp != nil {
		req["viewport"] = vp
	}
	out, err := t.client().Navigate(ctx, req)
	if err != nil {
		return mcp.HandleToolError(err, "browser_navigate"), nil
	}
	sessionID, _ := out["session_id"].(string)
	pageURL, _ := out["url"].(string)
	return mcp.HandleToolSuccess(fmt.Sprintf("Navigated to %s (session_id=%s). Use browser_click/browser_fill with this session_id.", pageURL, sessionID)), nil
}

func (t *automationTools) handleClick(ctx context.Context, request mcpgo.CallToolRequest) (*mcpgo.CallToolResult, error) {
	sessionID := strings.TrimSpace(request.GetString("session_id", ""))
	selector := strings.TrimSpace(request.GetString("selector", ""))
	if sessionID == "" || selector == "" {
		return mcp.HandleToolError(fmt.Errorf("missing required parameters: session_id, selector"), "browser_click"), nil
	}
	out, err := t.client().Click(ctx, map[string]any{"session_id": sessionID, "selector": selector})
	if err != nil {
		return mcp.HandleToolError(err, "browser_click"), nil
	}
	pageURL, _ := out["url"].(string)
	return mcp.HandleToolSuccess(fmt.Sprintf("Clicked %s — now at %s", selector, pageURL)), nil
}

func (t *automationTools) handleFill(ctx context.Context, request mcpgo.CallToolRequest) (*mcpgo.CallToolResult, error) {
	sessionID := strings.TrimSpace(request.GetString("session_id", ""))
	selector := strings.TrimSpace(request.GetString("selector", ""))
	value := request.GetString("value", "")
	if sessionID == "" || selector == "" {
		return mcp.HandleToolError(fmt.Errorf("missing required parameters: session_id, selector"), "browser_fill"), nil
	}
	_, err := t.client().Fill(ctx, map[string]any{"session_id": sessionID, "selector": selector, "value": value})
	if err != nil {
		return mcp.HandleToolError(err, "browser_fill"), nil
	}
	return mcp.HandleToolSuccess(fmt.Sprintf("Filled %s", selector)), nil
}

func (t *automationTools) handleA11yAudit(ctx context.Context, request mcpgo.CallToolRequest) (*mcpgo.CallToolResult, error) {
	url := strings.TrimSpace(request.GetString("url", ""))
	if url == "" {
		return mcp.HandleToolError(fmt.Errorf("missing required parameter: url"), "browser_a11y_audit"), nil
	}
	out, err := t.client().A11yAudit(ctx, map[string]any{"url": url})
	if err != nil {
		return mcp.HandleToolError(err, "browser_a11y_audit"), nil
	}
	count, _ := out["violation_count"].(float64)
	violations, _ := out["violations"].([]any)
	var b strings.Builder
	fmt.Fprintf(&b, "Accessibility audit: %d violation(s)\n", int(count))
	for i, raw := range violations {
		if i >= 10 {
			fmt.Fprintf(&b, "… and %d more\n", len(violations)-10)
			break
		}
		v, _ := raw.(map[string]any)
		fmt.Fprintf(&b, "- [%s] %s: %s\n", v["impact"], v["id"], v["help"])
	}
	return mcp.HandleToolSuccess(strings.TrimSpace(b.String())), nil
}

func (t *automationTools) handleMetrics(ctx context.Context, request mcpgo.CallToolRequest) (*mcpgo.CallToolResult, error) {
	url := strings.TrimSpace(request.GetString("url", ""))
	if url == "" {
		return mcp.HandleToolError(fmt.Errorf("missing required parameter: url"), "browser_metrics"), nil
	}
	out, err := t.client().Metrics(ctx, map[string]any{"url": url})
	if err != nil {
		return mcp.HandleToolError(err, "browser_metrics"), nil
	}
	metrics, _ := out["metrics"].(map[string]any)
	var b strings.Builder
	fmt.Fprintf(&b, "Performance metrics for %s:\n", out["url"])
	for _, k := range []string{"fcp_ms", "load_ms", "dom_content_loaded_ms", "dom_nodes", "resource_count", "transfer_size"} {
		if v, ok := metrics[k]; ok {
			fmt.Fprintf(&b, "- %s: %v\n", k, v)
		}
	}
	return mcp.HandleToolSuccess(strings.TrimSpace(b.String())), nil
}
