package ticketing

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/camronwood/neural-junkie/internal/config"
)

type linearProvider struct {
	settings config.LinearConfig
}

func NewLinearProvider(settings config.LinearConfig) Provider {
	return &linearProvider{settings: settings}
}

func (p *linearProvider) Name() string { return "linear" }

func (p *linearProvider) gql(ctx context.Context, query string, variables map[string]interface{}) ([]byte, error) {
	key := strings.TrimSpace(p.settings.APIKey)
	if key == "" {
		return nil, fmt.Errorf("linear api_key not configured")
	}
	payload, err := json.Marshal(map[string]interface{}{"query": query, "variables": variables})
	if err != nil {
		return nil, err
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, "https://api.linear.app/graphql", strings.NewReader(string(payload)))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Authorization", key)
	req.Header.Set("Content-Type", "application/json")
	client := &http.Client{Timeout: 45 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	data, err := io.ReadAll(io.LimitReader(resp.Body, 2<<20))
	if err != nil {
		return nil, err
	}
	if resp.StatusCode >= 400 {
		return data, fmt.Errorf("linear graphql: %s", strings.TrimSpace(string(data)))
	}
	return data, nil
}

func (p *linearProvider) Get(ctx context.Context, id string) (string, error) {
	query := `query($id: String!) { issue(id: $id) { id identifier title description priority state { name } assignee { name } } }`
	data, err := p.gql(ctx, query, map[string]interface{}{"id": strings.TrimSpace(id)})
	return string(data), err
}

func (p *linearProvider) Search(ctx context.Context, query string, max int) (string, error) {
	if max <= 0 {
		max = 20
	}
	filter := map[string]interface{}{}
	if q := strings.TrimSpace(query); q != "" {
		filter["title"] = map[string]interface{}{"containsIgnoreCase": q}
	}
	if team := strings.TrimSpace(p.settings.DefaultTeamID); team != "" {
		filter["team"] = map[string]interface{}{"id": map[string]interface{}{"eq": team}}
	}
	gql := `query($filter: IssueFilter, $first: Int!) { issues(filter: $filter, first: $first) { nodes { id identifier title state { name } priority } } }`
	data, err := p.gql(ctx, gql, map[string]interface{}{"filter": filter, "first": max})
	return string(data), err
}

func (p *linearProvider) Comment(ctx context.Context, id, body string) (string, error) {
	gql := `mutation($issueId: String!, $body: String!) { commentCreate(input: { issueId: $issueId, body: $body }) { success comment { id body } } }`
	data, err := p.gql(ctx, gql, map[string]interface{}{"issueId": strings.TrimSpace(id), "body": body})
	return string(data), err
}

func (p *linearProvider) Create(ctx context.Context, req CreateRequest) (string, error) {
	teamID := strings.TrimSpace(p.settings.DefaultTeamID)
	if teamID == "" {
		return "", fmt.Errorf("linear default_team_id required to create issues")
	}
	input := map[string]interface{}{
		"teamId":      teamID,
		"title":       req.Title,
		"description": req.Description,
	}
	gql := `mutation($input: IssueCreateInput!) { issueCreate(input: $input) { success issue { id identifier url } } }`
	data, err := p.gql(ctx, gql, map[string]interface{}{"input": input})
	return string(data), err
}

func (p *linearProvider) Transition(ctx context.Context, id, status string) (string, error) {
	gql := `mutation($id: String!, $stateId: String!) { issueUpdate(id: $id, input: { stateId: $stateId }) { success issue { id state { name } } } }`
	data, err := p.gql(ctx, gql, map[string]interface{}{"id": strings.TrimSpace(id), "stateId": strings.TrimSpace(status)})
	return string(data), err
}

func (p *linearProvider) Assign(ctx context.Context, id, assignee string) (string, error) {
	gql := `mutation($id: String!, $assigneeId: String!) { issueUpdate(id: $id, input: { assigneeId: $assigneeId }) { success issue { id assignee { name } } } }`
	data, err := p.gql(ctx, gql, map[string]interface{}{"id": strings.TrimSpace(id), "assigneeId": strings.TrimSpace(assignee)})
	return string(data), err
}

func (p *linearProvider) SetPriority(ctx context.Context, id, priority string) (string, error) {
	gql := `mutation($id: String!, $priority: Int!) { issueUpdate(id: $id, input: { priority: $priority }) { success issue { id priority } } }`
	pri := linearPriorityNumber(priority)
	data, err := p.gql(ctx, gql, map[string]interface{}{"id": strings.TrimSpace(id), "priority": pri})
	return string(data), err
}

func linearPriorityNumber(priority string) int {
	switch strings.ToLower(strings.TrimSpace(priority)) {
	case "urgent", "p0", "1":
		return 1
	case "high", "p1", "2":
		return 2
	case "medium", "p2", "3":
		return 3
	case "low", "p3", "p4", "4":
		return 4
	default:
		return 3
	}
}
