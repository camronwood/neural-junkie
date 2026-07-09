package hub

import (
	"fmt"
	"log"
	"strings"
	"time"

	"github.com/camronwood/neural-junkie/internal/agent"
	"github.com/camronwood/neural-junkie/internal/collaboration"
	"github.com/camronwood/neural-junkie/internal/protocol"
)

const collabPlanningHandoffRedispatchAfter = 15 * time.Second

// KickPlanningDiscussionWatchdog clears handoff throttle and re-sends turn prompts for silent participants.
func (h *Hub) KickPlanningDiscussionWatchdog(collabID string) {
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
	if channel == "" || h.IsChannelHeld(channel) {
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

	targetID := silent[0]
	if turnID, err := h.collabManager.GetCurrentTurnAgent(c.ID); err == nil && strings.TrimSpace(turnID) != "" {
		for _, pid := range silent {
			if pid == turnID {
				targetID = turnID
				break
			}
		}
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
		h.collabWatchdogMu.Lock()
		if h.collabWatchdogPlanningHandoff == nil {
			h.collabWatchdogPlanningHandoff = make(map[string]time.Time)
		}
		delete(h.collabWatchdogPlanningHandoff, c.ID)
		h.collabWatchdogMu.Unlock()
		h.tickPlanningDiscussionWatchdog(c, time.Now())
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

	body := "Collaboration turn handoff: next participant, please continue the plan discussion and refine task assignments."
	if c.Discussion != nil && c.Discussion.TotalMessageCount == 0 {
		body = fmt.Sprintf(
			"@%s -- You're up first for: %s\n\nPropose a **minimal** task list (3–6 lines) with concrete deliverable paths (`- Task N: @Agent - Write collabs/<id>/file.md …`). Defer debate until tasks are drafted; use each participant's lane.",
			name, strings.TrimSpace(c.Description),
		)
	}

	msg := protocol.NewMessage(
		protocol.MessageTypeCollabDiscussion,
		c.Channel,
		protocol.AgentInfo{ID: "system", Name: "System", Type: protocol.AgentTypeGeneral},
		body,
	)
	msg.SetCollaborationID(c.ID)
	msg.SetCollaborationPhase(string(collaboration.PhasePlanning))
	msg.Mentions = []string{agentID}
	if msg.Metadata == nil {
		msg.Metadata = map[string]interface{}{}
	}
	msg.Metadata["collab_internal_event"] = true
	if ctx := c.SourceWorkspaceContext; len(ctx) > 0 {
		msg.Metadata["workspace_context"] = ctx
		msg.Metadata[agent.MetadataContextScope] = agent.ContextScopeOutline
	}

	if err := h.SendMessage(msg); err != nil {
		log.Printf("[Collaboration] Watchdog planning handoff send failed (collab %s): %v", c.ID[:8], err)
		return false
	}
	return true
}
