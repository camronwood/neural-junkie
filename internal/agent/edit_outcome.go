package agent

import (
	"strings"
	"sync"
	"sync/atomic"
)

// EditOutcomeSnapshot is a point-in-time view of edit-loop counters for tests/metrics.
type EditOutcomeSnapshot struct {
	Counts map[string]int64 `json:"counts"`
}

var (
	editOutcomeMu      sync.Mutex
	editOutcomeByKey   = map[string]*atomic.Int64{}
)

// RecordEditOutcome increments a counter keyed by tool/code/strategy/auto-approve outcome.
func RecordEditOutcome(tool, code, strategy, autoApproveOutcome string) {
	key := editOutcomeKey(tool, code, strategy, autoApproveOutcome)
	editOutcomeMu.Lock()
	ctr, ok := editOutcomeByKey[key]
	if !ok {
		ctr = &atomic.Int64{}
		editOutcomeByKey[key] = ctr
	}
	editOutcomeMu.Unlock()
	ctr.Add(1)
}

// GetEditOutcomeSnapshot returns a copy of all edit-outcome counters.
func GetEditOutcomeSnapshot() EditOutcomeSnapshot {
	editOutcomeMu.Lock()
	defer editOutcomeMu.Unlock()
	out := make(map[string]int64, len(editOutcomeByKey))
	for k, ctr := range editOutcomeByKey {
		out[k] = ctr.Load()
	}
	return EditOutcomeSnapshot{Counts: out}
}

// ResetEditOutcomeCounters clears counters (tests only).
func ResetEditOutcomeCounters() {
	editOutcomeMu.Lock()
	defer editOutcomeMu.Unlock()
	editOutcomeByKey = map[string]*atomic.Int64{}
}

func editOutcomeKey(tool, code, strategy, autoApproveOutcome string) string {
	parts := []string{
		strings.TrimSpace(tool),
		strings.TrimSpace(code),
		strings.TrimSpace(strategy),
		strings.TrimSpace(autoApproveOutcome),
	}
	for i, p := range parts {
		if p == "" {
			parts[i] = "-"
		}
	}
	return strings.Join(parts, "|")
}
