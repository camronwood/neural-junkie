package config

import (
	"os"
	"strings"
	"time"
)

const defaultSessionSummaryTimeout = 90 * time.Second

// SessionSummaryModel returns the Ollama tag for hub session summaries.
// Override with NJ_SESSION_SUMMARY_MODEL.
func SessionSummaryModel() string {
	if v := strings.TrimSpace(os.Getenv("NJ_SESSION_SUMMARY_MODEL")); v != "" {
		return v
	}
	return SessionSummaryOllamaModel
}

// SessionSummaryTimeout returns the max wait for an async session summary LLM call.
// Override with NJ_SESSION_SUMMARY_TIMEOUT (Go duration string, e.g. "90s").
func SessionSummaryTimeout() time.Duration {
	raw := strings.TrimSpace(os.Getenv("NJ_SESSION_SUMMARY_TIMEOUT"))
	if raw == "" {
		return defaultSessionSummaryTimeout
	}
	d, err := time.ParseDuration(raw)
	if err != nil || d <= 0 {
		return defaultSessionSummaryTimeout
	}
	return d
}
