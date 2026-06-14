package slack

import (
	"testing"

	slackapi "github.com/slack-go/slack"
)

func TestShouldProcessPolledDM(t *testing.T) {
	if !shouldProcessPolledDM(slackapi.Message{Msg: slackapi.Msg{User: "U1", Text: "hello"}}, "B1") {
		t.Fatal("expected user message")
	}
	if shouldProcessPolledDM(slackapi.Message{Msg: slackapi.Msg{User: "B1", Text: "bot says hi"}}, "B1") {
		t.Fatal("expected bot user skipped")
	}
	if shouldProcessPolledDM(slackapi.Message{Msg: slackapi.Msg{User: "U1", Text: "x", BotID: "B99"}}, "B1") {
		t.Fatal("expected bot_id skipped")
	}
	if shouldProcessPolledDM(slackapi.Message{Msg: slackapi.Msg{User: "U1", Text: ""}}, "B1") {
		t.Fatal("expected empty text skipped")
	}
	if shouldProcessPolledDM(slackapi.Message{Msg: slackapi.Msg{User: "U1", Text: "edit", SubType: "message_changed"}}, "B1") {
		t.Fatal("expected subtype skipped")
	}
}

func TestIsMissingScopeErr(t *testing.T) {
	if !isMissingScopeErr(errString("missing_scope: im:history")) {
		t.Fatal("expected missing_scope")
	}
}

func TestIsChannelNotFoundErr(t *testing.T) {
	if !isChannelNotFoundErr(errString("channel_not_found")) {
		t.Fatal("expected channel_not_found")
	}
}

type errString string

func (e errString) Error() string { return string(e) }
