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
