package hub

import (
	"strings"

	"github.com/camronwood/neural-junkie/internal/collaboration"
	"github.com/camronwood/neural-junkie/internal/protocol"
)

func isCollabTurnHandoffContent(content string) bool {
	return strings.Contains(content, "Collaboration turn handoff") ||
		strings.Contains(content, "You're up first")
}

// resolveCollabParticipantLiveID maps a collaboration participant ID to the
// in-process runtime agent ID (hub restart / re-register can change IDs).
func (h *Hub) resolveCollabParticipantLiveID(c *collaboration.Collaboration, participantID string) string {
	participantID = strings.TrimSpace(participantID)
	if h == nil || participantID == "" {
		return participantID
	}
	if _, err := h.GetAgent(participantID); err == nil {
		return participantID
	}
	if c != nil {
		for _, a := range c.Agents {
			if a.AgentID != participantID {
				continue
			}
			if info := h.FindLiveAgentByDisplayName(a.AgentName, a.AgentType); info != nil {
				return info.ID
			}
			break
		}
	}
	return participantID
}

// normalizeCollabTurnHandoffMentions rewrites Mentions on system turn prompts so
// shouldRespond sees the live runtime agent ID (not a stale collaboration ID).
func (h *Hub) normalizeCollabTurnHandoffMentions(msg *protocol.Message) {
	if h == nil || msg == nil {
		return
	}
	if msg.Type != protocol.MessageTypeCollabDiscussion || !msg.IsFromSystem() {
		return
	}
	if !isCollabTurnHandoffContent(msg.Content) {
		return
	}

	var collab *collaboration.Collaboration
	if cid := msg.GetCollaborationID(); cid != "" && h.collabManager != nil {
		collab, _ = h.collabManager.GetCollaboration(cid)
	}

	seen := make(map[string]struct{})
	out := make([]string, 0, len(msg.Mentions)+1)
	add := func(id string) {
		id = strings.TrimSpace(id)
		if id == "" {
			return
		}
		id = h.resolveCollabParticipantLiveID(collab, id)
		if _, ok := seen[id]; ok {
			return
		}
		seen[id] = struct{}{}
		out = append(out, id)
	}
	for _, id := range msg.Mentions {
		add(id)
	}
	// First-turn prompts also contain the collaboration goal, which may mention
	// every participant. Only recover a target from content when the explicit
	// Mentions field did not resolve to a live agent; otherwise parsing the full
	// body wakes every participant at once and defeats round-robin planning.
	hasLiveTarget := false
	for _, id := range out {
		if _, err := h.GetAgent(id); err == nil {
			hasLiveTarget = true
			break
		}
	}
	if !hasLiveTarget {
		leading := strings.TrimSpace(msg.Content)
		if idx := strings.Index(leading, "--"); idx >= 0 {
			leading = leading[:idx]
		}
		for _, name := range protocol.ParseMentions(leading) {
			name = strings.TrimSpace(name)
			if name == "" {
				continue
			}
			if info := h.FindLiveAgentByDisplayName(name, ""); info != nil {
				add(info.ID)
			}
		}
	}
	if len(out) > 0 {
		var live []string
		for _, id := range out {
			if _, err := h.GetAgent(id); err == nil {
				live = append(live, id)
			}
		}
		if len(live) > 0 {
			out = live
		}
		msg.Mentions = out
	}
}
