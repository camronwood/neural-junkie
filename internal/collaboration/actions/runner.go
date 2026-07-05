package actions

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/url"
	"os/exec"
	"strings"
	"time"

	"github.com/camronwood/neural-junkie/internal/collaboration"
	"github.com/camronwood/neural-junkie/internal/connectors"
	"github.com/camronwood/neural-junkie/internal/runbooklibrary"
)

const maxResponseBytes = 256 * 1024

// Result is stored as task output (JSON envelope).
type Result struct {
	Summary    string                 `json:"summary"`
	ActionType string                 `json:"action_type"`
	Data       map[string]interface{} `json:"data,omitempty"`
}

// SlackPostFunc posts a message to Slack and returns the message timestamp.
type SlackPostFunc func(ctx context.Context, channelID, text, threadTS, username string) (ts string, err error)

// ValidateSlackChannelFunc checks that the bot can post to channelID.
type ValidateSlackChannelFunc func(channelID string) error

// Config holds hub-level limits for action execution.
type Config struct {
	AllowedHosts         []string
	SMSEnabled           bool
	SlackEnabled         bool
	SlackPost            SlackPostFunc
	ValidateSlackChannel ValidateSlackChannelFunc
	WebSearchQuery       func(ctx context.Context, query string) ([]map[string]interface{}, error)
}

// Runner executes collaboration action tasks.
type Runner struct {
	Config Config
	Client *http.Client
}

func NewRunner(cfg Config) *Runner {
	return &Runner{
		Config: cfg,
		Client: &http.Client{Timeout: 60 * time.Second},
	}
}

// Execute runs an action task and returns JSON output for the task record.
func (r *Runner) Execute(ctx context.Context, collab *collaboration.Collaboration, task collaboration.CollaborationTask) (string, error) {
	if task.Action == nil {
		return "", fmt.Errorf("action task missing action spec")
	}
	typ := strings.ToLower(strings.TrimSpace(task.Action.Type))
	cfg := interpolateConfig(task.Action.Config, collab, task)
	if task.Action.ConnectorID != "" {
		if prof, err := connectors.Get(task.Action.ConnectorID); err == nil {
			cfg = connectors.ApplyToHTTPConfig(cfg, prof)
		}
	}

	var res Result
	var err error
	switch typ {
	case "http_get":
		res, err = r.httpGet(ctx, cfg)
	case "http_post":
		res, err = r.httpPost(ctx, cfg)
	case "webhook":
		res, err = r.webhook(ctx, cfg)
	case "web_search":
		res, err = r.webSearch(ctx, cfg)
	case "sms":
		res, err = r.sms(cfg)
	case "slack_message":
		res, err = r.slackMessage(ctx, cfg)
	case "mcp_tool":
		res, err = r.mcpTool(ctx, cfg)
	case "shell":
		res, err = r.shell(ctx, collab, cfg)
	case "wait_human":
		return "", fmt.Errorf("wait_human tasks require explicit approval via task API")
	case "git_status":
		res, err = r.gitStatus(ctx, collab)
	case "git_diff":
		res, err = r.gitDiff(ctx, collab, cfg)
	default:
		return "", fmt.Errorf("unknown action type %q", typ)
	}
	if err != nil {
		return "", err
	}
	res.ActionType = typ
	b, err := json.Marshal(res)
	if err != nil {
		return "", err
	}
	return string(b), nil
}

func (r *Runner) httpGet(ctx context.Context, cfg map[string]interface{}) (Result, error) {
	u := stringVal(cfg, "url")
	if u == "" {
		return Result{}, fmt.Errorf("http_get requires url")
	}
	if err := r.checkHost(u); err != nil {
		return Result{}, err
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, u, nil)
	if err != nil {
		return Result{}, err
	}
	applyHeaders(req, cfg)
	resp, err := r.Client.Do(req)
	if err != nil {
		return Result{}, err
	}
	defer resp.Body.Close()
	body, _ := io.ReadAll(io.LimitReader(resp.Body, maxResponseBytes))
	return Result{
		Summary: fmt.Sprintf("HTTP %d %s", resp.StatusCode, u),
		Data: map[string]interface{}{
			"status_code": resp.StatusCode,
			"body":        string(body),
		},
	}, nil
}

func (r *Runner) httpPost(ctx context.Context, cfg map[string]interface{}) (Result, error) {
	u := stringVal(cfg, "url")
	if u == "" {
		return Result{}, fmt.Errorf("http_post requires url")
	}
	if err := r.checkHost(u); err != nil {
		return Result{}, err
	}
	payload := cfg["body"]
	var body io.Reader
	if payload != nil {
		b, err := json.Marshal(payload)
		if err != nil {
			return Result{}, err
		}
		body = bytes.NewReader(b)
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, u, body)
	if err != nil {
		return Result{}, err
	}
	req.Header.Set("Content-Type", "application/json")
	applyHeaders(req, cfg)
	resp, err := r.Client.Do(req)
	if err != nil {
		return Result{}, err
	}
	defer resp.Body.Close()
	respBody, _ := io.ReadAll(io.LimitReader(resp.Body, maxResponseBytes))
	return Result{
		Summary: fmt.Sprintf("HTTP POST %d %s", resp.StatusCode, u),
		Data: map[string]interface{}{
			"status_code": resp.StatusCode,
			"body":        string(respBody),
		},
	}, nil
}

func (r *Runner) webhook(ctx context.Context, cfg map[string]interface{}) (Result, error) {
	cfg = map[string]interface{}{"url": stringVal(cfg, "url"), "body": cfg["payload"]}
	return r.httpPost(ctx, cfg)
}

func (r *Runner) webSearch(ctx context.Context, cfg map[string]interface{}) (Result, error) {
	q := stringVal(cfg, "query")
	if q == "" {
		return Result{}, fmt.Errorf("web_search requires query")
	}
	if r.Config.WebSearchQuery != nil {
		results, err := r.Config.WebSearchQuery(ctx, q)
		if err != nil {
			return Result{}, err
		}
		return Result{Summary: fmt.Sprintf("web search: %d results", len(results)), Data: map[string]interface{}{"results": results}}, nil
	}
	return Result{
		Summary: "web_search stub (configure WebSearchQuery in hub)",
		Data:    map[string]interface{}{"query": q, "results": []interface{}{}},
	}, nil
}

func (r *Runner) sms(cfg map[string]interface{}) (Result, error) {
	if !r.Config.SMSEnabled {
		return Result{}, fmt.Errorf("sms actions are disabled; enable in server config")
	}
	to := stringVal(cfg, "to")
	body := stringVal(cfg, "body")
	if to == "" || body == "" {
		return Result{}, fmt.Errorf("sms requires to and body")
	}
	return Result{Summary: fmt.Sprintf("sms queued to %s (provider not configured in v1 stub)", to), Data: map[string]interface{}{"to": to}}, nil
}

func (r *Runner) slackMessage(ctx context.Context, cfg map[string]interface{}) (Result, error) {
	if !r.Config.SlackEnabled {
		return Result{}, fmt.Errorf("slack_message actions are disabled; connect Slack in Settings")
	}
	if r.Config.SlackPost == nil {
		return Result{}, fmt.Errorf("slack_message not configured on hub")
	}
	channelID := stringVal(cfg, "channel_id")
	text := stringVal(cfg, "text")
	if channelID == "" {
		return Result{}, fmt.Errorf("slack_message requires channel_id")
	}
	if text == "" {
		return Result{}, fmt.Errorf("slack_message requires text")
	}
	if r.Config.ValidateSlackChannel != nil {
		if err := r.Config.ValidateSlackChannel(channelID); err != nil {
			return Result{}, err
		}
	}
	threadTS := stringVal(cfg, "thread_ts")
	username := stringVal(cfg, "username")
	ts, err := r.Config.SlackPost(ctx, channelID, text, threadTS, username)
	if err != nil {
		return Result{}, err
	}
	return Result{
		Summary: fmt.Sprintf("Slack message posted to %s", channelID),
		Data: map[string]interface{}{
			"channel_id": channelID,
			"ts":         ts,
		},
	}, nil
}

func (r *Runner) checkHost(rawURL string) error {
	u, err := url.Parse(rawURL)
	if err != nil {
		return err
	}
	if u.Scheme != "http" && u.Scheme != "https" {
		return fmt.Errorf("unsupported URL scheme %q", u.Scheme)
	}
	host := u.Hostname()
	if host == "" {
		return fmt.Errorf("missing host")
	}
	if ip := net.ParseIP(host); ip != nil {
		if ip.IsLoopback() || ip.IsPrivate() {
			return fmt.Errorf("SSRF: private/loopback IP not allowed")
		}
	}
	lower := strings.ToLower(host)
	if lower == "localhost" || strings.HasSuffix(lower, ".local") {
		return fmt.Errorf("SSRF: host %q not allowed", host)
	}
	if len(r.Config.AllowedHosts) == 0 {
		return nil
	}
	for _, allowed := range r.Config.AllowedHosts {
		if strings.EqualFold(allowed, host) || strings.HasSuffix(lower, "."+strings.ToLower(allowed)) {
			return nil
		}
	}
	return fmt.Errorf("host %q not in allowlist", host)
}

func stringVal(m map[string]interface{}, key string) string {
	if m == nil {
		return ""
	}
	v, ok := m[key]
	if !ok || v == nil {
		return ""
	}
	switch t := v.(type) {
	case string:
		return t
	default:
		return fmt.Sprint(t)
	}
}

func applyHeaders(req *http.Request, cfg map[string]interface{}) {
	h, ok := cfg["headers"].(map[string]interface{})
	if !ok {
		return
	}
	for k, v := range h {
		req.Header.Set(k, fmt.Sprint(v))
	}
}

func interpolateConfig(cfg map[string]interface{}, collab *collaboration.Collaboration, task collaboration.CollaborationTask) map[string]interface{} {
	if cfg == nil {
		return nil
	}
	inputs := map[string]string{}
	if collab != nil && collab.RunInputs != nil {
		inputs = collab.RunInputs
	}
	out := make(map[string]interface{}, len(cfg))
	for k, v := range cfg {
		if s, ok := v.(string); ok {
			s = runbooklibrary.InterpolateString(s, collab, task, inputs)
			out[k] = s
		} else {
			out[k] = v
		}
	}
	return out
}

func (r *Runner) shell(ctx context.Context, collab *collaboration.Collaboration, cfg map[string]interface{}) (Result, error) {
	cmdStr := stringVal(cfg, "command")
	if cmdStr == "" {
		return Result{}, fmt.Errorf("shell requires command")
	}
	cwd := strings.TrimSpace(collab.WorkingDirectory)
	if cwd == "" {
		cwd = "."
	}
	cmd := exec.CommandContext(ctx, "sh", "-c", cmdStr)
	cmd.Dir = cwd
	out, err := cmd.CombinedOutput()
	if err != nil {
		return Result{}, fmt.Errorf("shell failed: %w: %s", err, string(out))
	}
	text := string(out)
	if len(text) > maxResponseBytes {
		text = text[:maxResponseBytes] + "...(truncated)"
	}
	return Result{
		Summary: fmt.Sprintf("shell exit 0 (%d bytes)", len(out)),
		Data:    map[string]interface{}{"output": text, "cwd": cwd},
	}, nil
}

func (r *Runner) mcpTool(ctx context.Context, cfg map[string]interface{}) (Result, error) {
	tool := stringVal(cfg, "tool")
	if tool == "" {
		tool = stringVal(cfg, "name")
	}
	if tool == "" {
		return Result{}, fmt.Errorf("mcp_tool requires tool name")
	}
	args, _ := cfg["arguments"].(map[string]interface{})
	return Result{
		Summary: fmt.Sprintf("mcp_tool %s invoked (hub stub — wire MCP client for full execution)", tool),
		Data:    map[string]interface{}{"tool": tool, "arguments": args},
	}, nil
}

func (r *Runner) gitStatus(ctx context.Context, collab *collaboration.Collaboration) (Result, error) {
	cwd := strings.TrimSpace(collab.WorkingDirectory)
	if cwd == "" {
		return Result{}, fmt.Errorf("git_status requires collaboration working directory")
	}
	cmd := exec.CommandContext(ctx, "git", "status", "--porcelain")
	cmd.Dir = cwd
	out, err := cmd.CombinedOutput()
	if err != nil {
		return Result{}, fmt.Errorf("git status: %w", err)
	}
	text := string(out)
	return Result{
		Summary: fmt.Sprintf("git status (%d lines)", strings.Count(text, "\n")),
		Data:    map[string]interface{}{"porcelain": text},
	}, nil
}

func (r *Runner) gitDiff(ctx context.Context, collab *collaboration.Collaboration, cfg map[string]interface{}) (Result, error) {
	cwd := strings.TrimSpace(collab.WorkingDirectory)
	if cwd == "" {
		return Result{}, fmt.Errorf("git_diff requires collaboration working directory")
	}
	args := []string{"diff"}
	if path := stringVal(cfg, "path"); path != "" {
		args = append(args, "--", path)
	}
	cmd := exec.CommandContext(ctx, "git", args...)
	cmd.Dir = cwd
	out, err := cmd.CombinedOutput()
	if err != nil {
		return Result{}, fmt.Errorf("git diff: %w", err)
	}
	text := string(out)
	if len(text) > maxResponseBytes {
		text = text[:maxResponseBytes] + "...(truncated)"
	}
	return Result{
		Summary: fmt.Sprintf("git diff (%d bytes)", len(text)),
		Data:    map[string]interface{}{"diff": text},
	}, nil
}
