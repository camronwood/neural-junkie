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
	subAssigneeBulletRe         = regexp.MustCompile(`^[-*]\s+\*\*@[^*]+\*\*:`)
	detailedPlanBoundaryRe      = regexp.MustCompile(`(?i)^#{1,4}\s+detailed\s+plan\b`)
	taskDependencyProseRe       = regexp.MustCompile(`(?i)^task\s+\d+\s+(?:depends\s+on|can\s+be\s+started|should\s+reference)\b`)
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
// planContentForTaskExtraction drops narrative "Detailed Plan" sections that repeat
// numbered steps as prose (they are not executable task rows).
func planContentForTaskExtraction(planContent string) string {
	lines := strings.Split(planContent, "\n")
	var kept []string
	for _, line := range lines {
		if detailedPlanBoundaryRe.MatchString(strings.TrimSpace(line)) {
			break
		}
		kept = append(kept, line)
	}
	out := strings.TrimSpace(strings.Join(kept, "\n"))
	if out == "" {
		return strings.TrimSpace(planContent)
	}
	return out
}

func ExtractTasksFromPlan(planContent string, agents []CollaborationAgent) []CollaborationTask {
	planContent = planContentForTaskExtraction(planContent)
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

		if isNumberedBoldOverviewLine(trimmed) {
			task, next := parseNumberedBoldOverviewTask(lines, i, agentByName, now)
			if task != nil && !isWeakTaskFragment(task.Description) {
				deps, end := collectDependencyLines(lines, next)
				task.Dependencies = deps
				tasks = append(tasks, *task)
				i = end - 1
			} else {
				i = next - 1
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
			if task != nil && !isWeakTaskFragment(task.Title) && !isWeakTaskFragment(task.Description) {
				deps, _ := collectDependencyLines(lines, next)
				task.Dependencies = deps
				tasks = append(tasks, *task)
				i = next - 1
			}
		}
	}

	NormalizeDependencies(tasks)
	InferSynthesisTaskDependencies(tasks)
	tasks = DedupeTasks(tasks)
	AssignRoundRobinToUnassignedTasks(tasks, agents)
	return tasks
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
	if subAssigneeBulletRe.MatchString(trimmed) {
		return false
	}
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
		if isTaskDependencyProse(withoutPrefix) {
			return false
		}
		// "Task N: @Agent - ..." rows only; dependency prose has no assignee.
		return agentMentionRe.MatchString(withoutPrefix)
	}
	// Numbered plan steps like "1. @Architect, review docs/..." are not executable tasks.
	if hasListPrefix && numberedPlanStepRe.MatchString(trimmed) {
		return false
	}
	// Milestone / prose bullets without @assignee are not tasks.
	return hasListPrefix && strings.Contains(withoutPrefix, "@")
}

func isMarkdownNumberedBoldTask(line string) bool {
	trimmed := strings.TrimSpace(line)
	if !markdownNumberedBoldTaskRe.MatchString(trimmed) {
		return false
	}
	m := markdownNumberedBoldTaskRe.FindStringSubmatch(trimmed)
	if len(m) < 3 {
		return false
	}
	tail := strings.TrimSpace(m[2])
	// Require explicit assignee on the task row (## Tasks style), not nested plan steps.
	if assigneeParenRe.MatchString(tail) {
		return true
	}
	return agentMentionRe.MatchString(tail) && strings.Contains(tail, "`pending`")
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

// isTaskDependencyProse detects plan notes like "Task 1 depends on Task 2 ..." that are not executable tasks.
func isTaskDependencyProse(line string) bool {
	trimmed := strings.TrimSpace(line)
	if trimmed == "" {
		return false
	}
	rest := strings.TrimSpace(taskListPrefixRe.ReplaceAllString(trimmed, ""))
	if taskDependencyProseRe.MatchString(rest) {
		return true
	}
	lower := strings.ToLower(rest)
	return strings.HasPrefix(lower, "depends on ") ||
		strings.HasPrefix(lower, "can be started ") ||
		strings.HasPrefix(lower, "should reference ")
}

func isWeakTaskFragment(desc string) bool {
	// Explicit file deliverables (e.g. "Document findings in collabs/<id>/findings.md")
	// are valid even when the prose sounds vague.
	if TaskRequiresFileDeliverable(CollaborationTask{Description: desc}) {
		return false
	}
	if len(ReferencedDeliverablePaths(CollaborationTask{Description: desc})) > 0 {
		return false
	}

	lower := strings.ToLower(strings.TrimSpace(desc))
	if strings.Contains(desc, "**:") || strings.HasPrefix(desc, "**") {
		return true
	}
	weak := []string{
		"task is to ",
		"should perform ",
		"review will ",
		"will ensure ",
		"specific actions",
		"example actions",
		"document findings",
		"files to review",
		"popular standards",
		"incorporate findings from",
		"reference research findings",
		"research and gather",
	}
	for _, p := range weak {
		if strings.HasPrefix(lower, p) || lower == strings.TrimSpace(p) {
			return true
		}
	}
	return false
}

func isNumberedBoldOverviewLine(line string) bool {
	if !markdownNumberedBoldTaskRe.MatchString(strings.TrimSpace(line)) {
		return false
	}
	return !isMarkdownNumberedBoldTask(line)
}

// parseNumberedBoldOverviewTask handles "1. **Title**" with "- **@Agent**: action" sub-bullets.
func parseNumberedBoldOverviewTask(lines []string, i int, agents map[string]CollaborationAgent, now time.Time) (*CollaborationTask, int) {
	trimmed := strings.TrimSpace(lines[i])
	m := markdownNumberedBoldTaskRe.FindStringSubmatch(trimmed)
	if len(m) < 2 {
		return nil, i + 1
	}
	title := strings.TrimSpace(m[1])
	if title == "" || len(title) < 4 {
		return nil, i + 1
	}

	assignedTo := ""
	assignedName := ""
	desc := title
	next := i + 1
	for j := i + 1; j < len(lines) && j < i+8; j++ {
		raw := lines[j]
		sub := strings.TrimSpace(raw)
		if sub == "" {
			next = j + 1
			continue
		}
		if isTaskHeading(sub) || isMarkdownNumberedBoldTask(sub) || isNumberedBoldOverviewLine(sub) {
			break
		}
		if isTaskListLine(sub) && !subAssigneeBulletRe.MatchString(sub) {
			break
		}
		if !strings.HasPrefix(raw, " ") && !strings.HasPrefix(raw, "\t") && j > i+1 {
			break
		}
		if subAssigneeBulletRe.MatchString(sub) {
			for _, mention := range agentMentionRe.FindAllStringSubmatch(sub, -1) {
				name := strings.ToLower(mention[1])
				if agent, ok := agents[name]; ok {
					assignedTo = agent.AgentID
					assignedName = agent.AgentName
					break
				}
			}
			if idx := strings.Index(sub, ":"); idx >= 0 {
				if tail := strings.TrimSpace(sub[idx+1:]); tail != "" {
					desc = title + " — " + tail
				}
			}
			next = j + 1
			break
		}
		next = j + 1
	}
	if assignedTo == "" {
		return nil, next
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
	}, next
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

	if assignedTo == "" {
		return nil
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
	if assignedTo == "" {
		return nil
	}
	desc = mergeTaskContextIntoDescription(desc, context)

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

// AssignRoundRobinToUnassignedTasks fills AssignedTo / AssignedName on tasks parsed
// without @participant mentions so execution routing and plan validation succeed.
func AssignRoundRobinToUnassignedTasks(tasks []CollaborationTask, agents []CollaborationAgent) {
	if len(agents) == 0 || len(tasks) == 0 {
		return
	}
	now := time.Now()
	ri := 0
	for i := range tasks {
		if strings.TrimSpace(tasks[i].AssignedTo) != "" {
			continue
		}
		ag := agents[ri%len(agents)]
		ri++
		tasks[i].AssignedTo = ag.AgentID
		tasks[i].AssignedName = ag.AgentName
		tasks[i].UpdatedAt = now
	}
}
