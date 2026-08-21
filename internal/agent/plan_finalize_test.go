package agent

import (
	"context"
	"strings"
	"testing"

	"github.com/camronwood/neural-junkie/internal/protocol"
)

type stubPlanAI struct {
	replies []string
	calls   int
}

func (s *stubPlanAI) GenerateResponse(_ context.Context, _ string, _ []protocol.Message) (string, error) {
	if s.calls >= len(s.replies) {
		return "", nil
	}
	out := s.replies[s.calls]
	s.calls++
	return out, nil
}

func (s *stubPlanAI) GenerateVisionResponse(context.Context, string, []byte, string, []protocol.Message) (string, error) {
	return "", nil
}

func (s *stubPlanAI) GetModel() string { return "stub" }

func TestPreparePlanMarkdown_normalizesPseudoYAML(t *testing.T) {
	raw := `yaml plan: hello-world actions: - description: Add HelloWorld to main.go`
	got, ok := preparePlanMarkdown(raw)
	if !ok {
		t.Fatalf("expected parseable plan, got %q", got)
	}
	for _, part := range []string{"name:", "todos:", "## Out of scope"} {
		if !strings.Contains(got, part) {
			t.Fatalf("missing %q in %q", part, got)
		}
	}
}

func TestRecoverMalformedPlanReply_strictRetry(t *testing.T) {
	mock := &stubPlanAI{replies: []string{`---
name: HelloWorld plan
overview: Add HelloWorld.
todos:
  - id: add-fn
    content: Add HelloWorld in main.go
    status: pending
isProject: false
---

# HelloWorld

## Out of scope

- Tests
`}}
	a := NewAgent(protocol.AgentTypeAssistant, "Planner", nil, mock, nil)
	msg := &protocol.Message{Metadata: map[string]interface{}{"editor_mode": "plan", "composer_mode": "plan"}}
	got, ok := a.recoverMalformedPlanReply(context.Background(), msg, mock, "yaml plan: broken")
	if !ok {
		t.Fatal("expected retry success")
	}
	if _, ok := preparePlanMarkdown(got); !ok {
		t.Fatalf("retry not parseable: %q", got)
	}
}
