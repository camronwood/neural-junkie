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

func TestLocalEscalationLadderMovesFrom3BToLargerInstalledModel(t *testing.T) {
	installed := map[string]struct{}{
		"qwen2.5:3b":  {},
		"qwen3.5:9b":  {},
		"qwen3.5:27b": {},
	}
	attempts := []protocol.RoutingAttempt{{
		ProviderID: "ollama-local", Model: "qwen2.5:3b", Tier: "standard",
		FailureReason: agent.ConversationReasonQualityGateFailure,
	}}
	got := chooseNextLocalChatModel(
		installed,
		[]string{"qwen3.5:27b", "qwen3.5:9b", "qwen2.5:3b"},
		"qwen2.5:3b",
		attempts,
	)
	if got != "qwen3.5:9b" {
		t.Fatalf("next model = %q, want first capability-ranked larger local model", got)
	}
	attempts = append(attempts, protocol.RoutingAttempt{ProviderID: "ollama-local", Model: got})
	if again := chooseNextLocalChatModel(installed, []string{"qwen3.5:9b"}, got, attempts); again != "" {
		t.Fatalf("tier repeated as %q; each tier must be attempted at most once", again)
	}
}

func TestChatFrontierRequiresExplicitConsentAfterLocalExhaustion(t *testing.T) {
	restoreChatRoutingGlobals(t)
	appConfig = config.DefaultConfig()
	appConfig.Routing.ModelCapabilityRoutingEnabled = false
	appConfig.AI.Providers = append(appConfig.AI.Providers, config.ProviderConfig{
		ID: "frontier", Type: "openai-compatible", Endpoint: "http://localhost:1234", Model: "frontier-model",
	})
	appConfig.Implementation.ReliableProviderID = "frontier"
	globalProviderCache = ai.NewProviderCache()

	base := ai.NewOllamaProviderWithConfig("http://localhost:11434", "qwen2.5:3b")
	msg := protocol.NewMessage(protocol.MessageTypeChat, "dm", protocol.AgentInfo{Name: "Camron"}, "Try again")
	msg.Metadata[protocol.MetadataRoutingAttempts] = []protocol.RoutingAttempt{
		{ProviderID: "qwen2.5:3b", Model: "qwen2.5:3b", Tier: "standard", FailureReason: agent.ConversationReasonQualityGateFailure},
		{ProviderID: "qwen3.5:9b", Model: "qwen3.5:9b", Tier: "reliable", FailureReason: agent.ConversationReasonQualityGateFailure},
	}
	trust := agent.ConversationTrustDecision{Tier: agent.ConversationTierReliable}

	blocked := (chatRoutingRuntime{}).EffectiveAI(context.Background(), base, protocol.AgentInfo{Name: "Assistant"}, msg, trust)
	if blocked.GetModel() == "frontier-model" {
		t.Fatal("configured frontier was used without explicit consent")
	}

	appConfig.Routing.FrontierEscalationEnabled = true
	allowed := (chatRoutingRuntime{}).EffectiveAI(context.Background(), base, protocol.AgentInfo{Name: "Assistant"}, msg, trust)
	if allowed.GetModel() != "frontier-model" {
		t.Fatalf("model = %q, want frontier after exhausted local attempts and consent", allowed.GetModel())
	}
	attempts := protocol.ExtractRoutingMeta(msg).Attempts
	if len(attempts) != 3 || attempts[2].Tier != "frontier" {
		t.Fatalf("attempts = %+v, want final frontier attempt metadata", attempts)
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
