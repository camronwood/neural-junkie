package incident

import (
	"context"
	"fmt"
	"log"
	"strings"

	"github.com/camronwood/neural-junkie/internal/config"
	mcp "github.com/camronwood/neural-junkie/internal/mcp"
	mcpgo "github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server"
)

// IncidentMCP provides MCP tools for incident triage and Jira Cloud.
type IncidentMCP struct {
	mcpServer  *server.MCPServer
	httpServer *server.StreamableHTTPServer
	config     *mcp.MCPServerConfig
}

// NewIncidentMCP creates a new incident MCP server.
func NewIncidentMCP() (*IncidentMCP, error) {
	cfg := mcp.GetMCPServerConfig("INCIDENT")
	mcpServer, httpServer, err := mcp.NewMCPServer(cfg)
	if err != nil {
		return nil, fmt.Errorf("failed to create MCP server: %w", err)
	}
	i := &IncidentMCP{mcpServer: mcpServer, httpServer: httpServer, config: cfg}
	i.registerTools()
	return i, nil
}

func (i *IncidentMCP) Start() error {
	return mcp.StartMCPServer(i.httpServer, i.config.Port)
}

func (i *IncidentMCP) GetMCPServer() *server.MCPServer {
	return i.mcpServer
}

func jiraSettings() config.JiraConfig {
	if cfg := mcp.AppConfig(); cfg != nil {
		return cfg.JiraSettings()
	}
	return config.JiraConfig{}
}

func (i *IncidentMCP) registerTools() {
	i.mcpServer.AddTool(mcp.CreateTool(
		"jira_get_issue",
		"Fetch a Jira issue by key (e.g. PROJ-123).",
		mcp.CreateStringInputSchema("issue_key", "Jira issue key"),
		nil,
	), i.handleGetIssue)

	i.mcpServer.AddTool(mcp.CreateTool(
		"jira_search_issues",
		"Search Jira issues with JQL.",
		mcp.CreateObjectInputSchema(map[string]interface{}{
			"jql": map[string]interface{}{"type": "string", "description": "JQL query"},
			"max_results": map[string]interface{}{"type": "number", "description": "Max results (default 20)"},
		}, []string{"jql"}),
		nil,
	), i.handleSearchIssues)

	i.mcpServer.AddTool(mcp.CreateTool(
		"jira_add_comment",
		"Add a triage or status comment to a Jira issue.",
		mcp.CreateObjectInputSchema(map[string]interface{}{
			"issue_key": map[string]interface{}{"type": "string"},
			"body":      map[string]interface{}{"type": "string", "description": "Comment text"},
		}, []string{"issue_key", "body"}),
		nil,
	), i.handleAddComment)

	i.mcpServer.AddTool(mcp.CreateTool(
		"jira_summarize_issue",
		"Return a concise triage summary for a Jira issue.",
		mcp.CreateStringInputSchema("issue_key", "Jira issue key"),
		nil,
	), i.handleSummarizeIssue)

	log.Printf("Registered %d incident MCP tools", len(i.mcpServer.ListTools()))
}

func (i *IncidentMCP) handleGetIssue(ctx context.Context, request mcpgo.CallToolRequest) (*mcpgo.CallToolResult, error) {
	client, err := clientFromHub()
	if err != nil {
		return mcp.HandleToolError(err, "jira_get_issue"), nil
	}
	key := strings.TrimSpace(request.GetString("issue_key", ""))
	out, err := client.GetIssue(ctx, key)
	if err != nil {
		return mcp.HandleToolError(err, "jira_get_issue"), nil
	}
	return mcp.HandleToolSuccess(out), nil
}

func (i *IncidentMCP) handleSearchIssues(ctx context.Context, request mcpgo.CallToolRequest) (*mcpgo.CallToolResult, error) {
	client, err := clientFromHub()
	if err != nil {
		return mcp.HandleToolError(err, "jira_search_issues"), nil
	}
	jql := strings.TrimSpace(request.GetString("jql", ""))
	if jql == "" {
		settings := jiraSettings()
		if pk := strings.TrimSpace(settings.DefaultProjectKey); pk != "" {
			jql = fmt.Sprintf("project = %s AND status != Done ORDER BY updated DESC", pk)
		}
	}
	max := 20
	if raw, ok := request.GetArguments()["max_results"]; ok {
		switch v := raw.(type) {
		case float64:
			max = int(v)
		case int:
			max = v
		}
	}
	out, err := client.SearchIssues(ctx, jql, max)
	if err != nil {
		return mcp.HandleToolError(err, "jira_search_issues"), nil
	}
	return mcp.HandleToolSuccess(out), nil
}

func (i *IncidentMCP) handleAddComment(ctx context.Context, request mcpgo.CallToolRequest) (*mcpgo.CallToolResult, error) {
	client, err := clientFromHub()
	if err != nil {
		return mcp.HandleToolError(err, "jira_add_comment"), nil
	}
	key := strings.TrimSpace(request.GetString("issue_key", ""))
	body := strings.TrimSpace(request.GetString("body", ""))
	out, err := client.AddComment(ctx, key, body)
	if err != nil {
		return mcp.HandleToolError(err, "jira_add_comment"), nil
	}
	return mcp.HandleToolSuccess(out), nil
}

func (i *IncidentMCP) handleSummarizeIssue(ctx context.Context, request mcpgo.CallToolRequest) (*mcpgo.CallToolResult, error) {
	client, err := clientFromHub()
	if err != nil {
		return mcp.HandleToolError(err, "jira_summarize_issue"), nil
	}
	key := strings.TrimSpace(request.GetString("issue_key", ""))
	out, err := client.SummarizeIssue(ctx, key)
	if err != nil {
		return mcp.HandleToolError(err, "jira_summarize_issue"), nil
	}
	return mcp.HandleToolSuccess(out), nil
}
