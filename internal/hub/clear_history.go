package hub

import (
	"fmt"
	"strings"

	"github.com/camronwood/neural-junkie/internal/protocol"
)

// ClearChannelHistory removes all persisted messages for a channel and its threads,
// then broadcasts history_resync so clients and agents refetch empty history.
func (h *Hub) ClearChannelHistory(channelName string) error {
	channelName = strings.TrimSpace(channelName)
	if channelName == "" {
		return fmt.Errorf("channel name is required")
	}

	h.mu.Lock()
	if _, ok := h.channels[channelName]; !ok {
		h.mu.Unlock()
		return fmt.Errorf("channel %s not found", channelName)
	}

	h.messages[channelName] = []*protocol.Message{}

	var threadIDs []string
	for tid, meta := range h.threadMetadata {
		if meta != nil && meta.Channel == channelName {
			threadIDs = append(threadIDs, tid)
		}
	}
	for _, tid := range threadIDs {
		delete(h.threads, tid)
		delete(h.threadMetadata, tid)
		delete(h.threadParentAuthors, tid)
		delete(h.threadSubscribers, tid)
	}
	h.clearChannelContextLocked(channelName)
	h.mu.Unlock()

	systemFrom := protocol.AgentInfo{ID: "system", Name: "System", Type: protocol.AgentTypeGeneral}
	resync := protocol.NewMessage(protocol.MessageTypeAgentStatus, channelName, systemFrom, "")
	resync.Metadata[MetadataKeyHistoryResync] = true
	h.BroadcastDirect(channelName, resync)

	return nil
}
