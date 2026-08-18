package agent

import (
	"fmt"
	"regexp"
	"strings"

	"github.com/camronwood/neural-junkie/internal/ai"
	"github.com/camronwood/neural-junkie/internal/protocol"
)

var planModeOutOfScopeRE = regexp.MustCompile(`(?i)out of scope`)

func appendPlanModePrompt(system *strings.Builder) {
	system.WriteString("=== PLAN MODE ===\n")
	system.WriteString("Research with read_file, grep, glob_file_search, semantic_search. Do not guess file contents. Cite existing paths.\n")
	system.WriteString("Ignore third-party libraries (site-packages, node_modules). Discard unrelated retrieved chunks.\n")
	system.WriteString("Do NOT propose file edits, call propose_file_edit, or emit [FILE_CHANGE] blocks.\n")
	system.WriteString("Reply with YAML frontmatter (name, overview, todos with id/content/status pending) then markdown. ")
	system.WriteString("MUST include a heading exactly `## Out of scope`. Optional mermaid. Todos start pending.\n\n")
	system.WriteString("```markdown\n")
	system.WriteString("---\n")
	system.WriteString("name: Short title\n")
	system.WriteString("overview: One or two sentences.\n")
	system.WriteString("todos:\n")
	system.WriteString("  - id: step-one\n")
	system.WriteString("    content: First step with a file path\n")
	system.WriteString("    status: pending\n")
	system.WriteString("isProject: false\n")
	system.WriteString("---\n\n")
	system.WriteString("# Title\n\n")
	system.WriteString("## Out of scope\n")
	system.WriteString("- Follow-ups this plan will not do.\n")
	system.WriteString("```\n\n")
}

func (a *Agent) buildComposerPlanPrompt(msg *protocol.Message) string {
	var system strings.Builder
	name := "Assistant"
	if a != nil && strings.TrimSpace(a.Info.Name) != "" {
		name = a.Info.Name
	}
	fmt.Fprintf(&system, "You are %s. Produce a structured implementation plan, not a lecture and not file edits.\n", name)
	appendPlanModePrompt(&system)
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

// ensurePlanModeStructure appends a required Out of scope heading when a YAML
// plan omitted it (common on small local models).
func ensurePlanModeStructure(response string) string {
	trimmed := strings.TrimSpace(response)
	if trimmed == "" {
		return response
	}
	if planModeOutOfScopeRE.MatchString(trimmed) {
		return response
	}
	if !strings.Contains(trimmed, "todos:") {
		return response
	}
	return strings.TrimRight(response, " \t\r\n") + "\n\n## Out of scope\n- Follow-ups not listed in todos.\n"
}
