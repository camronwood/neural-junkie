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
	_ = collab
	if appConfig == nil || !appConfig.Collaboration.SmartRoutingEnabled {
		return base
	}
	if msg == nil || msg.Type != protocol.MessageTypeCollabTask || strings.TrimSpace(msg.GetTaskID()) == "" {
		return base
	}
	if msg.Metadata != nil {
		if pid, ok := msg.Metadata["task_provider_id"].(string); ok && strings.TrimSpace(pid) != "" {
			if p, err := globalProviderCache.Get(appConfig, strings.TrimSpace(pid)); err == nil {
				return applyLoRAModelOverride(ctx, p, info, msg, "task_provider_metadata")
			}
		}
	}

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
	p, err := globalProviderCache.Get(appConfig, selID)
	if err != nil {
		log.Printf("[collab-routing] %s: fallback to base provider: %v (reason=%s)", info.Name, err, reason)
		return base
	}
	log.Printf("[collab-routing] %s: provider_id=%s reason=%s", info.Name, selID, reason)
	return applyLoRAModelOverride(ctx, p, info, msg, reason)
}

func applyLoRAModelOverride(ctx context.Context, p ai.AIProvider, info protocol.AgentInfo, msg *protocol.Message, providerReason string) ai.AIProvider {
	if p == nil {
		return p
	}
	ollamaBase, ok := p.(*ai.OllamaProvider)
	if !ok {
		return p
	}

	if msg.Metadata != nil {
		if m, ok := msg.Metadata["task_ollama_model"].(string); ok {
			tag := strings.TrimSpace(m)
			if tag != "" && strings.HasPrefix(tag, "nj-") && !hasLoRACapability(capLoRAAdapters) {
				return p
			}
			if tag != "" {
				log.Printf("[collab-routing] %s: model=%s reason=task_ollama_model (provider=%s)", info.Name, tag, providerReason)
				return ai.OllamaWithModel(ollamaBase, tag)
			}
		}
	}

	if !hasLoRACapability(capLoRAAdapters) {
		return p
	}

	loraTags := collectInstalledLoRATags(ctx)
	agentModel := agentModelForName(info.Name)
	tag, tagReason := routing.SelectComposedTag(routing.LoRAInput{
		TaskText:      msg.Content,
		AgentType:     string(info.Type),
		AgentModel:    agentModel,
		InstalledTags: loraTags,
	})
	if tag == "" {
		return p
	}
	log.Printf("[collab-routing] %s: model=%s reason=%s (provider=%s)", info.Name, tag, tagReason, providerReason)
	return ai.OllamaWithModel(ollamaBase, tag)
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
