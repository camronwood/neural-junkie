package slack

import (
	"strings"

	"github.com/camronwood/neural-junkie/internal/protocol"
)

const slackMaxText = 3900

// ShouldPostToSlack returns whether a hub message should be mirrored to Slack.
func ShouldPostToSlack(msg *protocol.Message, binding *Binding) bool {
	if msg == nil || binding == nil || !binding.Enabled {
		return false
	}
	if msg.Channel != binding.NJChannel {
		return false
	}
	if isSlackInboundEcho(msg) {
		return false
	}
	switch msg.Type {
	case protocol.MessageTypeAnswer, protocol.MessageTypeChat, protocol.MessageTypeQuestion:
		if strings.TrimSpace(msg.Content) == "" && ExtractGeneratedImage(msg) == nil {
			return false
		}
	case protocol.MessageTypeFileChange, protocol.MessageTypeToolApproval:
		if msg.From.ID != binding.AgentID {
			return false
		}
		return true
	default:
		return false
	}
	if msg.From.ID == binding.AgentID {
		return true
	}
	return isNJHumanSender(msg)
}

// isSlackInboundEcho skips messages that originated from Slack (avoid loops).
func isSlackInboundEcho(msg *protocol.Message) bool {
	if strings.HasPrefix(msg.From.ID, "slack:") {
		return true
	}
	if msg.Metadata != nil {
		if src, _ := msg.Metadata["source"].(string); src == "slack" || src == "slack_inbox" || src == "slack_human_dm" {
			return true
		}
	}
	return false
}

// isNJHumanSender is a hub user replying from NJ (not a Slack mirror identity).
func isNJHumanSender(msg *protocol.Message) bool {
	return protocol.IsUserLikeSender(msg.From) && !strings.HasPrefix(msg.From.ID, "slack:")
}

// OutboundSlackUsername picks the chat.postMessage display name (chat:write.customize).
func OutboundSlackUsername(msg *protocol.Message, binding *Binding, bridgeDisplayName string) string {
	if msg != nil && binding != nil && msg.From.ID == binding.AgentID {
		if n := strings.TrimSpace(msg.From.Name); n != "" {
			return sanitizeSlackPostUsername(n)
		}
	}
	if isNJHumanSender(msg) {
		if n := strings.TrimSpace(msg.From.Name); n != "" {
			return sanitizeSlackPostUsername(n)
		}
	}
	return sanitizeSlackPostUsername(bridgeDisplayName)
}

func sanitizeSlackPostUsername(name string) string {
	name = strings.TrimSpace(name)
	if name == "" {
		return ""
	}
	if len(name) > 80 {
		return name[:80]
	}
	return name
}

// FormatSlackText prepares text for chat.postMessage.
func FormatSlackText(msg *protocol.Message) string {
	switch msg.Type {
	case protocol.MessageTypeFileChange:
		return "Neural Junkie: a file change needs approval in the desktop app."
	case protocol.MessageTypeToolApproval:
		return "Neural Junkie: a tool call needs approval in the desktop app."
	default:
		return splitForSlack(msg.Content)
	}
}

func splitForSlack(content string) string {
	content = strings.TrimSpace(content)
	if len(content) <= slackMaxText {
		return content
	}
	var parts []string
	for len(content) > slackMaxText {
		parts = append(parts, content[:slackMaxText])
		content = content[slackMaxText:]
	}
	if content != "" {
		parts = append(parts, content)
	}
	return strings.Join(parts, "\n\n…\n\n")
}

// ThreadTSForOutbound resolves Slack thread_ts for a hub message.
func ThreadTSForOutbound(msg *protocol.Message, threads *ThreadMap, binding *Binding) string {
	if binding == nil {
		return ""
	}
	if threads != nil && msg.ReplyTo != "" {
		if ts := threads.SlackTSForNJMessage(msg.ReplyTo); ts != "" {
			return ts
		}
	}
	if !binding.ReplyInThread {
		return ""
	}
	if msg.IsThreadReply || msg.ThreadID != "" {
		tid := msg.GetThreadID()
		if tid != "" {
			if ts := threads.SlackThreadTS(tid); ts != "" {
				return ts
			}
		}
	}
	if msg.Metadata != nil {
		if ts, ok := msg.Metadata["slack_thread_ts"].(string); ok && ts != "" {
			return ts
		}
	}
	// NJ UI replies (especially from humans) often lack ReplyTo/ThreadID; keep Slack conversation threaded.
	if threads != nil && binding.ReplyInThread {
		if ts := threads.ChannelParentTS(binding.SlackChannelID); ts != "" {
			return ts
		}
	}
	return ""
}

// ShouldPostInboxToSlack returns whether a hub inbox message should post to Slack.
func ShouldPostInboxToSlack(msg *protocol.Message, inbox *InboxConfig) bool {
	if msg == nil || inbox == nil || !inbox.Enabled {
		return false
	}
	if msg.Channel != inbox.NJChannel {
		return false
	}
	if isSlackInboundEcho(msg) {
		return false
	}
	switch msg.Type {
	case protocol.MessageTypeAnswer, protocol.MessageTypeChat, protocol.MessageTypeQuestion:
		if strings.TrimSpace(msg.Content) == "" && ExtractGeneratedImage(msg) == nil {
			return false
		}
	case protocol.MessageTypeFileChange, protocol.MessageTypeToolApproval:
		if msg.From.ID != inbox.AgentID {
			return false
		}
		return true
	case protocol.MessageTypeSystemInfo:
		// Relay generation failures (e.g. Ollama down) so Slack inbox / human-DM away
		// users see feedback instead of silence.
		if msg.From.ID != inbox.AgentID || strings.TrimSpace(msg.Content) == "" {
			return false
		}
		return msg.GetErrorCode() != ""
	default:
		return false
	}
	if msg.From.ID == inbox.AgentID {
		return true
	}
	return isNJHumanSender(msg)
}

// InboxOutboundTarget resolves Slack channel and thread for an inbox hub message.
func InboxOutboundTarget(msg *protocol.Message, inbox *InboxConfig, threads *ThreadMap) (channelID, threadTS string) {
	if msg == nil || inbox == nil {
		return "", ""
	}
	if ch := msg.SlackReplyChannelID(); ch != "" {
		return ch, msg.SlackReplyThreadTS()
	}
	if threads != nil {
		if msg.ReplyTo != "" {
			if ch, ts, _, ok := threads.InboxReplyRoute(msg.ReplyTo); ok {
				return ch, ts
			}
		}
		tid := msg.GetThreadID()
		if tid == "" {
			tid = msg.ThreadID
		}
		if tid != "" {
			if ch, ts, _, ok := threads.InboxReplyRoute(tid); ok {
				return ch, ts
			}
			if ts := threads.SlackThreadTS(tid); ts != "" {
				dm := inbox.SlackDMChannelID
				if dm == "" {
					dm = chFromInboxDefault(inbox, msg)
				}
				return dm, ts
			}
		}
	}
	dm := inbox.SlackDMChannelID
	if dm == "" {
		dm = chFromInboxDefault(inbox, msg)
	}
	ts := ""
	if threads != nil && dm != "" {
		ts = threads.ChannelParentTS(dm)
	}
	return dm, ts
}

func chFromInboxDefault(inbox *InboxConfig, msg *protocol.Message) string {
	if msg != nil && msg.Metadata != nil {
		if ch, _ := msg.Metadata[protocol.SlackMetaReplyChannelID].(string); ch != "" {
			return ch
		}
	}
	return inbox.SlackDMChannelID
}

// OutboundInboxSlackUsername picks display name for inbox outbound posts.
func OutboundInboxSlackUsername(msg *protocol.Message, inbox *InboxConfig, bridgeDisplayName string) string {
	if msg != nil && inbox != nil && msg.From.ID == inbox.AgentID {
		if n := strings.TrimSpace(msg.From.Name); n != "" {
			return sanitizeSlackPostUsername(n)
		}
	}
	if isNJHumanSender(msg) {
		if n := strings.TrimSpace(msg.From.Name); n != "" {
			return sanitizeSlackPostUsername(n)
		}
	}
	return sanitizeSlackPostUsername(bridgeDisplayName)
}

// InboxOutboundHumanDM reports whether an agent reply should post via the user token.
func InboxOutboundHumanDM(msg *protocol.Message, threads *ThreadMap) bool {
	if msg == nil || threads == nil {
		return false
	}
	if msg.ReplyTo != "" {
		if _, _, human, ok := threads.InboxReplyRoute(msg.ReplyTo); ok && human {
			return true
		}
	}
	tid := msg.GetThreadID()
	if tid == "" {
		tid = msg.ThreadID
	}
	if tid != "" {
		if _, _, human, ok := threads.InboxReplyRoute(tid); ok && human {
			return true
		}
	}
	return false
}

// FormatHumanDMOutboundText prefixes agent text for human-DM replies.
func FormatHumanDMOutboundText(msg *protocol.Message, inbox *InboxConfig) string {
	text := FormatSlackText(msg)
	if inbox == nil {
		return text
	}
	prefix := HumanDMReplyPrefix(inbox.HumanDMAway, inbox.OwnerSlackUserName)
	return prefix + text
}
