package collaboration

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// TaskPathIssue records a path referenced in a plan or task that is missing on disk.
type TaskPathIssue struct {
	TaskIndex int    // 0 = plan artifact; 1+ = task number
	TaskTitle string
	Path      string
}

// ExtractPathReferences finds repo-relative path strings mentioned in text.
func ExtractPathReferences(text, repoRoot string) []string {
	text = strings.TrimSpace(text)
	if text == "" {
		return nil
	}
	repoRoot = strings.TrimSpace(repoRoot)
	seen := make(map[string]bool)
	var out []string
	add := func(p string) {
		p = normalizePathReference(p, repoRoot)
		if p == "" || seen[p] {
			return
		}
		if !looksLikeRepoPath(p) {
			return
		}
		seen[p] = true
		out = append(out, p)
	}
	for _, m := range collabPathRefPattern.FindAllStringSubmatch(text, -1) {
		if len(m) > 1 {
			add(m[1])
		}
	}
	if repoRoot != "" {
		absRepo, err := filepath.Abs(repoRoot)
		if err == nil {
			lower := strings.ToLower(text)
			repoLower := strings.ToLower(absRepo)
			if idx := strings.Index(lower, repoLower); idx >= 0 {
				rest := text[idx+len(absRepo):]
				for _, m := range collabPathRefPattern.FindAllStringSubmatch(rest, -1) {
					if len(m) > 1 {
						add(m[1])
					}
				}
			}
		}
	}
	return out
}

func normalizePathReference(p, repoRoot string) string {
	p = strings.Trim(p, "`\"' ")
	p = strings.TrimPrefix(p, "./")
	p = strings.TrimSuffix(p, "/")
	p = strings.TrimSuffix(p, ".")
	if p == "" {
		return ""
	}
	if strings.HasPrefix(p, "/") && repoRoot != "" {
		absRepo, err := filepath.Abs(repoRoot)
		if err != nil {
			return ""
		}
		absRef, err := filepath.Abs(p)
		if err != nil {
			return ""
		}
		rel, err := filepath.Rel(absRepo, absRef)
		if err != nil || strings.HasPrefix(rel, "..") {
			return ""
		}
		return filepath.ToSlash(rel)
	}
	return filepath.ToSlash(p)
}

func looksLikeRepoPath(p string) bool {
	if p == "" || strings.Contains(p, "://") {
		return false
	}
	if strings.Contains(p, "/") {
		return true
	}
	return strings.Contains(filepath.Base(p), ".")
}

func pathExistsInRepo(repoRoot, rel string) bool {
	rel = strings.TrimSpace(rel)
	if rel == "" {
		return false
	}
	if strings.HasPrefix(filepath.ToSlash(rel), "collabs/") {
		return true
	}
	abs := filepath.Join(repoRoot, filepath.FromSlash(rel))
	st, err := os.Stat(abs)
	return err == nil && (st.IsDir() || strings.Contains(filepath.Base(rel), "."))
}

// ValidateCollaborationPaths checks plan and task text for references to missing paths.
func ValidateCollaborationPaths(c *Collaboration) []TaskPathIssue {
	repoRoot := ""
	if c != nil {
		repoRoot = strings.TrimSpace(c.SourceRepoPath)
	}
	if repoRoot == "" {
		return nil
	}
	var issues []TaskPathIssue
	seen := make(map[string]bool)

	recordMissing := func(taskIndex int, title, path string) {
		key := fmt.Sprintf("%d:%s", taskIndex, path)
		if seen[key] {
			return
		}
		seen[key] = true
		issues = append(issues, TaskPathIssue{
			TaskIndex: taskIndex,
			TaskTitle: title,
			Path:      path,
		})
	}

	checkText := func(taskIndex int, title, text string) {
		for _, ref := range ExtractPathReferences(text, repoRoot) {
			if pathExistsInRepo(repoRoot, ref) {
				continue
			}
			recordMissing(taskIndex, title, ref)
		}
	}

	if c.Plan != nil && strings.TrimSpace(c.Plan.Content) != "" {
		checkText(0, "Plan", c.Plan.Content)
	}
	for i, task := range c.Tasks {
		body := strings.TrimSpace(task.Title + "\n" + task.Description)
		title := strings.TrimSpace(task.Title)
		if title == "" {
			title = fmt.Sprintf("Task %d", i+1)
		}
		checkText(i+1, title, body)
	}
	return issues
}

// FormatTaskPathWarnings renders a user-facing markdown warning block.
func FormatTaskPathWarnings(issues []TaskPathIssue, repoRoot string) string {
	if len(issues) == 0 {
		return ""
	}
	var b strings.Builder
	b.WriteString("\n\n⚠️ **Path check** (referenced in the plan/tasks but not found under the project root):\n")
	if repoRoot != "" {
		b.WriteString(fmt.Sprintf("**Project:** `%s`\n\n", repoRoot))
	}
	for _, iss := range issues {
		where := "Plan"
		if iss.TaskIndex > 0 {
			where = fmt.Sprintf("Task %d (%s)", iss.TaskIndex, iss.TaskTitle)
		}
		b.WriteString(fmt.Sprintf("- `%s` — cited in **%s**\n", iss.Path, where))
	}
	b.WriteString("\nConsider **Revise** to fix paths before agents run, or approve anyway if the references are intentional.\n")
	return b.String()
}
