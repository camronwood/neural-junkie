package hub

import (
	"log"

	"github.com/camronwood/neural-junkie/internal/collaboration"
	"github.com/camronwood/neural-junkie/internal/protocol"
)

func (h *Hub) maybeSyncTaskStatusFromPlanHandoff(msg *protocol.Message, collabID string) {
	if msg == nil || h.collabManager == nil || msg.IsFromSystem() {
		return
	}
	if msg.Type != protocol.MessageTypeChat &&
		msg.Type != protocol.MessageTypeAnswer &&
		msg.Type != protocol.MessageTypeCollabDiscussion {
		return
	}

	snap, err := h.collabManager.GetCollaborationSnapshot(collabID)
	if err != nil || snap == nil || snap.Phase != collaboration.PhaseExecuting {
		return
	}

	taskIDs := collaboration.SyncTaskStatusFromPlanHandoff(msg.Content, snap.Tasks)
	if len(taskIDs) == 0 {
		return
	}

	channel := msg.Channel
	if channel == "" {
		channel = snap.Channel
	}

	for _, taskID := range taskIDs {
		if err := h.collabManager.UpdateTaskStatus(collabID, taskID, collaboration.TaskCompleted, "Marked complete from plan handoff"); err != nil {
			log.Printf("[Collaboration] plan handoff task update %s: %v", taskID[:8], err)
			continue
		}
	}

	if h.collabManager.AllTasksComplete(collabID) {
		h.requestFinalRecapAndFinalize(collabID, channel, "All tasks are done.", collaboration.FinalizeOptions{})
	}
}
