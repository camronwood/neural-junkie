package codereview

import (
	"context"
	"fmt"
	"log"
	"os"
	"path/filepath"

	mcp "github.com/camronwood/neural-junkie/internal/mcp"
	"github.com/camronwood/neural-junkie/internal/mcp/shared"
	mcpgo "github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server"
)

// CodeReviewMCP provides read-only diagnostic MCP tools for code review.
type CodeReviewMCP struct {
	mcpServer  *server.MCPServer
	httpServer *server.StreamableHTTPServer
	config     *mcp.MCPServerConfig
}

// NewCodeReviewMCP creates a new Code Review MCP server.
func NewCodeReviewMCP() (*CodeReviewMCP, error) {
	config := mcp.GetMCPServerConfig("code-review")
	mcpServer, httpServer, err := mcp.NewMCPServer(config)
	if err != nil {
		return nil, fmt.Errorf("failed to create MCP server: %w", err)
	}
	c := &CodeReviewMCP{mcpServer: mcpServer, httpServer: httpServer, config: config}
	c.registerTools()
	return c, nil
}

func (c *CodeReviewMCP) Start() error {
	if c.httpServer == nil {
		return fmt.Errorf("MCP server not configured")
	}
	return mcp.StartMCPServer(c.httpServer, c.config.Port)
}

func (c *CodeReviewMCP) GetMCPServer() *server.MCPServer {
	return c.mcpServer
}

func (c *CodeReviewMCP) registerTools() {
	c.mcpServer.AddTool(mcp.CreateTool(
		"analyze_go_code",
		"Analyze Go code using go vet (read-only review)",
		mcp.CreateStringInputSchema("file_path", "Path to Go file or directory"),
		nil,
	), c.handleAnalyzeGoCode)

	c.mcpServer.AddTool(mcp.CreateTool(
		"run_go_tests",
		"Run Go tests for review (read-only)",
		mcp.CreateStringInputSchema("package_path", "Go package path to test"),
		nil,
	), c.handleRunGoTests)

	c.mcpServer.AddTool(mcp.CreateTool(
		"run_eslint",
		"Run ESLint for frontend code review",
		mcp.CreateStringInputSchema("target_path", "Path to lint"),
		nil,
	), c.handleRunESLint)

	c.mcpServer.AddTool(mcp.CreateTool(
		"run_typescript_check",
		"Run TypeScript check for review",
		mcp.CreateStringInputSchema("project_path", "Project directory with tsconfig.json"),
		nil,
	), c.handleRunTypescriptCheck)

	log.Printf("Registered %d Code Review MCP tools", len(c.mcpServer.ListTools()))
}

func (c *CodeReviewMCP) workingDir(path string) string {
	if filepath.IsAbs(path) {
		if info, err := os.Stat(path); err == nil && !info.IsDir() {
			return filepath.Dir(path)
		}
		return path
	}
	return shared.FindProjectRoot(path, "go.mod", "package.json")
}

func (c *CodeReviewMCP) handleAnalyzeGoCode(ctx context.Context, request mcpgo.CallToolRequest) (*mcpgo.CallToolResult, error) {
	filePath := request.GetString("file_path", "")
	if filePath == "" || !shared.PathExists(filePath) {
		return mcp.HandleToolError(fmt.Errorf("file_path not found"), "analyze_go_code"), nil
	}
	out, err := shared.RunCommand(ctx, c.workingDir(filePath), "go", "vet", filePath)
	return mcp.HandleToolSuccess(shared.FormatCommandResult("go vet:", out, err)), nil
}

func (c *CodeReviewMCP) handleRunGoTests(ctx context.Context, request mcpgo.CallToolRequest) (*mcpgo.CallToolResult, error) {
	pkg := request.GetString("package_path", ".")
	out, err := shared.RunCommand(ctx, c.workingDir(pkg), "go", "test", "-timeout", "30s", pkg)
	return mcp.HandleToolSuccess(shared.FormatCommandResult("go test:", out, err)), nil
}

func (c *CodeReviewMCP) handleRunESLint(ctx context.Context, request mcpgo.CallToolRequest) (*mcpgo.CallToolResult, error) {
	target := request.GetString("target_path", "")
	root := shared.FindProjectRoot(target, "package.json")
	if !shared.ProjectHasESLint(root) {
		return mcp.HandleToolSuccess(shared.ESLintNotConfiguredMessage(root)), nil
	}
	out, err := shared.RunCommand(ctx, root, "npx", "--yes", "eslint", target)
	return mcp.HandleToolSuccess(shared.FormatCommandResult("eslint:", out, err)), nil
}

func (c *CodeReviewMCP) handleRunTypescriptCheck(ctx context.Context, request mcpgo.CallToolRequest) (*mcpgo.CallToolResult, error) {
	root := shared.FindProjectRoot(request.GetString("project_path", "."), "tsconfig.json", "package.json")
	if !shared.PathExists(filepath.Join(root, "tsconfig.json")) {
		return mcp.HandleToolError(fmt.Errorf("tsconfig.json not found in %s", root), "run_typescript_check"), nil
	}
	if !shared.ProjectHasTypeScript(root) {
		return mcp.HandleToolSuccess(shared.TypeScriptNotConfiguredMessage(root)), nil
	}
	out, err := shared.RunTypeScriptCheck(ctx, root)
	return mcp.HandleToolSuccess(shared.FormatCommandResult("TypeScript check:", out, err)), nil
}
