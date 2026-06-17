package hub

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/camronwood/neural-junkie/internal/agent"
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

func TestPrepareSessionSnapshotStripsPlanningDiscussionCollaborationData(t *testing.T) {
	h := NewHub()
	ch := "persist-planning-disc"
	_ = h.CreateChannel(ch, "c", "test")

	planMsg := protocol.NewMessage(
		protocol.MessageTypeCollabDiscussion,
		ch,
		protocol.AgentInfo{ID: "a1", Name: "Agent", Type: protocol.AgentTypeBackend},
		"planning turn",
	)
	planMsg.Metadata = map[string]interface{}{
		"collaboration_data": map[string]interface{}{
			"id":    "collab-1",
			"phase": string(collaboration.PhasePlanning),
		},
	}

	snap := &SessionSnapshot{
		SavedAt: time.Now(),
		Channels: map[string]*ChannelSnapshot{
			ch: {Name: ch, Messages: []*protocol.Message{}},
		},
		Collaborations: map[string]*collaboration.Collaboration{
			"collab-1": {
				ID:      "collab-1",
				Channel: ch,
				Phase:   collaboration.PhaseExecuting,
				PlanningDiscussion: &collaboration.DiscussionSession{
					ID:              "disc-1",
					CollaborationID: "collab-1",
					Messages:        []*protocol.Message{planMsg},
				},
			},
		},
	}

	prepareSessionSnapshotForPersist(snap)
	c := snap.Collaborations["collab-1"]
	if c == nil || c.PlanningDiscussion == nil || len(c.PlanningDiscussion.Messages) == 0 {
		t.Fatal("expected planning discussion messages")
	}
	md := c.PlanningDiscussion.Messages[0].Metadata
	if md != nil {
		if _, ok := md["collaboration_data"]; ok {
			t.Fatal("collaboration_data should be stripped from planning_discussion messages")
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

func TestPrepareSessionSnapshotStripsWorkspaceContextBodies(t *testing.T) {
	h := NewHub()
	ch := "persist-ws"
	_ = h.CreateChannel(ch, "c", "test")

	bigContent := strings.Repeat("x", 50_000)
	msg := protocol.NewMessage(protocol.MessageTypeChat, ch, protocol.AgentInfo{ID: "u", Name: "Camron", Type: protocol.AgentTypeGeneral}, "review code")
	msg.Metadata = map[string]interface{}{
		"workspace_context": map[string]interface{}{
			"workspace_path": "/proj",
			"workspace_name": "proj",
			"file_tree":      strings.Repeat("a", 20_000),
			"open_files": []interface{}{
				map[string]interface{}{"path": "main.go", "content": bigContent},
			},
		},
		"prompt_attachments": []interface{}{
			map[string]interface{}{"type": "file", "content": bigContent},
		},
		agent.MetadataGrantedHubDataAccess: map[string]interface{}{
			"entries": []interface{}{
				map[string]interface{}{"path": "last-session.json", "content": bigContent},
			},
		},
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
	md := msgs[0].Metadata
	if md == nil {
		t.Fatal("expected metadata")
	}
	if _, ok := md["prompt_attachments"]; ok {
		t.Fatal("prompt_attachments should be stripped from persisted messages")
	}
	ws, ok := md["workspace_context"].(map[string]interface{})
	if !ok {
		t.Fatal("expected slim workspace_context")
	}
	if _, ok := ws["open_files"]; ok {
		t.Fatal("open_files bodies should be stripped from persisted workspace_context")
	}
	tree, _ := ws["file_tree"].(string)
	if len(tree) > maxPersistFileTreeBytes+32 {
		t.Fatalf("file_tree should be truncated, got len %d", len(tree))
	}
	grant, ok := md[agent.MetadataGrantedHubDataAccess].(map[string]interface{})
	if !ok {
		t.Fatal("expected slim granted_hub_data_access")
	}
	entries, _ := grant["entries"].([]interface{})
	if len(entries) != 1 {
		t.Fatalf("expected 1 grant entry, got %d", len(entries))
	}
	entry, _ := entries[0].(map[string]interface{})
	if _, ok := entry["content"]; ok {
		t.Fatal("granted hub data content should be stripped on persist")
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
