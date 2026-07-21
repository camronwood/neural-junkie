package agent

import (
	"context"

	"github.com/camronwood/neural-junkie/internal/ai"
	"github.com/camronwood/neural-junkie/internal/protocol"
)

// ChatRouting optionally overrides the AI provider/model for normal chat and DM turns.
type ChatRouting interface {
	EffectiveAI(ctx context.Context, base ai.AIProvider, info protocol.AgentInfo, msg *protocol.Message, trust ConversationTrustDecision) ai.AIProvider
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

// EscalateConversationProvider requests one reliable-provider reroute after a
// response quality gate fails. Callers remain responsible for generating the
// single retry; repeated calls for the same message are rejected.
func (a *Agent) EscalateConversationProvider(ctx context.Context, msg *protocol.Message) (ai.AIProvider, bool) {
	if a == nil || msg == nil || msg.Type == protocol.MessageTypeCollabTask || globalChatRouting == nil {
		return nil, false
	}
	if msg.Metadata == nil {
		msg.Metadata = make(map[string]interface{})
	}
	if metadataBool(msg.Metadata, conversationReliableReroutedKey) {
		return nil, false
	}
	msg.Metadata[conversationReliableReroutedKey] = true
	msg.Metadata[conversationQualityFailureKey] = true

	previous := a.LastRoutingSnapshot().ConversationTier
	trust := a.ClassifyConversationTrust(msg)
	if previous == "" {
		previous = string(ConversationTierStandard)
	}
	trust.EscalatedFrom = ConversationTier(previous)
	base := a.GetAIProvider()
	eff := globalChatRouting.EffectiveAI(ctx, base, a.Info, msg, trust)
	if eff == nil {
		eff = base
	}
	a.recordConversationTrust(eff, trust)
	return eff, eff != nil
}
