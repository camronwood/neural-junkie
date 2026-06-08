package memory

import (
	"testing"

	"github.com/camronwood/neural-junkie/internal/protocol"
)

func TestChunkText_splitsLongContent(t *testing.T) {
	text := stringsRepeat("word ", 200)
	parts := ChunkText(text, 100, 20)
	if len(parts) < 2 {
		t.Fatalf("expected multiple chunks, got %d", len(parts))
	}
}

func TestMessageChunks_skipsNoise(t *testing.T) {
	msg := protocol.NewMessage(protocol.MessageTypeAgentStatus, "ch", protocol.AgentInfo{ID: "s"}, "status")
	if len(MessageChunks(msg)) != 0 {
		t.Fatal("expected no chunks for agent status")
	}
}

func TestMessageChunks_userMessage(t *testing.T) {
	msg := protocol.NewMessage(protocol.MessageTypeQuestion, "dm-u-a", protocol.AgentInfo{ID: "u", Name: "Camron", Type: "human"}, "hello")
	msg.ID = "m1"
	chunks := MessageChunks(msg)
	if len(chunks) != 1 || chunks[0].SourceID != "m1" {
		t.Fatalf("unexpected chunks: %+v", chunks)
	}
}

func TestCollabIDFromRelPath(t *testing.T) {
	if got := CollabIDFromRelPath("collabs/abc-123/findings.md"); got != "abc-123" {
		t.Fatalf("got %q", got)
	}
}

func stringsRepeat(s string, n int) string {
	out := ""
	for i := 0; i < n; i++ {
		out += s
	}
	return out
}
