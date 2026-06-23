package config

import "time"

const (
	defaultContextBudgetKB          = 32
	defaultIdeContextBudgetKB       = 48
	defaultImplSessionBudgetKB      = 64
	defaultAgentRuntimeBudgetKB     = 192
	defaultAgentRuntimeToolBytes    = 48000
	defaultAgentRuntimeRetrieveTurn = 12
	defaultMaxHistoryMessages       = 10
	defaultAgentMaxSteps            = 100
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
	// AgentRuntimeBudgetKB caps agent-runtime v2 prompts (default 192KB or derived from num_ctx).
	AgentRuntimeBudgetKB int `json:"agent_runtime_budget_kb,omitempty"`
	// AgentRuntimeMaxToolBytes caps compressed tool output in agent-runtime mode.
	AgentRuntimeMaxToolBytes int `json:"agent_runtime_max_tool_bytes,omitempty"`
	// AgentRuntimeMaxRetrievePerTurn raises nj_retrieve_context budget during agent loops.
	AgentRuntimeMaxRetrievePerTurn int `json:"agent_runtime_max_retrieve_per_turn,omitempty"`
	// AgentMaxSteps guardrail for open-ended agent runtime (default 100).
	AgentMaxSteps int `json:"agent_max_steps,omitempty"`
	// AgentTimeoutMinutes max wall clock for agent runtime (default 60).
	AgentTimeoutMinutes int `json:"agent_timeout_minutes,omitempty"`
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

// AgentRuntimeBudgetBytes returns the agent-runtime prompt byte cap.
func (p PerformanceConfig) AgentRuntimeBudgetBytes(numCtx int) int {
	kb := p.AgentRuntimeBudgetKB
	if kb <= 0 {
		kb = budgetKBFromNumCtx(numCtx, defaultAgentRuntimeBudgetKB)
	}
	return kb * 1024
}

// AgentRuntimeMaxToolBytesOrDefault returns tool output cap for agent-runtime mode.
func (p PerformanceConfig) AgentRuntimeMaxToolBytesOrDefault() int {
	if p.AgentRuntimeMaxToolBytes > 0 {
		return p.AgentRuntimeMaxToolBytes
	}
	return defaultAgentRuntimeToolBytes
}

// AgentRuntimeMaxRetrievePerTurnOrDefault returns retrieve cap per tool-loop turn.
func (p PerformanceConfig) AgentRuntimeMaxRetrievePerTurnOrDefault() int {
	if p.AgentRuntimeMaxRetrievePerTurn > 0 {
		return p.AgentRuntimeMaxRetrievePerTurn
	}
	return defaultAgentRuntimeRetrieveTurn
}

// AgentMaxStepsOrDefault returns the step guardrail for agent runtime v2.
func (p PerformanceConfig) AgentMaxStepsOrDefault() int {
	if p.AgentMaxSteps > 0 && p.AgentMaxSteps <= 500 {
		return p.AgentMaxSteps
	}
	return defaultAgentMaxSteps
}

// AgentTimeout returns max duration for agent runtime v2.
func (p PerformanceConfig) AgentTimeout() time.Duration {
	m := p.AgentTimeoutMinutes
	if m <= 0 {
		m = 60
	}
	if m > 180 {
		m = 180
	}
	return time.Duration(m) * time.Minute
}

// budgetKBFromNumCtx maps Ollama num_ctx to a conservative prompt budget in KB.
func budgetKBFromNumCtx(numCtx, fallbackKB int) int {
	if numCtx <= 0 {
		return fallbackKB
	}
	// ~2 bytes per token heuristic for mixed code/text prompts.
	kb := (numCtx * 2) / 1024
	if kb < 64 {
		return 64
	}
	if kb > 512 {
		return 512
	}
	return kb
}
