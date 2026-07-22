package hub

import "github.com/camronwood/neural-junkie/internal/protocol"

// GetChannelMessagesMerged returns the newest limit messages from SQLite+memory merge.
// Uses a bounded ListChannelMessages page — never a full channel export — so agent
// bootstrap stays fast even when ~/.neural-junkie/messages.db is large.
func (h *Hub) GetChannelMessagesMerged(channelName string, limit int) ([]*protocol.Message, error) {
	if h == nil || channelName == "" {
		return nil, nil
	}
	if limit <= 0 {
		limit = 50
	}
	return h.GetMessagesPage(channelName, limit, "")
}
