package hub

import (
	"strings"
	"testing"
	"time"

	"github.com/camronwood/neural-junkie/internal/protocol"
)

func TestExportChannelMessagesDedupesAndSorts(t *testing.T) {
	h := NewHub()
	ch := "export-test"
	h.CreateChannel(ch, "", "")
	from := protocol.AgentInfo{ID: "u1", Name: "User", Type: "human"}
	m1 := protocol.NewMessage(protocol.MessageTypeChat, ch, from, "first")
	m1.Timestamp = time.Now().Add(-2 * time.Minute)
	m2 := protocol.NewMessage(protocol.MessageTypeChat, ch, from, "second")
	m2.Timestamp = time.Now().Add(-1 * time.Minute)
	h.mu.Lock()
	h.messages[ch] = []*protocol.Message{m1, m2}
	h.mu.Unlock()

	out := h.ExportChannelMessages(ch)
	if len(out) != 2 {
		t.Fatalf("expected 2 messages, got %d", len(out))
	}
	if out[0].Content != "first" || out[1].Content != "second" {
		t.Fatalf("unexpected order: %q then %q", out[0].Content, out[1].Content)
	}
}

func TestFormatChannelExportMarkdown(t *testing.T) {
	from := protocol.AgentInfo{ID: "a1", Name: "Arch", Type: protocol.AgentTypeGeneral}
	msg := protocol.NewMessage(protocol.MessageTypeChat, "collab-1", from, "hello")
	md := FormatChannelExportMarkdown("collab-1", []*protocol.Message{msg})
	if !strings.Contains(md, "# Channel export: #collab-1") {
		t.Fatal("expected channel header")
	}
	if !strings.Contains(md, "Arch") || !strings.Contains(md, "hello") {
		t.Fatal("expected sender and content in markdown")
	}
}
