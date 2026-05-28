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
	"github.com/camronwood/neural-junkie/internal/hub"
	"github.com/camronwood/neural-junkie/internal/protocol"
)

var editorSessions = hub.NewEditorSessionStore()

func handleDevAgentTurn(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}
	if !requireSoftwareDevPack(w) {
		return
	}
	var req struct {
		WorkspaceID string                   `json:"workspace_id"`
		Instruction string                   `json:"instruction"`
		SessionID   string                   `json:"session_id"`
		Mode        string                   `json:"mode"`
		AgentType   string                   `json:"agent_type"`
		Path        string                   `json:"path"`
		Selection   string                   `json:"selection"`
		Metadata    map[string]interface{}   `json:"metadata"`
		Attachments []map[string]interface{} `json:"attachments"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid JSON", http.StatusBadRequest)
		return
	}
	if strings.TrimSpace(req.WorkspaceID) == "" || strings.TrimSpace(req.Instruction) == "" {
		http.Error(w, "workspace_id and instruction required", http.StatusBadRequest)
		return
	}
	mode := strings.TrimSpace(req.Mode)
	if mode == "" {
		mode = "agent"
	}
	ws, ok := workspaceManager.GetWorkspace(strings.TrimSpace(req.WorkspaceID))
	if !ok {
		http.Error(w, "Workspace not found", http.StatusNotFound)
		return
	}
	ag, info := pickSpecialistForFastEdit(strings.TrimSpace(req.AgentType))
	if ag == nil {
		http.Error(w, "No in-process specialist available", http.StatusBadRequest)
		return
	}
	chName := agent.EditorAgentChannelID(req.WorkspaceID)
	sess := editorSessions.GetOrCreate(req.WorkspaceID, chName, string(info.Type), mode, strings.TrimSpace(req.SessionID))

	prompt := req.Instruction
	if mode == "ask" {
		prompt = "[ASK mode — read-only tools, no file edits]\n" + prompt
	} else {
		prompt += "\n\nEmit [FILE_CHANGE] blocks when edits are needed."
	}
	if req.Selection != "" {
		prompt += "\n\nSelected code:\n```\n" + req.Selection + "\n```"
	}
	relPath := strings.TrimSpace(req.Path)
	content := ""
	if relPath != "" {
		full := filepath.Join(ws.Path, filepath.FromSlash(strings.TrimPrefix(relPath, "/")))
		if b, err := os.ReadFile(full); err == nil {
			content = string(b)
		}
	}

	msg := protocol.NewMessage(protocol.MessageTypeChat, chName, protocol.AgentInfo{
		ID: "human", Name: "Developer", Type: "human",
	}, prompt)
	msg.Metadata = map[string]interface{}{
		"editor_session_id": sess.SessionID,
		"editor_mode":       mode,
		"workspace_context": map[string]interface{}{
			"workspace_name": ws.Name,
			"workspace_path": ws.Path,
			"open_files": []map[string]interface{}{
				{"path": relPath, "language": "text", "content": content, "is_active": true},
			},
		},
	}
	if req.Metadata != nil {
		for k, v := range req.Metadata {
			msg.Metadata[k] = v
		}
	}
	if len(req.Attachments) > 0 {
		msg.Metadata["prompt_attachments"] = req.Attachments
	}

	_ = editorSessions.HistoryMessages(sess.SessionID)
	ctx, cancel := context.WithTimeout(r.Context(), 300*time.Second)
	defer cancel()
	cleaned, proposed, err := agent.RunDevAgentTurn(ctx, ag, chName, msg)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	editorSessions.AppendTurn(sess.SessionID, "user", req.Instruction)
	editorSessions.AppendTurn(sess.SessionID, "assistant", cleaned)

	out := map[string]interface{}{
		"response":    cleaned,
		"proposed":    proposed,
		"session_id":  sess.SessionID,
		"channel":     chName,
		"agent":       info.Name,
		"agent_type":  info.Type,
	}
	if proposed {
		pending := chatHub.GetFileChangeManager().ListPendingFileChanges("default")
		if len(pending) > 0 {
			out["change_ids"] = []string{pending[len(pending)-1].ID}
		}
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(out)
}
