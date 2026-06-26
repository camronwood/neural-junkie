package agent

import (
	"testing"

	"github.com/camronwood/neural-junkie/internal/protocol"
)

func TestShouldAppendFileChangeApprovalPrompt(t *testing.T) {
	auto := &protocol.Message{Metadata: map[string]interface{}{"editor_agent_trust": "auto_apply_edits"}}
	if shouldAppendFileChangeApprovalPrompt(auto) {
		t.Fatal("auto_apply should not append approval prompt")
	}
	interactive := &protocol.Message{Metadata: map[string]interface{}{"editor_agent_trust": "interactive"}}
	if !shouldAppendFileChangeApprovalPrompt(interactive) {
		t.Fatal("interactive should append approval prompt")
	}
}

func TestImplementationSessionOutcomeRegistrationFailed(t *testing.T) {
	a := &Agent{}
	msg := &protocol.Message{Metadata: map[string]interface{}{"editor_agent_trust": "auto_apply_edits"}}
	state := &ImplementationSessionState{
		ProposedCount:        2,
		FilesChanged:         []string{"Makefile"},
		RegistrationErrors:   []string{"src-tauri/vite.config.js: preflight rejected"},
	}
	outcome := a.buildImplementationSessionOutcome(msg, state, false)
	if outcome["outcome"] != "proposal_registration_failed" {
		t.Fatalf("outcome = %v", outcome["outcome"])
	}
}
