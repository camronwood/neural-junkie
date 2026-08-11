package hub

import (
	"log"
	"strings"

	"github.com/camronwood/neural-junkie/internal/agent"
	"github.com/camronwood/neural-junkie/internal/protocol"
	"github.com/camronwood/neural-junkie/internal/turnledger"
)

// channelMaintainsTurnLedger mirrors session-summary harness exclusions and
// includes collaboration channels for multi-agent long-conversation tracking.
func channelMaintainsTurnLedger(chType protocol.ChannelType, channel string) bool {
	if !ChannelMaintainsSessionSummary(channel) {
		return false
	}
	switch chType {
	case protocol.ChannelTypeDM, protocol.ChannelTypeCustom, protocol.ChannelTypePublic,
		protocol.ChannelTypeCollaboration:
		return true
	}
	channel = strings.TrimSpace(strings.ToLower(channel))
	return strings.HasPrefix(channel, "dm-")
}

// noteTurnLedger appends a durable turn-ledger row asynchronously.
func (h *Hub) noteTurnLedger(msg *protocol.Message) {
	if h == nil || msg == nil || strings.TrimSpace(msg.Channel) == "" {
		return
	}
	chType := h.GetChannelType(msg.Channel)
	if !channelMaintainsTurnLedger(chType, msg.Channel) {
		return
	}
	switch msg.Type {
	case protocol.MessageTypeStreamDelta, protocol.MessageTypeStreamEnd,
		protocol.MessageTypeAgentStatus, protocol.MessageTypeSystemInfo:
		return
	case protocol.MessageTypeQuestion, protocol.MessageTypeChat, protocol.MessageTypeAnswer:
		// record
	default:
		return
	}

	speaker := ""
	speakerType := "agent"
	if msg.From.Name != "" {
		speaker = msg.From.Name
	} else {
		speaker = msg.From.ID
	}
	if protocol.IsUserLikeSender(msg.From) {
		speakerType = "human"
	}

	goalID := ""
	h.mu.RLock()
	if st := h.conversationState[msg.Channel]; st != nil && st.CurrentGoal != nil {
		goalID = st.CurrentGoal.ID
	}
	h.mu.RUnlock()
	if speakerType == "human" {
		h.RememberConversationSurface(msg.Channel, goalID, msg.ID, msg.Content)
	}

	ev := turnledger.Entry{
		Channel:     msg.Channel,
		MessageID:   msg.ID,
		Speaker:     speaker,
		SpeakerType: speakerType,
		MsgType:     string(msg.Type),
		Mode:        metadataString(msg.Metadata, "conversation_mode"),
		Intent:      metadataString(msg.Metadata, "resolved_intent"),
		Excerpt:     msg.Content,
		GoalID:      goalID,
		CollabID:    msg.GetCollaborationID(),
		TraceID:     metadataString(msg.Metadata, "trace_id"),
	}
	channel := msg.Channel
	go func() {
		if err := turnledger.Append(channel, ev); err != nil {
			log.Printf("[Hub] turn ledger append failed channel=%s: %v", channel, err)
		}
	}()
}

// GetChannelTurnLedger returns the last limit turn-ledger entries for a channel
// in the agent.TurnLedgerRow shape used by the Memory-stage overlay.
func (h *Hub) GetChannelTurnLedger(channel string, limit int) []agent.TurnLedgerRow {
	if h == nil || strings.TrimSpace(channel) == "" {
		return nil
	}
	entries, err := turnledger.ReadTail(channel, limit)
	if err != nil {
		log.Printf("[Hub] turn ledger read failed channel=%s: %v", channel, err)
		return nil
	}
	out := make([]agent.TurnLedgerRow, 0, len(entries))
	for _, e := range entries {
		out = append(out, agent.TurnLedgerRow{
			Speaker:     e.Speaker,
			SpeakerType: e.SpeakerType,
			Excerpt:     e.Excerpt,
			Entities:    e.Entities,
		})
	}
	return out
}

// GetChannelTurnLedgerRaw returns durable ledger entries for API/debug surfaces.
func (h *Hub) GetChannelTurnLedgerRaw(channel string, limit int) []turnledger.Entry {
	if h == nil || strings.TrimSpace(channel) == "" {
		return nil
	}
	entries, err := turnledger.ReadTail(channel, limit)
	if err != nil {
		log.Printf("[Hub] turn ledger read failed channel=%s: %v", channel, err)
		return nil
	}
	return entries
}
