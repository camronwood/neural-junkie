package hub

import (
	"fmt"
	"path/filepath"
	"strings"

	"github.com/camronwood/neural-junkie/internal/collaboration"
	"github.com/camronwood/neural-junkie/internal/protocol"
	"github.com/google/uuid"
)

// pathUnderProjectCollabDeliverable reports paths inside …/collabs/<uuid>/ (project deliverable sandboxes).
func pathUnderProjectCollabDeliverable(absPath string) bool {
	clean := filepath.Clean(absPath)
	parts := strings.Split(clean, string(filepath.Separator))
	dirName := collaboration.ProjectCollabsDirName
	for i := 0; i < len(parts)-1; i++ {
		if parts[i] != dirName {
			continue
		}
		if _, err := uuid.Parse(parts[i+1]); err == nil {
			return true
		}
	}
	return false
}

func (h *Hub) pathUnderCollabAssets(absPath string) bool {
	base, err := h.collabAssetsBaseDir()
	if err != nil || strings.TrimSpace(base) == "" {
		return false
	}
	base = filepath.Clean(base)
	absPath = filepath.Clean(absPath)
	rel, err := filepath.Rel(base, absPath)
	if err != nil {
		return false
	}
	if rel == ".." || strings.HasPrefix(rel, ".."+string(filepath.Separator)) {
		return false
	}
	return true
}

func (h *Hub) collabAssetsBaseDir() (string, error) {
	if h == nil || h.collabManager == nil {
		return "", fmt.Errorf("collaboration manager unavailable")
	}
	return h.collabManager.CollabAssetsBaseDir()
}

const (
	metadataCollabSourceMode = "collab_source_mode"
	metadataCollabSourcePath = "collab_source_path"
)

// resolveCollaborateSourceRepoPath picks the project/research root for a collaboration.
// Priority: CLI flags (--no-workspace, --repo) → outbound metadata → active workspace snapshot.
func (h *Hub) resolveCollaborateSourceRepoPath(msg *protocol.Message, flags collaborateFlagParse) (path string, warning string) {
	if flags.NoWorkspace {
		return "", ""
	}
	if p := strings.TrimSpace(flags.RepoPath); p != "" {
		return h.validateCollaborateSourcePath(p)
	}
	if msg != nil && msg.Metadata != nil {
		if mode, _ := msg.Metadata[metadataCollabSourceMode].(string); strings.EqualFold(strings.TrimSpace(mode), "none") {
			return "", ""
		}
		if mode, _ := msg.Metadata[metadataCollabSourceMode].(string); strings.EqualFold(strings.TrimSpace(mode), "path") {
			if p, _ := msg.Metadata[metadataCollabSourcePath].(string); strings.TrimSpace(p) != "" {
				return h.validateCollaborateSourcePath(p)
			}
		}
	}
	return h.resolveCollaborateSourceRepoPathFromWorkspaceMeta(msg)
}

func (h *Hub) resolveCollaborateSourceRepoPathFromWorkspaceMeta(msg *protocol.Message) (path string, warning string) {
	raw := workspacePathFromMessageMetadata(msg)
	if raw == "" {
		return "", ""
	}
	return h.validateCollaborateSourcePath(raw)
}

func (h *Hub) validateCollaborateSourcePath(raw string) (path string, warning string) {
	abs, err := filepath.Abs(strings.TrimSpace(raw))
	if err != nil {
		return "", ""
	}
	abs = filepath.Clean(abs)

	if h.pathUnderCollabAssets(abs) {
		return "", fmt.Sprintf(
			"⚠️ **Ignored workspace** `%s` — it is a Neural Junkie collaboration folder, not a source repository.\n\nPick a project checkout, use **Choose folder**, or start with `--no-workspace` for research-only collabs.",
			abs,
		)
	}

	if pathUnderProjectCollabDeliverable(abs) {
		return "", fmt.Sprintf(
			"⚠️ **Ignored workspace** `%s` — it is a collaboration deliverables folder (`%s/<id>/`), not the project repository root.\n\nOpen the git repo root in the file explorer or pass `--repo /path/to/repo`.",
			abs, collaboration.ProjectCollabsDirName,
		)
	}

	return abs, ""
}

// workspaceContextForCollaboration builds SourceWorkspaceContext for createOpts.
// When goal is non-empty and a repo path is bound, file_tree is populated with goal-relevant paths first.
func workspaceContextForCollaboration(msg *protocol.Message, flags collaborateFlagParse, sourceRepoPath, goal string) map[string]interface{} {
	if flags.NoWorkspace {
		return nil
	}
	if p := strings.TrimSpace(flags.RepoPath); p != "" {
		abs, _ := filepath.Abs(p)
		ctx := map[string]interface{}{
			"workspace_path": filepath.Clean(abs),
			"workspace_name": filepath.Base(abs),
		}
		enrichSourceWorkspaceOutline(ctx, abs, goal)
		return ctx
	}
	if msg != nil && msg.Metadata != nil {
		if mode, _ := msg.Metadata[metadataCollabSourceMode].(string); strings.EqualFold(strings.TrimSpace(mode), "none") {
			return nil
		}
		if mode, _ := msg.Metadata[metadataCollabSourceMode].(string); strings.EqualFold(strings.TrimSpace(mode), "path") {
			if p, _ := msg.Metadata[metadataCollabSourcePath].(string); strings.TrimSpace(p) != "" {
				abs, _ := filepath.Abs(p)
				ctx := map[string]interface{}{
					"workspace_path": filepath.Clean(abs),
					"workspace_name": filepath.Base(abs),
				}
				enrichSourceWorkspaceOutline(ctx, abs, goal)
				return ctx
			}
		}
	}
	if sourceRepoPath != "" {
		ctx := map[string]interface{}{
			"workspace_path": sourceRepoPath,
			"workspace_name": filepath.Base(sourceRepoPath),
		}
		enrichSourceWorkspaceOutline(ctx, sourceRepoPath, goal)
		return ctx
	}
	ctx := workspaceContextFromMessageMetadata(msg)
	if ctx != nil {
		// Harness and desktop sends use outline scope during /collaborate — drop stale open-file bodies.
		delete(ctx, "open_files")
		if p, _ := ctx["workspace_path"].(string); strings.TrimSpace(p) != "" {
			enrichSourceWorkspaceOutline(ctx, p, goal)
		}
		if msg != nil && msg.Metadata != nil {
			if linked, ok := msg.Metadata["linked_workspaces"]; ok {
				ctx["linked_workspaces"] = linked
			}
		}
	}
	return ctx
}
