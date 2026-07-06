package slack

import (
	"strings"
	"testing"

	"github.com/camronwood/neural-junkie/internal/config"
	"github.com/camronwood/neural-junkie/internal/protocol"
	"github.com/camronwood/neural-junkie/internal/testutil"
)

type smokeHubStub struct {
	sent []*protocol.Message
}

func (s *smokeHubStub) SendMessage(msg *protocol.Message) error {
	s.sent = append(s.sent, msg)
	return nil
}
func (s *smokeHubStub) Subscribe(string) (chan *protocol.Message, error) { return nil, nil }
func (s *smokeHubStub) Unsubscribe(string, chan *protocol.Message)      {}
func (s *smokeHubStub) GetChannel(string) (*protocol.Channel, error)    { return nil, nil }
func (s *smokeHubStub) ListChannels() []*protocol.Channel               { return nil }
func (s *smokeHubStub) CreateChannelWithType(string, string, string, protocol.ChannelType, string) *protocol.Channel {
	return nil
}
func (s *smokeHubStub) AddAgentToChannel(string, string) error { return nil }
func (s *smokeHubStub) ResolveAgentID(string, string) (string, error) {
	return "agent-1", nil
}
func (s *smokeHubStub) SetChannelDisplay(string, string, string) error { return nil }

func TestValidateSmokeChannelRejectsIM(t *testing.T) {
	err := ValidateSmokeChannel(nil, "D01234567")
	if err == nil || !containsStr(err.Error(), "IM/DM") {
		t.Fatalf("expected IM rejection, got %v", err)
	}
}

func TestChannelNameAllowed(t *testing.T) {
	for _, name := range []string{"nj-smoke-test", "nj-smoke", "NJ-SMOKE-TEST"} {
		if !channelNameAllowed(name) {
			t.Fatalf("expected %q allowed", name)
		}
	}
	if channelNameAllowed("general") {
		t.Fatal("general should not be allowed")
	}
}

func TestSmokeInboundNoAgentRoute(t *testing.T) {
	hub := &smokeHubStub{}
	threads, err := NewThreadMap()
	if err != nil {
		t.Fatal(err)
	}
	binding := &Binding{
		SlackChannelID: "C1",
		NJChannel:      "slack:C1",
		AgentID:        "agent-1",
		Enabled:        true,
		Policy:         config.SlackPolicyMentionOnly,
	}
	msg, err := SmokeInbound(hub, binding, threads, "UBOT", smokeInboundMarker)
	if err != nil {
		t.Fatal(err)
	}
	if msg.Metadata[protocol.SlackMetaRouteAgentID] != nil {
		t.Fatalf("expected no agent route metadata, got %v", msg.Metadata[protocol.SlackMetaRouteAgentID])
	}
	if len(msg.Mentions) != 0 {
		t.Fatalf("expected no mentions, got %v", msg.Mentions)
	}
	if msg.Metadata["nj_smoke"] != true {
		t.Fatal("expected nj_smoke metadata")
	}
}

func TestSmokeAllowOutboundEnv(t *testing.T) {
	t.Setenv("SLACK_SMOKE_ALLOW", "")
	if SmokeAllowOutboundEnv() {
		t.Fatal("expected false")
	}
	t.Setenv("SLACK_SMOKE_ALLOW", "1")
	if !SmokeAllowOutboundEnv() {
		t.Fatal("expected true")
	}
}

func TestRunSmokeOutboundBlockedWithoutEnv(t *testing.T) {
	testutil.IsolateNeuralJunkieHome(t)
	cfg := config.DefaultConfig()
	cfg.Slack.Enabled = true
	cfg.Slack.AppToken = "xapp-test"
	cfg.Slack.BotToken = "xoxb-test"
	res := RunSmoke(t.Context(), cfg, nil, nil, nil, nil, nil, SmokeOptions{
		ChannelID:     "C1",
		Outbound:      true,
		AllowOutbound: true,
		EnvAllow:      false,
	})
	found := false
	for _, c := range res.Checks {
		if c.ID == "outbound_post" && c.Status == "fail" && strings.Contains(c.Detail, "SLACK_SMOKE_ALLOW") {
			found = true
		}
	}
	if !found {
		t.Fatalf("expected outbound env guard check, got %+v", res.Checks)
	}
}

func containsStr(s, sub string) bool {
	return len(s) >= len(sub) && (s == sub || len(sub) == 0 || indexStr(s, sub) >= 0)
}

func indexStr(s, sub string) int {
	for i := 0; i+len(sub) <= len(s); i++ {
		if s[i:i+len(sub)] == sub {
			return i
		}
	}
	return -1
}
