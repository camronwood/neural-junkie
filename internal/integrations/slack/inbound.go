package slack

import (
	"strings"

	"github.com/camronwood/neural-junkie/internal/config"
	"github.com/camronwood/neural-junkie/internal/protocol"
)

// InboundInput is normalized Slack message data for hub ingestion.
type InboundInput struct {
	WorkspaceID    string
	ChannelID      string
	UserID         string
	UserName       string
	Text           string
	SlackTS        string
	ThreadTS       string
	IsAppMention   bool
	BotID          string
	Subtype        string
}

// ShouldIgnoreInbound returns true for bot/self/system messages.
func ShouldIgnoreInbound(in InboundInput, botUserID string) bool {
	if in.BotID != "" {
		return true
	}
	if in.Subtype == "bot_message" || in.Subtype == "message_changed" || in.Subtype == "message_deleted" {
		return true
	}
	if botUserID != "" && in.UserID == botUserID {
		return true
	}
	text := strings.TrimSpace(in.Text)
	return text == ""
}

// ShouldTriggerAgent decides whether to forward the message to the hub.
func ShouldTriggerAgent(in InboundInput, b *Binding, botUserID string) bool {
	if b == nil || !b.Enabled {
		return false
	}
	if in.IsAppMention {
		return true
	}
	switch b.Policy {
	case config.SlackPolicyAlways:
		return true
	case config.SlackPolicyQuestions:
		return looksLikeQuestion(in.Text)
	case config.SlackPolicyMentionOnly:
		return botUserID != "" && strings.Contains(in.Text, "<@"+botUserID+">")
	default:
		return in.IsAppMention
	}
}

func looksLikeQuestion(text string) bool {
	t := strings.ToLower(strings.TrimSpace(text))
	if strings.Contains(t, "?") {
		return true
	}
	for _, prefix := range []string{"how ", "what ", "why ", "when ", "where ", "can ", "should ", "help ", "please "} {
		if strings.HasPrefix(t, prefix) {
			return true
		}
	}
	return false
}

// BuildHubMessage converts Slack input into a protocol.Message for SendMessage.
func BuildHubMessage(in InboundInput, b *Binding, threads *ThreadMap, botUserID string) *protocol.Message {
	content := StripBotMention(strings.TrimSpace(in.Text), botUserID)
	threadID, replyTo, isThread := threads.ResolveInbound(in.ChannelID, in.SlackTS, in.ThreadTS)
	if isThread && threadID == in.ThreadTS {
		// First thread reply: parent ts becomes NJ thread root id placeholder until stored.
		threadID = in.ThreadTS
		replyTo = in.ThreadTS
	}

	from := protocol.AgentInfo{
		ID:   "slack:" + in.UserID,
		Name: in.UserName,
		Type: protocol.AgentTypeGeneral,
	}

	msgType := protocol.MessageTypeChat
	if looksLikeQuestion(content) {
		msgType = protocol.MessageTypeQuestion
	}

	msg := protocol.NewMessage(msgType, b.NJChannel, from, content)
	if isThread {
		msg.ThreadID = threadID
		msg.ReplyTo = replyTo
		msg.IsThreadReply = true
	}

	if msg.Metadata == nil {
		msg.Metadata = make(map[string]interface{})
	}
	msg.Metadata["source"] = "slack"
	msg.Metadata["slack_channel_id"] = in.ChannelID
	msg.Metadata["slack_ts"] = in.SlackTS
	msg.Metadata["slack_user_id"] = in.UserID
	if in.ThreadTS != "" {
		msg.Metadata["slack_thread_ts"] = in.ThreadTS
	}

	if ShouldMentionAgent(in, b, botUserID) {
		msg.Mentions = []string{b.AgentID}
	}

	return msg
}

// ShouldMentionAgent sets explicit agent mention for hub routing.
func ShouldMentionAgent(in InboundInput, b *Binding, botUserID string) bool {
	if in.IsAppMention {
		return true
	}
	switch b.Policy {
	case config.SlackPolicyMentionOnly:
		return botUserID != "" && strings.Contains(in.Text, "<@"+botUserID+">")
	case config.SlackPolicyAlways, config.SlackPolicyQuestions:
		return true
	default:
		return in.IsAppMention
	}
}

// StripBotMention removes leading bot mention from display content.
func StripBotMention(text, botUserID string) string {
	if botUserID == "" {
		return text
	}
	mention := "<@" + botUserID + ">"
	return strings.TrimSpace(strings.ReplaceAll(text, mention, ""))
}
