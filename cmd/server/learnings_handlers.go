package main

import (
	"encoding/json"
	"net/http"
	"strings"

	learningpkg "github.com/camronwood/neural-junkie/internal/learning"
	"github.com/camronwood/neural-junkie/internal/lora/export"
	"github.com/camronwood/neural-junkie/internal/protocol"
)

var learningStore *learningpkg.Store

func handleLearningsRoute(w http.ResponseWriter, r *http.Request) {
	path := strings.TrimPrefix(r.URL.Path, "/api/learnings")
	path = strings.Trim(path, "/")
	if path == "stats" {
		handleLearningsStats(w, r)
		return
	}
	if path != "" {
		handleLearningByID(w, r, path)
		return
	}
	switch r.Method {
	case http.MethodGet:
		handleLearningsList(w, r)
	case http.MethodPost:
		handleLearningsCreate(w, r)
	default:
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
	}
}

func handleLearningsList(w http.ResponseWriter, r *http.Request) {
	if !requirePersonalLearning(w) {
		return
	}
	if learningStore == nil {
		writeJSON(w, http.StatusOK, []learningpkg.Entry{})
		return
	}
	agentID := strings.TrimSpace(r.URL.Query().Get("agent_id"))
	writeJSON(w, http.StatusOK, learningStore.List(agentID))
}

func handleLearningsCreate(w http.ResponseWriter, r *http.Request) {
	if !requirePersonalLearning(w) {
		return
	}
	if learningStore == nil {
		http.Error(w, "learning store unavailable", http.StatusServiceUnavailable)
		return
	}
	var body struct {
		AgentID         string `json:"agent_id"`
		AgentType       string `json:"agent_type"`
		AgentName       string `json:"agent_name"`
		Content         string `json:"content"`
		Category        string `json:"category"`
		SourceChannel   string `json:"source_channel"`
		SourceMessageID string `json:"source_message_id"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		http.Error(w, "Invalid JSON: "+err.Error(), http.StatusBadRequest)
		return
	}
	if body.AgentID == "" {
		http.Error(w, "agent_id required", http.StatusBadRequest)
		return
	}
	entry, err := learningStore.Add(learningpkg.Entry{
		AgentID:         body.AgentID,
		AgentType:       body.AgentType,
		AgentName:       body.AgentName,
		Content:         body.Content,
		Category:        learningpkg.Category(body.Category),
		SourceChannel:   body.SourceChannel,
		SourceMessageID: body.SourceMessageID,
	})
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	writeJSON(w, http.StatusOK, entry)
}

func handleLearningByID(w http.ResponseWriter, r *http.Request, id string) {
	if !requirePersonalLearning(w) {
		return
	}
	if r.Method != http.MethodDelete {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}
	if learningStore == nil {
		http.Error(w, "learning store unavailable", http.StatusServiceUnavailable)
		return
	}
	if err := learningStore.Forget(id); err != nil {
		http.Error(w, err.Error(), http.StatusNotFound)
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"ok": true})
}

func handleLearningsStats(w http.ResponseWriter, r *http.Request) {
	if !requirePersonalLearning(w) {
		return
	}
	agentID := strings.TrimSpace(r.URL.Query().Get("agent_id"))
	if agentID == "" {
		http.Error(w, "agent_id required", http.StatusBadRequest)
		return
	}
	count := 0
	if learningStore != nil {
		count = learningStore.CountForAgent(agentID)
	}
	ready := false
	previewRows := 0
	if info, err := chatHub.GetAgent(agentID); err == nil {
		ctx := buildExpertTrainContext(info)
		if v, ok := ctx["ready"].(bool); ok {
			ready = v
		}
		if v, ok := ctx["preview_rows"].(int); ok {
			previewRows = v
		}
	}
	writeJSON(w, http.StatusOK, map[string]any{
		"agent_id":        agentID,
		"learning_count":  count,
		"preview_rows":    previewRows,
		"min_rows":        export.MinRows,
		"ready_for_lora":  ready,
	})
}

func maybeEmitLearningProposal(msg *protocol.Message) {
	if !personalLearningActive() || msg == nil || chatHub == nil {
		return
	}
	if !protocol.IsUserLikeSender(msg.From) {
		return
	}
	if strings.HasPrefix(strings.TrimSpace(msg.Content), "/") {
		return
	}
	draft := learningpkg.ExtractDraftFromMessage(msg.Content)
	if draft == "" && !learningpkg.HasLearningTrigger(msg.Content) {
		return
	}
	if draft == "" {
		draft = strings.TrimSpace(msg.Content)
	}
	target := resolveLearningTargetAgent(msg)
	if target == nil {
		return
	}
	proposal := protocol.NewMessage(
		protocol.MessageTypeSystemInfo,
		msg.Channel,
		protocol.AgentInfo{ID: "system", Name: "System", Type: protocol.AgentTypeModerator},
		"Learning proposal — confirm in the dialog to save for "+target.Name+".",
	)
	if proposal.Metadata == nil {
		proposal.Metadata = map[string]any{}
	}
	proposal.Metadata["client_action"] = map[string]any{
		"type":              "learning_proposal",
		"agent_id":          target.ID,
		"agent_name":        target.Name,
		"agent_type":        string(target.Type),
		"draft":             draft,
		"category":          string(learningpkg.CategoryPreference),
		"source_message_id": msg.ID,
		"source_channel":    msg.Channel,
	}
	_ = chatHub.SendMessage(proposal)
}

func resolveLearningTargetAgent(msg *protocol.Message) *protocol.AgentInfo {
	if msg == nil || chatHub == nil {
		return nil
	}
	mentions := protocol.ParseMentions(msg.Content)
	for _, m := range mentions {
		name := strings.TrimPrefix(strings.TrimSpace(m), "@")
		for _, a := range chatHub.ListAgents() {
			if strings.EqualFold(a.Name, name) {
				cp := *a
				return &cp
			}
		}
	}
	agents, err := chatHub.GetChannelAgents(msg.Channel)
	if err != nil || len(agents) == 0 {
		return nil
	}
	for _, a := range agents {
		if a.Type != protocol.AgentTypeCLI && a.Type != protocol.AgentTypeModerator && !protocol.IsUserLikeSender(protocol.AgentInfo{Type: a.Type, Name: a.Name}) {
			cp := a
			return &cp
		}
	}
	return nil
}
