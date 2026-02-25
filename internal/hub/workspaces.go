package hub

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"
)

// Workspace represents a configured workspace/repository
type Workspace struct {
	ID        string    `json:"id"`
	Name      string    `json:"name"`
	Path      string    `json:"path"`
	CreatedAt time.Time `json:"created_at"`
	LastUsed  time.Time `json:"last_used"`
	IsGitRepo bool      `json:"is_git_repo"`
	GitRemote string    `json:"git_remote,omitempty"`
	GitBranch string    `json:"git_branch,omitempty"`
}

// WorkspaceManager manages workspace storage and operations
type WorkspaceManager struct {
	workspaces  map[string]*Workspace
	storagePath string
	mutex       sync.RWMutex
}

// NewWorkspaceManager creates a new workspace manager
func NewWorkspaceManager() (*WorkspaceManager, error) {
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return nil, fmt.Errorf("failed to get home directory: %w", err)
	}

	storagePath := filepath.Join(homeDir, ".neural-junkie", "workspaces.json")

	wm := &WorkspaceManager{
		workspaces:  make(map[string]*Workspace),
		storagePath: storagePath,
	}

	// Load existing workspaces
	if err := wm.loadWorkspaces(); err != nil {
		// If file doesn't exist, that's okay - start with empty workspaces
		if !os.IsNotExist(err) {
			return nil, fmt.Errorf("failed to load workspaces: %w", err)
		}
	}

	return wm, nil
}

// loadWorkspaces loads workspaces from storage
func (wm *WorkspaceManager) loadWorkspaces() error {
	wm.mutex.Lock()
	defer wm.mutex.Unlock()

	data, err := os.ReadFile(wm.storagePath)
	if err != nil {
		return err
	}

	var workspaces []*Workspace
	if err := json.Unmarshal(data, &workspaces); err != nil {
		return fmt.Errorf("failed to unmarshal workspaces: %w", err)
	}

	// Convert slice to map
	wm.workspaces = make(map[string]*Workspace)
	pruned := 0
	for _, workspace := range workspaces {
		if shouldPruneWorkspace(workspace) {
			pruned++
			continue
		}
		wm.workspaces[workspace.ID] = workspace
	}
	if pruned > 0 {
		if err := wm.saveWorkspacesLocked(); err != nil {
			return fmt.Errorf("failed to persist pruned workspaces: %w", err)
		}
	}

	return nil
}

// PruneUnavailableTestWorkspaces removes leaked test temp workspaces that no
// longer exist on disk and persists the cleanup if any were removed.
func (wm *WorkspaceManager) PruneUnavailableTestWorkspaces() (int, error) {
	wm.mutex.Lock()
	defer wm.mutex.Unlock()

	removed := 0
	for id, workspace := range wm.workspaces {
		if shouldPruneWorkspace(workspace) {
			delete(wm.workspaces, id)
			removed++
		}
	}
	if removed == 0 {
		return 0, nil
	}
	if err := wm.saveWorkspacesLocked(); err != nil {
		return removed, fmt.Errorf("failed to save pruned workspaces: %w", err)
	}
	return removed, nil
}

func shouldPruneWorkspace(workspace *Workspace) bool {
	if workspace == nil || strings.TrimSpace(workspace.Path) == "" {
		return true
	}
	if !isEphemeralTempTestWorkspace(workspace.Path, workspace.Name) {
		return false
	}
	info, err := os.Stat(workspace.Path)
	return err != nil || !info.IsDir()
}

func isEphemeralTempTestWorkspace(path, name string) bool {
	lowerPath := strings.ToLower(path)
	lowerName := strings.ToLower(name)
	if strings.Contains(lowerPath, "testcommandintegration") ||
		strings.Contains(lowerPath, "testcommandparsing") ||
		strings.Contains(lowerName, "testcommandintegration") ||
		strings.Contains(lowerName, "testcommandparsing") {
		return true
	}
	// Go test temp dirs on macOS commonly look like /var/folders/.../T/TestFoo...
	return strings.Contains(path, "/var/folders/") && strings.Contains(path, "/T/Test")
}

// saveWorkspacesLocked persists the in-memory workspaces to disk.
// Caller MUST already hold wm.mutex (read or write).
func (wm *WorkspaceManager) saveWorkspacesLocked() error {
	workspaces := make([]*Workspace, 0, len(wm.workspaces))
	for _, workspace := range wm.workspaces {
		workspaces = append(workspaces, workspace)
	}

	data, err := json.MarshalIndent(workspaces, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal workspaces: %w", err)
	}

	if err := os.MkdirAll(filepath.Dir(wm.storagePath), 0755); err != nil {
		return fmt.Errorf("failed to create directory: %w", err)
	}

	if err := os.WriteFile(wm.storagePath, data, 0644); err != nil {
		return fmt.Errorf("failed to write workspaces file: %w", err)
	}

	return nil
}

// AddWorkspace adds a new workspace
func (wm *WorkspaceManager) AddWorkspace(name, path string) (*Workspace, error) {
	// Validate path exists
	if _, err := os.Stat(path); err != nil {
		return nil, fmt.Errorf("path does not exist: %w", err)
	}

	// Resolve absolute path
	absPath, err := filepath.Abs(path)
	if err != nil {
		return nil, fmt.Errorf("failed to resolve absolute path: %w", err)
	}

	// Check if workspace already exists
	for _, workspace := range wm.workspaces {
		if workspace.Path == absPath {
			return workspace, nil // Return existing workspace
		}
	}

	workspace := &Workspace{
		ID:        fmt.Sprintf("workspace_%d", time.Now().Unix()),
		Name:      name,
		Path:      absPath,
		CreatedAt: time.Now(),
		LastUsed:  time.Now(),
	}

	// Check if it's a git repository
	if _, err := os.Stat(filepath.Join(absPath, ".git")); err == nil {
		workspace.IsGitRepo = true
	}

	wm.mutex.Lock()
	wm.workspaces[workspace.ID] = workspace
	err = wm.saveWorkspacesLocked()
	if err != nil {
		delete(wm.workspaces, workspace.ID)
	}
	wm.mutex.Unlock()

	if err != nil {
		return nil, fmt.Errorf("failed to save workspace: %w", err)
	}

	return workspace, nil
}

// GetWorkspace gets a workspace by ID
func (wm *WorkspaceManager) GetWorkspace(id string) (*Workspace, bool) {
	wm.mutex.RLock()
	defer wm.mutex.RUnlock()

	workspace, exists := wm.workspaces[id]
	return workspace, exists
}

// ListWorkspaces returns all workspaces
func (wm *WorkspaceManager) ListWorkspaces() []*Workspace {
	wm.mutex.RLock()
	defer wm.mutex.RUnlock()

	workspaces := make([]*Workspace, 0, len(wm.workspaces))
	for _, workspace := range wm.workspaces {
		workspaces = append(workspaces, workspace)
	}

	return workspaces
}

// RemoveWorkspace removes a workspace
func (wm *WorkspaceManager) RemoveWorkspace(id string) error {
	wm.mutex.Lock()
	defer wm.mutex.Unlock()

	if _, exists := wm.workspaces[id]; !exists {
		return fmt.Errorf("workspace not found")
	}

	delete(wm.workspaces, id)

	if err := wm.saveWorkspacesLocked(); err != nil {
		return fmt.Errorf("failed to save workspaces after removal: %w", err)
	}

	return nil
}

// UpdateWorkspaceLastUsed updates the last used timestamp
func (wm *WorkspaceManager) UpdateWorkspaceLastUsed(id string) error {
	wm.mutex.Lock()
	defer wm.mutex.Unlock()

	workspace, exists := wm.workspaces[id]
	if !exists {
		return fmt.Errorf("workspace not found")
	}

	workspace.LastUsed = time.Now()

	if err := wm.saveWorkspacesLocked(); err != nil {
		return fmt.Errorf("failed to save workspaces after update: %w", err)
	}

	return nil
}

// FindWorkspaceByPath finds a workspace by its path
func (wm *WorkspaceManager) FindWorkspaceByPath(path string) (*Workspace, bool) {
	wm.mutex.RLock()
	defer wm.mutex.RUnlock()

	absPath, err := filepath.Abs(path)
	if err != nil {
		return nil, false
	}

	for _, workspace := range wm.workspaces {
		if workspace.Path == absPath {
			return workspace, true
		}
	}

	return nil, false
}
