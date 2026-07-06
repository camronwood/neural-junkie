package incident

import (
	"context"
	"fmt"
	"log"
	"strings"

	"github.com/camronwood/neural-junkie/internal/config"
	"github.com/camronwood/neural-junkie/internal/incidentsidecar"
	"github.com/camronwood/neural-junkie/internal/integrations/ticketing"
	mcp "github.com/camronwood/neural-junkie/internal/mcp"
	mcpgo "github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server"
)

// IncidentMCP provides MCP tools for incident triage and multi-provider ticketing.
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
	// Jira-specific (v1 compat)
	i.mcpServer.AddTool(mcp.CreateTool("jira_get_issue", "Fetch a Jira issue by key (e.g. PROJ-123).", mcp.CreateStringInputSchema("issue_key", "Jira issue key"), nil), i.handleGetIssue)
	i.mcpServer.AddTool(mcp.CreateTool("jira_search_issues", "Search Jira issues with JQL.", mcp.CreateObjectInputSchema(map[string]interface{}{
		"jql": map[string]interface{}{"type": "string"}, "max_results": map[string]interface{}{"type": "number"},
	}, []string{"jql"}), nil), i.handleSearchIssues)
	i.mcpServer.AddTool(mcp.CreateTool("jira_add_comment", "Add a triage comment to a Jira issue (requires write mode + approval).", mcp.CreateObjectInputSchema(map[string]interface{}{
		"issue_key": map[string]interface{}{"type": "string"}, "body": map[string]interface{}{"type": "string"},
	}, []string{"issue_key", "body"}), nil), i.handleAddComment)
	i.mcpServer.AddTool(mcp.CreateTool("jira_summarize_issue", "Return a concise triage summary for a Jira issue.", mcp.CreateStringInputSchema("issue_key", "Jira issue key"), nil), i.handleSummarizeIssue)
	i.mcpServer.AddTool(mcp.CreateTool("jira_create_issue", "Create a Jira issue (requires write mode + approval).", mcp.CreateObjectInputSchema(map[string]interface{}{
		"summary": map[string]interface{}{"type": "string"}, "description": map[string]interface{}{"type": "string"},
		"priority": map[string]interface{}{"type": "string"}, "issue_type": map[string]interface{}{"type": "string"},
	}, []string{"summary"}), nil), i.handleCreateIssue)
	i.mcpServer.AddTool(mcp.CreateTool("jira_assign_issue", "Assign a Jira issue (requires write mode + approval).", mcp.CreateObjectInputSchema(map[string]interface{}{
		"issue_key": map[string]interface{}{"type": "string"}, "assignee": map[string]interface{}{"type": "string"},
	}, []string{"issue_key", "assignee"}), nil), i.handleAssignIssue)
	i.mcpServer.AddTool(mcp.CreateTool("jira_transition_issue", "Transition Jira issue status (requires write mode + approval).", mcp.CreateObjectInputSchema(map[string]interface{}{
		"issue_key": map[string]interface{}{"type": "string"}, "status": map[string]interface{}{"type": "string"},
	}, []string{"issue_key", "status"}), nil), i.handleTransitionIssue)
	i.mcpServer.AddTool(mcp.CreateTool("jira_set_priority", "Set Jira issue priority (requires write mode + approval).", mcp.CreateObjectInputSchema(map[string]interface{}{
		"issue_key": map[string]interface{}{"type": "string"}, "priority": map[string]interface{}{"type": "string"},
	}, []string{"issue_key", "priority"}), nil), i.handleSetPriority)

	// Unified ticketing
	providerSchema := map[string]interface{}{"provider": map[string]interface{}{"type": "string", "description": "jira|github|linear"}}
	i.mcpServer.AddTool(mcp.CreateTool("ticket_get", "Fetch ticket from default or specified provider.", mcp.CreateObjectInputSchema(map[string]interface{}{
		"provider": providerSchema["provider"], "id": map[string]interface{}{"type": "string"},
	}, []string{"id"}), nil), i.handleTicketGet)
	i.mcpServer.AddTool(mcp.CreateTool("ticket_search", "Search tickets.", mcp.CreateObjectInputSchema(map[string]interface{}{
		"provider": providerSchema["provider"], "query": map[string]interface{}{"type": "string"}, "max_results": map[string]interface{}{"type": "number"},
	}, []string{"query"}), nil), i.handleTicketSearch)
	i.mcpServer.AddTool(mcp.CreateTool("ticket_comment", "Add comment to ticket (write mode + approval).", mcp.CreateObjectInputSchema(map[string]interface{}{
		"provider": providerSchema["provider"], "id": map[string]interface{}{"type": "string"}, "body": map[string]interface{}{"type": "string"},
	}, []string{"id", "body"}), nil), i.handleTicketComment)
	i.mcpServer.AddTool(mcp.CreateTool("ticket_create", "Create ticket (write mode + approval).", mcp.CreateObjectInputSchema(map[string]interface{}{
		"provider": providerSchema["provider"], "title": map[string]interface{}{"type": "string"}, "description": map[string]interface{}{"type": "string"},
		"priority": map[string]interface{}{"type": "string"},
	}, []string{"title"}), nil), i.handleTicketCreate)
	i.mcpServer.AddTool(mcp.CreateTool("ticket_transition", "Transition ticket status (write mode + approval).", mcp.CreateObjectInputSchema(map[string]interface{}{
		"provider": providerSchema["provider"], "id": map[string]interface{}{"type": "string"}, "status": map[string]interface{}{"type": "string"},
	}, []string{"id", "status"}), nil), i.handleTicketTransition)
	i.mcpServer.AddTool(mcp.CreateTool("ticket_assign", "Assign ticket (write mode + approval).", mcp.CreateObjectInputSchema(map[string]interface{}{
		"provider": providerSchema["provider"], "id": map[string]interface{}{"type": "string"}, "assignee": map[string]interface{}{"type": "string"},
	}, []string{"id", "assignee"}), nil), i.handleTicketAssign)
	i.mcpServer.AddTool(mcp.CreateTool("ticket_set_priority", "Set ticket priority (write mode + approval).", mcp.CreateObjectInputSchema(map[string]interface{}{
		"provider": providerSchema["provider"], "id": map[string]interface{}{"type": "string"}, "priority": map[string]interface{}{"type": "string"},
	}, []string{"id", "priority"}), nil), i.handleTicketSetPriority)

	// Alert sources (read-only)
	i.mcpServer.AddTool(mcp.CreateTool("pagerduty_get_incident", "Fetch PagerDuty incident by ID.", mcp.CreateStringInputSchema("incident_id", "PagerDuty incident ID"), nil), i.handlePagerDutyGet)
	i.mcpServer.AddTool(mcp.CreateTool("pagerduty_list_incidents", "List triggered/acknowledged PagerDuty incidents.", mcp.CreateObjectInputSchema(map[string]interface{}{}, nil), nil), i.handlePagerDutyList)
	i.mcpServer.AddTool(mcp.CreateTool("sentry_get_issue", "Fetch Sentry issue by ID.", mcp.CreateStringInputSchema("issue_id", "Sentry issue ID"), nil), i.handleSentryGetIssue)
	i.mcpServer.AddTool(mcp.CreateTool("sentry_get_event", "Fetch latest Sentry event for an issue.", mcp.CreateStringInputSchema("issue_id", "Sentry issue ID"), nil), i.handleSentryGetEvent)

	// Sidecar-backed trace + postmortem
	i.mcpServer.AddTool(mcp.CreateTool("incident_parse_stack_trace", "Parse stack trace into frames, suspect files, repro steps, severity hint.", mcp.CreateStringInputSchema("trace", "Raw stack trace text"), nil), i.handleParseTrace)
	i.mcpServer.AddTool(mcp.CreateTool("incident_link_code_locations", "Return suspect file:line list from stack trace for BackendEngineer consult.", mcp.CreateStringInputSchema("trace", "Raw stack trace text"), nil), i.handleLinkCode)
	i.mcpServer.AddTool(mcp.CreateTool("incident_generate_postmortem", "Generate postmortem draft from incident metadata and timeline.", mcp.CreateObjectInputSchema(map[string]interface{}{
		"issue_key": map[string]interface{}{"type": "string"}, "timeline_markdown": map[string]interface{}{"type": "string"},
		"severity": map[string]interface{}{"type": "string"}, "summary": map[string]interface{}{"type": "string"},
	}, []string{"issue_key"}), nil), i.handleGeneratePostmortem)

	log.Printf("Registered %d incident MCP tools", len(i.mcpServer.ListTools()))
}

// --- Jira handlers ---

func (i *IncidentMCP) handleGetIssue(ctx context.Context, request mcpgo.CallToolRequest) (*mcpgo.CallToolResult, error) {
	client, err := clientFromHub()
	if err != nil {
		return mcp.HandleToolError(err, "jira_get_issue"), nil
	}
	out, err := client.GetIssue(ctx, strings.TrimSpace(request.GetString("issue_key", "")))
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
		if pk := strings.TrimSpace(jiraSettings().DefaultProjectKey); pk != "" {
			jql = fmt.Sprintf("project = %s AND status != Done ORDER BY updated DESC", pk)
		}
	}
	max := intArg(request, "max_results", 20)
	out, err := client.SearchIssues(ctx, jql, max)
	if err != nil {
		return mcp.HandleToolError(err, "jira_search_issues"), nil
	}
	return mcp.HandleToolSuccess(out), nil
}

func (i *IncidentMCP) handleAddComment(ctx context.Context, request mcpgo.CallToolRequest) (*mcpgo.CallToolResult, error) {
	if err := requireWrite("jira_add_comment"); err != nil {
		return mcp.HandleToolError(err, "jira_add_comment"), nil
	}
	client, err := clientFromHub()
	if err != nil {
		return mcp.HandleToolError(err, "jira_add_comment"), nil
	}
	out, err := client.AddComment(ctx, request.GetString("issue_key", ""), request.GetString("body", ""))
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
	out, err := client.SummarizeIssue(ctx, request.GetString("issue_key", ""))
	if err != nil {
		return mcp.HandleToolError(err, "jira_summarize_issue"), nil
	}
	return mcp.HandleToolSuccess(out), nil
}

func (i *IncidentMCP) handleCreateIssue(ctx context.Context, request mcpgo.CallToolRequest) (*mcpgo.CallToolResult, error) {
	if err := requireWrite("jira_create_issue"); err != nil {
		return mcp.HandleToolError(err, "jira_create_issue"), nil
	}
	client, err := clientFromHub()
	if err != nil {
		return mcp.HandleToolError(err, "jira_create_issue"), nil
	}
	out, err := client.CreateIssue(ctx, CreateIssueRequest{
		ProjectKey: jiraSettings().DefaultProjectKey,
		Summary:    request.GetString("summary", ""),
		Description: request.GetString("description", ""),
		Priority:    request.GetString("priority", ""),
		IssueType:   request.GetString("issue_type", ""),
	})
	if err != nil {
		return mcp.HandleToolError(err, "jira_create_issue"), nil
	}
	return mcp.HandleToolSuccess(out), nil
}

func (i *IncidentMCP) handleAssignIssue(ctx context.Context, request mcpgo.CallToolRequest) (*mcpgo.CallToolResult, error) {
	if err := requireWrite("jira_assign_issue"); err != nil {
		return mcp.HandleToolError(err, "jira_assign_issue"), nil
	}
	client, err := clientFromHub()
	if err != nil {
		return mcp.HandleToolError(err, "jira_assign_issue"), nil
	}
	out, err := client.AssignIssue(ctx, request.GetString("issue_key", ""), request.GetString("assignee", ""))
	if err != nil {
		return mcp.HandleToolError(err, "jira_assign_issue"), nil
	}
	return mcp.HandleToolSuccess(out), nil
}

func (i *IncidentMCP) handleTransitionIssue(ctx context.Context, request mcpgo.CallToolRequest) (*mcpgo.CallToolResult, error) {
	if err := requireWrite("jira_transition_issue"); err != nil {
		return mcp.HandleToolError(err, "jira_transition_issue"), nil
	}
	client, err := clientFromHub()
	if err != nil {
		return mcp.HandleToolError(err, "jira_transition_issue"), nil
	}
	out, err := client.TransitionIssue(ctx, request.GetString("issue_key", ""), request.GetString("status", ""))
	if err != nil {
		return mcp.HandleToolError(err, "jira_transition_issue"), nil
	}
	return mcp.HandleToolSuccess(out), nil
}

func (i *IncidentMCP) handleSetPriority(ctx context.Context, request mcpgo.CallToolRequest) (*mcpgo.CallToolResult, error) {
	if err := requireWrite("jira_set_priority"); err != nil {
		return mcp.HandleToolError(err, "jira_set_priority"), nil
	}
	client, err := clientFromHub()
	if err != nil {
		return mcp.HandleToolError(err, "jira_set_priority"), nil
	}
	out, err := client.SetPriority(ctx, request.GetString("issue_key", ""), request.GetString("priority", ""))
	if err != nil {
		return mcp.HandleToolError(err, "jira_set_priority"), nil
	}
	return mcp.HandleToolSuccess(out), nil
}

// --- Unified ticket handlers ---

func (i *IncidentMCP) handleTicketGet(ctx context.Context, request mcpgo.CallToolRequest) (*mcpgo.CallToolResult, error) {
	p, err := providerFromRequest(request.GetString("provider", ""))
	if err != nil {
		return mcp.HandleToolError(err, "ticket_get"), nil
	}
	out, err := p.Get(ctx, request.GetString("id", ""))
	if err != nil {
		return mcp.HandleToolError(err, "ticket_get"), nil
	}
	return mcp.HandleToolSuccess(out), nil
}

func (i *IncidentMCP) handleTicketSearch(ctx context.Context, request mcpgo.CallToolRequest) (*mcpgo.CallToolResult, error) {
	p, err := providerFromRequest(request.GetString("provider", ""))
	if err != nil {
		return mcp.HandleToolError(err, "ticket_search"), nil
	}
	out, err := p.Search(ctx, request.GetString("query", ""), intArg(request, "max_results", 20))
	if err != nil {
		return mcp.HandleToolError(err, "ticket_search"), nil
	}
	return mcp.HandleToolSuccess(out), nil
}

func (i *IncidentMCP) handleTicketComment(ctx context.Context, request mcpgo.CallToolRequest) (*mcpgo.CallToolResult, error) {
	if err := requireWrite("ticket_comment"); err != nil {
		return mcp.HandleToolError(err, "ticket_comment"), nil
	}
	p, err := providerFromRequest(request.GetString("provider", ""))
	if err != nil {
		return mcp.HandleToolError(err, "ticket_comment"), nil
	}
	out, err := p.Comment(ctx, request.GetString("id", ""), request.GetString("body", ""))
	if err != nil {
		return mcp.HandleToolError(err, "ticket_comment"), nil
	}
	return mcp.HandleToolSuccess(out), nil
}

func (i *IncidentMCP) handleTicketCreate(ctx context.Context, request mcpgo.CallToolRequest) (*mcpgo.CallToolResult, error) {
	if err := requireWrite("ticket_create"); err != nil {
		return mcp.HandleToolError(err, "ticket_create"), nil
	}
	p, err := providerFromRequest(request.GetString("provider", ""))
	if err != nil {
		return mcp.HandleToolError(err, "ticket_create"), nil
	}
	out, err := p.Create(ctx, ticketing.CreateRequest{
		Title: request.GetString("title", ""), Description: request.GetString("description", ""),
		Priority: request.GetString("priority", ""),
	})
	if err != nil {
		return mcp.HandleToolError(err, "ticket_create"), nil
	}
	return mcp.HandleToolSuccess(out), nil
}

func (i *IncidentMCP) handleTicketTransition(ctx context.Context, request mcpgo.CallToolRequest) (*mcpgo.CallToolResult, error) {
	if err := requireWrite("ticket_transition"); err != nil {
		return mcp.HandleToolError(err, "ticket_transition"), nil
	}
	p, err := providerFromRequest(request.GetString("provider", ""))
	if err != nil {
		return mcp.HandleToolError(err, "ticket_transition"), nil
	}
	out, err := p.Transition(ctx, request.GetString("id", ""), request.GetString("status", ""))
	if err != nil {
		return mcp.HandleToolError(err, "ticket_transition"), nil
	}
	return mcp.HandleToolSuccess(out), nil
}

func (i *IncidentMCP) handleTicketAssign(ctx context.Context, request mcpgo.CallToolRequest) (*mcpgo.CallToolResult, error) {
	if err := requireWrite("ticket_assign"); err != nil {
		return mcp.HandleToolError(err, "ticket_assign"), nil
	}
	p, err := providerFromRequest(request.GetString("provider", ""))
	if err != nil {
		return mcp.HandleToolError(err, "ticket_assign"), nil
	}
	out, err := p.Assign(ctx, request.GetString("id", ""), request.GetString("assignee", ""))
	if err != nil {
		return mcp.HandleToolError(err, "ticket_assign"), nil
	}
	return mcp.HandleToolSuccess(out), nil
}

func (i *IncidentMCP) handleTicketSetPriority(ctx context.Context, request mcpgo.CallToolRequest) (*mcpgo.CallToolResult, error) {
	if err := requireWrite("ticket_set_priority"); err != nil {
		return mcp.HandleToolError(err, "ticket_set_priority"), nil
	}
	p, err := providerFromRequest(request.GetString("provider", ""))
	if err != nil {
		return mcp.HandleToolError(err, "ticket_set_priority"), nil
	}
	out, err := p.SetPriority(ctx, request.GetString("id", ""), request.GetString("priority", ""))
	if err != nil {
		return mcp.HandleToolError(err, "ticket_set_priority"), nil
	}
	return mcp.HandleToolSuccess(out), nil
}

// --- PagerDuty / Sentry ---

func (i *IncidentMCP) handlePagerDutyGet(ctx context.Context, request mcpgo.CallToolRequest) (*mcpgo.CallToolResult, error) {
	cfg := mcp.AppConfig()
	if cfg == nil {
		return mcp.HandleToolError(fmt.Errorf("config unavailable"), "pagerduty_get_incident"), nil
	}
	c := ticketing.NewPagerDutyClient(cfg.PagerDutySettings())
	out, err := c.GetIncident(ctx, request.GetString("incident_id", ""))
	if err != nil {
		return mcp.HandleToolError(err, "pagerduty_get_incident"), nil
	}
	return mcp.HandleToolSuccess(out), nil
}

func (i *IncidentMCP) handlePagerDutyList(ctx context.Context, request mcpgo.CallToolRequest) (*mcpgo.CallToolResult, error) {
	cfg := mcp.AppConfig()
	if cfg == nil {
		return mcp.HandleToolError(fmt.Errorf("config unavailable"), "pagerduty_list_incidents"), nil
	}
	c := ticketing.NewPagerDutyClient(cfg.PagerDutySettings())
	out, err := c.ListIncidents(ctx, []string{"triggered", "acknowledged"})
	if err != nil {
		return mcp.HandleToolError(err, "pagerduty_list_incidents"), nil
	}
	return mcp.HandleToolSuccess(out), nil
}

func (i *IncidentMCP) handleSentryGetIssue(ctx context.Context, request mcpgo.CallToolRequest) (*mcpgo.CallToolResult, error) {
	cfg := mcp.AppConfig()
	if cfg == nil {
		return mcp.HandleToolError(fmt.Errorf("config unavailable"), "sentry_get_issue"), nil
	}
	c := ticketing.NewSentryClient(cfg.SentrySettings())
	out, err := c.GetIssue(ctx, request.GetString("issue_id", ""))
	if err != nil {
		return mcp.HandleToolError(err, "sentry_get_issue"), nil
	}
	return mcp.HandleToolSuccess(out), nil
}

func (i *IncidentMCP) handleSentryGetEvent(ctx context.Context, request mcpgo.CallToolRequest) (*mcpgo.CallToolResult, error) {
	cfg := mcp.AppConfig()
	if cfg == nil {
		return mcp.HandleToolError(fmt.Errorf("config unavailable"), "sentry_get_event"), nil
	}
	c := ticketing.NewSentryClient(cfg.SentrySettings())
	out, err := c.GetLatestEvent(ctx, request.GetString("issue_id", ""))
	if err != nil {
		return mcp.HandleToolError(err, "sentry_get_event"), nil
	}
	return mcp.HandleToolSuccess(out), nil
}

// --- Sidecar trace / postmortem ---

func (i *IncidentMCP) handleParseTrace(ctx context.Context, request mcpgo.CallToolRequest) (*mcpgo.CallToolResult, error) {
	out, err := incidentsidecar.DefaultClient.PostJSON("/api/incident/parse-trace", map[string]interface{}{
		"trace": request.GetString("trace", ""),
	})
	if err != nil {
		return mcp.HandleToolError(err, "incident_parse_stack_trace"), nil
	}
	return mcp.HandleToolSuccess(out), nil
}

func (i *IncidentMCP) handleLinkCode(ctx context.Context, request mcpgo.CallToolRequest) (*mcpgo.CallToolResult, error) {
	out, err := incidentsidecar.DefaultClient.PostJSON("/api/incident/parse-trace", map[string]interface{}{
		"trace": request.GetString("trace", ""),
	})
	if err != nil {
		return mcp.HandleToolError(err, "incident_link_code_locations"), nil
	}
	return mcp.HandleToolSuccess(out), nil
}

func (i *IncidentMCP) handleGeneratePostmortem(ctx context.Context, request mcpgo.CallToolRequest) (*mcpgo.CallToolResult, error) {
	body := map[string]interface{}{
		"issue_key": request.GetString("issue_key", ""),
		"timeline_markdown": request.GetString("timeline_markdown", ""),
		"severity": request.GetString("severity", ""),
		"summary": request.GetString("summary", ""),
	}
	out, err := incidentsidecar.DefaultClient.PostJSON("/api/incident/postmortem/draft", body)
	if err != nil {
		return mcp.HandleToolError(err, "incident_generate_postmortem"), nil
	}
	return mcp.HandleToolSuccess(out), nil
}

func intArg(request mcpgo.CallToolRequest, key string, defaultVal int) int {
	if raw, ok := request.GetArguments()[key]; ok {
		switch v := raw.(type) {
		case float64:
			return int(v)
		case int:
			return v
		}
	}
	return defaultVal
}
