package collaboration

import (
	"fmt"
	"strings"
)

// NormalizeAndValidateTasksForExecution parses the plan, filters low-quality rows,
// merges near-duplicates, caps task count, and returns human-readable warnings.
func NormalizeAndValidateTasksForExecution(c *Collaboration) ([]CollaborationTask, []string) {
	if c == nil || c.Plan == nil {
		return nil, nil
	}
	planContent := strings.TrimSpace(c.Plan.Content)
	if planContent == "" {
		return nil, nil
	}

	var warnings []string
	tasks := ExtractTasksFromPlan(planContent, c.Agents)
	EnrichTasksWithPlanDeliverables(planContent, tasks)
	NormalizeTaskDeliverablePathsForSandbox(c, tasks)
	EnrichTasksWithContextPaths(tasks, c.SourceRepoPath)

	filtered := make([]CollaborationTask, 0, len(tasks))
	droppedWeak := 0
	for _, t := range tasks {
		if !taskPassesExecutionQuality(t) {
			droppedWeak++
			continue
		}
		filtered = append(filtered, t)
	}
	if droppedWeak > 0 {
		warnings = append(warnings, fmt.Sprintf("Dropped %d vague or malformed task line(s) from the plan.", droppedWeak))
	}
	tasks = mergeNearDuplicateTasks(filtered)
	tasks, conflictWarnings := mergeConflictingDeliverableTasks(tasks)
	warnings = append(warnings, conflictWarnings...)

	if len(tasks) > MaxTasksPerCollaboration {
		warnings = append(warnings, fmt.Sprintf("Task list truncated to %d (plan had more).", MaxTasksPerCollaboration))
		tasks = tasks[:MaxTasksPerCollaboration]
	}

	if w := warnMissingFileDeliverableTasks(c, tasks); w != "" {
		warnings = append(warnings, w)
	}
	if w := warnMissingSourceRepoForFileCollab(c, tasks); w != "" {
		warnings = append(warnings, w)
	}

	if proseWarnings := ApplyPlanDependencyProse(planContent, tasks); len(proseWarnings) > 0 {
		warnings = append(warnings, proseWarnings...)
	}
	inferDepsFromTaskDescriptions(tasks)
	NormalizeDependencies(tasks)

	return tasks, warnings
}

func warnMissingSourceRepoForFileCollab(c *Collaboration, tasks []CollaborationTask) string {
	if c == nil || strings.TrimSpace(c.SourceRepoPath) != "" || !goalNeedsFileDeliverables(c) {
		return ""
	}
	for _, t := range tasks {
		if TaskRequiresFileDeliverable(t) {
			return "No project workspace is bound — deliverables will land in an isolated sandbox. Re-start with `/collaborate --workspace` (or pick a folder in the desktop collab form) to read your repo and write under `<project>/collabs/<id>/`."
		}
	}
	return ""
}

func taskPassesExecutionQuality(t CollaborationTask) bool {
	title := strings.TrimSpace(t.Title)
	desc := strings.TrimSpace(t.Description)
	if title == "" && desc == "" {
		return false
	}
	combined := strings.TrimSpace(title + " " + desc)
	if combined == "" {
		return false
	}
	if TaskRequiresFileDeliverable(t) {
		return true
	}
	if isTaskDependencyProse(combined) {
		return false
	}
	if isWeakTaskFragment(combined) {
		return false
	}
	lower := strings.ToLower(combined)
	if strings.HasPrefix(lower, "review current ") && !strings.Contains(lower, "collabs/") && !strings.Contains(lower, ".md") {
		return false
	}
	return true
}

// mergeConflictingDeliverableTasks drops cross-assignee rows that target the same deliverable path.
func mergeConflictingDeliverableTasks(tasks []CollaborationTask) ([]CollaborationTask, []string) {
	if len(tasks) < 2 {
		return tasks, nil
	}
	type slot struct {
		idx   int
		score int
	}
	byPath := make(map[string]slot)
	drop := make(map[int]bool)
	var warnings []string
	for i, t := range tasks {
		if drop[i] {
			continue
		}
		paths := ReferencedDeliverablePaths(t)
		if len(paths) == 0 {
			continue
		}
		primary := paths[0]
		sc := taskDeliverableScore(t)
		if prev, ok := byPath[primary]; ok {
			kept := tasks[prev.idx]
			if drop[prev.idx] {
				byPath[primary] = slot{idx: i, score: sc}
				continue
			}
			if sameAssignee(t, kept) {
				if sc > prev.score {
					byPath[primary] = slot{idx: i, score: sc}
				}
				continue
			}
			if sc > prev.score {
				drop[prev.idx] = true
				byPath[primary] = slot{idx: i, score: sc}
				warnings = append(warnings,
					fmt.Sprintf("Dropped conflicting task for %s (kept @%s over @%s).",
						primary, t.AssignedName, kept.AssignedName))
			} else {
				drop[i] = true
				warnings = append(warnings,
					fmt.Sprintf("Dropped conflicting task for %s (kept @%s over @%s).",
						primary, kept.AssignedName, t.AssignedName))
			}
		} else {
			byPath[primary] = slot{idx: i, score: sc}
		}
	}
	if len(warnings) == 0 {
		return tasks, nil
	}
	out := make([]CollaborationTask, 0, len(tasks))
	for i, t := range tasks {
		if !drop[i] {
			out = append(out, t)
		}
	}
	return out, warnings
}

func mergeNearDuplicateTasks(tasks []CollaborationTask) []CollaborationTask {
	if len(tasks) < 2 {
		return tasks
	}
	out := make([]CollaborationTask, 0, len(tasks))
	for _, t := range tasks {
		merged := false
		for i, kept := range out {
			if !sameAssignee(t, kept) {
				continue
			}
			if !tasksNearDuplicate(t, kept) {
				continue
			}
			if taskDeliverableScore(t) > taskDeliverableScore(kept) {
				out[i] = t
			}
			merged = true
			break
		}
		if !merged {
			out = append(out, t)
		}
	}
	return DedupeTasks(out)
}

func sameAssignee(a, b CollaborationTask) bool {
	if a.AssignedTo != "" && b.AssignedTo != "" {
		return a.AssignedTo == b.AssignedTo
	}
	return strings.EqualFold(strings.TrimSpace(a.AssignedName), strings.TrimSpace(b.AssignedName))
}

func tasksNearDuplicate(a, b CollaborationTask) bool {
	pa := ReferencedDeliverablePaths(a)
	pb := ReferencedDeliverablePaths(b)
	if len(pa) > 0 && len(pb) > 0 {
		// Same assignee with different file outputs are distinct tasks (e.g. synthesize vs findings.md).
		setA := make(map[string]bool, len(pa))
		for _, p := range pa {
			setA[p] = true
		}
		overlap := false
		for _, p := range pb {
			if setA[p] {
				overlap = true
				break
			}
		}
		if !overlap {
			return false
		}
	}
	if (len(pa) > 0) != (len(pb) > 0) {
		// One file deliverable and one chat-only row — never collapse.
		return false
	}
	ta := normalizeTaskCompareKey(a)
	tb := normalizeTaskCompareKey(b)
	if ta == "" || tb == "" {
		return false
	}
	if ta == tb {
		return true
	}
	if strings.HasPrefix(ta, tb) || strings.HasPrefix(tb, ta) {
		return true
	}
	return tokenOverlapRatio(ta, tb) >= 0.72
}

func normalizeTaskCompareKey(t CollaborationTask) string {
	s := strings.ToLower(strings.TrimSpace(t.Title + " " + t.Description))
	return taskDedupeNoise.ReplaceAllString(s, " ")
}

func tokenOverlapRatio(a, b string) float64 {
	aw := tokenSet(a)
	bw := tokenSet(b)
	if len(aw) == 0 || len(bw) == 0 {
		return 0
	}
	inter := 0
	for w := range aw {
		if bw[w] {
			inter++
		}
	}
	denom := len(aw)
	if len(bw) > denom {
		denom = len(bw)
	}
	return float64(inter) / float64(denom)
}

func tokenSet(s string) map[string]bool {
	parts := strings.Fields(s)
	out := make(map[string]bool, len(parts))
	for _, p := range parts {
		if len(p) < 3 {
			continue
		}
		out[p] = true
	}
	return out
}

func taskDeliverableScore(t CollaborationTask) int {
	score := 0
	combined := strings.ToLower(t.Title + " " + t.Description)
	if strings.Contains(combined, "collabs/") {
		score += 4
	}
	if strings.Contains(combined, ".md") || strings.Contains(combined, ".yaml") || strings.Contains(combined, ".yml") {
		score += 3
	}
	for _, v := range []string{"write", "draft", "create", "produce", "implement", "emit"} {
		if strings.Contains(combined, v) {
			score += 1
		}
	}
	if len(strings.TrimSpace(t.Description)) > 40 {
		score += 1
	}
	return score
}

func goalNeedsFileDeliverables(c *Collaboration) bool {
	text := strings.ToLower(strings.TrimSpace(c.Description))
	if c.Plan != nil {
		text += " " + strings.ToLower(c.Plan.Content)
	}
	needles := []string{"markdown", "write", "document", "deliverable", "implement", "produce", "file", ".md"}
	for _, n := range needles {
		if strings.Contains(text, n) {
			return true
		}
	}
	return false
}

func taskNamesFileDeliverable(t CollaborationTask) bool {
	combined := strings.ToLower(t.Title + " " + t.Description)
	if strings.Contains(combined, "collabs/") {
		return true
	}
	if taskFileExtRE.MatchString(combined) && taskFileVerbRE.MatchString(combined) {
		return true
	}
	return false
}

func warnMissingFileDeliverableTasks(c *Collaboration, tasks []CollaborationTask) string {
	if !goalNeedsFileDeliverables(c) || len(tasks) == 0 {
		return ""
	}
	for _, t := range tasks {
		if taskNamesFileDeliverable(t) {
			return ""
		}
	}
	return "Goal looks file-oriented but no task names a concrete path (e.g. Write collabs/<id>/findings.md). Assign a file deliverable to the appropriate domain owner."
}

// FormatApproveWarnings renders plan validation warnings for chat/UI.
func FormatApproveWarnings(warnings []string) string {
	if len(warnings) == 0 {
		return ""
	}
	var b strings.Builder
	b.WriteString("\n\n**Plan validation**\n")
	for _, w := range warnings {
		b.WriteString("- ")
		b.WriteString(w)
		b.WriteString("\n")
	}
	return b.String()
}
