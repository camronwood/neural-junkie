package agent

import (
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
	mu   sync.Mutex
	snap RoutingSnapshot
}

func (a *Agent) resetRoutingSnapshot() {
	if a == nil {
		return
	}
	a.routingSnap.mu.Lock()
	a.routingSnap.snap = RoutingSnapshot{}
	a.routingSnap.mu.Unlock()
}

// RecordRoutingSnapshot stores routing metadata for the current response.
func (a *Agent) RecordRoutingSnapshot(snap RoutingSnapshot) {
	if a == nil {
		return
	}
	a.routingSnap.mu.Lock()
	defer a.routingSnap.mu.Unlock()
	if snap.Reason != "" {
		a.routingSnap.snap.Reason = snap.Reason
	}
	if snap.Source != "" {
		a.routingSnap.snap.Source = snap.Source
	}
	if snap.Domain != "" {
		a.routingSnap.snap.Domain = snap.Domain
	}
	if snap.CostTier != "" {
		a.routingSnap.snap.CostTier = snap.CostTier
	}
	if snap.KnowledgeRoute != "" {
		a.routingSnap.snap.KnowledgeRoute = snap.KnowledgeRoute
	}
	if snap.KnowledgeReason != "" {
		a.routingSnap.snap.KnowledgeReason = snap.KnowledgeReason
	}
	if len(snap.KnowledgeTargets) > 0 {
		a.routingSnap.snap.KnowledgeTargets = append([]string(nil), snap.KnowledgeTargets...)
	}
	if len(snap.KnowledgeExecuted) > 0 {
		for _, path := range snap.KnowledgeExecuted {
			found := false
			for _, existing := range a.routingSnap.snap.KnowledgeExecuted {
				if existing == path {
					found = true
					break
				}
			}
			if !found {
				a.routingSnap.snap.KnowledgeExecuted = append(a.routingSnap.snap.KnowledgeExecuted, path)
			}
		}
	}
	if snap.ProviderID != "" {
		a.routingSnap.snap.ProviderID = snap.ProviderID
	}
	if snap.ChatModel != "" {
		a.routingSnap.snap.ChatModel = snap.ChatModel
	}
	if snap.ToolModel != "" {
		a.routingSnap.snap.ToolModel = snap.ToolModel
	}
	if snap.ComposerMode != "" {
		a.routingSnap.snap.ComposerMode = snap.ComposerMode
	}
	if snap.ContextScope != "" {
		a.routingSnap.snap.ContextScope = snap.ContextScope
	}
	if snap.ImplSession {
		a.routingSnap.snap.ImplSession = true
	}
	if snap.ClassifierIntent != "" {
		a.routingSnap.snap.ClassifierIntent = snap.ClassifierIntent
	}
	if snap.ClassifierLoRATag != "" {
		a.routingSnap.snap.ClassifierLoRATag = snap.ClassifierLoRATag
	}
	if snap.ClassifierConfidence > 0 {
		a.routingSnap.snap.ClassifierConfidence = snap.ClassifierConfidence
	}
	a.routingSnap.snap.ClassifierToolNeed = snap.ClassifierToolNeed || a.routingSnap.snap.ClassifierToolNeed
	if snap.ConversationTier != "" {
		a.routingSnap.snap.ConversationTier = snap.ConversationTier
	}
	if len(snap.ConversationReasons) > 0 {
		a.routingSnap.snap.ConversationReasons = append([]string(nil), snap.ConversationReasons...)
	}
	if snap.ConversationEscalatedFrom != "" {
		a.routingSnap.snap.ConversationEscalatedFrom = snap.ConversationEscalatedFrom
	}
	if len(snap.Attempts) > 0 {
		a.routingSnap.snap.Attempts = append([]protocol.RoutingAttempt(nil), snap.Attempts...)
	}
	if len(snap.FailureEvidence) > 0 {
		a.routingSnap.snap.FailureEvidence = append([]string(nil), snap.FailureEvidence...)
	}
}

// RecordRoutingFromProvider captures provider id and model from an AI provider.
func (a *Agent) RecordRoutingFromProvider(provider ai.AIProvider, reason, source string) {
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
	a.RecordRoutingSnapshot(snap)
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

// LastRoutingSnapshot returns a copy of routing metadata for the current turn.
func (a *Agent) LastRoutingSnapshot() RoutingSnapshot {
	if a == nil {
		return RoutingSnapshot{}
	}
	a.routingSnap.mu.Lock()
	defer a.routingSnap.mu.Unlock()
	return a.routingSnap.snap
}

// ApplyRoutingMetadataToResponse stamps routing_* keys on the response message.
func (a *Agent) ApplyRoutingMetadataToResponse(msg *protocol.Message) {
	if a == nil || msg == nil {
		return
	}
	a.routingSnap.mu.Lock()
	snap := a.routingSnap.snap
	a.routingSnap.mu.Unlock()

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
