package collaboration

import (
	"regexp"
	"strconv"
	"strings"
)

// handoffCompleteRE matches plan handoff lines like "Task 2 (@Cursor) — Complete".
var handoffCompleteRE = regexp.MustCompile(`(?im)^\s*(?:[-*]\s*)?task\s+(\d+)\b(?:\s*\([^)]+\))?\s*(?:—|--|-|→|:)\s*(?:complete|completed|done)\s*$`)

// SyncTaskStatusFromPlanHandoff marks tasks completed when handoff text lists them as done.
// Returns task IDs that were updated. Only upgrades to completed; never downgrades.
func SyncTaskStatusFromPlanHandoff(content string, tasks []CollaborationTask) []string {
	if strings.TrimSpace(content) == "" || len(tasks) == 0 {
		return nil
	}
	seen := make(map[int]bool)
	var updated []string
	for _, line := range strings.Split(content, "\n") {
		m := handoffCompleteRE.FindStringSubmatch(line)
		if len(m) < 2 {
			continue
		}
		n, err := strconv.Atoi(m[1])
		if err != nil || n < 1 || n > len(tasks) || seen[n] {
			continue
		}
		seen[n] = true
		t := &tasks[n-1]
		if t.Status == TaskCompleted {
			continue
		}
		updated = append(updated, t.ID)
	}
	return updated
}
