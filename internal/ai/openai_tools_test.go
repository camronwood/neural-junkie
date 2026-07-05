package ai

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestOpenAIGenerateResponseWithTools(t *testing.T) {
	calls := 0
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		calls++
		if calls == 1 {
			_ = json.NewEncoder(w).Encode(map[string]any{
				"choices": []map[string]any{
					{
						"index": 0,
						"message": map[string]any{
							"role": "assistant",
							"tool_calls": []map[string]any{
								{
									"id":   "call_1",
									"type": "function",
									"function": map[string]any{
										"name":      "analyze_sequence",
										"arguments": `{"sequence":"MKTAY"}`,
									},
								},
							},
						},
						"finish_reason": "tool_calls",
					},
				},
			})
			return
		}
		_ = json.NewEncoder(w).Encode(map[string]any{
			"choices": []map[string]any{
				{
					"index": 0,
					"message": map[string]any{
						"role":    "assistant",
						"content": "The sequence looks valid.",
					},
					"finish_reason": "stop",
				},
			},
		})
	}))
	defer srv.Close()

	p := NewOpenAICompatProvider(srv.URL+"/v1", "", "test-model", nil)
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
			if req.ID != "call_1" {
				t.Fatalf("unexpected tool id %q", req.ID)
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
		t.Fatalf("expected 2 openai calls, got %d", calls)
	}
}

func TestOpenAIGenerateResponseWithToolsUnsupported(t *testing.T) {
	calls := 0
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		calls++
		w.WriteHeader(http.StatusBadRequest)
		_, _ = w.Write([]byte(`{"error":{"message":"model does not support tools"}}`))
	}))
	defer srv.Close()

	p := NewOpenAICompatProvider(srv.URL+"/v1", "", "test-model", nil)
	tools := []ClaudeToolDefinition{{
		Name:        "analyze_sequence",
		Description: "Analyze sequence",
		InputSchema: json.RawMessage(`{"type":"object","properties":{"sequence":{"type":"string"}}}`),
	}}

	_, err := p.GenerateResponseWithTools(context.Background(), "user msg", nil, tools, nil)
	if err != ErrNativeToolsUnsupported {
		t.Fatalf("expected ErrNativeToolsUnsupported, got %v", err)
	}
	if !p.NativeToolsMarkedUnsupported() {
		t.Fatal("expected native tools marked unsupported")
	}
	if p.SupportsTools() {
		t.Fatal("SupportsTools should be false after unsupported")
	}
	if calls != 1 {
		t.Fatalf("expected 1 call, got %d", calls)
	}
}

func TestOpenAICompatSupportsToolsOptimistic(t *testing.T) {
	p := NewOpenAICompatProvider("http://localhost:1234/v1", "", "test", nil)
	if !p.SupportsTools() {
		t.Fatal("expected optimistic SupportsTools before probe")
	}
}

func TestLMStudioSupportsToolsDelegation(t *testing.T) {
	lm := NewLMStudioProviderWithConfig("http://localhost:1234/v1", "qwen-test")
	if !lm.SupportsTools() {
		t.Fatal("expected optimistic SupportsTools")
	}
	lm.MarkNativeToolsUnsupported()
	if lm.SupportsTools() {
		t.Fatal("expected SupportsTools false after mark")
	}
}
