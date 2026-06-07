package protocol

import "strings"

// Slack bridge metadata keys (set by internal/integrations/slack).
const (
	SlackMetaRouteAgentID      = "slack_route_agent_id"
	SlackMetaAppMention        = "slack_app_mention"
	SlackMetaUserDisplayName   = "slack_user_display_name"
	SlackMetaUsername          = "slack_username"
	SlackMetaInbox             = "slack_inbox"
	SlackMetaReplyChannelID    = "slack_reply_channel_id"
	SlackMetaReplyThreadTS     = "slack_reply_thread_ts"
	SlackMetaForwardRule       = "slack_forward_rule"
	SlackMetaPermalink         = "slack_permalink"
	SlackMetaSourceChannelName = "slack_source_channel_name"
	SlackMetaOriginalAuthor    = "slack_original_author"
	SlackMetaHumanDM           = "slack_human_dm"
	SlackMetaManualReply       = "slack_manual_reply"
)

// IsSlackMirrorChannel reports hub channels mirrored from Slack (slack:…).
func IsSlackMirrorChannel(channel string) bool {
	return strings.HasPrefix(channel, "slack:")
}

// IsSlackInboxChannel reports personal inbox hub channels (slack:inbox:U…).
func IsSlackInboxChannel(channel string) bool {
	return strings.HasPrefix(channel, "slack:inbox:")
}

// IsSlackInboxPeerChannel reports per-peer inbox channels (slack:inbox:U_OWNER:U_PEER).
func IsSlackInboxPeerChannel(channel string) bool {
	parts := strings.Split(channel, ":")
	return len(parts) >= 4 && parts[0] == "slack" && parts[1] == "inbox"
}

// IsSlackManualInboxReply reports hub inbox lines surfaced for manual owner reply.
func (m *Message) IsSlackManualInboxReply() bool {
	if m == nil || m.Metadata == nil {
		return false
	}
	v, ok := m.Metadata[SlackMetaManualReply].(bool)
	return ok && v
}

// SlackInboxAwaitingManualReply reports inbox messages the owner should answer manually.
func (m *Message) SlackInboxAwaitingManualReply() bool {
	if m.IsSlackManualInboxReply() {
		return true
	}
	if m == nil || !IsSlackInboxChannel(m.Channel) || m.Metadata == nil {
		return false
	}
	inbox, ok := m.Metadata[SlackMetaInbox].(bool)
	if !ok || !inbox || m.SlackRoutedAgentID() != "" {
		return false
	}
	return strings.HasPrefix(m.From.ID, "slack:")
}

// SlackRoutedAgentID returns the agent the Slack bridge routed this line to.
func (m *Message) SlackRoutedAgentID() string {
	if m == nil || !IsSlackMirrorChannel(m.Channel) || m.Metadata == nil {
		return ""
	}
	id, _ := m.Metadata[SlackMetaRouteAgentID].(string)
	return strings.TrimSpace(id)
}

// SlackReplyChannelID returns the Slack channel for outbound inbox replies.
func (m *Message) SlackReplyChannelID() string {
	if m == nil || m.Metadata == nil {
		return ""
	}
	id, _ := m.Metadata[SlackMetaReplyChannelID].(string)
	return strings.TrimSpace(id)
}

// SlackReplyThreadTS returns the Slack thread_ts for outbound inbox replies.
func (m *Message) SlackReplyThreadTS() string {
	if m == nil || m.Metadata == nil {
		return ""
	}
	ts, _ := m.Metadata[SlackMetaReplyThreadTS].(string)
	return strings.TrimSpace(ts)
}

// IsSlackHumanDMMessage reports hub messages ingested from the owner's human DMs.
func IsSlackHumanDMMessage(msg *Message) bool {
	if msg == nil || msg.Metadata == nil {
		return false
	}
	if v, ok := msg.Metadata[SlackMetaHumanDM].(bool); ok && v {
		return true
	}
	src, _ := msg.Metadata["source"].(string)
	return src == "slack_human_dm"
}
