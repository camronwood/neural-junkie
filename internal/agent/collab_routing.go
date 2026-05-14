package agent

import (
	"context"

	"github.com/camronwood/neural-junkie/internal/ai"
	"github.com/camronwood/neural-junkie/internal/protocol"
)

// CollabRouting optionally overrides the AI provider for collaboration execution tasks.
type CollabRouting interface {
	EffectiveAI(ctx context.Context, base ai.AIProvider, info protocol.AgentInfo, collab CollaborationInfo, msg *protocol.Message) ai.AIProvider
}

var globalCollabRouting CollabRouting

// SetGlobalCollabRouting registers the server implementation (e.g. from cmd/server).
func SetGlobalCollabRouting(r CollabRouting) {
	globalCollabRouting = r
}

// EffectiveAIProvider returns the provider to use for this message (collab routing or base).
func (a *Agent) EffectiveAIProvider(ctx context.Context, msg *protocol.Message) ai.AIProvider {
	if a == nil {
		return nil
	}
	base := a.GetAIProvider()
	if globalCollabRouting == nil {
		return base
	}
	return globalCollabRouting.EffectiveAI(ctx, base, a.Info, a.getCollaborationContext(msg), msg)
}
