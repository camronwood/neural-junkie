package collaboration

import (
	"os"
	"path/filepath"
	"regexp"
	"strings"
)

var collabPathRefPattern = regexp.MustCompile(`(?:^|[\s"'(])([./]?[a-zA-Z0-9][a-zA-Z0-9_./-]*(?:/[a-zA-Z0-9_./-]+)+)`)
var taskSingleFileRefPattern = regexp.MustCompile(`(?i)(?:^|[\s"'(,\-])([a-zA-Z0-9][a-zA-Z0-9._-]*\.(?:md|go|ts|tsx|js|json|yaml|yml|py|rs|txt))`)

// InferTaskContextPaths returns repo-relative paths an assignee should read for a task.
func InferTaskContextPaths(task CollaborationTask, repoRoot string) []string {
	text := strings.TrimSpace(task.Title + "\n" + task.Description)
	if text == "" {
		return nil
	}
	seen := make(map[string]bool)
	var paths []string
	add := func(p string) {
		p = normalizeInferredPathCandidate(p)
		if p == "" || seen[p] {
			return
		}
		seen[p] = true
		paths = append(paths, p)
	}
	for _, m := range collabPathRefPattern.FindAllStringSubmatch(text, -1) {
		if len(m) > 1 {
			add(m[1])
		}
	}
	for _, m := range taskSingleFileRefPattern.FindAllStringSubmatch(text, -1) {
		if len(m) > 1 {
			add(m[1])
		}
	}
	lower := strings.ToLower(text)
	if strings.Contains(lower, "resource") && strings.Contains(lower, "api") {
		add("core/resource-api")
		add("resource-api/json_endpoints")
	}
	if strings.Contains(lower, "schema") {
		add("resource-api/json_endpoints")
	}
	if strings.Contains(lower, "endpoint") && strings.Contains(lower, "json") {
		add("resource-api/json_endpoints")
	}
	repoRoot = strings.TrimSpace(repoRoot)
	if repoRoot != "" {
		var verified []string
		for _, p := range paths {
			abs := filepath.Join(repoRoot, p)
			if st, err := os.Stat(abs); err == nil {
				if st.IsDir() || strings.Contains(filepath.Base(p), ".") {
					verified = append(verified, p)
				}
			}
		}
		paths = verified
	}
	if len(paths) > 10 {
		paths = paths[:10]
	}
	return paths
}

// normalizeInferredPathCandidate trims quotes and trailing prose punctuation from path tokens.
func normalizeInferredPathCandidate(p string) string {
	p = strings.Trim(p, "`\"' ")
	p = strings.TrimPrefix(p, "./")
	for len(p) > 0 {
		last := p[len(p)-1]
		switch last {
		case ')', ']', ',', ';', ':', '!', '?':
			p = strings.TrimSpace(p[:len(p)-1])
		case '.':
			trimmed := strings.TrimSuffix(p, ".")
			if trimmed == "" {
				return p
			}
			p = trimmed
		default:
			return p
		}
	}
	return p
}

// MergeContextPaths combines path lists in order, deduplicated.
func MergeContextPaths(parts ...[]string) []string {
	seen := make(map[string]bool)
	var out []string
	for _, list := range parts {
		for _, p := range list {
			p = strings.TrimSpace(p)
			if p == "" || seen[p] {
				continue
			}
			seen[p] = true
			out = append(out, p)
		}
	}
	return out
}

// EnrichTasksWithContextPaths fills ContextPaths on tasks when missing or empty.
func EnrichTasksWithContextPaths(tasks []CollaborationTask, repoRoot string) {
	repoRoot = strings.TrimSpace(repoRoot)
	if repoRoot == "" {
		return
	}
	for i := range tasks {
		inferred := InferTaskContextPaths(tasks[i], repoRoot)
		if len(inferred) == 0 {
			continue
		}
		if tasks[i].Options == nil {
			tasks[i].Options = &TaskExecutionOptions{}
		}
		merged := append([]string{}, tasks[i].Options.ContextPaths...)
		seen := make(map[string]bool)
		for _, p := range merged {
			seen[p] = true
		}
		for _, p := range inferred {
			if !seen[p] {
				seen[p] = true
				merged = append(merged, p)
			}
		}
		tasks[i].Options.ContextPaths = merged
	}
}
