package collaboration

import (
	"path/filepath"
	"regexp"
	"strings"
)

var (
	taskFileVerbRE     = regexp.MustCompile(`(?i)\b(write|create|draft|produce|emit)\b`)
	taskFileExtRE      = regexp.MustCompile(`(?i)[\w./-]+\.(md|markdown|yaml|yml|json|txt|go|rs|ts|tsx|py)`)
	taskPathTokenRE    = regexp.MustCompile(`(?i)(?:collabs/[\w-]+/[\w./-]+|[\w][\w./-]*\.(?:md|markdown|yaml|yml|json|txt|go|rs|ts|tsx|py))`)
)

// TaskRequiresFileDeliverable is true when task text asks for a concrete file output.
func TaskRequiresFileDeliverable(t CollaborationTask) bool {
	combined := strings.TrimSpace(t.Title + " " + t.Description)
	if combined == "" {
		return false
	}
	if !taskFileVerbRE.MatchString(combined) && !strings.Contains(strings.ToLower(combined), "[file_change]") {
		return false
	}
	return taskFileExtRE.MatchString(combined) || strings.Contains(strings.ToLower(combined), "collabs/")
}

// ReferencedDeliverablePaths extracts repo-relative paths from task text.
func ReferencedDeliverablePaths(t CollaborationTask) []string {
	combined := t.Title + " " + t.Description
	seen := make(map[string]bool)
	var out []string
	for _, m := range taskPathTokenRE.FindAllString(combined, -1) {
		p := filepath.ToSlash(strings.Trim(m, "`\"' "))
		if p == "" || seen[p] {
			continue
		}
		seen[p] = true
		out = append(out, p)
	}
	return out
}

// TaskDispatchFileDeliverableNote returns extra instructions for file-shaped tasks.
func TaskDispatchFileDeliverableNote(t CollaborationTask) string {
	if !TaskRequiresFileDeliverable(t) {
		return ""
	}
	return "\n\n**Deliverable required:** Emit a canonical `[FILE_CHANGE]` block before `TASK_STATUS: completed`:\n" +
		"```\n[FILE_CHANGE]\noperation: create\npath: collabs/<id>/file.md\n```new\n<file content>\n```\n[/FILE_CHANGE]\n```\n" +
		"Conversation-only or inline `[FILE_CHANGE] path` without a hub proposal does not write to disk until approved in Pending changes."
}
