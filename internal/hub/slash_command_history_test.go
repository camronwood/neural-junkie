package hub

import (
	"strings"
	"testing"

	"github.com/camronwood/neural-junkie/internal/protocol"
)

func TestSendMessageSlashCommandPersistsHumanAndSystemResponse(t *testing.T) {
	h := NewHub()
	ch := "slash-history-test"
	h.CreateChannel(ch, "Slash history", "test")

	user := protocol.AgentInfo{ID: "user-1", Name: "Camron", Type: protocol.AgentTypeGeneral}
	msg := protocol.NewMessage(protocol.MessageTypeQuestion, ch, user, "/help")

	if err := h.SendMessage(msg); err != nil {
		t.Fatalf("SendMessage: %v", err)
	}

	msgs, err := h.GetMessages(ch, 50)
	if err != nil {
		t.Fatalf("GetMessages: %v", err)
	}
	if len(msgs) < 2 {
		t.Fatalf("expected at least 2 messages (command + response), got %d", len(msgs))
	}

	var commandLine, systemReply *protocol.Message
	for _, m := range msgs {
		if m == nil {
			continue
		}
		if m.Content == "/help" {
			commandLine = m
		}
		if m.Content != "/help" && (strings.Contains(m.Content, "Available Commands") || m.From.Name == "System") {
			if systemReply == nil || strings.Contains(m.Content, "Available Commands") {
				systemReply = m
			}
		}
	}
	if commandLine == nil {
		t.Fatal("human /help command line not found in channel history")
	}
	if commandLine.Metadata == nil || commandLine.Metadata[protocol.MetadataSlashCommand] != true {
		t.Fatalf("expected slash_command metadata on human line, got %#v", commandLine.Metadata)
	}
	if systemReply == nil {
		t.Fatal("system help response not found in channel history")
	}
	if commandLine.ID == systemReply.ID {
		t.Fatal("command line and system response should be distinct messages")
	}
}

func TestSendMessageSlashCommandUnknownStillPersistsCommandLine(t *testing.T) {
	h := NewHub()
	ch := "slash-unknown-test"
	h.CreateChannel(ch, "Slash unknown", "test")

	user := protocol.AgentInfo{ID: "user-1", Name: "Camron", Type: protocol.AgentTypeGeneral}
	msg := protocol.NewMessage(protocol.MessageTypeQuestion, ch, user, "/not-a-real-command-xyz")

	if err := h.SendMessage(msg); err != nil {
		t.Fatalf("SendMessage: %v", err)
	}

	msgs, err := h.GetMessages(ch, 50)
	if err != nil {
		t.Fatalf("GetMessages: %v", err)
	}
	foundCommand := false
	for _, m := range msgs {
		if m != nil && m.Content == "/not-a-real-command-xyz" {
			foundCommand = true
			break
		}
	}
	if !foundCommand {
		t.Fatal("unknown slash command line not persisted")
	}
}
