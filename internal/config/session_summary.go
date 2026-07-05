package config

import "time"

const defaultSessionSummaryTimeout = 90 * time.Second

// SessionSummaryModel returns the Ollama tag for hub session summaries.
func SessionSummaryModel() string {
	return AppConfig().ResolvedSessionSummary().ModelOrDefault()
}

// SessionSummaryTimeout returns the max wait for an async session summary LLM call.
func SessionSummaryTimeout() time.Duration {
	sec := AppConfig().ResolvedSessionSummary().TimeoutOrDefault()
	return time.Duration(sec) * time.Second
}
