package config

import (
	"testing"
	"time"
)

func TestSessionSummaryModel_default(t *testing.T) {
	t.Setenv("NJ_SESSION_SUMMARY_MODEL", "")
	if got := SessionSummaryModel(); got != SessionSummaryOllamaModel {
		t.Fatalf("SessionSummaryModel() = %q, want %q", got, SessionSummaryOllamaModel)
	}
}

func TestSessionSummaryModel_envOverride(t *testing.T) {
	t.Setenv("NJ_SESSION_SUMMARY_MODEL", "llama3.2:3b")
	if got := SessionSummaryModel(); got != "llama3.2:3b" {
		t.Fatalf("SessionSummaryModel() = %q, want llama3.2:3b", got)
	}
}

func TestSessionSummaryTimeout_default(t *testing.T) {
	t.Setenv("NJ_SESSION_SUMMARY_TIMEOUT", "")
	if got := SessionSummaryTimeout(); got != 90*time.Second {
		t.Fatalf("SessionSummaryTimeout() = %v, want 90s", got)
	}
}

func TestSessionSummaryTimeout_envOverride(t *testing.T) {
	t.Setenv("NJ_SESSION_SUMMARY_TIMEOUT", "2m")
	if got := SessionSummaryTimeout(); got != 2*time.Minute {
		t.Fatalf("SessionSummaryTimeout() = %v, want 2m", got)
	}
}

func TestSessionSummaryTimeout_invalidFallsBack(t *testing.T) {
	t.Setenv("NJ_SESSION_SUMMARY_TIMEOUT", "not-a-duration")
	if got := SessionSummaryTimeout(); got != 90*time.Second {
		t.Fatalf("SessionSummaryTimeout() = %v, want default 90s", got)
	}
}
