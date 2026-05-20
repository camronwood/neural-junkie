package mcp

import (
	"fmt"
	"log"
	"strings"

	mcpgo "github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server"
)

// MCPServerConfig holds configuration for MCP servers
type MCPServerConfig struct {
	Enabled bool
	Port    int
	Name    string
	Version string
}

// defaultPorts matches env.example (MCP_*_PORT).
var defaultPorts = map[string]int{
	"BACKEND":  8081,
	"DEVOPS":   8082,
	"DATABASE": 8083,
	"FRONTEND": 8084,
	"SECURITY": 8085,
	"BIOLOGY":  8087,
}

// GetMCPServerConfig returns configuration for a specific agent type (BACKEND, BIOLOGY, …).
// Enablement comes from hub config (Settings → MCP / domain packs), not environment variables.
func GetMCPServerConfig(agentType string) *MCPServerConfig {
	key := strings.ToUpper(strings.TrimSpace(agentType))
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
	return &MCPServerConfig{
		Enabled: enabled,
		Port:    port,
		Name:    fmt.Sprintf("%s-agent-mcp", strings.ToLower(key)),
		Version: "1.0.0",
	}
}

// NewMCPServer creates a new MCP server with common configuration
func NewMCPServer(config *MCPServerConfig) (*server.MCPServer, *server.StreamableHTTPServer, error) {
	if !config.Enabled {
		return nil, nil, fmt.Errorf("MCP server disabled for %s", config.Name)
	}

	mcpServer := server.NewMCPServer(config.Name, config.Version)

	httpServer := server.NewStreamableHTTPServer(
		mcpServer,
		server.WithEndpointPath("/mcp"),
	)

	log.Printf("Created MCP server: %s v%s on port %d", config.Name, config.Version, config.Port)

	return mcpServer, httpServer, nil
}

// StartMCPServer starts the MCP server in a goroutine
func StartMCPServer(httpServer *server.StreamableHTTPServer, port int) error {
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
