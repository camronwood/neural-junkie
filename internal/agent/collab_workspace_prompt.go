package agent

import (
	"regexp"
	"strings"

	"github.com/camronwood/neural-junkie/internal/collaboration"
	"github.com/camronwood/neural-junkie/internal/protocol"
)

var rawToolJSONDiscussionRE = regexp.MustCompile(`(?s)^\s*\{\s*"name"\s*:\s*"[^"]+"\s*,\s*"arguments"\s*:\s*\{`)

func appendCollaborationWorkspaceInstructions(b *strings.Builder, info CollaborationInfo, agentType protocol.AgentType) {
	if b == nil {
		return
	}
	repo := strings.TrimSpace(info.SourceRepoPath)
	if repo == "" {
		b.WriteString("\n=== PROJECT WORKSPACE ===\n")
		b.WriteString("No user project workspace is bound to this collaboration yet. ")
		b.WriteString("Plans must ask the user to start /collaborate from an open project folder (or use --workspace). ")
		b.WriteString("Do NOT invent infrastructure (Kubernetes, Helm, etc.) without evidence from the user.\n")
		return
	}

	goal := strings.TrimSpace(info.Description)
	b.WriteString("\n=== PROJECT WORKSPACE (reference) ===\n")
	if goal != "" {
		b.WriteString("**Collaboration goal (answer this first):** ")
		b.WriteString(goal)
		b.WriteString("\n")
	}
	b.WriteString("**Project root:** `")
	b.WriteString(repo)
	b.WriteString("`\n")
	if rel := collaboration.ProjectCollabRelPath(info.ID); rel != "" {
		b.WriteString("**Deliverables folder:** `")
		b.WriteString(rel)
		b.WriteString("` — write outputs here when tasks require files.\n")
	}
	if out := strings.TrimSpace(info.WorkingDirectory); out != "" && out != repo {
		b.WriteString("**Execution directory:** `")
		b.WriteString(out)
		b.WriteString("`\n")
	}
	b.WriteString("\nWorkspace rules:\n")
	b.WriteString("- Lead with progress, decisions, and answers to the goal — not file inventories or repeated \"grounding\" preambles.\n")
	b.WriteString("- Cite specific paths only when they support a point (one or two examples is enough).\n")
	b.WriteString("- Do NOT recommend kubectl, Helm, or Kubernetes unless the file tree or open files show k8s/Helm usage.\n")
	b.WriteString("- Do NOT suggest or run docker-compose, npm, yarn, make, or similar build/deploy commands just because those files appear in the tree — use them only when the assigned task explicitly requires running or deploying something.\n")
	b.WriteString("- Do NOT start with \"Grounding: I loaded N files\"; the user cares about outcomes, not how many files you opened.\n")
	if focus := collaborationWorkspaceFocusHint(goal); focus != "" {
		b.WriteString(focus)
	}

	tree := fileTreeFromWorkspaceContext(info.SourceWorkspaceContext)
	if tree != "" && !stackSignalsKubernetes(tree) && agentType == protocol.AgentTypeDevOps {
		b.WriteString("- **Platform note:** The shared file tree does not show Kubernetes/Helm assets — focus on this repo's actual stack (languages, CI, AWS, apps) instead of cluster tooling.\n")
	}
}

func collaborationWorkspaceFocusHint(goal string) string {
	lower := strings.ToLower(strings.TrimSpace(goal))
	if lower == "" {
		return ""
	}
	if strings.Contains(lower, "resource") && strings.Contains(lower, "api") {
		return "- **Focus paths:** `resource-api/json_endpoints/` and `docs/tim/` — use the JSON endpoint descriptors there; do not invent `core/sample/main.go`, `index.js`, or `api.js` unless they appear in the file tree.\n"
	}
	if strings.Contains(lower, "schema") || strings.Contains(lower, "standardiz") || strings.Contains(lower, "registr") {
		return "- **Focus paths:** start with `resource-api/json_endpoints/` for schema/registration work; cite real filenames from the tree.\n"
	}
	if strings.Contains(lower, "core/sample/main.go") &&
		(strings.Contains(lower, "readme.md") || strings.Contains(lower, "readme")) &&
		(strings.Contains(lower, " only") || strings.Contains(lower, "only.")) {
		return "- **Focus paths:** `README.md` and `core/sample/main.go` only — do not cite or discuss `src/`, React, or frontend files unless the task names them.\n"
	}
	return ""
}

func fileTreeFromWorkspaceContext(ctx map[string]interface{}) string {
	if len(ctx) == 0 {
		return ""
	}
	if tree, ok := ctx["file_tree"].(string); ok {
		return tree
	}
	return ""
}

func stackSignalsKubernetes(fileTree string) bool {
	lower := strings.ToLower(fileTree)
	needles := []string{
		"kubernetes", "k8s", "helm", "chart.yaml", "charts/",
		"deployment.yaml", "kustomization", "/manifests/",
	}
	for _, n := range needles {
		if strings.Contains(lower, n) {
			return true
		}
	}
	return false
}

// isCollabTurnHandoffContent reports System prompts that target one participant.
func isCollabTurnHandoffContent(content string) bool {
	return strings.Contains(content, "Collaboration turn handoff") ||
		strings.Contains(content, "You're up first")
}

// isCollabTurnPromptForAgent returns true for System turn prompts that should wake
// the next collaboration participant (not seed banners marked collab_internal_event).
// Handoffs always carry an explicit Mentions entry for the next speaker — do not
// also admit every agent where IsAgentTurn happens to be true (duplicate replies).
func isCollabTurnPromptForAgent(msg *protocol.Message, collabID, agentID string, collab CollaborationClient) bool {
	if msg == nil || collab == nil || collabID == "" || agentID == "" {
		return false
	}
	if msg.Type != protocol.MessageTypeCollabDiscussion || !msg.IsFromSystem() {
		return false
	}
	if !isCollabTurnHandoffContent(msg.Content) {
		return false
	}
	return msg.IsMentioned(agentID)
}

// collabPlanningSuppressMCPTools hides DevOps MCP tool catalogs during planning and
// execution when the shared workspace does not look like a Kubernetes repo (doc/API collabs).
func collabPlanningSuppressMCPTools(info CollaborationInfo, agentType protocol.AgentType) bool {
	if agentType != protocol.AgentTypeDevOps {
		return false
	}
	switch info.Phase {
	case "planning", "executing":
		return !stackSignalsKubernetes(fileTreeFromWorkspaceContext(info.SourceWorkspaceContext))
	default:
		return false
	}
}

// sanitizeCollabDiscussionResponse replaces raw tool-call JSON mistaken for chat
// during collaboration planning (common with MCP-equipped agents).
func sanitizeCollabDiscussionResponse(content string, collabInfo CollaborationInfo, agentType protocol.AgentType) string {
	trimmed := strings.TrimSpace(content)
	if trimmed == "" {
		return content
	}
	if collabInfo.Phase == "executing" {
		return collaboration.SanitizeCollabExecutionResponse(content, collabInfo.Phase)
	}
	if collabInfo.Phase != "planning" {
		return content
	}
	looksToolJSON := rawToolJSONDiscussionRE.MatchString(trimmed) ||
		strings.Contains(strings.ToLower(trimmed), `"name": "task-add"`) ||
		strings.Contains(strings.ToLower(trimmed), "task-add")
	looksKubectl := strings.Contains(strings.ToLower(trimmed), "kubectl")
	if !looksToolJSON && !looksKubectl {
		return content
	}
	if agentType == protocol.AgentTypeDevOps || agentType == protocol.AgentTypeArchitecture {
		return "For this planning turn, respond in prose (not JSON tool calls). " +
			"Propose or refine tasks using lines like `- Task N: @AgentName - description` focused on " +
			"API document schema standardization, registration, and a markdown deliverable."
	}
	stripped := stripFencedToolJSONBlocks(content)
	if strings.TrimSpace(stripped) == "" {
		return "I'll refine the task list in prose using lines like `- Task N: @AgentName - description`."
	}
	return stripped
}

var fencedToolJSONBlockRE = regexp.MustCompile("(?is)```(?:json)?\\s*\\{[^`]*\"name\"\\s*:\\s*\"[^\"]+\"[^`]*\\}[^`]*```")

func stripFencedToolJSONBlocks(content string) string {
	out := fencedToolJSONBlockRE.ReplaceAllString(content, "")
	out = rawToolJSONDiscussionRE.ReplaceAllString(out, "")
	return strings.TrimSpace(out)
}

func collaborationTurnHandoffBody(phase string) string {
	switch phase {
	case "executing":
		return "Collaboration turn handoff: next participant, please continue your assigned task or answer execution Q&A. Do not reopen plan negotiation unless the user requested a revision."
	case "planning":
		return "Collaboration turn handoff: next participant, please continue the plan discussion and refine task assignments."
	default:
		return ""
	}
}

func messageHasWorkspaceContext(msg *protocol.Message) bool {
	if msg == nil || msg.Metadata == nil {
		return false
	}
	raw, ok := msg.Metadata["workspace_context"]
	if !ok {
		return false
	}
	ctx, ok := raw.(map[string]interface{})
	if !ok {
		return false
	}
	path, _ := ctx["workspace_path"].(string)
	return strings.TrimSpace(path) != ""
}

func inheritWorkspaceContextFromCollaboration(dst *protocol.Message, ctx map[string]interface{}) {
	if dst == nil || len(ctx) == 0 {
		return
	}
	if dst.Metadata == nil {
		dst.Metadata = map[string]interface{}{}
	}
	if messageHasWorkspaceContext(dst) {
		return
	}
	dst.Metadata["workspace_context"] = ctx
	if _, ok := dst.Metadata[MetadataContextScope]; !ok {
		dst.Metadata[MetadataContextScope] = ContextScopeOutline
	}
}
