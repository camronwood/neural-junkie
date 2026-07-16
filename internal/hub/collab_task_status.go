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

	completedAny := false
	for _, taskID := range taskIDs {
		var task *collaboration.CollaborationTask
		for i := range snap.Tasks {
			if snap.Tasks[i].ID == taskID {
				task = &snap.Tasks[i]
				break
			}
		}
		if task == nil {
			continue
		}
		var optionPaths []string
		if task.Options != nil {
			optionPaths = task.Options.ContextPaths
		}
		contextPaths := collaboration.MergeContextPaths(
			collaboration.InferTaskContextPaths(*task, snap.SourceRepoPath),
			optionPaths,
		)
		policy := collaboration.NewDeliverablePolicy(*task, snap.Description, contextPaths)
		if policy.RequiresFile() && !h.collabTaskDeliverableSatisfied(snap, task, msg) {
			_ = h.maybeWarnPrematureTaskCompletion(msg, collabID, task, snap)
			short := taskID
			if len(short) > 8 {
				short = short[:8]
			}
			log.Printf("[Collaboration] plan handoff skipped complete for %s: file deliverable not satisfied", short)
			continue
		}
		if err := h.collabManager.UpdateTaskStatus(collabID, taskID, collaboration.TaskCompleted, "Marked complete from plan handoff"); err != nil {
			short := taskID
			if len(short) > 8 {
				short = short[:8]
			}
			log.Printf("[Collaboration] plan handoff task update %s: %v", short, err)
			continue
		}
		completedAny = true
	}

	if completedAny && h.collabManager.AllTasksComplete(collabID) {
		h.requestFinalRecapAndFinalize(collabID, channel, "All tasks are done.", collaboration.FinalizeOptions{})
	}
}
