package main

import (
	"context"
	"testing"

	"github.com/camronwood/neural-junkie/internal/agent"
	"github.com/camronwood/neural-junkie/internal/ai"
	"github.com/camronwood/neural-junkie/internal/config"
	"github.com/camronwood/neural-junkie/internal/protocol"
	"github.com/camronwood/neural-junkie/internal/routing/capabilities"
)

func TestChatRoutingRuntimeElevatedUsesCapabilityRanking(t *testing.T) {
	restoreChatRoutingGlobals(t)
	appConfig = config.DefaultConfig()
	appConfig.Routing.ModelCapabilityRoutingEnabled = true
	capabilities.SetGlobal(&capabilities.Profiles{TaskClasses: map[string][]string{
		string(capabilities.TaskChat):      {"ranked-chat:latest", "base:latest"},
		string(capabilities.TaskImplement): {"ranked-chat:latest"},
	}})

	base := ai.NewOllamaProviderWithConfig("http://localhost:11434", "base:latest")
	msg := protocol.NewMessage(protocol.MessageTypeChat, "dm", protocol.AgentInfo{Name: "Camron"}, "Explain this architecture")
	got := (chatRoutingRuntime{}).EffectiveAI(context.Background(), base, protocol.AgentInfo{Name: "Assistant"}, msg, agent.ConversationTrustDecision{
		Tier:    agent.ConversationTierElevated,
		Reasons: []string{agent.ConversationReasonLargeContext},
	})
	if got.GetModel() != "ranked-chat:latest" {
		t.Fatalf("model = %q, want capability-ranked model", got.GetModel())
	}
}

func TestChatRoutingRuntimeReliableUsesConfiguredProvider(t *testing.T) {
	restoreChatRoutingGlobals(t)
	appConfig = config.DefaultConfig()
	appConfig.AI.Providers = append(appConfig.AI.Providers, config.ProviderConfig{
		ID: "reliable-chat", Type: "ollama", Endpoint: "http://localhost:11434", Model: "reliable:14b",
	})
	appConfig.Implementation.ReliableProviderID = "reliable-chat"
	globalProviderCache = ai.NewProviderCache()

	base := ai.NewOllamaProviderWithConfig("http://localhost:11434", "base:latest")
	msg := protocol.NewMessage(protocol.MessageTypeChat, "dm", protocol.AgentInfo{Name: "Camron"}, "That answer was wrong")
	got := (chatRoutingRuntime{}).EffectiveAI(context.Background(), base, protocol.AgentInfo{Name: "Assistant"}, msg, agent.ConversationTrustDecision{
		Tier:    agent.ConversationTierReliable,
		Reasons: []string{agent.ConversationReasonUserCorrection},
	})
	if got.GetModel() != "reliable:14b" {
		t.Fatalf("model = %q, want configured reliable provider", got.GetModel())
	}
}

func TestChatRoutingRuntimeReliableFallsBackSafely(t *testing.T) {
	restoreChatRoutingGlobals(t)
	appConfig = config.DefaultConfig()
	appConfig.Implementation.ReliableProviderID = "missing"
	globalProviderCache = ai.NewProviderCache()

	base := ai.NewMockProvider()
	msg := protocol.NewMessage(protocol.MessageTypeChat, "dm", protocol.AgentInfo{Name: "Camron"}, "Try again")
	got := (chatRoutingRuntime{}).EffectiveAI(context.Background(), base, protocol.AgentInfo{Name: "Assistant"}, msg, agent.ConversationTrustDecision{
		Tier: agent.ConversationTierReliable,
	})
	if got != base {
		t.Fatalf("provider = %T, want base fallback", got)
	}
}

func restoreChatRoutingGlobals(t *testing.T) {
	t.Helper()
	oldConfig := appConfig
	oldCache := globalProviderCache
	oldProfiles := capabilities.Global()
	t.Cleanup(func() {
		appConfig = oldConfig
		globalProviderCache = oldCache
		capabilities.SetGlobal(oldProfiles)
	})
}
