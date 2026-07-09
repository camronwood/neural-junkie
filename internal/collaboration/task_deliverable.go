package collaboration

import (
	"path/filepath"
	"regexp"
	"strings"
)

var (
	taskFileVerbRE     = regexp.MustCompile(`(?i)\b(write|create|draft|produce|emit)\b`)
	taskFileExtRE      = regexp.MustCompile(`(?i)[\w./-]+\.(md|markdown|yaml|yml|json|txt|go|rs|ts|tsx|py|html|css|js)`)
	taskPathTokenRE    = regexp.MustCompile(`(?i)(?:collabs/[\w-]+/[\w./-]+\.(?:md|markdown|yaml|yml|json|txt|go|rs|ts|tsx|py|html|css|js)|[\w][\w./-]*\.(?:md|markdown|yaml|yml|json|txt|go|rs|ts|tsx|py|html|css|js))`)
	deliverableExtRE   = regexp.MustCompile(`(?i)\.(md|markdown|yaml|yml|json|txt|go|rs|ts|tsx|py|html|css|js)$`)
)

// sanitizePathToken rejects truncated or corrupt path tokens from task titles.
func sanitizePathToken(token string) string {
	token = filepath.ToSlash(strings.Trim(token, "`\"' "))
	if token == "" {
		return ""
	}
	if strings.Contains(token, "...") || strings.Contains(token, "<|") {
		return ""
	}
	if !deliverableExtRE.MatchString(token) {
		return ""
	}
	return token
}

// TaskRequiresFileDeliverable is true when task text asks for a concrete file output.
func TaskRequiresFileDeliverable(t CollaborationTask) bool {
	combined := strings.TrimSpace(t.Title + " " + t.Description)
	if combined == "" {
		return false
	}
	lower := strings.ToLower(combined)
	// Explicit collabs/<id>/*.md paths count even without "write/create" verbs (e.g. "Document findings in collabs/.../findings.md").
	if strings.Contains(lower, "collabs/") && taskFileExtRE.MatchString(combined) {
		return true
	}
	if !taskFileVerbRE.MatchString(combined) && !strings.Contains(lower, "[file_change]") {
		return false
	}
	return taskFileExtRE.MatchString(combined) || strings.Contains(lower, "collabs/")
}

// TaskLooksLikeMarkdownDeliverable is true when the assigned deliverable is prose/docs (.md).
func TaskLooksLikeMarkdownDeliverable(t CollaborationTask) bool {
	if !TaskRequiresFileDeliverable(t) {
		return false
	}
	combined := strings.ToLower(strings.TrimSpace(t.Title + " " + t.Description))
	if strings.Contains(combined, ".md") || strings.Contains(combined, ".markdown") {
		return true
	}
	for _, p := range ReferencedDeliverablePaths(t) {
		ext := strings.ToLower(filepath.Ext(p))
		if ext == ".md" || ext == ".markdown" {
			return true
		}
	}
	return false
}

// ReferencedDeliverablePaths extracts repo-relative paths from task text.
func ReferencedDeliverablePaths(t CollaborationTask) []string {
	combined := t.Title + " " + t.Description
	seen := make(map[string]bool)
	var candidates []string
	for _, m := range taskPathTokenRE.FindAllString(combined, -1) {
		p := sanitizePathToken(m)
		if p == "" || seen[p] {
			continue
		}
		seen[p] = true
		candidates = append(candidates, p)
	}
	var collab []string
	var other []string
	for _, p := range candidates {
		if strings.HasPrefix(strings.ToLower(p), "collabs/") {
			collab = append(collab, p)
		} else {
			other = append(other, p)
		}
	}
	if len(collab) > 0 {
		return collab
	}
	return other
}

// TaskDispatchFileDeliverableNote returns extra instructions for file-shaped tasks.
func TaskDispatchFileDeliverableNote(t CollaborationTask) string {
	if !TaskRequiresFileDeliverable(t) {
		return ""
	}
	note := "\n\n**Deliverable required:** Emit a canonical `[FILE_CHANGE]` block before `TASK_STATUS: completed`:\n" +
		"```\n[FILE_CHANGE]\noperation: create\npath: collabs/<id>/file.md\n```new\n<file content>\n```\n[/FILE_CHANGE]\n```\n" +
		"Conversation-only or inline `[FILE_CHANGE] path` without a hub proposal does not write to disk until approved in Pending changes."
	if TaskLooksLikeMarkdownDeliverable(t) {
		note += "\n\n**Do not run build/deploy tooling** (docker-compose, npm, make, kubectl, etc.) unless this task text explicitly asks you to build, deploy, or execute the app. Read reference files from the project; ship the markdown via `[FILE_CHANGE]`."
	}
	lower := strings.ToLower(strings.TrimSpace(t.Title + " " + t.Description))
	if strings.Contains(lower, "findings.md") && strings.Contains(lower, "summariz") {
		note += "\n\n**Research deliverable:** Read each source file named in this task from the project workspace (MCP read tools). Write `findings.md` with concrete bullets grounded in those files — not a task list, plan recap, or guessed stack."
	}
	return note
}
