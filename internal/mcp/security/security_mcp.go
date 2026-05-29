package security

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"os/exec"
	"strings"
	"time"

	mcp "github.com/camronwood/neural-junkie/internal/mcp"
	"github.com/camronwood/neural-junkie/internal/mcp/shared"
	mcpgo "github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server"
)

// SecurityMCP provides MCP tools for security analysis.
type SecurityMCP struct {
	mcpServer  *server.MCPServer
	httpServer *server.StreamableHTTPServer
	config     *mcp.MCPServerConfig
}

// NewSecurityMCP creates a new Security MCP server.
func NewSecurityMCP() (*SecurityMCP, error) {
	config := mcp.GetMCPServerConfig("security")
	mcpServer, httpServer, err := mcp.NewMCPServer(config)
	if err != nil {
		return nil, fmt.Errorf("failed to create MCP server: %w", err)
	}
	s := &SecurityMCP{mcpServer: mcpServer, httpServer: httpServer, config: config}
	s.registerTools()
	return s, nil
}

func (s *SecurityMCP) Start() error {
	if s.httpServer == nil {
		return fmt.Errorf("MCP server not configured")
	}
	return mcp.StartMCPServer(s.httpServer, s.config.Port)
}

func (s *SecurityMCP) GetMCPServer() *server.MCPServer {
	return s.mcpServer
}

func (s *SecurityMCP) registerTools() {
	s.mcpServer.AddTool(mcp.CreateTool(
		"run_gosec",
		"Run gosec static security analysis on Go code",
		mcp.CreateStringInputSchema("module_path", "Path to Go module directory"),
		nil,
	), s.handleRunGosec)

	s.mcpServer.AddTool(mcp.CreateTool(
		"run_npm_audit",
		"Run npm audit for Node.js project vulnerabilities",
		mcp.CreateStringInputSchema("project_path", "Path to directory with package-lock.json or npm-shrinkwrap"),
		nil,
	), s.handleRunNpmAudit)

	s.mcpServer.AddTool(mcp.CreateTool(
		"scan_secrets",
		"Scan repository for leaked secrets using gitleaks",
		mcp.CreateStringInputSchema("source_path", "Path to directory to scan"),
		nil,
	), s.handleScanSecrets)

	s.mcpServer.AddTool(mcp.CreateTool(
		"check_go_vulnerabilities",
		"Run govulncheck on Go module",
		mcp.CreateStringInputSchema("module_path", "Path to Go module directory"),
		nil,
	), s.handleCheckGoVulnerabilities)

	s.mcpServer.AddTool(mcp.CreateTool(
		"validate_security_headers",
		"Check HTTP security headers (HSTS, CSP, X-Frame-Options) for a URL",
		mcp.CreateStringInputSchema("url", "HTTP or HTTPS URL to check"),
		nil,
	), s.handleValidateSecurityHeaders)

	log.Printf("Registered %d Security MCP tools", len(s.mcpServer.ListTools()))
}

func (s *SecurityMCP) handleRunGosec(ctx context.Context, request mcpgo.CallToolRequest) (*mcpgo.CallToolResult, error) {
	modulePath := request.GetString("module_path", ".")
	root := shared.FindProjectRoot(modulePath, "go.mod")
	out, err := shared.RunCommand(ctx, root, "gosec", "-fmt=text", "./...")
	if err != nil && !commandExists("gosec") {
		return mcp.HandleToolSuccess(mcp.MissingBinaryMessage("gosec", "Install: go install github.com/securego/gosec/v2/cmd/gosec@latest")), nil
	}
	return mcp.HandleToolSuccess(shared.FormatCommandResult("gosec:", out, err)), nil
}

func (s *SecurityMCP) handleRunNpmAudit(ctx context.Context, request mcpgo.CallToolRequest) (*mcpgo.CallToolResult, error) {
	projectPath := request.GetString("project_path", ".")
	root := shared.FindProjectRoot(projectPath, "package.json")
	out, err := shared.RunCommand(ctx, root, "npm", "audit", "--json")
	if err != nil && !commandExists("npm") {
		return mcp.HandleToolSuccess(mcp.MissingBinaryMessage("npm", "Install Node.js/npm to run npm audit.")), nil
	}
	return mcp.HandleToolSuccess(shared.FormatCommandResult("npm audit:", out, err)), nil
}

func (s *SecurityMCP) handleScanSecrets(ctx context.Context, request mcpgo.CallToolRequest) (*mcpgo.CallToolResult, error) {
	sourcePath := request.GetString("source_path", ".")
	if !shared.PathExists(sourcePath) {
		return mcp.HandleToolError(fmt.Errorf("source_path not found"), "scan_secrets"), nil
	}
	if !commandExists("gitleaks") {
		return mcp.HandleToolSuccess(mcp.MissingBinaryMessage("gitleaks", "Install from https://github.com/gitleaks/gitleaks")), nil
	}
	out, err := shared.RunCommand(ctx, "", "gitleaks", "detect", "--source", sourcePath, "--no-git")
	return mcp.HandleToolSuccess(shared.FormatCommandResult("gitleaks:", out, err)), nil
}

func (s *SecurityMCP) handleCheckGoVulnerabilities(ctx context.Context, request mcpgo.CallToolRequest) (*mcpgo.CallToolResult, error) {
	modulePath := request.GetString("module_path", ".")
	root := shared.FindProjectRoot(modulePath, "go.mod")
	out, err := shared.RunCommand(ctx, root, "govulncheck", "./...")
	if err != nil && !commandExists("govulncheck") {
		return mcp.HandleToolSuccess(mcp.MissingBinaryMessage("govulncheck", "Install: go install golang.org/x/vuln/cmd/govulncheck@latest")), nil
	}
	return mcp.HandleToolSuccess(shared.FormatCommandResult("govulncheck:", out, err)), nil
}

func (s *SecurityMCP) handleValidateSecurityHeaders(ctx context.Context, request mcpgo.CallToolRequest) (*mcpgo.CallToolResult, error) {
	url := strings.TrimSpace(request.GetString("url", ""))
	if url == "" {
		return mcp.HandleToolError(fmt.Errorf("url is required"), "validate_security_headers"), nil
	}
	reqCtx, cancel := context.WithTimeout(ctx, 15*time.Second)
	defer cancel()
	req, err := http.NewRequestWithContext(reqCtx, http.MethodHead, url, nil)
	if err != nil {
		return mcp.HandleToolError(err, "validate_security_headers"), nil
	}
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return mcp.HandleToolError(err, "validate_security_headers"), nil
	}
	defer resp.Body.Close()

	checks := map[string]string{
		"Strict-Transport-Security": "HSTS",
		"Content-Security-Policy":   "CSP",
		"X-Frame-Options":           "Clickjacking protection",
		"X-Content-Type-Options":    "MIME sniffing protection",
		"Referrer-Policy":           "Referrer policy",
	}
	var b strings.Builder
	fmt.Fprintf(&b, "Security headers for %s (status %d):\n", url, resp.StatusCode)
	for header, desc := range checks {
		val := resp.Header.Get(header)
		if val == "" {
			fmt.Fprintf(&b, "  MISSING %s (%s)\n", header, desc)
		} else {
			fmt.Fprintf(&b, "  OK %s: %s\n", header, val)
		}
	}
	return mcp.HandleToolSuccess(b.String()), nil
}

func commandExists(name string) bool {
	_, err := exec.LookPath(name)
	return err == nil
}
