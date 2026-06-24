package main

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/camronwood/neural-junkie/internal/hub"
	"github.com/camronwood/neural-junkie/internal/mcp_export"
	"github.com/camronwood/neural-junkie/internal/protocol"
)

func handleImport(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}
	if _, ok := ensureMutationAccess(w, r, ""); !ok {
		return
	}

	// Parse request body
	var request struct {
		FilePath string `json:"file_path"`
	}

	if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
		http.Error(w, "Invalid JSON", http.StatusBadRequest)
		return
	}

	if request.FilePath == "" {
		http.Error(w, "file_path is required", http.StatusBadRequest)
		return
	}

	// Create a command handler to process the import
	commandHandler, err := hub.NewCommandHandler(chatHub)
	if err != nil {
		http.Error(w, fmt.Sprintf("Failed to create command handler: %v", err), http.StatusInternalServerError)
		return
	}

	// Create a mock message for the import command
	msg := &protocol.Message{
		ID:        "import-" + strconv.FormatInt(time.Now().UnixNano(), 10),
		Type:      protocol.MessageTypeSystemInfo,
		Channel:   "general",
		From:      protocol.AgentInfo{ID: "cli", Name: "CLI", Type: "system"},
		Content:   fmt.Sprintf("/import-agent-mcp %s", request.FilePath),
		Timestamp: time.Now(),
	}

	// Process the import command
	response, err := commandHandler.ProcessCommand(context.Background(), msg)
	if err != nil {
		http.Error(w, fmt.Sprintf("Import failed: %v", err), http.StatusInternalServerError)
		return
	}

	// Return success response
	responseData := map[string]interface{}{
		"success": true,
		"message": response.Content,
	}

	// Try to extract agent info from the response
	if strings.Contains(response.Content, "Imported") {
		// Parse agent name from response
		parts := strings.Fields(response.Content)
		for i, part := range parts {
			if part == "agent" && i+2 < len(parts) {
				responseData["name"] = strings.Trim(parts[i+1], "'")
				break
			}
		}
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(responseData)
}

// handleExports lists exported agents for the CLI.

func handleExports(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	storage, err := mcp_export.NewExportStorage()
	if err != nil {
		http.Error(w, fmt.Sprintf("Failed to open export storage: %v", err), http.StatusInternalServerError)
		return
	}

	exports, err := storage.ListExports()
	if err != nil {
		http.Error(w, fmt.Sprintf("Failed to list exports: %v", err), http.StatusInternalServerError)
		return
	}

	type exportJSON struct {
		Name          string `json:"name"`
		Type          string `json:"type"`
		ResourceCount int    `json:"resourceCount"`
		PromptCount   int    `json:"promptCount"`
		FileSize      int64  `json:"fileSize"`
		Description   string `json:"description,omitempty"`
		ExportPath    string `json:"exportPath,omitempty"`
	}

	out := make([]exportJSON, 0, len(exports))
	for _, e := range exports {
		out = append(out, exportJSON{
			Name:          e.Name,
			Type:          e.Type,
			ResourceCount: e.ResourceCount,
			PromptCount:   e.PromptCount,
			FileSize:      e.FileSize,
			Description:   e.Description,
			ExportPath:    e.ExportPath,
		})
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(out)
}

// handleExport exports an agent via the hub command handler.

func handleExport(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var request struct {
		AgentType  string `json:"agent_type"`
		AgentName  string `json:"agent_name"`
		OutputPath string `json:"output_path"`
	}

	if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
		http.Error(w, "Invalid JSON", http.StatusBadRequest)
		return
	}
	if request.AgentName == "" {
		http.Error(w, "agent_name is required", http.StatusBadRequest)
		return
	}

	commandHandler, err := hub.NewCommandHandler(chatHub)
	if err != nil {
		http.Error(w, fmt.Sprintf("Failed to create command handler: %v", err), http.StatusInternalServerError)
		return
	}

	msg := &protocol.Message{
		ID:        "export-" + strconv.FormatInt(time.Now().UnixNano(), 10),
		Type:      protocol.MessageTypeSystemInfo,
		Channel:   "general",
		From:      protocol.AgentInfo{ID: "cli", Name: "CLI", Type: "system"},
		Content:   fmt.Sprintf("/export-agent-mcp %s", request.AgentName),
		Timestamp: time.Now(),
	}

	response, err := commandHandler.ProcessCommand(context.Background(), msg)
	if err != nil {
		http.Error(w, fmt.Sprintf("Export failed: %v", err), http.StatusInternalServerError)
		return
	}

	responseData := map[string]interface{}{
		"success": strings.Contains(response.Content, "✅"),
		"message": response.Content,
	}

	storage, err := mcp_export.NewExportStorage()
	if err == nil {
		if exports, err := storage.ListExports(); err == nil {
			for _, e := range exports {
				if strings.EqualFold(e.Name, request.AgentName) {
					responseData["resources"] = float64(e.ResourceCount)
					responseData["prompts"] = float64(e.PromptCount)
					responseData["size"] = float64(e.FileSize)
					responseData["name"] = e.Name
					responseData["type"] = e.Type
					break
				}
			}
		}
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(responseData)
}

// File system API handlers
