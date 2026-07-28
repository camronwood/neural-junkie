package hub

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"strings"
	"time"

	"github.com/camronwood/neural-junkie/internal/agent"
	"github.com/camronwood/neural-junkie/internal/chatcontext"
	"github.com/camronwood/neural-junkie/internal/config"
	"github.com/camronwood/neural-junkie/internal/protocol"
)

const (
	summaryRefreshUserTurns   = 3
	summaryTranscriptMessages = 12
)

type summaryRefreshInput struct {
	Prompt        string
	LastMessageID string
	BaseVersion   int
}

func channelMaintainsSessionSummary(chType protocol.ChannelType, channel string) bool {
	if !ChannelMaintainsSessionSummary(channel) {
		return false
	}
	if chType == protocol.ChannelTypeDM || chType == protocol.ChannelTypeCustom || chType == protocol.ChannelTypePublic {
		return true
	}
	channel = strings.TrimSpace(strings.ToLower(channel))
	return strings.HasPrefix(channel, "dm-")
}

// noteChannelActivity updates turn counters and may schedule an async summary refresh.
func (h *Hub) noteChannelActivity(msg *protocol.Message) {
	if msg == nil || strings.TrimSpace(msg.Channel) == "" {
		return
	}
	chType := h.GetChannelType(msg.Channel)
	if !channelMaintainsSessionSummary(chType, msg.Channel) {
		return
	}
	if msg.Type == protocol.MessageTypeStreamDelta ||
		msg.Type == protocol.MessageTypeStreamEnd ||
		msg.Type == protocol.MessageTypeAgentStatus {
		return
	}

	h.mu.Lock()
	channel := msg.Channel
	var input summaryRefreshInput
	var gen uint64
	var genFn ChannelSummaryGenerator

	switch {
	case protocol.IsUserLikeSender(msg.From) &&
		(msg.Type == protocol.MessageTypeQuestion || msg.Type == protocol.MessageTypeChat):
		st := h.ensureChannelContextLocked(channel)
		st.UserTurns++
		if agent.ShouldForceSessionSummaryRefreshForMessage(msg) {
			st.UserTurns = summaryRefreshUserTurns
		}

	case msg.Type == protocol.MessageTypeSystemInfo &&
		strings.Contains(msg.Content, "Applied change"):
		if h.channelSummaryGen != nil {
			input = h.summaryRefreshInputLocked(channel)
			if input.Prompt != "" {
				gen = h.bumpSummaryRefreshGenLocked(channel)
				genFn = h.channelSummaryGen
				st := h.ensureChannelContextLocked(channel)
				st.UserTurns = 0
			}
		}

	case !protocol.IsUserLikeSender(msg.From) &&
		(msg.Type == protocol.MessageTypeChat || msg.Type == protocol.MessageTypeAnswer):
		st := h.ensureChannelContextLocked(channel)
		shouldRefresh := st.UserTurns >= summaryRefreshUserTurns
		if agent.ShouldForceSessionSummaryRefreshOnAgentResponse(msg.Content) {
			shouldRefresh = true
		}
		if !shouldRefresh && st.Summary == "" {
			filtered := chatcontext.FilterForLLM(h.messages[channel], "", 0)
			shouldRefresh = len(filtered) >= 4
		}
		if shouldRefresh && h.channelSummaryGen != nil {
			input = h.summaryRefreshInputLocked(channel)
			if input.Prompt != "" {
				gen = h.bumpSummaryRefreshGenLocked(channel)
				genFn = h.channelSummaryGen
				st.UserTurns = 0
			}
		}
	}
	h.mu.Unlock()

	if genFn != nil && input.Prompt != "" {
		go h.runSummaryRefresh(channel, gen, input, genFn)
	}
}

func (h *Hub) summaryRefreshInputLocked(channel string) summaryRefreshInput {
	msgs := h.messages[channel]
	if len(msgs) == 0 {
		return summaryRefreshInput{}
	}
	st := h.ensureChannelContextLocked(channel)
	start := 0
	if st.LastCompactedMessageID != "" {
		for i, msg := range msgs {
			if msg != nil && msg.ID == st.LastCompactedMessageID {
				start = i + 1
				break
			}
		}
	}
	superseded := map[string]bool{}
	if state := h.conversationState[channel]; state != nil {
		for id := range state.SupersededInstructions {
			superseded[id] = true
		}
	}
	delta := make([]*protocol.Message, 0, len(msgs)-start)
	lastID := ""
	for _, msg := range msgs[start:] {
		if msg == nil || superseded[msg.ID] || superseded[msg.ReplyTo] {
			continue
		}
		delta = append(delta, msg)
		lastID = msg.ID
	}
	transcript := chatcontext.FormatTranscript(delta, summaryTranscriptMessages)
	if transcript == "" {
		return summaryRefreshInput{}
	}
	stateJSON, _ := json.Marshal(cloneConversationState(h.conversationState[channel]))
	version := st.SummaryVersion
	if st.Summary != "" && version < 1 {
		version = 1
	}
	prompt := fmt.Sprintf(
		"Update the cumulative conversation digest. Preserve still-valid facts, decisions, corrections, open questions, and unfinished work. "+
			"Prefer dialogue facts (topic, constraints, user preferences) over coding task state when the transcript is casual chat. "+
			"Never restore instructions marked superseded.\n\nPREVIOUS DIGEST (v%d):\n%s\n\nSTRUCTURED STATE:\n%s\n\nTRANSCRIPT DELTA:\n%s",
		version, strings.TrimSpace(st.Summary), stateJSON, transcript,
	)
	return summaryRefreshInput{Prompt: prompt, LastMessageID: lastID, BaseVersion: version}
}

func (h *Hub) runSummaryRefresh(channel string, gen uint64, input summaryRefreshInput, genFn ChannelSummaryGenerator) {
	if genFn == nil {
		return
	}
	if h.isSummaryRefreshStale(channel, gen) {
		return
	}

	ctx, cancel := context.WithTimeout(context.Background(), config.SessionSummaryTimeout())
	defer cancel()

	type result struct {
		summary string
		err     error
	}
	done := make(chan result, 1)
	go func() {
		s, err := genFn(input.Prompt)
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
	summary = agent.ScrubStaleSessionSummary(summary, input.Prompt)

	h.mu.Lock()
	defer h.mu.Unlock()
	if h.isSummaryRefreshStaleLocked(channel, gen) {
		return
	}
	st := h.ensureChannelContextLocked(channel)
	st.Summary = summary
	st.SummaryVersion = input.BaseVersion + 1
	st.LastCompactedMessageID = input.LastMessageID
	st.UpdatedAt = time.Now()
	log.Printf("[Hub] session summary updated channel=%s version=%d len=%d", channel, st.SummaryVersion, len(summary))
}

// scheduleImmediateSummaryRefresh runs a summary pass without waiting for the next agent turn.
func (h *Hub) scheduleImmediateSummaryRefresh(channel string) {
	if h == nil || strings.TrimSpace(channel) == "" {
		return
	}
	chType := h.GetChannelType(channel)
	if !channelMaintainsSessionSummary(chType, channel) {
		return
	}

	h.mu.Lock()
	var input summaryRefreshInput
	var gen uint64
	var genFn ChannelSummaryGenerator
	if h.channelSummaryGen != nil {
		input = h.summaryRefreshInputLocked(channel)
		if input.Prompt != "" {
			gen = h.bumpSummaryRefreshGenLocked(channel)
			genFn = h.channelSummaryGen
			st := h.ensureChannelContextLocked(channel)
			st.UserTurns = 0
		}
	}
	h.mu.Unlock()

	if genFn != nil && input.Prompt != "" {
		go h.runSummaryRefresh(channel, gen, input, genFn)
	}
}
