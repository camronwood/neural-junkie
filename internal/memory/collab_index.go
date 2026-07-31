package memory

import (
	"context"
	"log"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/camronwood/neural-junkie/internal/collaboration"
)

// IndexReviewAssetPaths indexes durable collab markdown files after WriteReviewAssets.
// Also picks up findings.md (and other *.md) in the same directory so retrieval
// does not depend solely on the file-approval path.
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
	dir := strings.TrimSpace(paths.Directory)
	if dir == "" && strings.TrimSpace(paths.Plan) != "" {
		dir = filepath.Dir(paths.Plan)
	}
	if dir != "" {
		indexCollabDirMarkdown(dir, collabID, channel)
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

// indexCollabDirMarkdown indexes every *.md under a collab/review directory
// (findings.md, notes, deliverables) with a stable collabs|reviews rel path.
func indexCollabDirMarkdown(dir, collabID, channel string) {
	dir = strings.TrimSpace(dir)
	collabID = strings.TrimSpace(collabID)
	if dir == "" || collabID == "" {
		return
	}
	entries, err := os.ReadDir(dir)
	if err != nil {
		return
	}
	prefix := "reviews"
	if base := filepath.Base(filepath.Dir(dir)); base == "collabs" {
		prefix = "collabs"
	} else if filepath.Base(dir) == collabID && strings.Contains(filepath.ToSlash(dir), "/collabs/") {
		prefix = "collabs"
	}
	for _, e := range entries {
		if e.IsDir() {
			continue
		}
		name := e.Name()
		if !strings.HasSuffix(strings.ToLower(name), ".md") {
			continue
		}
		abs := filepath.Join(dir, name)
		rel := filepath.ToSlash(filepath.Join(prefix, collabID, name))
		IndexCollabFile(abs, rel, collabID, channel)
	}
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

// BackfillWorkspaceCollabs indexes markdown under <workspaceRoot>/collabs/<id>/*.md.
// Call on hub start and whenever a workspace root becomes active so findings.md
// survives restart without waiting for a new file approval.
func BackfillWorkspaceCollabs(ctx context.Context, workspaceRoot string) error {
	if !memoryEnabled() {
		return nil
	}
	workspaceRoot = strings.TrimSpace(workspaceRoot)
	if workspaceRoot == "" {
		return nil
	}
	collabsRoot := filepath.Join(workspaceRoot, collaboration.ProjectCollabsDirName)
	return backfillCollabTree(ctx, collabsRoot, "collabs")
}

// backfillCollabTree walks <root>/<collabID>/*.md and indexes each file.
// relPrefix is "collabs" or "reviews".
func backfillCollabTree(ctx context.Context, root, relPrefix string) error {
	root = strings.TrimSpace(root)
	if root == "" {
		return nil
	}
	entries, err := os.ReadDir(root)
	if err != nil {
		if os.IsNotExist(err) {
			return nil
		}
		return err
	}
	var indexed int
	for _, e := range entries {
		if !e.IsDir() {
			continue
		}
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
		}
		collabID := e.Name()
		dir := filepath.Join(root, collabID)
		_ = filepath.WalkDir(dir, func(path string, d os.DirEntry, err error) error {
			if err != nil || d.IsDir() {
				return nil
			}
			if !strings.HasSuffix(strings.ToLower(d.Name()), ".md") {
				return nil
			}
			relFromCollab, err := filepath.Rel(dir, path)
			if err != nil {
				return nil
			}
			rel := filepath.ToSlash(filepath.Join(relPrefix, collabID, relFromCollab))
			has, _ := globalStore.HasSource(path)
			if has {
				return nil
			}
			if err := indexCollabFile(ctx, path, rel, collabID, ""); err != nil {
				log.Printf("[memory] backfill %s: %v", rel, err)
				return nil
			}
			indexed++
			return nil
		})
	}
	if indexed > 0 {
		log.Printf("[memory] backfill %s indexed %d markdown files under %s", relPrefix, indexed, root)
	}
	return nil
}

// ScheduleWorkspaceBackfill indexes collabs under a workspace root in the background.
func ScheduleWorkspaceBackfill(workspaceRoot string) {
	if !memoryEnabled() || strings.TrimSpace(workspaceRoot) == "" {
		return
	}
	go func() {
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Minute)
		defer cancel()
		if err := BackfillWorkspaceCollabs(ctx, workspaceRoot); err != nil {
			log.Printf("[memory] backfill workspace collabs: %v", err)
		}
	}()
}
