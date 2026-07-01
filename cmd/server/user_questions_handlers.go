package main

import (
	"encoding/json"
	"net/http"
	"strings"

	"github.com/camronwood/neural-junkie/internal/hub"
)

func handleUserQuestions(w http.ResponseWriter, r *http.Request) {
	if r.Method != "POST" {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var req struct {
		AgentID   string   `json:"agent_id"`
		AgentName string   `json:"agent_name"`
		Channel   string   `json:"channel"`
		Question  string   `json:"question"`
		Options   []string `json:"options"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}
	if strings.TrimSpace(req.Question) == "" {
		http.Error(w, "question is required", http.StatusBadRequest)
		return
	}
	if req.Channel == "" {
		req.Channel = "general"
	}

	uqm := chatHub.GetUserQuestionManager()
	if uqm == nil {
		http.Error(w, "user question manager unavailable", http.StatusInternalServerError)
		return
	}
	answer, err := uqm.Ask(req.AgentID, req.AgentName, req.Channel, req.Question, req.Options, hub.UserQuestionTTL)
	if err != nil {
		http.Error(w, err.Error(), http.StatusRequestTimeout)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{"answer": answer})
}

func handleAnswerUserQuestion(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	questionID := strings.TrimPrefix(r.URL.Path, "/api/user-questions/answer/")
	if questionID == "" {
		http.Error(w, "question id required", http.StatusBadRequest)
		return
	}

	var req struct {
		Answer string `json:"answer"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	uqm := chatHub.GetUserQuestionManager()
	if uqm == nil {
		http.Error(w, "user question manager unavailable", http.StatusInternalServerError)
		return
	}
	if err := uqm.Answer(questionID, req.Answer); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{"status": "ok"})
}

func handlePendingUserQuestions(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	uqm := chatHub.GetUserQuestionManager()
	if uqm == nil {
		http.Error(w, "user question manager unavailable", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(uqm.ListPending())
}
