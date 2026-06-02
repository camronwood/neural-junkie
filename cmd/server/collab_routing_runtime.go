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
	if msg == nil || msg.Type != protocol.MessageTypeCollabTask || strings.TrimSpace(msg.GetTaskID()) == "" {
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
		selID, reason := routing.SelectProviderID(routing.Input{
			TaskText:          msg.Content,
			HasUserImages:     hasImages,
			Providers:         snap,
			DefaultProviderID: defaultID,
			AvailableLoRATags: loraTags,
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
	plan := routing.PlanTask(in)
	model := plan.ExpectedModel(in.Providers)
	return agent.TaskRoutingPlan{
		ProviderID: plan.ProviderID,
		Model:      model,
		Reason:     plan.RoutingReason(),
	}
}

func buildCollabPlanInput(ctx context.Context, assignee protocol.AgentInfo, taskText string, overrides agent.TaskRoutingOverrides) routing.PlanInput {
	in := routing.PlanInput{
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
	}
	if appConfig != nil {
		in.SmartRoutingEnabled = appConfig.Collaboration.SmartRoutingEnabled
		in.Providers = appConfig.ListProvidersSnapshot()
	}
	return in
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

	plan := routing.PlanTask(buildCollabPlanInput(ctx, info, msg.Content, overrides))
	tag := strings.TrimSpace(plan.OllamaModel)
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
