package hub

import (
	"fmt"
	"strings"

	"github.com/camronwood/neural-junkie/internal/collaboration"
	"github.com/camronwood/neural-junkie/internal/protocol"
)

func (h *Hub) collaborationWorkspaceContextSnapshot(snap *collaboration.Collaboration) map[string]interface{} {
	if snap == nil || strings.TrimSpace(snap.WorkingDirectory) == "" {
		return nil
	}
	workspacePath := strings.TrimSpace(snap.WorkingDirectory)
	name := strings.TrimSpace(snap.Title)
	if name == "" {
		name = "Collaboration"
	}
	wsName := fmt.Sprintf("Collaboration: %s", name)
	if snap.ExecutionMode == collaboration.ExecutionModeWorktree {
		wsName = fmt.Sprintf("Collab worktree: %s", name)
	}
	outputPath := collaboration.PlannedOutputDirectory(snap, "")
	fileTree := ".  (sandbox — empty until agents create files)\n"
	if snap.ExecutionMode == collaboration.ExecutionModeWorktree {
		fileTree = buildOutlineFileTree(workspacePath, 3)
		if fileTree == "" {
			fileTree = ".\n"
		}
	} else if sourcePath := strings.TrimSpace(snap.SourceRepoPath); sourcePath != "" {
		workspacePath = sourcePath
		wsName = fmt.Sprintf("Source workspace: %s", name)
		if stored := snap.SourceWorkspaceContext; len(stored) > 0 {
			if tree, ok := stored["file_tree"].(string); ok && strings.TrimSpace(tree) != "" {
				fileTree = tree
			}
		}
		if fileTree == ".  (sandbox — empty until agents create files)\n" {
			fileTree = buildCollabOutlineFileTree(sourcePath, snap.Description, 3)
			if fileTree == "" {
				fileTree = ".\n"
			}
		}
	} else if collaboration.UsesProjectCollabDir(snap) {
		fileTree = ".\n"
	} else {
		fileTree = ".  (sandbox — empty until agents create files)\n"
	}
	out := map[string]interface{}{
		"workspace_name": wsName,
		"workspace_path": workspacePath,
		"file_tree":      fileTree,
		"open_files":     []interface{}{},
	}
	if outputPath != "" && outputPath != workspacePath {
		out["collaboration_output_path"] = outputPath
	} else if strings.TrimSpace(snap.WorkingDirectory) != "" && snap.WorkingDirectory != workspacePath {
		out["collaboration_output_path"] = snap.WorkingDirectory
	}
	if h != nil && h.collabManager != nil {
		if baseDir, err := h.collabManager.CollabAssetsBaseDir(); err == nil && strings.TrimSpace(baseDir) != "" {
			paths := collaboration.CollabAssetPaths(snap, baseDir)
			out["review_assets_path"] = paths.Directory
			out["review_assets_files"] = []string{
				collaboration.ReviewAssetsPlanFileName,
				collaboration.ReviewAssetsPlanningSummaryName,
				collaboration.ReviewAssetsSessionSummaryName,
				collaboration.ReviewAssetsIndexFileName,
			}
		}
	}
	return out
}

// CollaborationCanDispatchTasks is true when collaboration_task messages may be sent.
func (h *Hub) CollaborationCanDispatchTasks(snap *collaboration.Collaboration) bool {
	if snap == nil || snap.Phase != collaboration.PhaseExecuting {
		return false
	}
	if strings.TrimSpace(snap.WorkingDirectory) == "" {
		if snap.ExecutionMode == collaboration.ExecutionModeWorktree {
			return false
		}
		return true
	}
	return snap.WorkspaceAcknowledged
}

// AcknowledgeCollaborationWorkspace marks the execution workspace as user-confirmed,
// creates a deferred git worktree when needed, dispatches task prompts once,
// and broadcasts a collaboration_status update.
func (h *Hub) AcknowledgeCollaborationWorkspace(collabID, sourceRepoPath string) error {
	if h.collabManager == nil {
		return fmt.Errorf("collaboration manager unavailable")
	}
	snap, err := h.collabManager.GetCollaborationSnapshot(collabID)
	if err != nil {
		return err
	}
	if snap == nil {
		return fmt.Errorf("collaboration not found")
	}
	if snap.ExecutionMode == collaboration.ExecutionModeWorktree && strings.TrimSpace(snap.WorkingDirectory) == "" {
		if _, err := h.collabManager.EnsureWorktree(collabID, sourceRepoPath); err != nil {
			return err
		}
	}
	already, _, err := h.collabManager.AcknowledgeWorkspace(collabID)
	if err != nil {
		return err
	}
	snap, err = h.collabManager.GetCollaborationSnapshot(collabID)
	if err != nil {
		return fmt.Errorf("collaboration snapshot: %w", err)
	}
	if snap == nil {
		return fmt.Errorf("collaboration snapshot: not found")
	}
	if !already && len(snap.Tasks) > 0 {
		h.dispatchReadyCollabTasks(snap, nil, false)
	}
	statusMsg := protocol.NewMessage(
		protocol.MessageTypeCollabStatus,
		snap.Channel,
		protocol.AgentInfo{ID: "system", Name: "System", Type: protocol.AgentTypeGeneral},
		fmt.Sprintf("✅ **Collaboration workspace ready** (`%s`).", collabID[:8]),
	)
	statusMsg.SetCollaborationID(collabID)
	statusMsg.SetCollaborationPhase(string(snap.Phase))
	if statusMsg.Metadata == nil {
		statusMsg.Metadata = map[string]interface{}{}
	}
	statusMsg.Metadata["collab_skip_attach_dispatch"] = true
	return h.SendMessage(statusMsg)
}
