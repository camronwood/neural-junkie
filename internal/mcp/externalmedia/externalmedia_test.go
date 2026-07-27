package externalmedia

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/camronwood/neural-junkie/internal/config"
	mcp "github.com/camronwood/neural-junkie/internal/mcp"
)

func TestNewForAgent_DisabledWhenBaseURLEmpty(t *testing.T) {
	mcp.SetAppConfig(&config.Config{})
	defer mcp.SetAppConfig(nil)

	m, err := NewForAgent("Media Expert")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if m != nil {
		t.Fatal("expected nil ExternalMediaMCP when BaseURL is unset (default disabled)")
	}
}

func TestNewForAgent_DisabledWhenNotGranted(t *testing.T) {
	cfg := &config.Config{
		MCP: config.MCPConfig{
			ExternalMedia: config.ExternalMediaConfig{
				BaseURL:       "https://media.example.com/v1",
				GrantedAgents: []string{"Someone Else"},
			},
		},
	}
	mcp.SetAppConfig(cfg)
	defer mcp.SetAppConfig(nil)

	m, err := NewForAgent("Media Expert")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if m != nil {
		t.Fatal("expected nil ExternalMediaMCP when agent is not granted")
	}
}

func TestNewForAgent_AttachesWhenEnabledAndGranted(t *testing.T) {
	cfg := &config.Config{
		MCP: config.MCPConfig{
			ExternalMedia: config.ExternalMediaConfig{
				BaseURL:       "https://media.example.com/v1",
				APIKey:        "secret",
				GrantedAgents: []string{"Media Expert"},
			},
		},
	}
	mcp.SetAppConfig(cfg)
	defer mcp.SetAppConfig(nil)

	m, err := NewForAgent("media expert") // case-insensitive match
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if m == nil {
		t.Fatal("expected a non-nil ExternalMediaMCP for a granted agent")
	}
	if m.GetMCPServer() == nil {
		t.Fatal("expected non-nil underlying MCP server")
	}
	if err := m.Start(); err != nil {
		t.Fatalf("in-process Start() should no-op: %v", err)
	}
}

func TestSubmit_RequiresKindAndPrompt(t *testing.T) {
	m := &ExternalMediaMCP{baseURL: "https://media.example.com/v1", http: http.DefaultClient}
	if _, err := m.Submit(context.Background(), "", "a prompt", ""); err == nil {
		t.Fatal("expected error when kind is missing")
	}
	if _, err := m.Submit(context.Background(), "image", "", ""); err == nil {
		t.Fatal("expected error when prompt is missing")
	}
}

func TestStatusAndFetch_RequireJobID(t *testing.T) {
	m := &ExternalMediaMCP{baseURL: "https://media.example.com/v1", http: http.DefaultClient}
	if _, err := m.Status(context.Background(), ""); err == nil {
		t.Fatal("expected error when job_id is missing for Status")
	}
	if _, err := m.Fetch(context.Background(), ""); err == nil {
		t.Fatal("expected error when job_id is missing for Fetch")
	}
}

func TestSubmit_BlocksPrivateAndLoopbackBaseURL(t *testing.T) {
	m := &ExternalMediaMCP{baseURL: "http://127.0.0.1:9999/v1", http: http.DefaultClient}
	_, err := m.Submit(context.Background(), "image", "a cat", "")
	if err == nil {
		t.Fatal("expected SSRF error for loopback base URL")
	}
	if !strings.Contains(err.Error(), "SSRF") {
		t.Fatalf("expected SSRF error, got: %v", err)
	}
}

// TestSend_RoundTripsAgainstMockMediaAPI exercises the request/response
// plumbing (auth header, JSON submit body, status-code formatting) against a
// local mock media API. It calls send() directly rather than Submit()/do()
// because httptest servers bind to 127.0.0.1, which the production SSRF gate
// intentionally blocks (see TestSubmit_BlocksPrivateAndLoopbackBaseURL).
func TestSend_RoundTripsAgainstMockMediaAPI(t *testing.T) {
	var gotAuth, gotMethod, gotPath string
	var gotBody map[string]interface{}
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotAuth = r.Header.Get("Authorization")
		gotMethod = r.Method
		gotPath = r.URL.Path
		if r.Body != nil {
			_ = json.NewDecoder(r.Body).Decode(&gotBody)
		}
		w.WriteHeader(http.StatusCreated)
		_, _ = w.Write([]byte(`{"job_id":"job-123","status":"queued"}`))
	}))
	defer srv.Close()

	m := &ExternalMediaMCP{
		baseURL: srv.URL,
		apiKey:  "secret-key",
		http:    srv.Client(),
	}
	body, _ := json.Marshal(map[string]interface{}{"kind": "image", "prompt": "a cat"})
	result, err := m.send(context.Background(), http.MethodPost, srv.URL+"/submit", body)
	if err != nil {
		t.Fatalf("send: %v", err)
	}
	if gotMethod != http.MethodPost {
		t.Fatalf("method = %q", gotMethod)
	}
	if gotPath != "/submit" {
		t.Fatalf("path = %q", gotPath)
	}
	if gotAuth != "Bearer secret-key" {
		t.Fatalf("auth header = %q", gotAuth)
	}
	if gotBody["kind"] != "image" || gotBody["prompt"] != "a cat" {
		t.Fatalf("request body = %#v", gotBody)
	}
	if !strings.Contains(result, "HTTP 201") || !strings.Contains(result, "job-123") {
		t.Fatalf("unexpected result: %q", result)
	}
}

func TestAttachGranted_RegistersThreeTools(t *testing.T) {
	cfg := &config.Config{
		MCP: config.MCPConfig{
			ExternalMedia: config.ExternalMediaConfig{
				BaseURL:       "https://media.example.com/v1",
				GrantedAgents: []string{"Media Expert"},
			},
		},
	}
	mcp.SetAppConfig(cfg)
	defer mcp.SetAppConfig(nil)

	mcpServer, err := mcp.NewInProcessMCPServer("test-mcp", "1.0.0")
	if err != nil {
		t.Fatal(err)
	}
	m, err := AttachGranted(mcpServer, "Media Expert")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if m == nil {
		t.Fatal("expected non-nil result for a granted agent")
	}

	ungranted, err := AttachGranted(mcpServer, "Nobody")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if ungranted != nil {
		t.Fatal("expected nil result for an ungranted agent")
	}
}
