package slack

import (
	"strings"
	"testing"

	"github.com/camronwood/neural-junkie/internal/protocol"
)

func TestShouldPostToSlack(t *testing.T) {
	b := &Binding{
		Enabled:   true,
		NJChannel: "slack:C1",
		AgentID:   "agent-1",
	}
	agentMsg := protocol.NewMessage(protocol.MessageTypeAnswer, "slack:C1", protocol.AgentInfo{ID: "agent-1"}, "hello")
	if !ShouldPostToSlack(agentMsg, b) {
		t.Fatal("expected agent answer to post")
	}
	wrongAgent := protocol.NewMessage(protocol.MessageTypeAnswer, "slack:C1", protocol.AgentInfo{ID: "other"}, "hi")
	if ShouldPostToSlack(wrongAgent, b) {
		t.Fatal("expected wrong agent skipped")
	}
	echo := protocol.NewMessage(protocol.MessageTypeChat, "slack:C1", protocol.AgentInfo{ID: "agent-1"}, "echo")
	echo.Metadata = map[string]interface{}{"source": "slack"}
	if ShouldPostToSlack(echo, b) {
		t.Fatal("expected slack-sourced echo skipped")
	}
	disabled := *b
	disabled.Enabled = false
	if ShouldPostToSlack(agentMsg, &disabled) {
		t.Fatal("expected disabled binding skipped")
	}
	fc := protocol.NewMessage(protocol.MessageTypeFileChange, "slack:C1", protocol.AgentInfo{ID: "agent-1"}, "")
	if !ShouldPostToSlack(fc, b) {
		t.Fatal("expected file change notice")
	}
	system := protocol.NewMessage(protocol.MessageTypeSystemInfo, "slack:C1", protocol.AgentInfo{ID: "agent-1"}, "sys")
	if ShouldPostToSlack(system, b) {
		t.Fatal("expected system message skipped")
	}
}

func TestFormatSlackText(t *testing.T) {
	long := strings.Repeat("x", 5000)
	out := FormatSlackText(protocol.NewMessage(protocol.MessageTypeAnswer, "c", protocol.AgentInfo{}, long))
	if !strings.Contains(out, "…") {
		t.Fatal("expected split marker for long content")
	}
	if len(out) < slackMaxText {
		t.Fatalf("expected combined length >= chunk size, got %d", len(out))
	}
	fc := FormatSlackText(protocol.NewMessage(protocol.MessageTypeFileChange, "c", protocol.AgentInfo{}, ""))
	if !strings.Contains(fc, "file change") {
		t.Fatalf("file change text: %q", fc)
	}
	ta := FormatSlackText(protocol.NewMessage(protocol.MessageTypeToolApproval, "c", protocol.AgentInfo{}, ""))
	if !strings.Contains(ta, "tool call") {
		t.Fatalf("tool approval text: %q", ta)
	}
}

func TestThreadTSForOutbound(t *testing.T) {
	threads := &ThreadMap{
		njToSlack: map[string]string{"thread-root": "1111.2222"},
	}
	b := &Binding{ReplyInThread: true}
	msg := protocol.NewMessage(protocol.MessageTypeAnswer, "slack:C1", protocol.AgentInfo{}, "ok")
	msg.ThreadID = "thread-root"
	msg.IsThreadReply = true
	if got := ThreadTSForOutbound(msg, threads, b); got != "1111.2222" {
		t.Fatalf("thread ts from map: got %q", got)
	}
	msg2 := protocol.NewMessage(protocol.MessageTypeAnswer, "slack:C1", protocol.AgentInfo{}, "ok")
	msg2.Metadata = map[string]interface{}{"slack_thread_ts": "9999.0001"}
	if got := ThreadTSForOutbound(msg2, threads, b); got != "9999.0001" {
		t.Fatalf("thread ts from metadata: got %q", got)
	}
	noThread := &Binding{ReplyInThread: false}
	if got := ThreadTSForOutbound(msg, threads, noThread); got != "" {
		t.Fatalf("expected empty when reply_in_thread false, got %q", got)
	}
}
