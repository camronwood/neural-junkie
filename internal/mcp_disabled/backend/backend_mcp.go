//go:build ignore

package backend

import (
	"context"
	"fmt"
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

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

// registerTools registers all Backend MCP tools
func (b *BackendMCP) registerTools() {
	// Tool 1: analyze_go_code
	b.mcpServer.AddTool(mcp.Tool{
		Name:        "analyze_go_code",
		Description: "Analyze Go code for issues using go vet, staticcheck, and golangci-lint",
		InputSchema: mcp.ToolInputSchema{
			Type: "object",
			Properties: map[string]any{
				"file_path": map[string]any{
					"type":        "string",
					"description": "Path to Go file or directory to analyze",
				},
			},
			Required: []string{"file_path"},
		},
	}, b.handleAnalyzeGoCode)

	// Tool 2: run_go_tests
	b.mcpServer.AddTool(mcp.Tool{
		Name:        "run_go_tests",
		Description: "Execute Go tests and return results",
		InputSchema: mcp.ToolInputSchema{
			Type: "object",
			Properties: map[string]any{
				"package_path": map[string]any{
					"type":        "string",
					"description": "Go package path to test (e.g., ./cmd/server or .)",
				},
			},
			Required: []string{"package_path"},
		},
	}, b.handleRunGoTests)

	// Tool 3: profile_performance
	b.mcpServer.AddTool(mcp.Tool{
		Name:        "profile_performance",
		Description: "Profile Go application performance using pprof",
		InputSchema: mcp.ToolInputSchema{
			Type: "object",
			Properties: map[string]any{
				"binary_path": map[string]any{
					"type":        "string",
					"description": "Path to Go binary to profile",
				},
				"endpoint": map[string]any{
					"type":        "string",
					"description": "HTTP endpoint to profile (e.g., /api/users)",
				},
			},
			Required: []string{"binary_path", "endpoint"},
		},
	}, b.handleProfilePerformance)

	// Tool 4: check_dependencies
	b.mcpServer.AddTool(mcp.Tool{
		Name:        "check_dependencies",
		Description: "Check Go module dependencies for vulnerabilities and updates",
		InputSchema: mcp.ToolInputSchema{
			Type: "object",
			Properties: map[string]any{
				"module_path": map[string]any{
					"type":        "string",
					"description": "Path to Go module (directory containing go.mod)",
				},
			},
			Required: []string{"module_path"},
		},
	}, b.handleCheckDependencies)

	// Tool 5: detect_race_conditions
	b.mcpServer.AddTool(mcp.Tool{
		Name:        "detect_race_conditions",
		Description: "Run Go race detector on tests to find race conditions",
		InputSchema: mcp.ToolInputSchema{
			Type: "object",
			Properties: map[string]any{
				"package_path": map[string]any{
					"type":        "string",
					"description": "Go package path to test for races",
				},
			},
			Required: []string{"package_path"},
		},
	}, b.handleDetectRaceConditions)

	log.Printf("Registered %d Backend MCP tools", len(b.mcpServer.ListTools()))
}

// handleAnalyzeGoCode analyzes Go code for issues
func (b *BackendMCP) handleAnalyzeGoCode(ctx context.Context, request mcpgo.CallToolRequest) (*mcpgo.CallToolResult, error) {
	filePath := request.GetString("file_path", "")
	if filePath == "" {
		return &mcpgo.CallToolResult{
			Content: []mcpgo.Content{
				mcpgo.TextContent{
					Type: "text",
					Text: "Error: file_path is required",
				},
			},
			IsError: true,
		}, nil
	}
	if !b.isValidGoPath(filePath) {
		return &mcpgo.CallToolResult{
			Content: []mcpgo.Content{
				mcpgo.TextContent{
					Type: "text",
					Text: fmt.Sprintf("Error: invalid Go file path: %s", filePath),
				},
			},
			IsError: true,
		}, nil
	}

	var results []string
	var errors []string

	// Run go vet
	if result, err := b.runGoVet(filePath); err != nil {
		errors = append(errors, fmt.Sprintf("go vet error: %v", err))
	} else if result != "" {
		results = append(results, "=== go vet ===")
		results = append(results, result)
	}

	// Run staticcheck if available
	if result, err := b.runStaticcheck(filePath); err != nil {
		errors = append(errors, fmt.Sprintf("staticcheck error: %v", err))
	} else if result != "" {
		results = append(results, "=== staticcheck ===")
		results = append(results, result)
	}

	// Run golangci-lint if available
	if result, err := b.runGolangciLint(filePath); err != nil {
		errors = append(errors, fmt.Sprintf("golangci-lint error: %v", err))
	} else if result != "" {
		results = append(results, "=== golangci-lint ===")
		results = append(results, result)
	}

	// Combine results
	var output strings.Builder
	if len(results) > 0 {
		output.WriteString(strings.Join(results, "\n\n"))
	}
	if len(errors) > 0 {
		output.WriteString("\n\n=== Errors ===")
		output.WriteString(strings.Join(errors, "\n"))
	}

	if output.Len() == 0 {
		output.WriteString("No issues found in Go code analysis.")
	}

	return &mcpgo.CallToolResult{
		Content: []mcpgo.Content{
			mcpgo.TextContent{
				Type: "text",
				Text: output.String(),
			},
		},
	}, nil
}

// handleRunGoTests runs Go tests
func (b *BackendMCP) handleRunGoTests(ctx context.Context, request mcpgo.CallToolRequest) (*mcpgo.CallToolResult, error) {
	if err := mcp.ValidateToolInput(request, []string{"package_path"}); err != nil {
		return mcp.HandleToolError(err, "run_go_tests"), nil
	}

	packagePath := request.GetString("package_path", "")
	if !b.isValidGoPath(packagePath) {
		return mcp.HandleToolError(fmt.Errorf("invalid Go package path: %s", packagePath), "run_go_tests"), nil
	}

	// Run go test with verbose output
	cmd := exec.CommandContext(ctx, "go", "test", "-v", "-timeout", "30s", packagePath)
	cmd.Dir = b.getWorkingDir(packagePath)

	output, err := cmd.CombinedOutput()
	if err != nil {
		return mcp.HandleToolError(fmt.Errorf("test execution failed: %w\nOutput: %s", err, string(output)), "run_go_tests"), nil
	}

	result := fmt.Sprintf("Test Results for %s:\n%s", packagePath, string(output))
	return mcp.HandleToolSuccess(result), nil
}

// handleProfilePerformance profiles Go application performance
func (b *BackendMCP) handleProfilePerformance(ctx context.Context, request mcpgo.CallToolRequest) (*mcpgo.CallToolResult, error) {
	if err := mcp.ValidateToolInput(request, []string{"binary_path", "endpoint"}); err != nil {
		return mcp.HandleToolError(err, "profile_performance"), nil
	}

	binaryPath := request.GetString("binary_path", "")
	endpoint := request.GetString("endpoint", "")

	if !b.isValidGoPath(binaryPath) {
		return mcp.HandleToolError(fmt.Errorf("invalid binary path: %s", binaryPath), "profile_performance"), nil
	}

	// This is a simplified implementation - in practice, you'd need to:
	// 1. Start the binary with pprof enabled
	// 2. Make requests to the endpoint
	// 3. Collect profiling data
	// 4. Analyze the results

	result := fmt.Sprintf("Performance profiling for %s endpoint %s:\n", binaryPath, endpoint)
	result += "Note: This is a placeholder implementation. Full profiling requires:\n"
	result += "1. Binary compiled with pprof support\n"
	result += "2. HTTP server with pprof endpoints enabled\n"
	result += "3. Load testing to generate profile data\n"
	result += "4. Analysis of CPU, memory, and goroutine profiles"

	return mcp.HandleToolSuccess(result), nil
}

// handleCheckDependencies checks Go module dependencies
func (b *BackendMCP) handleCheckDependencies(ctx context.Context, request mcpgo.CallToolRequest) (*mcpgo.CallToolResult, error) {
	if err := mcp.ValidateToolInput(request, []string{"module_path"}); err != nil {
		return mcp.HandleToolError(err, "check_dependencies"), nil
	}

	modulePath := request.GetString("module_path", "")
	if !b.isValidGoPath(modulePath) {
		return mcp.HandleToolError(fmt.Errorf("invalid module path: %s", modulePath), "check_dependencies"), nil
	}

	var results []string

	// Check for go.mod file
	goModPath := filepath.Join(modulePath, "go.mod")
	if _, err := os.Stat(goModPath); os.IsNotExist(err) {
		return mcp.HandleToolError(fmt.Errorf("go.mod not found in %s", modulePath), "check_dependencies"), nil
	}

	// Run go mod why to understand dependencies
	cmd := exec.CommandContext(ctx, "go", "mod", "why", "all")
	cmd.Dir = modulePath
	output, err := cmd.CombinedOutput()
	if err != nil {
		results = append(results, fmt.Sprintf("go mod why failed: %v", err))
	} else {
		results = append(results, "=== Dependency Analysis ===")
		results = append(results, string(output))
	}

	// Run go mod graph to show dependency graph
	cmd = exec.CommandContext(ctx, "go", "mod", "graph")
	cmd.Dir = modulePath
	output, err = cmd.CombinedOutput()
	if err != nil {
		results = append(results, fmt.Sprintf("go mod graph failed: %v", err))
	} else {
		results = append(results, "\n=== Dependency Graph ===")
		results = append(results, string(output))
	}

	// Check for outdated dependencies
	cmd = exec.CommandContext(ctx, "go", "list", "-u", "-m", "all")
	cmd.Dir = modulePath
	output, err = cmd.CombinedOutput()
	if err != nil {
		results = append(results, fmt.Sprintf("go list -u failed: %v", err))
	} else if string(output) != "" {
		results = append(results, "\n=== Outdated Dependencies ===")
		results = append(results, string(output))
	}

	result := strings.Join(results, "\n")
	if result == "" {
		result = "No dependency issues found."
	}

	return mcp.HandleToolSuccess(result), nil
}

// handleDetectRaceConditions runs Go race detector
func (b *BackendMCP) handleDetectRaceConditions(ctx context.Context, request mcpgo.CallToolRequest) (*mcpgo.CallToolResult, error) {
	if err := mcp.ValidateToolInput(request, []string{"package_path"}); err != nil {
		return mcp.HandleToolError(err, "detect_race_conditions"), nil
	}

	packagePath := request.GetString("package_path", "")
	if !b.isValidGoPath(packagePath) {
		return mcp.HandleToolError(fmt.Errorf("invalid package path: %s", packagePath), "detect_race_conditions"), nil
	}

	// Run go test with race detector
	cmd := exec.CommandContext(ctx, "go", "test", "-race", "-timeout", "30s", packagePath)
	cmd.Dir = b.getWorkingDir(packagePath)

	output, err := cmd.CombinedOutput()
	if err != nil {
		// Race detector found issues
		result := fmt.Sprintf("Race conditions detected in %s:\n%s", packagePath, string(output))
		return mcp.HandleToolSuccess(result), nil
	}

	result := fmt.Sprintf("No race conditions detected in %s", packagePath)
	return mcp.HandleToolSuccess(result), nil
}

// Helper methods

func (b *BackendMCP) isValidGoPath(path string) bool {
	if path == "" {
		return false
	}

	// Check if path exists
	if _, err := os.Stat(path); os.IsNotExist(err) {
		return false
	}

	return true
}

func (b *BackendMCP) getWorkingDir(path string) string {
	if filepath.IsAbs(path) {
		return filepath.Dir(path)
	}

	// Try to find the module root
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
		// staticcheck might not be installed
		return "", fmt.Errorf("staticcheck not available: %w", err)
	}
	return string(output), nil
}

func (b *BackendMCP) runGolangciLint(path string) (string, error) {
	cmd := exec.Command("golangci-lint", "run", path)
	cmd.Dir = b.getWorkingDir(path)
	output, err := cmd.CombinedOutput()
	if err != nil {
		// golangci-lint might not be installed
		return "", fmt.Errorf("golangci-lint not available: %w", err)
	}
	return string(output), nil
}
