package actions

import (
	"bytes"
	"context"
	"crypto/tls"
	"encoding/json"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/smtp"
	"net/url"
	"os/exec"
	"strconv"
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

// EmailMessage is the payload for outbound email actions.
type EmailMessage struct {
	From     string
	To       string
	Subject  string
	Body     string
	SMTPHost string
	SMTPPort int
	Username string
	Password string
}

// EmailSendFunc sends email (injectable for tests). When nil, net/smtp is used.
type EmailSendFunc func(ctx context.Context, msg EmailMessage) error

// Config holds hub-level limits for action execution.
type Config struct {
	AllowedHosts         []string
	SMSEnabled           bool // deprecated: SMS runs when a connector/url is present
	SlackEnabled         bool
	SlackPost            SlackPostFunc
	ValidateSlackChannel ValidateSlackChannelFunc
	WebSearchQuery       func(ctx context.Context, query string) ([]map[string]interface{}, error)
	EmailSend            EmailSendFunc
	HTTPClient           *http.Client // optional; tests inject RoundTrip
}

// Runner executes collaboration action tasks.
type Runner struct {
	Config Config
	Client *http.Client
}

func NewRunner(cfg Config) *Runner {
	client := cfg.HTTPClient
	if client == nil {
		client = &http.Client{Timeout: 60 * time.Second}
	}
	return &Runner{
		Config: cfg,
		Client: client,
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
			switch typ {
			case "sms":
				cfg = connectors.ApplyToSMSConfig(cfg, prof)
			case "email":
				cfg = connectors.ApplyToEmailConfig(cfg, prof)
			default:
				cfg = connectors.ApplyToHTTPConfig(cfg, prof)
			}
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
		res, err = r.sms(ctx, cfg)
	case "email":
		res, err = r.email(ctx, cfg)
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
	return Result{}, fmt.Errorf("web_search is not configured; set WebSearchQuery on the hub or remove this action task")
}

func (r *Runner) sms(ctx context.Context, cfg map[string]interface{}) (Result, error) {
	to := stringVal(cfg, "to")
	body := stringVal(cfg, "body")
	if to == "" || body == "" {
		return Result{}, fmt.Errorf("sms requires to and body")
	}
	u := stringVal(cfg, "url")
	if u == "" {
		return Result{}, fmt.Errorf("sms requires a connector with url (or url in action config)")
	}
	if err := r.checkHost(u); err != nil {
		return Result{}, err
	}
	from := stringVal(cfg, "from")
	format := strings.ToLower(strings.TrimSpace(stringVal(cfg, "format")))
	if format == "" {
		format = "form"
	}

	var reqBody io.Reader
	contentType := "application/x-www-form-urlencoded"
	switch format {
	case "json":
		payload := map[string]string{"to": to, "body": body}
		if from != "" {
			payload["from"] = from
		}
		b, err := json.Marshal(payload)
		if err != nil {
			return Result{}, err
		}
		reqBody = bytes.NewReader(b)
		contentType = "application/json"
	default:
		form := url.Values{}
		form.Set("To", to)
		form.Set("Body", body)
		if from != "" {
			form.Set("From", from)
		}
		reqBody = strings.NewReader(form.Encode())
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, u, reqBody)
	if err != nil {
		return Result{}, err
	}
	req.Header.Set("Content-Type", contentType)
	applyHeaders(req, cfg)
	resp, err := r.Client.Do(req)
	if err != nil {
		return Result{}, err
	}
	defer resp.Body.Close()
	respBody, _ := io.ReadAll(io.LimitReader(resp.Body, maxResponseBytes))
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return Result{}, fmt.Errorf("sms HTTP %d: %s", resp.StatusCode, truncate(string(respBody), 200))
	}
	return Result{
		Summary: fmt.Sprintf("SMS sent to %s", to),
		Data: map[string]interface{}{
			"to":          to,
			"status_code": resp.StatusCode,
			"body":        string(respBody),
		},
	}, nil
}

func (r *Runner) email(ctx context.Context, cfg map[string]interface{}) (Result, error) {
	to := stringVal(cfg, "to")
	subject := stringVal(cfg, "subject")
	body := stringVal(cfg, "body")
	if to == "" {
		return Result{}, fmt.Errorf("email requires to")
	}
	if subject == "" && body == "" {
		return Result{}, fmt.Errorf("email requires subject or body")
	}
	host := stringVal(cfg, "host")
	if host == "" {
		return Result{}, fmt.Errorf("email requires an SMTP connector with host (or host in action config)")
	}
	port := 587
	if p := stringVal(cfg, "port"); p != "" {
		n, err := strconv.Atoi(p)
		if err != nil || n <= 0 {
			return Result{}, fmt.Errorf("email invalid port %q", p)
		}
		port = n
	}
	from := stringVal(cfg, "from")
	if from == "" {
		from = stringVal(cfg, "username")
	}
	if from == "" {
		return Result{}, fmt.Errorf("email requires from (or username on the SMTP connector)")
	}
	msg := EmailMessage{
		From:     from,
		To:       to,
		Subject:  subject,
		Body:     body,
		SMTPHost: host,
		SMTPPort: port,
		Username: stringVal(cfg, "username"),
		Password: stringVal(cfg, "password"),
	}
	send := r.Config.EmailSend
	if send == nil {
		send = defaultSMTPSend
	}
	if err := send(ctx, msg); err != nil {
		return Result{}, err
	}
	return Result{
		Summary: fmt.Sprintf("Email sent to %s", to),
		Data: map[string]interface{}{
			"to":      to,
			"subject": subject,
			"from":    from,
		},
	}, nil
}

func defaultSMTPSend(_ context.Context, msg EmailMessage) error {
	addr := fmt.Sprintf("%s:%d", msg.SMTPHost, msg.SMTPPort)
	var auth smtp.Auth
	if msg.Username != "" {
		auth = smtp.PlainAuth("", msg.Username, msg.Password, msg.SMTPHost)
	}
	raw := buildRFC822(msg.From, msg.To, msg.Subject, msg.Body)

	// Port 465: implicit TLS. Otherwise STARTTLS when possible via smtp.SendMail
	// (Go's SendMail uses STARTTLS on 587 when the server advertises it).
	if msg.SMTPPort == 465 {
		tlsCfg := &tls.Config{ServerName: msg.SMTPHost}
		conn, err := tls.Dial("tcp", addr, tlsCfg)
		if err != nil {
			return fmt.Errorf("smtp tls dial: %w", err)
		}
		defer conn.Close()
		c, err := smtp.NewClient(conn, msg.SMTPHost)
		if err != nil {
			return err
		}
		defer c.Close()
		if auth != nil {
			if err := c.Auth(auth); err != nil {
				return err
			}
		}
		if err := c.Mail(msg.From); err != nil {
			return err
		}
		if err := c.Rcpt(msg.To); err != nil {
			return err
		}
		w, err := c.Data()
		if err != nil {
			return err
		}
		if _, err := w.Write(raw); err != nil {
			return err
		}
		if err := w.Close(); err != nil {
			return err
		}
		return c.Quit()
	}
	return smtp.SendMail(addr, auth, msg.From, []string{msg.To}, raw)
}

func buildRFC822(from, to, subject, body string) []byte {
	var b strings.Builder
	b.WriteString("From: " + from + "\r\n")
	b.WriteString("To: " + to + "\r\n")
	b.WriteString("Subject: " + subject + "\r\n")
	b.WriteString("MIME-Version: 1.0\r\n")
	b.WriteString("Content-Type: text/plain; charset=UTF-8\r\n")
	b.WriteString("\r\n")
	b.WriteString(body)
	return []byte(b.String())
}

func truncate(s string, n int) string {
	if len(s) <= n {
		return s
	}
	return s[:n] + "…"
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
	return Result{}, fmt.Errorf("mcp_tool %q is not wired on the hub; configure an MCP client or remove this action task", tool)
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
