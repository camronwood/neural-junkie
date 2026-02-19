package resources

import (
	"context"
	"fmt"
	"log"
	"os"
	"strconv"
	"strings"

	mcp "github.com/camronwood/neural-junkie/internal/mcp_disabled"
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

	// Get port from environment
	portStr := os.Getenv("MCP_RESOURCES_PORT")
	port := 8081 // default
	if portStr != "" {
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
	storage    *mcp.ExportStorage
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

	// Initialize export storage
	storage, err := mcp.NewExportStorage()
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

// registerTools registers all resource server MCP tools
func (rs *ResourceServer) registerTools() {
	// Tool 1: list_exported_agents
	rs.mcpServer.AddTool(mcp.CreateTool(
		"list_exported_agents",
		"List all available exported agents",
		mcp.CreateStringInputSchema("agent_type", "Filter by agent type (repo, helper, or all)"),
		rs.handleListExportedAgents,
	), rs.handleListExportedAgents)

	// Tool 2: get_agent_resource
	rs.mcpServer.AddTool(mcp.CreateTool(
		"get_agent_resource",
		"Get a specific resource from an exported agent",
		mcp.CreateMultiStringInputSchema(map[string]string{
			"agent_name":   "Name of the exported agent",
			"agent_type":   "Type of agent (repo or helper)",
			"resource_uri": "URI of the resource to retrieve",
		}),
		rs.handleGetAgentResource,
	), rs.handleGetAgentResource)

	// Tool 3: get_agent_prompt
	rs.mcpServer.AddTool(mcp.CreateTool(
		"get_agent_prompt",
		"Get a pre-configured prompt from an exported agent",
		mcp.CreateMultiStringInputSchema(map[string]string{
			"agent_name":  "Name of the exported agent",
			"agent_type":  "Type of agent (repo or helper)",
			"prompt_name": "Name of the prompt to retrieve",
		}),
		rs.handleGetAgentPrompt,
	), rs.handleGetAgentPrompt)

	// Tool 4: recreate_agent
	rs.mcpServer.AddTool(mcp.CreateTool(
		"recreate_agent",
		"Get instructions to recreate an agent from its export",
		mcp.CreateMultiStringInputSchema(map[string]string{
			"agent_name": "Name of the exported agent",
			"agent_type": "Type of agent (repo or helper)",
		}),
		rs.handleRecreateAgent,
	), rs.handleRecreateAgent)

	// Tool 5: get_agent_info
	rs.mcpServer.AddTool(mcp.CreateTool(
		"get_agent_info",
		"Get detailed information about an exported agent",
		mcp.CreateMultiStringInputSchema(map[string]string{
			"agent_name": "Name of the exported agent",
			"agent_type": "Type of agent (repo or helper)",
		}),
		rs.handleGetAgentInfo,
	), rs.handleGetAgentInfo)

	// Tool 6: search_agents
	rs.mcpServer.AddTool(mcp.CreateTool(
		"search_agents",
		"Search for agents by expertise or keywords",
		mcp.CreateStringInputSchema("query", "Search query (expertise, keywords, or description)"),
		rs.handleSearchAgents,
	), rs.handleSearchAgents)

	log.Printf("Registered %d Resource Server MCP tools", len(rs.mcpServer.ListTools()))
}

// handleListExportedAgents lists all available exported agents
func (rs *ResourceServer) handleListExportedAgents(ctx context.Context, request mcpgo.CallToolRequest) (*mcpgo.CallToolResult, error) {
	if err := mcp.ValidateToolInput(request, []string{"agent_type"}); err != nil {
		return mcp.HandleToolError(err, "list_exported_agents"), nil
	}

	agentType := request.GetString("agent_type", "all")

	exports, err := rs.storage.ListExports()
	if err != nil {
		return mcp.HandleToolError(fmt.Errorf("failed to list exports: %w", err), "list_exported_agents"), nil
	}

	// Filter by agent type if specified
	var filtered []mcp.ExportInfo
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
			result.WriteString(fmt.Sprintf("- Size: %d bytes\n", export.FileSize))
			result.WriteString(fmt.Sprintf("- Last Modified: %s\n", export.LastModified.Format("2006-01-02 15:04:05")))
			if export.Description != "" {
				result.WriteString(fmt.Sprintf("- Description: %s\n", export.Description))
			}
			result.WriteString("\n")
		}
	}

	return mcp.HandleToolSuccess(result.String()), nil
}

// handleGetAgentResource retrieves a specific resource from an exported agent
func (rs *ResourceServer) handleGetAgentResource(ctx context.Context, request mcpgo.CallToolRequest) (*mcpgo.CallToolResult, error) {
	if err := mcp.ValidateToolInput(request, []string{"agent_name", "agent_type", "resource_uri"}); err != nil {
		return mcp.HandleToolError(err, "get_agent_resource"), nil
	}

	agentName := request.GetString("agent_name", "")
	agentType := request.GetString("agent_type", "")
	resourceURI := request.GetString("resource_uri", "")

	// Load the export
	export, err := rs.storage.LoadExport(agentName, agentType)
	if err != nil {
		return mcp.HandleToolError(fmt.Errorf("failed to load export: %w", err), "get_agent_resource"), nil
	}

	// Find the resource
	resource := export.GetResourceByURI(resourceURI)
	if resource == nil {
		return mcp.HandleToolError(fmt.Errorf("resource not found: %s", resourceURI), "get_agent_resource"), nil
	}

	var result strings.Builder
	result.WriteString(fmt.Sprintf("=== Resource: %s ===\n", resource.Name))
	result.WriteString(fmt.Sprintf("URI: %s\n", resource.URI))
	result.WriteString(fmt.Sprintf("MIME Type: %s\n", resource.MimeType))
	result.WriteString(fmt.Sprintf("Size: %d bytes\n\n", resource.Size))
	result.WriteString("Content:\n")
	result.WriteString(resource.Content)

	return mcp.HandleToolSuccess(result.String()), nil
}

// handleGetAgentPrompt retrieves a pre-configured prompt from an exported agent
func (rs *ResourceServer) handleGetAgentPrompt(ctx context.Context, request mcpgo.CallToolRequest) (*mcpgo.CallToolResult, error) {
	if err := mcp.ValidateToolInput(request, []string{"agent_name", "agent_type", "prompt_name"}); err != nil {
		return mcp.HandleToolError(err, "get_agent_prompt"), nil
	}

	agentName := request.GetString("agent_name", "")
	agentType := request.GetString("agent_type", "")
	promptName := request.GetString("prompt_name", "")

	// Load the export
	export, err := rs.storage.LoadExport(agentName, agentType)
	if err != nil {
		return mcp.HandleToolError(fmt.Errorf("failed to load export: %w", err), "get_agent_prompt"), nil
	}

	// Find the prompt
	prompt := export.GetPromptByName(promptName)
	if prompt == nil {
		return mcp.HandleToolError(fmt.Errorf("prompt not found: %s", promptName), "get_agent_prompt"), nil
	}

	var result strings.Builder
	result.WriteString(fmt.Sprintf("=== Prompt: %s ===\n", prompt.Name))
	result.WriteString(fmt.Sprintf("Description: %s\n", prompt.Description))

	if len(prompt.Arguments) > 0 {
		result.WriteString("\nArguments:\n")
		for _, arg := range prompt.Arguments {
			required := ""
			if arg.Required {
				required = " (required)"
			}
			result.WriteString(fmt.Sprintf("- %s (%s)%s: %s\n", arg.Name, arg.Type, required, arg.Description))
		}
	}

	if len(prompt.Resources) > 0 {
		result.WriteString("\nResources:\n")
		for _, resource := range prompt.Resources {
			result.WriteString(fmt.Sprintf("- %s\n", resource))
		}
	}

	result.WriteString("\nPrompt Template:\n")
	result.WriteString(prompt.Prompt)

	return mcp.HandleToolSuccess(result.String()), nil
}

// handleRecreateAgent provides instructions to recreate an agent from its export
func (rs *ResourceServer) handleRecreateAgent(ctx context.Context, request mcpgo.CallToolRequest) (*mcpgo.CallToolResult, error) {
	if err := mcp.ValidateToolInput(request, []string{"agent_name", "agent_type"}); err != nil {
		return mcp.HandleToolError(err, "recreate_agent"), nil
	}

	agentName := request.GetString("agent_name", "")
	agentType := request.GetString("agent_type", "")

	// Load the export
	export, err := rs.storage.LoadExport(agentName, agentType)
	if err != nil {
		return mcp.HandleToolError(fmt.Errorf("failed to load export: %w", err), "recreate_agent"), nil
	}

	var result strings.Builder
	result.WriteString(fmt.Sprintf("=== Recreate Agent: %s ===\n\n", export.Agent.Name))

	result.WriteString("## Agent Information\n")
	result.WriteString(fmt.Sprintf("- Name: %s\n", export.Agent.Name))
	result.WriteString(fmt.Sprintf("- Type: %s\n", export.Agent.Type))
	result.WriteString(fmt.Sprintf("- Expertise: %s\n", strings.Join(export.Agent.Expertise, ", ")))
	if export.Agent.Description != "" {
		result.WriteString(fmt.Sprintf("- Description: %s\n", export.Agent.Description))
	}

	result.WriteString("\n## Recreation Instructions\n")

	if export.Agent.Type == "repo" {
		result.WriteString("### For Repository Agent:\n")
		result.WriteString("1. Ensure you have access to the repository at the specified path\n")
		result.WriteString("2. Use the system prompt provided below\n")
		result.WriteString("3. Load the architecture and source file resources\n")
		result.WriteString("4. Configure the agent with the expertise areas listed\n")
	} else if export.Agent.Type == "helper" {
		result.WriteString("### For Helper Agent:\n")
		result.WriteString("1. Set up the knowledge base directory structure\n")
		result.WriteString("2. Use the system prompt provided below\n")
		result.WriteString("3. Load the knowledge documents and configuration\n")
		result.WriteString("4. Configure the agent with the expertise areas and keywords\n")
	}

	result.WriteString("\n## System Prompt\n")
	result.WriteString(export.SystemPrompt)

	result.WriteString("\n## Available Resources\n")
	for _, resource := range export.Resources {
		result.WriteString(fmt.Sprintf("- %s (%s): %s\n", resource.URI, resource.MimeType, resource.Name))
	}

	result.WriteString("\n## Available Prompts\n")
	for _, prompt := range export.Prompts {
		result.WriteString(fmt.Sprintf("- %s: %s\n", prompt.Name, prompt.Description))
	}

	return mcp.HandleToolSuccess(result.String()), nil
}

// handleGetAgentInfo provides detailed information about an exported agent
func (rs *ResourceServer) handleGetAgentInfo(ctx context.Context, request mcpgo.CallToolRequest) (*mcpgo.CallToolResult, error) {
	if err := mcp.ValidateToolInput(request, []string{"agent_name", "agent_type"}); err != nil {
		return mcp.HandleToolError(err, "get_agent_info"), nil
	}

	agentName := request.GetString("agent_name", "")
	agentType := request.GetString("agent_type", "")

	// Load the export
	export, err := rs.storage.LoadExport(agentName, agentType)
	if err != nil {
		return mcp.HandleToolError(fmt.Errorf("failed to load export: %w", err), "get_agent_info"), nil
	}

	var result strings.Builder
	result.WriteString(fmt.Sprintf("=== Agent Information: %s ===\n\n", export.Agent.Name))

	result.WriteString("## Basic Information\n")
	result.WriteString(fmt.Sprintf("- Name: %s\n", export.Agent.Name))
	result.WriteString(fmt.Sprintf("- Type: %s\n", export.Agent.Type))
	result.WriteString(fmt.Sprintf("- Version: %s\n", export.Version))
	result.WriteString(fmt.Sprintf("- Exported: %s\n", export.ExportedAt.Format("2006-01-02 15:04:05")))

	if export.Agent.Description != "" {
		result.WriteString(fmt.Sprintf("- Description: %s\n", export.Agent.Description))
	}

	result.WriteString("\n## Expertise\n")
	for _, expertise := range export.Agent.Expertise {
		result.WriteString(fmt.Sprintf("- %s\n", expertise))
	}

	if len(export.Agent.Keywords) > 0 {
		result.WriteString("\n## Keywords\n")
		for _, keyword := range export.Agent.Keywords {
			result.WriteString(fmt.Sprintf("- %s\n", keyword))
		}
	}

	result.WriteString("\n## Statistics\n")
	result.WriteString(fmt.Sprintf("- Total Resources: %d\n", export.GetResourceCount()))
	result.WriteString(fmt.Sprintf("- Total Prompts: %d\n", export.GetPromptCount()))
	result.WriteString(fmt.Sprintf("- Total Size: %d bytes\n", export.GetTotalSize()))

	result.WriteString("\n## Resources\n")
	for _, resource := range export.Resources {
		result.WriteString(fmt.Sprintf("- %s (%s): %d bytes\n", resource.URI, resource.MimeType, resource.Size))
	}

	result.WriteString("\n## Prompts\n")
	for _, prompt := range export.Prompts {
		result.WriteString(fmt.Sprintf("- %s: %s\n", prompt.Name, prompt.Description))
	}

	return mcp.HandleToolSuccess(result.String()), nil
}

// handleSearchAgents searches for agents by expertise or keywords
func (rs *ResourceServer) handleSearchAgents(ctx context.Context, request mcpgo.CallToolRequest) (*mcpgo.CallToolResult, error) {
	if err := mcp.ValidateToolInput(request, []string{"query"}); err != nil {
		return mcp.HandleToolError(err, "search_agents"), nil
	}

	query := strings.ToLower(request.GetString("query", ""))

	exports, err := rs.storage.ListExports()
	if err != nil {
		return mcp.HandleToolError(fmt.Errorf("failed to list exports: %w", err), "search_agents"), nil
	}

	var matches []mcp.ExportInfo
	for _, export := range exports {
		// Search in name, description, and expertise
		if strings.Contains(strings.ToLower(export.Name), query) ||
			strings.Contains(strings.ToLower(export.Description), query) {
			matches = append(matches, export)
			continue
		}

		// Load full export to search expertise and keywords
		fullExport, err := rs.storage.LoadExport(export.Name, export.Type)
		if err != nil {
			continue
		}

		// Search in expertise
		for _, expertise := range fullExport.Agent.Expertise {
			if strings.Contains(strings.ToLower(expertise), query) {
				matches = append(matches, export)
				break
			}
		}

		// Search in keywords
		for _, keyword := range fullExport.Agent.Keywords {
			if strings.Contains(strings.ToLower(keyword), query) {
				matches = append(matches, export)
				break
			}
		}
	}

	var result strings.Builder
	result.WriteString(fmt.Sprintf("=== Search Results for: %s ===\n\n", query))

	if len(matches) == 0 {
		result.WriteString("No agents found matching your query.\n")
	} else {
		result.WriteString(fmt.Sprintf("Found %d matching agents:\n\n", len(matches)))
		for _, export := range matches {
			result.WriteString(fmt.Sprintf("**%s** (%s)\n", export.Name, export.Type))
			result.WriteString(fmt.Sprintf("- Resources: %d\n", export.ResourceCount))
			result.WriteString(fmt.Sprintf("- Prompts: %d\n", export.PromptCount))
			if export.Description != "" {
				result.WriteString(fmt.Sprintf("- Description: %s\n", export.Description))
			}
			result.WriteString("\n")
		}
	}

	return mcp.HandleToolSuccess(result.String()), nil
}
