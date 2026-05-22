package protocol

import "strings"

// Slack bridge metadata keys (set by internal/integrations/slack).
const (
	SlackMetaRouteAgentID     = "slack_route_agent_id"
	SlackMetaAppMention       = "slack_app_mention"
	SlackMetaUserDisplayName  = "slack_user_display_name"
	SlackMetaUsername         = "slack_username"
)

// IsSlackMirrorChannel reports hub channels mirrored from Slack (slack:C…).
func IsSlackMirrorChannel(channel string) bool {
	return strings.HasPrefix(channel, "slack:")
}

// SlackRoutedAgentID returns the agent the Slack bridge routed this line to.
func (m *Message) SlackRoutedAgentID() string {
	if m == nil || !IsSlackMirrorChannel(m.Channel) || m.Metadata == nil {
		return ""
	}
	id, _ := m.Metadata[SlackMetaRouteAgentID].(string)
	return strings.TrimSpace(id)
}
