package ai

import (
	"github.com/camronwood/neural-junkie/internal/protocol"
)

// ChatRoleForHistory maps a persisted channel message to an OpenAI-style role
// for provider conversation history ("user" or "assistant").
func ChatRoleForHistory(msg protocol.Message) string {
	if protocol.IsUserLikeSender(msg.From) {
		return "user"
	}
	return "assistant"
}
