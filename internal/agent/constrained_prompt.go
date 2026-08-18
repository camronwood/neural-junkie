package agent

import (
	"fmt"
	"strings"

	"github.com/camronwood/neural-junkie/internal/ai"
	"github.com/camronwood/neural-junkie/internal/protocol"
)

func (a *Agent) buildConstrainedComposerPrompt(msg *protocol.Message, intent TurnIntent) string {
	if msg != nil && msg.IdeEditorModeIsPlan() {
		return a.buildComposerPlanPrompt(msg)
	}
	if msg != nil && (msg.IdeEditorModeIsAsk() || isAskModeReadOnly(msg)) {
		return a.buildComposerAskPrompt(msg)
	}
	return a.buildComposerAgentPrompt(msg, intent)
}

func (a *Agent) constrainedAgentName() string {
	name := "Assistant"
	if a != nil && strings.TrimSpace(a.Info.Name) != "" {
		name = a.Info.Name
	}
	return name
}

func (a *Agent) buildComposerAskPrompt(msg *protocol.Message) string {
	var system strings.Builder
	fmt.Fprintf(&system, "You are %s. Answer from the workspace and cited files.\n", a.constrainedAgentName())
	system.WriteString("=== ASK MODE (READ-ONLY) ===\n")
	system.WriteString("Explain and advise only. Do NOT propose file edits, call propose_file_edit, or emit [FILE_CHANGE] blocks.\n")
	system.WriteString("Ignore third-party libraries (site-packages, node_modules). Discard unrelated retrieved chunks.\n\n")
	if a != nil {
		AppendUserAndAgentRules(&system, msg, &a.Info, ResolveUserRulesHubFallback(msg), compactUserRulesMarkdownBytes)
	}
	var user strings.Builder
	AppendWorkspaceContext(&user, msg)
	AppendPromptAttachments(&user, msg)
	if msg != nil {
		user.WriteString(strings.TrimSpace(msg.Content))
		user.WriteString("\n")
	}
	return system.String() + ai.SystemPromptSeparator + user.String()
}

func (a *Agent) buildComposerAgentPrompt(msg *protocol.Message, _ TurnIntent) string {
	var system strings.Builder
	fmt.Fprintf(&system, "You are %s. Implement the user's request with workspace tools, not a lecture.\n", a.constrainedAgentName())
	system.WriteString("=== AGENT MODE ===\n")
	system.WriteString("Use read_file, grep, glob_file_search, and semantic_search to inspect code. Do not guess file contents.\n")
	system.WriteString("Edit with search_replace or apply_patch; create files with propose_file_edit. Cite existing paths.\n")
	system.WriteString("Ignore third-party libraries (site-packages, node_modules). Discard unrelated retrieved chunks.\n")
	system.WriteString("Do not paste a FILE_CHANGE protocol encyclopedia. Prefer native edit tools.\n\n")
	if a != nil {
		AppendUserAndAgentRules(&system, msg, &a.Info, ResolveUserRulesHubFallback(msg), compactUserRulesMarkdownBytes)
	}
	var user strings.Builder
	AppendWorkspaceContext(&user, msg)
	AppendPromptAttachments(&user, msg)
	if msg != nil {
		user.WriteString(strings.TrimSpace(msg.Content))
		user.WriteString("\n")
	}
	return system.String() + ai.SystemPromptSeparator + user.String()
}
