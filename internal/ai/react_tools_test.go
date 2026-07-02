package ai

import (
	"context"
	"fmt"
	"strings"
	"sync"
	"testing"

	"github.com/camronwood/neural-junkie/internal/protocol"
)

type mockReactProvider struct {
	mu       sync.Mutex
	responses []string
	calls    int
}

func (m *mockReactProvider) GenerateResponse(_ context.Context, _ string, _ []protocol.Message) (string, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	if m.calls >= len(m.responses) {
		return "", fmt.Errorf("no more mock responses")
	}
	out := m.responses[m.calls]
	m.calls++
	return out, nil
}

func (m *mockReactProvider) GenerateVisionResponse(context.Context, string, []byte, string, []protocol.Message) (string, error) {
	return "", fmt.Errorf("not implemented")
}

func (m *mockReactProvider) GetModel() string { return "mock-react" }

func TestReActToolProviderSingleToolThenAnswer(t *testing.T) {
	mock := &mockReactProvider{
		responses: []string{
			`<tool_call>{"name":"read_file","arguments":{"path":"main.go"}}</tool_call>`,
			"The file contains package main.",
		},
	}
	provider := NewReActToolProvider(mock)
	tools := []ClaudeToolDefinition{{
		Name:        "read_file",
		Description: "Read a file",
		InputSchema: []byte(`{"type":"object","properties":{"path":{"type":"string"}}}`),
	}}
	var executed bool
	out, err := provider.GenerateResponseWithTools(context.Background(),
		"system\n---SYSTEM_PROMPT_END---\nread main",
		nil,
		tools,
		func(_ context.Context, req ToolUseRequest) (string, error) {
			executed = true
			if req.Name != "read_file" {
				t.Fatalf("tool %q", req.Name)
			}
			return "package main", nil
		},
	)
	if err != nil {
		t.Fatal(err)
	}
	if !executed {
		t.Fatal("tool not executed")
	}
	if !strings.Contains(out, "package main") {
		t.Fatalf("out=%q", out)
	}
	if mock.calls != 2 {
		t.Fatalf("calls=%d want 2", mock.calls)
	}
}

func TestReActToolProviderIterationCap(t *testing.T) {
	mock := &mockReactProvider{
		responses: []string{
			`<tool_call>{"name":"grep","arguments":{"pattern":"x"}}</tool_call>`,
			`<tool_call>{"name":"grep","arguments":{"pattern":"y"}}</tool_call>`,
			`<tool_call>{"name":"grep","arguments":{"pattern":"z"}}</tool_call>`,
		},
	}
	provider := NewReActToolProvider(mock)
	tools := []ClaudeToolDefinition{{Name: "grep", InputSchema: []byte(`{"type":"object"}`)}}
	ctx := WithToolLoopMaxIterations(context.Background(), 2)
	_, err := provider.GenerateResponseWithTools(ctx, "find", nil, tools,
		func(context.Context, ToolUseRequest) (string, error) { return "ok", nil },
	)
	if !IsReActToolLoopError(err) {
		t.Fatalf("err=%v", err)
	}
}
