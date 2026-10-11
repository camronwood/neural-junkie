package hub

import (
	"fmt"
	"log"
	"strings"

	"github.com/camronwood/neural-junkie/internal/collaboration"
	"github.com/camronwood/neural-junkie/internal/protocol"
)

// enforceExecutionMessageBudget rejects agent chat when executing-phase caps are exceeded.
// Check-only: the counter is incremented in recordExecutionMessageAccepted after the
// message is accepted into the channel transcript.
func (h *Hub) enforceExecutionMessageBudget(msg *protocol.Message) error {
	if msg == nil || h.collabManager == nil {
		return nil
	}
	if !executionBudgetEligible(msg) {
		return nil
	}
	collabID := msg.GetCollaborationID()
	if collabID == "" {
		return nil
	}

	snap, err := h.collabManager.GetCollaborationSnapshot(collabID)
	if err != nil || snap == nil || snap.Phase != collaboration.PhaseExecuting {
		return nil
	}

	overCap, err := h.collabManager.ExecutionMessageBudgetExceeded(collabID)
	if err != nil || !overCap {
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

// recordExecutionMessageAccepted increments the executing-phase counter only after
// a budget-eligible message has been accepted (persisted/broadcast).
func (h *Hub) recordExecutionMessageAccepted(msg *protocol.Message) {
	if msg == nil || h.collabManager == nil {
		return
	}
	if !executionBudgetEligible(msg) {
		return
	}
	collabID := msg.GetCollaborationID()
	if collabID == "" {
		return
	}
	snap, err := h.collabManager.GetCollaborationSnapshot(collabID)
	if err != nil || snap == nil || snap.Phase != collaboration.PhaseExecuting {
		return
	}
	if _, _, err := h.collabManager.IncrementExecutionMessageCount(collabID); err != nil {
		log.Printf("[Collaboration] IncrementExecutionMessageCount %s: %v", collabID[:8], err)
	}
}

func executionBudgetEligible(msg *protocol.Message) bool {
	if msg == nil {
		return false
	}
	if msg.IsFromSystem() || msg.From.ID == "system" {
		return false
	}
	if msg.Metadata != nil {
		if skip, ok := msg.Metadata["collab_skip_execution_budget"].(bool); ok && skip {
			return false
		}
	}
	switch msg.Type {
	case protocol.MessageTypeChat, protocol.MessageTypeAnswer, protocol.MessageTypeCollabDiscussion:
		return true
	default:
		return false
	}
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
	var snap *collaboration.Collaboration
	if collabID := strings.TrimSpace(msg.GetCollaborationID()); collabID != "" {
		if fresh, err := h.collabManager.GetCollaborationSnapshot(collabID); err == nil && fresh != nil {
			snap = fresh
		}
	}
	if snap == nil {
		snap = h.collabManager.GetByChannel(ch)
	}
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
		protocol.MessageTypeCollabTask, protocol.MessageTypeCollabRecap:
		// FileChange is intentionally excluded: cancelled/completed collabs must not
		// accept (or transcript-persist) late agent proposals after /cancel-plan.
		return nil
	}
	phase := string(snap.Phase)
	return fmt.Errorf(
		"collaboration channel is closed (%s); use slash commands or start a new /collaborate or /runbook",
		phase,
	)
}
