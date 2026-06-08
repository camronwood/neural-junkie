package agent

import (
	"strings"
	"testing"

	"github.com/camronwood/neural-junkie/internal/protocol"
)

func TestShouldDeferImplSessionForCombinedDeliveryExport(t *testing.T) {
	a := &Agent{
		Info: protocol.AgentInfo{ID: "a1", Name: "Assistant", Type: protocol.AgentTypeAssistant},
		Context: &ConversationContext{
			History: make(map[string][]*protocol.Message),
		},
	}
	msg := protocol.NewMessage(
		protocol.MessageTypeChat,
		"general",
		protocol.AgentInfo{ID: "u1", Name: "camronwood", Type: protocol.AgentTypeGeneral},
		"Can you write me a LinkedIn artical about the app in the workspace and save the file to the root of the workspace?",
	)
	if !shouldDeferImplSessionForCombinedDeliveryExport(a, msg) {
		t.Fatal("expected combined write+save to defer impl session")
	}
	msg2 := protocol.NewMessage(
		protocol.MessageTypeChat,
		"dm-test",
		protocol.AgentInfo{ID: "u1", Name: "User", Type: "human"},
		"store that artical in nj-artical-1.md",
	)
	if shouldDeferImplSessionForCombinedDeliveryExport(a, msg2) {
		t.Fatal("export-only should not defer")
	}
}

func TestShouldRunImplementationSession_combinedDeliveryExport(t *testing.T) {
	a := &Agent{
		Info: protocol.AgentInfo{ID: "a1", Name: "Assistant", Type: protocol.AgentTypeAssistant},
		Context: &ConversationContext{
			History: make(map[string][]*protocol.Message),
		},
	}
	msg := protocol.NewMessage(
		protocol.MessageTypeChat,
		"general",
		protocol.AgentInfo{ID: "u1", Name: "User", Type: "human"},
		"Write a LinkedIn article about this app and save it to article.md",
	)
	msg.Metadata = map[string]interface{}{
		"editor_mode":            "export",
		"implementation_session": true,
	}
	if shouldRunImplementationSession(a, msg) {
		t.Fatal("combined delivery+export should not run impl session without prior content")
	}
}

func TestExtractContentDeliveryBodyForExport(t *testing.T) {
	resp := "Certainly! Here's a draft:\n\n---\n\n### Introducing Neural Junkie\n\n" + stringsRepeat("Body paragraph. ", 30)
	body := extractContentDeliveryBodyForExport(resp)
	if !strings.Contains(body, "### Introducing Neural Junkie") {
		t.Fatalf("expected article body, got %q", body[:minLen(80, len(body))])
	}
}

func TestDefaultContentDeliveryExportPath(t *testing.T) {
	msg := protocol.NewMessage(protocol.MessageTypeChat, "general", protocol.AgentInfo{}, "write a linkedin post and save it")
	if got := defaultContentDeliveryExportPath(msg); got != "linkedin-article.md" {
		t.Fatalf("path=%q", got)
	}
}
