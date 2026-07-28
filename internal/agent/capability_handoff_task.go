package agent

import (
	"fmt"
	"regexp"
	"strings"
	"unicode"
)

var (
	topicMenuSplitRE = regexp.MustCompile(`(?i)\s+or\s+|,\s+`)
	bareHandoffTaskRE = regexp.MustCompile(`(?i)^(?:assist(?:_|\s+)?user|help(?:\s+me)?|explain|assist|support|continue|go|ok|okay)$`)
	assistWithMenuRE  = regexp.MustCompile(`(?i)^(?:assist(?:\s+with)?|help(?:\s+with)?|support)\b.+\bor\b`)
)

// ValidateCapabilityHandoffTask rejects greetings, topic menus, and other non-tasks
// before a temporary handoff room is opened.
func ValidateCapabilityHandoffTask(task string) error {
	task = strings.TrimSpace(task)
	if task == "" {
		return fmt.Errorf("capability handoff requires one bounded task")
	}
	if isSocialOrStatusPing(task) {
		return fmt.Errorf("capability handoff task is a greeting/status ping, not a bounded task")
	}
	if bareHandoffTaskRE.MatchString(task) {
		return fmt.Errorf("capability handoff task %q is too vague; ask one concrete question", task)
	}
	if wordCount(task) < 3 && !strings.Contains(task, "?") {
		return fmt.Errorf("capability handoff task %q is too short; ask one concrete question", task)
	}
	if looksLikeCapabilityTopicMenu(task) {
		return fmt.Errorf("capability handoff task looks like a topic menu, not one bounded task: %q", task)
	}
	return nil
}

func looksLikeCapabilityTopicMenu(task string) bool {
	lower := strings.ToLower(strings.TrimSpace(task))
	if lower == "" {
		return false
	}
	if assistWithMenuRE.MatchString(lower) {
		return true
	}
	if strings.Contains(lower, "or any other") || strings.Contains(lower, "or other") {
		return true
	}
	if !strings.Contains(lower, " or ") {
		return false
	}
	// Concrete tasks may still use "or" when naming files, errors, or quoted identifiers.
	if filePathRE.MatchString(task) || strings.Contains(task, "`") {
		return false
	}
	if hasConcreteHandoffAnchor(lower) {
		return false
	}
	parts := topicMenuSplitRE.Split(lower, -1)
	substantive := 0
	for _, part := range parts {
		part = strings.TrimSpace(part)
		if part == "" {
			continue
		}
		substantive++
	}
	return substantive >= 2
}

func hasConcreteHandoffAnchor(lower string) bool {
	for _, needle := range []string{
		"error", "stack", "traceback", "panic", "crash", "log",
		"pod/", "namespace", "http", "https://", "pr #", "issue #",
		".go", ".py", ".ts", ".rs", ".yaml", ".yml", ".json",
	} {
		if strings.Contains(lower, needle) {
			return true
		}
	}
	digitCount := 0
	for _, r := range lower {
		if unicode.IsDigit(r) {
			digitCount++
			if digitCount >= 2 {
				return true
			}
		}
	}
	return false
}

func wordCount(s string) int {
	fields := strings.Fields(s)
	return len(fields)
}
