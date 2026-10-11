package agent

import (
	"path/filepath"
	"testing"

	"github.com/camronwood/neural-junkie/internal/agent/checkpoint"
	"github.com/camronwood/neural-junkie/internal/protocol"
)

func TestImplCheckpointID_turnScoped(t *testing.T) {
	msg1 := protocol.NewMessage(protocol.MessageTypeQuestion, "general", protocol.AgentInfo{}, "edit a")
	msg1.ID = "turn-1"
	msg2 := protocol.NewMessage(protocol.MessageTypeQuestion, "general", protocol.AgentInfo{}, "edit b")
	msg2.ID = "turn-2"
	id1 := implCheckpointID(msg1)
	id2 := implCheckpointID(msg2)
	if id1 == "" || id2 == "" {
		t.Fatal("expected non-empty checkpoint ids")
	}
	if id1 == id2 {
		t.Fatalf("unrelated turns must not share checkpoint id: %q", id1)
	}
	if id1 != "general:turn:turn-1" {
		t.Fatalf("id1=%q", id1)
	}
}

func TestImplCheckpointID_collabTask(t *testing.T) {
	msg := protocol.NewMessage(protocol.MessageTypeCollabTask, "collab-scenarios", protocol.AgentInfo{}, "task")
	msg.ID = "msg-9"
	msg.SetCollaborationID("c1")
	msg.SetTaskID("t1")
	if got := implCheckpointID(msg); got != "c1:t1" {
		t.Fatalf("got %q", got)
	}
}

func TestCheckpointRestore_isolatedByTurn(t *testing.T) {
	dir := t.TempDir()
	store, err := checkpoint.Open(filepath.Join(dir, "ck.db"))
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = store.Close() })

	checkpointOnce.Do(func() {}) // ensure Once is spent so Open("") cannot overwrite
	prev := checkpointStore
	checkpointStore = store
	checkpointErr = nil
	t.Cleanup(func() { checkpointStore = prev })

	msg1 := protocol.NewMessage(protocol.MessageTypeQuestion, "implement-scenarios", protocol.AgentInfo{}, "first")
	msg1.ID = "m1"
	msg1.Metadata = map[string]interface{}{"agent_runtime_v2": true}
	state1 := &ImplementationSessionState{
		EditRound:    2,
		FilesChanged: []string{"stale.go"},
		Phase:        "edit",
	}
	persistImplSessionCheckpoint(msg1, state1, 0)

	msg2 := protocol.NewMessage(protocol.MessageTypeQuestion, "implement-scenarios", protocol.AgentInfo{}, "second")
	msg2.ID = "m2"
	msg2.Metadata = map[string]interface{}{"agent_runtime_v2": true}
	state2 := &ImplementationSessionState{Phase: "discover"}
	restoreImplSessionFromCheckpoint(msg2, state2)
	if len(state2.FilesChanged) != 0 || state2.EditRound != 0 {
		t.Fatalf("new turn restored unrelated checkpoint: %+v", state2)
	}

	// Same turn can restore.
	state1b := &ImplementationSessionState{Phase: "discover"}
	restoreImplSessionFromCheckpoint(msg1, state1b)
	if state1b.EditRound != 2 || len(state1b.FilesChanged) != 1 {
		t.Fatalf("same-turn restore failed: %+v", state1b)
	}

	// Clear on complete.
	persistImplSessionCheckpoint(msg1, state1, -1)
	state1c := &ImplementationSessionState{Phase: "discover"}
	restoreImplSessionFromCheckpoint(msg1, state1c)
	if state1c.EditRound != 0 || len(state1c.FilesChanged) != 0 {
		t.Fatalf("checkpoint should be cleared on complete: %+v", state1c)
	}
}
