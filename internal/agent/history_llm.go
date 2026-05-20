package agent

import (
	"time"

	"github.com/camronwood/neural-junkie/internal/chatcontext"
	"github.com/camronwood/neural-junkie/internal/protocol"
)

// MaxLLMHistoryMessages is the number of prior channel messages sent to the LLM
// (excluding the current turn, which lives in the prompt user section).
const MaxLLMHistoryMessages = 10

// unrespondedHistoryMaxAge limits replay of stale user messages when an agent joins a channel after restore.
const unrespondedHistoryMaxAge = 20 * time.Minute

// historyForGeneration returns channel history suitable for the model: omits noise,
// excludes the message being answered, and keeps the most recent MaxLLMHistoryMessages.
func historyForGeneration(history []*protocol.Message, excludeID string) []*protocol.Message {
	if len(history) == 0 {
		return history
	}
	out := make([]*protocol.Message, 0, len(history))
	for _, m := range history {
		if m == nil || (excludeID != "" && m.ID == excludeID) {
			continue
		}
		if omitMessageFromLLMHistory(m) {
			continue
		}
		out = append(out, m)
	}
	return chatcontext.TrimTail(out, MaxLLMHistoryMessages)
}

func omitMessageFromLLMHistory(m *protocol.Message) bool {
	return chatcontext.OmitFromLLMHistory(m)
}

// agentRespondedToUser reports whether the agent already addressed userMsg after it appeared in history.
func agentRespondedToUser(history []*protocol.Message, userIdx int, agentID, agentName, userMsgID string) bool {
	for j := userIdx + 1; j < len(history); j++ {
		m := history[j]
		if m == nil || !messageFromAgent(m, agentID, agentName) {
			continue
		}
		switch m.Type {
		case protocol.MessageTypeChat, protocol.MessageTypeAnswer, protocol.MessageTypeCollabDiscussion:
			return true
		case protocol.MessageTypeSystemInfo:
			if userMsgID != "" && m.ReplyTo == userMsgID {
				return true
			}
		}
	}
	return false
}

func messageFromAgent(m *protocol.Message, agentID, agentName string) bool {
	if m.From.ID == agentID {
		return true
	}
	return agentName != "" && m.From.Name == agentName
}

func messageTooOldForUnansweredReplay(m *protocol.Message) bool {
	if m == nil || m.Timestamp.IsZero() {
		return false
	}
	return time.Since(m.Timestamp) > unrespondedHistoryMaxAge
}

// recentUserHistoryOnly keeps the last n user messages (for compact retry).
func recentUserHistoryOnly(history []*protocol.Message, n int) []*protocol.Message {
	if n <= 0 || len(history) == 0 {
		return nil
	}
	var users []*protocol.Message
	for _, m := range history {
		if m != nil && protocol.IsUserLikeSender(m.From) && !omitMessageFromLLMHistory(m) {
			users = append(users, m)
		}
	}
	if len(users) <= n {
		return users
	}
	return users[len(users)-n:]
}
