package slack

import "testing"

func TestStripSlackMentionMarkup(t *testing.T) {
	got := StripSlackMentionMarkup("<@U0B5MLY2N2E> you there?")
	if got != "you there?" {
		t.Fatalf("got %q", got)
	}
}

func TestBuildInboxMessageStripsSlackMentions(t *testing.T) {
	inbox := &InboxConfig{
		Enabled:   true,
		AgentID:   "agent-1",
		NJChannel: "slack:inbox:U1",
	}
	in := InboundInput{
		ChannelID: "C1",
		UserID:    "U2",
		UserName:  "Alice",
		Text:      "<@U0B5MLY2N2E> what model are you running?",
		SlackTS:   "111.111",
	}
	forward := &ForwardMatch{
		RuleType:          ForwardRuleMentionOfMe,
		SourceChannelID:   "C1",
		SourceChannelName: "qwen-test",
		SourceTS:          "111.111",
		SourceAuthor:      "Alice",
	}
	threads, _ := NewThreadMap()
	msg := BuildInboxMessage(in, inbox, threads, forward)
	if contains(msg.Content, "<@U0B5MLY2N2E>") {
		t.Fatalf("content still has slack markup: %q", msg.Content)
	}
	if !contains(msg.Content, "what model are you running?") {
		t.Fatalf("content missing question: %q", msg.Content)
	}
}
