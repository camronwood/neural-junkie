package hub

import (
	"log"

	"github.com/camronwood/neural-junkie/internal/protocol"
)

// PersistentMessageStore persists chat messages outside in-memory caps.
type PersistentMessageStore interface {
	InsertMessage(msg *protocol.Message) error
	ListChannelMessages(channel string, limit int, beforeID string) ([]*protocol.Message, error)
}

// SetPersistentMessageStore wires optional SQLite (or other) durable storage.
func (h *Hub) SetPersistentMessageStore(store PersistentMessageStore) {
	if h == nil {
		return
	}
	h.mu.Lock()
	defer h.mu.Unlock()
	h.persistentStore = store
}

func (h *Hub) persistMessage(msg *protocol.Message) {
	if h == nil || h.persistentStore == nil || msg == nil {
		return
	}
	if msg.Type == protocol.MessageTypeStreamDelta || msg.Type == protocol.MessageTypeStreamEnd || msg.Type == protocol.MessageTypeAgentStatus {
		return
	}
	if err := h.persistentStore.InsertMessage(msg); err != nil {
		log.Printf("[hub] persist message: %v", err)
	}
}

// GetMessagesPage returns channel messages with optional cursor pagination.
func (h *Hub) GetMessagesPage(channelName string, limit int, beforeID string) ([]*protocol.Message, error) {
	if limit <= 0 {
		limit = 50
	}
	if beforeID != "" && h.persistentStore != nil {
		return h.persistentStore.ListChannelMessages(channelName, limit, beforeID)
	}
	return h.GetMessages(channelName, limit)
}

// MarkChannelDurable skips age-based prune for IDE-heavy channels.
func (h *Hub) MarkChannelDurable(channelName string) {
	if h == nil || channelName == "" {
		return
	}
	h.mu.Lock()
	defer h.mu.Unlock()
	if h.durableChannels == nil {
		h.durableChannels = map[string]bool{}
	}
	h.durableChannels[channelName] = true
}

func (h *Hub) isChannelDurable(channelName string) bool {
	if h == nil || h.durableChannels == nil {
		return false
	}
	return h.durableChannels[channelName]
}
