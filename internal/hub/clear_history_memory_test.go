package hub

import (
	"path/filepath"
	"testing"
	"time"

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

func TestClearChannelHistory_dismissesPendingUserQuestions(t *testing.T) {
	h := NewHub()
	name := "uq-clear-ch"
	_ = h.CreateChannel(name, "c", "test")
	uqm := h.GetUserQuestionManager()
	if uqm == nil {
		t.Fatal("expected user question manager")
	}
	done := make(chan struct{})
	go func() {
		_, _ = uqm.Ask("a1", "BE", name, "Pick one?", nil, 2*time.Second)
		close(done)
	}()
	time.Sleep(40 * time.Millisecond)
	if !uqm.HasPendingOnChannel(name) {
		t.Fatal("expected pending before clear")
	}
	if err := h.ClearChannelHistory(name); err != nil {
		t.Fatal(err)
	}
	<-done
	if uqm.HasPendingOnChannel(name) {
		t.Fatal("clear history should dismiss pending ask_user")
	}
}
