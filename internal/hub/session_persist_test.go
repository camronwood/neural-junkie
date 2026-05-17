package hub

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/camronwood/neural-junkie/internal/collaboration"
	"github.com/camronwood/neural-junkie/internal/protocol"
)

func TestPrepareSessionSnapshotStripsCollaborationData(t *testing.T) {
	h := NewHub()
	ch := "persist-strip"
	_ = h.CreateChannel(ch, "c", "test")

	msg := protocol.NewMessage(protocol.MessageTypeChat, ch, protocol.AgentInfo{ID: "u", Name: "Camron", Type: protocol.AgentTypeGeneral}, "hello")
	msg.Metadata = map[string]interface{}{
		"collaboration_data": map[string]interface{}{"id": "x"},
	}
	h.mu.Lock()
	h.appendChannelMessageLocked(ch, msg)
	h.mu.Unlock()

	snap := h.TakeSessionSnapshot()
	prepareSessionSnapshotForPersist(snap)
	msgs := snap.Channels[ch].Messages
	if len(msgs) == 0 {
		t.Fatal("expected messages")
	}
	if msgs[0].Metadata != nil {
		if _, ok := msgs[0].Metadata["collaboration_data"]; ok {
			t.Fatal("collaboration_data should be stripped from persisted messages")
		}
	}
}

func TestSessionSaveBoundedSizeWithCollabMetadata(t *testing.T) {
	h := NewHub()
	ch := "persist-size"
	_ = h.CreateChannel(ch, "c", "test")

	_ = h.RegisterAgent(&protocol.AgentInfo{ID: "a1", Name: "A", Type: protocol.AgentTypeBackend, Status: "active"})
	_ = h.RegisterAgent(&protocol.AgentInfo{ID: "a2", Name: "B", Type: protocol.AgentTypeFrontend, Status: "active"})
	cm := h.GetCollaborationManager()
	collab, err := cm.CreateCollaboration("size test", []string{"a1", "a2"}, ch, "t", collaboration.DiscussionConfig{})
	if err != nil {
		t.Fatalf("create: %v", err)
	}
	collabID := collab.ID

	for i := 0; i < 100; i++ {
		m := protocol.NewMessage(protocol.MessageTypeChat, ch, protocol.AgentInfo{ID: "a1", Name: "A", Type: protocol.AgentTypeBackend}, "msg")
		m.SetCollaborationID(collabID)
		m.SetCollaborationPhase(string(collaboration.PhasePlanning))
		h.mu.Lock()
		h.attachCollaborationData(m)
		h.appendChannelMessageLocked(ch, m)
		h.mu.Unlock()
	}

	dir := t.TempDir()
	path := filepath.Join(dir, "last-session.json")
	if err := h.SaveSessionToFile(path); err != nil {
		t.Fatalf("save: %v", err)
	}
	fi, err := os.Stat(path)
	if err != nil {
		t.Fatalf("stat: %v", err)
	}
	const maxBytes = 5 * 1024 * 1024
	if fi.Size() > maxBytes {
		t.Fatalf("session file too large: %d bytes (max %d)", fi.Size(), maxBytes)
	}
}

func TestSessionRestoreArchivesOversizedFile(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "last-session.json")
	if err := os.WriteFile(path, make([]byte, MaxSessionRestoreBytes+1), 0644); err != nil {
		t.Fatalf("write: %v", err)
	}
	h := NewHub()
	if err := h.LoadSessionFromFile(path); err != nil {
		t.Fatalf("load: %v", err)
	}
	if _, err := os.Stat(path); !os.IsNotExist(err) {
		t.Fatal("expected oversized session file to be archived away")
	}
	matches, _ := filepath.Glob(filepath.Join(dir, "last-session.archived-*.json"))
	if len(matches) != 1 {
		t.Fatalf("expected one archived session file, got %d", len(matches))
	}
}
