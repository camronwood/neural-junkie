package hub

import (
	"fmt"
	"strings"

	"github.com/camronwood/neural-junkie/internal/collaboration"
	"github.com/camronwood/neural-junkie/internal/protocol"
)

func looksLikePlaceholderDeliverableContent(content string) bool {
	return collaboration.LooksLikePlaceholderContent(content)
}

func (h *Hub) rejectFileChangeOnClosedCollab(msg *protocol.Message) error {
	if h == nil || msg == nil || h.collabManager == nil {
		return nil
	}
	collabID := strings.TrimSpace(msg.GetCollaborationID())
	if collabID == "" {
		return nil
	}
	snap, err := h.collabManager.GetCollaborationSnapshot(collabID)
	if err != nil || snap == nil {
		return nil
	}
	switch snap.Phase {
	case collaboration.PhaseCancelled, collaboration.PhaseCompleted:
		return fmt.Errorf("collaboration %s is %s", collabID[:8], snap.Phase)
	default:
		return nil
	}
}
