package hub

import (
	"os"
	"path/filepath"
	"sort"
	"strings"
)

// buildOutlineFileTree returns a shallow directory outline for workspace_context.
func buildOutlineFileTree(root string, maxDepth int) string {
	root = strings.TrimSpace(root)
	if root == "" || maxDepth < 1 {
		return ""
	}
	rootAbs, err := filepath.Abs(root)
	if err != nil {
		return ""
	}
	var b strings.Builder
	appendOutlineEntries(&b, rootAbs, rootAbs, 0, maxDepth)
	return b.String()
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
