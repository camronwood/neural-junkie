package learning

import (
	"regexp"
	"strings"
)

var learningTriggerPatterns = []*regexp.Regexp{
	regexp.MustCompile(`(?i)\bremember\s+that\b`),
	regexp.MustCompile(`(?i)\bremember\s+this\b`),
	regexp.MustCompile(`(?i)\bdon'?t\s+forget\b`),
	regexp.MustCompile(`(?i)\bi\s+prefer\b`),
	regexp.MustCompile(`(?i)\balways\s+use\b`),
	regexp.MustCompile(`(?i)\bnever\s+use\b`),
}

// ExtractDraftFromMessage returns candidate learning text after a trigger phrase, or "".
func ExtractDraftFromMessage(content string) string {
	content = strings.TrimSpace(content)
	if content == "" {
		return ""
	}
	lower := strings.ToLower(content)
	for _, pat := range learningTriggerPatterns {
		loc := pat.FindStringIndex(lower)
		if loc == nil {
			continue
		}
		rest := strings.TrimSpace(content[loc[1]:])
		rest = strings.TrimPrefix(rest, ":")
		rest = strings.TrimSpace(rest)
		if rest != "" {
			return rest
		}
		return content
	}
	return ""
}

// HasLearningTrigger reports whether content looks like a remember/preference phrase.
func HasLearningTrigger(content string) bool {
	return ExtractDraftFromMessage(content) != "" || strings.HasPrefix(strings.TrimSpace(strings.ToLower(content)), "remember ")
}
