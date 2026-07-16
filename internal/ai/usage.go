package ai

import "sync"

// InferenceUsage captures prompt/completion token and latency stats from a provider call.
type InferenceUsage struct {
	PromptTokens     int   `json:"prompt_tokens"`
	CompletionTokens int   `json:"completion_tokens"`
	TotalDurationNS  int64 `json:"total_duration_ns,omitempty"`
	PromptEvalNS     int64 `json:"prompt_eval_duration_ns,omitempty"`
	EvalDurationNS   int64 `json:"eval_duration_ns,omitempty"`
	Calls            int   `json:"calls,omitempty"`
}

// TTFTMs estimates time-to-first-token from prompt eval duration (milliseconds).
func (u InferenceUsage) TTFTMs() float64 {
	if u.PromptEvalNS <= 0 {
		return 0
	}
	return float64(u.PromptEvalNS) / 1e6
}

// TokPerS estimates generation tokens per second.
func (u InferenceUsage) TokPerS() float64 {
	if u.EvalDurationNS <= 0 || u.CompletionTokens <= 0 {
		return 0
	}
	return float64(u.CompletionTokens) / (float64(u.EvalDurationNS) / 1e9)
}

// Add merges usage from another call into the accumulator.
func (u *InferenceUsage) Add(other InferenceUsage) {
	if u == nil {
		return
	}
	u.PromptTokens += other.PromptTokens
	u.CompletionTokens += other.CompletionTokens
	u.TotalDurationNS += other.TotalDurationNS
	u.PromptEvalNS += other.PromptEvalNS
	u.EvalDurationNS += other.EvalDurationNS
	u.Calls += other.Calls
	if u.Calls == 0 && (other.PromptTokens > 0 || other.CompletionTokens > 0) {
		u.Calls = 1
	}
}

// UsageAware is implemented by providers that accumulate inference metrics across a session.
type UsageAware interface {
	ResetSessionUsage()
	TakeSessionUsage() InferenceUsage
}

// UsageAccumulator is a thread-safe helper for session usage.
type UsageAccumulator struct {
	mu   sync.Mutex
	sess InferenceUsage
	last InferenceUsage
}

func (a *UsageAccumulator) Record(u InferenceUsage) {
	if a == nil {
		return
	}
	if u.Calls == 0 && (u.PromptTokens > 0 || u.CompletionTokens > 0 || u.EvalDurationNS > 0) {
		u.Calls = 1
	}
	a.mu.Lock()
	defer a.mu.Unlock()
	a.last = u
	a.sess.Add(u)
}

func (a *UsageAccumulator) Last() InferenceUsage {
	if a == nil {
		return InferenceUsage{}
	}
	a.mu.Lock()
	defer a.mu.Unlock()
	return a.last
}

func (a *UsageAccumulator) ResetSession() {
	if a == nil {
		return
	}
	a.mu.Lock()
	defer a.mu.Unlock()
	a.sess = InferenceUsage{}
}

func (a *UsageAccumulator) TakeSession() InferenceUsage {
	if a == nil {
		return InferenceUsage{}
	}
	a.mu.Lock()
	defer a.mu.Unlock()
	out := a.sess
	a.sess = InferenceUsage{}
	return out
}

// MapUsage returns a JSON-friendly map for session outcomes (omits zeros).
func MapUsage(u InferenceUsage) map[string]interface{} {
	if u.PromptTokens == 0 && u.CompletionTokens == 0 && u.Calls == 0 {
		return nil
	}
	out := map[string]interface{}{
		"prompt_tokens":     u.PromptTokens,
		"completion_tokens": u.CompletionTokens,
		"calls":             u.Calls,
	}
	if ttft := u.TTFTMs(); ttft > 0 {
		out["ttft_ms"] = ttft
	}
	if tps := u.TokPerS(); tps > 0 {
		out["tok_per_s"] = tps
	}
	return out
}
