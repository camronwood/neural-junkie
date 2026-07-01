package hub

import (
	"testing"

	"github.com/camronwood/neural-junkie/internal/protocol"
)

func TestHubChannelAllowsIDEAutoApprove(t *testing.T) {
	msg := &protocol.Message{Metadata: map[string]interface{}{"implementation_session": true}}
	if hubChannelAllowsIDEAutoApprove("chat-scenarios", msg) {
		t.Fatal("chat-scenarios must not auto-approve")
	}
	if !hubChannelAllowsIDEAutoApprove("implement-scenarios", msg) {
		t.Fatal("implement-scenarios should auto-approve")
	}
	if !hubChannelAllowsIDEAutoApprove("general", msg) {
		t.Fatal("normal channels should auto-approve when trust allows")
	}
}

func TestChannelMaintainsSessionSummary_regressionHarness(t *testing.T) {
	cases := []struct {
		channel string
		want    bool
	}{
		{"implement-scenarios", false},
		{"chat-scenarios", false},
		{"learning-scenarios", false},
		{"collab-scenarios", false},
		{"collab-scenarios-solo", false},
		{"parity-scenarios", false},
		{"dm-chatscenario-backendengineer", false},
		{"general", true},
		{"dm-user-assistant", true},
	}
	for _, tc := range cases {
		if got := ChannelMaintainsSessionSummary(tc.channel); got != tc.want {
			t.Fatalf("ChannelMaintainsSessionSummary(%q) = %v, want %v", tc.channel, got, tc.want)
		}
	}
}

func TestChannelMaintainsSessionSummary_channelTypeEligible(t *testing.T) {
	if !channelMaintainsSessionSummary(protocol.ChannelTypePublic, "general") {
		t.Fatal("general public channel should maintain session summary")
	}
	if channelMaintainsSessionSummary(protocol.ChannelTypePublic, "implement-scenarios") {
		t.Fatal("implement-scenarios should not maintain session summary")
	}
	if !channelMaintainsSessionSummary(protocol.ChannelTypeDM, "dm-user-assistant") {
		t.Fatal("user DM should maintain session summary")
	}
}
