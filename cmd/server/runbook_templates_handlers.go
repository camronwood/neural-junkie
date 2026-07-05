package main

import (
	"encoding/json"
	"net/http"
	"os"
	"path/filepath"
	"strings"

	"github.com/camronwood/neural-junkie/internal/hub"
	"github.com/camronwood/neural-junkie/internal/runbooklibrary"
)

func runbookTemplatesDir() string {
	candidates := []string{
		filepath.Join("assets", "runbook-templates"),
		filepath.Join("neural-junkie", "assets", "runbook-templates"),
	}
	if root := chatHub.GetCollaborationAssetsRoot(); root != "" {
		candidates = append([]string{filepath.Join(root, "runbook-templates")}, candidates...)
	}
	for _, d := range candidates {
		if st, err := os.Stat(d); err == nil && st.IsDir() {
			return d
		}
	}
	return candidates[0]
}

func handleRunbookTemplatesRoute(w http.ResponseWriter, r *http.Request) {
	path := strings.TrimPrefix(r.URL.Path, "/api/runbook-templates")
	path = strings.Trim(path, "/")
	if path == "" {
		if r.Method == http.MethodGet {
			list, err := runbooklibrary.ListAllDefinitions(chatHub.GetCollaborationAssetsRoot(), serverPackRunbookDefinitions())
			if err != nil {
				http.Error(w, err.Error(), http.StatusInternalServerError)
				return
			}
			writeCollabJSON(w, list)
			return
		}
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}
	parts := strings.Split(path, "/")
	name := parts[0]
	if len(parts) == 2 && parts[1] == "instantiate" && r.Method == http.MethodPost {
		if _, ok := ensureMutationAccess(w, r, ""); !ok {
			return
		}
		handleRunbookTemplateInstantiate(w, r, name)
		return
	}
	http.Error(w, "Not found", http.StatusNotFound)
}

func handleRunbookTemplateInstantiate(w http.ResponseWriter, r *http.Request, name string) {
	var body struct {
		Channel   string            `json:"channel"`
		CreatedBy string            `json:"created_by"`
		AgentIDs  []string          `json:"agent_ids"`
		Inputs    map[string]string `json:"inputs"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		http.Error(w, "Invalid JSON", http.StatusBadRequest)
		return
	}
	if len(body.AgentIDs) < 1 {
		http.Error(w, "agent_ids required", http.StatusBadRequest)
		return
	}
	result, err := chatHub.InstantiateDefinition(name, 0, hub.RunbookCreateRequest{
		AgentIDs:  body.AgentIDs,
		Channel:   body.Channel,
		CreatedBy: body.CreatedBy,
		RunInputs: body.Inputs,
	})
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	snap, _ := chatHub.GetRunbookSnapshot(result.CollaborationID)
	writeCollabJSON(w, map[string]interface{}{
		"collaboration_id":      result.CollaborationID,
		"collaboration_channel": result.CollaborationChannel,
		"collaboration":         snap,
	})
}
