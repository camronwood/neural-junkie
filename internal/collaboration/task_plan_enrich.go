package collaboration

import (
	"fmt"
	"regexp"
	"strings"
)

var deliverableLineRE = regexp.MustCompile(`(?i)^-\s*\*\*deliverable:\*\*\s*` + "`?" + `([^` + "`" + `\n]+)` + "`?")

// mergeTaskContextIntoDescription folds sub-bullets (Deliverable, Details) into the task text
// so execution validators and file-deliverable detection see concrete paths.
func mergeTaskContextIntoDescription(desc string, context []string) string {
	desc = strings.TrimSpace(desc)
	if len(context) == 0 {
		return desc
	}
	var block []string
	if desc != "" {
		block = append(block, desc)
	}
	for _, line := range context {
		line = strings.TrimSpace(line)
		if line != "" {
			block = append(block, line)
		}
	}
	merged := strings.TrimSpace(strings.Join(block, "\n"))
	for _, line := range context {
		if m := deliverableLineRE.FindStringSubmatch(line); len(m) >= 2 {
			path := strings.Trim(m[1], "`\"' ")
			if path != "" && !strings.Contains(strings.ToLower(merged), "write ") {
				return fmt.Sprintf("Write %s — %s", path, desc)
			}
		}
	}
	return merged
}

// EnrichTasksWithPlanDeliverables copies deliverable lines from the approved plan into
// each task's description when the parser only kept the heading title.
func EnrichTasksWithPlanDeliverables(planContent string, tasks []CollaborationTask) {
	planContent = strings.TrimSpace(planContent)
	if planContent == "" || len(tasks) == 0 {
		return
	}
	sections := splitPlanTaskSections(planContent)
	for i := range tasks {
		titleKey := normalizeTaskSectionKey(tasks[i].Title)
		descKey := normalizeTaskSectionKey(tasks[i].Title + " " + tasks[i].Description)
		body, ok := sections[titleKey]
		if !ok {
			body, ok = sections[descKey]
		}
		if !ok {
			continue
		}
		if taskDeliverableScore(tasks[i]) >= taskDeliverableScore(CollaborationTask{Description: body}) {
			continue
		}
		tasks[i].Description = mergeTaskContextIntoDescription(tasks[i].Title, strings.Split(body, "\n"))
		if strings.TrimSpace(tasks[i].Title) == "" {
			tasks[i].Title = truncate(tasks[i].Description, 80)
		}
	}
}

// NormalizeTaskDeliverablePathsForSandbox flattens collabs/<id>/ paths when execution uses
// an isolated sandbox without a bound project repository.
func NormalizeTaskDeliverablePathsForSandbox(c *Collaboration, tasks []CollaborationTask) {
	if c == nil || UsesProjectCollabDir(c) || len(tasks) == 0 {
		return
	}
	prefix := ProjectCollabRelPath(c.ID) + "/"
	shortID := c.ID
	if len(shortID) > 8 {
		shortID = shortID[:8]
	}
	altPrefix := ProjectCollabsDirName + "/" + shortID + "/"
	for i := range tasks {
		tasks[i].Title = flattenSandboxDeliverablePath(tasks[i].Title, prefix, altPrefix)
		tasks[i].Description = flattenSandboxDeliverablePath(tasks[i].Description, prefix, altPrefix)
	}
}

func flattenSandboxDeliverablePath(text, prefix, altPrefix string) string {
	text = strings.ReplaceAll(text, prefix, "")
	text = strings.ReplaceAll(text, altPrefix, "")
	return strings.TrimSpace(text)
}

func splitPlanTaskSections(planContent string) map[string]string {
	lines := strings.Split(planContent, "\n")
	out := make(map[string]string)
	var currentKey string
	var buf []string
	flush := func() {
		if currentKey == "" || len(buf) == 0 {
			return
		}
		out[currentKey] = strings.TrimSpace(strings.Join(buf, "\n"))
	}
	for _, line := range lines {
		trimmed := strings.TrimSpace(line)
		if isTaskHeading(trimmed) || isMarkdownNumberedBoldTask(trimmed) {
			flush()
			buf = nil
			currentKey = normalizeTaskSectionKey(cleanTaskHeadingTitle(trimmed))
			continue
		}
		if currentKey != "" {
			if isTaskHeading(trimmed) || isTaskListLine(trimmed) {
				flush()
				buf = nil
				currentKey = normalizeTaskSectionKey(cleanTaskHeadingTitle(trimmed))
				continue
			}
			buf = append(buf, line)
		}
	}
	flush()
	return out
}

func cleanTaskHeadingTitle(line string) string {
	content := strings.TrimLeft(line, "# ")
	content = regexp.MustCompile(`\(@[a-zA-Z0-9]+(?:-[a-zA-Z0-9]+)*\)`).ReplaceAllString(content, "")
	content = mentionTokenRe.ReplaceAllString(content, "")
	content = taskNumberPrefixRe.ReplaceAllString(content, "")
	if m := markdownNumberedBoldTaskRe.FindStringSubmatch(strings.TrimSpace(content)); len(m) >= 2 {
		return strings.TrimSpace(m[1])
	}
	return strings.TrimSpace(content)
}

func normalizeTaskSectionKey(s string) string {
	s = strings.ToLower(strings.TrimSpace(s))
	s = taskDedupeNoise.ReplaceAllString(s, " ")
	return strings.TrimSpace(s)
}
