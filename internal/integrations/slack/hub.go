package slack

import (
	"context"

	"github.com/camronwood/neural-junkie/internal/hub"
	"github.com/camronwood/neural-junkie/internal/protocol"
)

// HubClient is the hub surface required by the Slack bridge.
type HubClient interface {
	SendMessage(msg *protocol.Message) error
	Subscribe(channelName string) (chan *protocol.Message, error)
	GetChannel(name string) (*protocol.Channel, error)
	CreateChannelWithType(name, description, project string, channelType protocol.ChannelType, createdBy string) *protocol.Channel
	AddAgentToChannel(agentID, channelName string) error
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

func (a HubAdapter) GetChannel(name string) (*protocol.Channel, error) {
	return a.H.GetChannel(name)
}

func (a HubAdapter) CreateChannelWithType(name, description, project string, channelType protocol.ChannelType, createdBy string) *protocol.Channel {
	return a.H.CreateChannelWithType(name, description, project, channelType, createdBy)
}

func (a HubAdapter) AddAgentToChannel(agentID, channelName string) error {
	return a.H.AddAgentToChannel(agentID, channelName)
}
