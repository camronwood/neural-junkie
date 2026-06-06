package frontend

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"strings"

	mcp "github.com/camronwood/neural-junkie/internal/mcp"
	"github.com/camronwood/neural-junkie/internal/mcp/shared"
	mcpgo "github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server"
)

// FrontendMCP provides MCP tools for frontend development.
type FrontendMCP struct {
	mcpServer  *server.MCPServer
	httpServer *server.StreamableHTTPServer
	config     *mcp.MCPServerConfig
}

// NewFrontendMCP creates a new Frontend MCP server.
func NewFrontendMCP() (*FrontendMCP, error) {
	config := mcp.GetMCPServerConfig("frontend")
	mcpServer, httpServer, err := mcp.NewMCPServer(config)
	if err != nil {
		return nil, fmt.Errorf("failed to create MCP server: %w", err)
	}
	f := &FrontendMCP{mcpServer: mcpServer, httpServer: httpServer, config: config}
	f.registerTools()
	return f, nil
}

func (f *FrontendMCP) Start() error {
	if f.httpServer == nil {
		return fmt.Errorf("MCP server not configured")
	}
	return mcp.StartMCPServer(f.httpServer, f.config.Port)
}

func (f *FrontendMCP) GetMCPServer() *server.MCPServer {
	return f.mcpServer
}

func (f *FrontendMCP) registerTools() {
	f.mcpServer.AddTool(mcp.CreateTool(
		"run_typescript_check",
		"Run TypeScript compiler check (tsc --noEmit) in a project directory",
		mcp.CreateStringInputSchema("project_path", "Path to directory containing tsconfig.json"),
		nil,
	), f.handleRunTypescriptCheck)

	f.mcpServer.AddTool(mcp.CreateTool(
		"run_eslint",
		"Run ESLint on a file or directory",
		mcp.CreateStringInputSchema("target_path", "Path to file or directory to lint"),
		nil,
	), f.handleRunESLint)

	f.mcpServer.AddTool(mcp.CreateTool(
		"check_package_json",
		"Parse package.json and summarize scripts, dependencies, and devDependencies",
		mcp.CreateStringInputSchema("project_path", "Path to directory containing package.json"),
		nil,
	), f.handleCheckPackageJSON)

	f.mcpServer.AddTool(mcp.CreateTool(
		"analyze_css",
		"Analyze CSS files (stylelint if available, else basic structural checks)",
		mcp.CreateStringInputSchema("css_path", "Path to CSS/SCSS file or directory"),
		nil,
	), f.handleAnalyzeCSS)

	log.Printf("Registered %d Frontend MCP tools", len(f.mcpServer.ListTools()))
}

func (f *FrontendMCP) handleRunTypescriptCheck(ctx context.Context, request mcpgo.CallToolRequest) (*mcpgo.CallToolResult, error) {
	projectPath := request.GetString("project_path", ".")
	root := shared.FindProjectRoot(projectPath, "tsconfig.json", "package.json")
	if !shared.PathExists(filepath.Join(root, "tsconfig.json")) {
		return mcp.HandleToolError(fmt.Errorf("tsconfig.json not found in %s", root), "run_typescript_check"), nil
	}
	if !shared.ProjectHasTypeScript(root) {
		return mcp.HandleToolSuccess(shared.TypeScriptNotConfiguredMessage(root)), nil
	}
	out, err := shared.RunTypeScriptCheck(ctx, root)
	return mcp.HandleToolSuccess(shared.FormatCommandResult("TypeScript check:", out, err)), nil
}

func (f *FrontendMCP) handleRunESLint(ctx context.Context, request mcpgo.CallToolRequest) (*mcpgo.CallToolResult, error) {
	target := request.GetString("target_path", "")
	if target == "" {
		return mcp.HandleToolError(fmt.Errorf("target_path is required"), "run_eslint"), nil
	}
	if !shared.PathExists(target) {
		return mcp.HandleToolError(fmt.Errorf("path not found: %s", target), "run_eslint"), nil
	}
	root := shared.FindProjectRoot(target, "package.json", ".eslintrc.js", ".eslintrc.json", "eslint.config.js")
	if !shared.ProjectHasESLint(root) {
		return mcp.HandleToolSuccess(shared.ESLintNotConfiguredMessage(root)), nil
	}
	out, err := shared.RunCommand(ctx, root, "npx", "--yes", "eslint", target)
	if err != nil && strings.Contains(out, "eslint") && strings.Contains(out, "not found") {
		return mcp.HandleToolSuccess(mcp.MissingBinaryMessage("eslint", "Run npm install eslint in the project or install globally.")), nil
	}
	return mcp.HandleToolSuccess(shared.FormatCommandResult("ESLint:", out, err)), nil
}

func (f *FrontendMCP) handleCheckPackageJSON(ctx context.Context, request mcpgo.CallToolRequest) (*mcpgo.CallToolResult, error) {
	projectPath := request.GetString("project_path", ".")
	root := shared.FindProjectRoot(projectPath, "package.json")
	pkgPath := filepath.Join(root, "package.json")
	data, err := os.ReadFile(pkgPath)
	if err != nil {
		return mcp.HandleToolError(fmt.Errorf("package.json not found: %w", err), "check_package_json"), nil
	}
	var pkg struct {
		Name            string            `json:"name"`
		Version         string            `json:"version"`
		Scripts         map[string]string `json:"scripts"`
		Dependencies    map[string]string `json:"dependencies"`
		DevDependencies map[string]string `json:"devDependencies"`
	}
	if err := json.Unmarshal(data, &pkg); err != nil {
		return mcp.HandleToolError(err, "check_package_json"), nil
	}
	var b strings.Builder
	fmt.Fprintf(&b, "Package: %s@%s\n\nScripts (%d):\n", pkg.Name, pkg.Version, len(pkg.Scripts))
	for k, v := range pkg.Scripts {
		fmt.Fprintf(&b, "  %s: %s\n", k, v)
	}
	fmt.Fprintf(&b, "\nDependencies (%d):\n", len(pkg.Dependencies))
	for k, v := range pkg.Dependencies {
		fmt.Fprintf(&b, "  %s: %s\n", k, v)
	}
	fmt.Fprintf(&b, "\nDevDependencies (%d):\n", len(pkg.DevDependencies))
	for k, v := range pkg.DevDependencies {
		fmt.Fprintf(&b, "  %s: %s\n", k, v)
	}
	return mcp.HandleToolSuccess(b.String()), nil
}

func (f *FrontendMCP) handleAnalyzeCSS(ctx context.Context, request mcpgo.CallToolRequest) (*mcpgo.CallToolResult, error) {
	cssPath := request.GetString("css_path", "")
	if cssPath == "" || !shared.PathExists(cssPath) {
		return mcp.HandleToolError(fmt.Errorf("css_path not found"), "analyze_css"), nil
	}
	root := shared.FindProjectRoot(cssPath, "package.json")
	out, err := shared.RunCommand(ctx, root, "npx", "--yes", "stylelint", cssPath)
	if err == nil || out != "" {
		return mcp.HandleToolSuccess(shared.FormatCommandResult("Stylelint:", out, err)), nil
	}
	data, readErr := os.ReadFile(cssPath)
	if readErr != nil {
		return mcp.HandleToolError(readErr, "analyze_css"), nil
	}
	content := string(data)
	var issues []string
	if strings.Count(content, "{") != strings.Count(content, "}") {
		issues = append(issues, "Mismatched brace count")
	}
	if strings.Contains(content, "!important") {
		issues = append(issues, "Contains !important declarations")
	}
	if len(issues) == 0 {
		return mcp.HandleToolSuccess("Basic CSS check: no structural issues found. Install stylelint for deeper analysis."), nil
	}
	return mcp.HandleToolSuccess("Basic CSS issues:\n- " + strings.Join(issues, "\n- ")), nil
}
