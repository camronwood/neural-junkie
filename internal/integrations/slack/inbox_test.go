package slack

import (
	"context"
	"errors"
	"testing"

	"github.com/camronwood/neural-junkie/internal/protocol"
)

func TestIsIMChannel(t *testing.T) {
	if !IsIMChannel("D123") {
		t.Fatal("expected DM channel")
	}
	if IsIMChannel("C123") {
		t.Fatal("expected channel not IM")
	}
}

func TestInboxOwnerAllowed(t *testing.T) {
	inbox := &InboxConfig{Enabled: true, OwnerSlackUserID: "U1"}
	if !InboxOwnerAllowed(inbox, "U1") {
		t.Fatal("owner should be allowed")
	}
	if InboxOwnerAllowed(inbox, "U2") {
		t.Fatal("non-owner should be denied")
	}
}

func TestBuildInboxMessageDirectDM(t *testing.T) {
	inbox := &InboxConfig{
		Enabled:          true,
		OwnerSlackUserID: "U1",
		AgentID:          "agent-1",
		NJChannel:        "slack:inbox:U1",
	}
	in := InboundInput{
		ChannelID: "D999",
		UserID:    "U1",
		UserName:  "Camron",
		Text:      "hello from slack",
		SlackTS:   "123.456",
	}
	threads, _ := NewThreadMap()
	msg := BuildInboxMessage(in, inbox, threads, nil, "")
	if msg.Channel != "slack:inbox:U1" {
		t.Fatalf("channel %q", msg.Channel)
	}
	if msg.Metadata[protocol.SlackMetaRouteAgentID] != "agent-1" {
		t.Fatalf("route agent: %v", msg.Metadata[protocol.SlackMetaRouteAgentID])
	}
	if msg.Metadata[protocol.SlackMetaReplyChannelID] != "D999" {
		t.Fatalf("reply channel: %v", msg.Metadata[protocol.SlackMetaReplyChannelID])
	}
	if msg.Metadata[protocol.SlackMetaInbox] != true {
		t.Fatal("expected slack_inbox metadata")
	}
}

func TestReconcileInboxAgentID(t *testing.T) {
	useTempHomeDir(t)
	store, err := NewInboxStore()
	if err != nil {
		t.Fatal(err)
	}
	_, err = store.Save(InboxConfig{
		Enabled:          true,
		OwnerSlackUserID: "U1",
		AgentID:          "old-id",
		AgentName:        "Assistant",
	})
	if err != nil {
		t.Fatal(err)
	}
	hub := &reconcileMockHub{resolved: "new-id"}
	ReconcileInboxAgentID(store, hub)
	cfg := store.Get()
	if cfg.AgentID != "new-id" {
		t.Fatalf("agent_id = %q want new-id", cfg.AgentID)
	}
}

type reconcileMockHub struct {
	resolved string
}

func (r *reconcileMockHub) SendMessage(msg *protocol.Message) error { return nil }
func (r *reconcileMockHub) Subscribe(channelName string) (chan *protocol.Message, error) {
	return make(chan *protocol.Message), nil
}
func (r *reconcileMockHub) Unsubscribe(channelName string, ch chan *protocol.Message) {
}
func (r *reconcileMockHub) ResolveAgentID(agentID, agentName string) (string, error) {
	if agentName == "Assistant" || agentID == "old-id" {
		return r.resolved, nil
	}
	return agentID, nil
}
func (r *reconcileMockHub) GetChannel(name string) (*protocol.Channel, error) {
	return nil, errors.New("not found")
}
func (r *reconcileMockHub) ListChannels() []*protocol.Channel { return nil }
func (r *reconcileMockHub) CreateChannelWithType(name, description, project string, channelType protocol.ChannelType, createdBy string) *protocol.Channel {
	return nil
}
func (r *reconcileMockHub) AddAgentToChannel(agentID, channelName string) error { return nil }
func (r *reconcileMockHub) SetChannelDisplay(name, displayName, description string) error {
	return nil
}

func TestBuildInboxMessageForwarded(t *testing.T) {
	inbox := &InboxConfig{
		Enabled:   true,
		AgentID:   "agent-1",
		NJChannel: "slack:inbox:U1",
	}
	in := InboundInput{
		ChannelID: "C1",
		UserID:    "U2",
		UserName:  "Alice",
		Text:      "please review",
		SlackTS:   "111.111",
		ThreadTS:  "100.100",
	}
	forward := &ForwardMatch{
		RuleType:          ForwardRuleMentionOfMe,
		SourceChannelID:   "C1",
		SourceChannelName: "eng",
		SourceTS:          "111.111",
		SourceThreadTS:    "100.100",
		SourceAuthor:      "Alice",
	}
	threads, _ := NewThreadMap()
	msg := BuildInboxMessage(in, inbox, threads, forward, "")
	if msg.Metadata[protocol.SlackMetaReplyChannelID] != "C1" {
		t.Fatalf("reply channel: %v", msg.Metadata[protocol.SlackMetaReplyChannelID])
	}
	if msg.Metadata[protocol.SlackMetaReplyThreadTS] != "100.100" {
		t.Fatalf("reply thread: %v", msg.Metadata[protocol.SlackMetaReplyThreadTS])
	}
	if msg.Metadata[protocol.SlackMetaForwardRule] != string(ForwardRuleMentionOfMe) {
		t.Fatalf("forward rule: %v", msg.Metadata[protocol.SlackMetaForwardRule])
	}
	if !contains(msg.Content, "[Forwarded from #eng — Alice]") {
		t.Fatalf("content: %q", msg.Content)
	}
}

func contains(s, sub string) bool {
	return len(s) >= len(sub) && (s == sub || len(sub) == 0 || indexOf(s, sub) >= 0)
}

func indexOf(s, sub string) int {
	for i := 0; i+len(sub) <= len(s); i++ {
		if s[i:i+len(sub)] == sub {
			return i
		}
	}
	return -1
}

func TestNJInboxChannelName(t *testing.T) {
	if got := NJInboxChannelName("U1"); got != "slack:inbox:U1" {
		t.Fatalf("got %q", got)
	}
}

func TestNJInboxPeerChannelName(t *testing.T) {
	got := NJInboxPeerChannelName("U1", "U2")
	if got != "slack:inbox:U1:U2" {
		t.Fatalf("got %q", got)
	}
	if !IsInboxPeerHubChannel(got, "U1") {
		t.Fatal("expected peer channel match")
	}
}

func TestEnsureInboxPeerChannel(t *testing.T) {
	hub := &reconcileMockHub{resolved: "agent-1"}
	inbox := InboxConfig{
		Enabled:          true,
		OwnerSlackUserID: "U1",
		AgentID:          "agent-1",
		AgentName:        "Assistant",
		NJChannel:        "slack:inbox:U1",
	}
	name, created, err := EnsureInboxPeerChannel(context.Background(), hub, nil, inbox, "U2", "Demo User")
	if err != nil {
		t.Fatal(err)
	}
	if name != "slack:inbox:U1:U2" {
		t.Fatalf("channel %q", name)
	}
	if !created {
		t.Fatal("expected created")
	}
}

func TestBuildInboxMessagePeerChannelOverride(t *testing.T) {
	inbox := &InboxConfig{Enabled: true, AgentID: "agent-1", NJChannel: "slack:inbox:U1"}
	in := InboundInput{ChannelID: "D1", UserID: "U2", Text: "hi", SlackTS: "1.1"}
	threads, _ := NewThreadMap()
	msg := BuildInboxMessage(in, inbox, threads, nil, "slack:inbox:U1:U2")
	if msg.Channel != "slack:inbox:U1:U2" {
		t.Fatalf("channel %q", msg.Channel)
	}
}
