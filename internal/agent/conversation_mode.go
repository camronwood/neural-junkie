package agent

import (
	"regexp"
	"strings"

	semantic "github.com/camronwood/neural-junkie/internal/intent"
	"github.com/camronwood/neural-junkie/internal/protocol"
)

const (
	MetadataConversationMode = "conversation_mode"

	ConversationModeChat    = "chat"
	ConversationModeCode    = "code"
	ConversationModeCollab  = "collab"
	ConversationModeClarify = "clarify"
)

var (
	taskVerbRE = regexp.MustCompile(`(?i)\b(review|refactor|debug|fix|implement|compile|lint|test|patch|edit|change|update|add|remove|rewrite|optimize|trace|diff|analyze|analyse)\b`)
	strongTaskVerbRE = regexp.MustCompile(`(?i)\b(refactor|debug|implement|compile|lint|patch|rewrite|trace|diff)\b`)
	filePathRE = regexp.MustCompile("(?:^|[\\s\"'`(])([./]?(?:[a-zA-Z0-9_-]+/)+[a-zA-Z0-9_-]+\\.[a-zA-Z0-9]+)")
	greetingRE = regexp.MustCompile(`(?i)^(?:@\w+\s+)?(?:hi|hello|hey|yo|sup|what'?s up|howdy|good (?:morning|afternoon|evening)|thanks|thank you|ok|okay|nice|cool)[!.?\s]*$`)
	hereMentionRE = regexp.MustCompile(`(?i)(?:^|[\s])@(?:here|channel|everyone)\b`)
	socialPingRE = regexp.MustCompile(`(?i)^(?:@(?:here|channel|everyone|\w+)\s+)*(?:what'?s\s+going\s+on|what\s+is\s+going\s+on|whats\s+up|how'?s\s+it\s+going|how\s+are\s+things|anyone\s+around|you\s+there|status\??|ping)[?!.\s]*$`)
	leadingMentionsRE = regexp.MustCompile(`(?i)^(?:\s*@(?:here|channel|everyone|\w+)\b)+\s*`)
	scanToolRE = regexp.MustCompile(`(?i)\b(summarize_scan_summary|summarize_scan_analysis|scan analysis|scan summary)\b`)
	editorOpenRE = regexp.MustCompile(`(?i)\b(file i have open|open in my editor|in my editor|editor open|active tab|active file|have open)\b`)
)

func stripLeadingMentions(content string) string {
	return strings.TrimSpace(leadingMentionsRE.ReplaceAllString(content, ""))
}

func hasHereOrChannelMention(content string) bool {
	return hereMentionRE.MatchString(content)
}

func hasStrongCodeTaskSignals(content string) bool {
	content = strings.TrimSpace(content)
	if content == "" {
		return false
	}
	if strongTaskVerbRE.MatchString(content) {
		return true
	}
	if filePathRE.MatchString(content) {
		return true
	}
	if strings.Contains(strings.ToLower(content), "@codebase") {
		return true
	}
	return hasScanOrEditorTaskSignals(content)
}

// isSocialOrStatusPing reports casual @here / vibe-check messages that should stay in chat mode.
func isSocialOrStatusPing(content string) bool {
	content = strings.TrimSpace(content)
	if content == "" {
		return false
	}
	if hasStrongCodeTaskSignals(content) {
		return false
	}
	stripped := stripLeadingMentions(content)
	if stripped == "" {
		return hasHereOrChannelMention(content)
	}
	if greetingRE.MatchString(content) || greetingRE.MatchString(stripped) {
		return true
	}
	if socialPingRE.MatchString(content) || socialPingRE.MatchString(stripped) {
		return true
	}
	if hasHereOrChannelMention(content) && len(stripped) <= 80 && !hasCodeTaskSignals(stripped) {
		return true
	}
	if semantic.LooksLikePresenceCheck(content) || semantic.LooksLikePresenceCheck(stripped) {
		return true
	}
	return false
}

func appendHereOrSocialPingPrompt(system *strings.Builder) {
	system.WriteString("=== CHANNEL PING ===\n")
	system.WriteString("This is a casual @here/@channel (or short status) ping. Reply in 1-3 sentences.\n")
	system.WriteString("Do not inventory the repo, invent file tours, or use tools unless the user explicitly asks for work.\n\n")
}

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
	if isWeakImplementationAffirmation(content) {
		return false
	}
	if userAffirmsPendingImplementation(content) {
		return true
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
	// Phrase/inference path is emergency rollback when no stamped turn decision exists.
	// Callers must prefer EffectiveConversationMode, which short-circuits on ExtractTurnDecision.
	if msg == nil {
		return ConversationModeChat
	}
	if channelType == protocol.ChannelTypeCollaboration {
		return ConversationModeCollab
	}
	content := strings.TrimSpace(msg.Content)
	if isSocialOrStatusPing(content) {
		return ConversationModeChat
	}
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
	if channelType == protocol.ChannelTypeCollaboration {
		return ConversationModeCollab
	}
	if decision, ok := protocol.ExtractTurnDecision(msg); ok {
		switch decision.Action {
		case semantic.ActionInspect, semantic.ActionDebug, semantic.ActionEdit, semantic.ActionRun, semantic.ActionContinue:
			return ConversationModeCode
		default:
			return ConversationModeChat
		}
	}
	if mode := ConversationModeFromMessage(msg); mode != "" {
		return mode
	}
	return inferConversationModeFromMessage(msg, channelType)
}

// ToolingConversationMode maps clarify → chat so agents ask before using tools.
func ToolingConversationMode(msg *protocol.Message, channelType protocol.ChannelType) string {
	mode := EffectiveConversationMode(msg, channelType)
	if mode == ConversationModeClarify {
		return ConversationModeChat
	}
	return mode
}

func appendConversationModeClarifyPrompt(system *strings.Builder) {
	system.WriteString("=== INTENT CLARIFICATION REQUIRED ===\n")
	system.WriteString("It is unclear whether the user wants a conversational answer or hands-on code/workspace work.\n")
	system.WriteString("Ask one short clarifying question in chat before using tools, editing files, or running commands.\n")
	system.WriteString("Offer clear choices, for example: discuss/explain vs edit/run in the workspace.\n")
	system.WriteString("Do not call tools or propose file edits until they choose.\n\n")
}

func appendConversationModeChatPrompt(system *strings.Builder) {
	system.WriteString("=== CHAT MODE (ADVISORY) ===\n")
	system.WriteString("Answer in conversation only. Do not edit files, run workspace tools, or dump repo findings.\n")
	system.WriteString("Retain named constraints from earlier turns (component names, section placement, accessibility rules).\n")
	system.WriteString("When summarizing a multi-turn design, restate those constraints explicitly in your reply.\n\n")
}

func appendGitInspectPrompt(system *strings.Builder, msg *protocol.Message) {
	if msg == nil || !semantic.LooksLikeGitInspectRequest(msg.Content) {
		return
	}
	decision, ok := protocol.ExtractTurnDecision(msg)
	if ok && decision.Action != semantic.ActionInspect && decision.Action != semantic.ActionDebug {
		return
	}
	system.WriteString("=== GIT INSPECTION (required this turn) ===\n")
	system.WriteString("The user asked to use git (status/history/diff/known-good). Do NOT speculate from memory.\n")
	system.WriteString("Before answering, call run_command with read-only git (start with `git status --short`, then `git log --oneline -n 20` and/or `git diff` / `git show` as needed).\n")
	system.WriteString("Ground the reply in that command output. Chat-only guesses about working configs or history are a failure.\n\n")
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
