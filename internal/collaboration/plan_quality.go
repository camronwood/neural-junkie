package collaboration

import (
	"strings"
)

const minValidPlanChars = 80

// ValidatePlanContent reports whether plan markdown is suitable for approval/synthesis.
func ValidatePlanContent(content string, agents []CollaborationAgent) (bool, string) {
	content = strings.TrimSpace(content)
	if len(content) < minValidPlanChars {
		return false, "plan too short"
	}
	if planHasCorruptTokens(content) {
		return false, "plan contains truncated or redacted path tokens"
	}
	if planHasRepeatedIdenticalLines(content) {
		return false, "plan contains repeated identical task lines"
	}
	tasks := ExtractTasksFromPlan(content, agents)
	if len(tasks) == 0 {
		return false, "no parseable tasks"
	}
	weak := 0
	for _, t := range tasks {
		if !taskPassesExecutionQuality(t) {
			weak++
		}
	}
	if weak*2 >= len(tasks) {
		return false, "too many vague or malformed task rows"
	}
	if deliverableAssigneeConflicts(tasks) > 0 {
		return false, "duplicate deliverable paths with conflicting assignees"
	}
	return true, ""
}

// ScorePlanContent ranks candidate plans for synthesis (higher is better, -1 invalid).
func ScorePlanContent(content string, agents []CollaborationAgent) int {
	ok, _ := ValidatePlanContent(content, agents)
	if !ok {
		return -1
	}
	tasks := ExtractTasksFromPlan(content, agents)
	score := len(tasks) * 10
	for _, t := range tasks {
		score += taskDeliverableScore(t)
		if len(t.Dependencies) > 0 {
			score += 3
		}
	}
	if strings.Contains(content, "depends:") || strings.Contains(strings.ToLower(content), "depends on task") {
		score += 5
	}
	return score
}

func planHasCorruptTokens(content string) bool {
	lower := strings.ToLower(content)
	if strings.Contains(content, "...") && taskFileExtRE.MatchString(content) {
		// Truncated path mid-token (e.g. collabs/b2... or index....)
		for _, m := range taskPathTokenRE.FindAllString(content, -1) {
			if strings.HasSuffix(m, "...") || strings.Contains(m, "...") {
				return true
			}
		}
	}
	if strings.Contains(content, "<|") || strings.Contains(lower, "redacted") {
		return true
	}
	return false
}

func planHasRepeatedIdenticalLines(content string) bool {
	seen := make(map[string]int)
	for _, line := range strings.Split(content, "\n") {
		trimmed := strings.TrimSpace(line)
		if !isTaskListLine(trimmed) {
			continue
		}
		key := strings.ToLower(trimmed)
		seen[key]++
		if seen[key] >= 2 {
			return true
		}
	}
	return false
}

func deliverableAssigneeConflicts(tasks []CollaborationTask) int {
	type owner struct {
		assignee string
		score    int
	}
	byPath := make(map[string]owner)
	conflicts := 0
	for _, t := range tasks {
		paths := ReferencedDeliverablePaths(t)
		if len(paths) == 0 {
			continue
		}
		primary := paths[0]
		assignee := strings.ToLower(strings.TrimSpace(t.AssignedName))
		if assignee == "" {
			assignee = t.AssignedTo
		}
		sc := taskDeliverableScore(t)
		if prev, ok := byPath[primary]; ok {
			if prev.assignee != assignee {
				conflicts++
			} else if sc > prev.score {
				byPath[primary] = owner{assignee, sc}
			}
		} else {
			byPath[primary] = owner{assignee, sc}
		}
	}
	return conflicts
}
