package hub

import (
	"context"

	"github.com/camronwood/neural-junkie/internal/collabworktree"
	"github.com/camronwood/neural-junkie/internal/workspacebackend"
)

// SetCollabWorktreeBackendResolver wires remote git worktree + validation for collaborations.
func (h *Hub) SetCollabWorktreeBackendResolver(fn func(repoPath string) workspacebackend.Backend) {
	if h == nil {
		return
	}
	h.worktreeBackendFn = fn
	if h.collabManager != nil {
		h.collabManager.SetWorktreeBackendResolver(fn)
	}
}

func (h *Hub) worktreeBackendForPath(path string) workspacebackend.Backend {
	if h == nil || h.worktreeBackendFn == nil {
		return nil
	}
	return h.worktreeBackendFn(path)
}

func (h *Hub) validateGitRepoForCollaboration(path string) error {
	if b := h.worktreeBackendForPath(path); b != nil && b.Kind() != workspacebackend.KindLocal {
		return collabworktree.ValidateGitRepoViaBackend(context.Background(), b)
	}
	return collabworktree.ValidateGitRepo(path)
}
