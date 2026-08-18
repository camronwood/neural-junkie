package main

import (
	"context"
	"encoding/json"
	"io"
	"log"
	"net/http"
	"strings"
	"time"

	"github.com/camronwood/neural-junkie/internal/agent"
	"github.com/camronwood/neural-junkie/internal/config"
	"github.com/camronwood/neural-junkie/internal/hub"
	"github.com/camronwood/neural-junkie/internal/protocol"
)

// handleTurnPrepare classifies a turn from a structural availability envelope and
// returns a context_request for the client to upload only the needed payloads.
func handleTurnPrepare(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}
	if !prepareDispatchEnabled(config.AppConfig()) {
		http.Error(w, "semantic prepare/dispatch disabled", http.StatusServiceUnavailable)
		return
	}
	msg, ok := decodeHumanTurnMessage(w, r)
	if !ok {
		return
	}
	ctx, cancel := context.WithTimeout(r.Context(), 90*time.Second)
	defer cancel()
	result, err := chatHub.PrepareTurn(ctx, msg)
	if err != nil {
		status := http.StatusServiceUnavailable
		if strings.Contains(err.Error(), "not eligible") {
			status = http.StatusBadRequest
		}
		http.Error(w, err.Error(), status)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(result)
}

// handleTurnDispatch finalizes a prepared turn (or falls back to /api/send behavior)
// after the client uploads requested context.
func handleTurnDispatch(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}
	msg, ok := decodeHumanTurnMessage(w, r)
	if !ok {
		return
	}
	if msg.Metadata == nil {
		msg.Metadata = make(map[string]interface{})
	}
	token, _ := msg.Metadata["prepare_token"].(string)
	token = strings.TrimSpace(token)
	if token == "" {
		// Compatibility: dispatch without prepare token behaves like send.
		if err := chatHub.SendMessage(msg); err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		maybeEmitLearningProposal(msg)
		writeSendMessageOKResponse(w)
		return
	}
	if _, ok := chatHub.PeekPreparedTurn(token); !ok {
		http.Error(w, "prepare token expired or unknown", http.StatusBadRequest)
		return
	}
	if err := chatHub.SendMessage(msg); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	maybeEmitLearningProposal(msg)
	writeSendMessageOKResponse(w)
}

func decodeHumanTurnMessage(w http.ResponseWriter, r *http.Request) (*protocol.Message, bool) {
	body, err := io.ReadAll(r.Body)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return nil, false
	}

	var fullMsg protocol.Message
	if err := json.Unmarshal(body, &fullMsg); err == nil && fullMsg.ID != "" {
		if _, ok := ensureMutationAccess(w, r, fullMsg.Channel); !ok {
			return nil, false
		}
		agent.MergeCodebaseAttachments(&fullMsg)
		agent.SanitizeInboundMessageMetadata(&fullMsg)
		return &fullMsg, true
	}

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
		return nil, false
	}
	sess, ok := ensureMutationAccess(w, r, req.Channel)
	if !ok {
		return nil, false
	}

	msgType := protocol.MessageType(req.Type)
	if msgType == "" {
		msgType = protocol.MessageTypeChat
	}
	senderID, senderName, senderType := defaultHumanSender()
	if sess != nil && strings.TrimSpace(sess.Username) != "" {
		senderName = sess.Username
		senderID = "human-" + slugifyHumanID(sess.Username)
		senderType = protocol.AgentType("human")
	}
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
		protocol.AgentInfo{ID: senderID, Name: senderName, Type: senderType},
		req.Content,
	)
	if req.ThreadID != "" {
		msg.ThreadID = req.ThreadID
		msg.IsThreadReply = req.IsThreadReply
	}
	if req.ReplyTo != "" {
		msg.ReplyTo = req.ReplyTo
	} else if req.Metadata != nil {
		if rt, ok := req.Metadata["reply_to"].(string); ok && strings.TrimSpace(rt) != "" {
			msg.ReplyTo = strings.TrimSpace(rt)
		}
	}
	if req.Metadata != nil {
		if msg.Metadata == nil {
			msg.Metadata = make(map[string]interface{})
		}
		for k, v := range req.Metadata {
			msg.Metadata[k] = v
		}
		if wsCtx, ok := msg.Metadata["workspace_context"]; ok {
			raw, _ := json.Marshal(wsCtx)
			if len(raw) > 500*1024 {
				log.Printf("Warning: workspace_context too large (%d bytes), removing open_files", len(raw))
				if ctxMap, ok := wsCtx.(map[string]interface{}); ok {
					delete(ctxMap, "open_files")
					msg.Metadata["workspace_context"] = ctxMap
				}
			}
		}
		agent.MergeCodebaseAttachments(msg)
		agent.SanitizeInboundMessageMetadata(msg)
	}
	if sess != nil {
		chatHub.AnnotateInboundUserMessage(msg, sess.Username)
	}
	return msg, true
}
