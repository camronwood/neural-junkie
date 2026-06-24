package agent

import (
	"strings"

	"github.com/camronwood/neural-junkie/internal/protocol"
)

// assistantPublicShouldRespond applies passive public-channel rules for Assistant.
func assistantPublicShouldRespond(a *Agent, msg *protocol.Message) bool {
	if msg.From.ID == a.Info.ID {
		return false
	}

	a.backfillMentionsFromContent(msg)
	contentMentions := exclusiveMentionTokens(msg)

	if msg.HasMentions() || len(contentMentions) > 0 {
		if msg.IsMentioned(a.Info.ID) || a.isMentionedByContentTokens(contentMentions) {
			return true
		}
		return false
	}

	if strings.Contains(strings.ToLower(msg.Content), strings.ToLower(a.Info.Name)) {
		return true
	}

	return messageAsksAboutNJPlatformHelp(msg.Content)
}

// messageAsksAboutNJPlatformHelp reports NJ app/platform help intent (union of app
// markers and legacy chat-moderator keyword triggers).
func messageAsksAboutNJPlatformHelp(content string) bool {
	if messageAsksAboutNJApp(content) {
		return true
	}

	lower := strings.ToLower(strings.TrimSpace(content))
	if lower == "" {
		return false
	}

	chatKeywords := []string{
		"how do i", "how to",
		"command", "/",
		"mention", "@",
		"thread", "channel",
		"agent", "help",
		"create repo", "repo agent",
		"create expert", "custom expert",
		"chat feature", "chat room",
	}
	for _, keyword := range chatKeywords {
		if strings.Contains(lower, keyword) {
			return true
		}
	}
	return false
}
