package agent

import (
	"testing"

	"github.com/camronwood/neural-junkie/internal/protocol"
)

func TestFormatRoutingThinkingDetailWithTierAndRetrieval(t *testing.T) {
	detail := formatRoutingThinkingDetailForTest(RoutingSnapshot{
		ChatModel:      "qwen3.5:9b",
		CostTier:       "standard",
		KnowledgeRoute: "codebase",
		Reason:         "capability_routing",
	})
	want := "chat: qwen3.5:9b · tier: standard · retrieval: codebase (capability_routing)"
	if detail != want {
		t.Fatalf("detail = %q, want %q", detail, want)
	}
}

func TestFormatRoutingThinkingDetailRetrievalOnly(t *testing.T) {
	detail := formatRoutingThinkingDetailForTest(RoutingSnapshot{
		KnowledgeRoute: "conversation_memory",
		CostTier:       "cheap",
	})
	want := "tier: cheap · retrieval: conversation_memory"
	if detail != want {
		t.Fatalf("detail = %q, want %q", detail, want)
	}
}

func TestRoutingTelemetryPayloadIncludesGovernance(t *testing.T) {
	msg := protocol.NewMessage(protocol.MessageTypeChat, "#general", protocol.AgentInfo{ID: "u1", Name: "User", Type: "human"}, "review turn_intent.go")
	msg.Metadata = map[string]interface{}{
		protocol.TurnMetaComposerMode: "agent",
		"context_scope":               "workspace",
	}
	payload := routingTelemetryPayloadForTest(RoutingSnapshot{
		ChatModel:       "qwen3.5:9b",
		Domain:          "software",
		CostTier:        "standard",
		KnowledgeRoute:  "codebase",
		KnowledgeReason: "codebase_cue",
		Reason:          "capability_routing",
		Source:          "capabilities",
	}, msg)

	for _, key := range []string{"chat_model", "domain", "cost_tier", "knowledge_route", "knowledge_reason"} {
		if payload[key] == nil || payload[key] == "" {
			t.Fatalf("expected payload[%q] to be set", key)
		}
	}
	gov, ok := payload["governance"].(map[string]interface{})
	if !ok {
		t.Fatal("expected governance map in payload")
	}
	if gov["composer_mode"] != "agent" {
		t.Fatalf("composer_mode = %v", gov["composer_mode"])
	}
	if gov["context_scope"] != "workspace" {
		t.Fatalf("context_scope = %v", gov["context_scope"])
	}
}
