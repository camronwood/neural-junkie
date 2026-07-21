package agent

// agentsDeferred reports whether agents should not start new turns on channel
// (user Stop / interject, or a pending ask_user question).
func agentsDeferred(hub HubClient, channel string) bool {
	if hub == nil || channel == "" {
		return false
	}
	if hub.IsChannelHeld(channel) {
		return true
	}
	type pendingQ interface {
		HasPendingUserQuestion(channel string) bool
	}
	if p, ok := hub.(pendingQ); ok {
		return p.HasPendingUserQuestion(channel)
	}
	return false
}
