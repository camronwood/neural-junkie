package hub

import (
	"strings"

	"github.com/camronwood/neural-junkie/internal/learning"
	"github.com/camronwood/neural-junkie/internal/mcp_export"
)

// GetAgentCustomRulesMarkdown returns the persisted custom-rules markdown for
// an agent, if any. Used when building Share Agent bundles.
func (h *Hub) GetAgentCustomRulesMarkdown(agentID string) string {
	h.mu.RLock()
	defer h.mu.RUnlock()
	if ag, ok := h.agents[agentID]; ok && ag != nil {
		return ag.CustomRulesMarkdown
	}
	return ""
}

// ImportShareLearnings merges learning entries from a Share Agent bundle into
// the global learning store, scoped to the newly imported agent. Entries that
// duplicate content already recorded for the agent (by content hash) are
// skipped.
func (h *Hub) ImportShareLearnings(agentID, agentName, agentType string, entries []mcp_export.LearningEntry) (added, skipped int) {
	if len(entries) == 0 {
		return 0, 0
	}
	seen := make(map[string]bool)
	for _, e := range learning.ListGlobal(agentID) {
		seen[e.ContentHash] = true
	}
	for _, le := range entries {
		content := strings.TrimSpace(le.Content)
		if content == "" {
			continue
		}
		hash := learning.ContentHash(content)
		if seen[hash] {
			skipped++
			continue
		}
		cat := learning.Category(strings.TrimSpace(le.Category))
		if cat == "" {
			cat = learning.CategoryFact
		}
		entry := learning.Entry{
			Scope:     learning.ScopeAgent,
			AgentID:   agentID,
			AgentType: agentType,
			AgentName: agentName,
			Content:   content,
			Category:  cat,
		}
		if _, err := learning.AddGlobal(entry); err != nil {
			skipped++
			continue
		}
		seen[hash] = true
		added++
	}
	return added, skipped
}
