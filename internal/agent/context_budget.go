package agent

import (
	"strings"

	"github.com/camronwood/neural-junkie/internal/ai"
)

const defaultContextBudgetBytes = 32 * 1024

const (
	maxBudgetSessionSummary = 2 * 1024
	maxBudgetHistoryBody    = 12 * 1024
	maxBudgetWorkspaceOutline = 4 * 1024
)

// ContextBudgetStats records truncation applied to a prompt.
type ContextBudgetStats struct {
	OriginalBytes int `json:"original_bytes"`
	FinalBytes    int `json:"final_bytes"`
	Truncated     bool `json:"truncated"`
}

func contextBudgetLimit() int {
	return defaultContextBudgetBytes
}

// applyContextBudget trims non-essential system sections when the prompt exceeds the budget.
// The user section after SystemPromptSeparator is never truncated.
func applyContextBudget(prompt string) (string, ContextBudgetStats) {
	stats := ContextBudgetStats{OriginalBytes: len(prompt)}
	limit := contextBudgetLimit()
	if len(prompt) <= limit {
		stats.FinalBytes = len(prompt)
		return prompt, stats
	}

	systemPart, userPart, hasSep := strings.Cut(prompt, ai.SystemPromptSeparator)
	if !hasSep {
		if len(prompt) > limit {
			stats.Truncated = true
			prompt = prompt[:limit] + "\n…(context truncated)\n"
		}
		stats.FinalBytes = len(prompt)
		return prompt, stats
	}

	systemPart = truncateMarkedSection(systemPart, "=== SESSION SUMMARY ===", maxBudgetSessionSummary)
	systemPart = truncateMarkedSection(systemPart, "=== WORKSPACE CONTEXT ===", maxBudgetWorkspaceOutline)
	systemPart = truncateMarkedSection(systemPart, "Grounding requirement:", maxBudgetWorkspaceOutline)

	combined := systemPart + ai.SystemPromptSeparator + userPart
	if len(combined) <= limit {
		stats.FinalBytes = len(combined)
		stats.Truncated = stats.FinalBytes < stats.OriginalBytes
		return combined, stats
	}

	if len(systemPart) > limit/2 {
		systemPart = systemPart[:limit/2] + "\n…(system context truncated)\n"
		stats.Truncated = true
	}
	combined = systemPart + ai.SystemPromptSeparator + userPart
	if len(combined) > limit && len(userPart) < limit/4 {
		over := len(combined) - limit
		if over < len(systemPart) {
			systemPart = systemPart[:len(systemPart)-over] + "\n…(truncated)\n"
			combined = systemPart + ai.SystemPromptSeparator + userPart
		}
	}
	stats.FinalBytes = len(combined)
	stats.Truncated = true
	return combined, stats
}

func truncateMarkedSection(body, marker string, maxBytes int) string {
	idx := strings.Index(body, marker)
	if idx < 0 {
		return body
	}
	end := strings.Index(body[idx:], "\n\n===")
	sectionEnd := len(body)
	if end > 0 {
		sectionEnd = idx + end
	}
	section := body[idx:sectionEnd]
	if len(section) <= maxBytes {
		return body
	}
	replacement := section[:maxBytes] + "\n…(section truncated)\n"
	return body[:idx] + replacement + body[sectionEnd:]
}
