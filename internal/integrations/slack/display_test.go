package slack

import "testing"

func TestFormatSlackChannelDisplayName(t *testing.T) {
	if got := FormatSlackChannelDisplayName("cursor-test"); got != "#cursor-test" {
		t.Fatalf("got %q", got)
	}
	if got := FormatSlackChannelDisplayName("#neural-junkie"); got != "#neural-junkie" {
		t.Fatalf("got %q", got)
	}
}

func TestSlackChannelDescription(t *testing.T) {
	if got := SlackChannelDescription("cursor-test"); got != "Slack: #cursor-test" {
		t.Fatalf("got %q", got)
	}
}
