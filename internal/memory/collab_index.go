package memory

import (
	"os"
	"path/filepath"
	"strings"

	"github.com/camronwood/neural-junkie/internal/collaboration"
)

// IndexReviewAssetPaths indexes durable collab markdown files after WriteReviewAssets.
func IndexReviewAssetPaths(paths *collaboration.ReviewAssetPaths, collabID, channel string) {
	if !memoryEnabled() || paths == nil || strings.TrimSpace(collabID) == "" {
		return
	}
	for _, item := range []struct {
		abs, name string
	}{
		{paths.Plan, collaboration.ReviewAssetsPlanFileName},
		{paths.PlanningSummary, collaboration.ReviewAssetsPlanningSummaryName},
		{paths.SessionSummary, collaboration.ReviewAssetsSessionSummaryName},
		{paths.Index, collaboration.ReviewAssetsIndexFileName},
	} {
		indexReviewFileIfPresent(item.abs, item.name, collabID, channel)
	}
}

func indexReviewFileIfPresent(absPath, fileName, collabID, channel string) {
	absPath = strings.TrimSpace(absPath)
	if absPath == "" {
		return
	}
	if _, err := os.Stat(absPath); err != nil {
		return
	}
	rel := fileName
	if dir := filepath.Dir(absPath); dir != "" {
		if base := filepath.Base(filepath.Dir(absPath)); base == "collabs" || base == collabID {
			rel = filepath.ToSlash(filepath.Join("collabs", collabID, fileName))
		} else {
			rel = filepath.ToSlash(filepath.Join("reviews", collabID, fileName))
		}
	}
	IndexCollabFile(absPath, rel, collabID, channel)
}

// IndexCollabMarkdownRel indexes a workspace-relative collabs/*.md path after file approval.
func IndexCollabMarkdownRel(workspaceRoot, relPath, channel string) {
	if !memoryEnabled() {
		return
	}
	relPath = filepath.ToSlash(strings.TrimSpace(relPath))
	if !strings.HasPrefix(relPath, "collabs/") || !strings.HasSuffix(strings.ToLower(relPath), ".md") {
		return
	}
	workspaceRoot = strings.TrimSpace(workspaceRoot)
	if workspaceRoot == "" {
		return
	}
	abs := filepath.Join(workspaceRoot, filepath.FromSlash(relPath))
	collabID := CollabIDFromRelPath(relPath)
	IndexCollabFile(abs, relPath, collabID, channel)
}
