package chatcontext

import (
	"regexp"
	"strings"

	"github.com/camronwood/neural-junkie/internal/protocol"
)

// MaxTranscriptMessages is the default cap when formatting channel text for summary LLM calls.
const MaxTranscriptMessages = 12

var promptLeakSubstrings = []string{
	"provide the necessary context",
	"be concise but complete",
	"be mindful of the user's background",
	"for complex questions, use",
	"offer specific details when",
	// Note: do NOT include "grounding: i loaded" — required openers must stay in
	// history for multi-turn continuation (strip via SanitizeForLLMHistory instead).
	"the model returned an empty reply",
	"open model library",
}

var groundingLoadedLineRE = regexp.MustCompile(`(?im)^\s*Grounding:\s*I loaded \d+ file\(s\) from the workspace context for this answer\.\s*\n?`)

// SanitizeForLLMHistory strips instruction-echo prefixes while keeping substantive reply body.
func SanitizeForLLMHistory(content string) string {
	content = groundingLoadedLineRE.ReplaceAllString(content, "")
	return strings.TrimSpace(content)
}

// OmitFromLLMHistory reports whether a message should be excluded from LLM chat history.
func OmitFromLLMHistory(m *protocol.Message) bool {
	if m == nil {
		return true
	}
	if m.IsFromSystem() {
		return true
	}
	switch m.Type {
	case protocol.MessageTypeSystemInfo,
		protocol.MessageTypeAgentJoin,
		protocol.MessageTypeAgentLeave,
		protocol.MessageTypeAgentStatus,
		protocol.MessageTypeStreamDelta,
		protocol.MessageTypeStreamEnd,
		protocol.MessageTypeToolApproval,
		protocol.MessageTypeCommandSuggestion,
		protocol.MessageTypeContextShare,
		protocol.MessageTypeRequestHelp,
		protocol.MessageTypeDesignOutput,
		protocol.MessageTypeFileChange,
		protocol.MessageTypeCollabPlan,
		protocol.MessageTypeCollabTask,
		protocol.MessageTypeCollabStatus:
		return true
	}

	if m.Type == protocol.MessageTypeCommandOutput {
		return strings.TrimSpace(m.Content) == ""
	}

	if protocol.IsUserLikeSender(m.From) {
		switch m.Type {
		case protocol.MessageTypeQuestion, protocol.MessageTypeChat, protocol.MessageTypeAnswer:
			// keep
		default:
			return true
		}
	} else {
		switch m.Type {
		case protocol.MessageTypeChat, protocol.MessageTypeAnswer, protocol.MessageTypeCollabDiscussion:
			// keep
		default:
			return true
		}
	}

	c := SanitizeForLLMHistory(m.Content)
	if c == "" {
		return true
	}
	lower := strings.ToLower(c)
	if strings.HasPrefix(lower, "sorry, i encountered an error") {
		return true
	}
	if strings.HasPrefix(lower, "error:") {
		return true
	}
	for _, leak := range promptLeakSubstrings {
		if strings.Contains(lower, leak) {
			return true
		}
	}
	return false
}

// FilterForLLM returns messages suitable for provider history or summary transcripts.
func FilterForLLM(messages []*protocol.Message, excludeID string, max int) []*protocol.Message {
	if len(messages) == 0 {
		return nil
	}
	out := make([]*protocol.Message, 0, len(messages))
	for _, m := range messages {
		if m == nil || (excludeID != "" && m.ID == excludeID) {
			continue
		}
		if OmitFromLLMHistory(m) {
			continue
		}
		out = append(out, m)
	}
	return TrimTail(out, max)
}

// TrimTail keeps the last max messages when max > 0.
func TrimTail(messages []*protocol.Message, max int) []*protocol.Message {
	if max <= 0 || len(messages) <= max {
		return messages
	}
	return messages[len(messages)-max:]
}
