package agent

import (
	"testing"
	"time"

	"github.com/camronwood/neural-junkie/internal/protocol"
)

func TestOmitMessageFromLLMHistory(t *testing.T) {
	sys := &protocol.Message{Type: protocol.MessageTypeSystemInfo, Content: "error: provider_error", From: protocol.AgentInfo{Name: "BiologyExpert", Type: protocol.AgentTypeBiology}}
	if !omitMessageFromLLMHistory(sys) {
		t.Fatal("system_info should be omitted")
	}

	user := protocol.NewMessage(protocol.MessageTypeQuestion, "dm", protocol.AgentInfo{Name: "Camron", Type: protocol.AgentTypeGeneral}, "hello")
	if omitMessageFromLLMHistory(user) {
		t.Fatal("user question should be kept")
	}

	bad := &protocol.Message{
		Type:    protocol.MessageTypeChat,
		Content: "Sorry, I encountered an error while generating a response.",
		From:    protocol.AgentInfo{Name: "BiologyExpert", Type: protocol.AgentTypeBiology},
	}
	if !omitMessageFromLLMHistory(bad) {
		t.Fatal("error boilerplate chat should be omitted")
	}

	delta := &protocol.Message{Type: protocol.MessageTypeStreamDelta, Content: "tok", From: protocol.AgentInfo{Name: "BiologyExpert"}}
	if !omitMessageFromLLMHistory(delta) {
		t.Fatal("stream delta should be omitted")
	}
}

func TestAgentRespondedToUser(t *testing.T) {
	user := &protocol.Message{ID: "u1", Type: protocol.MessageTypeQuestion, From: protocol.AgentInfo{Name: "Camron", Type: protocol.AgentTypeGeneral}}
	errReply := &protocol.Message{
		ID: "e1", Type: protocol.MessageTypeSystemInfo, ReplyTo: "u1",
		Content: "The model returned an empty reply.",
		From:    protocol.AgentInfo{ID: "bio1", Name: "BiologyExpert"},
	}
	history := []*protocol.Message{user, errReply}
	if !agentRespondedToUser(history, 0, "bio1", "BiologyExpert", "u1") {
		t.Fatal("system_info reply-to should count as responded")
	}

	history2 := []*protocol.Message{user, &protocol.Message{ID: "c1", Type: protocol.MessageTypeChat, From: protocol.AgentInfo{ID: "bio1", Name: "BiologyExpert"}, Content: "Hi there"}}
	if !agentRespondedToUser(history2, 0, "bio1", "BiologyExpert", "u1") {
		t.Fatal("chat should count as responded")
	}

	history3 := []*protocol.Message{user}
	if agentRespondedToUser(history3, 0, "bio1", "BiologyExpert", "u1") {
		t.Fatal("expected no response")
	}
}

func TestHistoryForGenerationExcludesCurrentAndTrims(t *testing.T) {
	var hist []*protocol.Message
	for i := 0; i < 15; i++ {
		hist = append(hist, protocol.NewMessage(protocol.MessageTypeQuestion, "dm",
			protocol.AgentInfo{Name: "Camron", Type: protocol.AgentTypeGeneral}, "msg"))
	}
	current := protocol.NewMessage(protocol.MessageTypeQuestion, "dm", protocol.AgentInfo{Name: "Camron"}, "current")
	current.ID = "current-id"
	hist = append(hist, current)

	out := historyForGeneration(hist, "current-id")
	if len(out) != MaxLLMHistoryMessages {
		t.Fatalf("want %d messages, got %d", MaxLLMHistoryMessages, len(out))
	}
	for _, m := range out {
		if m.ID == "current-id" {
			t.Fatal("current message should be excluded")
		}
	}
}

func TestRecentCompleteExchangesRetainsPairAndReferencedReply(t *testing.T) {
	user := protocol.AgentInfo{ID: "u", Name: "Camron", Type: "human"}
	assistant := protocol.AgentInfo{ID: "a", Name: "Assistant", Type: protocol.AgentTypeAssistant}
	u1 := protocol.NewMessage(protocol.MessageTypeChat, "dm", user, "original request")
	u1.ID = "u1"
	a1 := protocol.NewMessage(protocol.MessageTypeAnswer, "dm", assistant, "original answer")
	a1.ID, a1.ReplyTo = "a1", "u1"
	u2 := protocol.NewMessage(protocol.MessageTypeChat, "dm", user, "new request")
	u2.ID = "u2"
	a2 := protocol.NewMessage(protocol.MessageTypeAnswer, "dm", assistant, "new answer")
	a2.ID, a2.ReplyTo = "a2", "u2"
	current := protocol.NewMessage(protocol.MessageTypeChat, "dm", user, "expand your original answer")
	current.ID, current.ReplyTo = "u3", "a1"

	exchanges := recentCompleteExchanges([]*protocol.Message{u1, a1, u2, a2}, current, nil, 2)
	messages := messagesFromExchanges(exchanges)
	got := map[string]bool{}
	for _, message := range messages {
		got[message.ID] = true
	}
	for _, id := range []string{"u2", "a2", "a1"} {
		if !got[id] {
			t.Fatalf("referenced/latest exchange message %s missing: %+v", id, got)
		}
	}
}

func TestRecentCompleteExchangesExcludesSupersededInstruction(t *testing.T) {
	user := protocol.AgentInfo{ID: "u", Name: "Camron", Type: "human"}
	assistant := protocol.AgentInfo{ID: "a", Name: "Assistant", Type: protocol.AgentTypeAssistant}
	old := protocol.NewMessage(protocol.MessageTypeChat, "dm", user, "use Python")
	old.ID = "old"
	answer := protocol.NewMessage(protocol.MessageTypeAnswer, "dm", assistant, "I will use Python")
	answer.ID, answer.ReplyTo = "answer", "old"
	correction := protocol.NewMessage(protocol.MessageTypeChat, "dm", user, "use Rust instead")
	correction.ID = "correction"

	messages := messagesFromExchanges(recentCompleteExchanges(
		[]*protocol.Message{old, answer, correction}, nil, []string{"old"}, 10,
	))
	for _, message := range messages {
		if message.ID == "old" || message.ID == "answer" {
			t.Fatalf("superseded instruction or its stale reply remained in context: %s", message.ID)
		}
	}
}

func TestMessageTooOldForUnansweredReplay(t *testing.T) {
	old := &protocol.Message{Timestamp: time.Now().Add(-2 * time.Hour)}
	if !messageTooOldForUnansweredReplay(old) {
		t.Fatal("expected old")
	}
	fresh := &protocol.Message{Timestamp: time.Now().Add(-2 * time.Minute)}
	if messageTooOldForUnansweredReplay(fresh) {
		t.Fatal("expected fresh")
	}
}

func TestResolveContextScopeForChannel_DMFullToOutline(t *testing.T) {
	msg := &protocol.Message{Metadata: map[string]interface{}{"workspace_context": map[string]interface{}{"workspace_name": "proj"}}}
	if got := ResolveContextScopeForChannel(msg, protocol.ChannelTypeDM); got != ContextScopeOutline {
		t.Fatalf("DM full legacy scope want outline, got %q", got)
	}
	if got := ResolveContextScopeForChannel(msg, protocol.ChannelTypePublic); got != ContextScopeFull {
		t.Fatalf("general channel want full, got %q", got)
	}
}

func TestResolveContextScopeForChannel_DMImplementationToFocus(t *testing.T) {
	msg := &protocol.Message{
		Content: "add a settings modal with dark/light themes",
		Metadata: map[string]interface{}{
			MetadataContextScope: ContextScopeFull,
			"workspace_context":  map[string]interface{}{"workspace_name": "proj"},
		},
	}
	if got := ResolveContextScopeForChannel(msg, protocol.ChannelTypeDM); got != ContextScopeFocus {
		t.Fatalf("DM implementation want focus, got %q", got)
	}
}

func TestResolveContextScopeForChannel_DMWorkspaceDirectiveToFocus(t *testing.T) {
	msg := &protocol.Message{
		Content: "use the open workspace it has all the files you need",
		Metadata: map[string]interface{}{
			MetadataContextScope: ContextScopeOutline,
			"workspace_context":  map[string]interface{}{"workspace_name": "proj", "workspace_path": "/proj"},
		},
	}
	if got := ResolveContextScopeForChannel(msg, protocol.ChannelTypeDM); got != ContextScopeFocus {
		t.Fatalf("DM workspace directive want focus, got %q", got)
	}
}

func TestMessageNeedsWorkspaceFileLoad_workspaceDirective(t *testing.T) {
	a := &Agent{Info: protocol.AgentInfo{ID: "fe-1", Name: "FrontendEngineer", Type: protocol.AgentTypeFrontend}}
	msg := protocol.NewMessage(protocol.MessageTypeQuestion, "dm-u-fe", protocol.AgentInfo{Name: "camron"}, "use the open workspace it has all the files you need")
	if !messageNeedsWorkspaceFileLoad(a, msg) {
		t.Fatal("expected workspace directive to need file load")
	}
}
