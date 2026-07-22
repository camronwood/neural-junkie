package main

import (
	"encoding/json"
	"net/http"
	"strings"

	"github.com/camronwood/neural-junkie/internal/collaboration"
)

func handleCollaborations(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	channel := strings.TrimSpace(r.URL.Query().Get("channel"))
	includeTerminal := strings.EqualFold(strings.TrimSpace(r.URL.Query().Get("include_terminal")), "true")
	collaborations := chatHub.ListCollaborationSnapshots(channel, includeTerminal)

	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(collaborations)
}

func handleCollaborationWorkspaceAck(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}
	if _, ok := ensureMutationAccess(w, r, ""); !ok {
		return
	}
	var req struct {
		CollaborationID string `json:"collaboration_id"`
		SourceRepoPath  string `json:"source_repo_path"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid JSON", http.StatusBadRequest)
		return
	}
	id := strings.TrimSpace(req.CollaborationID)
	if id == "" {
		http.Error(w, "collaboration_id required", http.StatusBadRequest)
		return
	}
	if err := chatHub.AcknowledgeCollaborationWorkspace(id, strings.TrimSpace(req.SourceRepoPath)); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

// handleCollaborationsSubRoute serves /api/collaborations/:id/...
func handleCollaborationsSubRoute(w http.ResponseWriter, r *http.Request) {
	path := strings.TrimPrefix(r.URL.Path, "/api/collaborations/")
	path = strings.Trim(path, "/")
	if path == "" {
		http.Error(w, "Not found", http.StatusNotFound)
		return
	}
	parts := strings.Split(path, "/")
	id := parts[0]
	if r.Method != http.MethodGet {
		if _, ok := ensureMutationAccess(w, r, ""); !ok {
			return
		}
	}
	if len(parts) == 1 {
		if r.Method == http.MethodGet {
			snap, err := chatHub.GetCollaborationManager().GetCollaborationSnapshot(id)
			if err != nil {
				http.Error(w, err.Error(), http.StatusNotFound)
				return
			}
			writeCollabJSON(w, snap)
			return
		}
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}
	if len(parts) == 2 && parts[1] == "pause" && r.Method == http.MethodPost {
		handleCollabPause(w, id, true)
		return
	}
	if len(parts) == 2 && parts[1] == "resume" && r.Method == http.MethodPost {
		handleCollabPause(w, id, false)
		return
	}
	if len(parts) >= 4 && parts[1] == "participant-requests" {
		agentID := parts[2]
		switch parts[3] {
		case "approve":
			if r.Method == http.MethodPost {
				handleCollabParticipantRequest(w, id, agentID, true)
				return
			}
		case "deny":
			if r.Method == http.MethodPost {
				handleCollabParticipantRequest(w, id, agentID, false)
				return
			}
		}
	}
	if len(parts) >= 4 && parts[1] == "tasks" {
		taskID := parts[2]
		switch parts[3] {
		case "complete":
			if r.Method == http.MethodPost {
				handleCollabTaskAction(w, id, taskID, "complete")
				return
			}
		case "reassign":
			if r.Method == http.MethodPost {
				handleCollabTaskReassign(w, r, id, taskID)
				return
			}
		case "redispatch":
			if r.Method == http.MethodPost {
				handleCollabTaskAction(w, id, taskID, "redispatch")
				return
			}
		case "skip":
			if r.Method == http.MethodPost {
				handleCollabTaskAction(w, id, taskID, "skip")
				return
			}
		case "approve":
			if r.Method == http.MethodPost {
				handleCollabTaskApprove(w, id, taskID)
				return
			}
		}
	}
	http.Error(w, "Not found", http.StatusNotFound)
}

func handleCollabPause(w http.ResponseWriter, collabID string, paused bool) {
	cm := chatHub.GetCollaborationManager()
	if cm == nil {
		http.Error(w, "collaboration manager unavailable", http.StatusServiceUnavailable)
		return
	}
	snap, err := cm.SetDispatchPaused(collabID, paused)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	writeCollabJSON(w, snap)
}

func handleCollabTaskAction(w http.ResponseWriter, collabID, taskID, action string) {
	cm := chatHub.GetCollaborationManager()
	if cm == nil {
		http.Error(w, "collaboration manager unavailable", http.StatusServiceUnavailable)
		return
	}
	var effects collaboration.UpdateTaskStatusResult
	var err error
	switch action {
	case "complete":
		effects, err = cm.UpdateTaskStatusWithEffects(collabID, taskID, collaboration.TaskCompleted, "Marked complete via API")
	case "skip":
		effects, err = cm.SkipTask(collabID, taskID, "skipped via API")
	case "redispatch":
		_, err = cm.ResetTaskDispatch(collabID, taskID)
		if err == nil {
			if snap, e := cm.GetCollaborationSnapshot(collabID); e == nil && snap != nil {
				chatHub.DispatchReadyCollabTasksForSnapshot(snap, true)
			}
		}
	default:
		http.Error(w, "unknown action", http.StatusBadRequest)
		return
	}
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	if effects.ShouldDispatchWave {
		if snap, e := cm.GetCollaborationSnapshot(collabID); e == nil && snap != nil {
			chatHub.DispatchReadyCollabTasksForSnapshot(snap, false)
		}
	}
	snap, _ := cm.GetCollaborationSnapshot(collabID)
	writeCollabJSON(w, snap)
}

func handleCollabTaskReassign(w http.ResponseWriter, r *http.Request, collabID, taskID string) {
	var body struct {
		AgentID string `json:"agent_id"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil || body.AgentID == "" {
		http.Error(w, "agent_id required", http.StatusBadRequest)
		return
	}
	cm := chatHub.GetCollaborationManager()
	snap, err := cm.ReassignTask(collabID, taskID, body.AgentID)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	writeCollabJSON(w, snap)
}

func handleCollabTaskApprove(w http.ResponseWriter, collabID, taskID string) {
	cm := chatHub.GetCollaborationManager()
	snap, err := cm.ApproveTaskDispatch(collabID, taskID)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	chatHub.ResolveCollabTaskApproval(collabID, taskID, "user", "approved", "")
	chatHub.DispatchReadyCollabTasksForSnapshot(snap, false)
	snap, _ = cm.GetCollaborationSnapshot(collabID)
	writeCollabJSON(w, snap)
}

func handleCollabParticipantRequest(w http.ResponseWriter, collabID, agentID string, approve bool) {
	var (
		snap *collaboration.Collaboration
		err  error
	)
	if approve {
		snap, err = chatHub.ApproveCollaborationParticipantRequest(collabID, agentID)
	} else {
		snap, err = chatHub.DenyCollaborationParticipantRequest(collabID, agentID)
	}
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	writeCollabJSON(w, snap)
}

func writeCollabJSON(w http.ResponseWriter, v interface{}) {
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(v)
}
