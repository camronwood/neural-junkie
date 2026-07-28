package main

import (
	"context"
	"log"
	"regexp"
	"strconv"
	"strings"

	"github.com/camronwood/neural-junkie/internal/agent"
	"github.com/camronwood/neural-junkie/internal/ai"
	"github.com/camronwood/neural-junkie/internal/protocol"
	"github.com/camronwood/neural-junkie/internal/routing"
	"github.com/camronwood/neural-junkie/internal/routing/capabilities"
)

func applyChatLoRATag(ctx context.Context, base *ai.OllamaProvider, info protocol.AgentInfo, msg *protocol.Message) ai.AIProvider {
	if !hasLoRACapability(capLoRAAdapters) || appConfig == nil {
		return base
	}
	loraTags := collectInstalledLoRATags(ctx)
	var dec routing.RoutingDecision
	if semanticDecision, ok := protocol.ExtractTurnDecision(msg); ok {
		dec = routing.DecisionFromSemantic(semanticDecision, loraTags)
	} else {
		dec = classifyTask(ctx, appConfig, msg.Content, string(info.Type), agentModelForName(info.Name), false, loraTags)
	}
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
		if local := nextLocalChatProvider(ctx, base, info, msg); local != nil {
			log.Printf("[chat-routing] %s: tier=reliable local_provider=%s model=%s reasons=%s", info.Name, providerIDForLog(local), local.GetModel(), strings.Join(trust.Reasons, ","))
			recordChatRoutingAttempt(msg, local, string(trust.Tier), "local_escalation")
			return local
		}
		if appConfig.Routing.FrontierEscalationEnabled {
			if frontier := configuredReliableChatProvider(info, true); frontier != nil {
				log.Printf("[chat-routing] %s: tier=frontier provider=%s reasons=%s", info.Name, providerIDForLog(frontier), strings.Join(trust.Reasons, ","))
				recordChatRoutingAttempt(msg, frontier, "frontier", "frontier_after_local_exhaustion")
				return frontier
			}
		}
		// No unique local tier remains and frontier use lacks explicit consent.
		trust.Tier = agent.ConversationTierElevated
	}
	if trust.Tier == "" || trust.Tier == agent.ConversationTierStandard {
		if ollamaBase, ok := base.(*ai.OllamaProvider); ok {
			selected := applyChatLoRATag(ctx, ollamaBase, info, msg)
			recordChatRoutingAttempt(msg, selected, string(agent.ConversationTierStandard), "standard_route")
			return selected
		}
		recordChatRoutingAttempt(msg, base, string(agent.ConversationTierStandard), "standard_route")
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
	class := chatCapabilityClass(msg, info, caps)
	installed := collectInstalledOllamaTags(ctx)
	tagFilter := ollamaToolCapableTagFilter(ctx)
	needsTools := capabilities.RequiresToolCapableModel(class)
	if decision, ok := protocol.ExtractTurnDecision(msg); ok {
		needsTools = needsTools || routing.DecisionFromSemantic(decision, nil).ToolNeed
	}
	var sel capabilities.SelectResult
	if needsTools {
		sel = capabilities.SelectOllamaTagWithFilter(capabilities.Global(), class, installed, ollamaBase.GetModel(), tagFilter)
	} else {
		sel = capabilities.SelectOllamaTag(capabilities.Global(), class, installed, ollamaBase.GetModel())
	}
	tag := strings.TrimSpace(sel.Tag)
	if tag == "" || tag == strings.TrimSpace(ollamaBase.GetModel()) {
		recordChatRoutingAttempt(msg, base, string(trust.Tier), sel.Reason)
		return base
	}
	log.Printf("[chat-routing] %s: model=%s reason=%s", info.Name, tag, sel.Reason)
	selected := ai.OllamaWithModel(ollamaBase, tag)
	recordChatRoutingAttempt(msg, selected, string(trust.Tier), sel.Reason)
	return selected
}

func configuredReliableChatProvider(info protocol.AgentInfo, frontierOnly bool) ai.AIProvider {
	if appConfig == nil || globalProviderCache == nil {
		return nil
	}
	id := strings.TrimSpace(appConfig.Implementation.ReliableProviderID)
	if id == "" {
		return nil
	}
	if frontierOnly {
		cfg := appConfig.GetProvider(id)
		if cfg == nil || strings.EqualFold(strings.TrimSpace(cfg.Type), "ollama") {
			return nil
		}
	}
	provider, err := globalProviderCache.Get(appConfig, id)
	if err != nil {
		log.Printf("[chat-routing] %s: reliable provider %s unavailable: %v", info.Name, id, err)
		return nil
	}
	return provider
}

func nextLocalChatProvider(ctx context.Context, base ai.AIProvider, info protocol.AgentInfo, msg *protocol.Message) ai.AIProvider {
	if appConfig == nil || !appConfig.Routing.LocalEscalationEnabled {
		return nil
	}
	ollamaBase, ok := base.(*ai.OllamaProvider)
	if !ok {
		return configuredLocalReliableChatProvider(info, msg)
	}
	installed := collectInstalledOllamaTags(ctx)
	attempts := protocol.ExtractRoutingMeta(msg).Attempts
	current := strings.TrimSpace(ollamaBase.GetModel())
	if len(attempts) > 0 && strings.TrimSpace(attempts[len(attempts)-1].Model) != "" {
		current = strings.TrimSpace(attempts[len(attempts)-1].Model)
	}
	caps := protocol.ResolveTurnCapabilities(msg)
	class := chatCapabilityClass(msg, info, caps)
	var candidates []string
	if appConfig.Routing.ModelCapabilityRoutingEnabled && capabilityProfilesLoaded() {
		candidates = append(candidates, capabilities.Global().Tags(class)...)
	}
	candidates = append(candidates, appConfig.Implementation.ReliableToolModelOrDefault())
	if tag := chooseNextLocalChatModel(installed, candidates, current, attempts); tag != "" {
		return ai.OllamaWithModel(ollamaBase, tag)
	}
	return configuredLocalReliableChatProvider(info, msg)
}

func chatCapabilityClass(msg *protocol.Message, info protocol.AgentInfo, caps protocol.TurnCapabilities) capabilities.TaskClass {
	if decision, ok := protocol.ExtractTurnDecision(msg); ok {
		return capabilities.ClassifySemantic(decision, caps.ComposerMode == "ask", caps.CanRunImplSession)
	}
	return capabilities.ClassifyChat(capabilities.ChatInput{
		Text: msg.Content, AgentType: string(info.Type),
		AskMode: caps.ComposerMode == "ask", ImplSession: caps.CanRunImplSession,
	})
}

func chooseNextLocalChatModel(installed map[string]struct{}, candidates []string, current string, attempts []protocol.RoutingAttempt) string {
	current = strings.TrimSpace(current)
	currentRank := -1
	for i, candidate := range candidates {
		if strings.TrimSpace(candidate) == current {
			currentRank = i
			break
		}
	}
	if currentRank > 0 {
		for i := currentRank - 1; i >= 0; i-- {
			tag := strings.TrimSpace(candidates[i])
			if tag != "" && installedOllamaTag(installed, tag) && !chatAttempted(attempts, "", tag) {
				return tag
			}
		}
		return ""
	}

	currentSize, hasCurrentSize := modelParameterBillions(current)
	bestTag := ""
	bestSize := 0.0
	for _, tag := range candidates {
		tag = strings.TrimSpace(tag)
		if tag == "" || tag == current || !installedOllamaTag(installed, tag) ||
			chatAttempted(attempts, "", tag) {
			continue
		}
		if hasCurrentSize {
			size, ok := modelParameterBillions(tag)
			if ok {
				if size <= currentSize || (bestTag != "" && bestSize > 0 && size >= bestSize) {
					continue
				}
				bestTag, bestSize = tag, size
				continue
			}
			if bestTag != "" {
				continue
			}
		}
		if bestTag == "" {
			bestTag = tag
		}
	}
	return bestTag
}

var modelParameterRE = regexp.MustCompile(`(?i)(\d+(?:\.\d+)?)b(?:\b|-)`)

func modelParameterBillions(tag string) (float64, bool) {
	match := modelParameterRE.FindStringSubmatch(tag)
	if len(match) != 2 {
		return 0, false
	}
	size, err := strconv.ParseFloat(match[1], 64)
	return size, err == nil
}

func configuredLocalReliableChatProvider(info protocol.AgentInfo, msg *protocol.Message) ai.AIProvider {
	if appConfig == nil {
		return nil
	}
	id := strings.TrimSpace(appConfig.Implementation.ReliableProviderID)
	cfg := appConfig.GetProvider(id)
	if cfg == nil || !strings.EqualFold(strings.TrimSpace(cfg.Type), "ollama") {
		return nil
	}
	provider := configuredReliableChatProvider(info, false)
	if provider == nil || chatAttempted(protocol.ExtractRoutingMeta(msg).Attempts, providerIDForLog(provider), provider.GetModel()) {
		return nil
	}
	return provider
}

func installedOllamaTag(installed map[string]struct{}, tag string) bool {
	if len(installed) == 0 {
		return false
	}
	if _, ok := installed[tag]; ok {
		return true
	}
	base := strings.SplitN(tag, ":", 2)[0]
	for candidate := range installed {
		if candidate == tag || strings.HasPrefix(candidate, base+":") {
			return true
		}
	}
	return false
}

func recordChatRoutingAttempt(msg *protocol.Message, provider ai.AIProvider, tier, reason string) {
	if msg == nil || provider == nil {
		return
	}
	if msg.Metadata == nil {
		msg.Metadata = make(map[string]interface{})
	}
	attempts := protocol.ExtractRoutingMeta(msg).Attempts
	id, model := providerIDForLog(provider), strings.TrimSpace(provider.GetModel())
	if chatAttempted(attempts, id, model) {
		return
	}
	attempts = append(attempts, protocol.RoutingAttempt{
		ProviderID: id, Model: model, Tier: tier, Reason: reason,
	})
	msg.Metadata[protocol.MetadataRoutingAttempts] = attempts
}

func chatAttempted(attempts []protocol.RoutingAttempt, providerID, model string) bool {
	for _, attempt := range attempts {
		sameProvider := providerID == "" || attempt.ProviderID == providerID
		if sameProvider && strings.TrimSpace(attempt.Model) == strings.TrimSpace(model) {
			return true
		}
	}
	return false
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
