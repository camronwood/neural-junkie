package hub

import (
	"testing"

	"github.com/camronwood/neural-junkie/internal/agent"
	"github.com/camronwood/neural-junkie/internal/protocol"
)

func TestSendMessageAttachesUserRulesMetadata(t *testing.T) {
	h := NewHub()
	h.CreateChannel("general", "General", "")
	if err := h.SetUserRulesMarkdown("Camron", "My name is Camron."); err != nil {
		t.Fatal(err)
	}

	msg := protocol.NewMessage(
		protocol.MessageTypeChat,
		"general",
		protocol.AgentInfo{ID: "human-camronwood", Name: "camronwood", Type: "human"},
		"What is my name?",
	)
	h.AnnotateInboundUserMessage(msg, "Camron")
	if err := h.SendMessage(msg); err != nil {
		t.Fatal(err)
	}

	raw, ok := msg.Metadata[agent.MetadataUserRulesMarkdown]
	if !ok {
		t.Fatal("expected user_rules_markdown on inbound message")
	}
	if raw.(string) != "My name is Camron." {
		t.Fatalf("got rules %q", raw)
	}
}
