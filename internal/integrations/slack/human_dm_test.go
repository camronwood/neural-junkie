package slack

import (
	"testing"

	"github.com/camronwood/neural-junkie/internal/protocol"
)

func TestBuildHumanDMInboxMessage(t *testing.T) {
	inbox := &InboxConfig{
		Enabled:            true,
		OwnerSlackUserID:   "U1",
		OwnerSlackUserName: "Camron",
		AgentID:            "agent-1",
		NJChannel:          "slack:inbox:U1",
	}
	in := InboundInput{
		ChannelID: "D_PEER",
		UserID:    "U2",
		UserName:  "Alice",
		Text:      "hey are you there?",
		SlackTS:   "123.456",
		ThreadTS:  "100.100",
	}
	threads, _ := NewThreadMap()
	msg := BuildHumanDMInboxMessage(in, inbox, threads, "Alice", true, "slack:inbox:U1:U2")
	if msg.Channel != "slack:inbox:U1:U2" {
		t.Fatalf("channel %q", msg.Channel)
	}
	if msg.Metadata["source"] != "slack_human_dm" {
		t.Fatalf("source: %v", msg.Metadata["source"])
	}
	if msg.Metadata[protocol.SlackMetaHumanDM] != true {
		t.Fatal("expected slack_human_dm flag")
	}
	if msg.Metadata[protocol.SlackMetaReplyChannelID] != "D_PEER" {
		t.Fatalf("reply channel: %v", msg.Metadata[protocol.SlackMetaReplyChannelID])
	}
	if msg.SlackReplyThreadTS() != "" {
		t.Fatalf("human DM should not set reply thread ts, got %v", msg.Metadata[protocol.SlackMetaReplyThreadTS])
	}
	if msg.Content == "" || msg.Content[:12] != "[DM from Ali" {
		t.Fatalf("content header: %q", msg.Content)
	}
}

func TestInboxOutboundHumanDM(t *testing.T) {
	threads, _ := NewThreadMap()
	inbox := &InboxConfig{Enabled: true, OwnerSlackUserID: "U1", NJChannel: "slack:inbox:U1"}
	_ = threads.RegisterHumanDMReplyRoute("thread-1", "D_PEER", "111.222")

	msg := protocol.NewMessage(protocol.MessageTypeAnswer, "slack:inbox:U1", protocol.AgentInfo{ID: "agent-1"}, "reply")
	msg.ThreadID = "thread-1"
	if !InboxOutboundHumanDM(msg, threads, inbox) {
		t.Fatal("expected human DM outbound")
	}
	_ = threads.RegisterInboxReplyRoute("bot-thread", "D_BOT", "333.444")
	botMsg := protocol.NewMessage(protocol.MessageTypeAnswer, "slack:inbox:U1", protocol.AgentInfo{ID: "agent-1"}, "bot reply")
	botMsg.ThreadID = "bot-thread"
	if InboxOutboundHumanDM(botMsg, threads, inbox) {
		t.Fatal("expected bot inbox route to skip human DM path")
	}
	peerMsg := protocol.NewMessage(protocol.MessageTypeChat, "slack:inbox:U1:U2", protocol.AgentInfo{ID: "user-1", Type: protocol.AgentTypeGeneral}, "hey")
	if !InboxOutboundHumanDM(peerMsg, threads, inbox) {
		t.Fatal("expected peer inbox channel to use human DM outbound")
	}
}

func TestFormatHumanDMOutboundText(t *testing.T) {
	inbox := &InboxConfig{
		OwnerSlackUserName: "Camron",
		HumanDMAway:        HumanDMAwayConfig{ReplyPrefix: "Assistant (for %s)"},
	}
	msg := protocol.NewMessage(protocol.MessageTypeAnswer, "slack:inbox:U1", protocol.AgentInfo{ID: "agent-1"}, "On it.")
	got := FormatHumanDMOutboundText(msg, inbox)
	want := "Assistant (for Camron): On it."
	if got != want {
		t.Fatalf("got %q want %q", got, want)
	}
}
