package collaboration

import (
	"fmt"
	"regexp"
	"strconv"
	"strings"
)

// depLineRe matches "depends: 1, 2", "after: task-3", "- depends: 1".
var depLineRe = regexp.MustCompile(`(?i)^(?:[-*]\s+)?(?:depends|after|blocked[- ]?by)\s*:\s*(.+)$`)

// ParseDependencyRefs extracts raw dependency references from a depends/after line.
func ParseDependencyRefs(line string) []string {
	m := depLineRe.FindStringSubmatch(strings.TrimSpace(line))
	if m == nil {
		return nil
	}
	raw := strings.TrimSpace(m[1])
	if raw == "" {
		return nil
	}
	parts := strings.Split(raw, ",")
	var refs []string
	for _, p := range parts {
		p = strings.TrimSpace(p)
		if p != "" {
			refs = append(refs, p)
		}
	}
	return refs
}

// NormalizeDependencies maps task dependency refs (1-based indices, "task-N", id prefixes)
// to task IDs in slice order. Mutates tasks in place.
func NormalizeDependencies(tasks []CollaborationTask) {
	if len(tasks) == 0 {
		return
	}
	byIndex := make(map[int]string)
	byIDPrefix := make(map[string]string)
	for i, t := range tasks {
		byIndex[i+1] = t.ID
		if t.ID != "" {
			byIDPrefix[strings.ToLower(t.ID)] = t.ID
			if len(t.ID) >= 8 {
				byIDPrefix[strings.ToLower(t.ID[:8])] = t.ID
			}
		}
	}
	for i := range tasks {
		if len(tasks[i].Dependencies) == 0 {
			continue
		}
		resolved := make([]string, 0, len(tasks[i].Dependencies))
		seen := make(map[string]bool)
		for _, ref := range tasks[i].Dependencies {
			id := resolveDepRef(ref, byIndex, byIDPrefix, tasks)
			if id == "" || seen[id] {
				continue
			}
			seen[id] = true
			resolved = append(resolved, id)
		}
		tasks[i].Dependencies = resolved
	}
}

func resolveDepRef(ref string, byIndex map[int]string, byIDPrefix map[string]string, tasks []CollaborationTask) string {
	ref = strings.TrimSpace(ref)
	if ref == "" {
		return ""
	}
	lower := strings.ToLower(ref)

	// 1-based index
	if n, err := strconv.Atoi(ref); err == nil && n >= 1 {
		if id, ok := byIndex[n]; ok {
			return id
		}
	}

	// task-3 or task 3
	if strings.HasPrefix(lower, "task") {
		digits := strings.TrimLeft(strings.TrimPrefix(lower, "task"), "- ")
		if n, err := strconv.Atoi(strings.TrimSpace(digits)); err == nil && n >= 1 {
			if id, ok := byIndex[n]; ok {
				return id
			}
		}
	}

	if id, ok := byIDPrefix[lower]; ok {
		return id
	}
	for _, t := range tasks {
		if strings.EqualFold(t.ID, ref) {
			return t.ID
		}
		if len(t.ID) >= 8 && strings.HasPrefix(strings.ToLower(t.ID), lower) {
			return t.ID
		}
	}
	return ""
}

// ValidateDAG returns an error if dependencies reference missing tasks or form a cycle.
func ValidateDAG(tasks []CollaborationTask) error {
	if len(tasks) == 0 {
		return nil
	}
	ids := make(map[string]int)
	for i, t := range tasks {
		if t.ID == "" {
			return fmt.Errorf("task %d has no id", i+1)
		}
		ids[t.ID] = i
	}
	for i, t := range tasks {
		for _, dep := range t.Dependencies {
			if dep == t.ID {
				return fmt.Errorf("task %d depends on itself", i+1)
			}
			if _, ok := ids[dep]; !ok {
				return fmt.Errorf("task %d references unknown dependency %q", i+1, dep)
			}
		}
	}
	// Cycle detection via DFS
	state := make(map[string]int) // 0 unvisited, 1 visiting, 2 done
	var visit func(id string) error
	visit = func(id string) error {
		switch state[id] {
		case 1:
			return fmt.Errorf("dependency cycle detected")
		case 2:
			return nil
		}
		state[id] = 1
		idx := ids[id]
		for _, dep := range tasks[idx].Dependencies {
			if err := visit(dep); err != nil {
				return err
			}
		}
		state[id] = 2
		return nil
	}
	for id := range ids {
		if state[id] == 0 {
			if err := visit(id); err != nil {
				return err
			}
		}
	}
	return nil
}

// IsTaskReady reports whether all dependencies are completed.
func IsTaskReady(task CollaborationTask, tasks []CollaborationTask) bool {
	if len(task.Dependencies) == 0 {
		return true
	}
	statusByID := make(map[string]TaskStatus, len(tasks))
	for _, t := range tasks {
		statusByID[t.ID] = t.Status
	}
	for _, depID := range task.Dependencies {
		if statusByID[depID] != TaskCompleted {
			return false
		}
	}
	return true
}

// ReadyTasks returns pending tasks that are ready and have not been prompt-dispatched.
func ReadyTasks(tasks []CollaborationTask) []CollaborationTask {
	var out []CollaborationTask
	for _, t := range tasks {
		if t.Status != TaskPending {
			continue
		}
		if t.PromptDispatched {
			continue
		}
		if !IsTaskReady(t, tasks) {
			continue
		}
		out = append(out, t)
	}
	return out
}

// BlockedBy returns titles of incomplete dependency tasks blocking this task.
func BlockedBy(task CollaborationTask, tasks []CollaborationTask) []string {
	if len(task.Dependencies) == 0 {
		return nil
	}
	byID := make(map[string]CollaborationTask, len(tasks))
	for _, t := range tasks {
		byID[t.ID] = t
	}
	var titles []string
	for _, depID := range task.Dependencies {
		dep, ok := byID[depID]
		if !ok {
			continue
		}
		if dep.Status != TaskCompleted {
			title := dep.Title
			if title == "" {
				title = depID[:8]
			}
			titles = append(titles, title)
		}
	}
	return titles
}

// TaskByID returns a pointer to the task with the given ID, or nil.
func TaskByID(tasks []CollaborationTask, id string) *CollaborationTask {
	for i := range tasks {
		if tasks[i].ID == id {
			return &tasks[i]
		}
	}
	return nil
}

// FormatDependencyHandoff builds upstream output text for task prompts.
func FormatDependencyHandoff(task CollaborationTask, tasks []CollaborationTask) string {
	if len(task.Dependencies) == 0 {
		return ""
	}
	var b strings.Builder
	b.WriteString("\n\n**Upstream completed work:**\n")
	for _, depID := range task.Dependencies {
		dep := TaskByID(tasks, depID)
		if dep == nil || dep.Status != TaskCompleted {
			continue
		}
		b.WriteString("\n- **")
		b.WriteString(dep.Title)
		b.WriteString("** (@")
		b.WriteString(dep.AssignedName)
		b.WriteString("):\n")
		out := strings.TrimSpace(dep.Output)
		if out == "" {
			out = "(no summary recorded)"
		}
		if len(out) > 2000 {
			out = out[:2000] + "\n... (truncated)"
		}
		b.WriteString(out)
		b.WriteString("\n")
	}
	return b.String()
}
