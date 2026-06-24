package main

import (
	"encoding/json"
	"net/http"
	"strconv"
	"strings"

	"github.com/camronwood/neural-junkie/internal/agent"
	"github.com/camronwood/neural-junkie/internal/hub"
	"github.com/camronwood/neural-junkie/internal/protocol"
)

func threadChannelForACL(threadID string, r *http.Request) string {
	if c := strings.TrimSpace(r.URL.Query().Get("channel")); c != "" {
		return c
	}
	if chatHub == nil {
		return ""
	}
	meta, err := chatHub.GetThreadMetadata(threadID)
	if err != nil || meta == nil {
		return ""
	}
	return strings.TrimSpace(meta.Channel)
}

func handleThreads(w http.ResponseWriter, r *http.Request) {
	// Parse URL path: /api/threads/{threadID}/messages or /api/threads/{threadID}/reply or /api/threads/{threadID}/metadata
	path := r.URL.Path

	// Remove /api/threads/ prefix
	if len(path) <= len("/api/threads/") {
		http.Error(w, "Invalid thread URL", http.StatusBadRequest)
		return
	}

	pathParts := strings.Split(strings.TrimPrefix(path, "/api/threads/"), "/")
	if len(pathParts) < 2 {
		http.Error(w, "Invalid thread URL", http.StatusBadRequest)
		return
	}

	threadID := pathParts[0]
	action := pathParts[1]

	switch action {
	case "messages":
		handleThreadMessages(w, r, threadID)
	case "reply":
		handleThreadReply(w, r, threadID)
	case "metadata":
		handleThreadMetadata(w, r, threadID)
	case "parent-author":
		handleThreadParentAuthor(w, r, threadID)
	default:
		http.Error(w, "Unknown thread action", http.StatusBadRequest)
	}
}

func handleThreadMessages(w http.ResponseWriter, r *http.Request, threadID string) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}
	if !ensureChannelReadAccess(w, r, threadChannelForACL(threadID, r)) {
		return
	}

	limitStr := r.URL.Query().Get("limit")
	limit := 50
	if limitStr != "" {
		if l, err := strconv.Atoi(limitStr); err == nil && l > 0 {
			limit = l
		}
	}

	messages, err := chatHub.GetThreadMessages(threadID, limit)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	json.NewEncoder(w).Encode(messages)
}

func handleThreadReply(w http.ResponseWriter, r *http.Request, threadID string) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var req struct {
		Channel  string                 `json:"channel"`
		Content  string                 `json:"content"`
		Metadata map[string]interface{} `json:"metadata,omitempty"`
		From     *struct {
			ID   string `json:"id"`
			Name string `json:"name"`
			Type string `json:"type"`
		} `json:"from"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	sess, ok := ensureMutationAccess(w, r, req.Channel)
	if !ok {
		return
	}

	senderID, senderName, senderType := defaultHumanSender()

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
		protocol.MessageTypeChat,
		req.Channel,
		protocol.AgentInfo{
			ID:   senderID,
			Name: senderName,
			Type: senderType,
		},
		req.Content,
	)

	// Mark as thread reply
	msg.ThreadID = threadID
	msg.IsThreadReply = true

	if req.Metadata != nil {
		for k, v := range req.Metadata {
			msg.Metadata[k] = v
		}
		agent.MergeCodebaseAttachments(msg)
		agent.SanitizeInboundMessageMetadata(msg)
	}

	chatHub.AnnotateInboundUserMessage(msg, sess.Username)

	if err := chatHub.SendMessage(msg); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	json.NewEncoder(w).Encode(map[string]string{"status": "ok"})
}

func handleThreadMetadata(w http.ResponseWriter, r *http.Request, threadID string) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}
	if !ensureChannelReadAccess(w, r, threadChannelForACL(threadID, r)) {
		return
	}

	metadata, err := chatHub.GetThreadMetadata(threadID)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	json.NewEncoder(w).Encode(metadata)
}

func handleThreadParentAuthor(w http.ResponseWriter, r *http.Request, threadID string) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}
	if !ensureChannelReadAccess(w, r, threadChannelForACL(threadID, r)) {
		return
	}

	authorID := chatHub.GetThreadParentAuthor(threadID)

	// Return the author ID as JSON
	response := map[string]string{"author_id": authorID}
	json.NewEncoder(w).Encode(response)
}

// initializeModeratorAgent creates and starts the system moderator agent
