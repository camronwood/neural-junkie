package jira

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/camronwood/neural-junkie/internal/config"
)

// Client calls Jira Cloud REST API v3.
type Client struct {
	BaseURL  string
	Email    string
	APIToken string
	HTTP     *http.Client
}

func NewClient(settings config.JiraConfig) (*Client, error) {
	base := settings.BaseURLTrimmed()
	if base == "" {
		return nil, fmt.Errorf("jira base_url not configured")
	}
	email := strings.TrimSpace(settings.Email)
	token := strings.TrimSpace(settings.APIToken)
	if email == "" || token == "" {
		return nil, fmt.Errorf("jira email and api_token required")
	}
	return &Client{
		BaseURL:  base,
		Email:    email,
		APIToken: token,
		HTTP:     &http.Client{Timeout: 45 * time.Second},
	}, nil
}

func (c *Client) authHeader() string {
	raw := c.Email + ":" + c.APIToken
	return "Basic " + base64.StdEncoding.EncodeToString([]byte(raw))
}

func (c *Client) do(ctx context.Context, method, path string, body io.Reader) ([]byte, int, error) {
	reqURL := c.BaseURL + path
	req, err := http.NewRequestWithContext(ctx, method, reqURL, body)
	if err != nil {
		return nil, 0, err
	}
	req.Header.Set("Authorization", c.authHeader())
	req.Header.Set("Accept", "application/json")
	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}
	resp, err := c.HTTP.Do(req)
	if err != nil {
		return nil, 0, err
	}
	defer resp.Body.Close()
	data, err := io.ReadAll(io.LimitReader(resp.Body, 2<<20))
	if err != nil {
		return nil, resp.StatusCode, err
	}
	if resp.StatusCode >= 400 {
		return data, resp.StatusCode, fmt.Errorf("jira %s %s: %s", method, path, strings.TrimSpace(string(data)))
	}
	return data, resp.StatusCode, nil
}

// GetMyself verifies credentials.
func (c *Client) GetMyself(ctx context.Context) (string, error) {
	data, _, err := c.do(ctx, http.MethodGet, "/rest/api/3/myself", nil)
	return string(data), err
}

// GetIssue fetches an issue by key.
func (c *Client) GetIssue(ctx context.Context, key string) (string, error) {
	key = strings.TrimSpace(key)
	if key == "" {
		return "", fmt.Errorf("issue key required")
	}
	path := fmt.Sprintf("/rest/api/3/issue/%s?fields=summary,description,status,priority,labels,assignee,reporter,comment", key)
	data, _, err := c.do(ctx, http.MethodGet, path, nil)
	return string(data), err
}

// SearchIssues runs JQL search.
func (c *Client) SearchIssues(ctx context.Context, jql string, maxResults int) (string, error) {
	jql = strings.TrimSpace(jql)
	if jql == "" {
		return "", fmt.Errorf("jql required")
	}
	if maxResults <= 0 {
		maxResults = 20
	}
	if maxResults > 50 {
		maxResults = 50
	}
	path := fmt.Sprintf("/rest/api/3/search?jql=%s&maxResults=%d&fields=summary,status,priority,assignee",
		url.QueryEscape(jql), maxResults)
	data, _, err := c.do(ctx, http.MethodGet, path, nil)
	return string(data), err
}

// AddComment adds a comment body (plain text stored as ADF).
func (c *Client) AddComment(ctx context.Context, key, body string) (string, error) {
	key = strings.TrimSpace(key)
	body = strings.TrimSpace(body)
	if key == "" || body == "" {
		return "", fmt.Errorf("issue key and body required")
	}
	payload := map[string]interface{}{
		"body": adfParagraph(body),
	}
	raw, err := json.Marshal(payload)
	if err != nil {
		return "", err
	}
	path := fmt.Sprintf("/rest/api/3/issue/%s/comment", key)
	data, _, err := c.do(ctx, http.MethodPost, path, strings.NewReader(string(raw)))
	return string(data), err
}

// SummarizeIssue returns a concise text summary for agent context.
func (c *Client) SummarizeIssue(ctx context.Context, key string) (string, error) {
	raw, err := c.GetIssue(ctx, key)
	if err != nil {
		return "", err
	}
	var doc struct {
		Key    string `json:"key"`
		Fields struct {
			Summary     string          `json:"summary"`
			Description json.RawMessage `json:"description"`
			Status      struct {
				Name string `json:"name"`
			} `json:"status"`
			Priority struct {
				Name string `json:"name"`
			} `json:"priority"`
			Labels []string `json:"labels"`
		} `json:"fields"`
	}
	if err := json.Unmarshal([]byte(raw), &doc); err != nil {
		return raw, nil
	}
	var b strings.Builder
	fmt.Fprintf(&b, "Issue: %s\nSummary: %s\nStatus: %s\nPriority: %s\n",
		doc.Key, doc.Fields.Summary, doc.Fields.Status.Name, doc.Fields.Priority.Name)
	if len(doc.Fields.Labels) > 0 {
		fmt.Fprintf(&b, "Labels: %s\n", strings.Join(doc.Fields.Labels, ", "))
	}
	if len(doc.Fields.Description) > 0 {
		b.WriteString("Description: ")
		b.Write(doc.Fields.Description)
		b.WriteByte('\n')
	}
	return b.String(), nil
}

func adfParagraph(text string) map[string]interface{} {
	return map[string]interface{}{
		"type":    "doc",
		"version": 1,
		"content": []map[string]interface{}{
			{
				"type": "paragraph",
				"content": []map[string]interface{}{
					{"type": "text", "text": text},
				},
			},
		},
	}
}

// CreateIssueRequest holds fields for Jira issue creation.
type CreateIssueRequest struct {
	ProjectKey  string
	Summary     string
	Description string
	Priority    string
	Labels      []string
	IssueType   string
}

// CreateIssue creates a new Jira issue.
func (c *Client) CreateIssue(ctx context.Context, req CreateIssueRequest) (string, error) {
	pk := strings.TrimSpace(req.ProjectKey)
	if pk == "" {
		return "", fmt.Errorf("project key required")
	}
	summary := strings.TrimSpace(req.Summary)
	if summary == "" {
		return "", fmt.Errorf("summary required")
	}
	issueType := strings.TrimSpace(req.IssueType)
	if issueType == "" {
		issueType = "Bug"
	}
	fields := map[string]interface{}{
		"project":   map[string]string{"key": pk},
		"summary":   summary,
		"issuetype": map[string]string{"name": issueType},
	}
	if desc := strings.TrimSpace(req.Description); desc != "" {
		fields["description"] = adfParagraph(desc)
	}
	if pri := strings.TrimSpace(req.Priority); pri != "" {
		fields["priority"] = map[string]string{"name": pri}
	}
	if len(req.Labels) > 0 {
		fields["labels"] = req.Labels
	}
	payload := map[string]interface{}{"fields": fields}
	raw, err := json.Marshal(payload)
	if err != nil {
		return "", err
	}
	data, _, err := c.do(ctx, http.MethodPost, "/rest/api/3/issue", strings.NewReader(string(raw)))
	return string(data), err
}

// AssignIssue assigns a Jira issue by accountId or email lookup is not supported — pass accountId.
func (c *Client) AssignIssue(ctx context.Context, key, assignee string) (string, error) {
	key = strings.TrimSpace(key)
	assignee = strings.TrimSpace(assignee)
	if key == "" || assignee == "" {
		return "", fmt.Errorf("issue key and assignee required")
	}
	payload := map[string]interface{}{
		"accountId": assignee,
	}
	if strings.Contains(assignee, "@") {
		payload = map[string]interface{}{"emailAddress": assignee}
	}
	raw, err := json.Marshal(payload)
	if err != nil {
		return "", err
	}
	path := fmt.Sprintf("/rest/api/3/issue/%s/assignee", key)
	data, _, err := c.do(ctx, http.MethodPut, path, strings.NewReader(string(raw)))
	return string(data), err
}

// TransitionIssue moves issue to target status name or transition id.
func (c *Client) TransitionIssue(ctx context.Context, key, target string) (string, error) {
	key = strings.TrimSpace(key)
	target = strings.TrimSpace(target)
	if key == "" || target == "" {
		return "", fmt.Errorf("issue key and transition/status required")
	}
	transitionID, err := c.resolveTransitionID(ctx, key, target)
	if err != nil {
		return "", err
	}
	payload := map[string]interface{}{
		"transition": map[string]string{"id": transitionID},
	}
	raw, err := json.Marshal(payload)
	if err != nil {
		return "", err
	}
	path := fmt.Sprintf("/rest/api/3/issue/%s/transitions", key)
	data, _, err := c.do(ctx, http.MethodPost, path, strings.NewReader(string(raw)))
	return string(data), err
}

func (c *Client) resolveTransitionID(ctx context.Context, key, target string) (string, error) {
	path := fmt.Sprintf("/rest/api/3/issue/%s/transitions", key)
	data, _, err := c.do(ctx, http.MethodGet, path, nil)
	if err != nil {
		return "", err
	}
	var doc struct {
		Transitions []struct {
			ID   string `json:"id"`
			Name string `json:"name"`
			To   struct {
				Name string `json:"name"`
			} `json:"to"`
		} `json:"transitions"`
	}
	if err := json.Unmarshal(data, &doc); err != nil {
		return "", err
	}
	targetLower := strings.ToLower(target)
	for _, tr := range doc.Transitions {
		if tr.ID == target || strings.EqualFold(tr.Name, target) || strings.EqualFold(tr.To.Name, target) {
			return tr.ID, nil
		}
		if strings.Contains(strings.ToLower(tr.Name), targetLower) || strings.Contains(strings.ToLower(tr.To.Name), targetLower) {
			return tr.ID, nil
		}
	}
	return "", fmt.Errorf("no transition matching %q for issue %s", target, key)
}

// SetPriority updates issue priority by name.
func (c *Client) SetPriority(ctx context.Context, key, priority string) (string, error) {
	key = strings.TrimSpace(key)
	priority = strings.TrimSpace(priority)
	if key == "" || priority == "" {
		return "", fmt.Errorf("issue key and priority required")
	}
	payload := map[string]interface{}{
		"fields": map[string]interface{}{
			"priority": map[string]string{"name": priority},
		},
	}
	raw, err := json.Marshal(payload)
	if err != nil {
		return "", err
	}
	path := fmt.Sprintf("/rest/api/3/issue/%s", key)
	data, _, err := c.do(ctx, http.MethodPut, path, strings.NewReader(string(raw)))
	return string(data), err
}
