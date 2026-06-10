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

// Unsubscribe removes a subscription
func (h *Hub) Unsubscribe(channelName string, ch chan *protocol.Message) {
	h.mu.Lock()
	defer h.mu.Unlock()

	subs, ok := h.subscribers[channelName]
	if !ok {
		return
	}

	for i, sub := range subs {
		if sub == ch {
			h.subscribers[channelName] = append(subs[:i], subs[i+1:]...)
			close(ch)
			break
		}
	}
}

// broadcast sends a message to all subscribers of a channel (must be called with lock held)
func (h *Hub) broadcast(channelName string, msg *protocol.Message) {
	subs, ok := h.subscribers[channelName]
	if !ok {
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
		log.Printf("[Hub] broadcast: dropped %d/%d messages on channel %q (subscriber buffer full)", dropped, len(subs), channelName)
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
