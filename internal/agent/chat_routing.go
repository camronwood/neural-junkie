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

// EscalateConversationProvider requests the next unique provider/model after a
// deterministic response quality-gate failure. The router owns the local-first
// ladder; repeated tiers are rejected here as a final safety net.
func (a *Agent) EscalateConversationProvider(ctx context.Context, msg *protocol.Message) (ai.AIProvider, bool) {
	if a == nil || msg == nil || msg.Type == protocol.MessageTypeCollabTask || globalChatRouting == nil {
		return nil, false
	}
	if msg.Metadata == nil {
		msg.Metadata = make(map[string]interface{})
	}
	msg.Metadata[conversationQualityFailureKey] = true
	markLastRoutingAttemptFailed(msg, ConversationReasonQualityGateFailure)

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
	attempts := protocol.ExtractRoutingMeta(msg).Attempts
	providerID := providerIDFromAI(eff)
	if providerID == "" {
		providerID = eff.GetModel()
	}
	if !routingAttemptContains(attempts, providerID, eff.GetModel()) {
		attempts = append(attempts, protocol.RoutingAttempt{
			ProviderID: providerID,
			Model:      eff.GetModel(),
			Tier:       string(trust.Tier),
			Reason:     ConversationReasonQualityGateFailure,
		})
		msg.Metadata[protocol.MetadataRoutingAttempts] = attempts
	} else if len(attempts) > 0 && attempts[len(attempts)-1].FailureReason != "" {
		a.recordRoutingEvidenceFromMessage(msg)
		return nil, false
	}
	a.recordConversationTrust(eff, trust)
	a.recordRoutingEvidenceFromMessage(msg)
	return eff, eff != nil
}

func (a *Agent) recordRoutingEvidenceFromMessage(msg *protocol.Message) {
	meta := protocol.ExtractRoutingMeta(msg)
	a.RecordRoutingSnapshot(RoutingSnapshot{
		Attempts:        meta.Attempts,
		FailureEvidence: meta.FailureEvidence,
	})
}

func markLastRoutingAttemptFailed(msg *protocol.Message, reason string) {
	if msg == nil || msg.Metadata == nil {
		return
	}
	meta := protocol.ExtractRoutingMeta(msg)
	if len(meta.Attempts) > 0 {
		last := len(meta.Attempts) - 1
		meta.Attempts[last].FailureReason = reason
		msg.Metadata[protocol.MetadataRoutingAttempts] = meta.Attempts
	}
	meta.FailureEvidence = appendUniqueString(meta.FailureEvidence, reason)
	msg.Metadata[protocol.MetadataRoutingFailureEvidence] = meta.FailureEvidence
}

func routingAttemptContains(attempts []protocol.RoutingAttempt, providerID, model string) bool {
	for _, attempt := range attempts {
		if attempt.ProviderID == providerID && attempt.Model == model {
			return true
		}
	}
	return false
}
