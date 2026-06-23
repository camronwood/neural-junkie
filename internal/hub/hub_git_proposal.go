package hub

import (
	"encoding/json"
	"log"

	"github.com/camronwood/neural-junkie/internal/hub/gitchange"
	"github.com/camronwood/neural-junkie/internal/protocol"
)

func (h *Hub) registerGitChangeProposal(msg *protocol.Message, raw interface{}) {
	if h == nil || h.gitChangeManager == nil || msg == nil {
		return
	}
	b, err := json.Marshal(raw)
	if err != nil {
		return
	}
	var payload struct {
		ID          string   `json:"id"`
		Operation   string   `json:"operation"`
		Message     string   `json:"message"`
		Paths       []string `json:"paths"`
		WorkspaceID string   `json:"workspace_id"`
	}
	if err := json.Unmarshal(b, &payload); err != nil {
		log.Printf("[hub] git proposal parse: %v", err)
		return
	}
	_, err = h.gitChangeManager.Propose(gitchange.Proposal{
		ID:          payload.ID,
		Operation:   gitchange.Operation(payload.Operation),
		Message:     payload.Message,
		Paths:       payload.Paths,
		WorkspaceID: payload.WorkspaceID,
		Agent:       msg.From,
		Channel:     msg.Channel,
	})
	if err != nil {
		log.Printf("[hub] git proposal register: %v", err)
	}
}
