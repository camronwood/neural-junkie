package agent

import (
	"strings"

	"github.com/camronwood/neural-junkie/internal/ai"
	"github.com/camronwood/neural-junkie/internal/protocol"
)

const defaultContextBudgetBytes = 32 * 1024

const (
	maxBudgetSessionSummary   = 2 * 1024
	maxBudgetRelevantMemory   = 1536
	maxBudgetHistoryBody      = 12 * 1024
	maxBudgetWorkspaceOutline = 4 * 1024
	ideContextBudgetBytes     = 48 * 1024
	ideWorkspaceOutlineBytes  = 16 * 1024
	implSessionBudgetBytes    = 64 * 1024
)

const (
	rulesSectionStart = "=== USER-CONFIGURED RULES ==="
	rulesSectionEnd   = "=== END USER-CONFIGURED RULES ==="
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
func applyContextBudget(prompt string) (string, ContextBudgetStats) {
	return applyContextBudgetWithLimit(prompt, contextBudgetLimit(), maxBudgetWorkspaceOutline)
}

func applyContextBudgetForMessage(msg *protocol.Message, prompt string) (string, ContextBudgetStats) {
	limit := contextBudgetLimit()
	outline := maxBudgetWorkspaceOutline
	if msg != nil && msg.ImplementationSession() {
		limit = implSessionBudgetBytes
		outline = ideWorkspaceOutlineBytes
	} else if msg != nil && msg.IdeRouteAgentType() != "" {
		limit = ideContextBudgetBytes
		outline = ideWorkspaceOutlineBytes
	}
	return applyContextBudgetWithLimit(prompt, limit, outline)
}

func applyContextBudgetWithLimit(prompt string, limit, workspaceOutlineCap int) (string, ContextBudgetStats) {
	stats := ContextBudgetStats{OriginalBytes: len(prompt)}
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

	systemPart, rulesBlock := peelProtectedRulesSection(systemPart)
	systemPart = truncateMarkedSection(systemPart, "=== SESSION SUMMARY ===", maxBudgetSessionSummary)
	systemPart = truncateMarkedSection(systemPart, "=== RELEVANT PAST CONTEXT ===", maxBudgetRelevantMemory)
	systemPart = truncateMarkedSection(systemPart, "=== WORKSPACE CONTEXT ===", workspaceOutlineCap)
	systemPart = truncateMarkedSection(systemPart, "Grounding requirement:", workspaceOutlineCap)
	systemPart = rulesBlock + systemPart

	combined := systemPart + ai.SystemPromptSeparator + userPart
	if len(combined) <= limit {
		stats.FinalBytes = len(combined)
		stats.Truncated = stats.FinalBytes < stats.OriginalBytes
		return combined, stats
	}

	if len(systemPart) > limit/2 {
		systemPart, rulesBlock = peelProtectedRulesSection(systemPart)
		systemPart = systemPart[:limit/2] + "\n…(system context truncated)\n"
		systemPart = rulesBlock + systemPart
		stats.Truncated = true
	}
	combined = systemPart + ai.SystemPromptSeparator + userPart
	if len(combined) > limit && len(userPart) < limit/4 {
		over := len(combined) - limit
		if over < len(systemPart) {
			systemPart, rulesBlock = peelProtectedRulesSection(systemPart)
			if over < len(systemPart) {
				systemPart = systemPart[:len(systemPart)-over] + "\n…(truncated)\n"
			}
			systemPart = rulesBlock + systemPart
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

func peelProtectedRulesSection(body string) (rest, rules string) {
	start := strings.Index(body, rulesSectionStart)
	if start < 0 {
		return body, ""
	}
	endRel := strings.Index(body[start:], rulesSectionEnd)
	if endRel < 0 {
		return body, ""
	}
	end := start + endRel + len(rulesSectionEnd)
	for end < len(body) && (body[end] == '\n' || body[end] == '\r') {
		end++
	}
	rules = body[start:end]
	rest = body[:start] + body[end:]
	return rest, rules
}
