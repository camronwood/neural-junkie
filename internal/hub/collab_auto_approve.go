package hub

import (
	"fmt"
	"log"
	"path/filepath"
	"strings"

	"github.com/camronwood/neural-junkie/internal/collaboration"
	"github.com/camronwood/neural-junkie/internal/filechange"
	"github.com/camronwood/neural-junkie/internal/protocol"
)

func (h *Hub) collabAutoApproveDeliverablesEnabled() bool {
	if h == nil || h.commandHandler == nil || h.commandHandler.appConfig == nil {
		return true
	}
	return h.commandHandler.appConfig.Collaboration.AutoApproveDeliverablesEnabled()
}

func isPathUnderCollabDeliverable(collabID, wsRoot, absPath string) bool {
	collabID = strings.TrimSpace(collabID)
	wsRoot = strings.TrimSpace(wsRoot)
	if collabID == "" || wsRoot == "" || absPath == "" {
		return false
	}
	rel, err := filepath.Rel(wsRoot, absPath)
	if err != nil {
		return false
	}
	rel = filepath.ToSlash(rel)
	if rel == ".." || strings.HasPrefix(rel, "../") {
		return false
	}
	prefix := collaboration.ProjectCollabRelPath(collabID)
	if prefix == "" {
		return false
	}
	return rel == prefix || strings.HasPrefix(rel, prefix+"/")
}

func (h *Hub) maybeAutoApproveCollabFileChange(msg *protocol.Message, change *filechange.FileChange, operation filechange.FileOperation, wsRoot string) {
	if h == nil || msg == nil || change == nil || h.fileChangeManager == nil || h.collabManager == nil {
		return
	}
	if !h.collabAutoApproveDeliverablesEnabled() {
		return
	}
	collabID := strings.TrimSpace(msg.GetCollaborationID())
	if collabID == "" {
		return
	}
	if operation != filechange.FileOperationCreate && operation != filechange.FileOperationEdit {
		return
	}
	snap, err := h.collabManager.GetCollaborationSnapshot(collabID)
	if err != nil || snap == nil || snap.Phase != collaboration.PhaseExecuting {
		return
	}
	if !isPathUnderCollabDeliverable(collabID, wsRoot, change.FilePath) {
		return
	}

	approvedBy := "system"
	if msg.From.ID != "" {
		approvedBy = msg.From.ID
	}
	approved, err := h.fileChangeManager.ApproveFileChange(change.ID, approvedBy)
	if err != nil {
		log.Printf("[Collaboration] Auto-approve deliverable %s for %s: %v", change.ID, collabID[:8], err)
		return
	}
	h.NotifyFileChangeApproved(approved, approvedBy)

	relPath := change.FilePath
	if rel, err := filepath.Rel(wsRoot, change.FilePath); err == nil {
		relPath = filepath.ToSlash(rel)
	}
	channel := msg.Channel
	if channel == "" {
		channel = snap.Channel
	}
	h.broadcastCollabSystem(channel, collabID, fmt.Sprintf("✅ **Auto-approved collab deliverable:** `%s`", relPath))
	h.maybeCompleteCollabTasksAfterDeliverable(collabID, channel)
}

func (h *Hub) maybeCompleteCollabTasksAfterDeliverable(collabID, channel string) {
	if h == nil || h.collabManager == nil {
		return
	}
	snap, err := h.collabManager.GetCollaborationSnapshot(collabID)
	if err != nil || snap == nil || snap.Phase != collaboration.PhaseExecuting {
		return
	}
	for _, task := range snap.Tasks {
		// File proposals can be registered before the agent's final task reply
		// advances pending -> in_progress. A dispatched task with its concrete
		// deliverable already auto-approved is complete regardless of that race.
		if task.Status != collaboration.TaskInProgress &&
			!(task.Status == collaboration.TaskPending && task.PromptDispatched) {
			continue
		}
		if !collaboration.NewDeliverablePolicy(task, snap.Description, nil).RequiresFile() {
			continue
		}
		if !h.collabTaskDeliverableSatisfied(snap, &task, nil) {
			continue
		}
		effects, err := h.collabManager.UpdateTaskStatusWithEffects(collabID, task.ID, collaboration.TaskCompleted, "Deliverable file auto-approved")
		if err != nil {
			log.Printf("[Collaboration] Auto-complete task %s after deliverable: %v", task.ID[:8], err)
			continue
		}
		if h.collabManager.AllTasksComplete(collabID) {
			h.requestFinalRecapAndFinalize(collabID, channel, "All tasks are done.", collaboration.FinalizeOptions{})
			return
		}
		if effects.ShouldDispatchWave {
			if fresh, err := h.collabManager.GetCollaborationSnapshot(collabID); err == nil && fresh != nil {
				h.dispatchReadyCollabTasks(fresh, nil, false)
			}
		}
	}
}
