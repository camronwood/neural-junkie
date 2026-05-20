package hub

import (
	"fmt"
	"log"
	"strings"

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
			collabID[:8], snap.MaxExecutionMessagesLimit()),
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

// rejectClosedCollaborationChannel blocks chat on completed/cancelled collaboration channels.
// Slash commands and system/internal messages are still allowed.
func (h *Hub) rejectClosedCollaborationChannel(msg *protocol.Message) error {
	if msg == nil || h.collabManager == nil {
		return nil
	}
	ch := strings.TrimSpace(msg.Channel)
	if ch == "" {
		return nil
	}
	snap := h.collabManager.GetByChannel(ch)
	if snap == nil {
		return nil
	}
	if snap.Phase != collaboration.PhaseCompleted && snap.Phase != collaboration.PhaseCancelled {
		return nil
	}
	content := strings.TrimSpace(msg.Content)
	if len(content) > 0 && content[0] == '/' {
		return nil
	}
	if msg.IsFromSystem() || msg.From.ID == "system" {
		return nil
	}
	if msg.Metadata != nil {
		if internal, ok := msg.Metadata["collab_internal_event"].(bool); ok && internal {
			return nil
		}
	}
	switch msg.Type {
	case protocol.MessageTypeSystemInfo, protocol.MessageTypeCollabStatus, protocol.MessageTypeCollabPlan,
		protocol.MessageTypeCollabTask, protocol.MessageTypeCollabRecap, protocol.MessageTypeFileChange:
		return nil
	}
	phase := string(snap.Phase)
	return fmt.Errorf(
		"collaboration channel is closed (%s); use slash commands or start a new /collaborate or /runbook",
		phase,
	)
}
