package test

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/camronwood/neural-junkie/internal/collaboration"
	"github.com/camronwood/neural-junkie/internal/hub"
	"github.com/camronwood/neural-junkie/internal/protocol"
)

// TestCollabSmokePhaseTransitions exercises /collaborate lifecycle via the same HTTP
// surface the desktop uses: POST /api/send, GET /api/collaborations, and
// POST /api/collaboration-workspace-ack. One synthetic agent discussion turn
// (RecordMessage) stands in for a bounded discussion completing — agents normally
// do this when they post collaboration_discussion messages.
func TestCollabSmokePhaseTransitions(t *testing.T) {
	srv, h := newCollabSmokeAPIServer(t)
	defer srv.Close()
	base := srv.URL

	const smokeChannel = "collab-smoke"

	// 1) Start collaboration with tight limits
	sendResp := apiSend(t, base, smokeChannel,
		"/collaborate --rounds 1 --messages 1 @RustExpert @SecurityExpert nj collab smoke probe")
	if sendResp["collaboration_id"] == "" || sendResp["collaboration_channel"] == "" {
		t.Fatalf("expected collaboration redirect in send response, got %#v", sendResp)
	}
	collabID := sendResp["collaboration_id"]
	collabCh := sendResp["collaboration_channel"]

	assertCollabPhase(t, base, collabCh, collabID, collaboration.PhasePlanning)

	// 2) Simulate one agent turn exhausting the message budget → reviewing
	simulateDiscussionBudgetExhausted(t, h, collabID, "a1", "RustExpert", protocol.AgentTypeRust)
	assertCollabPhase(t, base, collabCh, collabID, collaboration.PhaseReviewing)

	// 3) Approve plan → executing
	apiSend(t, base, collabCh, "/approve-plan "+collabID[:8])
	assertCollabPhase(t, base, collabCh, collabID, collaboration.PhaseExecuting)

	// 4) Workspace ack via API
	apiWorkspaceAck(t, base, collabID)
	snap := getCollabSnapshot(t, base, collabCh, collabID)
	if !snap.WorkspaceAcknowledged {
		t.Fatalf("expected workspace_acknowledged after POST /api/collaboration-workspace-ack")
	}

	// 5) Cancel → terminal
	apiSend(t, base, collabCh, "/cancel-plan "+collabID[:8])
	assertCollabPhase(t, base, collabCh, collabID, collaboration.PhaseCancelled)
}

func newCollabSmokeAPIServer(t *testing.T) (*httptest.Server, *hub.Hub) {
	t.Helper()
	h := hub.NewHub()
	if _, err := h.GetChannel("general"); err != nil {
		h.CreateChannel("general", "General", "")
	}
	h.CreateChannel("collab-smoke", "Collab smoke", "")
	registerTwoCollabAgents(t, h)

	mux := http.NewServeMux()
	mux.HandleFunc("/api/send", collabSmokeHandleSend(h))
	mux.HandleFunc("/api/collaborations", collabSmokeHandleCollaborations(h))
	mux.HandleFunc("/api/collaboration-workspace-ack", collabSmokeHandleWorkspaceAck(h))

	return httptest.NewServer(mux), h
}

func collabSmokeHandleSend(h *hub.Hub) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
			return
		}
		body, err := io.ReadAll(r.Body)
		if err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}

		var fullMsg protocol.Message
		if err := json.Unmarshal(body, &fullMsg); err == nil && fullMsg.ID != "" {
			if err := h.SendMessage(&fullMsg); err != nil {
				http.Error(w, err.Error(), http.StatusInternalServerError)
				return
			}
			collabSmokeWriteSendOK(w, h)
			return
		}

		var req struct {
			Channel string `json:"channel"`
			Content string `json:"content"`
			Type    string `json:"type"`
			From    *struct {
				ID   string `json:"id"`
				Name string `json:"name"`
				Type string `json:"type"`
			} `json:"from"`
		}
		if err := json.Unmarshal(body, &req); err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}

		msgType := protocol.MessageType(req.Type)
		if msgType == "" {
			msgType = protocol.MessageTypeChat
		}
		senderID, senderName, senderType := "human-user", "Human User", protocol.AgentTypeGeneral
		if req.From != nil {
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

		msg := protocol.NewMessage(msgType, req.Channel, protocol.AgentInfo{
			ID: senderID, Name: senderName, Type: senderType,
		}, req.Content)
		if err := h.SendMessage(msg); err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		collabSmokeWriteSendOK(w, h)
	}
}

func collabSmokeWriteSendOK(w http.ResponseWriter, h *hub.Hub) {
	resp := map[string]string{"status": "ok"}
	if ch, ok := h.GetCommandHandler().(*hub.CommandHandler); ok {
		if collabCh, collabID, ok2 := ch.TakeCollaborateRedirect(); ok2 {
			resp["collaboration_channel"] = collabCh
			resp["collaboration_id"] = collabID
		}
	}
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(resp)
}

func collabSmokeHandleCollaborations(h *hub.Hub) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
			return
		}
		channel := strings.TrimSpace(r.URL.Query().Get("channel"))
		includeTerminal := strings.EqualFold(strings.TrimSpace(r.URL.Query().Get("include_terminal")), "true")
		collabs := h.ListCollaborationSnapshots(channel, includeTerminal)
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(collabs)
	}
}

func collabSmokeHandleWorkspaceAck(h *hub.Hub) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
			return
		}
		var req struct {
			CollaborationID string `json:"collaboration_id"`
		}
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			http.Error(w, "Invalid JSON", http.StatusBadRequest)
			return
		}
		id := strings.TrimSpace(req.CollaborationID)
		if id == "" {
			http.Error(w, "collaboration_id required", http.StatusBadRequest)
			return
		}
		if err := h.AcknowledgeCollaborationWorkspace(id); err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
		w.WriteHeader(http.StatusNoContent)
	}
}

func apiSend(t *testing.T, base, channel, content string) map[string]string {
	t.Helper()
	payload := map[string]any{
		"channel": channel,
		"content": content,
		"type":    "question",
		"from":    map[string]string{"name": "CollabSmoke", "type": "human"},
	}
	raw, _ := json.Marshal(payload)
	resp, err := http.Post(base+"/api/send", "application/json", bytes.NewReader(raw))
	if err != nil {
		t.Fatalf("POST /api/send: %v", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		t.Fatalf("POST /api/send status %d: %s", resp.StatusCode, string(body))
	}
	var out map[string]string
	if err := json.NewDecoder(resp.Body).Decode(&out); err != nil {
		t.Fatalf("decode send response: %v", err)
	}
	return out
}

func apiWorkspaceAck(t *testing.T, base, collabID string) {
	t.Helper()
	payload := map[string]string{"collaboration_id": collabID}
	raw, _ := json.Marshal(payload)
	resp, err := http.Post(base+"/api/collaboration-workspace-ack", "application/json", bytes.NewReader(raw))
	if err != nil {
		t.Fatalf("POST workspace-ack: %v", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusNoContent {
		body, _ := io.ReadAll(resp.Body)
		t.Fatalf("POST workspace-ack status %d: %s", resp.StatusCode, string(body))
	}
}

func getCollabSnapshot(t *testing.T, base, channel, collabID string) *collaboration.Collaboration {
	t.Helper()
	url := fmt.Sprintf("%s/api/collaborations?channel=%s&include_terminal=true", base, channel)
	resp, err := http.Get(url)
	if err != nil {
		t.Fatalf("GET collaborations: %v", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		t.Fatalf("GET collaborations status %d: %s", resp.StatusCode, string(body))
	}
	var collabs []*collaboration.Collaboration
	if err := json.NewDecoder(resp.Body).Decode(&collabs); err != nil {
		t.Fatalf("decode collaborations: %v", err)
	}
	for _, c := range collabs {
		if c != nil && c.ID == collabID {
			return c
		}
	}
	t.Fatalf("collaboration %s not found on channel %s", collabID[:8], channel)
	return nil
}

func assertCollabPhase(t *testing.T, base, channel, collabID string, want collaboration.CollaborationPhase) {
	t.Helper()
	deadline := time.Now().Add(2 * time.Second)
	for time.Now().Before(deadline) {
		snap := getCollabSnapshot(t, base, channel, collabID)
		if snap.Phase == want {
			return
		}
		time.Sleep(25 * time.Millisecond)
	}
	snap := getCollabSnapshot(t, base, channel, collabID)
	t.Fatalf("phase: got %q want %q (discussion=%v)", snap.Phase, want, snap.Discussion)
}

func simulateDiscussionBudgetExhausted(t *testing.T, h *hub.Hub, collabID, agentID, agentName string, agentType protocol.AgentType) {
	t.Helper()
	cm := h.GetCollaborationManager()
	collab, err := cm.GetCollaboration(collabID)
	if err != nil {
		t.Fatalf("GetCollaboration: %v", err)
	}
	msg := protocol.NewMessage(
		protocol.MessageTypeCollabDiscussion,
		collab.Channel,
		protocol.AgentInfo{ID: agentID, Name: agentName, Type: agentType},
		"collab-smoke: synthetic turn to exhaust discussion budget",
	)
	msg.SetCollaborationID(collabID)
	if err := cm.RecordMessage(collabID, msg); err != nil {
		t.Fatalf("RecordMessage: %v", err)
	}
}
