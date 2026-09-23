package actions

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"testing"

	"github.com/camronwood/neural-junkie/internal/collaboration"
)

type roundTripFunc func(*http.Request) (*http.Response, error)

func (f roundTripFunc) RoundTrip(req *http.Request) (*http.Response, error) {
	return f(req)
}

func testRunnerWithTransport(fn roundTripFunc) *Runner {
	r := NewRunner(Config{})
	r.Client = &http.Client{Transport: fn}
	return r
}

func TestHTTPGetAgainstTestServer(t *testing.T) {
	r := testRunnerWithTransport(func(_ *http.Request) (*http.Response, error) {
		return &http.Response{
			StatusCode: http.StatusOK,
			Body:       io.NopCloser(strings.NewReader(`{"ok":true}`)),
			Header:     make(http.Header),
		}, nil
	})
	out, err := r.Execute(context.Background(), &collaboration.Collaboration{Description: "goal"}, collaboration.CollaborationTask{
		Kind:   collaboration.TaskKindAction,
		Action: &collaboration.TaskActionSpec{Type: "http_get", Config: map[string]interface{}{"url": "https://example.com/health"}},
	})
	if err != nil {
		t.Fatal(err)
	}
	var envelope Result
	if err := json.Unmarshal([]byte(out), &envelope); err != nil {
		t.Fatal(err)
	}
	if envelope.ActionType != "http_get" {
		t.Fatalf("action_type = %q", envelope.ActionType)
	}
	if envelope.Data["status_code"].(float64) != 200 {
		t.Fatalf("status_code = %v", envelope.Data["status_code"])
	}
}

func TestHTTPGetBlocksLocalhost(t *testing.T) {
	r := NewRunner(Config{})
	_, err := r.Execute(context.Background(), &collaboration.Collaboration{}, collaboration.CollaborationTask{
		Kind:   collaboration.TaskKindAction,
		Action: &collaboration.TaskActionSpec{Type: "http_get", Config: map[string]interface{}{"url": "http://127.0.0.1:9999/"}},
	})
	if err == nil || !strings.Contains(err.Error(), "SSRF") {
		t.Fatalf("expected SSRF error, got %v", err)
	}
}

func TestHTTPGetAllowlist(t *testing.T) {
	r := testRunnerWithTransport(func(_ *http.Request) (*http.Response, error) {
		return &http.Response{
			StatusCode: http.StatusNoContent,
			Body:       io.NopCloser(strings.NewReader("")),
			Header:     make(http.Header),
		}, nil
	})
	r.Config.AllowedHosts = []string{"example.com"}

	_, err := r.Execute(context.Background(), &collaboration.Collaboration{}, collaboration.CollaborationTask{
		Kind:   collaboration.TaskKindAction,
		Action: &collaboration.TaskActionSpec{Type: "http_get", Config: map[string]interface{}{"url": "https://example.com/ok"}},
	})
	if err != nil {
		t.Fatal(err)
	}

	_, err = r.Execute(context.Background(), &collaboration.Collaboration{}, collaboration.CollaborationTask{
		Kind:   collaboration.TaskKindAction,
		Action: &collaboration.TaskActionSpec{Type: "http_get", Config: map[string]interface{}{"url": "https://other.example.org/"}},
	})
	if err == nil || !strings.Contains(err.Error(), "allowlist") {
		t.Fatalf("expected allowlist error, got %v", err)
	}
}

func TestWebSearchUsesProvider(t *testing.T) {
	r := NewRunner(Config{
		WebSearchQuery: func(_ context.Context, q string) ([]map[string]interface{}, error) {
			return []map[string]interface{}{{"title": "hit", "query": q}}, nil
		},
	})
	out, err := r.Execute(context.Background(), &collaboration.Collaboration{}, collaboration.CollaborationTask{
		Kind:   collaboration.TaskKindAction,
		Action: &collaboration.TaskActionSpec{Type: "web_search", Config: map[string]interface{}{"query": "neural junkie"}},
	})
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(out, "hit") {
		t.Fatalf("output = %s", out)
	}
}

func TestSMSRequiresURL(t *testing.T) {
	r := NewRunner(Config{})
	_, err := r.Execute(context.Background(), &collaboration.Collaboration{}, collaboration.CollaborationTask{
		Kind:   collaboration.TaskKindAction,
		Action: &collaboration.TaskActionSpec{Type: "sms", Config: map[string]interface{}{"to": "+19716785014", "body": "hi"}},
	})
	if err == nil || !strings.Contains(err.Error(), "url") {
		t.Fatalf("expected url required, got %v", err)
	}
}

func TestSMSPostsForm(t *testing.T) {
	var gotMethod, gotCT, gotBody string
	r := testRunnerWithTransport(func(req *http.Request) (*http.Response, error) {
		gotMethod = req.Method
		gotCT = req.Header.Get("Content-Type")
		b, _ := io.ReadAll(req.Body)
		gotBody = string(b)
		return &http.Response{
			StatusCode: http.StatusOK,
			Body:       io.NopCloser(strings.NewReader(`{"ok":true}`)),
			Header:     make(http.Header),
		}, nil
	})
	out, err := r.Execute(context.Background(), &collaboration.Collaboration{}, collaboration.CollaborationTask{
		Kind: collaboration.TaskKindAction,
		Action: &collaboration.TaskActionSpec{Type: "sms", Config: map[string]interface{}{
			"url":  "https://example.com/sms",
			"to":   "+19716785014",
			"from": "+15551212",
			"body": "ping",
		}},
	})
	if err != nil {
		t.Fatal(err)
	}
	if gotMethod != http.MethodPost {
		t.Fatalf("method = %s", gotMethod)
	}
	if !strings.Contains(gotCT, "application/x-www-form-urlencoded") {
		t.Fatalf("content-type = %q", gotCT)
	}
	if !strings.Contains(gotBody, "To=%2B19716785014") || !strings.Contains(gotBody, "Body=ping") {
		t.Fatalf("body = %q", gotBody)
	}
	if !strings.Contains(out, "SMS sent") {
		t.Fatalf("output = %s", out)
	}
}

func TestSMSHTTPError(t *testing.T) {
	r := testRunnerWithTransport(func(_ *http.Request) (*http.Response, error) {
		return &http.Response{
			StatusCode: http.StatusBadRequest,
			Body:       io.NopCloser(strings.NewReader(`nope`)),
			Header:     make(http.Header),
		}, nil
	})
	_, err := r.Execute(context.Background(), &collaboration.Collaboration{}, collaboration.CollaborationTask{
		Kind: collaboration.TaskKindAction,
		Action: &collaboration.TaskActionSpec{Type: "sms", Config: map[string]interface{}{
			"url": "https://example.com/sms", "to": "+1", "body": "hi",
		}},
	})
	if err == nil || !strings.Contains(err.Error(), "400") {
		t.Fatalf("expected HTTP 400, got %v", err)
	}
}

func TestSMSBlocksLocalhost(t *testing.T) {
	r := NewRunner(Config{})
	_, err := r.Execute(context.Background(), &collaboration.Collaboration{}, collaboration.CollaborationTask{
		Kind: collaboration.TaskKindAction,
		Action: &collaboration.TaskActionSpec{Type: "sms", Config: map[string]interface{}{
			"url": "http://127.0.0.1:9/sms", "to": "+1", "body": "hi",
		}},
	})
	if err == nil || !strings.Contains(err.Error(), "SSRF") {
		t.Fatalf("expected SSRF, got %v", err)
	}
}

func TestEmailRequiresHost(t *testing.T) {
	r := NewRunner(Config{})
	_, err := r.Execute(context.Background(), &collaboration.Collaboration{}, collaboration.CollaborationTask{
		Kind: collaboration.TaskKindAction,
		Action: &collaboration.TaskActionSpec{Type: "email", Config: map[string]interface{}{
			"to": "camronwood@gmail.com", "subject": "hi", "body": "test",
		}},
	})
	if err == nil || !strings.Contains(err.Error(), "host") {
		t.Fatalf("expected host required, got %v", err)
	}
}

func TestEmailSendInjected(t *testing.T) {
	var got EmailMessage
	r := NewRunner(Config{
		EmailSend: func(_ context.Context, msg EmailMessage) error {
			got = msg
			return nil
		},
	})
	out, err := r.Execute(context.Background(), &collaboration.Collaboration{RunInputs: map[string]string{"dest": "camronwood@gmail.com"}}, collaboration.CollaborationTask{
		Kind: collaboration.TaskKindAction,
		Action: &collaboration.TaskActionSpec{Type: "email", Config: map[string]interface{}{
			"to": "{{inputs.dest}}", "subject": "NJ {{task.title}}", "body": "hello",
			"host": "smtp.example.com", "port": "587", "from": "nj@example.com", "username": "nj", "password": "x",
		}},
		Title: "alert",
	})
	if err != nil {
		t.Fatal(err)
	}
	if got.To != "camronwood@gmail.com" || got.Subject != "NJ alert" || got.Body != "hello" {
		t.Fatalf("got = %#v", got)
	}
	if !strings.Contains(out, "Email sent") {
		t.Fatalf("output = %s", out)
	}
}

func TestEmailSendFailure(t *testing.T) {
	r := NewRunner(Config{
		EmailSend: func(_ context.Context, _ EmailMessage) error {
			return fmt.Errorf("smtp refused")
		},
	})
	_, err := r.Execute(context.Background(), &collaboration.Collaboration{}, collaboration.CollaborationTask{
		Kind: collaboration.TaskKindAction,
		Action: &collaboration.TaskActionSpec{Type: "email", Config: map[string]interface{}{
			"to": "a@b.c", "subject": "s", "body": "b", "host": "smtp.example.com", "from": "nj@example.com",
		}},
	})
	if err == nil || !strings.Contains(err.Error(), "refused") {
		t.Fatalf("expected refused, got %v", err)
	}
}

func TestSlackMessageDisabledByDefault(t *testing.T) {
	r := NewRunner(Config{SlackEnabled: false})
	_, err := r.Execute(context.Background(), &collaboration.Collaboration{}, collaboration.CollaborationTask{
		Kind: collaboration.TaskKindAction,
		Action: &collaboration.TaskActionSpec{
			Type:   "slack_message",
			Config: map[string]interface{}{"channel_id": "C012", "text": "hi"},
		},
	})
	if err == nil || !strings.Contains(err.Error(), "disabled") {
		t.Fatalf("expected disabled, got %v", err)
	}
}

func TestSlackMessageRequiresFields(t *testing.T) {
	r := NewRunner(Config{
		SlackEnabled: true,
		SlackPost: func(_ context.Context, _, _, _, _ string) (string, error) {
			return "1234.5678", nil
		},
	})
	_, err := r.Execute(context.Background(), &collaboration.Collaboration{}, collaboration.CollaborationTask{
		Kind:   collaboration.TaskKindAction,
		Action: &collaboration.TaskActionSpec{Type: "slack_message", Config: map[string]interface{}{"channel_id": "C012"}},
	})
	if err == nil || !strings.Contains(err.Error(), "text") {
		t.Fatalf("expected text required, got %v", err)
	}
	_, err = r.Execute(context.Background(), &collaboration.Collaboration{}, collaboration.CollaborationTask{
		Kind:   collaboration.TaskKindAction,
		Action: &collaboration.TaskActionSpec{Type: "slack_message", Config: map[string]interface{}{"text": "hi"}},
	})
	if err == nil || !strings.Contains(err.Error(), "channel_id") {
		t.Fatalf("expected channel_id required, got %v", err)
	}
}

func TestSlackMessagePosts(t *testing.T) {
	var gotChannel, gotText, gotThread, gotUser string
	r := NewRunner(Config{
		SlackEnabled: true,
		SlackPost: func(_ context.Context, channelID, text, threadTS, username string) (string, error) {
			gotChannel, gotText, gotThread, gotUser = channelID, text, threadTS, username
			return "1710000000.000100", nil
		},
	})
	out, err := r.Execute(context.Background(), &collaboration.Collaboration{Description: "goal"}, collaboration.CollaborationTask{
		Title: "notify",
		Kind:  collaboration.TaskKindAction,
		Action: &collaboration.TaskActionSpec{
			Type: "slack_message",
			Config: map[string]interface{}{
				"channel_id": "C01234567",
				"text":       "Runbook {{task.title}}: {{collab.description}}",
				"thread_ts":  "1710000000.000000",
				"username":   "Runbook Bot",
			},
		},
	})
	if err != nil {
		t.Fatal(err)
	}
	if gotChannel != "C01234567" || gotText != "Runbook notify: goal" || gotThread != "1710000000.000000" || gotUser != "Runbook Bot" {
		t.Fatalf("slack post args = %q %q %q %q", gotChannel, gotText, gotThread, gotUser)
	}
	var envelope Result
	if err := json.Unmarshal([]byte(out), &envelope); err != nil {
		t.Fatal(err)
	}
	if envelope.ActionType != "slack_message" {
		t.Fatalf("action_type = %q", envelope.ActionType)
	}
	if envelope.Data["ts"] != "1710000000.000100" {
		t.Fatalf("ts = %v", envelope.Data["ts"])
	}
}

func TestHTTPPostAndWebhook(t *testing.T) {
	r := testRunnerWithTransport(func(req *http.Request) (*http.Response, error) {
		if req.Method != http.MethodPost {
			t.Fatalf("method = %s", req.Method)
		}
		return &http.Response{
			StatusCode: http.StatusCreated,
			Body:       io.NopCloser(strings.NewReader(`{"id":1}`)),
			Header:     make(http.Header),
		}, nil
	})
	out, err := r.Execute(context.Background(), &collaboration.Collaboration{}, collaboration.CollaborationTask{
		Kind:   collaboration.TaskKindAction,
		Action: &collaboration.TaskActionSpec{Type: "http_post", Config: map[string]interface{}{"url": "https://example.com/hook", "body": map[string]interface{}{"ok": true}}},
	})
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(out, "201") {
		t.Fatalf("post output = %s", out)
	}

	whOut, err := r.Execute(context.Background(), &collaboration.Collaboration{}, collaboration.CollaborationTask{
		Kind:   collaboration.TaskKindAction,
		Action: &collaboration.TaskActionSpec{Type: "webhook", Config: map[string]interface{}{"url": "https://example.com/wh", "payload": "ping"}},
	})
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(whOut, "webhook") && !strings.Contains(whOut, "http_post") {
		// envelope action_type is webhook
		var env Result
		_ = json.Unmarshal([]byte(whOut), &env)
		if env.ActionType != "webhook" {
			t.Fatalf("webhook action_type = %q", env.ActionType)
		}
	}
}

func TestWebSearchRequiresProvider(t *testing.T) {
	r := NewRunner(Config{})
	_, err := r.Execute(context.Background(), &collaboration.Collaboration{}, collaboration.CollaborationTask{
		Kind:   collaboration.TaskKindAction,
		Action: &collaboration.TaskActionSpec{Type: "web_search", Config: map[string]interface{}{"query": "neural junkie"}},
	})
	if err == nil || !strings.Contains(err.Error(), "not configured") {
		t.Fatalf("expected not configured error, got %v", err)
	}
}

func TestMCPToolFailsWithoutClient(t *testing.T) {
	r := NewRunner(Config{})
	_, err := r.Execute(context.Background(), &collaboration.Collaboration{}, collaboration.CollaborationTask{
		Kind:   collaboration.TaskKindAction,
		Action: &collaboration.TaskActionSpec{Type: "mcp_tool", Config: map[string]interface{}{"tool": "search"}},
	})
	if err == nil || !strings.Contains(err.Error(), "not wired") {
		t.Fatalf("expected not wired error, got %v", err)
	}
}

func TestInterpolateConfig(t *testing.T) {
	cfg := interpolateConfig(
		map[string]interface{}{"url": "{{task.title}}-{{collab.description}}"},
		&collaboration.Collaboration{Description: "goal"},
		collaboration.CollaborationTask{Title: "t1", Description: "d1"},
	)
	if cfg["url"] != "t1-goal" {
		t.Fatalf("url = %v", cfg["url"])
	}
}
