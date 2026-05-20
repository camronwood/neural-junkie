package hub

import (
	"context"
	"fmt"
	"log"
	"strings"
	"sync"
	"time"

	"github.com/camronwood/neural-junkie/internal/ai"
	"github.com/camronwood/neural-junkie/internal/collaboration"
	"github.com/camronwood/neural-junkie/internal/protocol"
)

const recapTimeout = 90 * time.Second

var recapTimeoutMu sync.Mutex
var recapTimeoutCancel = map[string]func(){}

func (h *Hub) wireCollaborationRecaps() {
	if h.collabManager == nil {
		return
	}
	h.collabManager.SetOnEnterReviewing(h.onCollaborationEnterReviewing)
}

func (h *Hub) onCollaborationEnterReviewing(collabID string) {
	if h.collabManager == nil {
		return
	}
	snap, err := h.collabManager.GetCollaborationSnapshot(collabID)
	if err != nil || snap == nil {
		return
	}
	if snap.PlanningRecapStatus == collaboration.RecapStatusComplete ||
		snap.PlanningRecapStatus == collaboration.RecapStatusPending {
		return
	}
	h.dispatchCollaborationRecap(snap, collaboration.RecapKindPreApproval)
}

// dispatchCollaborationRecap prompts the selected facilitator agent to post a user-facing recap.
func (h *Hub) dispatchCollaborationRecap(snap *collaboration.Collaboration, kind collaboration.RecapKind) {
	if snap == nil || h.collabManager == nil {
		return
	}
	agentID := collaboration.SelectRecapFacilitator(snap, kind)
	if agentID == "" {
		log.Printf("[CollaborationRecap] No facilitator for %s recap on %s", kind, snap.ID[:8])
		h.handleRecapDispatchFailure(snap.ID, kind)
		return
	}

	agentName := collaboration.FacilitatorDisplayName(snap, agentID)
	agentName = strings.TrimPrefix(agentName, "@")

	switch kind {
	case collaboration.RecapKindPreApproval:
		if err := h.collabManager.MarkPlanningRecapDispatched(snap.ID, agentID); err != nil {
			log.Printf("[CollaborationRecap] %v", err)
			return
		}
	case collaboration.RecapKindFinal:
		if err := h.collabManager.MarkSessionRecapDispatched(snap.ID, agentID); err != nil {
			log.Printf("[CollaborationRecap] %v", err)
			return
		}
	}

	ctx := collaboration.BuildRecapContext(snap, kind)
	prompt := buildRecapPrompt(kind, agentName, ctx)

	recapMsg := protocol.NewMessage(
		protocol.MessageTypeCollabRecap,
		snap.Channel,
		protocol.AgentInfo{ID: "system", Name: "System", Type: protocol.AgentTypeGeneral},
		prompt,
	)
	recapMsg.SetCollaborationID(snap.ID)
	recapMsg.SetCollaborationPhase(string(snap.Phase))
	recapMsg.Mentions = []string{agentID}
	if recapMsg.Metadata == nil {
		recapMsg.Metadata = map[string]interface{}{}
	}
	recapMsg.Metadata["recap_kind"] = string(kind)
	recapMsg.Metadata["recap_assignee"] = agentID
	recapMsg.Metadata["recap_context"] = ctx

	if err := h.SendMessage(recapMsg); err != nil {
		log.Printf("[CollaborationRecap] Failed to dispatch %s recap for %s: %v", kind, snap.ID[:8], err)
		h.handleRecapDispatchFailure(snap.ID, kind)
		return
	}

	log.Printf("[CollaborationRecap] Dispatched %s recap to %s for collaboration %s", kind, agentName, snap.ID[:8])
	h.scheduleRecapTimeout(snap.ID, kind)
}

func buildRecapPrompt(kind collaboration.RecapKind, agentName, ctx string) string {
	var b strings.Builder
	if agentName == "" {
		agentName = "facilitator"
	}
	b.WriteString(fmt.Sprintf("@%s — deliver a **session recap directly to the user** (not to other agents).\n\n", agentName))
	switch kind {
	case collaboration.RecapKindPreApproval:
		b.WriteString("Planning is complete. Summarize what the team discussed, decided, and proposed.\n")
		b.WriteString("Include the plan and tasks clearly. If this was research-only, emphasize findings and recommendations.\n")
		b.WriteString("End with what the user should do next (review the plan, then `/approve-plan` when ready).\n")
	default:
		b.WriteString("Execution is complete (or the user closed the session). Summarize what was accomplished during this collaboration.\n")
		b.WriteString("Cover task outcomes, research findings, files changed (if any), and open questions.\n")
		b.WriteString("If no code was written, say so explicitly and focus on research and decisions.\n")
		b.WriteString("End with concrete next steps for the user.\n")
	}
	b.WriteString("\nDo **not** emit TASK_STATUS lines or new plan blocks. Do not @mention other agents unless quoting them.\n\n")
	b.WriteString("---\n\n")
	b.WriteString(ctx)
	return b.String()
}

func (h *Hub) scheduleRecapTimeout(collabID string, kind collaboration.RecapKind) {
	key := collabID + ":" + string(kind)
	recapTimeoutMu.Lock()
	if cancel, ok := recapTimeoutCancel[key]; ok {
		cancel()
	}
	ctx, cancel := context.WithCancel(context.Background())
	recapTimeoutCancel[key] = cancel
	recapTimeoutMu.Unlock()

	go func() {
		timer := time.NewTimer(recapTimeout)
		defer timer.Stop()
		select {
		case <-ctx.Done():
			return
		case <-timer.C:
			h.onRecapTimeout(collabID, kind)
		}
	}()
}

func (h *Hub) clearRecapTimeout(collabID string, kind collaboration.RecapKind) {
	key := collabID + ":" + string(kind)
	recapTimeoutMu.Lock()
	if cancel, ok := recapTimeoutCancel[key]; ok {
		cancel()
		delete(recapTimeoutCancel, key)
	}
	recapTimeoutMu.Unlock()
}

func (h *Hub) onRecapTimeout(collabID string, kind collaboration.RecapKind) {
	if h.collabManager == nil {
		return
	}
	snap, err := h.collabManager.GetCollaborationSnapshot(collabID)
	if err != nil || snap == nil {
		return
	}

	var status string
	switch kind {
	case collaboration.RecapKindPreApproval:
		status = snap.PlanningRecapStatus
	case collaboration.RecapKindFinal:
		status = snap.SessionRecapStatus
	default:
		return
	}
	if status != collaboration.RecapStatusPending {
		return
	}

	log.Printf("[CollaborationRecap] Timeout waiting for %s recap on %s — using fallback", kind, collabID[:8])
	fallback := h.generateRecapFallback(snap, kind)
	agentID := ""
	switch kind {
	case collaboration.RecapKindPreApproval:
		agentID = snap.PlanningRecapAgentID
		if fallback != "" {
			_ = h.collabManager.CompletePlanningRecap(collabID, agentID, fallback)
		} else {
			h.collabManager.FailPlanningRecap(collabID)
		}
		h.broadcastPlanningRecapReady(snap, fallback)
	case collaboration.RecapKindFinal:
		agentID = snap.SessionRecapAgentID
		if fallback != "" {
			_ = h.collabManager.CompleteSessionRecap(collabID, agentID, fallback)
		} else {
			h.collabManager.FailSessionRecap(collabID)
		}
		h.maybeFinalizeAfterRecap(collabID)
	}
}

func (h *Hub) handleRecapDispatchFailure(collabID string, kind collaboration.RecapKind) {
	switch kind {
	case collaboration.RecapKindPreApproval:
		h.collabManager.FailPlanningRecap(collabID)
	case collaboration.RecapKindFinal:
		h.collabManager.FailSessionRecap(collabID)
		h.maybeFinalizeAfterRecap(collabID)
	}
}

func (h *Hub) generateRecapFallback(snap *collaboration.Collaboration, kind collaboration.RecapKind) string {
	if snap == nil {
		return ""
	}
	ctx := collaboration.BuildRecapContext(snap, kind)
	prompt := fmt.Sprintf("Write a concise user-facing collaboration session recap in markdown bullets.\n\n%s", ctx)

	provider := h.recapAIProvider()
	if provider == nil {
		return deterministicRecapFallback(snap, kind)
	}
	out, err := provider.GenerateResponse(context.Background(), prompt, nil)
	if err != nil || strings.TrimSpace(out) == "" {
		return deterministicRecapFallback(snap, kind)
	}
	return strings.TrimSpace(out)
}

func (h *Hub) recapAIProvider() ai.AIProvider {
	if h.commandHandler != nil && h.commandHandler.aiProvider != nil {
		return h.commandHandler.aiProvider
	}
	return nil
}

func deterministicRecapFallback(snap *collaboration.Collaboration, kind collaboration.RecapKind) string {
	var b strings.Builder
	if kind == collaboration.RecapKindPreApproval {
		b.WriteString("### Planning summary\n\n")
	} else {
		b.WriteString("### Session summary\n\n")
	}
	b.WriteString(fmt.Sprintf("- **Goal:** %s\n", snap.Description))
	if snap.Plan != nil && strings.TrimSpace(snap.Plan.Content) != "" {
		b.WriteString("- **Plan:** see collaboration plan artifact in the panel.\n")
	}
	for _, t := range snap.Tasks {
		b.WriteString(fmt.Sprintf("- **Task:** %s (%s)\n", t.Title, t.Status))
	}
	if kind == collaboration.RecapKindPreApproval {
		b.WriteString("\n_Use `/approve-plan` when you are ready to execute._\n")
	}
	return b.String()
}

func (h *Hub) maybeProcessRecapReply(msg *protocol.Message) bool {
	if msg == nil || h.collabManager == nil || msg.IsFromSystem() {
		return false
	}
	collabID := msg.GetCollaborationID()
	if collabID == "" {
		return false
	}

	// Only treat as recap reply if this agent was the pending recap assignee.
	snap, err := h.collabManager.GetCollaborationSnapshot(collabID)
	if err != nil || snap == nil {
		return false
	}

	kind, assignee := pendingRecapAssignee(snap, msg.From.ID)
	if kind == "" {
		return false
	}

	h.clearRecapTimeout(collabID, kind)
	text := strings.TrimSpace(msg.Content)
	if text == "" {
		return false
	}

	switch kind {
	case collaboration.RecapKindPreApproval:
		if err := h.collabManager.CompletePlanningRecap(collabID, assignee, text); err != nil {
			log.Printf("[CollaborationRecap] CompletePlanningRecap: %v", err)
			return false
		}
		h.broadcastPlanningRecapReady(snap, text)
		return true
	case collaboration.RecapKindFinal:
		if err := h.collabManager.CompleteSessionRecap(collabID, assignee, text); err != nil {
			log.Printf("[CollaborationRecap] CompleteSessionRecap: %v", err)
			return false
		}
		h.maybeFinalizeAfterRecap(collabID)
		return true
	}
	return false
}

func pendingRecapAssignee(snap *collaboration.Collaboration, fromAgentID string) (collaboration.RecapKind, string) {
	if snap == nil || fromAgentID == "" {
		return "", ""
	}
	if snap.PlanningRecapStatus == collaboration.RecapStatusPending && snap.PlanningRecapAgentID == fromAgentID {
		return collaboration.RecapKindPreApproval, fromAgentID
	}
	if snap.SessionRecapStatus == collaboration.RecapStatusPending && snap.SessionRecapAgentID == fromAgentID {
		return collaboration.RecapKindFinal, fromAgentID
	}
	return "", ""
}

func (h *Hub) broadcastPlanningRecapReady(snap *collaboration.Collaboration, recapText string) {
	if snap == nil {
		return
	}
	ch := snap.Channel
	if ch == "" {
		ch = "general"
	}
	body := fmt.Sprintf("📋 **Session summary ready** (`%s`) — you can review the recap above and run `/approve-plan %s` when ready.",
		snap.ID[:8], snap.ID[:8])
	if snap.PlanningRecapStatus == collaboration.RecapStatusFailed {
		body = fmt.Sprintf("⚠️ **Session summary unavailable** (`%s`) — you may still `/approve-plan %s`.", snap.ID[:8], snap.ID[:8])
	}
	_ = recapText

	statusMsg := protocol.NewMessage(
		protocol.MessageTypeCollabStatus,
		ch,
		protocol.AgentInfo{ID: "system", Name: "System", Type: protocol.AgentTypeGeneral},
		body,
	)
	statusMsg.SetCollaborationID(snap.ID)
	statusMsg.SetCollaborationPhase(string(collaboration.PhaseReviewing))
	if statusMsg.Metadata == nil {
		statusMsg.Metadata = map[string]interface{}{}
	}
	statusMsg.Metadata["collab_internal_event"] = true
	h.attachCollaborationData(statusMsg)
	_ = h.SendMessage(statusMsg)
}

func (h *Hub) requestFinalRecapAndFinalize(collabID, channel, reason string, opts collaboration.FinalizeOptions) {
	if h.collabManager == nil {
		return
	}
	snap, err := h.collabManager.GetCollaborationSnapshot(collabID)
	if err != nil || snap == nil {
		return
	}
	if snap.Phase != collaboration.PhaseExecuting {
		h.finalizeAndBroadcastCollaboration(collabID, channel, reason, opts)
		return
	}
	if snap.SessionRecapStatus == collaboration.RecapStatusComplete {
		h.finalizeAndBroadcastCollaboration(collabID, channel, reason, opts)
		return
	}
	if snap.SessionRecapStatus == collaboration.RecapStatusPending {
		return
	}
	if err := h.collabManager.SetAwaitingFinalize(collabID, channel, reason, opts); err != nil {
		log.Printf("[CollaborationRecap] SetAwaitingFinalize: %v", err)
	}
	h.dispatchCollaborationRecap(snap, collaboration.RecapKindFinal)
}

func (h *Hub) maybeFinalizeAfterRecap(collabID string) {
	if h.collabManager == nil {
		return
	}
	channel, reason, opts, awaiting := h.collabManager.TakeAwaitingFinalize(collabID)
	if !awaiting {
		return
	}
	if channel == "" {
		if snap, _ := h.collabManager.GetCollaborationSnapshot(collabID); snap != nil {
			channel = snap.Channel
		}
	}
	if reason == "" {
		reason = "All tasks are done."
	}
	h.finalizeAndBroadcastCollaboration(collabID, channel, reason, opts)
}
