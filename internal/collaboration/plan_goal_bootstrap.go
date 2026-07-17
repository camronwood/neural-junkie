package collaboration

import (
	"fmt"
	"log"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/google/uuid"
)

var (
	goalFileAssigneeRe  = regexp.MustCompile(`(?i)([a-z0-9][a-z0-9_./-]*\.md)\s*\(@([a-zA-Z][a-zA-Z0-9]*)\)`)
	goalInlineTaskRe    = regexp.MustCompile(`(?i)Task\s+\d+\s*:?\s*@[^\n;]+`)
	goalExactlyNTasksRe = regexp.MustCompile(`(?i)exactly\s+(\d+|one|two|three|four|five|six|seven|eight|nine|ten)\s+(?:[a-z-]+\s+)*tasks?`)
)

var goalExactCountWords = map[string]int{
	"one": 1, "two": 2, "three": 3, "four": 4, "five": 5,
	"six": 6, "seven": 7, "eight": 8, "nine": 9, "ten": 10,
}

func parseGoalExactTaskCount(raw string) (int, bool) {
	raw = strings.ToLower(strings.TrimSpace(raw))
	if n, err := strconv.Atoi(raw); err == nil {
		return n, true
	}
	n, ok := goalExactCountWords[raw]
	return n, ok
}

// goalPinsExactTaskList is true when the /collaborate goal pins an explicit exact task list
// that should replace freestyle discussion plans instead of merging with them.
func goalPinsExactTaskList(goal string, goalTaskCount int) bool {
	if goalTaskCount <= 0 {
		return false
	}
	goalLower := strings.ToLower(strings.TrimSpace(goal))
	if goalTaskCount == 1 && (strings.Contains(goalLower, "plan one task") || strings.Contains(goalLower, "this exact line")) {
		return true
	}
	if strings.Contains(goalLower, "using these exact lines") ||
		strings.Contains(goalLower, "use exactly:") ||
		strings.Contains(goalLower, "plan exactly:") {
		return true
	}
	if m := goalExactlyNTasksRe.FindStringSubmatch(goalLower); len(m) == 2 {
		if n, ok := parseGoalExactTaskCount(m[1]); ok && n == goalTaskCount {
			return true
		}
	}
	return false
}

func substituteCollabIDPlaceholders(text, collabID string) string {
	text = strings.ReplaceAll(text, "<collab-id>", collabID)
	text = strings.ReplaceAll(text, "<id>", collabID)
	return text
}

func extractInlineGoalTasks(goal string, agents []CollaborationAgent) []CollaborationTask {
	var merged []CollaborationTask
	for _, segment := range splitCompoundTaskLine(goal) {
		segment = strings.TrimSpace(segment)
		if segment == "" {
			continue
		}
		if !strings.HasPrefix(strings.ToLower(segment), "task ") && !strings.HasPrefix(segment, "-") {
			continue
		}
		if tasks := ExtractTasksFromPlan("## Plan\n\n"+segment, agents); len(tasks) > 0 {
			merged = append(merged, tasks...)
		}
	}
	if len(merged) > 0 {
		return merged
	}
	matches := goalInlineTaskRe.FindAllString(goal, -1)
	if len(matches) == 0 {
		return nil
	}
	for _, m := range matches {
		for _, segment := range splitCompoundTaskLine(m) {
			if tasks := ExtractTasksFromPlan("## Plan\n\n"+strings.TrimSpace(segment), agents); len(tasks) > 0 {
				merged = append(merged, tasks...)
			}
		}
	}
	for _, m := range goalUnassignedDeliverableRe.FindAllString(goal, -1) {
		if tasks := ExtractTasksFromPlan("## Plan\n\n"+strings.TrimSpace(m), agents); len(tasks) > 0 {
			merged = append(merged, tasks...)
		}
	}
	return merged
}

func splitGoalClauses(goal string) []string {
	parts := strings.Split(goal, ";")
	var out []string
	for _, p := range parts {
		if s := strings.TrimSpace(p); s != "" {
			out = append(out, s)
		}
	}
	return out
}

// ExtractTasksFromCollaborationGoal parses executable tasks from the /collaborate goal
// when agents did not produce a structured plan during discussion.
func ExtractTasksFromCollaborationGoal(description, collabID string, agents []CollaborationAgent) []CollaborationTask {
	goal := substituteCollabIDPlaceholders(strings.TrimSpace(description), collabID)
	if goal == "" {
		return nil
	}
	if inline := extractInlineGoalTasks(goal, agents); len(inline) > 0 {
		return DedupeTasks(inline)
	}
	if strings.Contains(goal, ";") {
		var merged []CollaborationTask
		for _, clause := range splitGoalClauses(goal) {
			if tasks := ExtractTasksFromPlan("## Plan\n\n"+clause, agents); len(tasks) > 0 {
				merged = append(merged, tasks...)
			}
		}
		if len(merged) > 0 {
			return DedupeTasks(merged)
		}
	}
	if tasks := ExtractTasksFromPlan("## Plan\n\n"+goal, agents); len(tasks) > 0 {
		return DedupeTasks(tasks)
	}
	for _, clause := range splitGoalClauses(goal) {
		if tasks := extractGoalFileAssigneeTasks(clause, collabID, agents); len(tasks) > 0 {
			return DedupeTasks(tasks)
		}
	}
	return extractGoalFileAssigneeTasks(goal, collabID, agents)
}

func extractGoalFileAssigneeTasks(text, collabID string, agents []CollaborationAgent) []CollaborationTask {
	agentByName := make(map[string]CollaborationAgent)
	for _, a := range agents {
		agentByName[strings.ToLower(a.AgentName)] = a
	}
	rel := ProjectCollabRelPath(collabID)
	now := time.Now()
	var tasks []CollaborationTask
	for _, m := range goalFileAssigneeRe.FindAllStringSubmatch(text, -1) {
		file := strings.TrimSpace(m[1])
		agentName := strings.TrimSpace(m[2])
		agent, ok := agentByName[strings.ToLower(agentName)]
		if !ok {
			continue
		}
		path := file
		if rel != "" && !strings.Contains(file, "/") {
			path = rel + "/" + file
		}
		desc := "Write " + path
		tasks = append(tasks, CollaborationTask{
			ID:           uuid.New().String(),
			Title:        truncate(desc, 80),
			Description:  desc,
			AssignedTo:   agent.AgentID,
			AssignedName: agent.AgentName,
			Status:       TaskPending,
			CreatedAt:    now,
			UpdatedAt:    now,
		})
	}
	return tasks
}

func formatPlanContentFromTasks(tasks []CollaborationTask) string {
	var lines []string
	for i, t := range tasks {
		name := strings.TrimSpace(t.AssignedName)
		if name == "" {
			name = "Agent"
		}
		lines = append(lines, fmt.Sprintf("Task %d: @%s - %s", i+1, name, t.Description))
	}
	return "## Plan\n\n" + strings.Join(lines, "\n")
}

// ensurePlanTasksFromGoalLocked fills plan/tasks from the collaboration goal when
// discussion synthesis produced no executable rows. Caller must hold cm.mu.
func (cm *CollaborationManager) ensurePlanTasksFromGoalLocked(c *Collaboration) {
	if c == nil {
		return
	}
	goalTasks := ExtractTasksFromCollaborationGoal(c.Description, c.ID, c.Agents)
	if len(goalTasks) == 0 {
		return
	}
	if goalPinsExactTaskList(c.Description, len(goalTasks)) {
		if c.Plan == nil {
			c.Plan = &SharedArtifact{}
		}
		planContent := formatPlanContentFromTasks(goalTasks)
		c.Plan.Content = planContent
		c.Plan.Version++
		c.Plan.UpdatedAt = time.Now()
		c.Plan.Status = ArtifactProposed
		if len(goalTasks) > maxTasksLimit() {
			goalTasks = goalTasks[:maxTasksLimit()]
		}
		c.Tasks = goalTasks
		c.UpdatedAt = time.Now()
		log.Printf("[CollaborationManager] Applied pinned goal task list (%d) for %s", len(c.Tasks), c.ID[:8])
		return
	}
	if c.Plan == nil {
		c.Plan = &SharedArtifact{}
	}
	existing := ExtractTasksFromPlan(strings.TrimSpace(c.Plan.Content), c.Agents)
	merged := mergeGoalTasksMissingDeliverables(existing, goalTasks)
	if len(merged) == 0 {
		return
	}
	planContent := formatPlanContentFromTasks(merged)
	if strings.TrimSpace(c.Plan.Content) == planContent {
		if len(c.Tasks) == len(merged) {
			return
		}
	} else if len(existing) > 0 && len(merged) == len(existing) && strings.Contains(strings.ToLower(c.Plan.Content), "findings.md") {
		return
	}
	c.Plan.Content = planContent
	c.Plan.Version++
	c.Plan.UpdatedAt = time.Now()
	c.Plan.Status = ArtifactProposed
	if len(merged) > maxTasksLimit() {
		merged = merged[:maxTasksLimit()]
	}
	c.Tasks = merged
	c.UpdatedAt = time.Now()
	log.Printf("[CollaborationManager] Bootstrapped %d tasks from goal for %s", len(c.Tasks), c.ID[:8])
}

func mergeGoalTasksMissingDeliverables(existing, goal []CollaborationTask) []CollaborationTask {
	if len(goal) == 0 {
		return existing
	}
	havePath := make(map[string]bool)
	for _, t := range existing {
		for _, p := range ReferencedDeliverablePaths(t) {
			havePath[strings.ToLower(p)] = true
		}
		combined := strings.ToLower(t.Title + " " + t.Description)
		if strings.Contains(combined, "findings.md") {
			havePath["findings.md"] = true
		}
	}
	out := append([]CollaborationTask(nil), existing...)
	for _, gt := range goal {
		paths := ReferencedDeliverablePaths(gt)
		missing := false
		if len(paths) == 0 {
			combined := strings.ToLower(gt.Title + " " + gt.Description)
			if strings.Contains(combined, "findings.md") && !havePath["findings.md"] {
				missing = true
			}
		} else {
			for _, p := range paths {
				key := strings.ToLower(p)
				if !havePath[key] {
					missing = true
					break
				}
			}
		}
		if missing {
			out = append(out, gt)
			for _, p := range paths {
				havePath[strings.ToLower(p)] = true
			}
			if strings.Contains(strings.ToLower(gt.Title+" "+gt.Description), "findings.md") {
				havePath["findings.md"] = true
			}
		}
	}
	return DedupeTasks(out)
}
