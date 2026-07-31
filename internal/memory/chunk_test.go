package memory

import (
	"strings"
	"testing"
	"unicode"
	"unicode/utf8"

	"github.com/camronwood/neural-junkie/internal/protocol"
)

func TestChunkText_splitsLongContent(t *testing.T) {
	text := stringsRepeat("word ", 200)
	parts := ChunkText(text, 100, 20)
	if len(parts) < 2 {
		t.Fatalf("expected multiple chunks, got %d", len(parts))
	}
}

func TestChunkText_prefersSentenceBoundary(t *testing.T) {
	text := strings.Repeat("alpha beta gamma. ", 40) + "FINAL_SENTENCE_MARKER ends here. " + strings.Repeat("noise word ", 40)
	parts := ChunkText(text, 200, 40)
	if len(parts) < 2 {
		t.Fatalf("expected split, got %d", len(parts))
	}
	// First chunk should end near a sentence boundary, not mid-word.
	first := strings.TrimSpace(parts[0])
	lastRune, _ := utf8.DecodeLastRuneInString(first)
	if lastRune != '.' && !unicode.IsSpace(lastRune) {
		// softChunkEnd returns index after '.', so trimmed content should end with '.'
		if !strings.HasSuffix(first, ".") {
			t.Fatalf("expected sentence-boundary end, got %q", first[len(first)-20:])
		}
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
