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
)

// IsSlackMirrorChannel reports hub channels mirrored from Slack (slack:…).
func IsSlackMirrorChannel(channel string) bool {
	return strings.HasPrefix(channel, "slack:")
}

// IsSlackInboxChannel reports personal inbox hub channels (slack:inbox:U…).
func IsSlackInboxChannel(channel string) bool {
	return strings.HasPrefix(channel, "slack:inbox:")
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
