package hub

import (
	"fmt"
	"log"

	"github.com/camronwood/neural-junkie/internal/collaboration"
	"github.com/camronwood/neural-junkie/internal/protocol"
)

// enforceExecutionMessageBudget rejects agent chat when executing-phase caps are exceeded.
func (h *Hub) enforceExecutionMessageBudget(msg *protocol.Message) error {
	if msg == nil || h.collabManager == nil {
		return nil
	}
	collabID := msg.GetCollaborationID()
	if collabID == "" {
		return nil
	}
	if msg.IsFromSystem() || msg.From.ID == "system" {
		return nil
	}
	if msg.Metadata != nil {
		if skip, ok := msg.Metadata["collab_skip_execution_budget"].(bool); ok && skip {
			return nil
		}
	}
	switch msg.Type {
	case protocol.MessageTypeChat, protocol.MessageTypeAnswer, protocol.MessageTypeCollabDiscussion:
	default:
		return nil
	}

	snap, err := h.collabManager.GetCollaborationSnapshot(collabID)
	if err != nil || snap == nil || snap.Phase != collaboration.PhaseExecuting {
		return nil
	}

	_, overCap, err := h.collabManager.IncrementExecutionMessageCount(collabID)
	if err != nil {
		return nil
	}
	if !overCap {
		return nil
	}

	warn := protocol.NewMessage(
		protocol.MessageTypeCollabStatus,
		msg.Channel,
		protocol.AgentInfo{ID: "system", Name: "System", Type: protocol.AgentTypeGeneral},
		fmt.Sprintf("⛔ **Execution message limit reached** for collaboration `%s` (%d messages). Further agent posts in this session are blocked.",
			collabID[:8], collaboration.MaxExecutionMessages),
	)
	warn.SetCollaborationID(collabID)
	warn.SetCollaborationPhase(string(collaboration.PhaseExecuting))
	if warn.Metadata == nil {
		warn.Metadata = make(map[string]interface{})
	}
	warn.Metadata["collab_internal_event"] = true
	warn.Metadata["collab_skip_execution_budget"] = true
	if sendErr := h.SendMessage(warn); sendErr != nil {
		log.Printf("[Collaboration] execution cap notice failed: %v", sendErr)
	}
	return fmt.Errorf("execution message budget exceeded for collaboration %s", collabID[:8])
}
