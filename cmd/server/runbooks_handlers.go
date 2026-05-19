package main

import (
	"encoding/json"
	"net/http"
	"strings"

	"github.com/camronwood/neural-junkie/internal/collaboration"
	"github.com/camronwood/neural-junkie/internal/hub"
)

func handleRunbooksRoute(w http.ResponseWriter, r *http.Request) {
	path := strings.TrimPrefix(r.URL.Path, "/api/runbooks")
	path = strings.Trim(path, "/")
	if path == "" {
		if r.Method == http.MethodPost {
			handleRunbooksCreate(w, r)
			return
		}
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}
	parts := strings.Split(path, "/")
	id := parts[0]
	if len(parts) == 1 {
		if r.Method == http.MethodGet {
			handleRunbookGet(w, r, id)
			return
		}
		if r.Method == http.MethodPut {
			handleRunbookUpdate(w, r, id)
			return
		}
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}
	switch parts[1] {
	case "suggest-assignee":
		if r.Method == http.MethodPost {
			handleRunbookSuggestAssignee(w, r, id)
			return
		}
	case "parse-plan":
		if r.Method == http.MethodPost {
			handleRunbookParsePlan(w, r, id)
			return
		}
	case "submit":
		if r.Method == http.MethodPost {
			handleRunbookSubmit(w, r, id)
			return
		}
	case "start":
		if r.Method == http.MethodPost {
			handleRunbookStart(w, r, id)
			return
		}
	}
	http.Error(w, "Not found", http.StatusNotFound)
}

func handleRunbooksCreate(w http.ResponseWriter, r *http.Request) {
	var req hub.RunbookCreateRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid JSON", http.StatusBadRequest)
		return
	}
	result, err := chatHub.CreateRunbookSession(req)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	snap, _ := chatHub.GetRunbookSnapshot(result.CollaborationID)
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(map[string]interface{}{
		"collaboration_id":      result.CollaborationID,
		"collaboration_channel": result.CollaborationChannel,
		"collaboration":         snap,
	})
}

func handleRunbookGet(w http.ResponseWriter, r *http.Request, id string) {
	snap, err := chatHub.GetRunbookSnapshot(id)
	if err != nil {
		http.Error(w, err.Error(), http.StatusNotFound)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(snap)
}

func handleRunbookUpdate(w http.ResponseWriter, r *http.Request, id string) {
	var body struct {
		Title       string                           `json:"title"`
		Description string                           `json:"description"`
		AgentIDs    []string                         `json:"agent_ids"`
		Tasks       []collaboration.CollaborationTask `json:"tasks"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		http.Error(w, "Invalid JSON", http.StatusBadRequest)
		return
	}
	snap, err := chatHub.UpdateRunbookSession(id, collaboration.RunbookUpdatePayload{
		Title:       body.Title,
		Description: body.Description,
		AgentIDs:    body.AgentIDs,
		Tasks:       body.Tasks,
	})
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(snap)
}

func handleRunbookSuggestAssignee(w http.ResponseWriter, r *http.Request, id string) {
	var body struct {
		Title       string `json:"title"`
		Description string `json:"description"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		http.Error(w, "Invalid JSON", http.StatusBadRequest)
		return
	}
	cm := chatHub.GetCollaborationManager()
	if cm == nil {
		http.Error(w, "collaboration unavailable", http.StatusServiceUnavailable)
		return
	}
	s, err := cm.SuggestRunbookAssignee(id, body.Title, body.Description)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	if s == nil {
		_ = json.NewEncoder(w).Encode(map[string]string{"reason": "low_confidence"})
		return
	}
	_ = json.NewEncoder(w).Encode(s)
}

func handleRunbookParsePlan(w http.ResponseWriter, r *http.Request, id string) {
	var body struct {
		Markdown string `json:"markdown"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		http.Error(w, "Invalid JSON", http.StatusBadRequest)
		return
	}
	snap, err := chatHub.GetRunbookSnapshot(id)
	if err != nil {
		http.Error(w, err.Error(), http.StatusNotFound)
		return
	}
	cm := chatHub.GetCollaborationManager()
	if cm == nil {
		http.Error(w, "collaboration unavailable", http.StatusServiceUnavailable)
		return
	}
	tasks, err := collaboration.ParsePlanTasks(body.Markdown, snap.Agents)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(map[string]interface{}{"tasks": tasks})
}

func handleRunbookSubmit(w http.ResponseWriter, r *http.Request, id string) {
	snap, err := chatHub.SubmitRunbookForReview(id)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(snap)
}

func handleRunbookStart(w http.ResponseWriter, r *http.Request, id string) {
	snap, err := chatHub.StartRunbook(id)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	if chatHub.CollaborationCanDispatchTasks(snap) {
		chatHub.DispatchReadyCollabTasksForSnapshot(snap, false)
	}
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(snap)
}
