package hub

import (
	"testing"

	"github.com/camronwood/neural-junkie/internal/protocol"
)

func TestUpdateChangeProposalStatusReusesDurableMessage(t *testing.T) {
	for _, kind := range []protocol.ChangeProposalKind{
		protocol.ChangeProposalKindFile,
		protocol.ChangeProposalKindGit,
	} {
		t.Run(string(kind), func(t *testing.T) {
			h := NewHub()
			msg := protocol.NewMessage(
				protocol.MessageTypeChat,
				"general",
				protocol.AgentInfo{ID: "agent", Name: "Agent"},
				"",
			)
			msg.Metadata[protocol.MetaChangeProposal] = protocol.ChangeProposalCard{
				Version:   1,
				Kind:      kind,
				ID:        "proposal-1",
				Status:    protocol.ChangeProposalStatusPending,
				Operation: "edit",
			}
			h.mu.Lock()
			h.appendChannelMessageLocked("general", msg)
			h.mu.Unlock()

			h.UpdateChangeProposalStatus(
				"general",
				"proposal-1",
				protocol.ChangeProposalStatusRejected,
				"not now",
				"",
			)

			messages := h.messages["general"]
			if len(messages) != 1 {
				t.Fatalf("expected one durable row, got %d", len(messages))
			}
			if messages[0].ID != msg.ID {
				t.Fatalf("message id changed: got %q want %q", messages[0].ID, msg.ID)
			}
			card, ok := protocol.ParseChangeProposalCard(
				messages[0].Metadata[protocol.MetaChangeProposal],
			)
			if !ok {
				t.Fatal("updated proposal card was not parseable")
			}
			if card.Status != protocol.ChangeProposalStatusRejected || card.Reason != "not now" {
				t.Fatalf("unexpected card state: %#v", card)
			}
		})
	}
}
