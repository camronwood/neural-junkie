package actions

import (
	"context"
	"encoding/json"
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

func TestSMSDisabledByDefault(t *testing.T) {
	r := NewRunner(Config{SMSEnabled: false})
	_, err := r.Execute(context.Background(), &collaboration.Collaboration{}, collaboration.CollaborationTask{
		Kind:   collaboration.TaskKindAction,
		Action: &collaboration.TaskActionSpec{Type: "sms", Config: map[string]interface{}{"to": "+1", "body": "hi"}},
	})
	if err == nil || !strings.Contains(err.Error(), "disabled") {
		t.Fatalf("expected disabled, got %v", err)
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
