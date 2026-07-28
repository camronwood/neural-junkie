package agent

import (
	"strings"
	"testing"

	"github.com/camronwood/neural-junkie/internal/protocol"
)

func TestSessionSummaryBlock_continueThreadLanguage(t *testing.T) {
	a := &Agent{
		Info: protocol.AgentInfo{Name: "Assistant", Type: protocol.AgentTypeAssistant},
		Hub:  summaryStubHub{summary: "User is planning a Lisbon trip."},
	}
	block := a.sessionSummaryBlock("dm-camron-assistant")
	if block == "" {
		t.Fatal("expected summary block")
	}
	if strings.Contains(block, "Answer ONLY the user's latest message") {
		t.Fatalf("anti-continue copy must be gone: %q", block)
	}
	if !strings.Contains(block, "Continue this conversation") {
		t.Fatalf("expected continue-thread language: %q", block)
	}
}

func TestAppendDurableConversationContext_omitsWorkStateInChat(t *testing.T) {
	envelope := protocol.TurnContextEnvelope{
		Goal: &protocol.TurnContextGoal{ID: "g1", Text: "Implement the API"},
		UnresolvedActions: []protocol.TurnContextAction{
			{ID: "a1", Description: "edit main.go"},
		},
		Corrections: []protocol.TurnContextCorrection{
			{Instruction: "Use Rust"},
		},
	}
	chatMsg := protocol.NewMessage(protocol.MessageTypeChat, "dm", protocol.AgentInfo{Name: "u"}, "what do you think about go?")
	chatMsg.Metadata = map[string]interface{}{MetadataConversationMode: ConversationModeChat}
	out := appendDurableConversationContext("BASE", envelope, chatMsg)
	if strings.Contains(out, "Implement the API") || strings.Contains(out, "edit main.go") {
		t.Fatalf("chat mode must omit stale implement goals/actions: %q", out)
	}
	if !strings.Contains(out, "Use Rust") {
		t.Fatalf("corrections must still inject: %q", out)
	}
	if !strings.Contains(out, "ACTIVE CORRECTION") {
		t.Fatalf("expected ACTIVE CORRECTION banner: %q", out)
	}

	codeMsg := protocol.NewMessage(protocol.MessageTypeChat, "dm", protocol.AgentInfo{Name: "u"}, "continue the implement")
	codeMsg.Metadata = map[string]interface{}{MetadataConversationMode: ConversationModeCode}
	outCode := appendDurableConversationContext("BASE", envelope, codeMsg)
	if !strings.Contains(outCode, "Implement the API") {
		t.Fatalf("code mode should keep goals: %q", outCode)
	}
}

func TestBuildDialogueAssistantPrompt_isSlim(t *testing.T) {
	a := &Agent{Info: protocol.AgentInfo{
		Name: "Assistant", Type: protocol.AgentTypeAssistant, AIModel: "llama3.1", AIProvider: "ollama",
	}}
	msg := protocol.NewMessage(protocol.MessageTypeChat, "dm", protocol.AgentInfo{Name: "u"}, "plan a weekend trip")
	msg.Metadata = map[string]interface{}{MetadataConversationMode: ConversationModeChat}
	if !a.shouldUseDialogueAssistantPrompt(msg) {
		t.Fatal("expected dialogue prompt for chat trip planning")
	}
	prompt := a.buildDialogueAssistantPrompt(msg)
	if strings.Contains(prompt, "/create-repo-agent") || strings.Contains(prompt, "=== SYSTEM COMMANDS") {
		t.Fatal("dialogue prompt must not dump command encyclopedia")
	}
	if !strings.Contains(prompt, "Continue this conversation") {
		t.Fatal("expected continuity cue in dialogue prompt")
	}
}

type summaryStubHub struct {
	shouldRespondTestHub
	summary string
}

func (s summaryStubHub) GetChannelSessionSummary(string) string { return s.summary }
func (s summaryStubHub) GetChannelType(string) protocol.ChannelType {
	return protocol.ChannelTypeDM
}
