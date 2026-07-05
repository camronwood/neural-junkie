package main

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/camronwood/neural-junkie/internal/hub"
	"github.com/camronwood/neural-junkie/internal/protocol"
)

func TestHandleDebugTurnTraceStructuredShape(t *testing.T) {
	origHub := chatHub
	t.Cleanup(func() { chatHub = origHub })

	h := hub.NewHub()
	h.CreateChannel("general", "", "")
	chatHub = h

	userMsg := protocol.NewMessage(protocol.MessageTypeChat, "general", protocol.AgentInfo{ID: "u1", Name: "User", Type: "human"}, "@codebase review auth")
	target := protocol.NewMessage(protocol.MessageTypeChat, "general", protocol.AgentInfo{ID: "a1", Name: "Dev", Type: protocol.AgentTypeBackend}, "reply")
	target.ReplyTo = userMsg.ID
	protocol.ApplyRoutingMeta(target, protocol.RoutingMeta{
		Model:           "qwen3.5:9b",
		Domain:          "software",
		CostTier:        "standard",
		KnowledgeRoute:  "codebase",
		KnowledgeReason: "codebase_cue",
		Reason:          "capability_routing",
		Source:          "capabilities",
		ComposerMode:    "agent",
		ContextScope:    "workspace",
		ImplSession:     true,
	})
	_ = h.SendMessage(userMsg)
	_ = h.SendMessage(target)

	req := httptest.NewRequest(http.MethodGet, "/api/debug/turn-trace?channel=general&message_id="+userMsg.ID, nil)
	rec := httptest.NewRecorder()
	handleDebugTurnTrace(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d body = %s", rec.Code, rec.Body.String())
	}
	var out map[string]interface{}
	if err := json.Unmarshal(rec.Body.Bytes(), &out); err != nil {
		t.Fatal(err)
	}
	routing, _ := out["routing"].(map[string]interface{})
	if routing["model"] != "qwen3.5:9b" {
		t.Fatalf("routing model = %v", routing["model"])
	}
	retrieval, _ := out["retrieval"].(map[string]interface{})
	if retrieval["mode"] != "codebase" {
		t.Fatalf("retrieval mode = %v", retrieval["mode"])
	}
	governance, _ := out["governance"].(map[string]interface{})
	if governance["composer_mode"] != "agent" {
		t.Fatalf("composer_mode = %v", governance["composer_mode"])
	}
}
