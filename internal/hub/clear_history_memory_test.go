package hub

import (
	"path/filepath"
	"testing"

	"github.com/camronwood/neural-junkie/internal/memory"
	"github.com/camronwood/neural-junkie/internal/protocol"
)

func TestClearChannelHistory_clearsMemory(t *testing.T) {
	dir := t.TempDir()
	store, err := memory.Open(filepath.Join(dir, "memory.db"))
	if err != nil {
		t.Fatal(err)
	}
	defer store.Close()
	memory.SetStore(store)

	_ = store.UpsertChunk(memory.Chunk{
		ID: "msg:1", SourceType: memory.SourceMessage, SourceID: "1", Channel: "mem-clear-ch",
		Content: "hello", ContentHash: "h",
	})

	h := NewHub()
	name := "mem-clear-ch"
	_ = h.CreateChannel(name, "c", "test")
	from := protocol.AgentInfo{ID: "u1", Name: "Camron", Type: "human"}
	m := protocol.NewMessage(protocol.MessageTypeQuestion, name, from, "hi")
	m.ID = "m1"
	h.mu.Lock()
	h.appendChannelMessageLocked(name, m)
	h.mu.Unlock()

	if err := h.ClearChannelHistory(name); err != nil {
		t.Fatal(err)
	}
	cands, _ := store.ListCandidates(name, "", 10)
	if len(cands) != 0 {
		t.Fatalf("expected memory cleared, got %d chunks", len(cands))
	}
}
