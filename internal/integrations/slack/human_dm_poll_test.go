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

func TestHumanDMIsNoteToSelf(t *testing.T) {
	owner := "U_OWNER"
	tests := []struct {
		name      string
		peer      string
		members   []string
		wantSelf  bool
	}{
		{"jot solo", owner, []string{owner}, true},
		{"jot empty members", owner, nil, true},
		{"peer dm", "U_PEER", []string{owner, "U_PEER"}, false},
		{"owner user field but two members", owner, []string{owner, "U_PEER"}, false},
		{"single non-owner member", "U_PEER", []string{"U_PEER"}, false},
	}
	for _, tc := range tests {
		got := humanDMIsNoteToSelf(owner, tc.peer, tc.members)
		if got != tc.wantSelf {
			t.Fatalf("%s: got %v want %v", tc.name, got, tc.wantSelf)
		}
	}
}
