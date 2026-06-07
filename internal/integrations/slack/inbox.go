package slack

import (
	"context"
	"fmt"
	"log"
	"strings"

	"github.com/camronwood/neural-junkie/internal/protocol"
)

// IsIMChannel reports Slack direct message channels (D…).
func IsIMChannel(channelID string) bool {
	return strings.HasPrefix(strings.TrimSpace(channelID), "D")
}

// ApplyInbox ensures the hub inbox channel exists and the agent is subscribed.
func ApplyInbox(ctx context.Context, hub HubClient, ensure AgentEnsurer, inbox InboxConfig) error {
	if !inbox.Enabled {
		return nil
	}
	if inbox.AgentID == "" {
		return fmt.Errorf("agent_id required")
	}
	if inbox.OwnerSlackUserID == "" {
		return fmt.Errorf("owner_slack_user_id required")
	}
	resolvedID, err := hub.ResolveAgentID(inbox.AgentID, inbox.AgentName)
	if err != nil {
		return err
	}
	inbox.AgentID = resolvedID
	if inbox.NJChannel == "" {
		inbox.NJChannel = NJInboxChannelName(inbox.OwnerSlackUserID)
	}
	desc := "Slack personal inbox"
	if inbox.OwnerSlackUserName != "" {
		desc = "Slack inbox — " + inbox.OwnerSlackUserName
	}
	if _, err := hub.GetChannel(inbox.NJChannel); err != nil {
		hub.CreateChannelWithType(inbox.NJChannel, desc, "", protocol.ChannelTypeCustom, "slack-inbox")
		log.Printf("[slack] created inbox hub channel %s", inbox.NJChannel)
	}
	if err := hub.AddAgentToChannel(inbox.AgentID, inbox.NJChannel); err != nil {
		return fmt.Errorf("add agent to inbox channel: %w", err)
	}
	if ensure != nil {
		if err := ensure(ctx, inbox.AgentID, inbox.NJChannel); err != nil {
			return fmt.Errorf("ensure inbox agent subscribed: %w", err)
		}
	}
	return nil
}

// EnsureInboxPeerChannel creates or updates a per-peer inbox hub channel.
func EnsureInboxPeerChannel(ctx context.Context, hub HubClient, ensure AgentEnsurer, inbox InboxConfig, peerUserID, peerDisplayName string) (channelName string, created bool, err error) {
	if !inbox.Enabled {
		return "", false, fmt.Errorf("inbox disabled")
	}
	if inbox.OwnerSlackUserID == "" {
		return "", false, fmt.Errorf("owner_slack_user_id required")
	}
	peerUserID = strings.TrimSpace(peerUserID)
	if peerUserID == "" {
		return "", false, fmt.Errorf("peer user id required")
	}
	name := NJInboxPeerChannelName(inbox.OwnerSlackUserID, peerUserID)
	display := strings.TrimSpace(peerDisplayName)
	if display == "" {
		display = peerUserID
	}
	desc := "Slack DM — " + display
	if _, err := hub.GetChannel(name); err != nil {
		hub.CreateChannelWithType(name, desc, "", protocol.ChannelTypeCustom, "slack-inbox")
		log.Printf("[slack] created inbox peer hub channel %s", name)
		created = true
	}
	_ = hub.SetChannelDisplay(name, display, desc)
	if inbox.AgentID == "" {
		return name, created, nil
	}
	resolvedID, err := hub.ResolveAgentID(inbox.AgentID, inbox.AgentName)
	if err != nil {
		return name, created, err
	}
	if err := hub.AddAgentToChannel(resolvedID, name); err != nil {
		return name, created, fmt.Errorf("add agent to peer inbox channel: %w", err)
	}
	if ensure != nil {
		if err := ensure(ctx, resolvedID, name); err != nil {
			return name, created, fmt.Errorf("ensure peer inbox agent subscribed: %w", err)
		}
	}
	return name, created, nil
}

// ReconcileInboxAgentID updates stored inbox agent_id when the hub restarted and UUIDs rotated.
func ReconcileInboxAgentID(store *InboxStore, hub HubClient) {
	if store == nil || hub == nil {
		return
	}
	cfg := store.Get()
	if !cfg.Enabled || cfg.AgentID == "" {
		return
	}
	resolved, err := hub.ResolveAgentID(cfg.AgentID, cfg.AgentName)
	if err != nil || resolved == cfg.AgentID {
		return
	}
	cfg.AgentID = resolved
	if _, err := store.Save(cfg); err != nil {
		log.Printf("[slack] reconcile inbox agent: %v", err)
		return
	}
	log.Printf("[slack] inbox agent_id reconciled to %s (%s)", resolved, cfg.AgentName)
}

// InboxOwnerAllowed reports whether the Slack user may use the personal inbox.
func InboxOwnerAllowed(inbox *InboxConfig, userID string) bool {
	if inbox == nil || !inbox.Enabled || inbox.OwnerSlackUserID == "" {
		return false
	}
	return strings.TrimSpace(userID) == strings.TrimSpace(inbox.OwnerSlackUserID)
}

// BuildInboxMessage converts a direct DM or forwarded line into a hub inbox message.
// njChannel overrides inbox.NJChannel when set (e.g. per-peer human DM channels).
func BuildInboxMessage(in InboundInput, inbox *InboxConfig, threads *ThreadMap, forward *ForwardMatch, njChannel string) *protocol.Message {
	content := StripSlackMentionMarkup(in.Text)
	if forward != nil {
		content = BuildForwardedContent(forward, content)
	}

	threadID, replyTo, isThread := threads.ResolveInbound(in.ChannelID, in.SlackTS, in.ThreadTS)
	if isThread && threadID == in.ThreadTS {
		if parentNJ := threads.NJMessageForSlackTS(in.ThreadTS); parentNJ != "" {
			threadID = parentNJ
		} else {
			threadID = in.ThreadTS
		}
		replyTo = in.SlackTS
	}

	from := protocol.AgentInfo{
		ID:   "slack:" + in.UserID,
		Name: in.UserName,
		Type: protocol.AgentTypeGeneral,
	}

	msgType := protocol.MessageTypeChat
	if looksLikeQuestion(content) {
		msgType = protocol.MessageTypeQuestion
	}

	targetChannel := strings.TrimSpace(njChannel)
	if targetChannel == "" && inbox != nil {
		targetChannel = inbox.NJChannel
	}

	msg := protocol.NewMessage(msgType, targetChannel, from, content)
	if isThread {
		msg.ThreadID = threadID
		msg.ReplyTo = replyTo
		msg.IsThreadReply = true
	}

	if msg.Metadata == nil {
		msg.Metadata = make(map[string]interface{})
	}
	msg.Metadata["source"] = "slack_inbox"
	msg.Metadata[protocol.SlackMetaInbox] = true
	msg.Metadata[protocol.SlackMetaRouteAgentID] = inbox.AgentID
	msg.Metadata["slack_channel_id"] = in.ChannelID
	msg.Metadata["slack_ts"] = in.SlackTS
	msg.Metadata["slack_user_id"] = in.UserID
	if in.UserName != "" {
		msg.Metadata[protocol.SlackMetaUserDisplayName] = in.UserName
	}
	if in.SlackUsername != "" {
		msg.Metadata[protocol.SlackMetaUsername] = in.SlackUsername
	}
	if in.ThreadTS != "" {
		msg.Metadata["slack_thread_ts"] = in.ThreadTS
	}

	replyChannel := inbox.SlackDMChannelID
	replyThreadTS := in.ThreadTS
	if replyThreadTS == "" {
		replyThreadTS = in.SlackTS
	}
	if forward != nil {
		msg.Metadata[protocol.SlackMetaForwardRule] = string(forward.RuleType)
		if forward.SourceChannelName != "" {
			msg.Metadata[protocol.SlackMetaSourceChannelName] = forward.SourceChannelName
		}
		if forward.SourceAuthor != "" {
			msg.Metadata[protocol.SlackMetaOriginalAuthor] = forward.SourceAuthor
		}
		if forward.Permalink != "" {
			msg.Metadata[protocol.SlackMetaPermalink] = forward.Permalink
		}
		replyChannel = forward.SourceChannelID
		if forward.SourceThreadTS != "" {
			replyThreadTS = forward.SourceThreadTS
		} else if forward.SourceTS != "" {
			replyThreadTS = forward.SourceTS
		}
	} else if IsIMChannel(in.ChannelID) {
		replyChannel = in.ChannelID
	}

	if replyChannel != "" {
		msg.Metadata[protocol.SlackMetaReplyChannelID] = replyChannel
	}
	if replyThreadTS != "" {
		msg.Metadata[protocol.SlackMetaReplyThreadTS] = replyThreadTS
	}

	return msg
}

// ApplyInboxManualReply marks an inbox message for manual owner reply (forward mode).
func ApplyInboxManualReply(msg *protocol.Message) {
	if msg == nil {
		return
	}
	if msg.Metadata == nil {
		msg.Metadata = make(map[string]interface{})
	}
	msg.Metadata[protocol.SlackMetaManualReply] = true
	delete(msg.Metadata, protocol.SlackMetaRouteAgentID)
}

// BuildForwardedContent wraps forwarded channel text for the agent.
func BuildForwardedContent(forward *ForwardMatch, text string) string {
	if forward == nil {
		return text
	}
	label := forward.SourceChannelName
	if label == "" {
		label = forward.SourceChannelID
	}
	author := forward.SourceAuthor
	if author == "" {
		author = "someone"
	}
	header := fmt.Sprintf("[Forwarded from #%s — %s]", strings.TrimPrefix(label, "#"), author)
	return header + "\n" + strings.TrimSpace(text)
}
