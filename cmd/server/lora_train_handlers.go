package main

import (
	"encoding/json"
	"net/http"
	"os"
	"path/filepath"
	"strings"

	"github.com/camronwood/neural-junkie/internal/collaboration"
	"github.com/camronwood/neural-junkie/internal/hfhub"
	"github.com/camronwood/neural-junkie/internal/lora/export"
	"github.com/camronwood/neural-junkie/internal/lora/train"
	"github.com/camronwood/neural-junkie/internal/protocol"
)

var loraTrainMgr *train.Manager

type hubMessageAdapter struct{}

func (hubMessageAdapter) GetMessages(channel string, limit int) ([]*protocol.Message, error) {
	return chatHub.GetMessages(channel, limit)
}

func (hubMessageAdapter) GetThreadMessages(threadID string, limit int) ([]*protocol.Message, error) {
	return chatHub.GetThreadMessages(threadID, limit)
}

type collabSnapshotAdapter struct{}

func (collabSnapshotAdapter) GetCollaborationSnapshot(id string) (*collaboration.Collaboration, error) {
	return chatHub.GetCollaborationManager().GetCollaborationSnapshot(id)
}

func repoRoot() string {
	if root := strings.TrimSpace(os.Getenv("NEURAL_JUNKIE_ROOT")); root != "" {
		return root
	}
	candidates := []string{
		".",
		filepath.Join("..", ".."),
		filepath.Join(filepath.Dir(os.Args[0]), "..", ".."),
	}
	for _, c := range candidates {
		p := filepath.Join(c, "scripts", "lora_train.py")
		if _, err := os.Stat(p); err == nil {
			abs, _ := filepath.Abs(c)
			return abs
		}
	}
	return "."
}

func handleLoraTrainRoute(w http.ResponseWriter, r *http.Request) {
	if !requireLoRACapability(w, capLoRATraining) {
		return
	}
	initLoraTrainManager()
	path := strings.TrimPrefix(r.URL.Path, "/api/lora/train")
	path = strings.Trim(path, "/")
	if path == "expert-context" {
		handleLoraTrainExpertContext(w, r)
		return
	}
	if path == "bases" {
		handleLoraTrainBases(w, r)
		return
	}
	if path == "preview" {
		handleLoraTrainPreview(w, r)
		return
	}
	if path == "dataset-preview" {
		handleLoraTrainDatasetPreview(w, r)
		return
	}
	if path == "index-bootstrap" {
		handleLoraTrainIndexBootstrap(w, r)
		return
	}
	if path == "" {
		switch r.Method {
		case http.MethodGet:
			handleLoraTrainList(w, r)
		case http.MethodPost:
			handleLoraTrainStart(w, r)
		default:
			http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		}
		return
	}
	if path == "active" {
		handleLoraTrainActive(w, r)
		return
	}
	handleLoraTrainByID(w, r, path)
}

func handleLoraTrainList(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"jobs": loraTrainMgr.List(50)})
}

func handleLoraTrainActive(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}
	job, ok := loraTrainMgr.Active()
	if !ok {
		writeJSON(w, http.StatusOK, map[string]any{"active": false})
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"active": true, "job": job})
}

func handleLoraTrainDatasetPreview(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodGet:
		q := r.URL.Query()
		req := exportRequestFromQuery(q)
		if q.Get("include_learnings") == "1" && learningStore != nil && personalLearningActive() {
			agentID := strings.TrimSpace(q.Get("agent_id"))
			if agentID != "" {
				req.ExtraRows = export.AppendExtraRows(export.ExportLearningsRows(learningStore.List(agentID)), nil)
			}
		}
		if q.Get("incremental") == "1" && loraAdapterRegistry != nil {
			agentID := strings.TrimSpace(q.Get("agent_id"))
			if e, ok := loraAdapterRegistry.ActiveForAgent(agentID); ok {
				req.SinceJobExportedAt = e.ExportedAt
			}
		}
		rows, err := loraTrainMgr.PreviewDataset(req)
		if err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
		writeJSON(w, http.StatusOK, map[string]any{"rows": rows, "count": len(rows), "min_rows": export.MinRows})
	case http.MethodPost:
		var body train.StartRequest
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			http.Error(w, "Invalid JSON: "+err.Error(), http.StatusBadRequest)
			return
		}
		attachLearningRows(&body)
		req := enrichLoraTrainRequest(&body)
		rows, err := loraTrainMgr.PreviewDataset(req)
		if err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
		writeJSON(w, http.StatusOK, map[string]any{"rows": rows, "count": len(rows), "min_rows": export.MinRows})
	default:
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
	}
}

func handleLoraTrainIndexBootstrap(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}
	var body struct {
		AgentID string `json:"agent_id"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		http.Error(w, "Invalid JSON: "+err.Error(), http.StatusBadRequest)
		return
	}
	agentID := strings.TrimSpace(body.AgentID)
	if agentID == "" {
		http.Error(w, "agent_id required", http.StatusBadRequest)
		return
	}
	index, err := loadRepoIndexForAgent(agentID)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	rows := export.BootstrapFromIndex(index)
	if len(rows) == 0 {
		http.Error(w, "no bootstrap rows could be generated from the index", http.StatusBadRequest)
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"rows": rows, "count": len(rows)})
}

func exportRequestFromQuery(q map[string][]string) export.Request {
	get := func(k string) string {
		if v, ok := q[k]; ok && len(v) > 0 {
			return strings.TrimSpace(v[0])
		}
		return ""
	}
	return export.Request{
		Source:            export.SourceKind(get("source")),
		SourceID:          get("source_id"),
		ThreadID:          get("thread_id"),
		AgentName:         get("agent_name"),
		ApprovedTasksOnly: get("approved_tasks_only") == "1",
	}
}

func handleLoraTrainPreview(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}
	q := r.URL.Query()
	req := export.Request{
		Source:    export.SourceKind(strings.TrimSpace(q.Get("source"))),
		SourceID:  strings.TrimSpace(q.Get("source_id")),
		ThreadID:  strings.TrimSpace(q.Get("thread_id")),
		AgentName: strings.TrimSpace(q.Get("agent_name")),
	}
	n, err := loraTrainMgr.Preview(req)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	learningRows := 0
	if q.Get("include_learnings") == "1" && learningStore != nil && personalLearningActive() {
		agentID := strings.TrimSpace(q.Get("agent_id"))
		if agentID != "" {
			learningRows = len(export.ExportLearningsRows(learningStore.List(agentID)))
		}
	}
	writeJSON(w, http.StatusOK, map[string]any{
		"row_count":         n + learningRows,
		"chat_rows":         n,
		"learning_rows":     learningRows,
		"include_learnings": learningRows > 0,
		"min_rows":          export.MinRows,
	})
}

func handleLoraTrainStart(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}
	var body train.StartRequest
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		http.Error(w, "Invalid JSON: "+err.Error(), http.StatusBadRequest)
		return
	}
	if body.IncludeLearnings && learningStore != nil && personalLearningActive() {
		attachLearningRows(&body)
	}
	job, err := loraTrainMgr.Start(r.Context(), body)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	writeJSON(w, http.StatusOK, job)
}

func handleLoraTrainByID(w http.ResponseWriter, r *http.Request, id string) {
	switch r.Method {
	case http.MethodGet:
		job, ok := loraTrainMgr.Get(id)
		if !ok {
			http.Error(w, "job not found", http.StatusNotFound)
			return
		}
		writeJSON(w, http.StatusOK, job)
	case http.MethodDelete:
		if err := loraTrainMgr.Cancel(id); err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
		job, _ := loraTrainMgr.Get(id)
		writeJSON(w, http.StatusOK, job)
	default:
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
	}
}

func handleLoraTrainExpertContext(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}
	agentID := strings.TrimSpace(r.URL.Query().Get("agent_id"))
	if agentID == "" {
		http.Error(w, "agent_id required", http.StatusBadRequest)
		return
	}
	info, err := chatHub.GetAgent(agentID)
	if err != nil {
		http.Error(w, err.Error(), http.StatusNotFound)
		return
	}
	writeJSON(w, http.StatusOK, buildExpertTrainContext(info))
}

func handleLoraTrainBases(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{
		"bases":              hfhub.LoRATrainingBases(),
		"default_code_base":  hfhub.DefaultLoRATrainingCodeBase,
		"default_biology_base": hfhub.BiologyLoRABaseTag,
	})
}

func buildExpertTrainContext(info *protocol.AgentInfo) map[string]any {
	base := hfhub.DefaultLoRATrainingBaseForAgent(string(info.Type))
	channels := chatHub.GetAgentChannels(info.ID)
	channelID := ""
	if len(channels) > 0 {
		channelID = channels[0]
	}

	result := map[string]any{
		"agent_id":                  info.ID,
		"agent_name":                info.Name,
		"agent_type":                string(info.Type),
		"suggested_base_ollama_tag": base,
		"supported_bases":           hfhub.LoRATrainingBases(),
		"min_rows":                  export.MinRows,
		"preview_rows":              0,
		"ready":                     false,
	}
	if channelID != "" {
		result["source_id"] = channelID
	}

	var expReq export.Request
	switch info.Type {
	case protocol.AgentTypeRepo:
		result["source"] = "repo"
		result["agent_name"] = info.Name
		if strings.TrimSpace(info.RepositoryPath) != "" {
			result["suggested_ollama_tag"] = hfhub.RepoLoRATag(info.RepositoryPath)
		} else {
			result["suggested_ollama_tag"] = "nj-repo-custom:14b"
		}
		expReq = export.Request{
			Source:    export.SourceRepo,
			SourceID:  channelID,
			AgentName: info.Name,
		}
	case protocol.AgentTypeAssistant:
		result["source"] = "channel"
		result["suggested_ollama_tag"] = hfhub.AssistantLoRATag(info.Name)
		expReq = export.Request{
			Source:    export.SourceChannel,
			SourceID:  channelID,
			AgentName: info.Name,
		}
	default:
		result["source"] = "channel"
		if tag := hfhub.SpecialistLoRATag(string(info.Type)); tag != "" {
			result["suggested_ollama_tag"] = tag
		} else if m := strings.TrimSpace(info.Model); strings.HasPrefix(m, "nj-") {
			result["suggested_ollama_tag"] = m
		}
		expReq = export.Request{
			Source:    export.SourceChannel,
			SourceID:  channelID,
			AgentName: info.Name,
		}
	}

	if channelID != "" && loraTrainMgr != nil {
		chatRows := 0
		if n, err := loraTrainMgr.Preview(expReq); err == nil {
			chatRows = n
		}
		learningRows := 0
		if learningStore != nil && personalLearningActive() {
			learningRows = len(export.ExportLearningsRows(learningStore.List(info.ID)))
		}
		total := chatRows + learningRows
		result["chat_rows"] = chatRows
		result["learning_rows"] = learningRows
		result["preview_rows"] = total
		result["ready"] = total >= export.MinRows

		deltaRows := 0
		if loraAdapterRegistry != nil {
			if active, ok := loraAdapterRegistry.ActiveForAgent(info.ID); ok {
				result["active_adapter_version"] = active.Version
				result["active_adapter_id"] = active.ID
				result["prior_adapter_id"] = active.ID
				if rows, err := loraTrainMgr.PreviewDataset(export.Request{
					Source:             expReq.Source,
					SourceID:           expReq.SourceID,
					AgentName:          expReq.AgentName,
					SinceJobExportedAt: active.ExportedAt,
				}); err == nil {
					deltaRows = len(rows)
				}
			}
		}
		result["delta_rows"] = deltaRows
		result["refresh_suggested"] = deltaRows >= export.DefaultRefreshDelta
	}
	return result
}
