package backend

import (
	"context"
	"fmt"
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	mcp "github.com/camronwood/neural-junkie/internal/mcp"
	mcpgo "github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server"
)

// BackendMCP provides MCP tools for Go/Backend development
type BackendMCP struct {
	mcpServer  *server.MCPServer
	httpServer *server.StreamableHTTPServer
	config     *mcp.MCPServerConfig
}

// NewBackendMCP creates a new Backend MCP server
func NewBackendMCP() (*BackendMCP, error) {
	config := mcp.GetMCPServerConfig("BACKEND")

	mcpServer, httpServer, err := mcp.NewMCPServer(config)
	if err != nil {
		return nil, fmt.Errorf("failed to create MCP server: %w", err)
	}

	b := &BackendMCP{
		mcpServer:  mcpServer,
		httpServer: httpServer,
		config:     config,
	}

	b.registerTools()

	return b, nil
}

// Start starts the Backend MCP server
func (b *BackendMCP) Start() error {
	if b.httpServer == nil {
		return fmt.Errorf("MCP server not configured")
	}

	return mcp.StartMCPServer(b.httpServer, b.config.Port)
}

// GetMCPServer returns the underlying MCP server
func (b *BackendMCP) GetMCPServer() *server.MCPServer {
	return b.mcpServer
}

func (b *BackendMCP) registerTools() {
	b.mcpServer.AddTool(mcp.CreateTool(
		"analyze_go_code",
		"Analyze Go code for issues using go vet, staticcheck, and golangci-lint",
		mcp.CreateStringInputSchema("file_path", "Path to Go file or directory to analyze"),
		nil,
	), b.handleAnalyzeGoCode)

	b.mcpServer.AddTool(mcp.CreateTool(
		"run_go_tests",
		"Execute Go tests and return results",
		mcp.CreateStringInputSchema("package_path", "Go package path to test (e.g., ./cmd/server or .)"),
		nil,
	), b.handleRunGoTests)

	b.mcpServer.AddTool(mcp.CreateTool(
		"profile_performance",
		"Profile Go application performance using pprof",
		mcp.CreateMultiStringInputSchema(map[string]string{
			"binary_path": "Path to Go binary to profile",
			"endpoint":    "HTTP endpoint to profile (e.g., /api/users)",
		}),
		nil,
	), b.handleProfilePerformance)

	b.mcpServer.AddTool(mcp.CreateTool(
		"check_dependencies",
		"Check Go module dependencies for vulnerabilities and updates",
		mcp.CreateStringInputSchema("module_path", "Path to Go module (directory containing go.mod)"),
		nil,
	), b.handleCheckDependencies)

	b.mcpServer.AddTool(mcp.CreateTool(
		"detect_race_conditions",
		"Run Go race detector on tests to find race conditions",
		mcp.CreateStringInputSchema("package_path", "Go package path to test for races"),
		nil,
	), b.handleDetectRaceConditions)

	log.Printf("Registered %d Backend MCP tools", len(b.mcpServer.ListTools()))
}

func (b *BackendMCP) handleAnalyzeGoCode(ctx context.Context, request mcpgo.CallToolRequest) (*mcpgo.CallToolResult, error) {
	filePath := request.GetString("file_path", "")
	if filePath == "" {
		return mcp.HandleToolError(fmt.Errorf("file_path is required"), "analyze_go_code"), nil
	}
	if !b.isValidGoPath(filePath) {
		return mcp.HandleToolError(fmt.Errorf("invalid Go file path: %s", filePath), "analyze_go_code"), nil
	}

	var results []string
	var errors []string

	if result, err := b.runGoVet(filePath); err != nil {
		errors = append(errors, fmt.Sprintf("go vet error: %v", err))
	} else if result != "" {
		results = append(results, "=== go vet ===", result)
	}

	if result, err := b.runStaticcheck(filePath); err != nil {
		errors = append(errors, fmt.Sprintf("staticcheck error: %v", err))
	} else if result != "" {
		results = append(results, "=== staticcheck ===", result)
	}

	if result, err := b.runGolangciLint(filePath); err != nil {
		errors = append(errors, fmt.Sprintf("golangci-lint error: %v", err))
	} else if result != "" {
		results = append(results, "=== golangci-lint ===", result)
	}

	var output strings.Builder
	if len(results) > 0 {
		output.WriteString(strings.Join(results, "\n\n"))
	}
	if len(errors) > 0 {
		output.WriteString("\n\n=== Errors ===\n")
		output.WriteString(strings.Join(errors, "\n"))
	}
	if output.Len() == 0 {
		output.WriteString("No issues found in Go code analysis.")
	}

	return mcp.HandleToolSuccess(output.String()), nil
}

func (b *BackendMCP) handleRunGoTests(ctx context.Context, request mcpgo.CallToolRequest) (*mcpgo.CallToolResult, error) {
	if err := mcp.ValidateToolInput(request, []string{"package_path"}); err != nil {
		return mcp.HandleToolError(err, "run_go_tests"), nil
	}

	packagePath := request.GetString("package_path", "")
	if !b.isValidGoPath(packagePath) {
		return mcp.HandleToolError(fmt.Errorf("invalid Go package path: %s", packagePath), "run_go_tests"), nil
	}

	cmd := exec.CommandContext(ctx, "go", "test", "-v", "-timeout", "30s", packagePath)
	cmd.Dir = b.getWorkingDir(packagePath)

	output, err := cmd.CombinedOutput()
	if err != nil {
		return mcp.HandleToolError(fmt.Errorf("test execution failed: %w\nOutput: %s", err, string(output)), "run_go_tests"), nil
	}

	return mcp.HandleToolSuccess(fmt.Sprintf("Test Results for %s:\n%s", packagePath, string(output))), nil
}

func (b *BackendMCP) handleProfilePerformance(ctx context.Context, request mcpgo.CallToolRequest) (*mcpgo.CallToolResult, error) {
	if err := mcp.ValidateToolInput(request, []string{"binary_path", "endpoint"}); err != nil {
		return mcp.HandleToolError(err, "profile_performance"), nil
	}

	binaryPath := request.GetString("binary_path", "")
	endpoint := request.GetString("endpoint", "")

	if !b.isValidGoPath(binaryPath) {
		return mcp.HandleToolError(fmt.Errorf("invalid binary path: %s", binaryPath), "profile_performance"), nil
	}

	result := fmt.Sprintf("Performance profiling for %s endpoint %s:\n", binaryPath, endpoint)
	result += "Note: Placeholder implementation. Full profiling requires pprof-enabled binary and load testing."

	return mcp.HandleToolSuccess(result), nil
}

func (b *BackendMCP) handleCheckDependencies(ctx context.Context, request mcpgo.CallToolRequest) (*mcpgo.CallToolResult, error) {
	if err := mcp.ValidateToolInput(request, []string{"module_path"}); err != nil {
		return mcp.HandleToolError(err, "check_dependencies"), nil
	}

	modulePath := request.GetString("module_path", "")
	if !b.isValidGoPath(modulePath) {
		return mcp.HandleToolError(fmt.Errorf("invalid module path: %s", modulePath), "check_dependencies"), nil
	}

	goModPath := filepath.Join(modulePath, "go.mod")
	if _, err := os.Stat(goModPath); os.IsNotExist(err) {
		return mcp.HandleToolError(fmt.Errorf("go.mod not found in %s", modulePath), "check_dependencies"), nil
	}

	var results []string

	cmd := exec.CommandContext(ctx, "go", "mod", "why", "all")
	cmd.Dir = modulePath
	output, err := cmd.CombinedOutput()
	if err != nil {
		results = append(results, fmt.Sprintf("go mod why failed: %v", err))
	} else {
		results = append(results, "=== Dependency Analysis ===", string(output))
	}

	cmd = exec.CommandContext(ctx, "go", "list", "-u", "-m", "all")
	cmd.Dir = modulePath
	output, err = cmd.CombinedOutput()
	if err != nil {
		results = append(results, fmt.Sprintf("go list -u failed: %v", err))
	} else if string(output) != "" {
		results = append(results, "\n=== Outdated Dependencies ===", string(output))
	}

	result := strings.Join(results, "\n")
	if result == "" {
		result = "No dependency issues found."
	}

	return mcp.HandleToolSuccess(result), nil
}

func (b *BackendMCP) handleDetectRaceConditions(ctx context.Context, request mcpgo.CallToolRequest) (*mcpgo.CallToolResult, error) {
	if err := mcp.ValidateToolInput(request, []string{"package_path"}); err != nil {
		return mcp.HandleToolError(err, "detect_race_conditions"), nil
	}

	packagePath := request.GetString("package_path", "")
	if !b.isValidGoPath(packagePath) {
		return mcp.HandleToolError(fmt.Errorf("invalid package path: %s", packagePath), "detect_race_conditions"), nil
	}

	cmd := exec.CommandContext(ctx, "go", "test", "-race", "-timeout", "30s", packagePath)
	cmd.Dir = b.getWorkingDir(packagePath)

	output, err := cmd.CombinedOutput()
	if err != nil {
		return mcp.HandleToolSuccess(fmt.Sprintf("Race conditions detected in %s:\n%s", packagePath, string(output))), nil
	}

	return mcp.HandleToolSuccess(fmt.Sprintf("No race conditions detected in %s", packagePath)), nil
}

func (b *BackendMCP) isValidGoPath(path string) bool {
	if path == "" {
		return false
	}
	_, err := os.Stat(path)
	return err == nil
}

func (b *BackendMCP) getWorkingDir(path string) string {
	if filepath.IsAbs(path) {
		if info, err := os.Stat(path); err == nil && !info.IsDir() {
			return filepath.Dir(path)
		}
		return path
	}

	dir, _ := os.Getwd()
	for {
		if _, err := os.Stat(filepath.Join(dir, "go.mod")); err == nil {
			return dir
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			break
		}
		dir = parent
	}
	return dir
}

func (b *BackendMCP) runGoVet(path string) (string, error) {
	cmd := exec.Command("go", "vet", path)
	cmd.Dir = b.getWorkingDir(path)
	output, err := cmd.CombinedOutput()
	if err != nil {
		return string(output), err
	}
	return "", nil
}

func (b *BackendMCP) runStaticcheck(path string) (string, error) {
	cmd := exec.Command("staticcheck", path)
	cmd.Dir = b.getWorkingDir(path)
	output, err := cmd.CombinedOutput()
	if err != nil {
		return "", fmt.Errorf("staticcheck not available: %w", err)
	}
	return string(output), nil
}

func (b *BackendMCP) runGolangciLint(path string) (string, error) {
	cmd := exec.Command("golangci-lint", "run", path)
	cmd.Dir = b.getWorkingDir(path)
	output, err := cmd.CombinedOutput()
	if err != nil {
		return "", fmt.Errorf("golangci-lint not available: %w", err)
	}
	return string(output), nil
}
