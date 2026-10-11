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
	maxBudgetTurnLedger       = 1536
	maxBudgetDurableState     = 1536
	maxBudgetRelevantMemory   = 1536
	maxBudgetHistoryBody      = 12 * 1024
	maxBudgetWorkspaceOutline = 4 * 1024
	ideWorkspaceOutlineBytes  = 16 * 1024
)

const (
	rulesSectionStart                 = "=== USER-CONFIGURED RULES ==="
	rulesSectionEnd                   = "=== END USER-CONFIGURED RULES ==="
	contextRetrieveCapabilityMetadata = "context_retrieval_available"
	contextRetrieveToolName           = "nj_retrieve_context"
	contextBudgetStatsMetadata        = "context_budget_stats"
)

// ContextBudgetStats records truncation applied to a prompt.
type ContextBudgetStats struct {
	OriginalBytes      int      `json:"original_bytes"`
	FinalBytes         int      `json:"final_bytes"`
	Truncated          bool     `json:"truncated"`
	CompressedSections []string `json:"compressed_sections,omitempty"`
}

func stampContextBudgetStats(msg *protocol.Message, stats ContextBudgetStats) {
	if msg == nil {
		return
	}
	if msg.Metadata == nil {
		msg.Metadata = map[string]any{}
	}
	msg.Metadata[contextBudgetStatsMetadata] = stats
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
	return applyContextBudgetWithLimit(prompt, contextBudgetLimit(), maxBudgetWorkspaceOutline, "", false)
}

func applyContextBudgetForMessage(msg *protocol.Message, prompt string) (string, ContextBudgetStats) {
	limit := contextBudgetLimit()
	outline := maxBudgetWorkspaceOutline
	channelID := ""
	canRetrieve := false
	if msg != nil {
		channelID = msg.Channel
		canRetrieve, _ = msg.Metadata[contextRetrieveCapabilityMetadata].(bool)
	}
	if msg != nil && (msg.ImplementationSession() || (agentRuntimeV2ForMessage(msg) && msg.IdeEditorMode() == "agent")) {
		limit = implSessionBudgetLimit(msg)
		outline = ideWorkspaceOutlineBytes
	} else if msg != nil && msg.IdeRouteAgentType() != "" {
		limit = ideContextBudgetLimit()
		outline = ideWorkspaceOutlineBytes
	}
	if p, ok := turnContextProfileFromMetadata(msg); ok && p.Constrained && isIDEComposerTurn(msg) {
		if p.MaxPromptBytes > 0 && (limit <= 0 || p.MaxPromptBytes < limit) {
			limit = p.MaxPromptBytes
		}
		outline = maxBudgetWorkspaceOutline
	}
	prompt = cacheStableSystemOrder(prompt)
	return applyContextBudgetWithLimit(prompt, limit, outline, channelID, canRetrieve)
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
		"=== DURABLE CONVERSATION STATE ===",
		"=== TURN LEDGER (recent) ===",
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

func applyContextBudgetWithLimit(prompt string, limit, workspaceOutlineCap int, channelID string, canRetrieve bool) (string, ContextBudgetStats) {
	stats := ContextBudgetStats{OriginalBytes: len(prompt)}

	systemPart, userPart, hasSep := strings.Cut(prompt, ai.SystemPromptSeparator)
	if hasSep {
		callID := uuid.NewString()
		systemPart, rulesBlock := peelProtectedRulesSection(systemPart)
		systemPart, label := compressMarkedSection(systemPart, "=== DURABLE CONVERSATION STATE ===", channelID, callID+"-durable", maxBudgetDurableState, canRetrieve)
		if label != "" {
			stats.CompressedSections = append(stats.CompressedSections, label)
			stats.Truncated = true
		}
		systemPart, label = compressMarkedSection(systemPart, "=== TURN LEDGER (recent) ===", channelID, callID+"-ledger", maxBudgetTurnLedger, canRetrieve)
		if label != "" {
			stats.CompressedSections = append(stats.CompressedSections, label)
			stats.Truncated = true
		}
		systemPart, label = compressMarkedSection(systemPart, "=== SESSION SUMMARY ===", channelID, callID+"-summary", maxBudgetSessionSummary, canRetrieve)
		if label != "" {
			stats.CompressedSections = append(stats.CompressedSections, label)
			stats.Truncated = true
		}
		systemPart, label = compressMarkedSection(systemPart, "=== RELEVANT PAST CONTEXT ===", channelID, callID+"-memory", maxBudgetRelevantMemory, canRetrieve)
		if label != "" {
			stats.CompressedSections = append(stats.CompressedSections, label)
			stats.Truncated = true
		}
		systemPart, label = compressMarkedSection(systemPart, "=== WORKSPACE CONTEXT ===", channelID, callID+"-workspace", workspaceOutlineCap, canRetrieve)
		if label != "" {
			stats.CompressedSections = append(stats.CompressedSections, label)
			stats.Truncated = true
		}
		systemPart, label = compressMarkedSection(systemPart, "Grounding requirement:", channelID, callID+"-grounding", workspaceOutlineCap, canRetrieve)
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
		if len(systemPart) > limit/2 {
			systemPart = truncateAtSectionBoundary(systemPart, limit/2)
		}
		systemPart = rulesBlock + systemPart
		stats.Truncated = true
	}
	combined = systemPart + ai.SystemPromptSeparator + userPart
	if len(combined) > limit && len(userPart) < limit/4 {
		over := len(combined) - limit
		if over < len(systemPart) {
			systemPart, rulesBlock = peelProtectedRulesSection(systemPart)
			target := len(systemPart) - over
			if target > 0 {
				systemPart = truncateAtSectionBoundary(systemPart, target)
			}
			systemPart = rulesBlock + systemPart
			combined = systemPart + ai.SystemPromptSeparator + userPart
		}
	}
	// Protected rules are intentionally peeled before ordinary system trimming,
	// but a large per-agent rules block can itself exceed the whole budget. The
	// previous path then returned an oversized prompt unchanged. Keep both rule
	// markers and the latest user message while enforcing the configured cap.
	if len(combined) > limit {
		separatorBytes := len(ai.SystemPromptSeparator)
		if maxUser := limit / 2; len(userPart) > maxUser {
			userPart = truncateAtLineBoundary(userPart, maxUser-len("\n…(user context truncated)\n")) + "\n…(user context truncated)\n"
		}
		systemBudget := limit - separatorBytes - len(userPart)
		if systemBudget < 0 {
			systemBudget = 0
		}
		systemPart = truncateSystemPreservingRules(systemPart, systemBudget)
		combined = systemPart + ai.SystemPromptSeparator + userPart
		if len(combined) > limit {
			combined = truncateAtSectionBoundary(combined, limit)
		}
	}
	stats.FinalBytes = len(combined)
	stats.Truncated = true
	return combined, stats
}

func truncateSystemPreservingRules(system string, maxBytes int) string {
	if maxBytes <= 0 {
		return ""
	}
	rest, rules := peelProtectedRulesSection(system)
	if rules == "" {
		if len(system) <= maxBytes {
			return system
		}
		return truncateAtSectionBoundary(system, maxBytes)
	}

	rulesBudget := maxBytes / 2
	if rulesBudget < len(rulesSectionStart)+len(rulesSectionEnd)+8 {
		rulesBudget = 0
	}
	if len(rules) > rulesBudget {
		rules = truncateRulesBlock(rules, rulesBudget)
	}
	restBudget := maxBytes - len(rules)
	if restBudget < 0 {
		restBudget = 0
	}
	if len(rest) > restBudget {
		rest = truncateAtSectionBoundary(rest, restBudget)
	}
	return rules + rest
}

// truncateAtSectionBoundary cuts text at `\n\n===` / blank-line boundaries so
// durable conversation state (and ACTIVE CORRECTION lines) are never mid-cut.
func truncateAtSectionBoundary(s string, maxBytes int) string {
	if maxBytes <= 0 {
		return ""
	}
	if len(s) <= maxBytes {
		return s
	}
	// Prefer keeping a complete durable corrections section when present.
	if durable := extractMarkedSection(s, "=== DURABLE CONVERSATION STATE ==="); durable != "" &&
		strings.Contains(durable, "ACTIVE CORRECTION") {
		without := removeMarkedSection(s, "=== DURABLE CONVERSATION STATE ===")
		if len(durable) <= maxBytes {
			remain := maxBytes - len(durable)
			if remain <= 0 {
				return durable
			}
			trimmed := truncateAtSectionBoundary(without, remain)
			if trimmed == "" {
				return durable
			}
			if strings.HasPrefix(strings.TrimLeft(durable, "\n"), "===") {
				return strings.TrimRight(trimmed, "\n") + "\n\n" + strings.TrimLeft(durable, "\n")
			}
			return strings.TrimRight(trimmed, "\n") + "\n\n" + strings.TrimSpace(durable)
		}
		// Durable alone exceeds budget: keep correction lines, drop trailing filler inside section.
		return truncateDurablePreservingCorrections(durable, maxBytes)
	}
	cut := s[:maxBytes]
	if idx := strings.LastIndex(cut, "\n\n==="); idx > 0 {
		return strings.TrimRight(cut[:idx], "\n") + "\n…(context truncated)\n"
	}
	if idx := strings.LastIndex(cut, "\n\n"); idx > 0 {
		return cut[:idx] + "\n…(context truncated)\n"
	}
	if idx := strings.LastIndex(cut, "\n"); idx > 0 {
		return cut[:idx] + "\n…(context truncated)\n"
	}
	return cut + "\n…(context truncated)\n"
}

func extractMarkedSection(body, marker string) string {
	idx := strings.Index(body, marker)
	if idx < 0 {
		return ""
	}
	end := strings.Index(body[idx:], "\n\n===")
	sectionEnd := len(body)
	if end > 0 {
		sectionEnd = idx + end
	}
	// Prefer explicit end marker when present.
	if endMarker := strings.Replace(marker, "=== ", "=== END ", 1); strings.HasPrefix(marker, "=== ") {
		if eidx := strings.Index(body[idx:], endMarker); eidx > 0 {
			sectionEnd = idx + eidx + len(endMarker)
			for sectionEnd < len(body) && (body[sectionEnd] == '\n' || body[sectionEnd] == '\r') {
				sectionEnd++
			}
		}
	}
	return body[idx:sectionEnd]
}

func removeMarkedSection(body, marker string) string {
	section := extractMarkedSection(body, marker)
	if section == "" {
		return body
	}
	idx := strings.Index(body, section)
	if idx < 0 {
		return body
	}
	return body[:idx] + body[idx+len(section):]
}

func truncateDurablePreservingCorrections(section string, maxBytes int) string {
	if len(section) <= maxBytes {
		return section
	}
	lines := strings.Split(section, "\n")
	var keep []string
	var rest []string
	for _, line := range lines {
		if strings.Contains(line, "ACTIVE CORRECTION") ||
			strings.HasPrefix(strings.TrimSpace(line), "Corrected name") ||
			strings.HasPrefix(strings.TrimSpace(line), "Superseded name") ||
			strings.HasPrefix(line, "=== DURABLE") ||
			strings.HasPrefix(line, "=== END DURABLE") ||
			strings.Contains(line, "authoritative for active corrections") ||
			strings.Contains(line, "use corrected names only") {
			keep = append(keep, line)
		} else {
			rest = append(rest, line)
		}
	}
	core := strings.Join(keep, "\n")
	if len(core) >= maxBytes {
		// Still too large: keep complete correction lines from the start.
		var out []string
		n := 0
		for _, line := range keep {
			if n+len(line)+1 > maxBytes {
				break
			}
			out = append(out, line)
			n += len(line) + 1
		}
		return strings.Join(out, "\n") + "\n…(durable state truncated)\n"
	}
	remain := maxBytes - len(core) - 1
	extra := ""
	if remain > 24 {
		extraBody := strings.Join(rest, "\n")
		if len(extraBody) > remain {
			extraBody = truncateAtLineBoundary(extraBody, remain-len("\n…(durable state truncated)\n"))
			extra = "\n" + extraBody + "\n…(durable state truncated)\n"
		} else if extraBody != "" {
			extra = "\n" + extraBody
		}
	}
	return core + extra
}

func truncateAtLineBoundary(s string, maxBytes int) string {
	if maxBytes <= 0 {
		return ""
	}
	if len(s) <= maxBytes {
		return s
	}
	cut := s[:maxBytes]
	if idx := strings.LastIndex(cut, "\n"); idx > 0 {
		return cut[:idx]
	}
	return cut
}

func truncateRulesBlock(rules string, maxBytes int) string {
	if maxBytes <= 0 {
		return ""
	}
	prefix := rulesSectionStart + "\n"
	suffix := "\n…(rules truncated)\n" + rulesSectionEnd + "\n"
	if maxBytes <= len(prefix)+len(suffix) {
		return ""
	}
	body := strings.TrimPrefix(rules, prefix)
	if idx := strings.LastIndex(body, rulesSectionEnd); idx >= 0 {
		body = body[:idx]
	}
	bodyBudget := maxBytes - len(prefix) - len(suffix)
	if len(body) > bodyBudget {
		body = body[:bodyBudget]
	}
	return prefix + body + suffix
}

func compressMarkedSection(body, marker, channelID, callID string, maxBytes int, canRetrieve bool) (string, string) {
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
		var replacement string
		if marker == "=== DURABLE CONVERSATION STATE ===" && strings.Contains(section, "ACTIVE CORRECTION") {
			replacement = truncateDurablePreservingCorrections(section, maxBytes)
		} else {
			replacement = truncateAtLineBoundary(section, maxBytes-len("\n…(section truncated)\n")) + "\n…(section truncated)\n"
		}
		return body[:idx] + replacement + body[sectionEnd:], marker
	}
	r := contextcompress.CompressSectionWithRetrieval(
		contextcompress.DefaultStore(),
		marker,
		channelID,
		callID,
		section,
		maxBytes,
		opts,
		canRetrieve,
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
