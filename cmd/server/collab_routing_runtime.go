package main

import (
	"context"
	"log"
	"strings"

	"github.com/camronwood/neural-junkie/internal/agent"
	"github.com/camronwood/neural-junkie/internal/ai"
	collabrouting "github.com/camronwood/neural-junkie/internal/collaboration/routing"
	"github.com/camronwood/neural-junkie/internal/protocol"
	"github.com/camronwood/neural-junkie/internal/routing"
	"github.com/camronwood/neural-junkie/internal/routing/capabilities"
)

type collabRoutingRuntime struct{}

func (collabRoutingRuntime) EffectiveAI(ctx context.Context, base ai.AIProvider, info protocol.AgentInfo, collab agent.CollaborationInfo, msg *protocol.Message) ai.AIProvider {
	if msg == nil {
		return base
	}
	if appConfig != nil {
		planningID := strings.TrimSpace(appConfig.Collaboration.PlanningProviderID)
		if planningID != "" && collab.Phase == "planning" && msg.Type == protocol.MessageTypeCollabDiscussion {
			p, perr := globalProviderCache.Get(appConfig, planningID)
			if perr == nil {
				log.Printf("[collab-routing] %s: planning_provider_id=%s", info.Name, planningID)
				return p
			}
			log.Printf("[collab-routing] %s: planning provider %q unavailable: %v", info.Name, planningID, perr)
		}
	}
	if msg.Type != protocol.MessageTypeCollabTask || strings.TrimSpace(msg.GetTaskID()) == "" {
		return base
	}
	if msg.Metadata != nil {
		if pid, ok := msg.Metadata["task_provider_id"].(string); ok && strings.TrimSpace(pid) != "" {
			if p, err := globalProviderCache.Get(appConfig, strings.TrimSpace(pid)); err == nil {
				return applyCollabModelOverrides(ctx, p, info, msg, "task_provider_metadata")
			}
		}
	}

	p := base
	providerReason := "default_agent_provider"
	if appConfig != nil && appConfig.Collaboration.SmartRoutingEnabled {
		loraTags := collectInstalledLoRATags(ctx)
		snap := appConfig.ListProvidersSnapshot()
		defaultID := defaultProviderIDForAgentName(info.Name)
		hasImages := len(protocol.ExtractUserImages(msg)) > 0 && info.SupportsVision
		dec := classifyTask(ctx, appConfig, msg.Content, string(info.Type), agentModelForName(info.Name), hasImages, loraTags)
		selID, reason := routing.PickProviderID(routing.ProviderPickInput{
			Decision:          dec,
			HasUserImages:     hasImages,
			Providers:         snap,
			DefaultProviderID: defaultID,
			InstalledTags:     loraTags,
		})
		if next, err := globalProviderCache.Get(appConfig, selID); err == nil {
			p = next
			providerReason = reason
			log.Printf("[collab-routing] %s: provider_id=%s reason=%s", info.Name, selID, reason)
		} else {
			log.Printf("[collab-routing] %s: fallback to base provider: %v (reason=%s)", info.Name, err, reason)
		}
	}

	return applyCollabModelOverrides(ctx, p, info, msg, providerReason)
}

func (collabRoutingRuntime) PlanTask(ctx context.Context, assignee protocol.AgentInfo, taskText string, overrides agent.TaskRoutingOverrides) agent.TaskRoutingPlan {
	in := buildCollabPlanInput(ctx, assignee, taskText, overrides)
	plan := collabrouting.PlanTask(in)
	model := plan.ExpectedModel(in.Providers)
	dec := classifyTask(ctx, appConfig, taskText, string(assignee.Type), in.AgentModel, in.HasUserImages, in.InstalledLoRATags)
	return agent.TaskRoutingPlan{
		ProviderID: plan.ProviderID,
		Model:      model,
		Reason:     plan.RoutingReason(),
		Source:     dec.Source,
		Domain:     dec.Domain,
		CostTier:   dec.CostTier,
	}
}

func buildCollabPlanInput(ctx context.Context, assignee protocol.AgentInfo, taskText string, overrides agent.TaskRoutingOverrides) collabrouting.PlanInput {
	in := collabrouting.PlanInput{
		TaskText:               taskText,
		AgentName:              assignee.Name,
		AgentType:              string(assignee.Type),
		AgentModel:             agentModelForName(assignee.Name),
		DefaultProviderID:      defaultProviderIDForAgentName(assignee.Name),
		TaskProviderOverride:   overrides.ProviderID,
		TaskOllamaModelOverride: overrides.OllamaModel,
		HasLoRACapability:      hasLoRACapability(capLoRAAdapters),
		InstalledLoRATags:      collectInstalledLoRATags(ctx),
		InstalledOllamaTags:    collectInstalledOllamaTags(ctx),
		OllamaTagToolFilter:    ollamaToolCapableTagFilter(ctx),
	}
	if appConfig != nil {
		in.SmartRoutingEnabled = appConfig.Collaboration.SmartRoutingEnabled
		in.ModelCapabilityRoutingEnabled = appConfig.Routing.ModelCapabilityRoutingEnabled && capabilityProfilesLoaded()
		in.Providers = appConfig.ListProvidersSnapshot()
	}
	return in
}

func capabilityProfilesLoaded() bool {
	p := capabilities.Global()
	return p != nil && len(p.Tags(capabilities.TaskImplement)) > 0
}

func applyCollabModelOverrides(ctx context.Context, p ai.AIProvider, info protocol.AgentInfo, msg *protocol.Message, providerReason string) ai.AIProvider {
	if p == nil {
		return p
	}
	ollamaBase, ok := p.(*ai.OllamaProvider)
	if !ok {
		return p
	}

	overrides := agent.TaskRoutingOverrides{}
	if msg.Metadata != nil {
		if m, ok := msg.Metadata["task_ollama_model"].(string); ok {
			tag := strings.TrimSpace(m)
			if tag != "" && strings.HasPrefix(tag, "nj-") && !hasLoRACapability(capLoRAAdapters) {
				return p
			}
			if tag != "" {
				overrides.OllamaModel = tag
			}
		}
	}

	plan := collabrouting.PlanTask(buildCollabPlanInput(ctx, info, msg.Content, overrides))
	tag := strings.TrimSpace(plan.OllamaModel)
	if plan.ModelReason == "deliverable_task_keep_agent_model" {
		log.Printf("[collab-routing] %s: model=%s reason=deliverable_task_keep_agent_model (provider=%s)", info.Name, tag, providerReason)
		return p
	}
	if tag == "" || strings.TrimSpace(plan.ModelReason) == "agent_default_model" {
		return p
	}
	if strings.HasPrefix(tag, "nj-") && !hasLoRACapability(capLoRAAdapters) {
		return p
	}
	log.Printf("[collab-routing] %s: model=%s reason=%s (provider=%s)", info.Name, tag, plan.ModelReason, providerReason)
	return ai.OllamaWithModel(ollamaBase, tag)
}

func collectInstalledOllamaTags(ctx context.Context) map[string]struct{} {
	out := make(map[string]struct{})
	if ollamaMgr == nil {
		return out
	}
	if !ollamaMgr.IsServerRunning(ctx) {
		return out
	}
	names, err := ollamaMgr.ListModels(ctx)
	if err != nil {
		return out
	}
	for _, name := range names {
		name = strings.TrimSpace(name)
		if name != "" {
			out[name] = struct{}{}
		}
	}
	return out
}

func agentModelForName(name string) string {
	if appConfig == nil {
		return ""
	}
	for _, a := range appConfig.Agents {
		if strings.EqualFold(strings.TrimSpace(a.Name), strings.TrimSpace(name)) {
			return strings.TrimSpace(a.Model)
		}
	}
	return ""
}

func collectInstalledLoRATags(ctx context.Context) map[string]struct{} {
	out := make(map[string]struct{})
	if appConfig == nil {
		return out
	}

	candidates := make(map[string]struct{})
	for _, pack := range appConfig.PackCatalog() {
		if !appConfig.IsPackEnabled(pack.ID) {
			continue
		}
		for _, la := range pack.LoRAAdapters {
			tag := strings.TrimSpace(la.OllamaTag)
			if tag == "" && la.AgentType != "" {
				tag = loraTagForAgentType(la.AgentType)
			}
			if strings.HasPrefix(tag, "nj-") {
				candidates[tag] = struct{}{}
			}
		}
	}
	for _, a := range appConfig.Agents {
		if m := strings.TrimSpace(a.Model); strings.HasPrefix(m, "nj-") {
			candidates[m] = struct{}{}
		}
	}

	if ollamaMgr == nil {
		for tag := range candidates {
			out[tag] = struct{}{}
		}
		return out
	}
	if !ollamaMgr.IsServerRunning(ctx) {
		return out
	}
	for tag := range candidates {
		ok, err := ollamaMgr.HasModel(ctx, tag)
		if err == nil && ok {
			out[tag] = struct{}{}
		}
	}
	return out
}

func loraTagForAgentType(agentType string) string {
	t := strings.ToLower(strings.TrimSpace(agentType))
	if t == "biology" {
		return "nj-biology:8b"
	}
	if t == "" {
		return ""
	}
	return "nj-" + t + ":14b"
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
