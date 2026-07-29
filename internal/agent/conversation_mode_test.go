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
	// Presence pings are stamped InteractionCasual by the classifier — not phrase-matched here.
	if isSocialOrStatusPing("are you here and ready to help?") {
		t.Fatal("unstructured presence text must not phrase-match as social ping")
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
	if !shouldMaintainSessionSummary(protocol.ChannelTypeCollaboration, "collab-demo") {
		t.Fatal("expected collaboration channel to maintain summary")
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

func TestAppendModalAccessibilityGapFollowUpPrompt(t *testing.T) {
	msg := protocol.NewMessage(protocol.MessageTypeChat, "dm", protocol.AgentInfo{Name: "User"},
		"This isn't quite done — where's aria-labelledby, tabIndex={-1} for empty focusable, and display:none filtering? Nested modal Escape isolation?")
	var b strings.Builder
	appendModalAccessibilityGapFollowUpPrompt(&b, msg)
	out := b.String()
	if !strings.Contains(out, "MODAL ACCESSIBILITY GAP-FILL") || !strings.Contains(out, "offsetParent") {
		t.Fatalf("expected gap-fill prompt: %q", out)
	}
	if !strings.Contains(out, "contains(document.activeElement)") {
		t.Fatal("expected containment guard guidance")
	}
}

func TestLooksLikeCodeCritiqueFollowUp(t *testing.T) {
	critique := "Hold on — you never restore focus (that `triggerElementRef` you declared is dead code). Fix the trigger-restore logic — show me the corrected effect block."
	if !looksLikeCodeCritiqueFollowUp(critique) {
		t.Fatal("expected modal critique to match")
	}
	if looksLikeCodeCritiqueFollowUp("What is a React ref?") {
		t.Fatal("generic question should not match")
	}
}

func TestAppendCodeCritiqueFollowUpPrompt(t *testing.T) {
	msg := protocol.NewMessage(protocol.MessageTypeChat, "dm", protocol.AgentInfo{Name: "User"}, "Hold on — dead code in `triggerElementRef`. Show me the corrected effect block.")
	var b strings.Builder
	appendCodeCritiqueFollowUpPrompt(&b, msg)
	out := b.String()
	if !strings.Contains(out, "CODE CORRECTION FOLLOW-UP") || !strings.Contains(out, "EVERY specific issue") {
		t.Fatalf("expected critique prompt: %q", out)
	}
	if !strings.Contains(out, "onClose") {
		t.Fatal("expected onClose guidance in critique prompt")
	}
}

func TestLooksLikeConcreteCodeRequest(t *testing.T) {
	ask := "I need an accessible modal in React — focus trap, Escape closes it. What's the actual implementation, not just \"use a library\"?"
	if !looksLikeConcreteCodeRequest(ask) {
		t.Fatal("expected initial implementation ask to match")
	}
	push := "This is still just prose — show me the actual hook implementation with real code."
	if !looksLikeConcreteCodeRequest(push) {
		t.Fatal("expected push-for-code message to match")
	}
	push1158 := "That's not an answer at all. Give me the actual mechanics — where does initial focus go, how do you trap Tab/Shift+Tab inside the dialog, how do you restore focus on close, and how does Escape wire up. Code or specifics, not a punt."
	if !looksLikeConcreteCodeRequest(push1158) {
		t.Fatal("expected 1158 iter2 push-for-mechanics message to match")
	}
	if looksLikeConcreteCodeRequest("What is React?") {
		t.Fatal("generic question should not match")
	}
}

func TestAppendConcreteCodeRequestPrompt(t *testing.T) {
	msg := protocol.NewMessage(protocol.MessageTypeChat, "dm", protocol.AgentInfo{Name: "User"},
		"What's the actual implementation, not just use a library?")
	var b strings.Builder
	appendConcreteCodeRequestPrompt(&b, msg)
	out := b.String()
	if !strings.Contains(out, "CONCRETE CODE REQUEST") || !strings.Contains(out, "onClose") {
		t.Fatalf("expected concrete code prompt: %q", out)
	}
}

func TestLooksLikeRepoFactAsk(t *testing.T) {
	ask := "Hey, I'm brand new to this repo — can you explain how the regression hub's health check actually works, and what HTTP path I'd hit to check it?"
	if !looksLikeRepoFactAsk(ask) {
		t.Fatal("expected health-check path ask to match")
	}
	if looksLikeRepoFactAsk("What is Go?") {
		t.Fatal("generic question should not match")
	}
}

func TestLooksLikeRepoFactChallengeFollowUp(t *testing.T) {
	challenge := "Okay wait, that's a lot of made-up-sounding stuff — is `/v1/hub/health/live` actually a real path in *this* repo, or a generic guess? Also, if I just added a route like `/api/v9/quantum-health` and called it the regression hub health check, would that be correct?"
	if !looksLikeRepoFactChallengeFollowUp(challenge) {
		t.Fatal("expected repo fact challenge to match")
	}
}

func TestLooksLikeRepoFactChallengeFollowUp_iter2(t *testing.T) {
	challenge := "Is that right, or were you just making up that /api/v1/hub/health path?"
	if !looksLikeRepoFactChallengeFollowUp(challenge) {
		t.Fatal("expected making-up challenge to match")
	}
}

func TestAppendRepoFactGroundingPrompt(t *testing.T) {
	msg := protocol.NewMessage(protocol.MessageTypeChat, "dm", protocol.AgentInfo{Name: "User"},
		"what HTTP path is the health check on?")
	var b strings.Builder
	appendRepoFactGroundingPrompt(&b, msg)
	out := b.String()
	if !strings.Contains(out, "REPO FACT GROUNDING") || !strings.Contains(out, "Do NOT invent") {
		t.Fatalf("expected repo fact prompt: %q", out)
	}
}
