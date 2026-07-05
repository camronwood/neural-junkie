package agent

import (
	"os"
	"path/filepath"
	"strings"

	"github.com/camronwood/neural-junkie/internal/protocol"
)

// appendCrossRepoHints adds lightweight dependency hints across scoped repositories.
func appendCrossRepoHints(prompt string, msg *protocol.Message) string {
	if msg == nil {
		return prompt
	}
	scoped := scopedWorkspacesFromMetadata(msg)
	if len(scoped) < 2 {
		return prompt
	}
	var hints []string
	for _, ref := range scoped[1:] {
		for _, line := range readCrossRepoHintLines(ref.Path) {
			hints = append(hints, line)
		}
	}
	if len(hints) == 0 {
		return prompt
	}
	var b strings.Builder
	b.WriteString(prompt)
	b.WriteString("\n\n=== CROSS_REPO_HINTS ===\n")
	b.WriteString("Lightweight cross-repository links detected in scoped repos:\n")
	for _, h := range hints {
		b.WriteString("- ")
		b.WriteString(h)
		b.WriteString("\n")
	}
	b.WriteString("=== END CROSS_REPO_HINTS ===\n")
	return b.String()
}

func readCrossRepoHintLines(repoPath string) []string {
	repoPath = strings.TrimSpace(repoPath)
	if repoPath == "" {
		return nil
	}
	name := filepath.Base(repoPath)
	var out []string
	if body, err := os.ReadFile(filepath.Join(repoPath, "go.mod")); err == nil {
		for _, line := range strings.Split(string(body), "\n") {
			line = strings.TrimSpace(line)
			if strings.HasPrefix(line, "replace ") {
				out = append(out, name+": "+line)
			}
		}
	}
	if body, err := os.ReadFile(filepath.Join(repoPath, "package.json")); err == nil {
		text := string(body)
		if strings.Contains(text, "file:") {
			out = append(out, name+": npm workspace/file: dependency present in package.json")
		}
	}
	if body, err := os.ReadFile(filepath.Join(repoPath, "docker-compose.yml")); err == nil {
		for _, line := range strings.Split(string(body), "\n") {
			if strings.Contains(strings.ToLower(line), "context:") {
				out = append(out, name+": "+strings.TrimSpace(line))
				break
			}
		}
	}
	if len(out) > 6 {
		out = out[:6]
	}
	return out
}
