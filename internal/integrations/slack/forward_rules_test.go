package slack

import (
	"testing"

	"github.com/camronwood/neural-junkie/internal/protocol"
)

func TestEvaluateForwardRulesMention(t *testing.T) {
	in := InboundInput{ChannelID: "C1", Text: "hey <@UOWNER> can you look?"}
	rules := []ForwardRule{{
		ID: "mentions", Type: ForwardRuleMentionOfMe, Enabled: true,
		SlackChannelIDs: []string{"C1"},
	}}
	match, ok := EvaluateForwardRules(in, rules, "UOWNER")
	if !ok || match == nil {
		t.Fatal("expected mention match")
	}
	if match.RuleType != ForwardRuleMentionOfMe {
		t.Fatalf("type %q", match.RuleType)
	}
}

func TestEvaluateForwardRulesPrefix(t *testing.T) {
	in := InboundInput{ChannelID: "C9", Text: "nj: summarize this thread"}
	rules := []ForwardRule{{
		ID: "prefix", Type: ForwardRulePrefix, Enabled: true, Prefix: "nj:",
		SlackChannelIDs: []string{"*"},
	}}
	match, ok := EvaluateForwardRules(in, rules, "U1")
	if !ok || match == nil {
		t.Fatal("expected prefix match")
	}
	if match.StrippedText != "summarize this thread" {
		t.Fatalf("stripped %q", match.StrippedText)
	}
}

func TestEvaluateForwardRulesChannelFilter(t *testing.T) {
	in := InboundInput{ChannelID: "C2", Text: "hey <@U1>"}
	rules := []ForwardRule{{
		ID: "mentions", Type: ForwardRuleMentionOfMe, Enabled: true,
		SlackChannelIDs: []string{"C1"},
	}}
	if _, ok := EvaluateForwardRules(in, rules, "U1"); ok {
		t.Fatal("expected no match for unwatched channel")
	}
}

func TestChannelMatchesRuleWildcard(t *testing.T) {
	if !channelMatchesRule("C9", []string{"*"}) {
		t.Fatal("wildcard should match")
	}
}

func TestBuildForwardedContent(t *testing.T) {
	forward := &ForwardMatch{SourceChannelName: "eng", SourceAuthor: "Alice"}
	got := BuildForwardedContent(forward, "ship it")
	if got != "[Forwarded from #eng — Alice]\nship it" {
		t.Fatalf("got %q", got)
	}
}

func TestEmojiMatches(t *testing.T) {
	if !emojiMatches("robot_face", "robot_face") {
		t.Fatal("expected match")
	}
	if emojiMatches("thumbsup", "robot_face") {
		t.Fatal("expected no match")
	}
}

func TestInboxOutboundTargetForwarded(t *testing.T) {
	inbox := &InboxConfig{Enabled: true, NJChannel: "slack:inbox:U1", SlackDMChannelID: "D1", AgentID: "a1"}
	threads, _ := NewThreadMap()
	_ = threads.RegisterInboxReplyRoute("root-msg", "CENG", "100.100")

	msg := protocol.NewMessage(protocol.MessageTypeAnswer, "slack:inbox:U1", protocol.AgentInfo{ID: "a1"}, "reply")
	msg.ReplyTo = "root-msg"

	ch, ts := InboxOutboundTarget(msg, inbox, threads)
	if ch != "CENG" || ts != "100.100" {
		t.Fatalf("got channel=%q thread=%q", ch, ts)
	}
}

func TestInboxOutboundTargetDMDefault(t *testing.T) {
	inbox := &InboxConfig{Enabled: true, NJChannel: "slack:inbox:U1", SlackDMChannelID: "D1", AgentID: "a1"}
	threads, _ := NewThreadMap()
	_ = threads.RegisterChannelParent("D1", "200.200")

	msg := protocol.NewMessage(protocol.MessageTypeAnswer, "slack:inbox:U1", protocol.AgentInfo{ID: "a1"}, "reply")
	ch, ts := InboxOutboundTarget(msg, inbox, threads)
	if ch != "D1" || ts != "200.200" {
		t.Fatalf("got channel=%q thread=%q", ch, ts)
	}
}

func TestShouldPostInboxToSlack(t *testing.T) {
	inbox := &InboxConfig{Enabled: true, NJChannel: "slack:inbox:U1", AgentID: "a1"}
	agentMsg := protocol.NewMessage(protocol.MessageTypeAnswer, "slack:inbox:U1", protocol.AgentInfo{ID: "a1"}, "hi")
	if !ShouldPostInboxToSlack(agentMsg, inbox) {
		t.Fatal("expected agent answer to post")
	}
	echo := protocol.NewMessage(protocol.MessageTypeChat, "slack:inbox:U1", protocol.AgentInfo{ID: "slack:U1"}, "echo")
	echo.Metadata = map[string]interface{}{"source": "slack_inbox"}
	if ShouldPostInboxToSlack(echo, inbox) {
		t.Fatal("expected echo skipped")
	}
	errMsg := protocol.NewMessage(protocol.MessageTypeSystemInfo, "slack:inbox:U1", protocol.AgentInfo{ID: "a1"}, "Ollama is not running")
	errMsg.SetErrorMetadata("provider_unavailable", true)
	if !ShouldPostInboxToSlack(errMsg, inbox) {
		t.Fatal("expected generation error to post to Slack")
	}
	plainSys := protocol.NewMessage(protocol.MessageTypeSystemInfo, "slack:inbox:U1", protocol.AgentInfo{ID: "a1"}, "status")
	if ShouldPostInboxToSlack(plainSys, inbox) {
		t.Fatal("expected system info without error_code to skip")
	}
}
