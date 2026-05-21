package slack

import (
	"context"
	"fmt"
	"log"

	"github.com/camronwood/neural-junkie/internal/config"
	"github.com/camronwood/neural-junkie/internal/protocol"
)

// ApplyBinding ensures the hub channel exists and the primary agent is joined and listening.
func ApplyBinding(ctx context.Context, hub HubClient, ensure AgentEnsurer, b Binding) error {
	if b.AgentID == "" {
		return fmt.Errorf("agent_id required")
	}
	if b.NJChannel == "" {
		b.NJChannel = NJChannelName(b.SlackChannelID)
	}
	if _, err := hub.GetChannel(b.NJChannel); err != nil {
		desc := "Slack bridge"
		if b.SlackChannelName != "" {
			desc = "Slack: #" + b.SlackChannelName
		}
		hub.CreateChannelWithType(b.NJChannel, desc, "", protocol.ChannelTypeCustom, "slack-bridge")
		log.Printf("[slack] created hub channel %s", b.NJChannel)
	}
	if err := hub.AddAgentToChannel(b.AgentID, b.NJChannel); err != nil {
		return fmt.Errorf("add agent to channel: %w", err)
	}
	if ensure != nil {
		if err := ensure(ctx, b.AgentID, b.NJChannel); err != nil {
			return fmt.Errorf("ensure agent subscribed: %w", err)
		}
	}
	return nil
}

// ApplyAllBindings reapplies every enabled binding (e.g. on bridge start).
func ApplyAllBindings(ctx context.Context, hub HubClient, ensure AgentEnsurer, store *BindingStore) error {
	for _, b := range store.List() {
		if !b.Enabled {
			continue
		}
		if err := ApplyBinding(ctx, hub, ensure, b); err != nil {
			log.Printf("[slack] binding %s: %v", b.SlackChannelID, err)
		}
	}
	return nil
}

// NewBindingFromRequest builds a binding with defaults from config.
func NewBindingFromRequest(workspaceID, slackChannelID, slackChannelName, agentID, agentName string, policy config.SlackPolicy, cfg *config.Config) Binding {
	if policy == "" && cfg != nil {
		policy = cfg.Slack.EffectiveDefaultPolicy()
	}
	if policy == "" {
		policy = config.SlackPolicyMentionOnly
	}
	return Binding{
		WorkspaceID:      workspaceID,
		SlackChannelID:   slackChannelID,
		SlackChannelName: slackChannelName,
		NJChannel:        NJChannelName(slackChannelID),
		AgentID:          agentID,
		AgentName:        agentName,
		Policy:           policy,
		ReplyInThread:    true,
		Enabled:          true,
	}
}
