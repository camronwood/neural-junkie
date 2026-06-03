package hub

import (
	"github.com/camronwood/neural-junkie/internal/protocol"
)

func (h *Hub) isChannelDM(channel string) bool {
	if channel == "" {
		return false
	}
	h.mu.RLock()
	ch := h.channels[channel]
	h.mu.RUnlock()
	t := protocol.ChannelTypePublic
	if ch != nil {
		t = ch.Type
	}
	return inferChannelTypeForName(channel, t) == protocol.ChannelTypeDM
}

func (h *Hub) primaryAgentIDForDM(channel string) string {
	h.mu.RLock()
	defer h.mu.RUnlock()
	ch, ok := h.channels[channel]
	if !ok || ch == nil {
		return ""
	}
	for _, ag := range ch.Agents {
		if ag.ID != "" {
			return ag.ID
		}
	}
	return ""
}

// normalizeDMMentionRouting clears spurious @tokens in DMs and routes plain messages to the channel partner.
func (h *Hub) normalizeDMMentionRouting(msg *protocol.Message, parsedMentions []string, resolvedIDs []string) {
	if msg == nil || !protocol.IsUserLikeSender(msg.From) || !h.isChannelDM(msg.Channel) {
		return
	}
	partnerID := h.primaryAgentIDForDM(msg.Channel)
	if partnerID == "" {
		return
	}

	if len(resolvedIDs) == 0 && len(parsedMentions) > 0 {
		msg.Mentions = nil
	}
	if !msg.HasMentions() {
		msg.Mentions = []string{partnerID}
		return
	}
	if msg.IsMentioned(partnerID) {
		return
	}
	other := false
	for _, id := range msg.Mentions {
		if id != "" && id != "__INVALID__" && id != partnerID {
			other = true
			break
		}
	}
	if !other {
		msg.Mentions = []string{partnerID}
	}
}
