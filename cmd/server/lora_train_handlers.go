package main

import (
	"encoding/json"
	"net/http"
	"os"
	"path/filepath"
	"strings"

	"github.com/camronwood/neural-junkie/internal/collaboration"
	"github.com/camronwood/neural-junkie/internal/config"
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

func initLoraTrainManager() {
	if loraTrainMgr != nil {
		return
	}
	root := repoRoot()
	script := filepath.Join(root, "scripts", "lora_train.py")
	loraTrainMgr = train.NewManager("", script, train.ResolvePython(root), hubMessageAdapter{}, collabSnapshotAdapter{})
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
	if path == "preview" {
		handleLoraTrainPreview(w, r)
		return
	}
	if path == "" {
		handleLoraTrainStart(w, r)
		return
	}
	handleLoraTrainByID(w, r, path)
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
		agentID := strings.TrimSpace(body.AgentID)
		if agentID != "" {
			body.LearningRows = export.ExportLearningsRows(learningStore.List(agentID))
		}
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

func buildExpertTrainContext(info *protocol.AgentInfo) map[string]any {
	base := config.DevOllamaCodeModel
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
		if n, err := loraTrainMgr.Preview(expReq); err == nil {
			result["preview_rows"] = n
			result["ready"] = n >= export.MinRows
		}
	}
	return result
}
