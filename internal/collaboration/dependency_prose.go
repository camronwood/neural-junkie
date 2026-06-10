package collaboration

import (
	"fmt"
	"regexp"
	"strconv"
	"strings"
)

// hardDepProseRe matches "Task N depends on Task M" (optional list prefix).
var hardDepProseRe = regexp.MustCompile(`(?i)^(?:[-*]\s+)?task\s+(\d+)\s+depends\s+on\s+(.+)$`)

// taskNumTokenRe finds task numbers in a dependency tail ("Task 2", "task 3", bare "2").
var taskNumTokenRe = regexp.MustCompile(`(?i)(?:task\s*)?(\d+)`)

var taskDescDepRe = regexp.MustCompile(`(?i)(?:based on (?:the plan in )?|depends on |after )task\s+(\d+)`)

// parseHardDependencyProse extracts 1-based task indices from dependency prose.
// Returns false for informational lines ("can be started", "should reference" without depends on).
func parseHardDependencyProse(line string) (fromIndex int, toIndices []int, ok bool) {
	trimmed := strings.TrimSpace(line)
	if trimmed == "" || !isTaskDependencyProse(trimmed) {
		return 0, nil, false
	}
	rest := strings.TrimSpace(taskListPrefixRe.ReplaceAllString(trimmed, ""))
	m := hardDepProseRe.FindStringSubmatch(rest)
	if m == nil {
		return 0, nil, false
	}
	from, err := strconv.Atoi(m[1])
	if err != nil || from < 1 {
		return 0, nil, false
	}
	tail := strings.TrimSpace(m[2])
	lowerTail := strings.ToLower(tail)
	if strings.HasPrefix(lowerTail, "task ") {
		// "depends on task 2" — parse task numbers from tail
	} else if !strings.Contains(lowerTail, "depend") {
		// tail should reference at least one task number
	}
	var deps []int
	seen := make(map[int]bool)
	for _, sub := range taskNumTokenRe.FindAllStringSubmatch(tail, -1) {
		if len(sub) < 2 {
			continue
		}
		n, err := strconv.Atoi(sub[1])
		if err != nil || n < 1 || n == from || seen[n] {
			continue
		}
		seen[n] = true
		deps = append(deps, n)
	}
	if len(deps) == 0 {
		return 0, nil, false
	}
	return from, deps, true
}

// ApplyPlanDependencyProse scans plan markdown for "Task N depends on Task M" prose
// and merges hard dependencies into tasks (1-based indices). Mutates tasks in place.
// Returns human-readable warnings (e.g. dropped cycle edges).
func ApplyPlanDependencyProse(planContent string, tasks []CollaborationTask) []string {
	if strings.TrimSpace(planContent) == "" || len(tasks) == 0 {
		return nil
	}
	// Snapshot deps before prose merge so we can revert on cycle.
	before := snapshotTaskDeps(tasks)
	var proseEdges []proseEdge

	for _, line := range strings.Split(planContent, "\n") {
		fromIdx, toIndices, ok := parseHardDependencyProse(line)
		if !ok {
			continue
		}
		if fromIdx > len(tasks) {
			continue
		}
		task := &tasks[fromIdx-1]
		for _, toIdx := range toIndices {
			if toIdx > len(tasks) || toIdx == fromIdx {
				continue
			}
			ref := strconv.Itoa(toIdx)
			if !depRefPresent(task.Dependencies, ref) {
				task.Dependencies = append(task.Dependencies, ref)
				proseEdges = append(proseEdges, proseEdge{from: fromIdx, to: toIdx, ref: ref})
			}
		}
	}

	if len(proseEdges) == 0 {
		return nil
	}

	NormalizeDependencies(tasks)
	if err := ValidateDAG(tasks); err != nil {
		restoreTaskDeps(tasks, before)
		return []string{fmt.Sprintf("Dropped dependency prose edge(s) — %v", err)}
	}
	return nil
}

type proseEdge struct {
	from int
	to   int
	ref  string
}

func snapshotTaskDeps(tasks []CollaborationTask) [][]string {
	out := make([][]string, len(tasks))
	for i, t := range tasks {
		if len(t.Dependencies) > 0 {
			out[i] = append([]string(nil), t.Dependencies...)
		}
	}
	return out
}

func restoreTaskDeps(tasks []CollaborationTask, deps [][]string) {
	for i := range tasks {
		if i < len(deps) && len(deps[i]) > 0 {
			tasks[i].Dependencies = append([]string(nil), deps[i]...)
		} else {
			tasks[i].Dependencies = nil
		}
	}
}

func depRefPresent(ss []string, want string) bool {
	for _, s := range ss {
		if s == want {
			return true
		}
	}
	return false
}

// inferDepsFromTaskDescriptions adds 1-based dependency refs from task title/description prose.
func inferDepsFromTaskDescriptions(tasks []CollaborationTask) {
	for i := range tasks {
		combined := tasks[i].Title + " " + tasks[i].Description
		for _, sub := range taskDescDepRe.FindAllStringSubmatch(combined, -1) {
			if len(sub) < 2 {
				continue
			}
			ref := sub[1]
			if depRefPresent(tasks[i].Dependencies, ref) {
				continue
			}
			n, err := strconv.Atoi(ref)
			if err != nil || n < 1 || n > len(tasks) || n == i+1 {
				continue
			}
			tasks[i].Dependencies = append(tasks[i].Dependencies, ref)
		}
	}
}
