package collaboration

import (
	"regexp"
	"strings"
)

var placeholderBracketRE = regexp.MustCompile(`\[[a-z][a-z0-9 _-]{2,}\]`)

// LooksLikePlaceholderContent reports template/stub bodies inventing unfilled sections.
// Shared by agent proposals and hub deliverable guards.
func LooksLikePlaceholderContent(content string) bool {
	content = strings.TrimSpace(content)
	if content == "" {
		return false
	}
	lower := strings.ToLower(content)
	markers := []string{
		"[insert ",
		"[todo",
		"[feature name]",
		"[brief description",
		"[step 1",
		"[explanation of",
		"[use case",
		"insert file name",
		"insert issues",
		"insert recommendations",
		"lorem ipsum",
		"--- title:",
		"# app name",
		"overview of the app",
		"feature 1",
		"feature 2",
		"feature 3",
		"achievement 1",
		"achievement 2",
		"achievement 3",
		"key achievements",
		"## features",
		"// your valid javascript",
	}
	for _, m := range markers {
		if strings.Contains(lower, m) {
			return true
		}
	}
	matches := placeholderBracketRE.FindAllString(lower, -1)
	if len(matches) > 2 && len(content) < 4000 {
		return true
	}
	return false
}
