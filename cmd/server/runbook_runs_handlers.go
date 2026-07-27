package main

import (
	"net/http"
	"strings"

	"github.com/camronwood/neural-junkie/internal/collaboration"
	"github.com/camronwood/neural-junkie/internal/hub"
	"github.com/camronwood/neural-junkie/internal/runbooklibrary"
	"github.com/camronwood/neural-junkie/internal/runbookruns"
	"github.com/camronwood/neural-junkie/internal/workflow"
)

func handleRunbookRunsRoute(w http.ResponseWriter, r *http.Request) {
	path := strings.TrimPrefix(r.URL.Path, "/api/runbook-runs")
	path = strings.Trim(path, "/")
	if path == "" {
		if r.Method != http.MethodGet {
			http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
			return
		}
		defID := r.URL.Query().Get("definition_id")
		runs, err := runbookruns.ListRuns(defID)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		writeCollabJSON(w, runs)
		return
	}
	parts := strings.Split(path, "/")
	collabID := parts[0]
	if len(parts) == 2 && parts[1] == "replay" && r.Method == http.MethodPost {
		if _, ok := ensureMutationAccess(w, r, ""); !ok {
			return
		}
		handleRunbookRunReplay(w, r, collabID)
		return
	}
	if len(parts) == 2 && parts[1] == "provenance" && r.Method == http.MethodGet {
		handleRunbookRunProvenance(w, r, collabID)
		return
	}
	if len(parts) == 2 && parts[1] == "progress" && r.Method == http.MethodGet {
		handleRunbookRunProgress(w, r, collabID)
		return
	}
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}
	snap, err := chatHub.GetRunbookSnapshot(collabID)
	if err != nil {
		http.Error(w, err.Error(), http.StatusNotFound)
		return
	}
	rec, _ := runbookruns.GetRun(collabID)
	writeCollabJSON(w, map[string]interface{}{
		"run":           rec,
		"collaboration": snap,
	})
}

// runbookTaskProgress is a lightweight status API for one runbook execution
// so the desktop app (or CLI/automation) can poll queue/progress visibility
// without re-fetching and re-deriving the full collaboration snapshot.
//
// GET /api/runbook-runs/{collabID}/progress
func handleRunbookRunProgress(w http.ResponseWriter, r *http.Request, collabID string) {
	snap, err := chatHub.GetRunbookSnapshot(collabID)
	if err != nil {
		http.Error(w, err.Error(), http.StatusNotFound)
		return
	}

	counts := map[string]int{
		string(collaboration.TaskPending):    0,
		string(collaboration.TaskInProgress): 0,
		string(collaboration.TaskCompleted):  0,
		string(collaboration.TaskBlocked):    0,
	}
	var queuedIDs, dispatchedIDs, blockedIDs, completedIDs []string
	for _, t := range snap.Tasks {
		counts[string(t.Status)]++
		switch t.Status {
		case collaboration.TaskInProgress:
			dispatchedIDs = append(dispatchedIDs, t.ID)
		case collaboration.TaskBlocked:
			blockedIDs = append(blockedIDs, t.ID)
		case collaboration.TaskCompleted:
			completedIDs = append(completedIDs, t.ID)
		}
	}
	for _, t := range collaboration.ReadyTasksForCollab(snap) {
		queuedIDs = append(queuedIDs, t.ID)
	}

	total := len(snap.Tasks)
	percent := 0.0
	if total > 0 {
		percent = 100 * float64(counts[string(collaboration.TaskCompleted)]) / float64(total)
	}

	writeCollabJSON(w, map[string]interface{}{
		"collaboration_id":       snap.ID,
		"phase":                  snap.Phase,
		"workspace_acknowledged": snap.WorkspaceAcknowledged,
		"tasks_dispatched":       snap.TasksDispatched,
		"dispatch_paused":        snap.DispatchPaused,
		"awaiting_finalize":      snap.AwaitingFinalize,
		"total_tasks":            total,
		"counts":                 counts,
		"percent_complete":       percent,
		"queued_task_ids":        queuedIDs,
		"in_progress_task_ids":   dispatchedIDs,
		"blocked_task_ids":       blockedIDs,
		"completed_task_ids":     completedIDs,
	})
}

// handleRunbookRunProvenance answers "where did this run come from and what
// happened during it": the run index record, the definition (and version)
// it was instantiated from, the live collaboration snapshot, and the
// append-only trace of phase/task events recorded for the run.
//
// GET /api/runbook-runs/{collabID}/provenance
func handleRunbookRunProvenance(w http.ResponseWriter, r *http.Request, collabID string) {
	rec, _ := runbookruns.GetRun(collabID)
	snap, snapErr := chatHub.GetRunbookSnapshot(collabID)

	definitionID := ""
	definitionVersion := 0
	if rec != nil {
		definitionID = rec.DefinitionID
		definitionVersion = rec.DefinitionVersion
	}
	if definitionID == "" && snapErr == nil {
		definitionID = snap.DefinitionID
		definitionVersion = snap.DefinitionVersion
	}

	var definition *runbooklibrary.RunbookDefinition
	if definitionID != "" {
		if def, err := runbooklibrary.LoadDefinition(definitionID, definitionVersion, chatHub.GetCollaborationAssetsRoot(), serverPackRunbookDefinitions()); err == nil {
			definition = def
		}
	}

	events, err := workflow.ReadEvents(collabID)
	if err != nil {
		events = []workflow.Event{}
	}

	if rec == nil && snapErr != nil && definition == nil && len(events) == 0 {
		http.Error(w, "no provenance found for run "+collabID, http.StatusNotFound)
		return
	}

	var collabOut interface{}
	if snapErr == nil {
		collabOut = snap
	}
	writeCollabJSON(w, map[string]interface{}{
		"run_id":        collabID,
		"run":           rec,
		"definition":    definition,
		"collaboration": collabOut,
		"events":        events,
	})
}

func handleRunbookRunReplay(w http.ResponseWriter, r *http.Request, collabID string) {
	snap, err := chatHub.GetRunbookSnapshot(collabID)
	if err != nil {
		http.Error(w, err.Error(), http.StatusNotFound)
		return
	}
	if snap.DefinitionID == "" {
		http.Error(w, "collaboration has no definition_id to replay", http.StatusBadRequest)
		return
	}
	agentIDs := make([]string, 0, len(snap.Agents))
	for _, a := range snap.Agents {
		agentIDs = append(agentIDs, a.AgentID)
	}
	result, err := chatHub.InstantiateDefinition(snap.DefinitionID, snap.DefinitionVersion, hub.RunbookCreateRequest{
		AgentIDs:  agentIDs,
		Channel:   snap.Channel,
		CreatedBy: snap.CreatedBy,
		RunInputs: snap.RunInputs,
	})
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	newSnap, _ := chatHub.GetRunbookSnapshot(result.CollaborationID)
	writeCollabJSON(w, map[string]interface{}{
		"collaboration_id":      result.CollaborationID,
		"collaboration_channel": result.CollaborationChannel,
		"collaboration":         newSnap,
	})
}
