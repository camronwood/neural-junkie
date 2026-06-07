package slack

import (
	"testing"
	"time"
)

func baseHumanDMInbox() InboxConfig {
	return InboxConfig{
		Enabled:          true,
		OwnerSlackUserID: "U1",
		AgentID:          "agent-1",
		NJChannel:        "slack:inbox:U1",
		HumanDMAway: HumanDMAwayConfig{
			Enabled:          true,
			ScheduleEnabled:  true,
			ScheduleTimezone: "America/Los_Angeles",
			WorkHours:        DefaultWorkHours(),
		},
	}
}

func laTime(year int, month time.Month, day, hour, min int) time.Time {
	loc, err := time.LoadLocation("America/Los_Angeles")
	if err != nil {
		panic(err)
	}
	return time.Date(year, month, day, hour, min, 0, 0, loc)
}

func TestShouldMonitorHumanDMsManualAway(t *testing.T) {
	inbox := baseHumanDMInbox()
	inbox.HumanDMAway.AwayEnabled = true
	now := laTime(2026, time.June, 1, 10, 0)
	if !ShouldMonitorHumanDMs(inbox, true, now) {
		t.Fatal("expected manual away to monitor during work hours")
	}
}

func TestShouldMonitorHumanDMsScheduleInsideWorkHours(t *testing.T) {
	inbox := baseHumanDMInbox()
	now := laTime(2026, time.June, 2, 10, 0) // Tuesday 10am
	if ShouldMonitorHumanDMs(inbox, true, now) {
		t.Fatal("expected inside work hours to skip monitoring")
	}
}

func TestShouldMonitorHumanDMsScheduleOutsideWorkHours(t *testing.T) {
	inbox := baseHumanDMInbox()
	now := laTime(2026, time.June, 2, 20, 0) // Tuesday 8pm
	if !ShouldMonitorHumanDMs(inbox, true, now) {
		t.Fatal("expected outside work hours to monitor")
	}
}

func TestShouldMonitorHumanDMsWeekendDefaultAway(t *testing.T) {
	inbox := baseHumanDMInbox()
	now := laTime(2026, time.June, 7, 10, 0) // Sunday
	if !ShouldMonitorHumanDMs(inbox, true, now) {
		t.Fatal("expected weekend with no work hours to monitor")
	}
}

func TestShouldMonitorHumanDMsRequiresTokenAndInbox(t *testing.T) {
	inbox := baseHumanDMInbox()
	now := laTime(2026, time.June, 2, 20, 0)
	if ShouldMonitorHumanDMs(inbox, false, now) {
		t.Fatal("expected missing user token to skip")
	}
	inbox.Enabled = false
	if ShouldMonitorHumanDMs(inbox, true, now) {
		t.Fatal("expected disabled inbox to skip")
	}
}

func TestHumanDMReplyPrefix(t *testing.T) {
	cfg := HumanDMAwayConfig{ReplyPrefix: "Assistant (for %s)"}
	got := HumanDMReplyPrefix(cfg, "Camron")
	if got != "Assistant (for Camron): " {
		t.Fatalf("prefix: %q", got)
	}
	gotEmpty := HumanDMReplyPrefix(HumanDMAwayConfig{}, "")
	if gotEmpty != "Assistant (for you): " {
		t.Fatalf("default prefix: %q", gotEmpty)
	}
}

func TestShouldPollHumanDMsForwardMode(t *testing.T) {
	inbox := baseHumanDMInbox()
	inbox.HumanDMAway.ScheduleEnabled = false
	inbox.ForwardEnabled = true
	now := laTime(2026, time.June, 7, 11, 30) // Sunday, away off
	if ShouldMonitorHumanDMs(inbox, true, now) {
		t.Fatal("expected away monitoring off")
	}
	if !ShouldPollHumanDMs(inbox, true, now) {
		t.Fatal("expected forward mode to poll human DMs")
	}
	if ShouldAutoReplyHumanDMs(inbox, true, now) {
		t.Fatal("forward mode should not auto-route to agent")
	}
}

func TestBuildHumanDMInboxMessageSkipsAgentRouteWhenForwardOnly(t *testing.T) {
	inbox := &InboxConfig{
		Enabled:          true,
		OwnerSlackUserID: "U1",
		AgentID:          "agent-1",
		NJChannel:        "slack:inbox:U1",
	}
	in := InboundInput{ChannelID: "D_PEER", UserID: "U2", UserName: "Demo User", Text: "you there?", SlackTS: "1.2"}
	threads, _ := NewThreadMap()
	msg := BuildHumanDMInboxMessage(in, inbox, threads, "Demo User", false, "slack:inbox:U1:U2")
	if msg.SlackRoutedAgentID() != "" {
		t.Fatalf("expected no agent route in forward-only mode, got %q", msg.SlackRoutedAgentID())
	}
	if !msg.IsSlackManualInboxReply() {
		t.Fatal("expected manual reply flag")
	}
}

func TestHumanDMMonitoringStatus(t *testing.T) {
	inbox := baseHumanDMInbox()
	now := laTime(2026, time.June, 2, 10, 0)
	if got := HumanDMMonitoringStatus(inbox, true, now); got != "inside_work_hours" {
		t.Fatalf("status: %q", got)
	}
	inbox.HumanDMAway.AwayEnabled = true
	if got := HumanDMMonitoringStatus(inbox, true, now); got != "monitoring_active" {
		t.Fatalf("away status: %q", got)
	}
	inbox.HumanDMAway.Enabled = false
	if got := HumanDMMonitoringStatus(inbox, true, now); got != "disabled" {
		t.Fatalf("disabled: %q", got)
	}
}
