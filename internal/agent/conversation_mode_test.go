package agent

import (
	"strings"
	"testing"

	"github.com/camronwood/neural-junkie/internal/ai"
	"github.com/camronwood/neural-junkie/internal/protocol"
)

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
