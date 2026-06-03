package agent

import (
	"github.com/camronwood/neural-junkie/internal/protocol"
)

// DMHumanMessageShouldRespond reports whether an agent should answer a human in a DM.
// In 1:1 DMs the channel partner responds by default; explicit @mentions of another agent opt out.
func DMHumanMessageShouldRespond(msg *protocol.Message, agentID string) bool {
	if msg == nil {
		return false
	}
	if !msg.HasMentions() {
		return true
	}
	if msg.IsMentioned(agentID) {
		return true
	}
	for _, id := range msg.Mentions {
		if id == "" || id == "__INVALID__" {
			continue
		}
		if id != agentID {
			return false
		}
	}
	// Spurious or unresolved @tokens — still talk to the DM partner.
	return true
}

func (a *Agent) isDMChannel(channel string) bool {
	return a.effectiveChannelType(channel) == protocol.ChannelTypeDM
}
