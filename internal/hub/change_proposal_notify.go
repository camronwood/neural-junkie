package hub

import (
	"log"
	"strings"

	"github.com/camronwood/neural-junkie/internal/protocol"
)

// UpdateChangeProposalStatus updates the original proposal message in place and
// rebroadcasts that same message ID. This keeps one durable card in history.
func (h *Hub) UpdateChangeProposalStatus(
	channel string,
	proposalID string,
	status protocol.ChangeProposalStatus,
	reason string,
	errText string,
) {
	if h == nil || strings.TrimSpace(proposalID) == "" {
		return
	}
	h.mu.Lock()
	defer h.mu.Unlock()

	channels := []string{strings.TrimSpace(channel)}
	if channels[0] == "" {
		channels = channels[:0]
		for name := range h.messages {
			channels = append(channels, name)
		}
	}
	for _, name := range channels {
		messages := h.messages[name]
		for i := len(messages) - 1; i >= 0; i-- {
			message := messages[i]
			if message == nil || message.Metadata == nil {
				continue
			}
			card, ok := protocol.ParseChangeProposalCard(message.Metadata[protocol.MetaChangeProposal])
			if !ok || card.ID != proposalID {
				continue
			}
			card.Status = status
			card.Reason = strings.TrimSpace(reason)
			card.Error = strings.TrimSpace(errText)
			updated := *message
			updated.Metadata = make(map[string]interface{}, len(message.Metadata))
			for key, value := range message.Metadata {
				updated.Metadata[key] = value
			}
			updated.Metadata[protocol.MetaChangeProposal] = card
			messages[i] = &updated
			h.messages[name] = messages
			h.broadcast(name, &updated)
			if h.persistentStore != nil {
				snap, err := protocol.CloneMessage(&updated)
				if err != nil {
					log.Printf("[hub] clone change proposal for persistence: %v", err)
				} else if snap != nil {
					go h.persistMessage(snap)
				}
			}
			return
		}
	}
}
