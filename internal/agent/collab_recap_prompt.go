package agent

import (
	"fmt"
	"strings"

	"github.com/camronwood/neural-junkie/internal/ai"
	"github.com/camronwood/neural-junkie/internal/protocol"
)

const collabRecapPromptMaxBytes = 8000

// buildCollabRecapPrompt builds a compact prompt for collaboration_recap turns.
// Full collab system prompts (plan artifact, workspace trees, MCP tools) blow past
// Ollama context limits and cause timeouts before finalize.
func buildCollabRecapPrompt(agentName string, msg *protocol.Message) string {
	var system strings.Builder
	system.WriteString(fmt.Sprintf("You are %s in Neural Junkie.\n", agentName))
	system.WriteString("Write a concise user-facing collaboration recap in markdown (under 400 words).\n")
	system.WriteString("Include: goal, key decisions or accomplishments, deliverables/files, open questions, next steps for the user.\n")
	system.WriteString("Do NOT emit TASK_STATUS lines, new plan blocks, or @mention other agents unless quoting them.\n")

	user := strings.TrimSpace(msg.Content)
	if len(user) > collabRecapPromptMaxBytes {
		user = user[:collabRecapPromptMaxBytes] + "\n…(recap context truncated)"
	}
	return system.String() + ai.SystemPromptSeparator + user
}

func isCollabRecapMessage(msg *protocol.Message) bool {
	return msg != nil && msg.Type == protocol.MessageTypeCollabRecap
}
