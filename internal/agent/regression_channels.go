package agent

import (
	"strings"

	"github.com/camronwood/neural-junkie/internal/protocol"
)

// channelAllowsImplementationSession reports whether the bounded implementation loop may run.
// Regression chat harness channels stay advisory unless the scenario explicitly sets
// implementation_session metadata (implement-scenarios always allows sessions).
func channelAllowsImplementationSession(channel string, msg *protocol.Message) bool {
	if msg != nil && msg.GetCollaborationID() != "" {
		return true
	}
	ch := strings.TrimSpace(channel)
	switch ch {
	case "implement-scenarios", "user-flow-scenarios", "parity-scenarios":
		return true
	}
	if isRegressionHarnessChatChannel(ch) {
		return msg != nil && msg.ImplementationSession()
	}
	if strings.HasSuffix(ch, "-scenarios-solo") {
		return msg != nil && msg.ImplementationSession()
	}
	return true
}

func isRegressionHarnessChatChannel(channel string) bool {
	ch := strings.TrimSpace(channel)
	if ch == "chat-scenarios" || ch == "learning-scenarios" {
		return true
	}
	return strings.HasPrefix(ch, "dm-chatscenario-")
}
