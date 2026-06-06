package hub

import (
	"fmt"
	"strings"

	"github.com/camronwood/neural-junkie/internal/collaboration"
	"github.com/camronwood/neural-junkie/internal/protocol"
)

func looksLikePlaceholderDeliverableContent(content string) bool {
	content = strings.TrimSpace(content)
	if content == "" {
		return false
	}
	lower := strings.ToLower(content)
	markers := []string{
		"[insert ",
		"[todo",
		"insert file name",
		"insert issues",
		"insert recommendations",
		"lorem ipsum",
	}
	for _, m := range markers {
		if strings.Contains(lower, m) {
			return true
		}
	}
	return false
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
