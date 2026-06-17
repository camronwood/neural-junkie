package hub

import "github.com/camronwood/neural-junkie/internal/filechange"

// SetFileChangeBackendFn registers a resolver that returns WorkspaceIO for a workspace root.
// When nil is returned, file changes use local os.* under SetWorkspaceRoot.
func (h *Hub) SetFileChangeBackendFn(fn func(workspaceRoot string) filechange.WorkspaceIO) {
	if h == nil {
		return
	}
	h.fileChangeBackendFn = fn
}

func (h *Hub) bindFileChangeBackend(wsRoot string) {
	exec := h.fileChangeManager.GetExecutor()
	exec.SetWorkspaceIO(nil)
	if h.fileChangeBackendFn != nil {
		if io := h.fileChangeBackendFn(wsRoot); io != nil {
			exec.SetWorkspaceIO(io)
		}
	}
}
