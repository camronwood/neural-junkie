package main

import (
	"net/http"
	"strings"

	"github.com/camronwood/neural-junkie/internal/hub"
	"github.com/camronwood/neural-junkie/internal/runbookruns"
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
