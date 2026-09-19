package ai

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
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

func TestGenerateResponseWithTools_setsIsErrorOnFailure(t *testing.T) {
	callCount := 0
	var secondReq map[string]any
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		callCount++
		w.Header().Set("Content-Type", "application/json")
		if callCount == 1 {
			_, _ = w.Write([]byte(`{
				"id":"msg_1",
				"role":"assistant",
				"stop_reason":"tool_use",
				"content":[{"type":"tool_use","id":"tu_err","name":"boom","input":{}}]
			}`))
			return
		}
		body, _ := io.ReadAll(r.Body)
		_ = json.Unmarshal(body, &secondReq)
		_, _ = w.Write([]byte(`{
			"id":"msg_2",
			"role":"assistant",
			"stop_reason":"end_turn",
			"content":[{"type":"text","text":"recovered"}]
		}`))
	}))
	defer srv.Close()

	p := NewClaudeProviderWithConfig("test-key", false, "", "claude-test")
	p.BaseURL = srv.URL
	tools := []ClaudeToolDefinition{{
		Name: "boom", Description: "fails",
		InputSchema: json.RawMessage(`{"type":"object","properties":{}}`),
	}}
	text, err := p.GenerateResponseWithTools(context.Background(),
		"system\n---SYSTEM_PROMPT_END---\nuser",
		nil,
		tools,
		func(ctx context.Context, req ToolUseRequest) (string, error) {
			return "", fmt.Errorf("not_found: missing snippet")
		},
	)
	if err != nil {
		t.Fatal(err)
	}
	if text != "recovered" {
		t.Fatalf("got %q", text)
	}
	msgs, _ := secondReq["messages"].([]any)
	if len(msgs) < 3 {
		t.Fatalf("expected tool follow-up messages, got %#v", secondReq["messages"])
	}
	userMsg, _ := msgs[len(msgs)-1].(map[string]any)
	content, _ := userMsg["content"].([]any)
	if len(content) == 0 {
		t.Fatal("missing tool_result content")
	}
	tr, _ := content[0].(map[string]any)
	if tr["is_error"] != true {
		t.Fatalf("expected is_error=true, got %#v", tr)
	}
}
