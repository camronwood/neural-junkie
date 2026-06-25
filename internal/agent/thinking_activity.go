package agent

import (
	"log"
	"strings"

	"github.com/camronwood/neural-junkie/internal/ai"
	"github.com/camronwood/neural-junkie/internal/protocol"
)

func toolActivityDetail(ev ai.ToolStepEvent) string {
	name := strings.TrimSpace(ev.Name)
	if name == "" {
		name = "tool"
	}
	preview := strings.TrimSpace(ev.Preview)
	switch ev.Kind {
	case "start":
		if preview != "" {
			return name + " — " + preview
		}
		return name
	case "error":
		if preview != "" {
			return name + " failed — " + truncateThinkingDetail(preview)
		}
		return name + " failed"
	case "result", "done":
		if preview != "" {
			return name + " — " + truncateThinkingDetail(preview)
		}
		return name + " done"
	default:
		if preview != "" {
			return name + " — " + preview
		}
		return name
	}
}

func truncateThinkingDetail(s string) string {
	s = strings.TrimSpace(s)
	const max = 120
	if len(s) <= max {
		return s
	}
	return s[:max] + "…"
}

// sendThinkingActivity broadcasts a live activity label for the typing indicator.
func (a *Agent) sendThinkingActivity(originalMsg *protocol.Message, activity protocol.ThinkingActivity, detail ...string) {
	if originalMsg == nil || a.Hub == nil {
		return
	}
	statusMsg := protocol.NewMessage(
		protocol.MessageTypeAgentStatus,
		originalMsg.Channel,
		a.Info,
		"",
	)
	if statusMsg.Metadata == nil {
		statusMsg.Metadata = make(map[string]interface{})
	}
	statusMsg.Metadata["thinking_status"] = string(protocol.ThinkingStatusStarted)
	statusMsg.Metadata["question_id"] = originalMsg.ID
	if activity != "" {
		statusMsg.Metadata["thinking_activity"] = string(activity)
	}
	if len(detail) > 0 {
		if d := strings.TrimSpace(detail[0]); d != "" {
			statusMsg.Metadata[protocol.MetadataThinkingActivityDetail] = d
		}
	}
	go func() {
		if err := a.Hub.SendMessage(statusMsg); err != nil {
			log.Printf("[%s] Warning: failed to send thinking activity: %v", a.Info.Name, err)
		}
	}()
}
