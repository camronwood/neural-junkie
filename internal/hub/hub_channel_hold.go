package hub

import (
	"time"

	"github.com/camronwood/neural-junkie/internal/protocol"
)

// ChannelHold records when a user interjected on a channel.
type ChannelHold struct {
	HeldAt time.Time `json:"held_at"`
	HeldBy string    `json:"held_by,omitempty"`
}

// IsChannelHeld reports whether agents should defer new turns on channel.
func (h *Hub) IsChannelHeld(channel string) bool {
	if h == nil || channel == "" {
		return false
	}
	h.mu.RLock()
	defer h.mu.RUnlock()
	_, ok := h.channelHolds[channel]
	return ok
}

// SetChannelHold sets or clears the hold flag for a channel.
func (h *Hub) SetChannelHold(channel string, held bool, heldBy string) {
	if h == nil || channel == "" {
		return
	}
	h.mu.Lock()
	if h.channelHolds == nil {
		h.channelHolds = make(map[string]ChannelHold)
	}
	if held {
		h.channelHolds[channel] = ChannelHold{
			HeldAt: time.Now().UTC(),
			HeldBy: heldBy,
		}
	} else {
		delete(h.channelHolds, channel)
	}
	h.mu.Unlock()
}

// broadcastChannelHold notifies subscribers of hold state (ephemeral agent_status).
func (h *Hub) broadcastChannelHold(channel string, held bool) {
	if h == nil || channel == "" {
		return
	}
	statusMsg := protocol.NewMessage(
		protocol.MessageTypeAgentStatus,
		channel,
		protocol.AgentInfo{ID: "system", Name: "System", Type: protocol.AgentTypeGeneral},
		"",
	)
	if statusMsg.Metadata == nil {
		statusMsg.Metadata = make(map[string]interface{})
	}
	statusMsg.Metadata[protocol.MetadataChannelHold] = held
	_ = h.SendMessage(statusMsg)
}

// broadcastChannelInterjectAbort signals external agent processes to cancel in-flight gens.
func (h *Hub) broadcastChannelInterjectAbort(channel string) {
	if h == nil || channel == "" {
		return
	}
	statusMsg := protocol.NewMessage(
		protocol.MessageTypeAgentStatus,
		channel,
		protocol.AgentInfo{ID: "system", Name: "System", Type: protocol.AgentTypeGeneral},
		"",
	)
	if statusMsg.Metadata == nil {
		statusMsg.Metadata = make(map[string]interface{})
	}
	statusMsg.Metadata[protocol.MetadataChannelInterjectAbort] = true
	_ = h.SendMessage(statusMsg)
}

// InterjectChannel pauses agent auto-continue on a channel (Cursor-style Stop).
func (h *Hub) InterjectChannel(channel, heldBy string) error {
	h.mu.RLock()
	_, exists := h.channels[channel]
	h.mu.RUnlock()
	if !exists {
		return &channelNotFoundError{channel: channel}
	}
	h.SetChannelHold(channel, true, heldBy)
	if h.commandHandler != nil {
		h.commandHandler.AbortRuntimeAgentsOnChannel(channel)
	}
	h.broadcastChannelInterjectAbort(channel)
	h.broadcastChannelHold(channel, true)
	return nil
}

type channelNotFoundError struct {
	channel string
}

func (e *channelNotFoundError) Error() string {
	return "channel " + e.channel + " not found"
}
