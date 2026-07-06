package ticketing

import (
	"context"
	"fmt"
	"strings"

	"github.com/camronwood/neural-junkie/internal/config"
)

// Provider abstracts read/write ticketing operations across backends.
type Provider interface {
	Name() string
	Get(ctx context.Context, id string) (string, error)
	Search(ctx context.Context, query string, max int) (string, error)
	Comment(ctx context.Context, id, body string) (string, error)
	Create(ctx context.Context, req CreateRequest) (string, error)
	Transition(ctx context.Context, id, status string) (string, error)
	Assign(ctx context.Context, id, assignee string) (string, error)
	SetPriority(ctx context.Context, id, priority string) (string, error)
}

// CreateRequest holds fields for creating a ticket/issue.
type CreateRequest struct {
	Title       string
	Description string
	Priority    string
	Labels      []string
	IssueType   string
}

// Registry resolves the active ticketing provider from hub config.
type Registry struct {
	cfg *config.Config
}

func NewRegistry(cfg *config.Config) *Registry {
	return &Registry{cfg: cfg}
}

func (r *Registry) DefaultProvider() (Provider, error) {
	if r == nil || r.cfg == nil {
		return nil, fmt.Errorf("hub config unavailable")
	}
	name := r.cfg.IncidentSettings().DefaultProviderOr("jira")
	return r.Provider(name)
}

func (r *Registry) Provider(name string) (Provider, error) {
	switch strings.ToLower(strings.TrimSpace(name)) {
	case "", "jira":
		return NewJiraProvider(r.cfg.JiraSettings()), nil
	case "github", "github_issues", "github-issues":
		return NewGitHubProvider(r.cfg.GitHubIssuesSettings()), nil
	case "linear":
		return NewLinearProvider(r.cfg.LinearSettings()), nil
	default:
		return nil, fmt.Errorf("unknown ticketing provider %q", name)
	}
}

// WriteAllowed reports whether mutating ticket operations are permitted.
func WriteAllowed(cfg *config.Config) bool {
	if cfg == nil {
		return false
	}
	return cfg.IncidentSettings().WriteModeEnabled()
}

// RequireApproval reports whether mutating ops need user approval.
func RequireApproval(cfg *config.Config) bool {
	if cfg == nil {
		return true
	}
	return cfg.IncidentSettings().RequireApprovalEnabled()
}

// MutatingToolNames lists MCP tools that modify external ticketing state.
var MutatingToolNames = map[string]bool{
	"jira_create_issue":       true,
	"jira_assign_issue":       true,
	"jira_transition_issue":   true,
	"jira_set_priority":       true,
	"jira_add_comment":        true,
	"ticket_create":           true,
	"ticket_assign":             true,
	"ticket_transition":         true,
	"ticket_comment":            true,
	"ticket_set_priority":       true,
}

func IsMutatingTool(name string) bool {
	return MutatingToolNames[name]
}

func WriteDeniedError(tool string) error {
	return fmt.Errorf("%s: incident write mode is disabled — enable Allow ticket mutations in Settings → Integrations → Incident", tool)
}
