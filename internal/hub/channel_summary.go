package hub

import (
	"context"
	"log"
	"strings"
	"time"

	"github.com/camronwood/neural-junkie/internal/chatcontext"
	"github.com/camronwood/neural-junkie/internal/protocol"
)

const (
	summaryRefreshUserTurns   = 3
	summaryTranscriptMessages = 12
	summaryLLMTimeout         = 30 * time.Second
)

// noteChannelActivity updates turn counters and may schedule an async summary refresh.
func (h *Hub) noteChannelActivity(msg *protocol.Message) {
	if msg == nil || strings.TrimSpace(msg.Channel) == "" {
		return
	}
	chType := h.GetChannelType(msg.Channel)
	if chType != protocol.ChannelTypeDM && chType != protocol.ChannelTypeCustom {
		return
	}
	if msg.Type == protocol.MessageTypeStreamDelta ||
		msg.Type == protocol.MessageTypeStreamEnd ||
		msg.Type == protocol.MessageTypeAgentStatus {
		return
	}

	h.mu.Lock()
	channel := msg.Channel
	var transcript string
	var gen uint64
	var genFn ChannelSummaryGenerator

	switch {
	case protocol.IsUserLikeSender(msg.From) &&
		(msg.Type == protocol.MessageTypeQuestion || msg.Type == protocol.MessageTypeChat):
		st := h.ensureChannelContextLocked(channel)
		st.UserTurns++

	case !protocol.IsUserLikeSender(msg.From) &&
		(msg.Type == protocol.MessageTypeChat || msg.Type == protocol.MessageTypeAnswer):
		st := h.ensureChannelContextLocked(channel)
		shouldRefresh := st.UserTurns >= summaryRefreshUserTurns
		if !shouldRefresh && st.Summary == "" {
			filtered := chatcontext.FilterForLLM(h.messages[channel], "", 0)
			shouldRefresh = len(filtered) >= 4
		}
		if shouldRefresh && h.channelSummaryGen != nil {
			transcript = h.transcriptForSummaryLocked(channel)
			if transcript != "" {
				gen = h.bumpSummaryRefreshGenLocked(channel)
				genFn = h.channelSummaryGen
				st.UserTurns = 0
			}
		}
	}
	h.mu.Unlock()

	if genFn != nil && transcript != "" {
		go h.runSummaryRefresh(channel, gen, transcript, genFn)
	}
}

func (h *Hub) transcriptForSummaryLocked(channel string) string {
	msgs := h.messages[channel]
	if len(msgs) == 0 {
		return ""
	}
	return chatcontext.FormatTranscript(msgs, summaryTranscriptMessages)
}

func (h *Hub) runSummaryRefresh(channel string, gen uint64, transcript string, genFn ChannelSummaryGenerator) {
	if genFn == nil {
		return
	}
	if h.isSummaryRefreshStale(channel, gen) {
		return
	}

	ctx, cancel := context.WithTimeout(context.Background(), summaryLLMTimeout)
	defer cancel()

	type result struct {
		summary string
		err     error
	}
	done := make(chan result, 1)
	go func() {
		s, err := genFn(transcript)
		done <- result{summary: s, err: err}
	}()

	var summary string
	var err error
	select {
	case <-ctx.Done():
		log.Printf("[Hub] session summary timeout channel=%s", channel)
		return
	case r := <-done:
		summary, err = r.summary, r.err
	}

	if h.isSummaryRefreshStale(channel, gen) {
		return
	}
	if err != nil {
		log.Printf("[Hub] session summary failed channel=%s: %v", channel, err)
		return
	}
	summary = strings.TrimSpace(summary)
	if summary == "" {
		return
	}

	h.mu.Lock()
	defer h.mu.Unlock()
	if h.isSummaryRefreshStaleLocked(channel, gen) {
		return
	}
	st := h.ensureChannelContextLocked(channel)
	st.Summary = summary
	st.UpdatedAt = time.Now()
	log.Printf("[Hub] session summary updated channel=%s len=%d", channel, len(summary))
}
