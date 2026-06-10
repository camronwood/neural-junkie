package collaboration

import (
	"regexp"
	"strings"
)

var taskDedupeNoise = regexp.MustCompile(`[^a-z0-9]+`)

// DedupeTasks removes near-duplicate tasks (same assignee + similar title).
func DedupeTasks(tasks []CollaborationTask) []CollaborationTask {
	if len(tasks) < 2 {
		return tasks
	}
	seen := make(map[string]bool)
	out := make([]CollaborationTask, 0, len(tasks))
	for _, t := range tasks {
		key := taskDedupeKey(t)
		if seen[key] {
			continue
		}
		seen[key] = true
		out = append(out, t)
	}
	return out
}

func taskDedupeKey(t CollaborationTask) string {
	title := strings.ToLower(strings.TrimSpace(t.Title))
	title = taskDedupeNoise.ReplaceAllString(title, " ")
	title = strings.TrimSpace(title)
	assignee := strings.ToLower(strings.TrimSpace(t.AssignedTo))
	if assignee == "" {
		assignee = strings.ToLower(strings.TrimSpace(t.AssignedName))
	}
	if paths := ReferencedDeliverablePaths(t); len(paths) > 0 {
		return assignee + "|" + paths[0]
	}
	if len(title) > 72 {
		title = title[:72]
	}
	return assignee + "|" + title
}
