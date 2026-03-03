package main

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/camronwood/neural-junkie/internal/collaboration"
	"github.com/camronwood/neural-junkie/internal/hub"
	"github.com/camronwood/neural-junkie/internal/protocol"
)

func TestHandleCollaborationsFiltersAndTerminalToggle(t *testing.T) {
	chatHub = hub.NewHub()
	chatHub.CreateChannel("proj-x", "Project X", "")

	agentA := &protocol.AgentInfo{ID: "a1", Name: "AgentA", Type: protocol.AgentTypeBackend, Status: "active"}
	agentB := &protocol.AgentInfo{ID: "a2", Name: "AgentB", Type: protocol.AgentTypeFrontend, Status: "active"}
	if err := chatHub.RegisterAgent(agentA); err != nil {
		t.Fatalf("register agentA: %v", err)
	}
	if err := chatHub.RegisterAgent(agentB); err != nil {
		t.Fatalf("register agentB: %v", err)
	}

	cm := chatHub.GetCollaborationManager()
	activeCollab, err := cm.CreateCollaboration("active general", []string{"a1", "a2"}, "general", "tester", collaboration.DiscussionConfig{})
	if err != nil {
		t.Fatalf("create active collaboration: %v", err)
	}
	terminalCollab, err := cm.CreateCollaboration("completed general", []string{"a1", "a2"}, "general", "tester", collaboration.DiscussionConfig{})
	if err != nil {
		t.Fatalf("create terminal collaboration: %v", err)
	}
	if _, err := cm.CompleteCollaboration(terminalCollab.ID); err != nil {
		t.Fatalf("complete collaboration: %v", err)
	}
	if _, err := cm.CreateCollaboration("active proj", []string{"a1", "a2"}, "proj-x", "tester", collaboration.DiscussionConfig{}); err != nil {
		t.Fatalf("create proj collaboration: %v", err)
	}

	req := httptest.NewRequest(http.MethodGet, "/api/collaborations?channel=general", nil)
	rec := httptest.NewRecorder()
	handleCollaborations(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", rec.Code)
	}
	var collabs []collaboration.Collaboration
	if err := json.Unmarshal(rec.Body.Bytes(), &collabs); err != nil {
		t.Fatalf("decode collaborations: %v", err)
	}
	if len(collabs) != 1 {
		t.Fatalf("expected 1 active general collaboration, got %d", len(collabs))
	}
	if collabs[0].ID != activeCollab.ID {
		t.Fatalf("expected active collaboration %s, got %s", activeCollab.ID, collabs[0].ID)
	}

	req = httptest.NewRequest(http.MethodGet, "/api/collaborations?channel=general&include_terminal=true", nil)
	rec = httptest.NewRecorder()
	handleCollaborations(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", rec.Code)
	}
	collabs = nil
	if err := json.Unmarshal(rec.Body.Bytes(), &collabs); err != nil {
		t.Fatalf("decode collaborations with terminal: %v", err)
	}
	if len(collabs) != 2 {
		t.Fatalf("expected 2 general collaborations when include_terminal=true, got %d", len(collabs))
	}
}
