package mcp

import (
	"fmt"
	"log"
	"strings"
	"sync"

	mcpgo "github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server"
)

// MCPServerConfig holds configuration for MCP servers
type MCPServerConfig struct {
	Enabled       bool
	Port          int
	Name          string
	Version       string
	InProcessOnly bool // when true, skip HTTP server (multi-instance agents)
}

// defaultPorts matches env.example (MCP_*_PORT).
var defaultPorts = map[string]int{
	"BACKEND":      8081,
	"DEVOPS":       8082,
	"DATABASE":     8083,
	"FRONTEND":     8084,
	"SECURITY":     8085,
	"BIOLOGY":      8087,
	"RUST":         8088,
	"CODE_REVIEW":  8089,
	"ARCHITECTURE": 8090,
	"CAD":          8091,
	"AWS":          8092,
	"INCIDENT":     8093,
}

// NormalizeAgentType maps agent type strings to MCP port config keys (e.g. code-review -> CODE_REVIEW).
func NormalizeAgentType(agentType string) string {
	key := strings.ToLower(strings.TrimSpace(agentType))
	key = strings.ReplaceAll(key, "-", "_")
	return strings.ToUpper(key)
}

var (
	mcpPortMu    sync.Mutex
	mcpPortsUsed = map[int]bool{}
)

// ResetMCPPortReservation clears HTTP port claims (tests only).
func ResetMCPPortReservation() {
	mcpPortMu.Lock()
	defer mcpPortMu.Unlock()
	mcpPortsUsed = map[int]bool{}
}

// reserveMCPHTTPPort claims port for the first specialist of a type; later agents share tools in-process only.
func reserveMCPHTTPPort(port int) bool {
	mcpPortMu.Lock()
	defer mcpPortMu.Unlock()
	if mcpPortsUsed[port] {
		return false
	}
	mcpPortsUsed[port] = true
	return true
}

// GetMCPServerConfig returns configuration for a specific agent type (BACKEND, BIOLOGY, …).
// Enablement comes from hub config (Settings → MCP / domain packs), not environment variables.
// When multiple in-process specialists share a type/port, only the first binds HTTP; others stay in-process.
func GetMCPServerConfig(agentType string) *MCPServerConfig {
	key := NormalizeAgentType(agentType)
	port := defaultPorts[key]
	if port == 0 {
		port = 8081
	}
	enabled := false
	if cfg := AppConfig(); cfg != nil {
		enabled = cfg.MCPEnabledForAgent(key)
		if p := cfg.MCPPort(key); p > 0 {
			port = p
		}
	}
	cfg := &MCPServerConfig{
		Enabled: enabled,
		Port:    port,
		Name:    fmt.Sprintf("%s-agent-mcp", strings.ToLower(key)),
		Version: "1.0.0",
	}
	if enabled && !reserveMCPHTTPPort(port) {
		cfg.InProcessOnly = true
	}
	return cfg
}

// NewMCPServer creates a new MCP server with common configuration.
func NewMCPServer(config *MCPServerConfig) (*server.MCPServer, *server.StreamableHTTPServer, error) {
	if config == nil {
		return nil, nil, fmt.Errorf("nil MCP config")
	}
	if !config.Enabled {
		return nil, nil, fmt.Errorf("MCP server disabled for %s", config.Name)
	}

	mcpServer := server.NewMCPServer(config.Name, config.Version)

	var httpServer *server.StreamableHTTPServer
	if !config.InProcessOnly {
		httpServer = server.NewStreamableHTTPServer(
			mcpServer,
			server.WithEndpointPath("/mcp"),
		)
		log.Printf("Created MCP server: %s v%s on port %d", config.Name, config.Version, config.Port)
	} else {
		log.Printf("Created in-process MCP server: %s v%s", config.Name, config.Version)
	}

	return mcpServer, httpServer, nil
}

// NewInProcessMCPServer creates an MCP server without an HTTP listener (repo/confluence agents).
func NewInProcessMCPServer(name, version string) (*server.MCPServer, error) {
	cfg := &MCPServerConfig{
		Enabled:       true,
		Name:          name,
		Version:       version,
		InProcessOnly: true,
	}
	mcpServer, _, err := NewMCPServer(cfg)
	return mcpServer, err
}

// StartMCPServer starts the MCP server in a goroutine. In-process-only servers no-op.
func StartMCPServer(httpServer *server.StreamableHTTPServer, port int) error {
	if httpServer == nil {
		return nil
	}
	addr := fmt.Sprintf(":%d", port)

	go func() {
		log.Printf("Starting MCP server on %s", addr)
		if err := httpServer.Start(addr); err != nil {
			log.Printf("MCP server failed to start: %v", err)
		}
	}()

	return nil
}

// CreateTool creates a standardized MCP tool definition
func CreateTool(name, description string, inputSchema mcpgo.ToolInputSchema, handler server.ToolHandlerFunc) mcpgo.Tool {
	return mcpgo.Tool{
		Name:        name,
		Description: description,
		InputSchema: inputSchema,
	}
}

// CreateEmptyInputSchema creates a tool schema with no parameters.
func CreateEmptyInputSchema() mcpgo.ToolInputSchema {
	return mcpgo.ToolInputSchema{
		Type:       "object",
		Properties: map[string]any{},
	}
}

// CreateStringInputSchema creates a simple string input schema
func CreateStringInputSchema(paramName, description string) mcpgo.ToolInputSchema {
	return mcpgo.ToolInputSchema{
		Type: "object",
		Properties: map[string]any{
			paramName: map[string]any{
				"type":        "string",
				"description": description,
			},
		},
		Required: []string{paramName},
	}
}

// CreateMultiStringInputSchema creates a schema with multiple string parameters
func CreateMultiStringInputSchema(params map[string]string) mcpgo.ToolInputSchema {
	properties := make(map[string]any)
	required := make([]string, 0, len(params))

	for name, description := range params {
		properties[name] = map[string]any{
			"type":        "string",
			"description": description,
		}
		required = append(required, name)
	}

	return mcpgo.ToolInputSchema{
		Type:       "object",
		Properties: properties,
		Required:   required,
	}
}

// CreateObjectInputSchema builds a tool schema from property maps and required keys.
func CreateObjectInputSchema(properties map[string]interface{}, required []string) mcpgo.ToolInputSchema {
	return mcpgo.ToolInputSchema{
		Type:       "object",
		Properties: properties,
		Required:   required,
	}
}

// HandleToolError creates a standardized error response
func HandleToolError(err error, toolName string) *mcpgo.CallToolResult {
	log.Printf("Tool %s error: %v", toolName, err)
	return &mcpgo.CallToolResult{
		Content: []mcpgo.Content{
			mcpgo.TextContent{
				Type: "text",
				Text: fmt.Sprintf("Error in %s: %v", toolName, err),
			},
		},
		IsError: true,
	}
}

// HandleToolSuccess creates a standardized success response
func HandleToolSuccess(result string) *mcpgo.CallToolResult {
	return &mcpgo.CallToolResult{
		Content: []mcpgo.Content{
			mcpgo.TextContent{
				Type: "text",
				Text: result,
			},
		},
	}
}

// ValidateToolInput validates required string parameters
func ValidateToolInput(request mcpgo.CallToolRequest, requiredParams []string) error {
	for _, param := range requiredParams {
		if request.GetString(param, "") == "" {
			return fmt.Errorf("missing required parameter: %s", param)
		}
	}
	return nil
}

// MissingBinaryMessage returns a helpful message when an optional CLI tool is not installed.
func MissingBinaryMessage(binary, installHint string) string {
	return fmt.Sprintf("%s is not installed or not on PATH. %s", binary, installHint)
}
