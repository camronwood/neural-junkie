package slack

import (
	"testing"

	slackapi "github.com/slack-go/slack"
)

func TestDisplayNameFromUser(t *testing.T) {
	u := &slackapi.User{
		Name:     "cannonwood",
		RealName: "Camron Wood",
		Profile: slackapi.UserProfile{
			DisplayName: "Camron",
			RealName:    "Camron Wood",
		},
	}
	display, handle := DisplayNameFromUser(u)
	if display != "Camron" || handle != "cannonwood" {
		t.Fatalf("got display=%q handle=%q", display, handle)
	}
}

func TestFormatSlackSenderLabel(t *testing.T) {
	if got := FormatSlackSenderLabel("Camron Wood", "cannonwood"); got != "Camron Wood (@cannonwood)" {
		t.Fatalf("got %q", got)
	}
	if got := FormatSlackSenderLabel("Camron", "Camron"); got != "Camron" {
		t.Fatalf("got %q", got)
	}
}
