package slack

import (
	"testing"

	slackapi "github.com/slack-go/slack"
)

func TestShouldProcessHumanDMMessage(t *testing.T) {
	owner := "U_OWNER"
	bot := "U_BOT"
	if !shouldProcessHumanDMMessage(slackapi.Message{Msg: slackapi.Msg{User: "U2", Text: "hello"}}, owner, bot) {
		t.Fatal("expected peer message")
	}
	if shouldProcessHumanDMMessage(slackapi.Message{Msg: slackapi.Msg{User: owner, Text: "mine"}}, owner, bot) {
		t.Fatal("expected owner message skipped")
	}
	if shouldProcessHumanDMMessage(slackapi.Message{Msg: slackapi.Msg{User: bot, Text: "bot"}}, owner, bot) {
		t.Fatal("expected bot user skipped")
	}
	if shouldProcessHumanDMMessage(slackapi.Message{Msg: slackapi.Msg{User: "U2", Text: "x", BotID: "B1"}}, owner, bot) {
		t.Fatal("expected bot_id skipped")
	}
	if shouldProcessHumanDMMessage(slackapi.Message{Msg: slackapi.Msg{User: "U2", Text: ""}}, owner, bot) {
		t.Fatal("expected empty text skipped")
	}
	if shouldProcessHumanDMMessage(slackapi.Message{Msg: slackapi.Msg{User: "U2", Text: "edit", SubType: "message_changed"}}, owner, bot) {
		t.Fatal("expected subtype skipped")
	}
}
