package agent

import (
	"testing"

	"github.com/camronwood/neural-junkie/internal/ai"
	"github.com/camronwood/neural-junkie/internal/protocol"
)

func TestShouldRespondToRepo_ConsultOnlyBlocksUnlessMentioned(t *testing.T) {
	hub := &captureHub{}
	ra, err := NewRepoAgentWithOptions("HiddenIndex", t.TempDir(), ai.NewMockProvider(), hub, RepoAgentOptions{ConsultOnly: true})
	if err != nil {
		t.Fatal(err)
	}

	msg := &protocol.Message{
		Channel: "general",
		Content: "what is the architecture?",
		From:    protocol.AgentInfo{ID: "u1", Name: "User", Type: "human"},
		Type:    protocol.MessageTypeQuestion,
	}
	if ra.shouldRespondToRepo(msg) {
		t.Fatal("consult-only agent should not auto-respond")
	}

	msg.Mentions = []string{ra.Info.ID}
	if !ra.shouldRespondToRepo(msg) {
		t.Fatal("consult-only agent should respond when explicitly mentioned")
	}
}

func TestShouldRunRepoConsult_TaskWithWorkspace(t *testing.T) {
	a := &Agent{
		Info: protocol.AgentInfo{Type: protocol.AgentTypeFrontend, Name: "FrontendEngineer"},
	}
	msg := &protocol.Message{
		Content: "fix the theme toggle",
		From:    protocol.AgentInfo{ID: "u1", Name: "User", Type: "human"},
		Metadata: map[string]interface{}{
			"workspace_context": map[string]interface{}{
				"workspace_path": t.TempDir(),
			},
		},
	}
	if !a.shouldRunRepoConsult(t.Context(), msg, IntentTask) {
		t.Fatal("expected repo consult on task with workspace")
	}
	if a.shouldRunRepoConsult(t.Context(), msg, IntentClosure) {
		t.Fatal("closure should skip repo consult")
	}
}
