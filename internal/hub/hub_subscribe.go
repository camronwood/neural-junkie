package hub

import (
	"fmt"
	"log"

	"github.com/camronwood/neural-junkie/internal/protocol"
)

func (h *Hub) Subscribe(channelName string) (chan *protocol.Message, error) {
	h.mu.Lock()
	defer h.mu.Unlock()

	if _, ok := h.channels[channelName]; !ok {
		return nil, fmt.Errorf("channel %s not found", channelName)
	}

	ch := make(chan *protocol.Message, 512)
	h.subscribers[channelName] = append(h.subscribers[channelName], ch)

	return ch, nil
}

// SubscribeUI creates a frontend UI subscription that receives all channel messages,
// including ephemeral stream deltas and UI-only agent_status broadcasts.
func (h *Hub) SubscribeUI(channelName string) (chan *protocol.Message, error) {
	h.mu.Lock()
	defer h.mu.Unlock()

	if _, ok := h.channels[channelName]; !ok {
		return nil, fmt.Errorf("channel %s not found", channelName)
	}

	ch := make(chan *protocol.Message, 512)
	h.uiSubscribers[channelName] = append(h.uiSubscribers[channelName], ch)

	return ch, nil
}

// Unsubscribe removes an agent/integration tier subscription.
func (h *Hub) Unsubscribe(channelName string, ch chan *protocol.Message) {
	h.mu.Lock()
	defer h.mu.Unlock()

	h.unsubscribeFrom(h.subscribers, channelName, ch)
}

// UnsubscribeUI removes a frontend UI tier subscription.
func (h *Hub) UnsubscribeUI(channelName string, ch chan *protocol.Message) {
	h.mu.Lock()
	defer h.mu.Unlock()

	h.unsubscribeFrom(h.uiSubscribers, channelName, ch)
}

func (h *Hub) unsubscribeFrom(subsByChannel map[string][]chan *protocol.Message, channelName string, ch chan *protocol.Message) {
	subs, ok := subsByChannel[channelName]
	if !ok {
		return
	}

	for i, sub := range subs {
		if sub == ch {
			subsByChannel[channelName] = append(subs[:i], subs[i+1:]...)
			close(ch)
			break
		}
	}
}

func deliverToAgentTier(msg *protocol.Message) bool {
	if msg == nil {
		return false
	}
	switch msg.Type {
	case protocol.MessageTypeStreamDelta, protocol.MessageTypeStreamEnd:
		return false
	case protocol.MessageTypeAgentStatus:
		return isControlPlaneAgentStatus(msg)
	default:
		return true
	}
}

func isControlPlaneAgentStatus(msg *protocol.Message) bool {
	if msg == nil || msg.Metadata == nil {
		return false
	}
	if _, ok := msg.Metadata[protocol.MetadataChannelHold]; ok {
		return true
	}
	if v, ok := msg.Metadata[protocol.MetadataChannelInterjectAbort].(bool); ok && v {
		return true
	}
	if v, ok := msg.Metadata[MetadataKeyHistoryResync].(bool); ok && v {
		return true
	}
	return false
}

// broadcast sends a message to channel subscribers (must be called with lock held).
func (h *Hub) broadcast(channelName string, msg *protocol.Message) {
	h.deliverToSubscribers(h.uiSubscribers[channelName], msg)

	if deliverToAgentTier(msg) {
		h.deliverToSubscribers(h.subscribers[channelName], msg)
	}
}

func (h *Hub) deliverToSubscribers(subs []chan *protocol.Message, msg *protocol.Message) {
	if len(subs) == 0 {
		return
	}

	dropped := 0
	for _, ch := range subs {
		select {
		case ch <- msg:
		default:
			dropped++
		}
	}
	if dropped > 0 {
		log.Printf("[Hub] broadcast: dropped %d/%d messages (subscriber buffer full)", dropped, len(subs))
	}
}

// BroadcastDirect sends a message to all subscribers of a channel without
// storing it in message history. Used for ephemeral messages like stream
// deltas that should reach the frontend but not pollute the history.
func (h *Hub) BroadcastDirect(channelName string, msg *protocol.Message) {
	h.mu.RLock()
	defer h.mu.RUnlock()

	h.broadcast(channelName, msg)
	if msg != nil && msg.IsInThread() {
		if threadID := msg.GetThreadID(); threadID != "" {
			h.broadcastToThread(threadID, msg)
		}
	}
}

// GetChannelAgents returns all agents in a channel
func (h *Hub) SubscribeToThread(threadID string) (chan *protocol.Message, error) {
	h.mu.Lock()
	defer h.mu.Unlock()

	ch := make(chan *protocol.Message, 100)
	h.threadSubscribers[threadID] = append(h.threadSubscribers[threadID], ch)

	return ch, nil
}

// UnsubscribeFromThread removes a thread subscription
func (h *Hub) UnsubscribeFromThread(threadID string, ch chan *protocol.Message) {
	h.mu.Lock()
	defer h.mu.Unlock()

	subs, ok := h.threadSubscribers[threadID]
	if !ok {
		return
	}

	for i, sub := range subs {
		if sub == ch {
			h.threadSubscribers[threadID] = append(subs[:i], subs[i+1:]...)
			close(ch)
			break
		}
	}
}

// broadcastToThread sends a message to all thread subscribers (must be called with lock held)
func (h *Hub) broadcastToThread(threadID string, msg *protocol.Message) {
	subs, ok := h.threadSubscribers[threadID]
	if !ok {
		return
	}

	for _, ch := range subs {
		select {
		case ch <- msg:
		default:
			// Channel full, skip
		}
	}
}

// GetRemovedAgents returns all agents that have been removed from conversations
