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
	taskHeadingRe = regexp.MustCompile(`(?m)^#{1,4}\s+(?:Task\s+\d+|Tasks?)`)
	planHeadingRe = regexp.MustCompile(`(?m)^#{1,4}\s+(?:Plan|Project Plan|Implementation Plan|Proposed Plan)`)
	taskItemRe    = regexp.MustCompile(`(?m)^[-*]\s+\*?\*?(?:Task\s+\d+[:\s]|@\w+[:\s])(.+)`)
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
	if loc == nil {
		return ""
	}
	return strings.TrimSpace(content[loc[0]:])
}

// ExtractTasksFromPlan parses a plan document and extracts individual
// tasks with their assigned agents. It recognises two formats:
//
//  1. Task list items: "- Task 1: @RustExpert - Build the CLI scaffold"
//  2. Task headings:   "### Task 1: Build the CLI scaffold (@RustExpert)"
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

	for _, line := range lines {
		trimmed := strings.TrimSpace(line)
		if trimmed == "" {
			continue
		}

		// Format: "- Task N: @AgentName - description" or "- @AgentName: description"
		if matches := taskItemRe.FindStringSubmatch(trimmed); len(matches) > 1 {
			task := parseTaskLine(trimmed, agentByName, now)
			if task != nil {
				tasks = append(tasks, *task)
			}
			continue
		}

		// Format: "### Task N: description (@AgentName)"
		if isTaskHeading(trimmed) {
			task := parseTaskHeading(trimmed, agentByName, now)
			if task != nil {
				tasks = append(tasks, *task)
			}
		}
	}

	return tasks
}

func isTaskHeading(line string) bool {
	lower := strings.ToLower(line)
	return strings.HasPrefix(lower, "#") && strings.Contains(lower, "task")
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

	// Strip the bullet, "Task N:", and agent mention to get the description
	desc := line
	desc = strings.TrimLeft(desc, "-* ")
	desc = regexp.MustCompile(`(?i)^Task\s+\d+[:\s]*`).ReplaceAllString(desc, "")
	desc = agentMentionRe.ReplaceAllString(desc, "")
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

func parseTaskHeading(line string, agents map[string]CollaborationAgent, now time.Time) *CollaborationTask {
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

	// Clean up parenthetical agent references: "description (@Agent)"
	desc := regexp.MustCompile(`\(@\w+\)`).ReplaceAllString(content, "")
	desc = agentMentionRe.ReplaceAllString(desc, "")
	desc = regexp.MustCompile(`(?i)^Task\s+\d+[:\s]*`).ReplaceAllString(desc, "")
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
