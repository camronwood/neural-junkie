package collaboration

import (
	"os"
	"path/filepath"
	"regexp"
	"strings"
)

var collabPathRefPattern = regexp.MustCompile(`(?:^|[\s"'(])([./]?[a-zA-Z0-9][a-zA-Z0-9_./-]*(?:/[a-zA-Z0-9_./-]+)+)`)
var taskSingleFileRefPattern = regexp.MustCompile(`(?i)(?:^|[\s"'(,\-])([a-zA-Z0-9][a-zA-Z0-9._-]*\.(?:md|go|ts|tsx|js|json|yaml|yml|py|rs|txt))`)
var shortCollabIDPrefixRE = regexp.MustCompile(`(?i)\b([0-9a-f]{8})\b`)

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
		for _, p := range expandShortCollabPrefixPaths(text, repoRoot) {
			add(p)
		}
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

// expandShortCollabPrefixPaths maps bare 8-hex prefixes (e.g. "reviewing b222bffe HTML/CSS")
// to concrete files under collabs/<uuid>/ when that directory exists in the workspace.
func expandShortCollabPrefixPaths(text, repoRoot string) []string {
	repoRoot = strings.TrimSpace(repoRoot)
	if repoRoot == "" || strings.TrimSpace(text) == "" {
		return nil
	}
	lower := strings.ToLower(text)
	wantsReviewAssets := strings.Contains(lower, "html") ||
		strings.Contains(lower, "css") ||
		strings.Contains(lower, "xss") ||
		strings.Contains(lower, "security") ||
		strings.Contains(lower, "architecture") ||
		strings.Contains(lower, "review")
	if !wantsReviewAssets {
		return nil
	}
	collabsRoot := filepath.Join(repoRoot, ProjectCollabsDirName)
	entries, err := os.ReadDir(collabsRoot)
	if err != nil {
		return nil
	}
	var out []string
	seenPrefix := make(map[string]bool)
	for _, m := range shortCollabIDPrefixRE.FindAllStringSubmatch(text, -1) {
		if len(m) < 2 {
			continue
		}
		prefix := strings.ToLower(m[1])
		if seenPrefix[prefix] {
			continue
		}
		seenPrefix[prefix] = true
		for _, e := range entries {
			if !e.IsDir() {
				continue
			}
			name := strings.ToLower(e.Name())
			if name != prefix && !strings.HasPrefix(name, prefix+"-") {
				continue
			}
			dirRel := filepath.ToSlash(filepath.Join(ProjectCollabsDirName, e.Name()))
			dirAbs := filepath.Join(collabsRoot, e.Name())
			files, err := os.ReadDir(dirAbs)
			if err != nil {
				continue
			}
			for _, f := range files {
				if f.IsDir() {
					continue
				}
				ext := strings.ToLower(filepath.Ext(f.Name()))
				switch ext {
				case ".html", ".css", ".md", ".markdown":
					out = append(out, dirRel+"/"+f.Name())
				}
			}
		}
	}
	return out
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
