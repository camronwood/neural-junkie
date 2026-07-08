package main

import (
	"encoding/json"
	"io"
	"log"
	"net/http"
	"os"
	"strconv"
	"strings"

	"github.com/camronwood/neural-junkie/internal/agent"
	"github.com/camronwood/neural-junkie/internal/config"
	"github.com/camronwood/neural-junkie/internal/hub"
	"github.com/camronwood/neural-junkie/internal/protocol"
)

func defaultHumanSender() (id string, name string, agentType protocol.AgentType) {
	if n := strings.TrimSpace(config.AppConfig().ResolvedAutomation().HumanName); n != "" {
		slug := strings.ToLower(strings.ReplaceAll(n, " ", "-"))
		return "human-" + slug, n, protocol.AgentTypeGeneral
	}
	if n := strings.TrimSpace(os.Getenv("NEURAL_JUNKIE_HUMAN_NAME")); n != "" {
		slug := strings.ToLower(strings.ReplaceAll(n, " ", "-"))
		return "human-" + slug, n, protocol.AgentTypeGeneral
	}
	if u := strings.TrimSpace(os.Getenv("USER")); u != "" {
		return "human-" + strings.ToLower(u), u, protocol.AgentTypeGeneral
	}
	return "human-user", "Human User", protocol.AgentTypeGeneral
}

func slugifyHumanID(name string) string {
	s := strings.ToLower(strings.TrimSpace(name))
	s = strings.ReplaceAll(s, " ", "-")
	var b strings.Builder
	for _, r := range s {
		if (r >= 'a' && r <= 'z') || (r >= '0' && r <= '9') || r == '-' || r == '_' {
			b.WriteRune(r)
		}
	}
	out := b.String()
	if out == "" {
		out = "anonymous"
	}
	return out
}

func handleMessages(w http.ResponseWriter, r *http.Request) {
	channel := r.URL.Query().Get("channel")
	if channel == "" {
		channel = "general"
	}
	if !ensureChannelReadAccess(w, r, channel) {
		return
	}

	limit := 50
	if v := strings.TrimSpace(r.URL.Query().Get("limit")); v != "" {
		if n, err := strconv.Atoi(v); err == nil && n > 0 {
			limit = n
		}
	}
	beforeID := strings.TrimSpace(r.URL.Query().Get("before"))

	messages, err := chatHub.GetMessagesPage(channel, limit, beforeID)
	if err != nil {
		http.Error(w, err.Error(), http.StatusNotFound)
		return
	}

	secret := strings.TrimSpace(config.AppConfig().ResolvedSecurity().FullMetadataSecret)
	if secret == "" {
		secret = strings.TrimSpace(os.Getenv("NEURAL_JUNKIE_FULL_METADATA_SECRET"))
	}
	allowFull := secret != "" && strings.TrimSpace(r.Header.Get("X-NJ-Full-Metadata")) == secret

	out := make([]*protocol.Message, 0, len(messages))
	for _, m := range messages {
		cp, cerr := protocol.CloneMessage(m)
		if cerr != nil || cp == nil {
			continue
		}
		if !allowFull {
			protocol.RedactImageBinaryMetadata(cp)
			agent.RedactSidecarSecrets(cp)
		}
		out = append(out, cp)
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(out)
}

func handleBroadcastDirect(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var msg protocol.Message
	if err := json.NewDecoder(r.Body).Decode(&msg); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	if _, ok := ensureMutationAccess(w, r, msg.Channel); !ok {
		return
	}

	chatHub.BroadcastDirect(msg.Channel, &msg)
	w.WriteHeader(http.StatusNoContent)
}

func writeSendMessageOKResponse(w http.ResponseWriter) {
	resp := map[string]string{"status": "ok"}
	if h := chatHub.GetCommandHandler(); h != nil {
		if ch, ok := h.(*hub.CommandHandler); ok {
			if collabCh, collabID, ok2 := ch.TakeCollaborateRedirect(); ok2 {
				resp["collaboration_channel"] = collabCh
				resp["collaboration_id"] = collabID
			}
			if dmCh, ok2 := ch.TakeDMRedirect(); ok2 {
				resp["dm_channel"] = dmCh
			}
		}
	}
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(resp)
}

func handleSendMessage(w http.ResponseWriter, r *http.Request) {
	// Try to decode as full message first (for agents)
	var fullMsg protocol.Message
	body, err := io.ReadAll(r.Body)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	// Try to parse as full message (agents send this)
	if err := json.Unmarshal(body, &fullMsg); err == nil && fullMsg.ID != "" {
		if _, ok := ensureMutationAccess(w, r, fullMsg.Channel); !ok {
			return
		}
		// This is a full message from an agent, use it directly
		if err := chatHub.SendMessage(&fullMsg); err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		writeSendMessageOKResponse(w)
		return
	}

	// Otherwise, parse as simplified request (for UI/human users)
	var req struct {
		Channel       string                 `json:"channel"`
		Content       string                 `json:"content"`
		Type          string                 `json:"type"`
		ThreadID      string                 `json:"thread_id,omitempty"`
		IsThreadReply bool                   `json:"is_thread_reply,omitempty"`
		ReplyTo       string                 `json:"reply_to,omitempty"`
		Metadata      map[string]interface{} `json:"metadata,omitempty"`
		From          *struct {
			ID   string `json:"id"`
			Name string `json:"name"`
			Type string `json:"type"`
		} `json:"from"`
	}

	if err := json.Unmarshal(body, &req); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	sess, ok := ensureMutationAccess(w, r, req.Channel)
	if !ok {
		return
	}

	msgType := protocol.MessageType(req.Type)
	if msgType == "" {
		msgType = protocol.MessageTypeChat
	}

	senderID, senderName, senderType := defaultHumanSender()
	if sess != nil && strings.TrimSpace(sess.Username) != "" {
		// Always bind the sender identity to the authenticated hub session when present.
		// This prevents confusing presence/@mention behavior when NEURAL_JUNKIE_HUMAN_NAME differs
		// across machines, and keeps LAN room membership auditable.
		senderName = sess.Username
		senderID = "human-" + slugifyHumanID(sess.Username)
		senderType = protocol.AgentType("human")
	}

	// Ignore client-supplied sender unless a hub token is configured (prevents browser/extension spoofing on loopback).
	if req.From != nil && hub.HubTokenConfigured() {
		if req.From.ID != "" {
			senderID = req.From.ID
		}
		if req.From.Name != "" {
			senderName = req.From.Name
		}
		if req.From.Type != "" {
			senderType = protocol.AgentType(req.From.Type)
		}
	}

	msg := protocol.NewMessage(
		msgType,
		req.Channel,
		protocol.AgentInfo{
			ID:   senderID,
			Name: senderName,
			Type: senderType,
		},
		req.Content,
	)

	// Preserve thread context if provided
	if req.ThreadID != "" {
		msg.ThreadID = req.ThreadID
		msg.IsThreadReply = req.IsThreadReply
	}

	// Preserve reply-to if provided
	if req.ReplyTo != "" {
		msg.ReplyTo = req.ReplyTo
	} else if req.Metadata != nil {
		if rt, ok := req.Metadata["reply_to"].(string); ok && strings.TrimSpace(rt) != "" {
			msg.ReplyTo = strings.TrimSpace(rt)
		}
	}

	// Copy metadata from the request (workspace_context, credentials, etc.)
	if req.Metadata != nil {
		if msg.Metadata == nil {
			msg.Metadata = make(map[string]interface{})
		}
		for k, v := range req.Metadata {
			msg.Metadata[k] = v
		}

		// Size guard: truncate workspace_context if it's too large
		if wsCtx, ok := msg.Metadata["workspace_context"]; ok {
			raw, _ := json.Marshal(wsCtx)
			if len(raw) > 500*1024 { // 500KB limit
				log.Printf("Warning: workspace_context too large (%d bytes), removing open_files to reduce size", len(raw))
				if ctxMap, ok := wsCtx.(map[string]interface{}); ok {
					// Remove open_files to drastically reduce size; keep the file tree
					delete(ctxMap, "open_files")
					msg.Metadata["workspace_context"] = ctxMap
				}
			}
		}
		agent.MergeCodebaseAttachments(msg)
		agent.SanitizeInboundMessageMetadata(msg)
	}

	chatHub.AnnotateInboundUserMessage(msg, sess.Username)

	if err := chatHub.SendMessage(msg); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	maybeEmitLearningProposal(msg)

	writeSendMessageOKResponse(w)
}
