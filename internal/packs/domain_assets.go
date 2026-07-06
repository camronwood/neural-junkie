package packs

import (
	"fmt"
	"os"
	"strings"
)

// ReadPackAssetMarkdown loads a markdown/text file relative to pack dir.
func ReadPackAssetMarkdown(packDir, relPath string) (string, error) {
	if strings.TrimSpace(relPath) == "" {
		return "", nil
	}
	path, err := ResolvePackRelativePath(packDir, relPath)
	if err != nil {
		return "", err
	}
	data, err := os.ReadFile(path)
	if err != nil {
		return "", fmt.Errorf("read pack asset %q: %w", relPath, err)
	}
	return string(data), nil
}

// IncidentPackContext loads severity rubric and workspace hints for IncidentManager.
func IncidentPackContext(packDir string, m *Manifest) string {
	if m == nil || packDir == "" {
		return ""
	}
	rubricRel := strings.TrimSpace(m.Assets.SeverityRubric)
	if rubricRel == "" {
		rubricRel = "assets/severity-rubric.md"
	}
	rubric, err := ReadPackAssetMarkdown(packDir, rubricRel)
	if err != nil || strings.TrimSpace(rubric) == "" {
		return ""
	}
	var b strings.Builder
	b.WriteString("=== INCIDENT PACK SEVERITY RUBRIC ===\n")
	b.WriteString(rubric)
	b.WriteString("\n\n")
	return b.String()
}
