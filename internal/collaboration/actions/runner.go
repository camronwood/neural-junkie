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
	"strings"
	"time"

	"github.com/camronwood/neural-junkie/internal/collaboration"
)

const maxResponseBytes = 256 * 1024

// Result is stored as task output (JSON envelope).
type Result struct {
	Summary    string                 `json:"summary"`
	ActionType string                 `json:"action_type"`
	Data       map[string]interface{} `json:"data,omitempty"`
}

// Config holds hub-level limits for action execution.
type Config struct {
	AllowedHosts   []string
	SMSEnabled     bool
	WebSearchQuery func(ctx context.Context, query string) ([]map[string]interface{}, error)
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
	case "mcp_tool":
		res, err = Result{Summary: "mcp_tool execution is routed via agent MCP at dispatch", ActionType: typ}, nil
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
	out := make(map[string]interface{}, len(cfg))
	for k, v := range cfg {
		if s, ok := v.(string); ok {
			s = strings.ReplaceAll(s, "{{collab.description}}", collab.Description)
			s = strings.ReplaceAll(s, "{{task.title}}", task.Title)
			s = strings.ReplaceAll(s, "{{task.description}}", task.Description)
			out[k] = s
		} else {
			out[k] = v
		}
	}
	return out
}
