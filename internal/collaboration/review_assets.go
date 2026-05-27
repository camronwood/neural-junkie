package collaboration

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

const (
	ReviewAssetsDirName             = "reviews"
	ReviewAssetsPlanFileName        = "plan.md"
	ReviewAssetsPlanningSummaryName = "planning-summary.md"
	ReviewAssetsSessionSummaryName  = "session-summary.md"
	ReviewAssetsIndexFileName       = "README.md"
)

// ReviewAssetPaths describes the durable markdown files for a collaboration.
type ReviewAssetPaths struct {
	Directory       string
	Plan            string
	PlanningSummary string
	SessionSummary  string
	Index           string
}

// ReviewAssetsDirectory returns the persistent review folder for a collaboration.
func ReviewAssetsDirectory(baseDir, collabID string) string {
	return filepath.Join(strings.TrimSpace(baseDir), ReviewAssetsDirName, strings.TrimSpace(collabID))
}

// ReviewAssetPathsFor returns all stable review asset paths for a collaboration.
func ReviewAssetPathsFor(baseDir, collabID string) ReviewAssetPaths {
	dir := ReviewAssetsDirectory(baseDir, collabID)
	return ReviewAssetPaths{
		Directory:       dir,
		Plan:            filepath.Join(dir, ReviewAssetsPlanFileName),
		PlanningSummary: filepath.Join(dir, ReviewAssetsPlanningSummaryName),
		SessionSummary:  filepath.Join(dir, ReviewAssetsSessionSummaryName),
		Index:           filepath.Join(dir, ReviewAssetsIndexFileName),
	}
}

// WriteReviewAssets writes any available plan and recap markdown to durable files.
func WriteReviewAssets(baseDir string, c *Collaboration) (*ReviewAssetPaths, error) {
	if c == nil {
		return nil, fmt.Errorf("collaboration is required")
	}
	if strings.TrimSpace(baseDir) == "" {
		return nil, fmt.Errorf("collaboration assets root is required")
	}
	if strings.TrimSpace(c.ID) == "" {
		return nil, fmt.Errorf("collaboration id is required")
	}

	paths := CollabAssetPaths(c, baseDir)
	if err := os.MkdirAll(paths.Directory, 0755); err != nil {
		return &paths, fmt.Errorf("create review assets directory: %w", err)
	}

	var errs []string
	if c.Plan != nil && strings.TrimSpace(c.Plan.Content) != "" {
		if err := writeMarkdownAsset(paths.Plan, c.Plan.Content); err != nil {
			errs = append(errs, fmt.Sprintf("%s: %v", ReviewAssetsPlanFileName, err))
		}
	}
	if strings.TrimSpace(c.PlanningRecap) != "" {
		if err := writeMarkdownAsset(paths.PlanningSummary, c.PlanningRecap); err != nil {
			errs = append(errs, fmt.Sprintf("%s: %v", ReviewAssetsPlanningSummaryName, err))
		}
	}
	if strings.TrimSpace(c.SessionRecap) != "" {
		if err := writeMarkdownAsset(paths.SessionSummary, c.SessionRecap); err != nil {
			errs = append(errs, fmt.Sprintf("%s: %v", ReviewAssetsSessionSummaryName, err))
		}
	}
	if err := writeMarkdownAsset(paths.Index, renderReviewAssetsIndex(c, paths)); err != nil {
		errs = append(errs, fmt.Sprintf("%s: %v", ReviewAssetsIndexFileName, err))
	}

	if len(errs) > 0 {
		return &paths, fmt.Errorf("write review assets: %s", strings.Join(errs, "; "))
	}
	return &paths, nil
}

func writeMarkdownAsset(path, content string) error {
	content = strings.TrimSpace(content)
	if content != "" {
		content += "\n"
	}

	dir := filepath.Dir(path)
	tmp, err := os.CreateTemp(dir, "."+filepath.Base(path)+".*.tmp")
	if err != nil {
		return err
	}
	tmpPath := tmp.Name()
	defer func() {
		_ = os.Remove(tmpPath)
	}()

	if _, err := tmp.WriteString(content); err != nil {
		_ = tmp.Close()
		return err
	}
	if err := tmp.Chmod(0644); err != nil {
		_ = tmp.Close()
		return err
	}
	if err := tmp.Close(); err != nil {
		return err
	}
	return os.Rename(tmpPath, path)
}

func renderReviewAssetsIndex(c *Collaboration, paths ReviewAssetPaths) string {
	var b strings.Builder
	b.WriteString("# Collaboration Review Assets\n\n")
	b.WriteString(fmt.Sprintf("- **ID:** `%s`\n", c.ID))
	if strings.TrimSpace(c.Title) != "" {
		b.WriteString(fmt.Sprintf("- **Title:** %s\n", c.Title))
	}
	if strings.TrimSpace(c.Description) != "" {
		b.WriteString(fmt.Sprintf("- **Goal:** %s\n", c.Description))
	}
	if c.Phase != "" {
		b.WriteString(fmt.Sprintf("- **Phase:** `%s`\n", c.Phase))
	}
	if strings.TrimSpace(paths.Directory) != "" {
		b.WriteString(fmt.Sprintf("- **Review assets:** `%s`\n", paths.Directory))
	}
	if c.ExecutionMode != "" {
		b.WriteString(fmt.Sprintf("- **Execution mode:** `%s`\n", c.ExecutionMode))
	}
	if strings.TrimSpace(c.WorkingDirectory) != "" {
		b.WriteString(fmt.Sprintf("- **Workspace:** `%s`\n", c.WorkingDirectory))
	}
	if strings.TrimSpace(c.SourceRepoPath) != "" {
		b.WriteString(fmt.Sprintf("- **Source repo:** `%s`\n", c.SourceRepoPath))
	}
	if strings.TrimSpace(c.WorktreeBranch) != "" {
		b.WriteString(fmt.Sprintf("- **Worktree branch:** `%s`\n", c.WorktreeBranch))
	}

	b.WriteString("\n## Files\n\n")
	if c.Plan != nil && strings.TrimSpace(c.Plan.Content) != "" {
		b.WriteString(fmt.Sprintf("- [`%s`](%s)\n", ReviewAssetsPlanFileName, ReviewAssetsPlanFileName))
	}
	if strings.TrimSpace(c.PlanningRecap) != "" {
		b.WriteString(fmt.Sprintf("- [`%s`](%s)\n", ReviewAssetsPlanningSummaryName, ReviewAssetsPlanningSummaryName))
	}
	if strings.TrimSpace(c.SessionRecap) != "" {
		b.WriteString(fmt.Sprintf("- [`%s`](%s)\n", ReviewAssetsSessionSummaryName, ReviewAssetsSessionSummaryName))
	}

	if len(c.Tasks) > 0 {
		b.WriteString("\n## Tasks\n\n")
		for i, task := range c.Tasks {
			assignee := strings.TrimSpace(task.AssignedName)
			if assignee == "" {
				assignee = "unassigned"
			}
			status := task.Status
			if status == "" {
				status = TaskPending
			}
			b.WriteString(fmt.Sprintf("%d. **%s** - `%s` (@%s)\n", i+1, task.Title, status, assignee))
		}
	}
	return b.String()
}
