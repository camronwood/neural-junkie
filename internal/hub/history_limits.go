package hub

import "github.com/camronwood/neural-junkie/internal/protocol"

const (
	// MaxHubChannelHistory is the maximum persisted channel messages per channel
	// (including thread replies mirrored into channel history). Older rows are dropped.
	MaxHubChannelHistory = 5000
	// MaxHubThreadHistory is the maximum messages stored per thread.
	MaxHubThreadHistory = 2000
)

func keepLastPtrSlice(msgs []*protocol.Message, max int) []*protocol.Message {
	if max <= 0 || len(msgs) <= max {
		return msgs
	}
	return msgs[len(msgs)-max:]
}

// enforceMaxChannelHistoryLocked drops oldest channel messages if over the cap and
// broadcasts an ephemeral history_resync so WebSocket clients refetch from the API.
// Caller must hold h.mu (write lock).
func (h *Hub) enforceMaxChannelHistoryLocked(channelName string) {
	msgs := h.messages[channelName]
	if len(msgs) <= MaxHubChannelHistory {
		return
	}
	h.messages[channelName] = keepLastPtrSlice(msgs, MaxHubChannelHistory)
	systemFrom := protocol.AgentInfo{ID: "system", Name: "System", Type: protocol.AgentTypeGeneral}
	resync := protocol.NewMessage(protocol.MessageTypeAgentStatus, channelName, systemFrom, "")
	resync.Metadata[MetadataKeyHistoryResync] = true
	h.broadcast(channelName, resync)
}

// appendChannelMessageLocked appends to channel history and enforces MaxHubChannelHistory.
// If msg.ID already exists in the channel, the existing row is replaced (avoids duplicate
// rows from double POST or reconnect replay).
// Caller must hold h.mu (write lock).
func (h *Hub) appendChannelMessageLocked(channelName string, msg *protocol.Message) {
	if msg == nil {
		return
	}
	msgs := h.messages[channelName]
	if msg.ID != "" {
		for i := len(msgs) - 1; i >= 0; i-- {
			if msgs[i] != nil && msgs[i].ID == msg.ID {
				msgs[i] = msg
				h.messages[channelName] = msgs
				h.enforceMaxChannelHistoryLocked(channelName)
				return
			}
		}
	}
	h.messages[channelName] = append(msgs, msg)
	h.enforceMaxChannelHistoryLocked(channelName)
}

// enforceMaxThreadHistoryLocked drops oldest thread messages if over the cap.
// Caller must hold h.mu (write lock).
func (h *Hub) enforceMaxThreadHistoryLocked(threadID string) {
	msgs := h.threads[threadID]
	if len(msgs) <= MaxHubThreadHistory {
		return
	}
	h.threads[threadID] = keepLastPtrSlice(msgs, MaxHubThreadHistory)
}

// appendThreadMessageLocked appends to thread storage and enforces MaxHubThreadHistory.
// Caller must hold h.mu (write lock).
func (h *Hub) appendThreadMessageLocked(threadID string, msg *protocol.Message) {
	h.threads[threadID] = append(h.threads[threadID], msg)
	h.enforceMaxThreadHistoryLocked(threadID)
}

// trimAllChannelAndThreadHistoryLocked enforces caps after session restore.
// Caller must hold h.mu (write lock).
func (h *Hub) trimAllChannelAndThreadHistoryLocked() {
	for name := range h.messages {
		h.enforceMaxChannelHistoryLocked(name)
	}
	for tid := range h.threads {
		h.enforceMaxThreadHistoryLocked(tid)
	}
}
