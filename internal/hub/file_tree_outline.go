package hub

import (
	"os"
	"path/filepath"
	"sort"
	"strings"
)

// buildOutlineFileTree returns a shallow directory outline for workspace_context.
func buildOutlineFileTree(root string, maxDepth int) string {
	return buildCollabOutlineFileTree(root, "", maxDepth)
}

// buildCollabOutlineFileTree returns a repo outline with goal-relevant paths listed first.
func buildCollabOutlineFileTree(root, goal string, maxDepth int) string {
	root = strings.TrimSpace(root)
	if root == "" || maxDepth < 1 {
		return ""
	}
	rootAbs, err := filepath.Abs(root)
	if err != nil {
		return ""
	}
	priorities := inferOutlinePriorityPaths(rootAbs, goal)
	if len(priorities) == 0 {
		var b strings.Builder
		appendOutlineEntries(&b, rootAbs, rootAbs, 0, maxDepth)
		return b.String()
	}
	var b strings.Builder
	b.WriteString("Key project paths (start here):\n")
	priorityDepth := maxDepth + 1
	if priorityDepth > 4 {
		priorityDepth = 4
	}
	for _, rel := range priorities {
		abs := filepath.Join(rootAbs, rel)
		st, err := os.Stat(abs)
		if err != nil {
			continue
		}
		b.WriteString("  ")
		b.WriteString(rel)
		if st.IsDir() {
			b.WriteString("/\n")
			appendOutlineEntries(&b, rootAbs, abs, 1, priorityDepth)
		} else {
			b.WriteString("\n")
		}
	}
	b.WriteString("\nRepository outline:\n")
	appendOutlineEntries(&b, rootAbs, rootAbs, 0, maxDepth)
	return b.String()
}

// inferOutlinePriorityPaths returns existing repo-relative paths to surface first for a collab goal.
func inferOutlinePriorityPaths(rootAbs, goal string) []string {
	goal = strings.ToLower(strings.TrimSpace(goal))
	if goal == "" {
		return nil
	}
	var candidates []string
	add := func(parts ...string) {
		for _, p := range parts {
			p = strings.Trim(p, "/")
			if p == "" {
				continue
			}
			candidates = append(candidates, p)
		}
	}
	if strings.Contains(goal, "resource") && strings.Contains(goal, "api") {
		add("resource-api/json_endpoints", "resource-api", "docs/tim", "docs")
	}
	if strings.Contains(goal, "schema") || strings.Contains(goal, "standardiz") || strings.Contains(goal, "registr") {
		add("resource-api/json_endpoints", "docs/tim")
	}
	if strings.Contains(goal, "json_endpoints") {
		add("resource-api/json_endpoints")
	}
	seen := make(map[string]bool)
	var out []string
	for _, rel := range candidates {
		if seen[rel] {
			continue
		}
		if _, err := os.Stat(filepath.Join(rootAbs, rel)); err != nil {
			continue
		}
		seen[rel] = true
		out = append(out, rel)
	}
	return out
}

// enrichSourceWorkspaceOutline adds file_tree to a collaboration workspace snapshot.
func enrichSourceWorkspaceOutline(ctx map[string]interface{}, repoRoot, goal string) {
	if ctx == nil {
		return
	}
	repoRoot = strings.TrimSpace(repoRoot)
	if repoRoot == "" {
		return
	}
	if tree, ok := ctx["file_tree"].(string); ok && strings.TrimSpace(tree) != "" {
		return
	}
	tree := buildCollabOutlineFileTree(repoRoot, goal, 3)
	if tree == "" {
		tree = ".\n"
	}
	ctx["file_tree"] = tree
}

func appendOutlineEntries(b *strings.Builder, rootAbs, dir string, depth, maxDepth int) {
	if depth >= maxDepth {
		return
	}
	entries, err := os.ReadDir(dir)
	if err != nil {
		return
	}
	sort.Slice(entries, func(i, j int) bool {
		if entries[i].IsDir() != entries[j].IsDir() {
			return entries[i].IsDir()
		}
		return entries[i].Name() < entries[j].Name()
	})
	indent := strings.Repeat("  ", depth)
	for _, e := range entries {
		name := e.Name()
		if strings.HasPrefix(name, ".") && name != "." {
			continue
		}
		if e.IsDir() {
			b.WriteString(indent)
			b.WriteString(name)
			b.WriteString("/\n")
			if depth+1 < maxDepth {
				appendOutlineEntries(b, rootAbs, filepath.Join(dir, name), depth+1, maxDepth)
			}
		} else if depth+1 >= maxDepth-1 {
			b.WriteString(indent)
			b.WriteString(name)
			b.WriteString("\n")
		}
	}
}
