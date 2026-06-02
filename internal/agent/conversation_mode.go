package agent

import (
	"regexp"
	"strings"

	"github.com/camronwood/neural-junkie/internal/protocol"
)

const (
	MetadataConversationMode = "conversation_mode"

	ConversationModeChat   = "chat"
	ConversationModeCode   = "code"
	ConversationModeCollab = "collab"
)

var (
	taskVerbRE = regexp.MustCompile(`(?i)\b(review|refactor|debug|fix|implement|compile|lint|test|patch|edit|change|update|add|remove|rewrite|optimize|trace|diff|analyze|analyse)\b`)
	filePathRE = regexp.MustCompile("(?:^|[\\s\"'`(])([./]?(?:[a-zA-Z0-9_-]+/)+[a-zA-Z0-9_-]+\\.[a-zA-Z0-9]+)")
	greetingRE = regexp.MustCompile(`(?i)^(?:@\w+\s+)?(?:hi|hello|hey|yo|sup|what'?s up|howdy|good (?:morning|afternoon|evening)|thanks|thank you|ok|okay|nice|cool)[!.?\s]*$`)
	scanToolRE = regexp.MustCompile(`(?i)\b(summarize_scan_summary|summarize_scan_analysis|scan analysis|scan summary)\b`)
	editorOpenRE = regexp.MustCompile(`(?i)\b(file i have open|open in my editor|in my editor|editor open|active tab|active file|have open)\b`)
)

// ConversationModeFromMessage returns chat/code/collab from outbound metadata (empty if unset).
func ConversationModeFromMessage(msg *protocol.Message) string {
	if msg == nil || msg.Metadata == nil {
		return ""
	}
	mode, _ := msg.Metadata[MetadataConversationMode].(string)
	return strings.TrimSpace(strings.ToLower(mode))
}

func hasScanOrEditorTaskSignals(content string) bool {
	content = strings.TrimSpace(content)
	if content == "" {
		return false
	}
	if scanToolRE.MatchString(content) {
		return true
	}
	if editorOpenRE.MatchString(content) {
		return true
	}
	return userRequestsEditorDocumentReview(content)
}

func hasCodeTaskSignals(content string) bool {
	content = strings.TrimSpace(content)
	if content == "" {
		return false
	}
	if hasScanOrEditorTaskSignals(content) {
		return true
	}
	if strings.Contains(strings.ToLower(content), "@codebase") {
		return true
	}
	if taskVerbRE.MatchString(content) {
		return true
	}
	if filePathRE.MatchString(content) {
		return true
	}
	if strings.Contains(content, "`") {
		return true
	}
	return false
}

func inferConversationModeFromMessage(msg *protocol.Message, channelType protocol.ChannelType) string {
	if msg == nil {
		return ConversationModeChat
	}
	if channelType == protocol.ChannelTypeCollaboration {
		return ConversationModeCollab
	}
	content := strings.TrimSpace(msg.Content)
	if hasCodeTaskSignals(content) {
		return ConversationModeCode
	}
	if greetingRE.MatchString(content) {
		return ConversationModeChat
	}
	if strings.Contains(content, "?") && len(content) >= 20 && !hasCodeTaskSignals(content) {
		return ConversationModeChat
	}
	if scope := ContextScopeFromMessage(msg); scope == ContextScopeNone && !hasCodeTaskSignals(content) {
		return ConversationModeChat
	}
	return ConversationModeCode
}

// EffectiveConversationMode returns metadata mode or server-side inference.
func EffectiveConversationMode(msg *protocol.Message, channelType protocol.ChannelType) string {
	if mode := ConversationModeFromMessage(msg); mode != "" {
		return mode
	}
	return inferConversationModeFromMessage(msg, channelType)
}

func ContextScopeFromMessage(msg *protocol.Message) string {
	if msg == nil || msg.Metadata == nil {
		return ""
	}
	scope, _ := msg.Metadata[MetadataContextScope].(string)
	return strings.TrimSpace(strings.ToLower(scope))
}

func shouldMaintainSessionSummary(chType protocol.ChannelType, channel string) bool {
	if chType == protocol.ChannelTypeDM || chType == protocol.ChannelTypeCustom || chType == protocol.ChannelTypePublic {
		return true
	}
	channel = strings.TrimSpace(strings.ToLower(channel))
	return strings.HasPrefix(channel, "dm-")
}
