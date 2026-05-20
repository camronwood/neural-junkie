package ai

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestOllamaGenerateResponseWithTools(t *testing.T) {
	calls := 0
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		calls++
		if calls == 1 {
			_ = json.NewEncoder(w).Encode(map[string]any{
				"message": map[string]any{
					"role": "assistant",
					"tool_calls": []map[string]any{
						{
							"type": "function",
							"function": map[string]any{
								"index": 0,
								"name":  "analyze_sequence",
								"arguments": map[string]string{
									"sequence": "MKTAY",
								},
							},
						},
					},
				},
				"done": true,
			})
			return
		}
		_ = json.NewEncoder(w).Encode(map[string]any{
			"message": map[string]any{
				"role":    "assistant",
				"content": "The sequence looks valid.",
			},
			"done": true,
		})
	}))
	defer srv.Close()

	p := NewOllamaProviderWithConfig(srv.URL, "test-model")
	tools := []ClaudeToolDefinition{{
		Name:        "analyze_sequence",
		Description: "Analyze sequence",
		InputSchema: json.RawMessage(`{"type":"object","properties":{"sequence":{"type":"string"}},"required":["sequence"]}`),
	}}

	out, err := p.GenerateResponseWithTools(context.Background(),
		"system\n---SYSTEM_PROMPT_END---\nfold this",
		nil,
		tools,
		func(ctx context.Context, req ToolUseRequest) (string, error) {
			if req.Name != "analyze_sequence" {
				t.Fatalf("unexpected tool %q", req.Name)
			}
			return "protein, 5 aa", nil
		},
	)
	if err != nil {
		t.Fatal(err)
	}
	if out == "" {
		t.Fatal("expected final content")
	}
	if calls < 2 {
		t.Fatalf("expected 2 ollama calls, got %d", calls)
	}
}

func TestOllamaGenerateResponseWithToolsUnsupportedFallback(t *testing.T) {
	calls := 0
	srv2 := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		calls++
		if calls == 1 {
			w.WriteHeader(http.StatusBadRequest)
			_, _ = w.Write([]byte(`{"error":"does not support tools"}`))
			return
		}
		_ = json.NewEncoder(w).Encode(map[string]any{
			"message": map[string]any{"role": "assistant", "content": "Hello from bio."},
			"done":    true,
		})
	}))
	defer srv2.Close()

	p2 := NewOllamaProviderWithConfig(srv2.URL, "custom-bio:8b")
	tools := []ClaudeToolDefinition{{
		Name:        "analyze_sequence",
		Description: "Analyze",
		InputSchema: json.RawMessage(`{"type":"object","properties":{}}`),
	}}
	out, err := p2.GenerateResponseWithTools(context.Background(),
		"system\n---SYSTEM_PROMPT_END---\nhi",
		nil,
		tools,
		func(context.Context, ToolUseRequest) (string, error) { return "", nil },
	)
	if err != nil {
		t.Fatal(err)
	}
	if out != "Hello from bio." {
		t.Fatalf("got %q", out)
	}
	if calls != 2 {
		t.Fatalf("expected 2 calls (tools fail + plain), got %d", calls)
	}
}

func TestOllamaSupportsToolsDefaultModel(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/api/show" {
			_ = json.NewEncoder(w).Encode(ollamaShowResponse{Capabilities: []string{"completion", "tools"}})
			return
		}
		http.NotFound(w, r)
	}))
	defer srv.Close()

	p := NewOllamaProviderWithConfig(srv.URL, "llama3.1")
	if !p.SupportsTools() {
		t.Fatal("expected SupportsTools true when Ollama reports tools capability")
	}
}
