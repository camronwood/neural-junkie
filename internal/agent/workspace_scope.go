package agent

import (
	"os"
	"path/filepath"
	"strings"

	"github.com/camronwood/neural-junkie/internal/protocol"
)

const (
	maxScopedWorkspaces = 4
	maxLinkedWorkspaces = 3
)

// WorkspaceRef is a repo root in a multi-workspace scope.
type WorkspaceRef struct {
	ID   string
	Path string
	Name string
}

// RepoConsultBlock is one consult result from the hub.
type RepoConsultBlock struct {
	Path      string
	AgentName string
	Text      string
}

// primaryWorkspaceFromMetadata returns the primary workspace from workspace_context.
func primaryWorkspaceFromMetadata(msg *protocol.Message) WorkspaceRef {
	if msg == nil || msg.Metadata == nil {
		return WorkspaceRef{}
	}
	raw, ok := msg.Metadata["workspace_context"]
	if !ok || raw == nil {
		return WorkspaceRef{}
	}
	m, ok := raw.(map[string]interface{})
	if !ok {
		return WorkspaceRef{}
	}
	return workspaceRefFromMap(m)
}

// linkedWorkspacesFromMetadata returns linked repos (excluding primary), capped.
func linkedWorkspacesFromMetadata(msg *protocol.Message) []WorkspaceRef {
	if msg == nil || msg.Metadata == nil {
		return nil
	}
	raw, ok := msg.Metadata[MetadataLinkedWorkspaces]
	if !ok || raw == nil {
		return nil
	}
	arr, ok := raw.([]interface{})
	if !ok {
		return nil
	}
	primary := primaryWorkspaceFromMetadata(msg)
	seen := map[string]bool{}
	if primary.Path != "" {
		seen[normalizeScopePath(primary.Path)] = true
	}
	var out []WorkspaceRef
	for _, item := range arr {
		if len(out) >= maxLinkedWorkspaces {
			break
		}
		m, ok := item.(map[string]interface{})
		if !ok {
			continue
		}
		ref := workspaceRefFromMap(m)
		if ref.Path == "" {
			continue
		}
		norm := normalizeScopePath(ref.Path)
		if seen[norm] {
			continue
		}
		seen[norm] = true
		out = append(out, ref)
	}
	return out
}

// scopedWorkspacesFromMetadata returns primary first, then linked (deduped, capped).
func scopedWorkspacesFromMetadata(msg *protocol.Message) []WorkspaceRef {
	primary := primaryWorkspaceFromMetadata(msg)
	linked := linkedWorkspacesFromMetadata(msg)
	if primary.Path == "" && len(linked) == 0 {
		return nil
	}
	out := make([]WorkspaceRef, 0, maxScopedWorkspaces)
	if primary.Path != "" {
		out = append(out, primary)
	}
	for _, ref := range linked {
		if len(out) >= maxScopedWorkspaces {
			break
		}
		out = append(out, ref)
	}
	return out
}

// resolveWorkspaceForRelativePath picks the repo root that contains relPath.
func resolveWorkspaceForRelativePath(msg *protocol.Message, relPath string) (WorkspaceRef, bool) {
	relPath = strings.TrimSpace(strings.TrimPrefix(relPath, "./"))
	if relPath == "" {
		return WorkspaceRef{}, false
	}
	scoped := scopedWorkspacesFromMetadata(msg)
	if len(scoped) == 0 {
		return WorkspaceRef{}, false
	}
	var matches []WorkspaceRef
	for _, ref := range scoped {
		abs := filepath.Join(ref.Path, relPath)
		if info, err := os.Stat(abs); err == nil && !info.IsDir() {
			matches = append(matches, ref)
		}
	}
	if len(matches) == 1 {
		return matches[0], true
	}
	if len(matches) > 1 {
		return WorkspaceRef{}, false
	}
	// Longest-prefix match for absolute or partial paths.
	relSlash := filepath.ToSlash(relPath)
	bestLen := -1
	var best WorkspaceRef
	for _, ref := range scoped {
		root := filepath.ToSlash(filepath.Clean(ref.Path))
		if strings.HasPrefix(relSlash, root+"/") || relSlash == root {
			if len(root) > bestLen {
				bestLen = len(root)
				best = ref
			}
		}
	}
	if bestLen >= 0 {
		return best, true
	}
	return scoped[0], true
}

func workspaceRefFromMap(m map[string]interface{}) WorkspaceRef {
	if m == nil {
		return WorkspaceRef{}
	}
	id, _ := m["workspace_id"].(string)
	path, _ := m["workspace_path"].(string)
	name, _ := m["workspace_name"].(string)
	return WorkspaceRef{ID: strings.TrimSpace(id), Path: strings.TrimSpace(path), Name: strings.TrimSpace(name)}
}

func normalizeScopePath(p string) string {
	p = strings.TrimSpace(p)
	if p == "" {
		return ""
	}
	if abs, err := filepath.Abs(p); err == nil {
		return filepath.Clean(abs)
	}
	return filepath.Clean(p)
}

// messageMentionsWorkspace reports whether content references a linked workspace by name or path segment.
func messageMentionsWorkspace(content string, ref WorkspaceRef) bool {
	content = strings.ToLower(content)
	if ref.Name != "" {
		seg := strings.ToLower(filepath.Base(ref.Name))
		if seg != "" && strings.Contains(content, seg) {
			return true
		}
	}
	if ref.Path != "" {
		seg := strings.ToLower(filepath.Base(filepath.Clean(ref.Path)))
		if seg != "" && strings.Contains(content, seg) {
			return true
		}
	}
	return false
}

// linkedWorkspaceHasOpenTab reports whether linked metadata includes open files for this workspace.
func linkedWorkspaceHasOpenTab(msg *protocol.Message, ref WorkspaceRef) bool {
	if msg == nil || msg.Metadata == nil || ref.Path == "" {
		return false
	}
	raw, ok := msg.Metadata[MetadataLinkedWorkspaces]
	if !ok {
		return false
	}
	arr, ok := raw.([]interface{})
	if !ok {
		return false
	}
	norm := normalizeScopePath(ref.Path)
	for _, item := range arr {
		m, ok := item.(map[string]interface{})
		if !ok {
			continue
		}
		p, _ := m["workspace_path"].(string)
		if normalizeScopePath(p) != norm {
			continue
		}
		if files, ok := m["open_files"].([]interface{}); ok && len(files) > 0 {
			return true
		}
	}
	return false
}

// selectReposForConsult picks repo paths to consult for a message (primary + up to 2 linked).
func selectReposForConsult(msg *protocol.Message, intent TurnIntent) []WorkspaceRef {
	scoped := scopedWorkspacesFromMetadata(msg)
	if len(scoped) == 0 {
		return nil
	}
	if intent == IntentClosure || intent == IntentLowSignal {
		return nil
	}
	content := ""
	if msg != nil {
		content = msg.Content
	}
	var out []WorkspaceRef
	primary := scoped[0]
	out = append(out, primary)
	linked := scoped[1:]
	if len(linked) == 0 {
		return out
	}
	var extras []WorkspaceRef
	for _, ref := range linked {
		if linkedWorkspaceHasOpenTab(msg, ref) || messageMentionsWorkspace(content, ref) {
			extras = append(extras, ref)
		}
	}
	if len(extras) == 0 && (intent == IntentTask || intent == IntentSubstantive) {
		extras = append(extras, linked[0])
	}
	for _, ref := range extras {
		if len(out) >= 3 {
			break
		}
		dup := false
		for _, existing := range out {
			if normalizeScopePath(existing.Path) == normalizeScopePath(ref.Path) {
				dup = true
				break
			}
		}
		if !dup {
			out = append(out, ref)
		}
	}
	return out
}
