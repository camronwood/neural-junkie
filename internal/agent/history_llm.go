package agent

import (
	"strings"
	"time"

	"github.com/camronwood/neural-junkie/internal/chatcontext"
	"github.com/camronwood/neural-junkie/internal/mcp"
	"github.com/camronwood/neural-junkie/internal/protocol"
)

// MaxLLMHistoryMessages is the default tail channel history sent to the LLM
// when hub performance settings are unavailable.
const MaxLLMHistoryMessages = 10

func maxLLMHistoryMessages() int {
	if cfg := mcp.AppConfig(); cfg != nil {
		return cfg.Performance.MaxHistoryMessagesOrDefault()
	}
	return MaxLLMHistoryMessages
}

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
		// Clone so channel history keeps the original grounding opener while the
		// model sees the substantive body (required for multi-turn continuation).
		cp := *m
		if sanitized := chatcontext.SanitizeForLLMHistory(m.Content); sanitized != "" && sanitized != strings.TrimSpace(m.Content) {
			cp.Content = sanitized
		}
		out = append(out, &cp)
	}
	return chatcontext.TrimTail(out, maxLLMHistoryMessages())
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
			// Count any post-user system_info from this agent. ReplyTo is preferred, but
			// SQLite historically dropped reply_to on reload — missing ReplyTo must not
			// trigger catch-up replay of already-failed turns.
			if userMsgID == "" || m.ReplyTo == "" || m.ReplyTo == userMsgID {
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

// shortenedConversationWindow keeps the last n messages for quality retries
// so retries stay dialogue-aware instead of amnesiac.
func shortenedConversationWindow(history []*protocol.Message, n int) []*protocol.Message {
	return trimHistoryTail(history, n)
}

// recentUserHistoryOnly keeps the last n user messages (for compact retry helpers).
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
