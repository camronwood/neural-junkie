package agent

import (
	"strings"

	"github.com/camronwood/neural-junkie/internal/collaboration"
	"github.com/camronwood/neural-junkie/internal/protocol"
)

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
	b.WriteString("- Do NOT start with \"Grounding: I loaded N files\"; the user cares about outcomes, not how many files you opened.\n")

	tree := fileTreeFromWorkspaceContext(info.SourceWorkspaceContext)
	if tree != "" && !stackSignalsKubernetes(tree) && agentType == protocol.AgentTypeDevOps {
		b.WriteString("- **Platform note:** The shared file tree does not show Kubernetes/Helm assets — focus on this repo's actual stack (languages, CI, AWS, apps) instead of cluster tooling.\n")
	}
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
