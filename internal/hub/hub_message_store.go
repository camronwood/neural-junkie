package hub

import (
	"fmt"
	"log"

	"github.com/camronwood/neural-junkie/internal/agent"
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
	defer func() {
		if r := recover(); r != nil {
			log.Printf("[hub] persist message panic: %v", r)
		}
	}()
	if msg.Type == protocol.MessageTypeStreamDelta || msg.Type == protocol.MessageTypeStreamEnd || msg.Type == protocol.MessageTypeAgentStatus {
		return
	}
	// Drop late async persists after clear-history emptied the in-memory channel.
	// New sends append to memory before queuePersistLocked, so live messages still land.
	if msg.Channel != "" && msg.ID != "" && !h.messagePresentInChannel(msg.Channel, msg.ID) {
		return
	}
	// msg should already be an exclusive snapshot; clone again so callers that pass
	// a live pointer still get a private copy for SQLite + memory index.
	persisted, err := protocol.CloneMessage(msg)
	if err != nil || persisted == nil {
		log.Printf("[hub] clone message for persistence: %v", err)
		return
	}
	if persisted.Metadata != nil {
		delete(persisted.Metadata, agent.MetadataAmbientState)
	}
	if err := h.persistentStore.InsertMessage(persisted); err != nil {
		log.Printf("[hub] persist message: %v", err)
		return
	}
	memory.IndexMessage(persisted)
}

func (h *Hub) messagePresentInChannel(channelName, msgID string) bool {
	h.mu.RLock()
	defer h.mu.RUnlock()
	for _, m := range h.messages[channelName] {
		if m != nil && m.ID == msgID {
			return true
		}
	}
	return false
}

// GetMessagesPage returns channel messages with optional cursor pagination.
// When the SQLite archive is available, pages are served from disk and merged with
// any newer in-memory rows so post-restart history is visible without requiring
// last-session restore.
func (h *Hub) GetMessagesPage(channelName string, limit int, beforeID string) ([]*protocol.Message, error) {
	if limit <= 0 {
		limit = 50
	}
	if h.persistentStore == nil {
		return h.GetMessages(channelName, limit)
	}
	if beforeID != "" || h.isChannelDurable(channelName) {
		return h.persistentStore.ListChannelMessages(channelName, limit, beforeID)
	}
	persisted, err := h.persistentStore.ListChannelMessages(channelName, limit, "")
	if err != nil {
		log.Printf("[hub] list persisted messages for %q: %v", channelName, err)
		return h.GetMessages(channelName, limit)
	}
	mem, memErr := h.GetMessages(channelName, limit)
	if memErr != nil || len(mem) == 0 {
		return persisted, nil
	}
	if len(persisted) == 0 {
		return mem, nil
	}
	return mergeMessagePages(mem, persisted, limit), nil
}

func mergeMessagePages(mem, persisted []*protocol.Message, limit int) []*protocol.Message {
	byID := make(map[string]*protocol.Message, len(mem)+len(persisted))
	order := make([]string, 0, len(mem)+len(persisted))
	for _, msg := range persisted {
		if msg == nil || msg.ID == "" {
			continue
		}
		if _, ok := byID[msg.ID]; !ok {
			order = append(order, msg.ID)
		}
		byID[msg.ID] = msg
	}
	for _, msg := range mem {
		if msg == nil || msg.ID == "" {
			continue
		}
		if _, ok := byID[msg.ID]; !ok {
			order = append(order, msg.ID)
		}
		byID[msg.ID] = msg // in-memory wins (newer card status / content)
	}
	out := make([]*protocol.Message, 0, len(order))
	for _, id := range order {
		out = append(out, byID[id])
	}
	if limit > 0 && len(out) > limit {
		out = out[len(out)-limit:]
	}
	return out
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
