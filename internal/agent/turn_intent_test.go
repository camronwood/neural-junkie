package agent

import (
	"testing"

	"github.com/camronwood/neural-junkie/internal/intent"
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
	if maxHistoryForIntent(IntentSubstantive, true) != 12 {
		t.Fatal("expected 12 history rows when summary present (summary must not shrink window)")
	}
	if maxHistoryForIntent(IntentSubstantive, false) != 12 {
		t.Fatal("expected 12 history rows without summary")
	}
	if maxHistoryForIntent(IntentTask, false) != 16 {
		t.Fatal("expected 16 for task")
	}
	if maxHistoryForIntent(IntentLowSignal, false) != 4 {
		t.Fatal("expected 4 for casual")
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

func TestConversationHistoryForIntent_biologyOllamaKeepsExchanges(t *testing.T) {
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
	foundAgent := false
	for _, m := range hist {
		if m != nil && m.From.Name == "BiologyExpert" {
			foundAgent = true
		}
	}
	if !foundAgent {
		t.Fatal("dialogue-first history must keep assistant exchanges, not user-only")
	}
}

func TestClassifyTurnIntent_openThreadFollowUpIsSubstantive(t *testing.T) {
	user := protocol.AgentInfo{ID: "u", Name: "User", Type: "human"}
	asst := protocol.AgentInfo{ID: "assistant", Name: "Assistant", Type: protocol.AgentTypeAssistant}
	history := []*protocol.Message{
		protocol.NewMessage(protocol.MessageTypeChat, "dm-u-a", user, "Help me plan a trip to Lisbon"),
		protocol.NewMessage(protocol.MessageTypeAnswer, "dm-u-a", asst, "Here are some Lisbon ideas"),
	}
	history[1].From.ID = "assistant"
	msg := protocol.NewMessage(protocol.MessageTypeChat, "dm-u-a", user, "I have turned on websearch now")
	msg.Metadata = map[string]interface{}{MetadataConversationMode: ConversationModeChat}
	if got := classifyTurnIntent(msg, protocol.ChannelTypeDM, "assistant", history); got != IntentSubstantive {
		t.Fatalf("open-thread capability notice got %v, want substantive", got)
	}
	cold := protocol.NewMessage(protocol.MessageTypeChat, "dm-u-a", user, "I have turned on websearch now")
	cold.Metadata = map[string]interface{}{MetadataConversationMode: ConversationModeChat}
	if got := classifyTurnIntent(cold, protocol.ChannelTypeDM, "assistant", nil); got != IntentLowSignal {
		t.Fatalf("cold capability notice without history got %v, want casual", got)
	}
}

// TestClassifyTurnIntentMusicGenerationIsTask documents stamp-first music turns:
// a stamped ActionMusic decision's InteractionTask drives IntentTask, rather than
// natural-language matching on the message content.
func TestClassifyTurnIntentMusicGenerationIsTask(t *testing.T) {
	msg := protocol.NewMessage(protocol.MessageTypeQuestion, "music", protocol.AgentInfo{Name: "Camron"}, "Can you generate me a song?")
	msg.Metadata = map[string]interface{}{MetadataConversationMode: ConversationModeChat}
	if err := protocol.StampTurnDecision(msg, intent.TurnDecision{
		SchemaVersion: intent.SchemaVersion, Interaction: intent.InteractionTask,
		RequestedAction: intent.ActionMusic, Action: intent.ActionMusic,
		Mutation: intent.MutationExternal, Confidence: 0.95, Source: intent.SourceLocalModel,
	}); err != nil {
		t.Fatal(err)
	}
	if got := classifyTurnIntent(msg, protocol.ChannelTypePublic, "music-1", nil); got != IntentTask {
		t.Fatalf("music generation should be IntentTask, got %v", got)
	}
}
