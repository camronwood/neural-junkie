package ticketing

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/camronwood/neural-junkie/internal/config"
)

type githubProvider struct {
	settings config.GitHubIssuesConfig
}

func NewGitHubProvider(settings config.GitHubIssuesConfig) Provider {
	return &githubProvider{settings: settings}
}

func (p *githubProvider) Name() string { return "github" }

func (p *githubProvider) repo() (owner, repo string, err error) {
	parts := strings.Split(strings.TrimSpace(p.settings.DefaultRepo), "/")
	if len(parts) != 2 || parts[0] == "" || parts[1] == "" {
		return "", "", fmt.Errorf("github default_repo must be owner/repo")
	}
	return parts[0], parts[1], nil
}

func (p *githubProvider) do(ctx context.Context, method, path string, body io.Reader) ([]byte, error) {
	token := strings.TrimSpace(p.settings.Token)
	if token == "" {
		return nil, fmt.Errorf("github token not configured")
	}
	req, err := http.NewRequestWithContext(ctx, method, "https://api.github.com"+path, body)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Authorization", "Bearer "+token)
	req.Header.Set("Accept", "application/vnd.github+json")
	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}
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
		return data, fmt.Errorf("github %s %s: %s", method, path, strings.TrimSpace(string(data)))
	}
	return data, nil
}

func (p *githubProvider) Get(ctx context.Context, id string) (string, error) {
	owner, repo, err := p.repo()
	if err != nil {
		return "", err
	}
	id = strings.TrimPrefix(strings.TrimSpace(id), "#")
	path := fmt.Sprintf("/repos/%s/%s/issues/%s", owner, repo, url.PathEscape(id))
	data, err := p.do(ctx, http.MethodGet, path, nil)
	return string(data), err
}

func (p *githubProvider) Search(ctx context.Context, query string, max int) (string, error) {
	owner, repo, err := p.repo()
	if err != nil {
		return "", err
	}
	if max <= 0 {
		max = 20
	}
	q := fmt.Sprintf("repo:%s/%s is:issue %s", owner, repo, strings.TrimSpace(query))
	path := fmt.Sprintf("/search/issues?q=%s&per_page=%d", url.QueryEscape(q), max)
	data, err := p.do(ctx, http.MethodGet, path, nil)
	return string(data), err
}

func (p *githubProvider) Comment(ctx context.Context, id, body string) (string, error) {
	owner, repo, err := p.repo()
	if err != nil {
		return "", err
	}
	id = strings.TrimPrefix(strings.TrimSpace(id), "#")
	payload, _ := json.Marshal(map[string]string{"body": body})
	path := fmt.Sprintf("/repos/%s/%s/issues/%s/comments", owner, repo, url.PathEscape(id))
	data, err := p.do(ctx, http.MethodPost, path, strings.NewReader(string(payload)))
	return string(data), err
}

func (p *githubProvider) Create(ctx context.Context, req CreateRequest) (string, error) {
	owner, repo, err := p.repo()
	if err != nil {
		return "", err
	}
	payload, _ := json.Marshal(map[string]interface{}{
		"title": req.Title,
		"body":  req.Description,
		"labels": req.Labels,
	})
	path := fmt.Sprintf("/repos/%s/%s/issues", owner, repo)
	data, err := p.do(ctx, http.MethodPost, path, strings.NewReader(string(payload)))
	return string(data), err
}

func (p *githubProvider) Transition(ctx context.Context, id, status string) (string, error) {
	owner, repo, err := p.repo()
	if err != nil {
		return "", err
	}
	id = strings.TrimPrefix(strings.TrimSpace(id), "#")
	state := "open"
	if strings.EqualFold(status, "closed") || strings.EqualFold(status, "done") {
		state = "closed"
	}
	payload, _ := json.Marshal(map[string]string{"state": state})
	path := fmt.Sprintf("/repos/%s/%s/issues/%s", owner, repo, url.PathEscape(id))
	data, err := p.do(ctx, http.MethodPatch, path, strings.NewReader(string(payload)))
	return string(data), err
}

func (p *githubProvider) Assign(ctx context.Context, id, assignee string) (string, error) {
	return "", fmt.Errorf("github assign: use @mention in comment or GitHub UI (assignees require user login lookup)")
}

func (p *githubProvider) SetPriority(ctx context.Context, id, priority string) (string, error) {
	return "", fmt.Errorf("github issues do not have native priority — use labels")
}
