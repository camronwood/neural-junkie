package agent

import (
	"testing"

	"github.com/camronwood/neural-junkie/internal/protocol"
)

func TestClassifyTurnIntent_closureThanks(t *testing.T) {
	msg := protocol.NewMessage(protocol.MessageTypeChat, "dm-u-a", protocol.AgentInfo{ID: "u", Name: "User"}, "ok thanks")
	if got := classifyTurnIntent(msg, protocol.ChannelTypeDM, "assistant", nil); got != IntentClosure {
		t.Fatalf("got %v, want closure", got)
	}
}

func TestClassifyTurnIntent_alreadySaid(t *testing.T) {
	msg := protocol.NewMessage(protocol.MessageTypeChat, "dm-u-a", protocol.AgentInfo{ID: "u", Name: "User"}, "I know you said that already")
	if got := classifyTurnIntent(msg, protocol.ChannelTypeDM, "assistant", nil); got != IntentClosure {
		t.Fatalf("got %v, want closure", got)
	}
}

func TestClassifyTurnIntent_lowSignal(t *testing.T) {
	msg := protocol.NewMessage(protocol.MessageTypeChat, "dm-u-a", protocol.AgentInfo{ID: "u", Name: "User"}, "nice")
	if got := classifyTurnIntent(msg, protocol.ChannelTypeDM, "assistant", nil); got != IntentLowSignal {
		t.Fatalf("got %v, want casual/low_signal", got)
	}
}

func TestClassifyTurnIntent_greetingIsCasual(t *testing.T) {
	msg := protocol.NewMessage(protocol.MessageTypeChat, "general", protocol.AgentInfo{ID: "u", Name: "User"}, "what's up?")
	if got := classifyTurnIntent(msg, protocol.ChannelTypePublic, "assistant", nil); got != IntentLowSignal {
		t.Fatalf("got %v, want casual", got)
	}
}

func TestClassifyTurnIntent_collabNotLowSignal(t *testing.T) {
	msg := protocol.NewMessage(protocol.MessageTypeChat, "collab-1", protocol.AgentInfo{ID: "u", Name: "User"}, "ok")
	if got := classifyTurnIntent(msg, protocol.ChannelTypeCollaboration, "goexpert", nil); got != IntentSubstantive {
		t.Fatalf("got %v, want substantive on collab", got)
	}
}

func TestClassifyTurnIntent_substantiveDistance(t *testing.T) {
	msg := protocol.NewMessage(protocol.MessageTypeQuestion, "dm-u-a", protocol.AgentInfo{ID: "u", Name: "User"},
		"How far is it from Collinsville IL to St Louis MO?")
	if got := classifyTurnIntent(msg, protocol.ChannelTypeDM, "assistant", nil); got != IntentSubstantive {
		t.Fatalf("got %v, want substantive", got)
	}
}

func TestMaxHistoryForIntent_withSummary(t *testing.T) {
	if maxHistoryForIntent(IntentSubstantive, true) != 4 {
		t.Fatal("expected 4 history rows when summary present")
	}
}

func TestClassifyTurnIntent_shortQuestionsSubstantive(t *testing.T) {
	cases := []struct {
		content string
		want    TurnIntent
	}{
		{"What?", IntentSubstantive},
		{"can you see my workspace?", IntentSubstantive},
		{"yes thats what I said?", IntentSubstantive}, // len 22 < 30
		{"what information do you get when I send you a prompt?", IntentSubstantive},
		{"what model are you?", IntentMeta},
	}
	for _, tc := range cases {
		msg := protocol.NewMessage(protocol.MessageTypeChat, "dm-u-be", protocol.AgentInfo{ID: "u", Name: "User"}, tc.content)
		msg.Metadata = map[string]interface{}{MetadataConversationMode: ConversationModeChat}
		got := classifyTurnIntent(msg, protocol.ChannelTypeDM, "backend-id", nil)
		if got != tc.want {
			t.Fatalf("content %q: got %v want %v", tc.content, got, tc.want)
		}
	}
}

func TestUserAsksAboutWorkspaceVisibility(t *testing.T) {
	if !userAsksAboutWorkspaceVisibility("can you see my workspace?") {
		t.Fatal("expected workspace visibility")
	}
	if userAsksAboutWorkspaceVisibility("What?") {
		t.Fatal("expected false for confused follow-up")
	}
}

func TestConversationHistoryForIntent_biologyOllamaUserOnly(t *testing.T) {
	a := &Agent{
		Info: protocol.AgentInfo{
			Name:       "BiologyExpert",
			Type:       protocol.AgentTypeBiology,
			AIProvider: "ollama",
			AIModel:    "koesn/llama3-openbiollm-8b:latest",
		},
		Context: &ConversationContext{
			History: map[string][]*protocol.Message{
				"dm-u-b": {
					protocol.NewMessage(protocol.MessageTypeChat, "dm-u-b", protocol.AgentInfo{Name: "User", Type: "human"}, "first question"),
					protocol.NewMessage(protocol.MessageTypeChat, "dm-u-b", protocol.AgentInfo{Name: "BiologyExpert", Type: protocol.AgentTypeBiology}, "long prior answer"),
					protocol.NewMessage(protocol.MessageTypeChat, "dm-u-b", protocol.AgentInfo{Name: "User", Type: "human"}, "follow up"),
				},
			},
		},
	}
	current := protocol.NewMessage(protocol.MessageTypeQuestion, "dm-u-b", protocol.AgentInfo{Name: "User", Type: "human"}, "follow up")
	current.ID = "current-id"
	hist := a.conversationHistoryForIntent(current, IntentSubstantive)
	for _, m := range hist {
		if m != nil && m.From.Name == "BiologyExpert" {
			t.Fatalf("biology ollama history should be user-only, got agent line: %q", m.Content)
		}
	}
}
