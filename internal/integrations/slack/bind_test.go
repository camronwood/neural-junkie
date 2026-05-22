package slack

import (
	"context"
	"errors"
	"testing"

	"github.com/camronwood/neural-junkie/internal/config"
	"github.com/camronwood/neural-junkie/internal/protocol"
)

type mockHub struct {
	channels      map[string]*protocol.Channel
	addAgentCalls []struct{ agentID, channel string }
	ensureCalls   []struct{ agentID, channel string }
	createCalled  bool
}

func (m *mockHub) SendMessage(msg *protocol.Message) error { return nil }

func (m *mockHub) Subscribe(channelName string) (chan *protocol.Message, error) {
	return make(chan *protocol.Message), nil
}

func (m *mockHub) Unsubscribe(channelName string, ch chan *protocol.Message) {}

func (m *mockHub) ResolveAgentID(agentID, agentName string) (string, error) {
	return agentID, nil
}

func (m *mockHub) GetChannel(name string) (*protocol.Channel, error) {
	if ch, ok := m.channels[name]; ok {
		return ch, nil
	}
	return nil, errors.New("not found")
}

func (m *mockHub) CreateChannelWithType(name, description, project string, channelType protocol.ChannelType, createdBy string) *protocol.Channel {
	m.createCalled = true
	ch := &protocol.Channel{
		Name:        name,
		Description: description,
		Type:        channelType,
		CreatedBy:   createdBy,
	}
	if m.channels == nil {
		m.channels = make(map[string]*protocol.Channel)
	}
	m.channels[name] = ch
	return ch
}

func (m *mockHub) SetChannelDisplay(name, displayName, description string) error {
	ch, err := m.GetChannel(name)
	if err != nil {
		return err
	}
	if displayName != "" {
		ch.DisplayName = displayName
	}
	if description != "" {
		ch.Description = description
	}
	return nil
}

func (m *mockHub) AddAgentToChannel(agentID, channelName string) error {
	m.addAgentCalls = append(m.addAgentCalls, struct{ agentID, channel string }{agentID, channelName})
	return nil
}

func TestApplyBindingCreatesChannelAndJoinsAgent(t *testing.T) {
	hub := &mockHub{channels: make(map[string]*protocol.Channel)}
	var ensured []string
	ensure := func(ctx context.Context, agentID, channelName string) error {
		ensured = append(ensured, agentID+":"+channelName)
		return nil
	}
	b := Binding{
		SlackChannelID:   "C1",
		SlackChannelName: "dev",
		AgentID:          "agent-42",
		Enabled:          true,
	}
	if err := ApplyBinding(context.Background(), hub, ensure, b); err != nil {
		t.Fatal(err)
	}
	if !hub.createCalled {
		t.Fatal("expected channel created")
	}
	if len(hub.addAgentCalls) != 1 || hub.addAgentCalls[0].agentID != "agent-42" {
		t.Fatalf("add agent calls: %+v", hub.addAgentCalls)
	}
	if len(ensured) != 1 || ensured[0] != "agent-42:slack:C1" {
		t.Fatalf("ensure: %v", ensured)
	}
}

func TestApplyBindingExistingChannel(t *testing.T) {
	hub := &mockHub{
		channels: map[string]*protocol.Channel{
			"slack:C1": {Name: "slack:C1", Type: protocol.ChannelTypeCustom},
		},
	}
	b := Binding{SlackChannelID: "C1", AgentID: "a1", NJChannel: "slack:C1", Enabled: true}
	if err := ApplyBinding(context.Background(), hub, nil, b); err != nil {
		t.Fatal(err)
	}
	if hub.createCalled {
		t.Fatal("should not create when channel exists")
	}
}

func TestApplyBindingRequiresAgentID(t *testing.T) {
	hub := &mockHub{channels: make(map[string]*protocol.Channel)}
	err := ApplyBinding(context.Background(), hub, nil, Binding{SlackChannelID: "C1"})
	if err == nil {
		t.Fatal("expected error")
	}
}

func TestNewBindingFromRequest(t *testing.T) {
	cfg := &config.Config{Slack: config.SlackConfig{DefaultPolicy: config.SlackPolicyQuestions}}
	b := NewBindingFromRequest("T1", "C1", "general", "a1", "Assistant", "", cfg)
	if b.WorkspaceID != "T1" || b.NJChannel != "slack:C1" || b.Policy != config.SlackPolicyQuestions {
		t.Fatalf("%+v", b)
	}
	if !b.Enabled || !b.ReplyInThread {
		t.Fatal("expected enabled threaded binding")
	}
	b2 := NewBindingFromRequest("", "C2", "", "a2", "", config.SlackPolicyAlways, nil)
	if b2.Policy != config.SlackPolicyAlways {
		t.Fatalf("policy %q", b2.Policy)
	}
}

func TestApplyAllBindingsSkipsDisabled(t *testing.T) {
	useTempHomeDir(t)
	store, err := NewBindingStore()
	if err != nil {
		t.Fatal(err)
	}
	_, _ = store.Upsert(Binding{SlackChannelID: "C1", AgentID: "a1", Enabled: true})
	_, _ = store.Upsert(Binding{SlackChannelID: "C2", AgentID: "a2", Enabled: false})
	hub := &mockHub{channels: make(map[string]*protocol.Channel)}
	if err := ApplyAllBindings(context.Background(), hub, nil, store); err != nil {
		t.Fatal(err)
	}
	if len(hub.addAgentCalls) != 1 {
		t.Fatalf("expected one apply, got %+v", hub.addAgentCalls)
	}
}
