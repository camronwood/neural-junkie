package main

import (
	"context"
	"log"

	"github.com/camronwood/neural-junkie/internal/agent"
	"github.com/camronwood/neural-junkie/internal/ai"
	implrouting "github.com/camronwood/neural-junkie/internal/implementation/routing"
	"github.com/camronwood/neural-junkie/internal/protocol"
)

type implementationRoutingRuntime struct{}

func (implementationRoutingRuntime) Plan(ctx context.Context, base ai.AIProvider, info protocol.AgentInfo, msg *protocol.Message) (agent.ImplementationRoutingPlan, ai.AIProvider) {
	plan := agent.ImplementationRoutingPlan{Reason: "default_agent_provider"}
	if appConfig == nil {
		return plan, base
	}
	cfg := appConfig.Implementation
	defaultID := ""
	for _, acfg := range appConfig.EnabledAgents() {
		if acfg.Name == info.Name {
			if p := appConfig.ProviderForAgent(acfg); p != nil {
				defaultID = p.ID
			}
			break
		}
	}
	taskText := ""
	if msg != nil {
		taskText = msg.Content
	}
	selID, toolModel, reason := implrouting.SelectProviderID(implrouting.Input{
		RoutingEnabled:      cfg.RoutingEnabled,
		LocalProviderID:     cfg.LocalProviderID,
		LocalToolModel:      cfg.LocalToolModelOrDefault(),
		FallbackProviderIDs: cfg.FallbackProviderIDs,
		Providers:           appConfig.ListProvidersSnapshot(),
		DefaultProviderID:   defaultID,
		TaskText:            taskText,
		AgentType:           string(info.Type),
	})
	plan = agent.ImplementationRoutingPlan{
		ProviderID: selID,
		ToolModel:  toolModel,
		Reason:     reason,
	}
	if selID == "" {
		return plan, base
	}
	p, err := globalProviderCache.Get(appConfig, selID)
	if err != nil {
		log.Printf("[impl-routing] %s: fallback to base: %v (reason=%s)", info.Name, err, reason)
		return plan, base
	}
	log.Printf("[impl-routing] %s: provider_id=%s tool_model=%s reason=%s", info.Name, selID, toolModel, reason)
	return plan, p
}
