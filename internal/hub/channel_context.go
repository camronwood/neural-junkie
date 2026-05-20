package hub

import (
	"time"
)

// ChannelSummaryGenerator produces a rolling session summary from a transcript string.
type ChannelSummaryGenerator func(transcript string) (summary string, err error)

// ChannelContextState holds per-channel session summary metadata (hub-owned).
type ChannelContextState struct {
	Summary   string
	UpdatedAt time.Time
	UserTurns int
}

func (h *Hub) ensureChannelContextLocked(channel string) *ChannelContextState {
	if h.channelContext == nil {
		h.channelContext = make(map[string]*ChannelContextState)
	}
	st, ok := h.channelContext[channel]
	if !ok || st == nil {
		st = &ChannelContextState{}
		h.channelContext[channel] = st
	}
	return st
}

// SetChannelSummaryGenerator wires the utility LLM used for async session summaries.
func (h *Hub) SetChannelSummaryGenerator(gen ChannelSummaryGenerator, modelLabel string) {
	h.mu.Lock()
	defer h.mu.Unlock()
	h.channelSummaryGen = gen
	h.channelSummaryModel = modelLabel
}

// GetChannelSessionSummary returns the persisted summary for a channel (empty if none).
func (h *Hub) GetChannelSessionSummary(channel string) string {
	h.mu.RLock()
	defer h.mu.RUnlock()
	if h.channelContext == nil {
		return ""
	}
	st := h.channelContext[channel]
	if st == nil {
		return ""
	}
	return st.Summary
}

func (h *Hub) clearChannelContextLocked(channel string) {
	if h.channelContext != nil {
		delete(h.channelContext, channel)
	}
	if h.channelSummaryRefreshGen != nil {
		h.channelSummaryRefreshGen[channel]++
	}
}

func (h *Hub) bumpSummaryRefreshGenLocked(channel string) uint64 {
	if h.channelSummaryRefreshGen == nil {
		h.channelSummaryRefreshGen = make(map[string]uint64)
	}
	h.channelSummaryRefreshGen[channel]++
	return h.channelSummaryRefreshGen[channel]
}

func (h *Hub) isSummaryRefreshStaleLocked(channel string, gen uint64) bool {
	if h.channelSummaryRefreshGen == nil {
		return true
	}
	return h.channelSummaryRefreshGen[channel] != gen
}

func (h *Hub) isSummaryRefreshStale(channel string, gen uint64) bool {
	h.mu.RLock()
	defer h.mu.RUnlock()
	return h.isSummaryRefreshStaleLocked(channel, gen)
}
