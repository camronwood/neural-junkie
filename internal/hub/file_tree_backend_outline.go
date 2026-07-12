package hub

import (
	"context"
	"fmt"
	"sort"
	"strings"

	"github.com/camronwood/neural-junkie/internal/workspacebackend"
)

func buildCollabOutlineViaBackend(b workspacebackend.Backend, goal string, maxDepth int) string {
	if b == nil || maxDepth < 1 {
		return ""
	}
	ctx := context.Background()
	priorities := inferOutlinePriorityPathsBackend(ctx, b, goal)
	if len(priorities) == 0 {
		var out strings.Builder
		appendBackendOutlineEntries(&out, b, ".", 0, maxDepth)
		return out.String()
	}
	var out strings.Builder
	out.WriteString("Key project paths (start here):\n")
	priorityDepth := maxDepth + 1
	if priorityDepth > 4 {
		priorityDepth = 4
	}
	for _, rel := range priorities {
		entries, err := b.ReadDir(ctx, rel)
		if err != nil || len(entries) == 0 {
			continue
		}
		out.WriteString("  ")
		out.WriteString(rel)
		out.WriteString("/\n")
		appendBackendOutlineEntries(&out, b, rel, 1, priorityDepth)
	}
	out.WriteString("\nRepository outline:\n")
	appendBackendOutlineEntries(&out, b, ".", 0, maxDepth)
	return out.String()
}

func inferOutlinePriorityPathsBackend(ctx context.Context, b workspacebackend.Backend, goal string) []string {
	goal = strings.ToLower(strings.TrimSpace(goal))
	if goal == "" || b == nil {
		return nil
	}
	var candidates []string
	add := func(parts ...string) {
		for _, p := range parts {
			p = strings.Trim(p, "/")
			if p != "" {
				candidates = append(candidates, p)
			}
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
		if _, err := b.Stat(ctx, rel); err != nil {
			continue
		}
		seen[rel] = true
		out = append(out, rel)
	}
	return out
}

func appendBackendOutlineEntries(b *strings.Builder, backend workspacebackend.Backend, rel string, depth, maxDepth int) {
	if depth >= maxDepth {
		return
	}
	ctx := context.Background()
	entries, err := backend.ReadDir(ctx, strings.TrimPrefix(rel, "/"))
	if err != nil {
		return
	}
	sort.Slice(entries, func(i, j int) bool {
		if entries[i].IsDir != entries[j].IsDir {
			return entries[i].IsDir
		}
		return entries[i].Name < entries[j].Name
	})
	indent := strings.Repeat("  ", depth)
	for _, e := range entries {
		name := e.Name
		if strings.HasPrefix(name, ".") && name != "." {
			continue
		}
		if e.IsDir && workspacebackend.IsWalkIgnoreDir(name) {
			continue
		}
		if e.IsDir {
			b.WriteString(indent)
			b.WriteString(name)
			b.WriteString("/\n")
			child := e.Path
			if child == "" {
				child = strings.TrimSuffix(rel, "/") + "/" + name
			}
			if depth+1 < maxDepth {
				appendBackendOutlineEntries(b, backend, child, depth+1, maxDepth)
			}
		} else if depth+1 >= maxDepth-1 {
			b.WriteString(indent)
			b.WriteString(name)
			b.WriteString("\n")
		}
	}
}

func buildWorktreeOutlineViaBackend(b workspacebackend.Backend, collabID string, maxDepth int) string {
	rel := fmt.Sprintf("collabs/worktrees/%s", strings.TrimSpace(collabID))
	ctx := context.Background()
	if _, err := b.Stat(ctx, rel); err != nil {
		rel = "."
	}
	var out strings.Builder
	appendBackendOutlineEntries(&out, b, rel, 0, maxDepth)
	if out.Len() == 0 {
		return ".\n"
	}
	return out.String()
}
