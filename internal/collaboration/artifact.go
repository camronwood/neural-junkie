package collaboration

import (
	"fmt"
	"regexp"
	"strings"
	"time"

	"github.com/google/uuid"
)

// UpdateArtifact applies a content edit to the collaboration's shared
// artifact, bumps the version, and records the edit in history.
func (cm *CollaborationManager) UpdateArtifact(collabID, editorID, editorName, content string) error {
	cm.mu.Lock()
	defer cm.mu.Unlock()

	c, ok := cm.collaborations[collabID]
	if !ok {
		return fmt.Errorf("collaboration %s not found", collabID)
	}
	if c.Plan == nil {
		return fmt.Errorf("no plan artifact for collaboration %s", collabID)
	}

	c.Plan.Version++
	c.Plan.Content = content
	c.Plan.UpdatedAt = time.Now()
	c.Plan.EditHistory = append(c.Plan.EditHistory, ArtifactEdit{
		EditorID:   editorID,
		EditorName: editorName,
		Content:    content,
		Version:    c.Plan.Version,
		Timestamp:  time.Now(),
	})
	c.UpdatedAt = time.Now()

	return nil
}

// GetArtifact returns the current plan artifact for a collaboration.
func (cm *CollaborationManager) GetArtifact(collabID string) (*SharedArtifact, error) {
	cm.mu.RLock()
	defer cm.mu.RUnlock()

	c, ok := cm.collaborations[collabID]
	if !ok {
		return nil, fmt.Errorf("collaboration %s not found", collabID)
	}
	return c.Plan, nil
}

// Task heading patterns used to detect structured plans in agent responses.
var (
	taskHeadingRe      = regexp.MustCompile(`(?m)^#{1,4}\s+(?:Task\s+\d+|Tasks?)`)
	planHeadingRe      = regexp.MustCompile(`(?m)^#{1,4}\s+(?:Plan|Project Plan|Implementation Plan|Proposed Plan)`)
	taskTitleHeadingRe = regexp.MustCompile(`(?i)^#{1,6}\s+Task\s+\d+`)
	taskListPrefixRe   = regexp.MustCompile(`^(?:[-*]|\d+\.)\s+`)
	taskNumberPrefixRe = regexp.MustCompile(`(?i)^Task\s+\d+[:\s-]*`)
	mentionLeadRe      = regexp.MustCompile(`^@[^\s:]+[:\s-]*`)
	mentionTokenRe     = regexp.MustCompile(`@([^\s:]+)`)
	numberedPlanStepRe          = regexp.MustCompile(`^\d+\.`)
	markdownNumberedBoldTaskRe  = regexp.MustCompile(`^\d+\.\s+\*\*(.+?)\*\*\s*(.*)$`)
	planMetadataBulletRe        = regexp.MustCompile(`(?i)^\*\*(?:dependencies|milestones|acceptance|status)\*\*`)
	assigneeParenRe             = regexp.MustCompile(`\(@([a-zA-Z0-9]+(?:-[a-zA-Z0-9]+)*)\)\s*$`)
)

// ExtractPlanFromResponse attempts to extract a structured plan from an
// agent's response text. It looks for markdown headings like "## Plan" or
// "## Tasks" and returns everything from that heading onward as the plan
// content. Returns empty string if no plan structure is detected.
func ExtractPlanFromResponse(content string) string {
	loc := planHeadingRe.FindStringIndex(content)
	if loc == nil {
		loc = taskHeadingRe.FindStringIndex(content)
	}
	if loc != nil {
		return strings.TrimSpace(content[loc[0]:])
	}
	return ExtractPlanFromTaskLists(content)
}

// ExtractPlanFromTaskLists returns a plan document when the text contains multiple
// structured task list lines (common in agent discussion replies).
func ExtractPlanFromTaskLists(content string) string {
	lines := strings.Split(content, "\n")
	var taskLines []string
	for _, line := range lines {
		trimmed := strings.TrimSpace(line)
		if isTaskListLine(trimmed) {
			taskLines = append(taskLines, trimmed)
		}
	}
	if len(taskLines) < 2 {
		return ""
	}
	return strings.TrimSpace("## Plan\n\n" + strings.Join(taskLines, "\n"))
}

// SynthesizePlanFromDiscussion builds plan content and tasks from discussion messages.
// It prefers the latest structured plan block instead of concatenating the full transcript.
func SynthesizePlanFromDiscussion(c *Collaboration) (planContent string, tasks []CollaborationTask) {
	if c == nil {
		return "", nil
	}
	disc := c.Discussion
	if disc == nil {
		disc = c.PlanningDiscussion
	}
	if disc == nil {
		return "", nil
	}
	for i := len(disc.Messages) - 1; i >= 0; i-- {
		m := disc.Messages[i]
		if m == nil || m.From.Name == "System" {
			continue
		}
		body := strings.TrimSpace(m.Content)
		if body == "" {
			continue
		}
		if extracted := ExtractPlanFromResponse(body); extracted != "" {
			planContent = extracted
			break
		}
	}
	if planContent == "" {
		var b strings.Builder
		for _, m := range disc.Messages {
			if m == nil || m.From.Name == "System" {
				continue
			}
			body := strings.TrimSpace(m.Content)
			if body == "" {
				continue
			}
			b.WriteString(body)
			b.WriteString("\n\n")
		}
		combined := strings.TrimSpace(b.String())
		if combined == "" {
			return "", nil
		}
		planContent = combined
		if len(planContent) > 16000 {
			planContent = planContent[:16000] + "\n... (truncated)"
		}
	}
	tasks = DedupeTasks(ExtractTasksFromPlan(planContent, c.Agents))
	return planContent, tasks
}

// ExtractTasksFromPlan parses a plan document and extracts individual
// tasks with their assigned agents. It recognises two formats:
//
//  1. Task list items: "- Task 1: @CodeReviewer - Review the CLI scaffold"
//  2. Task headings:   "### Task 1: Review the CLI scaffold (@CodeReviewer)"
//
// Returns a slice of CollaborationTask with IDs, descriptions, and
// the assigned agent name (caller must resolve to agent ID).
func ExtractTasksFromPlan(planContent string, agents []CollaborationAgent) []CollaborationTask {
	var tasks []CollaborationTask
	now := time.Now()

	lines := strings.Split(planContent, "\n")

	agentByName := make(map[string]CollaborationAgent)
	for _, a := range agents {
		agentByName[strings.ToLower(a.AgentName)] = a
	}

	for i := 0; i < len(lines); i++ {
		trimmed := strings.TrimSpace(lines[i])
		if trimmed == "" {
			continue
		}
		if isPlanMetadataBullet(trimmed) {
			continue
		}

		// Format: "1. **Title** - `pending` (@SoftwareArchitect)" (common in ## Tasks sections)
		if isMarkdownNumberedBoldTask(trimmed) {
			task := parseMarkdownNumberedBoldTask(trimmed, agentByName, now)
			if task != nil && !isWeakTaskFragment(task.Description) {
				deps, next := collectDependencyLines(lines, i+1)
				task.Dependencies = deps
				tasks = append(tasks, *task)
				i = skipTaskSubBullets(lines, next) - 1
			}
			continue
		}

		// Format: "- Task N: @AgentName - description", "- @AgentName: description",
		// or numbered list equivalents.
		if isTaskListLine(trimmed) {
			task := parseTaskLine(trimmed, agentByName, now)
			if task != nil && !isWeakTaskFragment(task.Description) {
				deps, next := collectDependencyLines(lines, i+1)
				task.Dependencies = deps
				tasks = append(tasks, *task)
				i = next - 1
			}
			continue
		}

		// Format: "### Task N: description (@AgentName)"
		if isTaskHeading(trimmed) {
			ctx, next := collectTaskHeadingContextWithEnd(lines, i+1)
			task := parseTaskHeading(trimmed, ctx, agentByName, now)
			if task != nil {
				deps, _ := collectDependencyLines(lines, next)
				task.Dependencies = deps
				tasks = append(tasks, *task)
				i = next - 1
			}
		}
	}

	NormalizeDependencies(tasks)
	return DedupeTasks(tasks)
}

// collectDependencyLines reads depends:/after: lines until the next task boundary.
// Returns raw refs and the index of the first line after the consumed block.
func collectDependencyLines(lines []string, start int) ([]string, int) {
	var refs []string
	i := start
	for i < len(lines) {
		trimmed := strings.TrimSpace(lines[i])
		if trimmed == "" {
			i++
			continue
		}
		if isTaskListLine(trimmed) || isTaskHeading(trimmed) {
			break
		}
		if depRefs := ParseDependencyRefs(trimmed); len(depRefs) > 0 {
			refs = append(refs, depRefs...)
			i++
			continue
		}
		break
	}
	return refs, i
}

func collectTaskHeadingContextWithEnd(lines []string, start int) ([]string, int) {
	context := make([]string, 0, 3)
	i := start
	for i < len(lines) && len(context) < 3 {
		trimmed := strings.TrimSpace(lines[i])
		if trimmed == "" {
			i++
			continue
		}
		if isTaskHeading(trimmed) || isTaskListLine(trimmed) {
			break
		}
		if len(ParseDependencyRefs(trimmed)) > 0 {
			break
		}
		context = append(context, trimmed)
		i++
	}
	return context, i
}

func isTaskHeading(line string) bool {
	return taskTitleHeadingRe.MatchString(strings.TrimSpace(line))
}

func isTaskListLine(line string) bool {
	trimmed := strings.TrimSpace(line)
	if isPlanMetadataBullet(trimmed) {
		return false
	}
	hasListPrefix := taskListPrefixRe.MatchString(trimmed)
	withoutPrefix := strings.TrimSpace(taskListPrefixRe.ReplaceAllString(trimmed, ""))
	if withoutPrefix == "" {
		return false
	}
	lower := strings.ToLower(withoutPrefix)
	if strings.HasPrefix(lower, "task ") {
		return true
	}
	// Numbered plan steps like "1. @Architect, review docs/..." are not executable tasks.
	if hasListPrefix && numberedPlanStepRe.MatchString(trimmed) {
		return false
	}
	// Milestone / prose bullets without @assignee are not tasks.
	return hasListPrefix && strings.Contains(withoutPrefix, "@")
}

func isMarkdownNumberedBoldTask(line string) bool {
	return markdownNumberedBoldTaskRe.MatchString(strings.TrimSpace(line))
}

func isPlanMetadataBullet(line string) bool {
	trimmed := strings.TrimSpace(line)
	if trimmed == "" {
		return false
	}
	rest := strings.TrimSpace(taskListPrefixRe.ReplaceAllString(trimmed, ""))
	if planMetadataBulletRe.MatchString(rest) {
		return true
	}
	lower := strings.ToLower(rest)
	switch {
	case strings.HasPrefix(lower, "assigned to:"),
		strings.HasPrefix(lower, "acceptance:"),
		strings.HasPrefix(lower, "depends:"),
		strings.HasPrefix(lower, "after:"):
		return true
	}
	return false
}

// skipTaskSubBullets advances past indented milestone lines under a numbered task.
func skipTaskSubBullets(lines []string, start int) int {
	i := start
	for i < len(lines) {
		raw := lines[i]
		if strings.TrimSpace(raw) == "" {
			i++
			continue
		}
		if isTaskHeading(raw) || isTaskListLine(strings.TrimSpace(raw)) || isMarkdownNumberedBoldTask(strings.TrimSpace(raw)) {
			break
		}
		if strings.HasPrefix(raw, " ") || strings.HasPrefix(raw, "\t") {
			i++
			continue
		}
		trimmed := strings.TrimSpace(raw)
		if isPlanMetadataBullet(trimmed) {
			i++
			continue
		}
		if taskListPrefixRe.MatchString(trimmed) && !strings.Contains(trimmed, "@") && !strings.HasPrefix(strings.ToLower(trimmed), "- task ") {
			i++
			continue
		}
		break
	}
	return i
}

func isWeakTaskFragment(desc string) bool {
	lower := strings.ToLower(strings.TrimSpace(desc))
	weak := []string{
		"task is to ",
		"should perform ",
		"review will ",
		"will ensure ",
	}
	for _, p := range weak {
		if strings.HasPrefix(lower, p) {
			return true
		}
	}
	return false
}

func parseMarkdownNumberedBoldTask(line string, agents map[string]CollaborationAgent, now time.Time) *CollaborationTask {
	m := markdownNumberedBoldTaskRe.FindStringSubmatch(strings.TrimSpace(line))
	if len(m) < 3 {
		return nil
	}
	title := strings.TrimSpace(m[1])
	tail := strings.TrimSpace(m[2])
	if title == "" {
		return nil
	}

	assignedTo := ""
	assignedName := ""
	if pm := assigneeParenRe.FindStringSubmatch(tail); len(pm) > 1 {
		name := strings.ToLower(pm[1])
		if agent, ok := agents[name]; ok {
			assignedTo = agent.AgentID
			assignedName = agent.AgentName
		}
	}
	if assignedTo == "" {
		for _, mention := range agentMentionRe.FindAllStringSubmatch(tail, -1) {
			name := strings.ToLower(mention[1])
			if agent, ok := agents[name]; ok {
				assignedTo = agent.AgentID
				assignedName = agent.AgentName
				break
			}
		}
	}

	desc := title
	if len(desc) < 12 {
		return nil
	}

	return &CollaborationTask{
		ID:           uuid.New().String(),
		Title:        truncate(desc, 80),
		Description:  desc,
		AssignedTo:   assignedTo,
		AssignedName: assignedName,
		Status:       TaskPending,
		CreatedAt:    now,
		UpdatedAt:    now,
	}
}

func collectTaskHeadingContext(lines []string, start int) []string {
	context := make([]string, 0, 3)
	for i := start; i < len(lines) && len(context) < 3; i++ {
		trimmed := strings.TrimSpace(lines[i])
		if trimmed == "" {
			continue
		}
		if isTaskHeading(trimmed) {
			break
		}
		context = append(context, trimmed)
	}
	return context
}

var agentMentionRe = regexp.MustCompile(`@([a-zA-Z0-9]+(?:-[a-zA-Z0-9]+)*)`)

func parseTaskLine(line string, agents map[string]CollaborationAgent, now time.Time) *CollaborationTask {
	mentions := agentMentionRe.FindAllStringSubmatch(line, -1)
	assignedTo := ""
	assignedName := ""
	for _, m := range mentions {
		name := strings.ToLower(m[1])
		if agent, ok := agents[name]; ok {
			assignedTo = agent.AgentID
			assignedName = agent.AgentName
			break
		}
	}

	// Strip bullet/number prefix, task numbering, and leading assignee mention.
	desc := strings.TrimSpace(taskListPrefixRe.ReplaceAllString(strings.TrimSpace(line), ""))
	desc = strings.TrimSpace(taskNumberPrefixRe.ReplaceAllString(desc, ""))
	desc = strings.TrimSpace(mentionLeadRe.ReplaceAllString(desc, ""))
	desc = mentionTokenRe.ReplaceAllString(desc, "")
	desc = strings.TrimLeft(desc, " -:–")
	desc = strings.TrimSpace(desc)

	if desc == "" {
		return nil
	}

	return &CollaborationTask{
		ID:           uuid.New().String(),
		Title:        truncate(desc, 80),
		Description:  desc,
		AssignedTo:   assignedTo,
		AssignedName: assignedName,
		Status:       TaskPending,
		CreatedAt:    now,
		UpdatedAt:    now,
	}
}

func parseTaskHeading(line string, context []string, agents map[string]CollaborationAgent, now time.Time) *CollaborationTask {
	// Remove leading '#' characters
	content := strings.TrimLeft(line, "# ")

	mentions := agentMentionRe.FindAllStringSubmatch(content, -1)
	assignedTo := ""
	assignedName := ""
	for _, m := range mentions {
		name := strings.ToLower(m[1])
		if agent, ok := agents[name]; ok {
			assignedTo = agent.AgentID
			assignedName = agent.AgentName
			break
		}
	}
	if assignedTo == "" {
		for _, ctx := range context {
			mentions = agentMentionRe.FindAllStringSubmatch(ctx, -1)
			for _, m := range mentions {
				name := strings.ToLower(m[1])
				if agent, ok := agents[name]; ok {
					assignedTo = agent.AgentID
					assignedName = agent.AgentName
					break
				}
			}
			if assignedTo != "" {
				break
			}
		}
	}

	// Clean up parenthetical agent references: "description (@Agent)"
	desc := regexp.MustCompile(`\(@[a-zA-Z0-9]+(?:-[a-zA-Z0-9]+)*\)`).ReplaceAllString(content, "")
	desc = mentionTokenRe.ReplaceAllString(desc, "")
	desc = taskNumberPrefixRe.ReplaceAllString(desc, "")
	desc = strings.TrimSpace(desc)

	if desc == "" {
		return nil
	}

	return &CollaborationTask{
		ID:           uuid.New().String(),
		Title:        truncate(desc, 80),
		Description:  desc,
		AssignedTo:   assignedTo,
		AssignedName: assignedName,
		Status:       TaskPending,
		CreatedAt:    now,
		UpdatedAt:    now,
	}
}
