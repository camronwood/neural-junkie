package agent

import (
	"strings"

	"github.com/camronwood/neural-junkie/internal/protocol"
)

// collaborationProactiveWorkspaceScan is false for most collaboration turns so agents
// answer the goal/tasks instead of re-loading a dozen files every reply.
func collaborationProactiveWorkspaceScan(msg *protocol.Message, info CollaborationInfo) bool {
	if msg == nil || info.ID == "" {
		return true
	}
	// No bound repo (--no-workspace / research-only planning): do not scan the open editor tree.
	if (info.Phase == "planning" || info.Phase == "reviewing") && len(info.SourceWorkspaceContext) == 0 {
		return false
	}
	switch msg.Type {
	case protocol.MessageTypeCollabTask:
		return true
	case protocol.MessageTypeCollabRecap:
		return false
	case protocol.MessageTypeCollabDiscussion:
		if info.Phase == "reviewing" || info.Phase == "approved" {
			return false
		}
		if info.Phase == "executing" {
			return collaborationMessageRequestsFileDive(msg.Content)
		}
		return collaborationMessageRequestsFileDive(msg.Content)
	default:
		if info.Phase == "executing" {
			return false
		}
	}
	if info.Phase == "planning" || info.Phase == "executing" {
		return collaborationMessageRequestsFileDive(msg.Content)
	}
	return true
}

// collaborationWorkspaceGroundingLine disables the forced "Grounding: I loaded N files" opener.
func collaborationWorkspaceGroundingLine(msg *protocol.Message, info CollaborationInfo) bool {
	if info.ID == "" {
		return true
	}
	if (info.Phase == "planning" || info.Phase == "reviewing") && len(info.SourceWorkspaceContext) == 0 {
		return false
	}
	if msg != nil && msg.Type == protocol.MessageTypeCollabTask {
		return false
	}
	return collaborationProactiveWorkspaceScan(msg, info)
}

// collaborationSkipExtraWorkspaceSection avoids duplicating file trees already in collab prompts.
func collaborationSkipExtraWorkspaceSection(info CollaborationInfo) bool {
	return info.ID != "" && len(info.SourceWorkspaceContext) > 0
}

func collaborationMessageRequestsFileDive(content string) bool {
	lower := strings.ToLower(strings.TrimSpace(content))
	if lower == "" {
		return false
	}
	needles := []string{
		"review ", "read ", "inspect ", "look at ", "open ",
		"walk through", "audit ", "analyze ", "analyse ",
		"in the repo", "in the codebase", "which file",
		".go", ".rs", ".py", ".ts", ".tsx", ".js",
		"src/", "internal/",
	}
	for _, n := range needles {
		if strings.Contains(lower, n) {
			return true
		}
	}
	return len(DetectFilePaths(content)) > 0
}
