package collaboration

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// ProjectCollabsDirName is the folder under a user project where collaboration
// artifacts and execution sandboxes are stored.
const ProjectCollabsDirName = "collabs"

// ProjectCollabDirectory returns <sourceRepo>/collabs/<collabID>.
func ProjectCollabDirectory(sourceRepoPath, collabID string) (string, error) {
	repo := strings.TrimSpace(sourceRepoPath)
	id := strings.TrimSpace(collabID)
	if repo == "" || id == "" {
		return "", fmt.Errorf("source repository and collaboration id are required")
	}
	abs, err := filepath.Abs(repo)
	if err != nil {
		return "", fmt.Errorf("source repository path: %w", err)
	}
	return filepath.Join(filepath.Clean(abs), ProjectCollabsDirName, id), nil
}

// ProjectCollabRelPath returns the path relative to the project root (collabs/<id>).
func ProjectCollabRelPath(collabID string) string {
	id := strings.TrimSpace(collabID)
	if id == "" {
		return ""
	}
	return filepath.ToSlash(filepath.Join(ProjectCollabsDirName, id))
}

// EnsureProjectCollabDir creates <sourceRepo>/collabs/<collabID> if needed.
func EnsureProjectCollabDir(sourceRepoPath, collabID string) (string, error) {
	dir, err := ProjectCollabDirectory(sourceRepoPath, collabID)
	if err != nil {
		return "", err
	}
	if err := os.MkdirAll(dir, 0755); err != nil {
		return "", fmt.Errorf("create project collab directory: %w", err)
	}
	return dir, nil
}

// UsesProjectCollabDir reports whether artifacts and execution use the in-repo collabs folder.
func UsesProjectCollabDir(c *Collaboration) bool {
	if c == nil {
		return false
	}
	return strings.TrimSpace(c.SourceRepoPath) != ""
}

// CollabAssetPaths returns durable markdown paths for a collaboration.
// When SourceRepoPath is set, files live under <repo>/collabs/<id>/.
// Otherwise they use <assetsBaseDir>/reviews/<id>/ (legacy layout).
func CollabAssetPaths(c *Collaboration, assetsBaseDir string) ReviewAssetPaths {
	if UsesProjectCollabDir(c) {
		if dir, err := ProjectCollabDirectory(c.SourceRepoPath, c.ID); err == nil {
			return reviewAssetPathsInDirectory(dir)
		}
	}
	return ReviewAssetPathsFor(assetsBaseDir, c.ID)
}

func reviewAssetPathsInDirectory(dir string) ReviewAssetPaths {
	dir = filepath.Clean(strings.TrimSpace(dir))
	return ReviewAssetPaths{
		Directory:       dir,
		Plan:            filepath.Join(dir, ReviewAssetsPlanFileName),
		PlanningSummary: filepath.Join(dir, ReviewAssetsPlanningSummaryName),
		SessionSummary:  filepath.Join(dir, ReviewAssetsSessionSummaryName),
		Index:           filepath.Join(dir, ReviewAssetsIndexFileName),
	}
}

// PlannedOutputDirectory returns the directory agents should write to, even before execution starts.
func PlannedOutputDirectory(c *Collaboration, assetsBaseDir string) string {
	if c == nil {
		return ""
	}
	if wd := strings.TrimSpace(c.WorkingDirectory); wd != "" {
		return wd
	}
	if UsesProjectCollabDir(c) {
		if dir, err := ProjectCollabDirectory(c.SourceRepoPath, c.ID); err == nil {
			return dir
		}
	}
	if strings.TrimSpace(assetsBaseDir) != "" && strings.TrimSpace(c.ID) != "" {
		return filepath.Join(filepath.Clean(assetsBaseDir), c.ID)
	}
	return ""
}
