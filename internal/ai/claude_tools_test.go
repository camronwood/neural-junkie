package ai

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestGenerateResponseWithToolsLoop(t *testing.T) {
	callCount := 0
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		callCount++
		w.Header().Set("Content-Type", "application/json")
		if callCount == 1 {
			_, _ = w.Write([]byte(`{
				"id":"msg_1",
				"role":"assistant",
				"stop_reason":"tool_use",
				"content":[
					{"type":"tool_use","id":"tu_1","name":"echo","input":{"msg":"hi"}}
				]
			}`))
			return
		}
		_, _ = w.Write([]byte(`{
			"id":"msg_2",
			"role":"assistant",
			"stop_reason":"end_turn",
			"content":[{"type":"text","text":"done with tool"}]
		}`))
	}))
	defer srv.Close()

	p := NewClaudeProviderWithConfig("test-key", false, "", "claude-test")
	p.BaseURL = srv.URL
	tools := []ClaudeToolDefinition{{
		Name:        "echo",
		Description: "echo",
		InputSchema: json.RawMessage(`{"type":"object","properties":{"msg":{"type":"string"}}}`),
	}}

	text, err := p.GenerateResponseWithTools(context.Background(),
		"system\n---SYSTEM_PROMPT_END---\nuser question",
		nil,
		tools,
		func(ctx context.Context, req ToolUseRequest) (string, error) {
			if req.Name != "echo" {
				t.Fatalf("unexpected tool %q", req.Name)
			}
			return "tool output", nil
		},
	)
	if err != nil {
		t.Fatal(err)
	}
	if text != "done with tool" {
		t.Fatalf("got %q", text)
	}
	if callCount != 2 {
		t.Fatalf("expected 2 API calls, got %d", callCount)
	}
}
