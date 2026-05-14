package main

import (
	"context"
	"log"
	"strings"

	"github.com/camronwood/neural-junkie/internal/agent"
	"github.com/camronwood/neural-junkie/internal/ai"
	"github.com/camronwood/neural-junkie/internal/collaboration/routing"
	"github.com/camronwood/neural-junkie/internal/protocol"
)

type collabRoutingRuntime struct{}

func (collabRoutingRuntime) EffectiveAI(ctx context.Context, base ai.AIProvider, info protocol.AgentInfo, collab agent.CollaborationInfo, msg *protocol.Message) ai.AIProvider {
	_ = ctx
	_ = collab
	if appConfig == nil || !appConfig.Collaboration.SmartRoutingEnabled {
		return base
	}
	if msg == nil || msg.Type != protocol.MessageTypeCollabTask || strings.TrimSpace(msg.GetTaskID()) == "" {
		return base
	}
	snap := appConfig.ListProvidersSnapshot()
	defaultID := defaultProviderIDForAgentName(info.Name)
	hasImages := len(protocol.ExtractUserImages(msg)) > 0 && info.SupportsVision
	selID, reason := routing.SelectProviderID(routing.Input{
		TaskText:            msg.Content,
		HasUserImages:       hasImages,
		Providers:           snap,
		DefaultProviderID:   defaultID,
	})
	p, err := globalProviderCache.Get(appConfig, selID)
	if err != nil {
		log.Printf("[collab-routing] %s: fallback to base provider: %v (reason=%s)", info.Name, err, reason)
		return base
	}
	log.Printf("[collab-routing] %s: provider_id=%s reason=%s", info.Name, selID, reason)
	return p
}

func defaultProviderIDForAgentName(name string) string {
	if appConfig == nil {
		return ""
	}
	for _, a := range appConfig.Agents {
		if strings.EqualFold(strings.TrimSpace(a.Name), strings.TrimSpace(name)) {
			if strings.TrimSpace(a.ProviderID) != "" {
				return a.ProviderID
			}
			return appConfig.AI.DefaultProviderID
		}
	}
	return appConfig.AI.DefaultProviderID
}
