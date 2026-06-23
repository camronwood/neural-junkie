package agent

import (
	"strings"

	"github.com/camronwood/neural-junkie/internal/ai"
	"github.com/camronwood/neural-junkie/internal/config"
	"github.com/camronwood/neural-junkie/internal/contextcompress"
	"github.com/camronwood/neural-junkie/internal/mcp"
	"github.com/camronwood/neural-junkie/internal/protocol"
	"github.com/google/uuid"
)

const (
	maxBudgetSessionSummary   = 2 * 1024
	maxBudgetRelevantMemory   = 1536
	maxBudgetHistoryBody      = 12 * 1024
	maxBudgetWorkspaceOutline = 4 * 1024
	ideWorkspaceOutlineBytes  = 16 * 1024
)

const (
	rulesSectionStart = "=== USER-CONFIGURED RULES ==="
	rulesSectionEnd   = "=== END USER-CONFIGURED RULES ==="
)

// ContextBudgetStats records truncation applied to a prompt.
type ContextBudgetStats struct {
	OriginalBytes      int      `json:"original_bytes"`
	FinalBytes         int      `json:"final_bytes"`
	Truncated          bool     `json:"truncated"`
	CompressedSections []string `json:"compressed_sections,omitempty"`
}

func performanceFromHub() config.PerformanceConfig {
	if cfg := mcp.AppConfig(); cfg != nil {
		return cfg.Performance
	}
	return config.PerformanceConfig{}
}

func contextBudgetLimit() int {
	return performanceFromHub().ContextBudgetBytes()
}

func ideContextBudgetLimit() int {
	return performanceFromHub().IdeContextBudgetBytes()
}

func implSessionBudgetLimit(msg *protocol.Message) int {
	perf := performanceFromHub()
	if msg != nil && agentRuntimeV2ForMessage(msg) && (msg.ImplementationSession() || msg.IdeEditorMode() == "agent") {
		return perf.AgentRuntimeBudgetBytes(ai.OllamaNumCtx())
	}
	return perf.ImplSessionBudgetBytes()
}

// applyContextBudget trims non-essential system sections when the prompt exceeds the budget.
func applyContextBudget(prompt string) (string, ContextBudgetStats) {
	return applyContextBudgetWithLimit(prompt, contextBudgetLimit(), maxBudgetWorkspaceOutline, "")
}

func applyContextBudgetForMessage(msg *protocol.Message, prompt string) (string, ContextBudgetStats) {
	limit := contextBudgetLimit()
	outline := maxBudgetWorkspaceOutline
	channelID := ""
	if msg != nil {
		channelID = msg.Channel
	}
	if msg != nil && (msg.ImplementationSession() || (agentRuntimeV2ForMessage(msg) && msg.IdeEditorMode() == "agent")) {
		limit = implSessionBudgetLimit(msg)
		outline = ideWorkspaceOutlineBytes
	} else if msg != nil && msg.IdeRouteAgentType() != "" {
		limit = ideContextBudgetLimit()
		outline = ideWorkspaceOutlineBytes
	}
	prompt = cacheStableSystemOrder(prompt)
	return applyContextBudgetWithLimit(prompt, limit, outline, channelID)
}

// cacheStableSystemOrder keeps semi-stable sections before volatile workspace blocks for prefix caching.
func cacheStableSystemOrder(prompt string) string {
	systemPart, userPart, hasSep := strings.Cut(prompt, ai.SystemPromptSeparator)
	if !hasSep {
		return prompt
	}
	systemPart = reorderSystemSections(systemPart)
	return systemPart + ai.SystemPromptSeparator + userPart
}

func reorderSystemSections(system string) string {
	markers := []string{
		"=== SESSION SUMMARY ===",
		"=== RELEVANT PAST CONTEXT ===",
		"=== WORKSPACE CONTEXT ===",
		"Grounding requirement:",
	}
	type block struct {
		marker string
		body   string
	}
	var extracted []block
	rest := system
	for _, marker := range markers {
		idx := strings.Index(rest, marker)
		if idx < 0 {
			continue
		}
		prefix := rest[:idx]
		rest = rest[idx:]
		end := strings.Index(rest[len(marker):], "\n\n===")
		sectionEnd := len(rest)
		if end >= 0 {
			sectionEnd = len(marker) + end
		} else if next := strings.Index(rest[len(marker):], "\n\nGrounding requirement:"); next >= 0 && marker != "Grounding requirement:" {
			sectionEnd = len(marker) + next
		}
		section := rest[:sectionEnd]
		rest = rest[sectionEnd:]
		extracted = append(extracted, block{marker: marker, body: section})
		rest = prefix + rest
	}
	if len(extracted) == 0 {
		return system
	}
	var b strings.Builder
	b.WriteString(strings.TrimRight(rest, "\n"))
	for _, blk := range extracted {
		if b.Len() > 0 {
			b.WriteString("\n\n")
		}
		b.WriteString(strings.TrimSpace(blk.body))
	}
	if !strings.HasSuffix(system, "\n") {
		b.WriteString("\n")
	}
	return b.String()
}

func applyContextBudgetWithLimit(prompt string, limit, workspaceOutlineCap int, channelID string) (string, ContextBudgetStats) {
	stats := ContextBudgetStats{OriginalBytes: len(prompt)}

	systemPart, userPart, hasSep := strings.Cut(prompt, ai.SystemPromptSeparator)
	if hasSep {
		callID := uuid.NewString()
		systemPart, rulesBlock := peelProtectedRulesSection(systemPart)
		systemPart, label := compressMarkedSection(systemPart, "=== SESSION SUMMARY ===", channelID, callID+"-summary", maxBudgetSessionSummary)
		if label != "" {
			stats.CompressedSections = append(stats.CompressedSections, label)
			stats.Truncated = true
		}
		systemPart, label = compressMarkedSection(systemPart, "=== RELEVANT PAST CONTEXT ===", channelID, callID+"-memory", maxBudgetRelevantMemory)
		if label != "" {
			stats.CompressedSections = append(stats.CompressedSections, label)
			stats.Truncated = true
		}
		systemPart, label = compressMarkedSection(systemPart, "=== WORKSPACE CONTEXT ===", channelID, callID+"-workspace", workspaceOutlineCap)
		if label != "" {
			stats.CompressedSections = append(stats.CompressedSections, label)
			stats.Truncated = true
		}
		systemPart, label = compressMarkedSection(systemPart, "Grounding requirement:", channelID, callID+"-grounding", workspaceOutlineCap)
		if label != "" {
			stats.CompressedSections = append(stats.CompressedSections, label)
			stats.Truncated = true
		}
		systemPart = rulesBlock + systemPart
		prompt = systemPart + ai.SystemPromptSeparator + userPart
		stats.OriginalBytes = len(prompt)
	}

	if len(prompt) <= limit {
		stats.FinalBytes = len(prompt)
		return prompt, stats
	}

	if !hasSep {
		if len(prompt) > limit {
			stats.Truncated = true
			prompt = prompt[:limit] + "\n…(context truncated)\n"
		}
		stats.FinalBytes = len(prompt)
		return prompt, stats
	}

	systemPart, userPart, _ = strings.Cut(prompt, ai.SystemPromptSeparator)
	systemPart, rulesBlock := peelProtectedRulesSection(systemPart)
	systemPart = rulesBlock + systemPart

	combined := systemPart + ai.SystemPromptSeparator + userPart
	if len(combined) <= limit {
		stats.FinalBytes = len(combined)
		if stats.FinalBytes < stats.OriginalBytes {
			stats.Truncated = true
		}
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

func compressMarkedSection(body, marker, channelID, callID string, maxBytes int) (string, string) {
	idx := strings.Index(body, marker)
	if idx < 0 {
		return body, ""
	}
	end := strings.Index(body[idx:], "\n\n===")
	sectionEnd := len(body)
	if end > 0 {
		sectionEnd = idx + end
	}
	section := body[idx:sectionEnd]
	if len(section) <= maxBytes {
		return body, ""
	}
	opts := contextcompress.RuntimeOptions()
	if !opts.Enabled {
		replacement := section[:maxBytes] + "\n…(section truncated)\n"
		return body[:idx] + replacement + body[sectionEnd:], marker
	}
	r := contextcompress.CompressSection(
		contextcompress.DefaultStore(),
		marker,
		channelID,
		callID,
		section,
		maxBytes,
		opts,
	)
	return body[:idx] + r.Text + body[sectionEnd:], marker
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
