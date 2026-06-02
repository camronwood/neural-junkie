package collaboration

import (
	"regexp"
	"strings"
)

var synthesisTaskRE = regexp.MustCompile(`(?i)(compile\s+findings|from\s+the\s+above\s+tasks?|synthesize|consolidate\s+findings|aggregate\s+findings|summarize\s+findings|combine\s+findings|produce\s+a\s+markdown\s+document.*findings)`)

// InferSynthesisTaskDependencies adds dependencies on all prior tasks when a task
// clearly synthesizes upstream work but the plan omitted explicit depends: lines.
func InferSynthesisTaskDependencies(tasks []CollaborationTask) bool {
	if len(tasks) < 2 {
		return false
	}
	changed := false
	for i := range tasks {
		if len(tasks[i].Dependencies) > 0 {
			continue
		}
		text := strings.TrimSpace(tasks[i].Title + "\n" + tasks[i].Description)
		if !synthesisTaskRE.MatchString(text) {
			continue
		}
		deps := make([]string, 0, i)
		for j := 0; j < i; j++ {
			if tasks[j].ID != "" {
				deps = append(deps, tasks[j].ID)
			}
		}
		if len(deps) == 0 {
			continue
		}
		tasks[i].Dependencies = deps
		changed = true
	}
	if changed {
		NormalizeDependencies(tasks)
	}
	return changed
}
