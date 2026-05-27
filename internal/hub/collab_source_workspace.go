package hub

import (
	"fmt"
	"path/filepath"
	"strings"

	"github.com/camronwood/neural-junkie/internal/collaboration"
	"github.com/camronwood/neural-junkie/internal/protocol"
	"github.com/google/uuid"
)

// resolveCollaborateSourceRepoPath returns a source repository path from outbound message
// metadata, or empty when the path is a Neural Junkie collaboration assets directory
// (sandbox, worktree, or reviews) rather than a real project checkout.
func (h *Hub) resolveCollaborateSourceRepoPath(msg *protocol.Message) (path string, warning string) {
	raw := workspacePathFromMessageMetadata(msg)
	if raw == "" {
		return "", ""
	}
	abs, err := filepath.Abs(raw)
	if err != nil {
		return "", ""
	}
	abs = filepath.Clean(abs)

	if h.pathUnderCollabAssets(abs) {
		return "", fmt.Sprintf(
			"⚠️ **Ignored active workspace** `%s` — it is a Neural Junkie collaboration folder, not a source repository.\n\nSelect your project in the desktop file explorer, or start with `--workspace` pointed at a git repo.",
			abs,
		)
	}

	if pathUnderProjectCollabDeliverable(abs) {
		return "", fmt.Sprintf(
			"⚠️ **Ignored active workspace** `%s` — it is a collaboration deliverables folder (`%s/<id>/`), not the project repository root.\n\nOpen the git repo root in the desktop file explorer (e.g. Phoenix), not a `collabs/…` subfolder.",
			abs, collaboration.ProjectCollabsDirName,
		)
	}

	return abs, ""
}

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
