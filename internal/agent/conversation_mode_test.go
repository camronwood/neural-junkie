package agent

import (
	"strings"
	"testing"

	"github.com/camronwood/neural-junkie/internal/ai"
	"github.com/camronwood/neural-junkie/internal/protocol"
)

func TestIsSocialOrStatusPing(t *testing.T) {
	if !isSocialOrStatusPing("@here whats going on!?!") {
		t.Fatal("expected @here status ping")
	}
	if !isSocialOrStatusPing("@here") {
		t.Fatal("expected bare @here")
	}
	if isSocialOrStatusPing("@here refactor cmd/server/main.go") {
		t.Fatal("code task with @here should not be social-only")
	}
}

func TestInferConversationMode_socialPing(t *testing.T) {
	msg := protocol.NewMessage(protocol.MessageTypeChat, "general", protocol.AgentInfo{ID: "u", Name: "User"}, "@here whats going on!?!")
	if got := inferConversationModeFromMessage(msg, protocol.ChannelTypePublic); got != ConversationModeChat {
		t.Fatalf("got %q want chat", got)
	}
}

func TestBuildCLIBridgePrompt_omitsNJProtocols(t *testing.T) {
	hub := shouldRespondTestHub{}
	ag := NewAgent(protocol.AgentTypeCLI, "ClaudeCode", []string{"code"}, ai.NewMockProvider(), hub)
	ag.Info.AIProvider = "claude-cli"
	ag.Info.AIModel = "claude-agent"
	msg := protocol.NewMessage(protocol.MessageTypeChat, "general", protocol.AgentInfo{ID: "u", Name: "User"}, "@here whats going on!?!")
	msg.Metadata = map[string]interface{}{MetadataConversationMode: "chat"}
	prompt := ag.buildPrompt(msg)
	if strings.Contains(prompt, "[FILE_CHANGE]") || strings.Contains(prompt, "[/FILE_CHANGE]") {
		t.Fatal("CLI bridge prompt should not include FILE_CHANGE protocol blocks")
	}
	if strings.Contains(prompt, "appendAskUser") || strings.Contains(strings.ToLower(prompt), "tool: ask_user") {
		t.Fatal("CLI bridge prompt should not include ask_user tool docs")
	}
	if !strings.Contains(prompt, "CLI coding agent") {
		t.Fatal("expected CLI bridge identity")
	}
	if !strings.Contains(prompt, "CHANNEL PING") {
		t.Fatal("expected social ping guidance")
	}
	if !strings.Contains(prompt, "Neural Junkie-native protocols") {
		t.Fatal("expected explicit NJ-protocol exclusion")
	}
}

func TestEffectiveConversationMode_chatMetadata(t *testing.T) {
	msg := protocol.NewMessage(protocol.MessageTypeChat, "general", protocol.AgentInfo{ID: "u", Name: "User"}, "hello")
	msg.Metadata = map[string]interface{}{MetadataConversationMode: "chat"}
	if got := EffectiveConversationMode(msg, protocol.ChannelTypePublic); got != ConversationModeChat {
		t.Fatalf("got %q want chat", got)
	}
}

func TestEffectiveConversationMode_clarifyMetadata(t *testing.T) {
	msg := protocol.NewMessage(protocol.MessageTypeChat, "general", protocol.AgentInfo{ID: "u", Name: "User"}, "How do I update SSO?")
	msg.Metadata = map[string]interface{}{MetadataConversationMode: ConversationModeClarify}
	if got := EffectiveConversationMode(msg, protocol.ChannelTypePublic); got != ConversationModeClarify {
		t.Fatalf("got %q want clarify", got)
	}
	if got := ToolingConversationMode(msg, protocol.ChannelTypePublic); got != ConversationModeChat {
		t.Fatalf("tooling mode got %q want chat", got)
	}
}

func TestClassifyTurnIntent_clarifyModeNotTask(t *testing.T) {
	msg := protocol.NewMessage(protocol.MessageTypeChat, "general", protocol.AgentInfo{ID: "u", Name: "User"}, "How do I update my AWS SSO?")
	msg.Metadata = map[string]interface{}{MetadataConversationMode: ConversationModeClarify}
	if got := classifyTurnIntent(msg, protocol.ChannelTypePublic, "a1", nil); got != IntentSubstantive {
		t.Fatalf("got %v want substantive", got)
	}
}

func TestClassifyTurnIntent_taskVerb(t *testing.T) {
	msg := protocol.NewMessage(protocol.MessageTypeChat, "general", protocol.AgentInfo{ID: "u", Name: "User"}, "review cmd/server/main.go")
	if got := classifyTurnIntent(msg, protocol.ChannelTypePublic, "a1", nil); got != IntentTask {
		t.Fatalf("got %v want task", got)
	}
}

func TestClassifyTurnIntent_chatModeForcesCasual(t *testing.T) {
	msg := protocol.NewMessage(protocol.MessageTypeChat, "general", protocol.AgentInfo{ID: "u", Name: "User"}, "what do you think about golang?")
	msg.Metadata = map[string]interface{}{MetadataConversationMode: "chat"}
	if got := classifyTurnIntent(msg, protocol.ChannelTypePublic, "a1", nil); got != IntentLowSignal {
		t.Fatalf("got %v want casual", got)
	}
}

func TestShouldMaintainSessionSummary_public(t *testing.T) {
	if !shouldMaintainSessionSummary(protocol.ChannelTypePublic, "general") {
		t.Fatal("expected public channel to maintain summary")
	}
}

func TestApplyContextBudget_truncatesSystem(t *testing.T) {
	long := strings.Repeat("x", 40*1024)
	prompt := "=== SESSION SUMMARY ===\n" + long + ai.SystemPromptSeparator + "user question"
	out, stats := applyContextBudget(prompt)
	if !stats.Truncated {
		t.Fatal("expected truncation")
	}
	if !strings.Contains(out, "user question") {
		t.Fatal("user section should remain")
	}
	if len(out) >= len(prompt) {
		t.Fatal("expected smaller prompt")
	}
}

func TestPromptPersonaTier_dm(t *testing.T) {
	hub := shouldRespondTestHub{}
	ag := NewAgent(protocol.AgentTypeBackend, "BackendEngineer", []string{"go"}, ai.NewMockProvider(), hub)
	msg := protocol.NewMessage(protocol.MessageTypeChat, "dm-user-backendengineer", protocol.AgentInfo{ID: "u", Name: "User"}, "hi")
	if tier := ag.promptPersonaTier(msg); tier != PersonaDirect {
		t.Fatalf("got %v want direct", tier)
	}
}

func TestClassifyTurnIntent_scanToolIsTask(t *testing.T) {
	msg := protocol.NewMessage(protocol.MessageTypeQuestion, "dm-u-b", protocol.AgentInfo{ID: "u", Name: "User"}, "Use summarize_scan_analysis on the file I have open")
	msg.Metadata = map[string]interface{}{MetadataConversationMode: "chat"}
	if got := classifyTurnIntent(msg, protocol.ChannelTypeDM, "bio1", nil); got != IntentTask {
		t.Fatalf("got %v want task", got)
	}
}

func TestHasScanOrEditorTaskSignals(t *testing.T) {
	if !hasScanOrEditorTaskSignals("summarize_scan_analysis on my open file") {
		t.Fatal("expected scan tool signal")
	}
	if !hasScanOrEditorTaskSignals("its open in my editor now") {
		t.Fatal("expected editor signal")
	}
}

func TestShouldIncludeToolingInPrompt_casualChat(t *testing.T) {
	hub := shouldRespondTestHub{}
	ag := NewAgent(protocol.AgentTypeBackend, "BackendEngineer", []string{"go"}, ai.NewMockProvider(), hub)
	msg := protocol.NewMessage(protocol.MessageTypeChat, "dm-u-b", protocol.AgentInfo{ID: "u", Name: "User"}, "hello")
	msg.Metadata = map[string]interface{}{MetadataConversationMode: "chat"}
	if ag.shouldIncludeToolingInPrompt(msg, IntentLowSignal) {
		t.Fatal("casual chat should not include tooling")
	}
}
