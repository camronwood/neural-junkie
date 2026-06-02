package filechange

import (
	"regexp"
	"strings"
)

// taskStatusInContentRE matches machine-readable TASK_STATUS lines agents sometimes
// embed inside [FILE_CHANGE] bodies (chat metadata, not file content).
var taskStatusInContentRE = regexp.MustCompile(`(?im)^[^\n]*\bTASK_STATUS\s*:\s*(pending|in_progress|completed|blocked)\b[^\n]*\n?`)

// SanitizeFileChangeContent removes TASK_STATUS lines from proposed file bodies.
func SanitizeFileChangeContent(content string) string {
	if strings.TrimSpace(content) == "" {
		return content
	}
	cleaned := taskStatusInContentRE.ReplaceAllString(content, "")
	cleaned = strings.TrimSpace(cleaned)
	cleaned = regexp.MustCompile(`\n{3,}`).ReplaceAllString(cleaned, "\n\n")
	return cleaned
}
