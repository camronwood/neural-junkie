package hub

import (
	"strings"

	"github.com/camronwood/neural-junkie/internal/protocol"
)

// isRegressionHarnessChatChannel reports scenario chat/DM channels where agents must stay advisory
// unless an implement scenario explicitly opts into implementation_session metadata.
func isRegressionHarnessChatChannel(channel string) bool {
	ch := strings.TrimSpace(channel)
	if ch == "" {
		return false
	}
	if ch == "chat-scenarios" || ch == "learning-scenarios" {
		return true
	}
	if strings.HasPrefix(ch, "dm-chatscenario-") {
		return true
	}
	return false
}

// hubChannelAllowsIDEAutoApprove gates hub auto-approve so long regression sweeps do not bleed
// file edits from implement-scenarios into chat scenario channels.
func hubChannelAllowsIDEAutoApprove(channel string, msg *protocol.Message) bool {
	ch := strings.TrimSpace(channel)
	if ch == "implement-scenarios" {
		return true
	}
	if isRegressionHarnessChatChannel(ch) {
		return false
	}
	if strings.HasSuffix(ch, "-scenarios-solo") {
		return msg != nil && msg.ImplementationSession()
	}
	return true
}
