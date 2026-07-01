package main

import (
	"context"
	"log"
	"strings"

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
	hints := agent.ImplementationRoutingHintsFromContext(ctx)
	selID, toolModel, reason := implrouting.SelectProviderID(implrouting.Input{
		RoutingEnabled:                cfg.RoutingEnabled,
		ModelCapabilityRoutingEnabled: appConfig.Routing.ModelCapabilityRoutingEnabled && capabilityProfilesLoaded(),
		LocalProviderID:               cfg.LocalProviderID,
		LocalToolModel:                cfg.LocalToolModelOrDefault(),
		ReliableToolModel:             cfg.ReliableToolModelOrDefault(),
		ReliableProviderID:            cfg.ReliableProviderID,
		FallbackProviderIDs:           cfg.FallbackProviderIDs,
		Providers:                     appConfig.ListProvidersSnapshot(),
		DefaultProviderID:             defaultID,
		TaskText:                      taskText,
		AgentType:                     string(info.Type),
		RepairAttempts:                hints.RepairAttempts,
		VerifyFailed:                  hints.VerifyFailed,
		BootFixIntent:                 hints.BootFixIntent,
		InstalledOllamaTags:           collectInstalledOllamaTags(ctx),
		OllamaTagToolFilter:           ollamaToolCapableTagFilter(ctx),
	})
	mainModel, mainReason := implrouting.SelectMainModel(implrouting.Input{
		RoutingEnabled:                cfg.RoutingEnabled,
		ModelCapabilityRoutingEnabled: appConfig.Routing.ModelCapabilityRoutingEnabled && capabilityProfilesLoaded(),
		LocalToolModel:                cfg.LocalToolModelOrDefault(),
		ReliableToolModel:             cfg.ReliableToolModelOrDefault(),
		ReliableProviderID:            cfg.ReliableProviderID,
		TaskText:                      taskText,
		AgentType:                     string(info.Type),
		RepairAttempts:                hints.RepairAttempts,
		VerifyFailed:                  hints.VerifyFailed,
		BootFixIntent:                 hints.BootFixIntent,
		InstalledOllamaTags:           collectInstalledOllamaTags(ctx),
		OllamaTagToolFilter:           ollamaToolCapableTagFilter(ctx),
	})
	plan = agent.ImplementationRoutingPlan{
		ProviderID: selID,
		ToolModel:  toolModel,
		Reason:     reason,
	}
	if mainReason != "" && mainReason != "default_agent_provider" {
		plan.Reason = mainReason
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
	if ollamaBase, ok := p.(*ai.OllamaProvider); ok {
		tag := strings.TrimSpace(mainModel)
		if hasLoRACapability(capLoRAAdapters) && tag == "" {
			loraTags := collectInstalledLoRATags(ctx)
			dec := classifyTask(ctx, appConfig, taskText, string(info.Type), agentModelForName(info.Name), false, loraTags)
			if t := strings.TrimSpace(dec.LoRATag); t != "" {
				tag = t
				mainReason = dec.Reason
			}
		}
		if tag != "" && tag != strings.TrimSpace(ollamaBase.GetModel()) {
			log.Printf("[impl-routing] %s: chat_model=%s reason=%s", info.Name, tag, mainReason)
			p = ai.OllamaWithModel(ollamaBase, tag)
		}
	}
	return plan, p
}
