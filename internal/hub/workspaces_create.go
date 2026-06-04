package hub

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"
)

// DefaultWorkspacesRoot is where new workspaces are created when no parent path is given.
func DefaultWorkspacesRoot() (string, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return "", fmt.Errorf("home directory: %w", err)
	}
	return filepath.Join(home, ".neural-junkie", "workspaces"), nil
}

func sanitizeWorkspaceDirName(name string) string {
	name = strings.TrimSpace(name)
	if name == "" {
		return "workspace"
	}
	var b strings.Builder
	for _, r := range name {
		switch {
		case r >= 'a' && r <= 'z', r >= 'A' && r <= 'Z', r >= '0' && r <= '9', r == '-', r == '_':
			b.WriteRune(r)
		case r == ' ' || r == '.':
			b.WriteRune('-')
		default:
			b.WriteRune('-')
		}
	}
	out := strings.Trim(b.String(), "-")
	if out == "" {
		return "workspace"
	}
	if len(out) > 64 {
		out = out[:64]
	}
	return out
}

// AddWorkspaceOptions controls create-vs-link behavior for POST /api/workspaces.
type AddWorkspaceOptions struct {
	Create     bool
	ParentPath string // when Create: optional parent directory; empty uses DefaultWorkspacesRoot
}

// AddWorkspace registers a workspace. When opts.Create is true, the directory is created if missing.
func (wm *WorkspaceManager) AddWorkspace(name, path string, opts AddWorkspaceOptions) (*Workspace, error) {
	name = strings.TrimSpace(name)
	if name == "" {
		return nil, fmt.Errorf("name required")
	}

	var absPath string
	var err error

	if opts.Create {
		absPath, err = resolveCreateWorkspacePath(name, path, opts.ParentPath)
		if err != nil {
			return nil, err
		}
		if err := os.MkdirAll(absPath, 0o755); err != nil {
			return nil, fmt.Errorf("create workspace directory: %w", err)
		}
	} else {
		path = strings.TrimSpace(path)
		if path == "" {
			return nil, fmt.Errorf("path required")
		}
		if _, err := os.Stat(path); err != nil {
			return nil, fmt.Errorf("path does not exist: %w", err)
		}
		absPath, err = filepath.Abs(path)
		if err != nil {
			return nil, fmt.Errorf("resolve absolute path: %w", err)
		}
	}

	wm.mutex.Lock()
	defer wm.mutex.Unlock()

	for _, workspace := range wm.workspaces {
		if workspace.Path == absPath {
			return workspace, nil
		}
	}

	workspace := &Workspace{
		ID:        fmt.Sprintf("workspace_%d", time.Now().UnixNano()),
		Name:      name,
		Path:      absPath,
		CreatedAt: time.Now(),
		LastUsed:  time.Now(),
	}

	if _, err := os.Stat(filepath.Join(absPath, ".git")); err == nil {
		workspace.IsGitRepo = true
	}

	wm.workspaces[workspace.ID] = workspace
	if err := wm.saveWorkspacesLocked(); err != nil {
		delete(wm.workspaces, workspace.ID)
		return nil, fmt.Errorf("failed to save workspace: %w", err)
	}
	return workspace, nil
}

func resolveCreateWorkspacePath(name, path, parentPath string) (string, error) {
	path = strings.TrimSpace(path)
	parentPath = strings.TrimSpace(parentPath)

	if path != "" {
		return filepath.Abs(path)
	}

	dirName := sanitizeWorkspaceDirName(name)
	var base string
	switch {
	case parentPath != "":
		info, err := os.Stat(parentPath)
		if err != nil {
			return "", fmt.Errorf("parent path: %w", err)
		}
		if !info.IsDir() {
			return "", fmt.Errorf("parent path is not a directory")
		}
		base = parentPath
	default:
		root, err := DefaultWorkspacesRoot()
		if err != nil {
			return "", err
		}
		base = root
	}

	if err := os.MkdirAll(base, 0o755); err != nil {
		return "", fmt.Errorf("ensure workspace parent: %w", err)
	}

	target := filepath.Join(base, dirName)
	if _, err := os.Stat(target); err == nil {
		return target, nil
	}
	return filepath.Abs(target)
}
