package agent

import (
	"strings"
	"sync"

	"github.com/camronwood/neural-junkie/internal/ai"
	"github.com/camronwood/neural-junkie/internal/protocol"
)

// RoutingSnapshot records which provider/model ran for the current turn.
type RoutingSnapshot struct {
	ProviderID      string
	ChatModel       string
	ToolModel       string
	Reason          string
	Source          string
	Domain          string
	CostTier        string
	KnowledgeRoute  string
	KnowledgeReason string
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
	if snap.ProviderID != "" {
		a.routingSnap.snap.ProviderID = snap.ProviderID
	}
	if snap.ChatModel != "" {
		a.routingSnap.snap.ChatModel = snap.ChatModel
	}
	if snap.ToolModel != "" {
		a.routingSnap.snap.ToolModel = snap.ToolModel
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
		ProviderID:      snap.ProviderID,
		Model:           snap.ChatModel,
		ToolModel:       snap.ToolModel,
		Reason:          snap.Reason,
		Source:          snap.Source,
		Domain:          snap.Domain,
		CostTier:        snap.CostTier,
		KnowledgeRoute:  snap.KnowledgeRoute,
		KnowledgeReason: snap.KnowledgeReason,
	})
}
