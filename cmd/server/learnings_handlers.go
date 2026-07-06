package main

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/camronwood/neural-junkie/internal/learning"
	"github.com/camronwood/neural-junkie/internal/protocol"
)

var learningStore *learning.Store

func handleLearningsRoute(w http.ResponseWriter, r *http.Request) {
	path := strings.TrimPrefix(r.URL.Path, "/api/learnings")
	path = strings.Trim(path, "/")
	switch path {
	case "stats":
		handleLearningsStats(w, r)
		return
	case "query":
		handleLearningsQuery(w, r)
		return
	case "export":
		handleLearningsExport(w, r)
		return
	case "import":
		handleLearningsImport(w, r)
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
		writeJSON(w, http.StatusOK, []learning.Entry{})
		return
	}
	userID := learningUserID(r)
	agentID := strings.TrimSpace(r.URL.Query().Get("agent_id"))
	agentType := strings.TrimSpace(r.URL.Query().Get("agent_type"))
	agentName := strings.TrimSpace(r.URL.Query().Get("agent_name"))
	if agentID != "" && chatHub != nil {
		if info, err := chatHub.GetAgent(agentID); err == nil {
			if agentType == "" {
				agentType = string(info.Type)
			}
			if agentName == "" {
				agentName = info.Name
			}
		}
	}
	if learningStore != nil {
		userID = learningStore.ResolveUserID(userID)
	}
	f := learning.Filter{
		AgentID:         agentID,
		AgentType:       agentType,
		AgentName:       agentName,
		UserID:          userID,
		Scope:           learning.Scope(strings.TrimSpace(r.URL.Query().Get("scope"))),
		CollaborationID: strings.TrimSpace(r.URL.Query().Get("collaboration_id")),
		IncludeLegacy:   true,
	}
	writeJSON(w, http.StatusOK, learningStore.ListFiltered(f))
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
		Scope           string `json:"scope"`
		UserID          string `json:"user_id"`
		AgentID         string `json:"agent_id"`
		AgentType       string `json:"agent_type"`
		AgentName       string `json:"agent_name"`
		CollaborationID string `json:"collaboration_id"`
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
	uid := learningUserID(r)
	if uid != "" {
		body.UserID = uid
	}
	entry, err := learningStore.Add(learning.Entry{
		Scope:           learning.Scope(body.Scope),
		UserID:          body.UserID,
		AgentID:         body.AgentID,
		AgentType:       body.AgentType,
		AgentName:       body.AgentName,
		CollaborationID: body.CollaborationID,
		Content:         body.Content,
		Category:        learning.Category(body.Category),
		SourceChannel:   body.SourceChannel,
		SourceMessageID: body.SourceMessageID,
	})
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	learning.ScheduleEmbed(entry)
	writeJSON(w, http.StatusOK, entry)
}

func handleLearningByID(w http.ResponseWriter, r *http.Request, id string) {
	if !requirePersonalLearning(w) {
		return
	}
	if learningStore == nil {
		http.Error(w, "learning store unavailable", http.StatusServiceUnavailable)
		return
	}
	switch r.Method {
	case http.MethodPut:
		var body struct {
			Content         string `json:"content"`
			Category        string `json:"category"`
			Scope           string `json:"scope"`
			CollaborationID string `json:"collaboration_id"`
		}
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			http.Error(w, "Invalid JSON: "+err.Error(), http.StatusBadRequest)
			return
		}
		patch := learning.UpdatePatch{}
		if body.Content != "" {
			patch.Content = &body.Content
		}
		if body.Category != "" {
			c := learning.Category(body.Category)
			patch.Category = &c
		}
		if body.Scope != "" {
			s := learning.Scope(body.Scope)
			patch.Scope = &s
		}
		if body.CollaborationID != "" {
			patch.CollaborationID = &body.CollaborationID
		}
		entry, err := learningStore.Update(id, patch)
		if err != nil {
			http.Error(w, err.Error(), http.StatusNotFound)
			return
		}
		learning.ScheduleEmbed(entry)
		writeJSON(w, http.StatusOK, entry)
	case http.MethodDelete:
		if err := learningStore.Forget(id); err != nil {
			http.Error(w, err.Error(), http.StatusNotFound)
			return
		}
		writeJSON(w, http.StatusOK, map[string]any{"ok": true})
	default:
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
	}
}

func handleLearningsQuery(w http.ResponseWriter, r *http.Request) {
	if !requirePersonalLearning(w) {
		return
	}
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}
	q := strings.TrimSpace(r.URL.Query().Get("q"))
	agentID := strings.TrimSpace(r.URL.Query().Get("agent_id"))
	scope := learning.Scope(strings.TrimSpace(r.URL.Query().Get("scope")))
	channel := strings.TrimSpace(r.URL.Query().Get("channel"))
	collabID := strings.TrimSpace(r.URL.Query().Get("collaboration_id"))
	if collabID == "" && channel != "" {
		collabID = learning.ResolveCollabID(channel)
	}
	ctx, cancel := context.WithTimeout(r.Context(), 500*time.Millisecond)
	defer cancel()
	pctx := learning.PromptContext{
		Query:           q,
		UserID:          learningUserID(r),
		Channel:         channel,
		CollaborationID: collabID,
	}
	if agentID != "" && chatHub != nil {
		if info, err := chatHub.GetAgent(agentID); err == nil {
			pctx.AgentType = string(info.Type)
			pctx.AgentName = info.Name
		}
	}
	if learningStore != nil {
		pctx.UserID = learningStore.ResolveUserID(pctx.UserID)
	}
	results := learning.QueryPreview(ctx, pctx, agentID, scope)
	writeJSON(w, http.StatusOK, map[string]any{
		"query":   q,
		"count":   len(results),
		"results": results,
	})
}

func handleLearningsExport(w http.ResponseWriter, r *http.Request) {
	if !requirePersonalLearning(w) {
		return
	}
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}
	if learningStore == nil {
		writeJSON(w, http.StatusOK, map[string]any{"entries": []learning.Entry{}})
		return
	}
	uid := learningUserID(r)
	entries := learningStore.ExportBundle(uid)
	writeJSON(w, http.StatusOK, map[string]any{
		"version": learning.StoreVersion,
		"user_id": uid,
		"entries": entries,
	})
}

func handleLearningsImport(w http.ResponseWriter, r *http.Request) {
	if !requirePersonalLearning(w) {
		return
	}
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}
	if learningStore == nil {
		http.Error(w, "learning store unavailable", http.StatusServiceUnavailable)
		return
	}
	raw, err := io.ReadAll(r.Body)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	var bundle struct {
		Entries []learning.Entry `json:"entries"`
	}
	if err := json.Unmarshal(raw, &bundle); err != nil {
		http.Error(w, "Invalid JSON", http.StatusBadRequest)
		return
	}
	uid := learningUserID(r)
	added, skipped := learningStore.ImportMerge(uid, bundle.Entries)
	for _, e := range bundle.Entries {
		if e.ID != "" {
			learning.ScheduleEmbed(e)
		}
	}
	writeJSON(w, http.StatusOK, map[string]any{"added": added, "skipped": skipped})
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
	uid := learningUserID(r)
	agentType := strings.TrimSpace(r.URL.Query().Get("agent_type"))
	agentName := strings.TrimSpace(r.URL.Query().Get("agent_name"))
	if chatHub != nil {
		if info, err := chatHub.GetAgent(agentID); err == nil {
			if agentType == "" {
				agentType = string(info.Type)
			}
			if agentName == "" {
				agentName = info.Name
			}
		}
	}
	count := 0
	globalCount := 0
	collabCount := 0
	if learningStore != nil {
		resolvedUID := learningStore.ResolveUserID(uid)
		count = learningStore.CountForAgent(agentID, agentType, agentName)
		globalCount = learningStore.CountByScope(resolvedUID, learning.ScopeGlobal)
		collabCount = learningStore.CountByScope(resolvedUID, learning.ScopeCollaboration)
	}
	ready := false
	previewRows := 0
	refreshSuggested := false
	suggestTraining := false
	activeVersion := 0
	if info, err := chatHub.GetAgent(agentID); err == nil {
		ctx := buildExpertTrainContext(info)
		if v, ok := ctx["ready"].(bool); ok {
			ready = v
		}
		if v, ok := ctx["preview_rows"].(int); ok {
			previewRows = v
		}
		if v, ok := ctx["refresh_suggested"].(bool); ok {
			refreshSuggested = v
		}
		if v, ok := ctx["suggest_training"].(bool); ok {
			suggestTraining = v
		}
		if v, ok := ctx["active_adapter_version"].(int); ok {
			activeVersion = v
		}
	}
	writeJSON(w, http.StatusOK, map[string]any{
		"agent_id":               agentID,
		"learning_count":         count,
		"global_count":           globalCount,
		"collab_count":           collabCount,
		"embedding_index_ready":  learning.IndexReady(),
		"preview_rows":           previewRows,
		"min_rows":               minRowsFromConfig(),
		"ready_for_lora":         ready,
		"refresh_suggested":      refreshSuggested,
		"suggest_training":       suggestTraining,
		"active_adapter_version": activeVersion,
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
	draft := learning.ExtractDraftFromMessage(msg.Content)
	if draft == "" && !learning.HasLearningTrigger(msg.Content) {
		return
	}
	if draft == "" {
		draft = strings.TrimSpace(msg.Content)
	}
	target := resolveLearningTargetAgent(msg)
	if target == nil {
		return
	}
	emitLearningProposal(msg.Channel, target, draft, learning.CategoryPreference, msg.ID, msg.Channel, "trigger")
}

func emitLearningProposal(channel string, target *protocol.AgentInfo, draft string, cat learning.Category, sourceMsgID, sourceChannel, source string) {
	if target == nil || !personalLearningActive() {
		return
	}
	collabID := learning.ResolveCollabID(channel)
	proposal := protocol.NewMessage(
		protocol.MessageTypeSystemInfo,
		channel,
		protocol.AgentInfo{ID: "system", Name: "System", Type: protocol.AgentTypeGeneral},
		"Learning proposal — confirm in the dialog to save for "+target.Name+".",
	)
	if proposal.Metadata == nil {
		proposal.Metadata = map[string]any{}
	}
	proposal.Metadata["client_action"] = map[string]any{
		"type":              "learning_proposal",
		"source":            source,
		"agent_id":          target.ID,
		"agent_name":        target.Name,
		"agent_type":        string(target.Type),
		"draft":             draft,
		"category":          string(cat),
		"source_message_id": sourceMsgID,
		"source_channel":    sourceChannel,
		"collaboration_id":  collabID,
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
		if a.Type != protocol.AgentTypeCLI && !protocol.IsUserLikeSender(protocol.AgentInfo{Type: a.Type, Name: a.Name}) {
			cp := a
			return &cp
		}
	}
	return nil
}
