package agent

import (
	"log"
	"strings"
	"sync"

	"github.com/camronwood/neural-junkie/internal/agent/checkpoint"
	"github.com/camronwood/neural-junkie/internal/protocol"
)

var (
	checkpointOnce  sync.Once
	checkpointStore *checkpoint.Store
	checkpointErr   error
)

func implCheckpointStore() *checkpoint.Store {
	checkpointOnce.Do(func() {
		checkpointStore, checkpointErr = checkpoint.Open("")
		if checkpointErr != nil {
			log.Printf("agent checkpoint store: %v", checkpointErr)
		}
	})
	return checkpointStore
}

// implCheckpointID keys checkpoints by turn/message (or collab task), never by
// channel alone — channel-only keys leaked FilesChanged / repair state across
// unrelated turns on the same channel.
func implCheckpointID(msg *protocol.Message) string {
	if msg == nil {
		return ""
	}
	collabID := msg.GetCollaborationID()
	taskID := msg.GetTaskID()
	if collabID != "" && taskID != "" {
		return collabID + ":" + taskID
	}
	turnID := strings.TrimSpace(msg.ID)
	if turnID == "" {
		return ""
	}
	base := strings.TrimSpace(msg.Channel)
	if msg.ThreadID != "" {
		base = base + ":" + msg.ThreadID
	}
	if base == "" {
		return "turn:" + turnID
	}
	return base + ":turn:" + turnID
}

func restoreImplSessionFromCheckpoint(msg *protocol.Message, state *ImplementationSessionState) {
	if msg == nil || state == nil || !agentRuntimeV2ForMessage(msg) {
		return
	}
	id := implCheckpointID(msg)
	if id == "" {
		return
	}
	st := implCheckpointStore()
	if st == nil {
		return
	}
	ck, err := st.Load(id)
	if err != nil || ck == nil || ck.Payload == nil {
		return
	}
	// Refuse restore when payload turn id does not match this message.
	if turn, _ := ck.Payload["turn_message_id"].(string); turn != "" && turn != strings.TrimSpace(msg.ID) {
		return
	}
	if v, ok := ck.Payload["edit_round"].(float64); ok {
		state.EditRound = int(v)
	}
	if v, ok := ck.Payload["repair_attempts"].(float64); ok {
		state.RepairAttempts = int(v)
		state.RepairUsed = state.RepairAttempts > 0
	}
	if v, ok := ck.Payload["phase"].(string); ok && v != "" {
		state.Phase = v
	}
	if files, ok := ck.Payload["files_changed"].([]interface{}); ok {
		for _, f := range files {
			if s, ok := f.(string); ok && s != "" {
				state.FilesChanged = append(state.FilesChanged, s)
			}
		}
	}
	if v, ok := ck.Payload["verify_output"].(string); ok {
		state.VerifyOutput = v
	}
}

func persistImplSessionCheckpoint(msg *protocol.Message, state *ImplementationSessionState, step int) {
	if msg == nil || state == nil || !agentRuntimeV2ForMessage(msg) {
		return
	}
	st := implCheckpointStore()
	if st == nil {
		return
	}
	id := implCheckpointID(msg)
	if id == "" {
		return
	}
	// Clear on complete so the next turn cannot inherit stale state.
	if step < 0 {
		_ = st.Delete(id)
		return
	}
	files := make([]interface{}, len(state.FilesChanged))
	for i, f := range state.FilesChanged {
		files[i] = f
	}
	_ = st.Save(checkpoint.State{
		ID:       id,
		Channel:  msg.Channel,
		ThreadID: msg.ThreadID,
		Step:     step,
		Payload: map[string]interface{}{
			"edit_round":       state.EditRound,
			"repair_attempts":  state.RepairAttempts,
			"phase":            state.Phase,
			"files_changed":    files,
			"proposed_count":   state.ProposedCount,
			"verify_output":    state.VerifyOutput,
			"collab_id":        msg.GetCollaborationID(),
			"task_id":          msg.GetTaskID(),
			"turn_message_id":  strings.TrimSpace(msg.ID),
			"agent_runtime_v2": true,
		},
	})
}

func clearImplSessionCheckpoint(msg *protocol.Message) {
	if msg == nil || !agentRuntimeV2ForMessage(msg) {
		return
	}
	st := implCheckpointStore()
	if st == nil {
		return
	}
	if id := implCheckpointID(msg); id != "" {
		_ = st.Delete(id)
	}
}
