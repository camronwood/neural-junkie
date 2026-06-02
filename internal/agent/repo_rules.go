package agent

import (
	"os"
	"path/filepath"
	"strings"
)

const maxProjectRulesBytes = 8 * 1024

// LoadProjectRulesMarkdown reads repo-level rules files from workspace root.
func LoadProjectRulesMarkdown(workspaceRoot string) string {
	workspaceRoot = strings.TrimSpace(workspaceRoot)
	if workspaceRoot == "" {
		return ""
	}
	var parts []string
	seen := map[string]bool{}

	add := func(content, label string) {
		content = strings.TrimSpace(content)
		if content == "" || seen[content] {
			return
		}
		seen[content] = true
		parts = append(parts, "### "+label+"\n"+content)
	}

	if b, err := os.ReadFile(filepath.Join(workspaceRoot, ".neural-junkie", "rules.md")); err == nil {
		add(string(b), ".neural-junkie/rules.md")
	}
	if b, err := os.ReadFile(filepath.Join(workspaceRoot, "AGENTS.md")); err == nil {
		add(string(b), "AGENTS.md")
	}
	if b, err := os.ReadFile(filepath.Join(workspaceRoot, ".cursorrules")); err == nil {
		add(string(b), ".cursorrules")
	}
	_ = filepath.WalkDir(filepath.Join(workspaceRoot, ".cursor", "rules"), func(path string, d os.DirEntry, err error) error {
		if err != nil || d.IsDir() || !strings.HasSuffix(strings.ToLower(path), ".md") {
			return nil
		}
		b, rerr := os.ReadFile(path)
		if rerr != nil {
			return nil
		}
		rel, _ := filepath.Rel(workspaceRoot, path)
		add(string(b), filepath.ToSlash(rel))
		return nil
	})

	out := strings.Join(parts, "\n\n")
	if len(out) > maxProjectRulesBytes {
		out = out[:maxProjectRulesBytes] + "\n…(project rules truncated)\n"
	}
	return out
}
