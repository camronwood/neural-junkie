package slack

import (
	"context"
	"fmt"
	"strings"

	"github.com/camronwood/neural-junkie/internal/hub"
	"github.com/camronwood/neural-junkie/internal/protocol"
)

// HubClient is the hub surface required by the Slack bridge.
type HubClient interface {
	SendMessage(msg *protocol.Message) error
	Subscribe(channelName string) (chan *protocol.Message, error)
	Unsubscribe(channelName string, ch chan *protocol.Message)
	GetChannel(name string) (*protocol.Channel, error)
	CreateChannelWithType(name, description, project string, channelType protocol.ChannelType, createdBy string) *protocol.Channel
	AddAgentToChannel(agentID, channelName string) error
	ResolveAgentID(agentID, agentName string) (string, error)
	SetChannelDisplay(name, displayName, description string) error
}

// AgentEnsurer starts an agent listening on a channel.
type AgentEnsurer func(ctx context.Context, agentID, channelName string) error

// HubAdapter wraps *hub.Hub for the bridge.
type HubAdapter struct {
	H *hub.Hub
}

func (a HubAdapter) SendMessage(msg *protocol.Message) error {
	return a.H.SendMessage(msg)
}

func (a HubAdapter) Subscribe(channelName string) (chan *protocol.Message, error) {
	return a.H.Subscribe(channelName)
}

func (a HubAdapter) Unsubscribe(channelName string, ch chan *protocol.Message) {
	a.H.Unsubscribe(channelName, ch)
}

func (a HubAdapter) GetChannel(name string) (*protocol.Channel, error) {
	return a.H.GetChannel(name)
}

func (a HubAdapter) CreateChannelWithType(name, description, project string, channelType protocol.ChannelType, createdBy string) *protocol.Channel {
	return a.H.CreateChannelWithType(name, description, project, channelType, createdBy)
}

func (a HubAdapter) AddAgentToChannel(agentID, channelName string) error {
	return a.H.AddAgentToChannel(agentID, channelName)
}

func (a HubAdapter) SetChannelDisplay(name, displayName, description string) error {
	return a.H.SetChannelDisplay(name, displayName, description)
}

func (a HubAdapter) ResolveAgentID(agentID, agentName string) (string, error) {
	if _, err := a.H.GetAgent(agentID); err == nil {
		return agentID, nil
	}
	name := strings.TrimSpace(agentName)
	if name == "" {
		name = "Assistant"
	}
	if ag := a.H.FindLiveAgentByDisplayName(name, ""); ag != nil {
		return ag.ID, nil
	}
	if strings.EqualFold(name, "assistant") {
		if ag := a.H.FindLiveAgentByDisplayName("Assistant", protocol.AgentTypeAssistant); ag != nil {
			return ag.ID, nil
		}
	}
	return "", fmt.Errorf("agent %q (%s) not found", agentID, agentName)
}
