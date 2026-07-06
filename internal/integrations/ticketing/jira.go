package ticketing

import (
	"context"

	jiraclient "github.com/camronwood/neural-junkie/internal/integrations/jira"
	"github.com/camronwood/neural-junkie/internal/config"
)

type jiraProvider struct {
	settings config.JiraConfig
}

func NewJiraProvider(settings config.JiraConfig) Provider {
	return &jiraProvider{settings: settings}
}

func (p *jiraProvider) Name() string { return "jira" }

func (p *jiraProvider) client() (*jiraclient.Client, error) {
	return jiraclient.NewClient(p.settings)
}

func (p *jiraProvider) Get(ctx context.Context, id string) (string, error) {
	c, err := p.client()
	if err != nil {
		return "", err
	}
	return c.GetIssue(ctx, id)
}

func (p *jiraProvider) Search(ctx context.Context, query string, max int) (string, error) {
	c, err := p.client()
	if err != nil {
		return "", err
	}
	return c.SearchIssues(ctx, query, max)
}

func (p *jiraProvider) Comment(ctx context.Context, id, body string) (string, error) {
	c, err := p.client()
	if err != nil {
		return "", err
	}
	return c.AddComment(ctx, id, body)
}

func (p *jiraProvider) Create(ctx context.Context, req CreateRequest) (string, error) {
	c, err := p.client()
	if err != nil {
		return "", err
	}
	return c.CreateIssue(ctx, jiraclient.CreateIssueRequest{
		ProjectKey:  p.settings.DefaultProjectKey,
		Summary:     req.Title,
		Description: req.Description,
		Priority:    req.Priority,
		Labels:      req.Labels,
		IssueType:   req.IssueType,
	})
}

func (p *jiraProvider) Transition(ctx context.Context, id, status string) (string, error) {
	c, err := p.client()
	if err != nil {
		return "", err
	}
	return c.TransitionIssue(ctx, id, status)
}

func (p *jiraProvider) Assign(ctx context.Context, id, assignee string) (string, error) {
	c, err := p.client()
	if err != nil {
		return "", err
	}
	return c.AssignIssue(ctx, id, assignee)
}

func (p *jiraProvider) SetPriority(ctx context.Context, id, priority string) (string, error) {
	c, err := p.client()
	if err != nil {
		return "", err
	}
	return c.SetPriority(ctx, id, priority)
}
