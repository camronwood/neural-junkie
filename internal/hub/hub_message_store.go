package hub

import (
	"fmt"
	"log"

	"github.com/camronwood/neural-junkie/internal/memory"
	"github.com/camronwood/neural-junkie/internal/protocol"
)

// PersistentMessageStore persists chat messages outside in-memory caps.
type PersistentMessageStore interface {
	InsertMessage(msg *protocol.Message) error
	ListChannelMessages(channel string, limit int, beforeID string) ([]*protocol.Message, error)
	SearchMessages(opts MessageSearchOptions) ([]*protocol.Message, error)
	// ClearChannelMessages removes all persisted rows for a channel (e.g. after clear-history).
	ClearChannelMessages(channel string) error
}

// MessageSearchOptions configures archive search.
type MessageSearchOptions struct {
	Channel string
	Query   string
	Limit   int
	Before  int64
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
		return
	}
	memory.IndexMessage(msg)
}

// GetMessagesPage returns channel messages with optional cursor pagination.
func (h *Hub) GetMessagesPage(channelName string, limit int, beforeID string) ([]*protocol.Message, error) {
	if limit <= 0 {
		limit = 50
	}
	if h.persistentStore != nil {
		if beforeID != "" || h.isChannelDurable(channelName) {
			return h.persistentStore.ListChannelMessages(channelName, limit, beforeID)
		}
	}
	return h.GetMessages(channelName, limit)
}

// SearchMessages searches persisted archive when store supports it.
func (h *Hub) SearchMessages(opts MessageSearchOptions) ([]*protocol.Message, error) {
	if h.persistentStore == nil {
		return nil, fmt.Errorf("message archive search unavailable")
	}
	return h.persistentStore.SearchMessages(opts)
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

// UnmarkChannelDurable allows age-based prune again for a channel.
func (h *Hub) UnmarkChannelDurable(channelName string) {
	if h == nil || channelName == "" {
		return
	}
	h.mu.Lock()
	defer h.mu.Unlock()
	if h.durableChannels != nil {
		delete(h.durableChannels, channelName)
	}
}
