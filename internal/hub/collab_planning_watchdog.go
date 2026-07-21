package hub

import (
	"log"
	"strings"
	"time"

	"github.com/camronwood/neural-junkie/internal/agent"
	"github.com/camronwood/neural-junkie/internal/collaboration"
	"github.com/camronwood/neural-junkie/internal/protocol"
)

const collabPlanningHandoffRedispatchAfter = 15 * time.Second

// kickPlanningDiscussionWatchdog clears handoff throttle and re-sends turn prompts for silent participants.
// Public entry is Hub.KickPlanningDiscussionWatchdog / CollabScheduler.OnPlanningKick.
func (h *Hub) kickPlanningDiscussionWatchdog(collabID string) {
	if h == nil || h.collabManager == nil || strings.TrimSpace(collabID) == "" {
		return
	}
	h.collabWatchdogMu.Lock()
	if h.collabWatchdogPlanningHandoff != nil {
		delete(h.collabWatchdogPlanningHandoff, collabID)
	}
	h.collabWatchdogMu.Unlock()
	for _, c := range h.collabManager.ListActive() {
		if c == nil || c.ID != collabID {
			continue
		}
		h.tickPlanningDiscussionWatchdog(c, time.Now())
		return
	}
}

// tickPlanningDiscussionWatchdog re-sends turn handoffs when planning participants stay silent.
func (h *Hub) tickPlanningDiscussionWatchdog(c *collaboration.Collaboration, now time.Time) {
	if h == nil || h.collabManager == nil || c == nil {
		return
	}
	if c.Phase != collaboration.PhasePlanning || c.Discussion == nil {
		return
	}
	if c.Discussion.Status != collaboration.DiscussionActive {
		return
	}
	channel := strings.TrimSpace(c.Channel)
	if channel == "" || h.ShouldDeferAgents(channel) {
		return
	}

	silent := h.collabManager.SilentPlanningParticipantIDs(c.ID)
	if len(silent) == 0 {
		return
	}

	h.collabWatchdogMu.Lock()
	if h.collabWatchdogPlanningHandoff == nil {
		h.collabWatchdogPlanningHandoff = make(map[string]time.Time)
	}
	last := h.collabWatchdogPlanningHandoff[c.ID]
	if !last.IsZero() && now.Sub(last) < collabPlanningHandoffRedispatchAfter {
		h.collabWatchdogMu.Unlock()
		return
	}
	h.collabWatchdogPlanningHandoff[c.ID] = now
	h.collabWatchdogMu.Unlock()

	// Only prompt the current turn holder when they are still silent. Picking silent[0]
	// can target the wrong agent after a generation_error (turn not advanced), which
	// causes RecordMessage to reject their reply and planning stalls.
	targetID := ""
	if turnID, err := h.collabManager.GetCurrentTurnAgent(c.ID); err == nil {
		turnID = strings.TrimSpace(turnID)
		for _, pid := range silent {
			if pid == turnID {
				targetID = turnID
				break
			}
		}
	}
	if targetID == "" {
		return
	}

	if h.sendPlanningTurnHandoff(c, targetID) {
		name := h.collabManager.ParticipantAgentName(c.ID, targetID)
		log.Printf("[Collaboration] Watchdog planning handoff for @%s (collab %s, silent=%d)",
			name, c.ID[:8], len(silent))
	}
}

// maybeKickPlanningDiscussionOnHumanMessage re-dispatches turn handoffs when the user
// steers planning mid-discussion so silent participants are not stuck behind throttle.
func (h *Hub) maybeKickPlanningDiscussionOnHumanMessage(collabID string) {
	if h == nil || h.collabManager == nil || strings.TrimSpace(collabID) == "" {
		return
	}
	for _, c := range h.collabManager.ListActive() {
		if c == nil || c.ID != collabID {
			continue
		}
		if c.Phase != collaboration.PhasePlanning || c.Discussion == nil {
			return
		}
		if c.Discussion.Status != collaboration.DiscussionActive {
			return
		}
		if len(h.collabManager.SilentPlanningParticipantIDs(c.ID)) == 0 {
			return
		}
		NewCollabScheduler(h).OnPlanningKick(c.ID)
		return
	}
}

func (h *Hub) sendPlanningTurnHandoff(c *collaboration.Collaboration, agentID string) bool {
	if h == nil || c == nil || strings.TrimSpace(agentID) == "" {
		return false
	}
	name := h.collabManager.ParticipantAgentName(c.ID, agentID)
	if name == "" {
		return false
	}

	body := collaboration.PlanningTurnHandoffBody

	msg := protocol.NewMessage(
		protocol.MessageTypeCollabDiscussion,
		c.Channel,
		protocol.AgentInfo{ID: "system", Name: "System", Type: protocol.AgentTypeGeneral},
		body,
	)
	msg.SetCollaborationID(c.ID)
	msg.SetCollaborationPhase(string(collaboration.PhasePlanning))
	msg.Mentions = []string{h.resolveCollabParticipantLiveID(c, agentID)}
	if msg.Metadata == nil {
		msg.Metadata = map[string]interface{}{}
	}
	msg.Metadata["collab_internal_event"] = true
	msg.Metadata["collab_turn_handoff"] = true
	if c.AttachWorkspaceContext {
		if ctx := c.SourceWorkspaceContext; len(ctx) > 0 {
			msg.Metadata["workspace_context"] = ctx
			msg.Metadata[agent.MetadataContextScope] = agent.ContextScopeOutline
		}
	}

	if err := h.SendMessage(msg); err != nil {
		log.Printf("[Collaboration] Watchdog planning handoff send failed (collab %s): %v", c.ID[:8], err)
		return false
	}
	return true
}

// dispatchPlanningHandoff sends a planning turn prompt for agentID (live ID remapped).
// Public entry is Hub.DispatchPlanningHandoff / CollabScheduler.OnPlanningNeedHandoff.
func (h *Hub) dispatchPlanningHandoff(collabID, agentID string) bool {
	if h == nil || h.collabManager == nil {
		return false
	}
	collabID = strings.TrimSpace(collabID)
	agentID = strings.TrimSpace(agentID)
	if collabID == "" || agentID == "" {
		return false
	}
	for _, c := range h.collabManager.ListActive() {
		if c == nil || c.ID != collabID {
			continue
		}
		if c.Phase != collaboration.PhasePlanning {
			return false
		}
		return h.sendPlanningTurnHandoff(c, agentID)
	}
	return false
}
