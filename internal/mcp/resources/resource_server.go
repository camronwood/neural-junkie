package resources

import (
	"context"
	"fmt"
	"log"
	"os"
	"strconv"
	"strings"

	mcp "github.com/camronwood/neural-junkie/internal/mcp"
	"github.com/camronwood/neural-junkie/internal/mcp_export"
	mcpgo "github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server"
)

// ResourceServerConfig holds configuration for the MCP resource server
type ResourceServerConfig struct {
	Enabled bool
	Port    int
	Name    string
	Version string
}

// GetResourceServerConfig returns configuration for the resource server
func GetResourceServerConfig() *ResourceServerConfig {
	enabled := os.Getenv("ENABLE_MCP_RESOURCES") == "true"

	port := 8086
	if portStr := os.Getenv("MCP_RESOURCES_PORT"); portStr != "" {
		if p, err := strconv.Atoi(portStr); err == nil {
			port = p
		}
	}

	return &ResourceServerConfig{
		Enabled: enabled,
		Port:    port,
		Name:    "neural-junkie-resources",
		Version: "1.0.0",
	}
}

// ResourceServer provides MCP tools for accessing exported agent knowledge
type ResourceServer struct {
	mcpServer  *server.MCPServer
	httpServer *server.StreamableHTTPServer
	config     *ResourceServerConfig
	storage    *mcp_export.ExportStorage
}

// NewResourceServer creates a new MCP resource server
func NewResourceServer() (*ResourceServer, error) {
	config := GetResourceServerConfig()
	if !config.Enabled {
		return nil, fmt.Errorf("MCP resource server is disabled")
	}

	mcpServer, httpServer, err := mcp.NewMCPServer(&mcp.MCPServerConfig{
		Enabled: true,
		Port:    config.Port,
		Name:    config.Name,
		Version: config.Version,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to create MCP server: %w", err)
	}

	storage, err := mcp_export.NewExportStorage()
	if err != nil {
		return nil, fmt.Errorf("failed to create export storage: %w", err)
	}

	rs := &ResourceServer{
		mcpServer:  mcpServer,
		httpServer: httpServer,
		config:     config,
		storage:    storage,
	}

	rs.registerTools()
	return rs, nil
}

// Start starts the MCP resource server
func (rs *ResourceServer) Start() error {
	if rs.httpServer == nil {
		return fmt.Errorf("MCP server not configured")
	}
	return mcp.StartMCPServer(rs.httpServer, rs.config.Port)
}

// GetMCPServer returns the underlying MCP server
func (rs *ResourceServer) GetMCPServer() *server.MCPServer {
	return rs.mcpServer
}

func (rs *ResourceServer) registerTools() {
	rs.mcpServer.AddTool(mcp.CreateTool(
		"list_exported_agents",
		"List all available exported agents",
		mcp.CreateStringInputSchema("agent_type", "Filter by agent type (repo, helper, or all)"),
		nil,
	), rs.handleListExportedAgents)

	rs.mcpServer.AddTool(mcp.CreateTool(
		"get_agent_resource",
		"Get a specific resource from an exported agent",
		mcp.CreateMultiStringInputSchema(map[string]string{
			"agent_name":   "Name of the exported agent",
			"agent_type":   "Type of agent (repo or helper)",
			"resource_uri": "URI of the resource to retrieve",
		}),
		nil,
	), rs.handleGetAgentResource)

	rs.mcpServer.AddTool(mcp.CreateTool(
		"get_agent_prompt",
		"Get a pre-configured prompt from an exported agent",
		mcp.CreateMultiStringInputSchema(map[string]string{
			"agent_name":  "Name of the exported agent",
			"agent_type":  "Type of agent (repo or helper)",
			"prompt_name": "Name of the prompt to retrieve",
		}),
		nil,
	), rs.handleGetAgentPrompt)

	rs.mcpServer.AddTool(mcp.CreateTool(
		"recreate_agent",
		"Get instructions to recreate an agent from its export",
		mcp.CreateMultiStringInputSchema(map[string]string{
			"agent_name": "Name of the exported agent",
			"agent_type": "Type of agent (repo or helper)",
		}),
		nil,
	), rs.handleRecreateAgent)

	rs.mcpServer.AddTool(mcp.CreateTool(
		"get_agent_info",
		"Get detailed information about an exported agent",
		mcp.CreateMultiStringInputSchema(map[string]string{
			"agent_name": "Name of the exported agent",
			"agent_type": "Type of agent (repo or helper)",
		}),
		nil,
	), rs.handleGetAgentInfo)

	rs.mcpServer.AddTool(mcp.CreateTool(
		"search_agents",
		"Search for agents by expertise or keywords",
		mcp.CreateStringInputSchema("query", "Search query (expertise, keywords, or description)"),
		nil,
	), rs.handleSearchAgents)

	log.Printf("Registered %d Resource Server MCP tools", len(rs.mcpServer.ListTools()))
}

func (rs *ResourceServer) handleListExportedAgents(ctx context.Context, request mcpgo.CallToolRequest) (*mcpgo.CallToolResult, error) {
	agentType := request.GetString("agent_type", "all")
	if agentType == "" {
		agentType = "all"
	}

	exports, err := rs.storage.ListExports()
	if err != nil {
		return mcp.HandleToolError(fmt.Errorf("failed to list exports: %w", err), "list_exported_agents"), nil
	}

	var filtered []mcp_export.ExportInfo
	for _, export := range exports {
		if agentType == "all" || export.Type == agentType {
			filtered = append(filtered, export)
		}
	}

	var result strings.Builder
	result.WriteString(fmt.Sprintf("=== Exported Agents (%s) ===\n\n", agentType))
	if len(filtered) == 0 {
		result.WriteString("No exported agents found.\n")
	} else {
		for _, export := range filtered {
			result.WriteString(fmt.Sprintf("**%s** (%s)\n", export.Name, export.Type))
			result.WriteString(fmt.Sprintf("- Path: %s\n", export.ExportPath))
			result.WriteString(fmt.Sprintf("- Resources: %d\n", export.ResourceCount))
			result.WriteString(fmt.Sprintf("- Prompts: %d\n", export.PromptCount))
			result.WriteString("\n")
		}
	}
	return mcp.HandleToolSuccess(result.String()), nil
}

func (rs *ResourceServer) handleGetAgentResource(ctx context.Context, request mcpgo.CallToolRequest) (*mcpgo.CallToolResult, error) {
	if err := mcp.ValidateToolInput(request, []string{"agent_name", "agent_type", "resource_uri"}); err != nil {
		return mcp.HandleToolError(err, "get_agent_resource"), nil
	}

	export, err := rs.storage.LoadExport(request.GetString("agent_name", ""), request.GetString("agent_type", ""))
	if err != nil {
		return mcp.HandleToolError(err, "get_agent_resource"), nil
	}

	resource := export.GetResourceByURI(request.GetString("resource_uri", ""))
	if resource == nil {
		return mcp.HandleToolError(fmt.Errorf("resource not found"), "get_agent_resource"), nil
	}

	text := fmt.Sprintf("=== Resource: %s ===\n%s\n", resource.Name, resource.Content)
	return mcp.HandleToolSuccess(text), nil
}

func (rs *ResourceServer) handleGetAgentPrompt(ctx context.Context, request mcpgo.CallToolRequest) (*mcpgo.CallToolResult, error) {
	if err := mcp.ValidateToolInput(request, []string{"agent_name", "agent_type", "prompt_name"}); err != nil {
		return mcp.HandleToolError(err, "get_agent_prompt"), nil
	}

	export, err := rs.storage.LoadExport(request.GetString("agent_name", ""), request.GetString("agent_type", ""))
	if err != nil {
		return mcp.HandleToolError(err, "get_agent_prompt"), nil
	}

	prompt := export.GetPromptByName(request.GetString("prompt_name", ""))
	if prompt == nil {
		return mcp.HandleToolError(fmt.Errorf("prompt not found"), "get_agent_prompt"), nil
	}

	return mcp.HandleToolSuccess(prompt.Prompt), nil
}

func (rs *ResourceServer) handleRecreateAgent(ctx context.Context, request mcpgo.CallToolRequest) (*mcpgo.CallToolResult, error) {
	if err := mcp.ValidateToolInput(request, []string{"agent_name", "agent_type"}); err != nil {
		return mcp.HandleToolError(err, "recreate_agent"), nil
	}

	export, err := rs.storage.LoadExport(request.GetString("agent_name", ""), request.GetString("agent_type", ""))
	if err != nil {
		return mcp.HandleToolError(err, "recreate_agent"), nil
	}

	var result strings.Builder
	result.WriteString(fmt.Sprintf("=== Recreate Agent: %s ===\n\n", export.Agent.Name))
	result.WriteString(export.SystemPrompt)
	return mcp.HandleToolSuccess(result.String()), nil
}

func (rs *ResourceServer) handleGetAgentInfo(ctx context.Context, request mcpgo.CallToolRequest) (*mcpgo.CallToolResult, error) {
	if err := mcp.ValidateToolInput(request, []string{"agent_name", "agent_type"}); err != nil {
		return mcp.HandleToolError(err, "get_agent_info"), nil
	}

	export, err := rs.storage.LoadExport(request.GetString("agent_name", ""), request.GetString("agent_type", ""))
	if err != nil {
		return mcp.HandleToolError(err, "get_agent_info"), nil
	}

	text := fmt.Sprintf("Agent: %s (%s)\nExpertise: %s\nResources: %d\nPrompts: %d\n",
		export.Agent.Name, export.Agent.Type,
		strings.Join(export.Agent.Expertise, ", "),
		export.GetResourceCount(), export.GetPromptCount())
	return mcp.HandleToolSuccess(text), nil
}

func (rs *ResourceServer) handleSearchAgents(ctx context.Context, request mcpgo.CallToolRequest) (*mcpgo.CallToolResult, error) {
	if err := mcp.ValidateToolInput(request, []string{"query"}); err != nil {
		return mcp.HandleToolError(err, "search_agents"), nil
	}

	query := strings.ToLower(request.GetString("query", ""))
	exports, err := rs.storage.ListExports()
	if err != nil {
		return mcp.HandleToolError(err, "search_agents"), nil
	}

	var matches []mcp_export.ExportInfo
	for _, export := range exports {
		if strings.Contains(strings.ToLower(export.Name), query) ||
			strings.Contains(strings.ToLower(export.Description), query) {
			matches = append(matches, export)
		}
	}

	var result strings.Builder
	result.WriteString(fmt.Sprintf("=== Search Results for: %s ===\n", query))
	if len(matches) == 0 {
		result.WriteString("No agents found.\n")
	} else {
		for _, export := range matches {
			result.WriteString(fmt.Sprintf("- %s (%s)\n", export.Name, export.Type))
		}
	}
	return mcp.HandleToolSuccess(result.String()), nil
}
