package main

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"regexp"
	"strings"

	"github.com/camronwood/neural-junkie/internal/filechange"
	"github.com/camronwood/neural-junkie/internal/pathutil"
	"github.com/camronwood/neural-junkie/internal/protocol"
)

func handleFileChanges(w http.ResponseWriter, r *http.Request) {
	if r.Method != "GET" {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// Get user ID from query parameter (for now, using a simple approach)
	userID := r.URL.Query().Get("user_id")
	if userID == "" {
		userID = "default" // Default user for demo
	}

	fileChangeManager := chatHub.GetFileChangeManager()
	pendingChanges := fileChangeManager.ListPendingFileChanges(userID)

	// Ensure we always return an array, never null
	if pendingChanges == nil {
		pendingChanges = []*filechange.FileChange{}
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(pendingChanges)
}

func handleProposeFileChangeFromMessage(w http.ResponseWriter, r *http.Request) {
	if r.Method != "POST" {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var req struct {
		Channel     string `json:"channel"`
		MessageID   string `json:"message_id"`
		WorkspaceID string `json:"workspace_id"`
		TargetPath  string `json:"target_path"`
		UserID      string `json:"user_id"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}
	if strings.TrimSpace(req.Channel) == "" || strings.TrimSpace(req.MessageID) == "" {
		http.Error(w, "channel and message_id are required", http.StatusBadRequest)
		return
	}
	if strings.TrimSpace(req.WorkspaceID) == "" {
		http.Error(w, "workspace_id is required", http.StatusBadRequest)
		return
	}

	workspace, ok := workspaceManager.GetWorkspace(req.WorkspaceID)
	if !ok || workspace == nil || strings.TrimSpace(workspace.Path) == "" {
		http.Error(w, "Workspace not found", http.StatusNotFound)
		return
	}

	msgs, err := chatHub.GetMessages(req.Channel, 1000)
	if err != nil {
		http.Error(w, fmt.Sprintf("Failed to load channel messages: %v", err), http.StatusBadRequest)
		return
	}

	var source *protocol.Message
	for i := len(msgs) - 1; i >= 0; i-- {
		if msgs[i] != nil && msgs[i].ID == req.MessageID {
			source = msgs[i]
			break
		}
	}
	if source == nil {
		http.Error(w, "Source message not found", http.StatusNotFound)
		return
	}
	if source.From.Type == "human" || source.Type != protocol.MessageTypeChat {
		http.Error(w, "Only agent chat messages can be proposed from", http.StatusBadRequest)
		return
	}

	newContent := extractLongestCodeFence(source.Content)
	newContent = stripEditorLineNumberPrefixes(newContent)
	if strings.TrimSpace(newContent) == "" {
		http.Error(w, "No editable content block found in message", http.StatusBadRequest)
		return
	}

	targetPath := strings.TrimSpace(req.TargetPath)
	if targetPath == "" {
		targetPath = inferTargetPathFromWorkspaceContext(source)
	}
	if targetPath == "" {
		http.Error(w, "No target file path available", http.StatusBadRequest)
		return
	}

	candidate := filepath.Clean(targetPath)
	if !filepath.IsAbs(candidate) {
		candidate = filepath.Join(workspace.Path, candidate)
	}
	absTarget, err := pathutil.WithinRoot(workspace.Path, candidate)
	if err != nil {
		http.Error(w, "Target path is outside workspace", http.StatusBadRequest)
		return
	}
	targetPath = absTarget

	info, statErr := os.Stat(targetPath)
	if statErr != nil || info.IsDir() {
		http.Error(w, "Target file does not exist or is not a file", http.StatusBadRequest)
		return
	}

	oldBytes, readErr := os.ReadFile(targetPath)
	if readErr != nil {
		http.Error(w, fmt.Sprintf("Failed to read target file: %v", readErr), http.StatusBadRequest)
		return
	}

	fileChangeManager := chatHub.GetFileChangeManager()
	fileChangeManager.GetExecutor().SetWorkspaceRoot(workspace.Path)
	change, err := fileChangeManager.ProposeFileChange(
		filechange.FileOperationEdit,
		targetPath,
		"",
		"",
		string(oldBytes),
		newContent,
		source.From,
		req.Channel,
	)
	if err != nil {
		http.Error(w, fmt.Sprintf("Failed to create file change proposal: %v", err), http.StatusBadRequest)
		return
	}

	systemFrom := protocol.AgentInfo{
		ID:     "system",
		Name:   "System",
		Type:   protocol.AgentTypeGeneral,
		Status: "active",
	}
	infoMsg := protocol.NewMessage(
		protocol.MessageTypeSystemInfo,
		req.Channel,
		systemFrom,
		fmt.Sprintf("Created file change proposal `%s` for `%s` from message `%s`.", change.ID, change.FilePath, source.ID),
	)
	if sendErr := chatHub.SendMessage(infoMsg); sendErr != nil {
		log.Printf("Failed to send propose-from-message system message: %v", sendErr)
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(change)
}

func extractLongestCodeFence(content string) string {
	re := regexp.MustCompile("(?s)```[a-zA-Z0-9_-]*\\s*\\n(.*?)\\n```")
	matches := re.FindAllStringSubmatch(content, -1)
	if len(matches) == 0 {
		return ""
	}
	longest := ""
	for _, m := range matches {
		if len(m) < 2 {
			continue
		}
		if len(m[1]) > len(longest) {
			longest = m[1]
		}
	}
	return longest
}

func stripEditorLineNumberPrefixes(content string) string {
	if content == "" {
		return content
	}
	re := regexp.MustCompile(`(?m)^\s*\d+\s*\|\s?`)
	return re.ReplaceAllString(content, "")
}

func inferTargetPathFromWorkspaceContext(msg *protocol.Message) string {
	if msg == nil || msg.Metadata == nil {
		return ""
	}
	wsCtxRaw, ok := msg.Metadata["workspace_context"]
	if !ok {
		return ""
	}
	wsCtx, ok := wsCtxRaw.(map[string]interface{})
	if !ok {
		return ""
	}
	openFilesRaw, ok := wsCtx["open_files"]
	if !ok {
		return ""
	}
	openFiles, ok := openFilesRaw.([]interface{})
	if !ok {
		return ""
	}
	for _, f := range openFiles {
		m, ok := f.(map[string]interface{})
		if !ok {
			continue
		}
		isActive, _ := m["is_active"].(bool)
		path, _ := m["path"].(string)
		if isActive && strings.TrimSpace(path) != "" {
			return path
		}
	}
	if len(openFiles) > 0 {
		if first, ok := openFiles[0].(map[string]interface{}); ok {
			if path, ok := first["path"].(string); ok {
				return path
			}
		}
	}
	return ""
}

func handleApproveFileChange(w http.ResponseWriter, r *http.Request) {
	if r.Method != "POST" {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// Extract change ID from URL path
	path := strings.TrimPrefix(r.URL.Path, "/api/file-changes/approve/")
	if path == "" {
		http.Error(w, "Change ID required", http.StatusBadRequest)
		return
	}

	// Get user ID from request body or query
	userID := r.URL.Query().Get("user_id")
	if userID == "" {
		userID = "default" // Default user for demo
	}

	fileChangeManager := chatHub.GetFileChangeManager()
	change, err := fileChangeManager.ApproveFileChange(path, userID)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	chatHub.NotifyFileChangeApproved(change, userID)

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(change)
}

func handleRejectFileChange(w http.ResponseWriter, r *http.Request) {
	if r.Method != "POST" {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// Extract change ID from URL path
	path := strings.TrimPrefix(r.URL.Path, "/api/file-changes/reject/")
	if path == "" {
		http.Error(w, "Change ID required", http.StatusBadRequest)
		return
	}

	// Get user ID and reason from request body
	var req struct {
		UserID string `json:"user_id"`
		Reason string `json:"reason"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		req.UserID = "default"
		req.Reason = "No reason provided"
	}

	if req.UserID == "" {
		req.UserID = "default"
	}

	fileChangeManager := chatHub.GetFileChangeManager()
	change, err := fileChangeManager.RejectFileChange(path, req.UserID, req.Reason)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(change)
}

func handleFileChangeDiff(w http.ResponseWriter, r *http.Request) {
	if r.Method != "GET" {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// Extract change ID from URL path
	path := strings.TrimPrefix(r.URL.Path, "/api/file-changes/")
	if path == "" {
		http.Error(w, "Change ID required", http.StatusBadRequest)
		return
	}

	fileChangeManager := chatHub.GetFileChangeManager()
	change, err := fileChangeManager.GetFileChange(path)
	if err != nil {
		http.Error(w, err.Error(), http.StatusNotFound)
		return
	}

	// Generate diff for edit operations
	var diff string
	if change.Operation == "edit" {
		// Simple diff implementation - in production, use a proper diff library
		diff = "--- Old content\n+++ New content\n"
		oldLines := strings.Split(change.OldContent, "\n")
		newLines := strings.Split(change.NewContent, "\n")

		maxLines := len(oldLines)
		if len(newLines) > maxLines {
			maxLines = len(newLines)
		}

		for i := 0; i < maxLines; i++ {
			oldLine := ""
			newLine := ""

			if i < len(oldLines) {
				oldLine = oldLines[i]
			}
			if i < len(newLines) {
				newLine = newLines[i]
			}

			if oldLine != newLine {
				diff += fmt.Sprintf("@@ -%d +%d @@\n", i+1, i+1)
				if oldLine != "" {
					diff += fmt.Sprintf("-%s\n", oldLine)
				}
				if newLine != "" {
					diff += fmt.Sprintf("+%s\n", newLine)
				}
			}
		}
	}

	response := map[string]interface{}{
		"change": change,
		"diff":   diff,
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

// handleToolApprovals creates a new tool approval request (called by the hook binary).
// The request blocks until the user approves/rejects or a timeout occurs.
