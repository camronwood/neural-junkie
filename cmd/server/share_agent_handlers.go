package main

import (
	"encoding/json"
	"net/http"

	"github.com/camronwood/neural-junkie/internal/hub"
)

// handleShareAgent builds a Share Agent bundle (export + custom rules +
// agent-scoped learnings + LoRA metadata, when present) for one agent and
// returns it as the response body so the desktop client can offer it as a
// download ("Agent Info -> Share").
//
// POST /api/agents/{id}/share
func handleShareAgent(w http.ResponseWriter, r *http.Request, agentID string) {
	// Note: the outer /api/agents/{id}/{action} router already restricts
	// this route to POST/PUT.
	if agentID == "" {
		http.Error(w, "Agent ID required", http.StatusBadRequest)
		return
	}

	commandHandler := chatHub.GetCommandHandler()
	if commandHandler == nil {
		http.Error(w, "Command handler not initialized", http.StatusServiceUnavailable)
		return
	}
	ch, ok := commandHandler.(*hub.CommandHandler)
	if !ok {
		http.Error(w, "Unsupported command handler type", http.StatusInternalServerError)
		return
	}

	bundle, err := ch.ExportAgentBundle(agentID)
	if err != nil {
		http.Error(w, err.Error(), http.StatusNotFound)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("Content-Disposition", "attachment; filename=\"share-agent-bundle.json\"")
	_ = json.NewEncoder(w).Encode(bundle)
}
