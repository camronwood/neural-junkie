package collaboration

import (
	"fmt"
	"log"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/camronwood/neural-junkie/internal/collabworktree"
)

// EnsureWorktree creates a git worktree when execution mode is worktree and
// WorkingDirectory is not yet set. sourceRepoPath is used when c.SourceRepoPath is empty.
func (cm *CollaborationManager) EnsureWorktree(collabID, sourceRepoPath string) (*Collaboration, error) {
	baseDir, err := cm.collabAssetsBaseDir()
	if err != nil {
		return nil, err
	}

	cm.mu.Lock()
	defer cm.mu.Unlock()

	c, ok := cm.collaborations[collabID]
	if !ok {
		return nil, fmt.Errorf("collaboration %s not found", collabID)
	}
	if c.ExecutionMode != ExecutionModeWorktree {
		return nil, fmt.Errorf("collaboration %s is not in worktree execution mode", collabID[:8])
	}
	if c.Phase != PhaseExecuting {
		return nil, fmt.Errorf("collaboration is in %s phase, expected executing", c.Phase)
	}
	if strings.TrimSpace(c.WorkingDirectory) != "" {
		return c, nil
	}

	repo := strings.TrimSpace(c.SourceRepoPath)
	if repo == "" {
		repo = strings.TrimSpace(sourceRepoPath)
	}
	if repo == "" {
		return nil, fmt.Errorf("source repository path is required for worktree execution")
	}
	if err := collabworktree.ValidateGitRepo(repo); err != nil {
		return nil, err
	}
	repoAbs, err := filepath.Abs(repo)
	if err != nil {
		return nil, fmt.Errorf("source repository path: %w", err)
	}
	c.SourceRepoPath = repoAbs

	res, err := collabworktree.Create(collabworktree.CreateOptions{
		SourceRepoPath: repoAbs,
		AssetsRoot:     baseDir,
		CollabID:       c.ID,
		Branch:         c.WorktreeBranch,
	})
	if err != nil {
		return nil, err
	}
	c.WorkingDirectory = res.WorktreePath
	c.WorktreeBranch = res.Branch
	c.UpdatedAt = time.Now()
	return c, nil
}

// CleanupWorktree removes the collaboration worktree directory (best effort).
func (cm *CollaborationManager) CleanupWorktree(collabID string) {
	cm.mu.RLock()
	c, ok := cm.collaborations[collabID]
	cm.mu.RUnlock()
	if !ok || c == nil {
		return
	}
	cm.cleanupWorktreeLocked(c)
}

func (cm *CollaborationManager) cleanupWorktreeLocked(c *Collaboration) {
	if c == nil || c.ExecutionMode != ExecutionModeWorktree {
		return
	}
	worktree := strings.TrimSpace(c.WorkingDirectory)
	if worktree == "" {
		return
	}
	if err := collabworktree.Remove(collabworktree.RemoveOptions{
		WorktreePath:   worktree,
		SourceRepoPath: c.SourceRepoPath,
		Branch:         c.WorktreeBranch,
		DeleteBranch:   false,
	}); err != nil {
		log.Printf("[CollaborationManager] worktree cleanup for %s: %v", c.ID[:8], err)
	}
}

// PlannedWorktreeDirectory returns the path where a worktree would be created.
func (cm *CollaborationManager) PlannedWorktreeDirectory(collabID string) (string, error) {
	baseDir, err := cm.collabAssetsBaseDir()
	if err != nil {
		return "", err
	}
	return collabworktree.PlannedWorktreePath(baseDir, collabID), nil
}

func (cm *CollaborationManager) createSandboxWorkingDir(c *Collaboration, baseDir string) error {
	if err := os.MkdirAll(baseDir, 0755); err != nil {
		return fmt.Errorf("collaboration working directory: mkdir base: %w", err)
	}
	workDir := filepath.Join(baseDir, c.ID)
	if err := os.MkdirAll(workDir, 0755); err != nil {
		return fmt.Errorf("collaboration working directory: mkdir: %w", err)
	}
	absWork, err := filepath.Abs(workDir)
	if err != nil {
		return fmt.Errorf("collaboration working directory: abs: %w", err)
	}
	c.WorkingDirectory = absWork
	return nil
}

func (cm *CollaborationManager) createWorktreeIfReady(c *Collaboration, baseDir string) error {
	if strings.TrimSpace(c.SourceRepoPath) == "" {
		// Deferred until workspace ack supplies repo path.
		return nil
	}
	if err := collabworktree.ValidateGitRepo(c.SourceRepoPath); err != nil {
		return err
	}
	res, err := collabworktree.Create(collabworktree.CreateOptions{
		SourceRepoPath: c.SourceRepoPath,
		AssetsRoot:     baseDir,
		CollabID:       c.ID,
		Branch:         c.WorktreeBranch,
	})
	if err != nil {
		return err
	}
	c.WorkingDirectory = res.WorktreePath
	c.WorktreeBranch = res.Branch
	return nil
}
