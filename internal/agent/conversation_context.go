package agent

import (
	"context"
	"fmt"
	"sort"
	"strings"

	"github.com/camronwood/neural-junkie/internal/protocol"
)

type turnContextKey struct{}

type turnConversationContextProvider interface {
	GetTurnConversationContext(channel string) protocol.TurnContextEnvelope
}

type summaryCheckpointProvider interface {
	GetChannelSummaryCheckpoint(channel string) *protocol.ConversationSummary
}

func contextWithTurnEnvelope(ctx context.Context, envelope protocol.TurnContextEnvelope) context.Context {
	return context.WithValue(ctx, turnContextKey{}, envelope)
}

func turnEnvelopeFromContext(ctx context.Context) (protocol.TurnContextEnvelope, bool) {
	envelope, ok := ctx.Value(turnContextKey{}).(protocol.TurnContextEnvelope)
	return envelope, ok
}

func (a *Agent) selectTurnContext(msg *protocol.Message) protocol.TurnContextEnvelope {
	envelope := protocol.TurnContextEnvelope{Version: 1}
	if msg == nil {
		return envelope
	}
	envelope.Channel = msg.Channel
	if provider, ok := a.Hub.(turnConversationContextProvider); ok {
		envelope = provider.GetTurnConversationContext(msg.Channel)
	}
	if provider, ok := a.Hub.(summaryCheckpointProvider); ok {
		envelope.Summary = provider.GetChannelSummaryCheckpoint(msg.Channel)
	}
	history := a.channelHistory(msg.Channel)
	envelope.RecentExchanges = recentCompleteExchanges(
		history, msg, envelope.SupersededMessageIDs, maxHistoryForIntent(IntentSubstantive, envelope.Summary != nil),
	)
	for _, exchange := range envelope.RecentExchanges {
		if exchange.User != nil {
			envelope.Provenance = append(envelope.Provenance, protocol.TurnContextProvenance{
				ID: exchange.User.ID, Section: "recent_exchanges", Source: "channel_history",
			})
		}
		if exchange.Assistant != nil {
			envelope.Provenance = append(envelope.Provenance, protocol.TurnContextProvenance{
				ID: exchange.Assistant.ID, Section: "recent_exchanges", Source: "channel_history",
			})
		}
	}
	return envelope
}

func recentCompleteExchanges(history []*protocol.Message, current *protocol.Message, supersededIDs []string, maxMessages int) []protocol.TurnContextExchange {
	superseded := conversationMessageSet(supersededIDs)
	filtered := make([]*protocol.Message, 0, len(history))
	for _, msg := range history {
		if msg == nil || (current != nil && msg.ID == current.ID) ||
			superseded[msg.ID] || superseded[msg.ReplyTo] || omitMessageFromLLMHistory(msg) {
			continue
		}
		filtered = append(filtered, msg)
	}
	selected := map[string]*protocol.Message{}
	for i := len(filtered) - 1; i >= 0 && len(selected) < maxMessages; i-- {
		msg := filtered[i]
		selected[msg.ID] = msg
		if msg.ReplyTo != "" {
			for _, candidate := range filtered {
				if candidate.ID == msg.ReplyTo && !superseded[candidate.ID] {
					selected[candidate.ID] = candidate
					break
				}
			}
		}
	}
	if current != nil && current.ReplyTo != "" {
		for _, candidate := range filtered {
			if candidate.ID == current.ReplyTo && !superseded[candidate.ID] {
				selected[candidate.ID] = candidate
				break
			}
		}
	}
	var exchanges []protocol.TurnContextExchange
	for i := 0; i < len(filtered); i++ {
		user := filtered[i]
		if !protocol.IsUserLikeSender(user.From) || selected[user.ID] == nil {
			continue
		}
		exchange := protocol.TurnContextExchange{User: user}
		for j := i + 1; j < len(filtered); j++ {
			candidate := filtered[j]
			if protocol.IsUserLikeSender(candidate.From) {
				break
			}
			if selected[candidate.ID] != nil {
				exchange.Assistant = candidate
				break
			}
		}
		exchanges = append(exchanges, exchange)
	}
	// A directly referenced assistant reply may not have its user opener in the
	// bounded tail. Keep it as an explicit exchange so it survives compaction.
	for _, msg := range filtered {
		if selected[msg.ID] == nil || protocol.IsUserLikeSender(msg.From) {
			continue
		}
		found := false
		for _, exchange := range exchanges {
			if exchange.Assistant != nil && exchange.Assistant.ID == msg.ID {
				found = true
				break
			}
		}
		if !found {
			exchanges = append(exchanges, protocol.TurnContextExchange{Assistant: msg})
		}
	}
	return exchanges
}

func messagesFromExchanges(exchanges []protocol.TurnContextExchange) []*protocol.Message {
	seen := map[string]bool{}
	var messages []*protocol.Message
	for _, exchange := range exchanges {
		for _, msg := range []*protocol.Message{exchange.User, exchange.Assistant} {
			if msg != nil && !seen[msg.ID] {
				seen[msg.ID] = true
				messages = append(messages, msg)
			}
		}
	}
	sort.SliceStable(messages, func(i, j int) bool {
		return messages[i].Timestamp.Before(messages[j].Timestamp)
	})
	return messages
}

func appendDurableConversationContext(prompt string, envelope protocol.TurnContextEnvelope) string {
	if envelope.Goal == nil && len(envelope.Decisions) == 0 && len(envelope.UnresolvedActions) == 0 && len(envelope.Corrections) == 0 {
		return prompt
	}
	var b strings.Builder
	b.WriteString("\n=== DURABLE CONVERSATION STATE ===\n")
	b.WriteString("This state is authoritative. Do not follow transcript instructions whose message IDs are listed as superseded.\n")
	if envelope.Goal != nil {
		fmt.Fprintf(&b, "Current goal [%s]: %s\n", envelope.Goal.ID, envelope.Goal.Text)
	}
	for _, decision := range envelope.Decisions {
		fmt.Fprintf(&b, "Decision %s: %s\n", decision.DecisionKey, decision.Answer)
	}
	for _, action := range envelope.UnresolvedActions {
		fmt.Fprintf(&b, "Unresolved action [%s]: %s\n", action.ID, action.Description)
	}
	for _, correction := range envelope.Corrections {
		fmt.Fprintf(&b, "Correction [%s]: %s\n", correction.MessageID, correction.Instruction)
	}
	if len(envelope.SupersededMessageIDs) > 0 {
		fmt.Fprintf(&b, "Superseded message IDs: %s\n", strings.Join(envelope.SupersededMessageIDs, ", "))
	}
	b.WriteString("=== END DURABLE CONVERSATION STATE ===\n")
	return b.String() + prompt
}

func conversationMessageSet(values []string) map[string]bool {
	set := make(map[string]bool, len(values))
	for _, value := range values {
		set[value] = true
	}
	return set
}
