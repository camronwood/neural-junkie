package hub

import "github.com/camronwood/neural-junkie/internal/protocol"

// GetChannelMessagesMerged returns the newest limit messages from SQLite+memory merge.
func (h *Hub) GetChannelMessagesMerged(channelName string, limit int) ([]*protocol.Message, error) {
	if h == nil || channelName == "" {
		return nil, nil
	}
	if limit <= 0 {
		limit = 50
	}
	all := h.ExportChannelMessages(channelName)
	if len(all) == 0 {
		return all, nil
	}
	if len(all) <= limit {
		return all, nil
	}
	out := make([]*protocol.Message, limit)
	copy(out, all[len(all)-limit:])
	return out, nil
}
