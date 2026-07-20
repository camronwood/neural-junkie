package agent

import (
	"fmt"
	"strings"

	"github.com/camronwood/neural-junkie/internal/ai"
	"github.com/camronwood/neural-junkie/internal/protocol"
)

// buildCLIBridgePrompt builds a slim prompt for CLI-backed agents (Claude Code, Codex, Cursor CLI, …).
// Those agents already have their own tools/persona; injecting Neural Junkie FILE_CHANGE / ask_user /
// MCP protocols causes persona bleed and confused refusals.
func (a *Agent) buildCLIBridgePrompt(msg *protocol.Message) string {
	var system strings.Builder
	system.WriteString(fmt.Sprintf("You are %s, a CLI coding agent invited into Neural Junkie chat.\n", a.Info.Name))
	system.WriteString(fmt.Sprintf("You are powered by %q via provider %q.\n", a.Info.AIModel, a.Info.AIProvider))
	system.WriteString("Reply as yourself using your own CLI tools and capabilities.\n")
	system.WriteString("Do NOT invent or follow Neural Junkie-native protocols (ask_user, propose_file_edit, FILE_CHANGE blocks, MCP tool JSON, or TASK_STATUS).\n")
	system.WriteString("Do NOT role-play as other agents or claim to be a different product.\n")
	system.WriteString("Keep replies concise unless the user asks for deep work.\n\n")

	if isSocialOrStatusPing(msg.Content) {
		appendHereOrSocialPingPrompt(&system)
	}

	collabInfo := a.getCollaborationContext(msg)
	if collabInfo.ID != "" {
		system.WriteString("=== COLLABORATION ===\n")
		system.WriteString(fmt.Sprintf("Goal: %s\n", collabInfo.Description))
		system.WriteString(fmt.Sprintf("Phase: %s\n", collabInfo.Phase))
		if role := strings.TrimSpace(collabInfo.AgentRole); role != "" {
			system.WriteString(fmt.Sprintf("Your role: %s\n", role))
		}
		system.WriteString("Focus on the collaboration goal; use your CLI tools for real file work when needed.\n\n")
	}

	AppendUserAndAgentRules(&system, msg, &a.Info, ResolveUserRulesHubFallback(msg), 0)
	a.appendMemoryForMessage(&system, msg, a.channelHistory(msg.Channel))
	AppendLearningsForMessage(&system, msg, &a.Info)

	var user strings.Builder
	if ws := a.resolveWorkspacePath(msg); ws != "" {
		user.WriteString(fmt.Sprintf("Workspace: %s\n\n", ws))
	}
	user.WriteString(strings.TrimSpace(msg.Content))
	user.WriteString("\n")
	AppendPromptAttachments(&user, msg)

	return system.String() + ai.SystemPromptSeparator + user.String()
}
