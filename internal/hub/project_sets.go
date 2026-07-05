package hub

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/google/uuid"
)

// ProjectSet groups related workspaces for cross-repo agent scope.
type ProjectSet struct {
	ID                 string    `json:"id"`
	Name               string    `json:"name"`
	PrimaryWorkspaceID string    `json:"primary_workspace_id"`
	MemberWorkspaceIDs []string  `json:"member_workspace_ids"`
	CreatedAt          time.Time `json:"created_at"`
}

// ProjectSetManager stores project sets on disk.
type ProjectSetManager struct {
	sets        map[string]*ProjectSet
	storagePath string
	mutex       sync.RWMutex
}

// NewProjectSetManager creates a project set manager.
func NewProjectSetManager() (*ProjectSetManager, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return nil, fmt.Errorf("home dir: %w", err)
	}
	pm := &ProjectSetManager{
		sets:        make(map[string]*ProjectSet),
		storagePath: filepath.Join(home, ".neural-junkie", "project-sets.json"),
	}
	if err := pm.load(); err != nil && !os.IsNotExist(err) {
		return nil, err
	}
	return pm, nil
}

func (pm *ProjectSetManager) load() error {
	pm.mutex.Lock()
	defer pm.mutex.Unlock()
	data, err := os.ReadFile(pm.storagePath)
	if err != nil {
		return err
	}
	var list []*ProjectSet
	if err := json.Unmarshal(data, &list); err != nil {
		return err
	}
	pm.sets = make(map[string]*ProjectSet, len(list))
	for _, ps := range list {
		if ps != nil && ps.ID != "" {
			pm.sets[ps.ID] = ps
		}
	}
	return nil
}

func (pm *ProjectSetManager) save() error {
	pm.mutex.RLock()
	list := make([]*ProjectSet, 0, len(pm.sets))
	for _, ps := range pm.sets {
		list = append(list, ps)
	}
	pm.mutex.RUnlock()
	data, err := json.MarshalIndent(list, "", "  ")
	if err != nil {
		return err
	}
	if err := os.MkdirAll(filepath.Dir(pm.storagePath), 0o755); err != nil {
		return err
	}
	return os.WriteFile(pm.storagePath, data, 0o644)
}

// ListProjectSets returns all project sets.
func (pm *ProjectSetManager) ListProjectSets() []*ProjectSet {
	pm.mutex.RLock()
	defer pm.mutex.RUnlock()
	out := make([]*ProjectSet, 0, len(pm.sets))
	for _, ps := range pm.sets {
		out = append(out, ps)
	}
	return out
}

// GetProjectSet returns a project set by ID.
func (pm *ProjectSetManager) GetProjectSet(id string) (*ProjectSet, bool) {
	pm.mutex.RLock()
	defer pm.mutex.RUnlock()
	ps, ok := pm.sets[id]
	return ps, ok
}

// CreateProjectSet creates a new project set after validating workspace IDs.
func (pm *ProjectSetManager) CreateProjectSet(name, primaryID string, memberIDs []string, wm *WorkspaceManager) (*ProjectSet, error) {
	name = strings.TrimSpace(name)
	if name == "" {
		return nil, fmt.Errorf("name required")
	}
	if err := validateProjectSetWorkspaces(primaryID, memberIDs, wm); err != nil {
		return nil, err
	}
	ps := &ProjectSet{
		ID:                 uuid.New().String(),
		Name:               name,
		PrimaryWorkspaceID: primaryID,
		MemberWorkspaceIDs: dedupeStrings(append([]string{}, memberIDs...)),
		CreatedAt:          time.Now().UTC(),
	}
	pm.mutex.Lock()
	pm.sets[ps.ID] = ps
	pm.mutex.Unlock()
	if err := pm.save(); err != nil {
		return nil, err
	}
	return ps, nil
}

// UpdateProjectSet updates an existing project set.
func (pm *ProjectSetManager) UpdateProjectSet(id, name, primaryID string, memberIDs []string, wm *WorkspaceManager) (*ProjectSet, error) {
	pm.mutex.Lock()
	ps, ok := pm.sets[id]
	if !ok {
		pm.mutex.Unlock()
		return nil, fmt.Errorf("project set not found")
	}
	if name = strings.TrimSpace(name); name != "" {
		ps.Name = name
	}
	if primaryID != "" {
		if err := validateProjectSetWorkspaces(primaryID, memberIDs, wm); err != nil {
			pm.mutex.Unlock()
			return nil, err
		}
		ps.PrimaryWorkspaceID = primaryID
	}
	if memberIDs != nil {
		if err := validateProjectSetWorkspaces(ps.PrimaryWorkspaceID, memberIDs, wm); err != nil {
			pm.mutex.Unlock()
			return nil, err
		}
		ps.MemberWorkspaceIDs = dedupeStrings(memberIDs)
	}
	pm.mutex.Unlock()
	if err := pm.save(); err != nil {
		return nil, err
	}
	return ps, nil
}

// DeleteProjectSet removes a project set.
func (pm *ProjectSetManager) DeleteProjectSet(id string) error {
	pm.mutex.Lock()
	delete(pm.sets, id)
	pm.mutex.Unlock()
	return pm.save()
}

func validateProjectSetWorkspaces(primaryID string, memberIDs []string, wm *WorkspaceManager) error {
	if wm == nil {
		return fmt.Errorf("workspace manager unavailable")
	}
	if primaryID == "" {
		return fmt.Errorf("primary_workspace_id required")
	}
	if _, ok := wm.GetWorkspace(primaryID); !ok {
		return fmt.Errorf("primary workspace not found: %s", primaryID)
	}
	for _, id := range memberIDs {
		if id == "" || id == primaryID {
			continue
		}
		if _, ok := wm.GetWorkspace(id); !ok {
			return fmt.Errorf("member workspace not found: %s", id)
		}
	}
	return nil
}

func dedupeStrings(in []string) []string {
	seen := map[string]bool{}
	var out []string
	for _, s := range in {
		s = strings.TrimSpace(s)
		if s == "" || seen[s] {
			continue
		}
		seen[s] = true
		out = append(out, s)
	}
	return out
}
