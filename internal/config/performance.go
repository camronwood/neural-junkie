package config

const (
	defaultContextBudgetKB     = 32
	defaultIdeContextBudgetKB  = 48
	defaultImplSessionBudgetKB = 64
	defaultMaxHistoryMessages  = 10
)

// PerformanceConfig tunes prompt size and history depth (same models, less RAM/latency).
type PerformanceConfig struct {
	// ContextBudgetKB caps default prompt bytes (×1024). 0 uses default 32KB.
	ContextBudgetKB int `json:"context_budget_kb,omitempty"`
	// IdeContextBudgetKB caps IDE-routed prompts. 0 uses default 48KB.
	IdeContextBudgetKB int `json:"ide_context_budget_kb,omitempty"`
	// ImplSessionBudgetKB caps implementation-session prompts. 0 uses default 64KB.
	ImplSessionBudgetKB int `json:"impl_session_budget_kb,omitempty"`
	// MaxHistoryMessages is the tail channel history sent to the LLM. 0 uses default 10.
	MaxHistoryMessages int `json:"max_history_messages,omitempty"`
	// ContextCompressEnabled enables reversible tool/section compression (default true when unset).
	ContextCompressEnabled *bool `json:"context_compress_enabled,omitempty"`
	// ContextCompressMaxToolBytes caps compressed tool output size (default 12000).
	ContextCompressMaxToolBytes int `json:"context_compress_max_tool_bytes,omitempty"`
	// ContextCacheMaxEntries is the LRU size for compression originals (default 500).
	ContextCacheMaxEntries int `json:"context_cache_max_entries,omitempty"`
	// ContextCacheTTLMinutes is cache TTL for compression originals (default 60).
	ContextCacheTTLMinutes int `json:"context_cache_ttl_minutes,omitempty"`
	// OutputShapingEnabled trims verbose model output after read-only tool steps (default false).
	OutputShapingEnabled bool `json:"output_shaping_enabled,omitempty"`
}

func (p PerformanceConfig) contextBudgetKBOr(defaultKB int) int {
	if p.ContextBudgetKB > 0 {
		return p.ContextBudgetKB
	}
	return defaultKB
}

// ContextBudgetBytes returns the default prompt byte cap.
func (p PerformanceConfig) ContextBudgetBytes() int {
	return p.contextBudgetKBOr(defaultContextBudgetKB) * 1024
}

// IdeContextBudgetBytes returns the IDE-routed prompt byte cap.
func (p PerformanceConfig) IdeContextBudgetBytes() int {
	kb := p.IdeContextBudgetKB
	if kb <= 0 {
		kb = defaultIdeContextBudgetKB
	}
	return kb * 1024
}

// ImplSessionBudgetBytes returns the implementation-session prompt byte cap.
func (p PerformanceConfig) ImplSessionBudgetBytes() int {
	kb := p.ImplSessionBudgetKB
	if kb <= 0 {
		kb = defaultImplSessionBudgetKB
	}
	return kb * 1024
}

// MaxHistoryMessagesOrDefault returns configured history tail depth.
func (p PerformanceConfig) MaxHistoryMessagesOrDefault() int {
	if p.MaxHistoryMessages > 0 && p.MaxHistoryMessages <= 50 {
		return p.MaxHistoryMessages
	}
	return defaultMaxHistoryMessages
}
