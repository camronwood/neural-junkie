package agent

import (
	"fmt"
	"path/filepath"
	"strings"

	"github.com/camronwood/neural-junkie/internal/collaboration"
	"github.com/camronwood/neural-junkie/internal/protocol"
)

// collaborationProactiveWorkspaceScan is false for most collaboration turns so agents
// answer the goal/tasks instead of re-loading a dozen files every reply.
func collaborationProactiveWorkspaceScan(msg *protocol.Message, info CollaborationInfo) bool {
	if msg == nil || info.ID == "" {
		return true
	}
	if msg.IsFromSystem() {
		return false
	}
	if msg.Metadata != nil {
		if internal, ok := msg.Metadata["collab_internal_event"].(bool); ok && internal {
			return false
		}
	}
	// No bound repo (--no-workspace / research-only planning): do not scan the open editor tree.
	if (info.Phase == "planning" || info.Phase == "reviewing") && len(info.SourceWorkspaceContext) == 0 {
		return false
	}
	switch msg.Type {
	case protocol.MessageTypeCollabTask:
		// Hub merges task_context_paths into open_files with context_scope=focus.
		// Bulk repo scan would load unrelated stack files (e.g. src/App.tsx) for markdown deliverables.
		if msg.Metadata != nil {
			if scope, ok := msg.Metadata[MetadataContextScope].(string); ok && scope == ContextScopeFocus {
				return false
			}
		}
		return true
	case protocol.MessageTypeCollabRecap:
		return false
	case protocol.MessageTypeCollabDiscussion:
		if info.Phase == "reviewing" || info.Phase == "approved" {
			return false
		}
		if info.Phase == "planning" {
			// Curated file_tree is enough; path tokens in task lines must not bulk-scan minimal-repo fixtures.
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

// collaborationFocusScopedTask is true when the hub bound this turn to explicit task_context_paths.
func collaborationFocusScopedTask(msg *protocol.Message) bool {
	if msg == nil || strings.TrimSpace(msg.GetCollaborationID()) == "" || msg.Metadata == nil {
		return false
	}
	scope, _ := msg.Metadata[MetadataContextScope].(string)
	if scope != ContextScopeFocus {
		return false
	}
	if msg.Type == protocol.MessageTypeCollabTask {
		return true
	}
	return strings.TrimSpace(msg.GetTaskID()) != "" &&
		strings.TrimSpace(msg.GetCollaborationPhase()) == "executing"
}

// collaborationDiscoveryToolNames are workspace inventory tools that bypass curated focus files.
var collaborationDiscoveryToolNames = map[string]bool{
	"list_dir":         true,
	"glob_file_search": true,
	"semantic_search":  true,
	"search_codebase":  true,
	"search_by_path":   true,
	"list_key_files":   true,
	"grep":             true,
}

// collaborationRestrictsDiscoveryTools hides repo-walk tools when context_scope=focus so
// assignees use the hub-merged open_files / task_context_paths instead of inventorying the fixture.
func collaborationRestrictsDiscoveryTools(msg *protocol.Message) bool {
	return collaborationFocusScopedTask(msg)
}

// effectiveMCPToolAllowlist returns the per-message MCP allowlist (agent allowlist, narrowed for focus scope).
func effectiveMCPToolAllowlist(a *Agent, msg *protocol.Message) []string {
	var base []string
	if a != nil {
		base = a.MCPToolAllowlist
	}
	base = withSharedWebSearchAllowlist(base)
	if outlinePlanningReadOnlyTools(msg) {
		return narrowMCPToolAllowlist(base, outlinePlanningAllowedTools)
	}
	if !collaborationRestrictsDiscoveryTools(msg) {
		return base
	}
	focused := []string{"read_file", "get_file_content"}
	return narrowMCPToolAllowlist(base, focused)
}

// outlinePlanningReadOnlyTools is true for outline/hint turns that are not yet an
// implementation session. Those turns should answer from workspace context without
// blocking on run_command tool approvals (e.g. npm install).
func outlinePlanningReadOnlyTools(msg *protocol.Message) bool {
	if msg == nil || msg.ImplementationSession() {
		return false
	}
	scope := ContextScopeFromMessage(msg)
	return scope == ContextScopeOutline || scope == ContextScopeHint
}

// outlinePlanningAllowedTools are read/discovery tools safe for outline planning.
var outlinePlanningAllowedTools = []string{
	"read_file", "get_file_content", "list_dir", "glob", "glob_file_search",
	"grep", "semantic_search", "web_search", "fetch_url",
}

func narrowMCPToolAllowlist(base, keep []string) []string {
	if len(base) == 0 {
		return append([]string{}, keep...)
	}
	allowed := make(map[string]bool, len(base))
	for _, name := range base {
		allowed[name] = true
	}
	var out []string
	for _, name := range keep {
		if allowed[name] {
			out = append(out, name)
		}
	}
	if len(out) == 0 {
		// Keep prior allowlist but still drop discovery tools at execution time.
		return base
	}
	return out
}

// withSharedWebSearchAllowlist keeps web_search/fetch_url available even when an
// agent uses a narrow MCP allowlist (maps, biology profiles, etc.).
func withSharedWebSearchAllowlist(base []string) []string {
	if len(base) == 0 {
		return base
	}
	out := append([]string{}, base...)
	for _, name := range []string{"web_search", "fetch_url"} {
		found := false
		for _, existing := range out {
			if existing == name {
				found = true
				break
			}
		}
		if !found {
			out = append(out, name)
		}
	}
	return out
}

func collaborationDiscoveryToolBlocked(msg *protocol.Message, toolName string) bool {
	return collaborationRestrictsDiscoveryTools(msg) && collaborationDiscoveryToolNames[toolName]
}

// collaborationFocusAllowedReadPaths lists repo-relative paths hub attached for a focus-scoped task.
func collaborationFocusAllowedReadPaths(msg *protocol.Message) []string {
	if msg == nil {
		return nil
	}
	seen := map[string]bool{}
	var out []string
	add := func(p string) {
		p = normalizeFocusRelPath("", p)
		if p == "" || seen[p] {
			return
		}
		seen[p] = true
		out = append(out, p)
	}
	for _, p := range taskContextPathsFromMessage(msg) {
		add(p)
	}
	if msg.Metadata == nil {
		return out
	}
	ws, _ := msg.Metadata["workspace_context"].(map[string]interface{})
	if ws == nil {
		return out
	}
	if files, ok := ws["open_files"].([]interface{}); ok {
		for _, f := range files {
			fm, ok := f.(map[string]interface{})
			if !ok {
				continue
			}
			if p, ok := fm["path"].(string); ok {
				add(p)
			}
		}
	}
	return out
}

func normalizeFocusRelPath(wsRoot, raw string) string {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return ""
	}
	raw = filepath.ToSlash(raw)
	raw = strings.TrimPrefix(raw, "./")
	wsRoot = strings.TrimSpace(wsRoot)
	if wsRoot != "" {
		absRoot, err := filepath.Abs(wsRoot)
		if err == nil {
			absRoot = filepath.ToSlash(filepath.Clean(absRoot))
			cand := raw
			if !filepath.IsAbs(raw) {
				cand = filepath.ToSlash(filepath.Join(absRoot, raw))
			} else {
				cand = filepath.ToSlash(filepath.Clean(raw))
			}
			if rel, err := filepath.Rel(absRoot, cand); err == nil {
				rel = filepath.ToSlash(rel)
				if rel != ".." && !strings.HasPrefix(rel, "../") {
					return rel
				}
			}
		}
	}
	if filepath.IsAbs(raw) {
		return filepath.ToSlash(filepath.Base(raw))
	}
	return raw
}

func collaborationFocusPathMatches(allowed, requested string) bool {
	allowed = strings.TrimSuffix(allowed, "/")
	requested = strings.TrimSuffix(requested, "/")
	if allowed == "" || requested == "" {
		return false
	}
	if allowed == requested {
		return true
	}
	// Dir allowlist entries cover children.
	if strings.HasPrefix(requested, allowed+"/") {
		return true
	}
	// Basename allow when task lists "README.md" and tool requests absolute/prefixed form already normalized.
	if filepath.Base(allowed) == filepath.Base(requested) && !strings.Contains(allowed, "/") {
		return filepath.Base(requested) == allowed
	}
	return false
}

// collaborationFocusReadPathAllowed is false when focus scope has an allowlist and path is outside it.
// Deliverable paths under collabs/<id>/ are always allowed so assignees can inspect stubs they must write.
func collaborationFocusReadPathAllowed(msg *protocol.Message, wsRoot, requested string) bool {
	if !collaborationRestrictsDiscoveryTools(msg) {
		return true
	}
	allowed := collaborationFocusAllowedReadPaths(msg)
	if len(allowed) == 0 {
		return true
	}
	rel := normalizeFocusRelPath(wsRoot, requested)
	if rel == "" {
		return false
	}
	if strings.HasPrefix(rel, collaboration.ProjectCollabsDirName+"/") {
		return true
	}
	for _, a := range allowed {
		if collaborationFocusPathMatches(a, rel) {
			return true
		}
	}
	return false
}

func collaborationFocusReadPathBlockMessage(msg *protocol.Message, wsRoot, requested string) string {
	allowed := collaborationFocusAllowedReadPaths(msg)
	rel := normalizeFocusRelPath(wsRoot, requested)
	if rel == "" {
		rel = strings.TrimSpace(requested)
	}
	return fmt.Sprintf(
		"ERROR: read path %q is outside this focus-scoped collaboration task. Allowed source paths: %s. Do not read sibling packages; ship the deliverable from the allowed sources only.",
		rel,
		strings.Join(allowed, ", "),
	)
}

// collaborationWorkspaceGroundingLine disables the forced "Grounding: I loaded N files" opener.
func collaborationWorkspaceGroundingLine(msg *protocol.Message, info CollaborationInfo) bool {
	if info.ID == "" {
		return true
	}
	// Collab planning/review should not force the generic grounding opener — prompts already forbid it.
	if info.Phase == "planning" || info.Phase == "reviewing" {
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
