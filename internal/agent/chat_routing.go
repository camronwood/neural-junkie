package agent

import (
	"context"

	"github.com/camronwood/neural-junkie/internal/ai"
	"github.com/camronwood/neural-junkie/internal/protocol"
)

// ChatRouting optionally overrides the AI provider/model for normal chat and DM turns.
type ChatRouting interface {
	EffectiveAI(ctx context.Context, base ai.AIProvider, info protocol.AgentInfo, msg *protocol.Message) ai.AIProvider
}

var globalChatRouting ChatRouting

// SetGlobalChatRouting registers the server chat routing implementation.
func SetGlobalChatRouting(r ChatRouting) {
	globalChatRouting = r
}

// GlobalChatRouting returns the registered chat routing implementation.
func GlobalChatRouting() ChatRouting {
	return globalChatRouting
}
