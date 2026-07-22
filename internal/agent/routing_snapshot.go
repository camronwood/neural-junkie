package agent

import (
	"context"
	"strings"
	"sync"

	"github.com/camronwood/neural-junkie/internal/ai"
	"github.com/camronwood/neural-junkie/internal/protocol"
)

// RoutingSnapshot records which provider/model ran for the current turn.
type RoutingSnapshot struct {
	ProviderID                string
	ChatModel                 string
	ToolModel                 string
	Reason                    string
	Source                    string
	Domain                    string
	CostTier                  string
	KnowledgeRoute            string
	KnowledgeReason           string
	KnowledgeTargets          []string
	KnowledgeExecuted         []string
	ComposerMode              string
	ContextScope              string
	ImplSession               bool
	ClassifierIntent          string
	ClassifierToolNeed        bool
	ClassifierConfidence      float64
	ClassifierLoRATag         string
	ConversationTier          string
	ConversationReasons       []string
	ConversationEscalatedFrom string
	Attempts                  []protocol.RoutingAttempt
	FailureEvidence           []string
}

type routingSnapshotHolder struct {
	mu    sync.Mutex
	byMsg map[string]*RoutingSnapshot
}

type turnRoutingContextKey struct{}

func contextWithTurnRouting(ctx context.Context, msgID string) context.Context {
	msgID = strings.TrimSpace(msgID)
	if ctx == nil || msgID == "" {
		return ctx
	}
	return context.WithValue(ctx, turnRoutingContextKey{}, msgID)
}

func turnRoutingMsgID(ctx context.Context) string {
	if ctx == nil {
		return ""
	}
	msgID, _ := ctx.Value(turnRoutingContextKey{}).(string)
	return strings.TrimSpace(msgID)
}

func mergeRoutingSnapshot(dst *RoutingSnapshot, snap RoutingSnapshot) {
	if dst == nil {
		return
	}
	if snap.Reason != "" {
		dst.Reason = snap.Reason
	}
	if snap.Source != "" {
		dst.Source = snap.Source
	}
	if snap.Domain != "" {
		dst.Domain = snap.Domain
	}
	if snap.CostTier != "" {
		dst.CostTier = snap.CostTier
	}
	if snap.KnowledgeRoute != "" {
		dst.KnowledgeRoute = snap.KnowledgeRoute
	}
	if snap.KnowledgeReason != "" {
		dst.KnowledgeReason = snap.KnowledgeReason
	}
	if len(snap.KnowledgeTargets) > 0 {
		dst.KnowledgeTargets = append([]string(nil), snap.KnowledgeTargets...)
	}
	if len(snap.KnowledgeExecuted) > 0 {
		for _, path := range snap.KnowledgeExecuted {
			found := false
			for _, existing := range dst.KnowledgeExecuted {
				if existing == path {
					found = true
					break
				}
			}
			if !found {
				dst.KnowledgeExecuted = append(dst.KnowledgeExecuted, path)
			}
		}
	}
	if snap.ProviderID != "" {
		dst.ProviderID = snap.ProviderID
	}
	if snap.ChatModel != "" {
		dst.ChatModel = snap.ChatModel
	}
	if snap.ToolModel != "" {
		dst.ToolModel = snap.ToolModel
	}
	if snap.ComposerMode != "" {
		dst.ComposerMode = snap.ComposerMode
	}
	if snap.ContextScope != "" {
		dst.ContextScope = snap.ContextScope
	}
	if snap.ImplSession {
		dst.ImplSession = true
	}
	if snap.ClassifierIntent != "" {
		dst.ClassifierIntent = snap.ClassifierIntent
	}
	if snap.ClassifierLoRATag != "" {
		dst.ClassifierLoRATag = snap.ClassifierLoRATag
	}
	if snap.ClassifierConfidence > 0 {
		dst.ClassifierConfidence = snap.ClassifierConfidence
	}
	dst.ClassifierToolNeed = snap.ClassifierToolNeed || dst.ClassifierToolNeed
	if snap.ConversationTier != "" {
		dst.ConversationTier = snap.ConversationTier
	}
	if len(snap.ConversationReasons) > 0 {
		dst.ConversationReasons = append([]string(nil), snap.ConversationReasons...)
	}
	if snap.ConversationEscalatedFrom != "" {
		dst.ConversationEscalatedFrom = snap.ConversationEscalatedFrom
	}
	if len(snap.Attempts) > 0 {
		dst.Attempts = append([]protocol.RoutingAttempt(nil), snap.Attempts...)
	}
	if len(snap.FailureEvidence) > 0 {
		dst.FailureEvidence = append([]string(nil), snap.FailureEvidence...)
	}
}

func (a *Agent) beginTurnRouting(msgID string) {
	if a == nil {
		return
	}
	msgID = strings.TrimSpace(msgID)
	if msgID == "" {
		return
	}
	a.routingSnap.mu.Lock()
	defer a.routingSnap.mu.Unlock()
	if a.routingSnap.byMsg == nil {
		a.routingSnap.byMsg = make(map[string]*RoutingSnapshot)
	}
	a.routingSnap.byMsg[msgID] = &RoutingSnapshot{}
}

func (a *Agent) endTurnRouting(msgID string) {
	if a == nil {
		return
	}
	msgID = strings.TrimSpace(msgID)
	if msgID == "" {
		return
	}
	a.routingSnap.mu.Lock()
	defer a.routingSnap.mu.Unlock()
	delete(a.routingSnap.byMsg, msgID)
}

func (a *Agent) resetRoutingSnapshot() {
	if a == nil {
		return
	}
	a.routingSnap.mu.Lock()
	a.routingSnap.byMsg = make(map[string]*RoutingSnapshot)
	a.routingSnap.mu.Unlock()
}

func normalizeRoutingMsgID(msgID string) string {
	msgID = strings.TrimSpace(msgID)
	if msgID == "" {
		// Test helpers and non-pipeline paths share one legacy bucket.
		return "_legacy"
	}
	return msgID
}

// RecordRoutingSnapshot stores routing metadata for the current response.
// Prefer RecordRoutingSnapshotCtx when a turn context is available so concurrent
// turns cannot merge evidence into one another.
func (a *Agent) RecordRoutingSnapshot(snap RoutingSnapshot) {
	a.RecordRoutingSnapshotFor("", snap)
}

// RecordRoutingSnapshotFor stores routing metadata for a specific inbound turn.
func (a *Agent) RecordRoutingSnapshotFor(msgID string, snap RoutingSnapshot) {
	if a == nil {
		return
	}
	msgID = normalizeRoutingMsgID(msgID)
	a.routingSnap.mu.Lock()
	defer a.routingSnap.mu.Unlock()
	if a.routingSnap.byMsg == nil {
		a.routingSnap.byMsg = make(map[string]*RoutingSnapshot)
	}
	dst, ok := a.routingSnap.byMsg[msgID]
	if !ok || dst == nil {
		dst = &RoutingSnapshot{}
		a.routingSnap.byMsg[msgID] = dst
	}
	mergeRoutingSnapshot(dst, snap)
}

// RecordRoutingSnapshotCtx stores routing metadata for the turn bound to ctx.
func (a *Agent) RecordRoutingSnapshotCtx(ctx context.Context, snap RoutingSnapshot) {
	a.RecordRoutingSnapshotFor(turnRoutingMsgID(ctx), snap)
}

// RecordRoutingFromProvider captures provider id and model from an AI provider.
func (a *Agent) RecordRoutingFromProvider(provider ai.AIProvider, reason, source string) {
	a.RecordRoutingFromProviderFor("", provider, reason, source)
}

// RecordRoutingFromProviderFor captures provider id/model for a specific turn.
func (a *Agent) RecordRoutingFromProviderFor(msgID string, provider ai.AIProvider, reason, source string) {
	if a == nil || provider == nil {
		return
	}
	snap := RoutingSnapshot{Reason: reason, Source: source}
	if id := providerIDFromAI(provider); id != "" {
		snap.ProviderID = id
	}
	if m := strings.TrimSpace(provider.GetModel()); m != "" {
		snap.ChatModel = m
	}
	a.RecordRoutingSnapshotFor(msgID, snap)
}

// RecordRoutingFromProviderCtx captures provider id/model for the turn in ctx.
func (a *Agent) RecordRoutingFromProviderCtx(ctx context.Context, provider ai.AIProvider, reason, source string) {
	a.RecordRoutingFromProviderFor(turnRoutingMsgID(ctx), provider, reason, source)
}

func providerIDFromAI(p ai.AIProvider) string {
	if p == nil {
		return ""
	}
	if ider, ok := p.(interface{ ProviderID() string }); ok {
		return strings.TrimSpace(ider.ProviderID())
	}
	return ""
}

// LastRoutingSnapshot returns a copy of routing metadata for the legacy/test bucket.
func (a *Agent) LastRoutingSnapshot() RoutingSnapshot {
	return a.LastRoutingSnapshotFor("")
}

// LastRoutingSnapshotFor returns a copy of routing metadata for a turn message ID.
func (a *Agent) LastRoutingSnapshotFor(msgID string) RoutingSnapshot {
	if a == nil {
		return RoutingSnapshot{}
	}
	msgID = normalizeRoutingMsgID(msgID)
	a.routingSnap.mu.Lock()
	defer a.routingSnap.mu.Unlock()
	snap := a.routingSnap.byMsg[msgID]
	if snap == nil {
		return RoutingSnapshot{}
	}
	return *snap
}

// LastRoutingSnapshotCtx returns routing metadata for the turn bound to ctx.
func (a *Agent) LastRoutingSnapshotCtx(ctx context.Context) RoutingSnapshot {
	return a.LastRoutingSnapshotFor(turnRoutingMsgID(ctx))
}

// ApplyRoutingMetadataToResponse stamps routing_* keys on the response message
// from the legacy/test routing bucket.
func (a *Agent) ApplyRoutingMetadataToResponse(msg *protocol.Message) {
	a.ApplyRoutingMetadataToResponseFor("", msg)
}

// ApplyRoutingMetadataToResponseFor stamps routing metadata from a specific turn.
func (a *Agent) ApplyRoutingMetadataToResponseFor(msgID string, msg *protocol.Message) {
	if a == nil || msg == nil {
		return
	}
	snap := a.LastRoutingSnapshotFor(msgID)

	if snap.ChatModel == "" && snap.Reason == "" {
		if strings.TrimSpace(a.Info.AIModel) != "" {
			snap.ChatModel = a.Info.AIModel
		} else if strings.TrimSpace(a.Info.Model) != "" {
			snap.ChatModel = a.Info.Model
		}
		if snap.Reason == "" {
			snap.Reason = "default_agent_provider"
		}
		if snap.Source == "" {
			snap.Source = "rules"
		}
	}

	protocol.ApplyRoutingMeta(msg, protocol.RoutingMeta{
		ProviderID:                snap.ProviderID,
		Model:                     snap.ChatModel,
		ToolModel:                 snap.ToolModel,
		Reason:                    snap.Reason,
		Source:                    snap.Source,
		Domain:                    snap.Domain,
		CostTier:                  snap.CostTier,
		KnowledgeRoute:            snap.KnowledgeRoute,
		KnowledgeReason:           snap.KnowledgeReason,
		KnowledgeTargets:          snap.KnowledgeTargets,
		KnowledgeExecuted:         snap.KnowledgeExecuted,
		ComposerMode:              snap.ComposerMode,
		ContextScope:              snap.ContextScope,
		ImplSession:               snap.ImplSession,
		ClassifierIntent:          snap.ClassifierIntent,
		ClassifierToolNeed:        snap.ClassifierToolNeed,
		ClassifierConfidence:      snap.ClassifierConfidence,
		ClassifierLoRATag:         snap.ClassifierLoRATag,
		ConversationTier:          snap.ConversationTier,
		ConversationReasons:       snap.ConversationReasons,
		ConversationEscalatedFrom: snap.ConversationEscalatedFrom,
		Attempts:                  snap.Attempts,
		FailureEvidence:           snap.FailureEvidence,
	})
}
