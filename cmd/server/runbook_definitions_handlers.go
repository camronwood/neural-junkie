package main

import (
	"encoding/json"
	"io"
	"net/http"
	"strconv"
	"strings"

	"github.com/camronwood/neural-junkie/internal/hub"
	"github.com/camronwood/neural-junkie/internal/runbooklibrary"
)

func handleRunbookDefinitionsRoute(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		if _, ok := ensureMutationAccess(w, r, ""); !ok {
			return
		}
	}
	path := strings.TrimPrefix(r.URL.Path, "/api/runbook-definitions")
	path = strings.Trim(path, "/")
	if path == "" {
		switch r.Method {
		case http.MethodGet:
			list, err := runbooklibrary.ListAllDefinitions(chatHub.GetCollaborationAssetsRoot(), serverPackRunbookDefinitions())
			if err != nil {
				http.Error(w, err.Error(), http.StatusInternalServerError)
				return
			}
			writeCollabJSON(w, list)
		case http.MethodPost:
			handleRunbookDefinitionCreate(w, r)
		default:
			http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		}
		return
	}
	if path == "import" && r.Method == http.MethodPost {
		handleRunbookDefinitionImport(w, r)
		return
	}
	parts := strings.Split(path, "/")
	id := parts[0]
	if len(parts) == 2 && parts[1] == "instantiate" && r.Method == http.MethodPost {
		handleRunbookDefinitionInstantiate(w, r, id)
		return
	}
	if len(parts) == 2 && parts[1] == "trigger" && r.Method == http.MethodPost {
		handleRunbookDefinitionTrigger(w, r, id)
		return
	}
	if len(parts) == 2 && parts[1] == "export" && r.Method == http.MethodGet {
		handleRunbookDefinitionExport(w, r, id)
		return
	}
	if len(parts) != 1 {
		http.Error(w, "Not found", http.StatusNotFound)
		return
	}
	switch r.Method {
	case http.MethodGet:
		handleRunbookDefinitionGet(w, r, id)
	case http.MethodPut:
		handleRunbookDefinitionUpdate(w, r, id)
	case http.MethodDelete:
		handleRunbookDefinitionDelete(w, r, id)
	default:
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
	}
}

func handleRunbookDefinitionGet(w http.ResponseWriter, r *http.Request, id string) {
	version := 0
	if v := r.URL.Query().Get("version"); v != "" {
		if n, err := strconv.Atoi(v); err == nil {
			version = n
		}
	}
	def, err := runbooklibrary.LoadDefinition(id, version, chatHub.GetCollaborationAssetsRoot(), serverPackRunbookDefinitions())
	if err != nil {
		http.Error(w, err.Error(), http.StatusNotFound)
		return
	}
	writeCollabJSON(w, def)
}

func handleRunbookDefinitionCreate(w http.ResponseWriter, r *http.Request) {
	var def runbooklibrary.RunbookDefinition
	if err := json.NewDecoder(r.Body).Decode(&def); err != nil {
		http.Error(w, "Invalid JSON", http.StatusBadRequest)
		return
	}
	if warns := runbooklibrary.ValidateDefinition(&def, runbooklibrary.MergeInputDefaults(&def, nil)); len(warns) > 0 {
		for _, msg := range warns {
			if strings.Contains(msg, "cycle") || strings.Contains(msg, "unknown dependency") {
				http.Error(w, msg, http.StatusBadRequest)
				return
			}
		}
	}
	saved, err := runbooklibrary.SaveUserDefinition(def)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	writeCollabJSON(w, saved)
}

func handleRunbookDefinitionUpdate(w http.ResponseWriter, r *http.Request, id string) {
	var def runbooklibrary.RunbookDefinition
	if err := json.NewDecoder(r.Body).Decode(&def); err != nil {
		http.Error(w, "Invalid JSON", http.StatusBadRequest)
		return
	}
	def.ID = id
	saved, err := runbooklibrary.SaveUserDefinition(def)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	writeCollabJSON(w, saved)
}

func handleRunbookDefinitionDelete(w http.ResponseWriter, r *http.Request, id string) {
	if err := runbooklibrary.DeleteUserDefinition(id); err != nil {
		http.Error(w, err.Error(), http.StatusNotFound)
		return
	}
	writeCollabJSON(w, map[string]string{"status": "deleted"})
}

func handleRunbookDefinitionInstantiate(w http.ResponseWriter, r *http.Request, id string) {
	var body struct {
		Channel   string            `json:"channel"`
		CreatedBy string            `json:"created_by"`
		AgentIDs  []string          `json:"agent_ids"`
		Inputs    map[string]string `json:"inputs"`
		Version   int               `json:"version"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		http.Error(w, "Invalid JSON", http.StatusBadRequest)
		return
	}
	if len(body.AgentIDs) < 1 {
		http.Error(w, "agent_ids required", http.StatusBadRequest)
		return
	}
	result, err := chatHub.InstantiateDefinition(id, body.Version, hub.RunbookCreateRequest{
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

// handleRunbookDefinitionExport returns a portable DefinitionBundle for one
// definition (optionally a specific ?version=), suitable for downloading and
// later importing into another Neural Junkie installation.
//
// GET /api/runbook-definitions/{id}/export[?version=N]
func handleRunbookDefinitionExport(w http.ResponseWriter, r *http.Request, id string) {
	version := 0
	if v := r.URL.Query().Get("version"); v != "" {
		if n, err := strconv.Atoi(v); err == nil {
			version = n
		}
	}
	def, err := runbooklibrary.LoadDefinition(id, version, chatHub.GetCollaborationAssetsRoot(), serverPackRunbookDefinitions())
	if err != nil {
		http.Error(w, err.Error(), http.StatusNotFound)
		return
	}
	bundle := runbooklibrary.NewDefinitionBundle(*def)
	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("Content-Disposition", "attachment; filename=\"runbook-"+id+".json\"")
	_ = json.NewEncoder(w).Encode(bundle)
}

// handleRunbookDefinitionImport accepts either a DefinitionBundle (from
// handleRunbookDefinitionExport) or a bare RunbookDefinition JSON body and
// saves it as a new user definition. Pass ?keep_id=true to preserve the
// original definition ID instead of minting a fresh one.
//
// POST /api/runbook-definitions/import[?keep_id=true]
func handleRunbookDefinitionImport(w http.ResponseWriter, r *http.Request) {
	if _, ok := ensureMutationAccess(w, r, ""); !ok {
		return
	}
	raw, err := io.ReadAll(r.Body)
	if err != nil {
		http.Error(w, "Failed to read body", http.StatusBadRequest)
		return
	}
	var bundle runbooklibrary.DefinitionBundle
	def := bundle.Definition
	if err := json.Unmarshal(raw, &bundle); err == nil && (bundle.Definition.ID != "" || len(bundle.Definition.Tasks) > 0) {
		def = bundle.Definition
	} else if err := json.Unmarshal(raw, &def); err != nil {
		http.Error(w, "Invalid JSON: expected a runbook definition or export bundle", http.StatusBadRequest)
		return
	}
	if len(def.Tasks) == 0 {
		http.Error(w, "definition has no tasks", http.StatusBadRequest)
		return
	}
	if warns := runbooklibrary.ValidateDefinition(&def, runbooklibrary.MergeInputDefaults(&def, nil)); len(warns) > 0 {
		for _, msg := range warns {
			if strings.Contains(msg, "cycle") || strings.Contains(msg, "unknown dependency") {
				http.Error(w, msg, http.StatusBadRequest)
				return
			}
		}
	}
	keepID := r.URL.Query().Get("keep_id") == "true"
	saved, err := runbooklibrary.ImportDefinitionBundle(def, keepID)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	writeCollabJSON(w, saved)
}

func handleRunbookDefinitionTrigger(w http.ResponseWriter, r *http.Request, id string) {
	if _, ok := ensureMutationAccess(w, r, ""); !ok {
		return
	}
	var body struct {
		Channel   string            `json:"channel"`
		CreatedBy string            `json:"created_by"`
		AgentIDs  []string          `json:"agent_ids"`
		Inputs    map[string]string `json:"inputs"`
		Version   int               `json:"version"`
		Token     string            `json:"token"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		http.Error(w, "Invalid JSON", http.StatusBadRequest)
		return
	}
	result, err := chatHub.TriggerRunbookDefinition(id, body.Version, hub.RunbookCreateRequest{
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
