// Package runtime defines Agent Runtime v2 — an open-ended plan → discover → edit → verify → repair loop.
// The live implementation is agent.runImplementationSessionStreaming when features.agent_runtime_v2 is enabled.
package runtime

import "time"

const (
	// DefaultMaxSteps is the tool-loop guardrail when performance.agent_max_steps is unset.
	DefaultMaxSteps = 100
	// DefaultMaxRepairRounds is the verify/repair cap per file cycle in v2.
	DefaultMaxRepairRounds = 5
	// DefaultMaxFilesPerCycle is the coordinated multi-file batch size per cycle.
	DefaultMaxFilesPerCycle = 50
	// DefaultTimeout is the session wall-clock limit when performance.agent_timeout is unset.
	DefaultTimeout = 60 * time.Minute
)

// Config mirrors hub performance.agent_* settings for documentation and tests.
type Config struct {
	MaxSteps           int
	MaxRepairRounds    int
	MaxFilesPerCycle   int
	Timeout            time.Duration
	AgentRuntimeBudget int // bytes
}

// DefaultConfig returns v2 guardrails aligned with internal/agent/agent_runtime_config.go.
func DefaultConfig() Config {
	return Config{
		MaxSteps:           DefaultMaxSteps,
		MaxRepairRounds:    DefaultMaxRepairRounds,
		MaxFilesPerCycle:   DefaultMaxFilesPerCycle,
		Timeout:            DefaultTimeout,
		AgentRuntimeBudget: 0, // derived from ollama.num_ctx at runtime
	}
}
