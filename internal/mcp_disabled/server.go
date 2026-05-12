package mcp

import (
	"fmt"
	"log"
	"os"
	"strconv"

	mcp "github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server"
)

// MCPServerConfig holds configuration for MCP servers
type MCPServerConfig struct {
	Enabled bool
	Port    int
	Name    string
	Version string
}

// GetMCPServerConfig returns configuration for a specific agent type
func GetMCPServerConfig(agentType string) *MCPServerConfig {
	enabled := os.Getenv("ENABLE_MCP") == "true"

	// Check agent-specific enable flag
	agentFlag := fmt.Sprintf("ENABLE_%s_MCP", agentType)
	agentEnabled := os.Getenv(agentFlag) == "true"

	// Get port from environment
	portFlag := fmt.Sprintf("MCP_%s_PORT", agentType)
	portStr := os.Getenv(portFlag)
	port := 18765 // default
	if portStr != "" {
		if p, err := strconv.Atoi(portStr); err == nil {
			port = p
		}
	}

	return &MCPServerConfig{
		Enabled: enabled && agentEnabled,
		Port:    port,
		Name:    fmt.Sprintf("%s-agent-mcp", agentType),
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

// CreateTool creates a standardized MCP tool with error handling
func CreateTool(name, description string, inputSchema mcp.ToolInputSchema, handler server.ToolHandlerFunc) mcp.Tool {
	return mcp.Tool{
		Name:        name,
		Description: description,
		InputSchema: inputSchema,
	}
}

// CreateStringInputSchema creates a simple string input schema
func CreateStringInputSchema(paramName, description string) mcp.ToolInputSchema {
	return mcp.ToolInputSchema{
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
func CreateMultiStringInputSchema(params map[string]string) mcp.ToolInputSchema {
	properties := make(map[string]any)
	required := make([]string, 0, len(params))

	for name, description := range params {
		properties[name] = map[string]any{
			"type":        "string",
			"description": description,
		}
		required = append(required, name)
	}

	return mcp.ToolInputSchema{
		Type:       "object",
		Properties: properties,
		Required:   required,
	}
}

// HandleToolError creates a standardized error response
func HandleToolError(err error, toolName string) *mcp.CallToolResult {
	log.Printf("Tool %s error: %v", toolName, err)
	return &mcp.CallToolResult{
		Content: []mcp.Content{
			mcp.TextContent{
				Type: "text",
				Text: fmt.Sprintf("Error in %s: %v", toolName, err),
			},
		},
		IsError: true,
	}
}

// HandleToolSuccess creates a standardized success response
func HandleToolSuccess(result string) *mcp.CallToolResult {
	return &mcp.CallToolResult{
		Content: []mcp.Content{
			mcp.TextContent{
				Type: "text",
				Text: result,
			},
		},
	}
}

// ValidateToolInput validates required string parameters
func ValidateToolInput(request mcp.CallToolRequest, requiredParams []string) error {
	for _, param := range requiredParams {
		if request.GetString(param, "") == "" {
			return fmt.Errorf("missing required parameter: %s", param)
		}
	}
	return nil
}
