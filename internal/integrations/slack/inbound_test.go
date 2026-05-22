package slack

import (
	"testing"

	"github.com/camronwood/neural-junkie/internal/config"
)

func TestShouldIgnoreInbound(t *testing.T) {
	if !ShouldIgnoreInbound(InboundInput{BotID: "B1", Text: "hi"}, "") {
		t.Fatal("expected bot message ignored")
	}
	if !ShouldIgnoreInbound(InboundInput{UserID: "U1", Text: "hi"}, "U1") {
		t.Fatal("expected self message ignored")
	}
	if ShouldIgnoreInbound(InboundInput{UserID: "U2", Text: "hello"}, "B1") {
		t.Fatal("expected human message accepted")
	}
}

func TestShouldTriggerAgentMentionOnly(t *testing.T) {
	b := &Binding{Enabled: true, Policy: config.SlackPolicyMentionOnly}
	in := InboundInput{Text: "hey <@B123> what is rust?"}
	if !ShouldTriggerAgent(in, b, "B123") {
		t.Fatal("expected mention trigger")
	}
	in2 := InboundInput{Text: "hey what is rust?"}
	if ShouldTriggerAgent(in2, b, "B123") {
		t.Fatal("expected no trigger without mention")
	}
}

func TestShouldIgnoreInboundSubtypes(t *testing.T) {
	for _, sub := range []string{"bot_message", "message_changed", "message_deleted"} {
		if !ShouldIgnoreInbound(InboundInput{Subtype: sub, Text: "x"}, "") {
			t.Fatalf("subtype %q should be ignored", sub)
		}
	}
}

func TestShouldIgnoreInboundEmptyText(t *testing.T) {
	if !ShouldIgnoreInbound(InboundInput{UserID: "U1", Text: "   "}, "") {
		t.Fatal("empty text should be ignored")
	}
}

func TestShouldTriggerAgentPolicies(t *testing.T) {
	always := &Binding{Enabled: true, Policy: config.SlackPolicyAlways}
	if !ShouldTriggerAgent(InboundInput{Text: "hi"}, always, "") {
		t.Fatal("always policy")
	}
	questions := &Binding{Enabled: true, Policy: config.SlackPolicyQuestions}
	if !ShouldTriggerAgent(InboundInput{Text: "how do I deploy?"}, questions, "") {
		t.Fatal("questions policy")
	}
	if ShouldTriggerAgent(InboundInput{Text: "thanks"}, questions, "") {
		t.Fatal("non-question should not trigger")
	}
	disabled := &Binding{Enabled: false, Policy: config.SlackPolicyAlways}
	if ShouldTriggerAgent(InboundInput{Text: "hi"}, disabled, "") {
		t.Fatal("disabled binding")
	}
}

func TestStripBotMention(t *testing.T) {
	got := StripBotMention("<@U_BOT>  what is Go?", "U_BOT")
	if got != "what is Go?" {
		t.Fatalf("got %q", got)
	}
}

func TestBuildHubMessageThreadParentNJID(t *testing.T) {
	b := &Binding{Enabled: true, NJChannel: "slack:C1", AgentID: "a1", Policy: config.SlackPolicyAlways}
	tm := &ThreadMap{
		njMessageTS: map[string]string{"parent-nj": "10.0"},
	}
	in := InboundInput{
		ChannelID: "C1",
		UserID:    "U1",
		UserName:  "Camron",
		Text:      "follow up",
		SlackTS:   "10.1",
		ThreadTS:  "10.0",
	}
	msg := BuildHubMessage(in, b, tm, "")
	if !msg.IsThreadReply || msg.ThreadID != "parent-nj" {
		t.Fatalf("thread id: got %q isReply=%v", msg.ThreadID, msg.IsThreadReply)
	}
}

func TestBuildHubMessageThread(t *testing.T) {
	b := &Binding{Enabled: true, NJChannel: "slack:C1", AgentID: "a1", Policy: config.SlackPolicyAlways}
	tm := &ThreadMap{
		roots:     map[string]map[string]string{"C1": {"10.0": "root-msg-id"}},
		njToSlack: map[string]string{"root-msg-id": "10.0"},
	}
	in := InboundInput{
		ChannelID: "C1",
		UserID:    "U1",
		UserName:  "Camron",
		Text:      "follow up",
		SlackTS:   "10.1",
		ThreadTS:  "10.0",
	}
	msg := BuildHubMessage(in, b, tm, "")
	if !msg.IsThreadReply || msg.ThreadID != "root-msg-id" || msg.ReplyTo != "10.1" {
		t.Fatalf("thread fields: id=%q reply=%q isReply=%v", msg.ThreadID, msg.ReplyTo, msg.IsThreadReply)
	}
}

func TestBuildHubMessageMentions(t *testing.T) {
	b := &Binding{
		Enabled:   true,
		NJChannel: "slack:C1",
		AgentID:   "agent-1",
		Policy:    config.SlackPolicyMentionOnly,
	}
	threads, _ := NewThreadMap()
	in := InboundInput{
		ChannelID:    "C1",
		UserID:       "U1",
		UserName:     "Camron",
		Text:         "<@B1> help",
		SlackTS:      "1.0",
		IsAppMention: true,
	}
	msg := BuildHubMessage(in, b, threads, "B1")
	if len(msg.Mentions) != 1 || msg.Mentions[0] != "agent-1" {
		t.Fatalf("mentions: %v", msg.Mentions)
	}
	if msg.Metadata["source"] != "slack" {
		t.Fatalf("metadata: %v", msg.Metadata)
	}
	if msg.Metadata["slack_user_display_name"] != "Camron" {
		t.Fatalf("display name: %v", msg.Metadata["slack_user_display_name"])
	}
	if msg.Metadata["slack_app_mention"] != true {
		t.Fatalf("expected slack_app_mention metadata")
	}
	if msg.Metadata["slack_route_agent_id"] != "agent-1" {
		t.Fatalf("route agent: %v", msg.Metadata["slack_route_agent_id"])
	}
}

func TestBuildHubMessageAlwaysRoutesWithoutMentions(t *testing.T) {
	b := &Binding{
		Enabled:   true,
		NJChannel: "slack:C1",
		AgentID:   "agent-1",
		Policy:    config.SlackPolicyAlways,
	}
	threads, _ := NewThreadMap()
	in := InboundInput{
		ChannelID:      "C1",
		UserID:         "U1",
		UserName:       "Camron Wood (@cannonwood)",
		SlackUsername:  "cannonwood",
		Text:           "thanks",
		SlackTS:        "1.0",
	}
	msg := BuildHubMessage(in, b, threads, "B1")
	if len(msg.Mentions) != 0 {
		t.Fatalf("always policy should not set mentions for plain text: %v", msg.Mentions)
	}
	if msg.Metadata["slack_route_agent_id"] != "agent-1" {
		t.Fatalf("route agent: %v", msg.Metadata["slack_route_agent_id"])
	}
	if msg.Metadata["slack_app_mention"] == true {
		t.Fatal("plain text should not be slack_app_mention")
	}
}
