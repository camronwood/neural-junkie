package main

import (
	"context"
	"encoding/json"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/camronwood/neural-junkie/internal/agent"
	"github.com/camronwood/neural-junkie/internal/config"
	"github.com/camronwood/neural-junkie/internal/hub"
	"github.com/camronwood/neural-junkie/internal/protocol"
)

func handleDevFastEdit(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}
	if !requireSoftwareDevPack(w) {
		return
	}
	var req struct {
		WorkspaceID string                 `json:"workspace_id"`
		Path        string                 `json:"path"`
		Instruction string                 `json:"instruction"`
		Selection   string                 `json:"selection"`
		AgentType   string                 `json:"agent_type"`
		Metadata    map[string]interface{} `json:"metadata"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid JSON", http.StatusBadRequest)
		return
	}
	if strings.TrimSpace(req.WorkspaceID) == "" || strings.TrimSpace(req.Instruction) == "" {
		http.Error(w, "workspace_id and instruction required", http.StatusBadRequest)
		return
	}
	ws, ok := workspaceManager.GetWorkspace(strings.TrimSpace(req.WorkspaceID))
	if !ok {
		http.Error(w, "Workspace not found", http.StatusNotFound)
		return
	}
	ag, info := pickSpecialistForFastEdit(strings.TrimSpace(req.AgentType))
	if ag == nil {
		http.Error(w, "No in-process specialist available; enable Software development pack", http.StatusBadRequest)
		return
	}
	relPath := strings.TrimSpace(req.Path)
	content := ""
	if relPath != "" {
		full := filepath.Join(ws.Path, filepath.FromSlash(strings.TrimPrefix(relPath, "/")))
		b, err := os.ReadFile(full)
		if err == nil {
			content = string(b)
		}
	}
	prompt := req.Instruction
	if req.Selection != "" {
		prompt += "\n\nSelected code:\n```\n" + req.Selection + "\n```"
	} else if content != "" {
		prompt += "\n\nFile " + relPath + ":\n```\n" + content + "\n```"
	}
	prompt += "\n\nUse search_replace or apply_patch for edits (preferred). Use propose_file_edit for new files. Emit [FILE_CHANGE] only if tools fail."

	msg := protocol.NewMessage(protocol.MessageTypeChat, "dev-fast-edit", protocol.AgentInfo{
		ID: "human", Name: "Developer", Type: "human",
	}, prompt)
	if req.Metadata != nil {
		msg.Metadata = req.Metadata
	} else {
		msg.Metadata = map[string]interface{}{}
	}
	msg.Metadata["workspace_context"] = map[string]interface{}{
		"workspace_name": ws.Name,
		"workspace_path": ws.Path,
		"file_tree":      "",
		"open_files": []map[string]interface{}{
			{"path": relPath, "language": "text", "content": content, "is_active": true},
		},
	}

	ctx, cancel := context.WithTimeout(r.Context(), 180*time.Second)
	defer cancel()
	cleaned, proposed, err := agent.RunFastEdit(ctx, ag, "dev-fast-edit", msg)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	out := map[string]interface{}{
		"response":  cleaned,
		"proposed":  proposed,
		"agent":     info.Name,
		"agent_type": info.Type,
	}
	if proposed {
		pending := chatHub.GetFileChangeManager().ListPendingFileChanges("default")
		if len(pending) > 0 {
			out["change_id"] = pending[len(pending)-1].ID
		}
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(out)
}

func pickSpecialistForFastEdit(agentType string) (*agent.Agent, protocol.AgentInfo) {
	if agentType == "" {
		agentType = "backend"
	}
	ch, ok := chatHub.GetCommandHandler().(*hub.CommandHandler)
	if !ok {
		return nil, protocol.AgentInfo{}
	}
	for _, info := range chatHub.ListAgents() {
		if info == nil || string(info.Type) != agentType {
			continue
		}
		if ra := ch.ResolveRuntimeAgentForFastEdit(info.ID); ra != nil {
			return ra, *info
		}
	}
	for _, info := range chatHub.ListAgents() {
		if info == nil || !config.IsDevSpecialistType(string(info.Type)) {
			continue
		}
		if ra := ch.ResolveRuntimeAgentForFastEdit(info.ID); ra != nil {
			return ra, *info
		}
	}
	return nil, protocol.AgentInfo{}
}
