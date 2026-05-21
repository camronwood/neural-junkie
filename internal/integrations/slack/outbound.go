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
	if msg.From.ID != binding.AgentID {
		return false
	}
	if msg.Metadata != nil {
		if src, _ := msg.Metadata["source"].(string); src == "slack" {
			return false
		}
	}
	switch msg.Type {
	case protocol.MessageTypeAnswer, protocol.MessageTypeChat:
		return strings.TrimSpace(msg.Content) != ""
	case protocol.MessageTypeFileChange, protocol.MessageTypeToolApproval:
		return true
	default:
		return false
	}
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
	if binding == nil || !binding.ReplyInThread {
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
	return ""
}
