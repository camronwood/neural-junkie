package main

import (
	"encoding/json"
	"log"
	"net/http"
	"os"
	"strings"

	"github.com/camronwood/neural-junkie/internal/agent"
	"github.com/camronwood/neural-junkie/internal/hub"
	"github.com/camronwood/neural-junkie/internal/protocol"
)

func handleDebugHubMemory(w http.ResponseWriter, r *http.Request) {
	if os.Getenv("NEURAL_JUNKIE_DEBUG") != "1" {
		http.NotFound(w, r)
		return
	}
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	enc := json.NewEncoder(w)
	enc.SetIndent("", "  ")
	if err := enc.Encode(chatHub.HubMemoryReport()); err != nil {
		log.Printf("handleDebugHubMemory: %v", err)
	}
}

func handleDebugDelegationResolve(w http.ResponseWriter, r *http.Request) {
	if os.Getenv("NEURAL_JUNKIE_DEBUG") != "1" {
		http.NotFound(w, r)
		return
	}
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}
	fromName := strings.TrimSpace(r.URL.Query().Get("from"))
	q := strings.TrimSpace(r.URL.Query().Get("q"))
	if fromName == "" || q == "" {
		http.Error(w, "from and q query parameters required", http.StatusBadRequest)
		return
	}
	ch, ok := chatHub.GetCommandHandler().(*hub.CommandHandler)
	if !ok || ch == nil {
		http.Error(w, "command handler unavailable", http.StatusServiceUnavailable)
		return
	}
	var fromInfo protocol.AgentInfo
	for _, ag := range chatHub.ListAgents() {
		if ag != nil && strings.EqualFold(ag.Name, fromName) {
			fromInfo = *ag
			break
		}
	}
	if fromInfo.ID == "" {
		http.Error(w, "agent not found: "+fromName, http.StatusNotFound)
		return
	}
	candidates := ch.ResolveConsultants(fromInfo, q)
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"from":       fromInfo.Name,
		"question":   q,
		"enabled":    ch.DelegationEnabled(),
		"candidates": candidates,
	})
}

func handleDebugChannelContext(w http.ResponseWriter, r *http.Request) {
	if os.Getenv("NEURAL_JUNKIE_DEBUG") != "1" {
		http.NotFound(w, r)
		return
	}
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}
	channel := strings.TrimSpace(r.URL.Query().Get("channel"))
	if channel == "" {
		http.Error(w, "channel query parameter required", http.StatusBadRequest)
		return
	}
	summary := chatHub.GetChannelSessionSummary(channel)
	msgs, _ := chatHub.GetMessages(channel, 50)
	out := map[string]interface{}{
		"channel":       channel,
		"channel_type":  chatHub.GetChannelType(channel),
		"summary_len":   len(summary),
		"history_count": len(msgs),
		"has_summary":   summary != "",
	}
	if summary != "" {
		out["session_summary"] = summary
	}
	if sample := strings.TrimSpace(r.URL.Query().Get("message")); sample != "" {
		msg := protocol.NewMessage(protocol.MessageTypeQuestion, channel, protocol.AgentInfo{ID: "debug", Name: "Debug", Type: "human"}, sample)
		if msg.Metadata == nil {
			msg.Metadata = map[string]interface{}{}
		}
		if mode := strings.TrimSpace(r.URL.Query().Get("conversation_mode")); mode != "" {
			msg.Metadata[agent.MetadataConversationMode] = mode
		}
		if scope := strings.TrimSpace(r.URL.Query().Get("context_scope")); scope != "" {
			msg.Metadata[agent.MetadataContextScope] = scope
		}
		chType := chatHub.GetChannelType(channel)
		intent := classifyTurnIntentForDebug(msg, chType)
		out["conversation_mode"] = agent.EffectiveConversationMode(msg, chType)
		out["resolved_intent"] = intent.String()
		out["context_scope"] = agent.ResolveContextScope(msg)
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(out)
}

func handleDebugRoutingClassify(w http.ResponseWriter, r *http.Request) {
	if os.Getenv("NEURAL_JUNKIE_DEBUG") != "1" {
		http.NotFound(w, r)
		return
	}
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}
	q := strings.TrimSpace(r.URL.Query().Get("q"))
	if q == "" {
		http.Error(w, "q query parameter required", http.StatusBadRequest)
		return
	}
	agentType := strings.TrimSpace(r.URL.Query().Get("agent_type"))
	loraTags := collectInstalledLoRATags(r.Context())
	dec := classifyTask(r.Context(), appConfig, q, agentType, "", false, loraTags)
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(dec)
}

func classifyTurnIntentForDebug(msg *protocol.Message, chType protocol.ChannelType) agent.TurnIntent {
	// Lightweight mirror for debug endpoint without spinning up a full agent.
	return agent.ClassifyTurnIntentPublic(msg, chType, "", nil)
}
