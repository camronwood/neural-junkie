package hub

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/camronwood/neural-junkie/internal/collaboration"
	"github.com/camronwood/neural-junkie/internal/filechange"
	"github.com/camronwood/neural-junkie/internal/protocol"
)

func (h *Hub) collabTaskDeliverableSatisfied(snap *collaboration.Collaboration, task *collaboration.CollaborationTask, msg *protocol.Message) bool {
	if task == nil || snap == nil {
		return true
	}
	root := strings.TrimSpace(snap.SourceRepoPath)
	if root == "" {
		root = strings.TrimSpace(snap.WorkingDirectory)
	}
	for _, rel := range collaboration.ReferencedDeliverablePaths(*task) {
		rel = filepath.ToSlash(strings.Trim(rel, "`\"' "))
		if rel == "" {
			continue
		}
		if collaboration.UsesProjectCollabDir(snap) && strings.TrimSpace(snap.SourceRepoPath) != "" {
			abs := filepath.Join(snap.SourceRepoPath, filepath.FromSlash(rel))
			if st, err := os.Stat(abs); err == nil && !st.IsDir() {
				if !collaboration.IsDeliverableStubFile(abs) {
					return true
				}
			}
		}
		if root != "" {
			norm := collaboration.NormalizeDeliverableRelPathForRoot(snap, rel)
			abs := filepath.Join(root, filepath.FromSlash(norm))
			if st, err := os.Stat(abs); err == nil && !st.IsDir() {
				if !collaboration.IsDeliverableStubFile(abs) {
					return true
				}
			}
		}
		if h.fileChangeManager != nil && msg != nil {
			base := filepath.Base(rel)
			for _, ch := range h.fileChangeManager.ListAllFileChanges() {
				if ch == nil || ch.Channel != msg.Channel {
					continue
				}
				if ch.Status != filechange.FileChangeStatusPending && ch.Status != filechange.FileChangeStatusApproved {
					continue
				}
				if strings.HasSuffix(filepath.ToSlash(ch.FilePath), base) || strings.Contains(filepath.ToSlash(ch.FilePath), rel) {
					return true
				}
			}
		}
	}
	return false
}

func (h *Hub) maybeWarnPrematureTaskCompletion(msg *protocol.Message, collabID string, task *collaboration.CollaborationTask, snap *collaboration.Collaboration) bool {
	if msg == nil || task == nil || snap == nil || h.collabManager == nil {
		return false
	}
	if !collaboration.NewDeliverablePolicy(*task, snap.Description, nil).RequiresFile() {
		return false
	}
	if h.collabTaskDeliverableSatisfied(snap, task, msg) {
		return false
	}
	ch := msg.Channel
	if ch == "" {
		ch = snap.Channel
	}
	warn := fmt.Sprintf(
		"⚠️ **Task not marked complete** (`%s`) — @%s reported done but no `[FILE_CHANGE]` or on-disk deliverable was found for a file task. Finish the file proposal or use `/collab-task-done` when satisfied.",
		snap.ID[:8],
		msg.From.Name,
	)
	h.broadcastCollabSystem(ch, collabID, warn)
	if task.Status != collaboration.TaskInProgress {
		_, _ = h.collabManager.UpdateTaskStatusWithEffects(collabID, task.ID, collaboration.TaskInProgress, "Awaiting file deliverable")
	}
	return true
}
