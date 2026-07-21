package main

import (
	"context"
	"log"
	"strings"

	"github.com/camronwood/neural-junkie/internal/agent"
	"github.com/camronwood/neural-junkie/internal/ai"
	"github.com/camronwood/neural-junkie/internal/protocol"
	"github.com/camronwood/neural-junkie/internal/routing/capabilities"
)

func applyChatLoRATag(ctx context.Context, base *ai.OllamaProvider, info protocol.AgentInfo, msg *protocol.Message) ai.AIProvider {
	if !hasLoRACapability(capLoRAAdapters) || appConfig == nil {
		return base
	}
	loraTags := collectInstalledLoRATags(ctx)
	dec := classifyTask(ctx, appConfig, msg.Content, string(info.Type), agentModelForName(info.Name), false, loraTags)
	tag := strings.TrimSpace(dec.LoRATag)
	if tag == "" {
		return base
	}
	log.Printf("[chat-routing] %s: lora_model=%s reason=%s", info.Name, tag, dec.Reason)
	return ai.OllamaWithModel(base, tag)
}

type chatRoutingRuntime struct{}

func (chatRoutingRuntime) EffectiveAI(ctx context.Context, base ai.AIProvider, info protocol.AgentInfo, msg *protocol.Message, trust agent.ConversationTrustDecision) ai.AIProvider {
	if msg == nil || base == nil || appConfig == nil {
		return base
	}
	if msg.Type == protocol.MessageTypeCollabTask {
		return base
	}
	if shouldSkipChatCapabilityRouting(msg) {
		return base
	}
	if trust.Tier == agent.ConversationTierReliable {
		if reliable := configuredReliableChatProvider(info); reliable != nil {
			log.Printf("[chat-routing] %s: tier=reliable provider=%s reasons=%s", info.Name, providerIDForLog(reliable), strings.Join(trust.Reasons, ","))
			return reliable
		}
		// A missing or invalid reliable provider must degrade safely. Prefer the
		// capability-ranked elevated route before falling back to the base.
		trust.Tier = agent.ConversationTierElevated
	}
	if trust.Tier == "" || trust.Tier == agent.ConversationTierStandard {
		if ollamaBase, ok := base.(*ai.OllamaProvider); ok {
			return applyChatLoRATag(ctx, ollamaBase, info, msg)
		}
		return base
	}
	if trust.Tier != agent.ConversationTierElevated ||
		!appConfig.Routing.ModelCapabilityRoutingEnabled || !capabilityProfilesLoaded() {
		return base
	}
	ollamaBase, ok := base.(*ai.OllamaProvider)
	if !ok {
		return base
	}

	caps := protocol.ResolveTurnCapabilities(msg)
	class := capabilities.ClassifyChat(capabilities.ChatInput{
		Text:        msg.Content,
		AgentType:   string(info.Type),
		AskMode:     caps.ComposerMode == "ask",
		ImplSession: caps.CanRunImplSession,
	})
	installed := collectInstalledOllamaTags(ctx)
	tagFilter := ollamaToolCapableTagFilter(ctx)
	needsTools := capabilities.RequiresToolCapableModel(class) ||
		agent.UserRequestsGeneratedImage(msg.Content) ||
		agent.UserRequestsGeneratedMusic(msg.Content)
	var sel capabilities.SelectResult
	if needsTools {
		sel = capabilities.SelectOllamaTagWithFilter(capabilities.Global(), class, installed, ollamaBase.GetModel(), tagFilter)
	} else {
		sel = capabilities.SelectOllamaTag(capabilities.Global(), class, installed, ollamaBase.GetModel())
	}
	tag := strings.TrimSpace(sel.Tag)
	if tag == "" || tag == strings.TrimSpace(ollamaBase.GetModel()) {
		return base
	}
	log.Printf("[chat-routing] %s: model=%s reason=%s", info.Name, tag, sel.Reason)
	return ai.OllamaWithModel(ollamaBase, tag)
}

func configuredReliableChatProvider(info protocol.AgentInfo) ai.AIProvider {
	if appConfig == nil || globalProviderCache == nil {
		return nil
	}
	id := strings.TrimSpace(appConfig.Implementation.ReliableProviderID)
	if id == "" {
		return nil
	}
	provider, err := globalProviderCache.Get(appConfig, id)
	if err != nil {
		log.Printf("[chat-routing] %s: reliable provider %s unavailable: %v", info.Name, id, err)
		return nil
	}
	return provider
}

func providerIDForLog(provider ai.AIProvider) string {
	if identified, ok := provider.(interface{ ProviderID() string }); ok {
		return identified.ProviderID()
	}
	return provider.GetModel()
}

func shouldSkipChatCapabilityRouting(msg *protocol.Message) bool {
	switch msg.Type {
	case protocol.MessageTypeChat, protocol.MessageTypeQuestion:
		return false
	default:
		return true
	}
}
