package agent

import (
	"strings"
	"testing"

	"github.com/camronwood/neural-junkie/internal/protocol"
)

func TestUserAsksAboutModelIdentity(t *testing.T) {
	if !userAsksAboutModelIdentity("what model are you running?") {
		t.Fatal("expected model identity detection")
	}
	if userAsksAboutModelIdentity("remind me in 5 minutes") {
		t.Fatal("did not expect model identity detection")
	}
}

func TestClassifyTurnIntent_modelIdentityMeta(t *testing.T) {
	msg := protocol.NewMessage(protocol.MessageTypeQuestion, "slack:C1",
		protocol.AgentInfo{ID: "u", Name: "Camron"}, "what model are you running?")
	if got := classifyTurnIntent(msg, protocol.ChannelTypeCustom, "assistant", nil); got != IntentMeta {
		t.Fatalf("got %v, want meta", got)
	}
}

func TestUseCompactAssistantOllamaPrompt(t *testing.T) {
	a := &Agent{
		Info: protocol.AgentInfo{
			Name:       "Assistant",
			Type:       protocol.AgentTypeAssistant,
			AIProvider: "ollama",
			AIModel:    "qwen2.5:7b",
		},
	}
	msg := protocol.NewMessage(protocol.MessageTypeChat, "slack:C1",
		protocol.AgentInfo{ID: "u", Name: "User"}, "hello")
	if !a.useCompactAssistantOllamaPrompt(msg) {
		t.Fatal("expected compact assistant prompt for qwen2.5:7b")
	}
}

func TestBuildCompactAssistantOllamaPrompt_size(t *testing.T) {
	SetUserRulesLookup(func(string) string { return "Prefer short answers." })
	t.Cleanup(func() { SetUserRulesLookup(nil) })

	a := &Agent{
		Info: protocol.AgentInfo{
			Name:       "Assistant",
			Type:       protocol.AgentTypeAssistant,
			AIProvider: "ollama",
			AIModel:    "qwen2.5:7b",
		},
	}
	msg := protocol.NewMessage(protocol.MessageTypeChat, "slack:C1",
		protocol.AgentInfo{ID: "u", Name: "User", Type: protocol.AgentTypeGeneral}, "what model?")
	prompt := a.buildCompactAssistantOllamaPrompt(msg)
	if len(prompt) > 2500 {
		t.Fatalf("compact assistant prompt too long: %d bytes", len(prompt))
	}
	if !strings.Contains(prompt, "qwen2.5:7b") {
		t.Fatal("expected model in compact prompt")
	}
	if !strings.Contains(prompt, "USER-CONFIGURED RULES") {
		t.Fatal("compact assistant prompt should include capped user rules")
	}
}

func TestLooksLikeContextStackEcho_modelQuestionWithCode(t *testing.T) {
	msg := protocol.NewMessage(protocol.MessageTypeQuestion, "slack:C1",
		protocol.AgentInfo{ID: "u", Name: "Camron"}, "what model are you running?")
	reply := "Sure! ```python\ndef add(a,b): return a+b\n```\nNo meeting notes today.\nI run qwen2.5:7b."
	if !looksLikeContextStackEcho(msg, reply) {
		t.Fatal("expected context stack echo detection")
	}
}

func TestLooksLikeContextStackEcho_MeetingAnswerNotRetried(t *testing.T) {
	msg := protocol.NewMessage(protocol.MessageTypeQuestion, "general",
		protocol.AgentInfo{ID: "u", Name: "Camron"}, "do you see the PHOENIX TEAM MEETING notes from today?")
	reply := `Yes, I found the PHOENIX TEAM MEETING notes from May 19.

Summary:
- Calculation methodology updates
- Infrastructure architectural improvements
- Reader deployment strategy

Action items:
- Revise official system documentation
- Update reader task worker refactor project story details`
	if looksLikeContextStackEcho(msg, reply) {
		t.Fatal("meeting-note answers should not trigger compact retry")
	}
}

func TestMessageAsksAboutMeetings_andEmail(t *testing.T) {
	if !messageAsksAboutMeetings("do you have meeting notes from today?") {
		t.Fatal("expected meeting query")
	}
	if messageAsksAboutEmail("what model are you running?") {
		t.Fatal("did not expect email query")
	}
	if !messageAsksAboutEmail("any emails this week?") {
		t.Fatal("expected email query")
	}
}
