package collaboration

import (
	"regexp"
	"strings"
)

var taskStatusLineRE = regexp.MustCompile(`(?im)^\s*TASK_STATUS\s*:\s*(pending|in_progress|completed|blocked)\s*$`)

// InferTaskStatusFromAgentReply extracts an explicit TASK_STATUS line or conservative phrases.
func InferTaskStatusFromAgentReply(content string) TaskStatus {
	if m := taskStatusLineRE.FindStringSubmatch(content); len(m) >= 2 {
		switch strings.ToLower(strings.TrimSpace(m[1])) {
		case string(TaskPending):
			return TaskPending
		case string(TaskInProgress):
			return TaskInProgress
		case string(TaskCompleted):
			return TaskCompleted
		case string(TaskBlocked):
			return TaskBlocked
		}
	}
	return inferTaskStatusFromContent(content)
}

func inferTaskStatusFromContent(content string) TaskStatus {
	lower := strings.ToLower(content)
	if strings.Contains(lower, "blocked") || strings.Contains(lower, "stuck") || strings.Contains(lower, "cannot proceed") {
		return TaskBlocked
	}
	if strings.Contains(lower, "completed") ||
		strings.Contains(lower, "done") ||
		strings.Contains(lower, "finished") ||
		strings.Contains(lower, "implemented") {
		return TaskCompleted
	}
	if strings.Contains(lower, "working on") || strings.Contains(lower, "in progress") || strings.Contains(lower, "started") {
		return TaskInProgress
	}
	return ""
}
