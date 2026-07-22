package packsidecar

import (
	"testing"
	"time"

	"github.com/camronwood/neural-junkie/internal/packs"
)

func TestSupervisorUsesBoundedBackoffAndTerminalState(t *testing.T) {
	manager := NewManager()
	defer manager.StopAll()
	manager.mu.Lock()
	manager.desired["pack"] = packs.SidecarEnv{PackID: "pack"}
	manager.mu.Unlock()

	now := time.Unix(1_000, 0)
	for attempt := 1; attempt <= maxRestarts; attempt++ {
		manager.noteSidecarFailure("pack", nil, now, "unhealthy")
		status := manager.Status()
		if len(status) != 1 || status[0].RestartAttempts != attempt {
			t.Fatalf("attempt %d status=%#v", attempt, status)
		}
		if attempt < maxRestarts && status[0].NextRestart.IsZero() {
			t.Fatalf("attempt %d should schedule restart", attempt)
		}
	}
	status := manager.Status()[0]
	if status.TerminalError != "unhealthy" || !status.NextRestart.IsZero() {
		t.Fatalf("terminal status=%#v", status)
	}
}

func TestManualRestartStateCanBeReset(t *testing.T) {
	manager := NewManager()
	defer manager.StopAll()
	manager.mu.Lock()
	manager.desired["pack"] = packs.SidecarEnv{PackID: "pack"}
	manager.restartAttempts["pack"] = maxRestarts
	manager.terminalErrors["pack"] = "failed"
	manager.restartAttempts["pack"] = 0
	delete(manager.terminalErrors, "pack")
	manager.mu.Unlock()
	status := manager.Status()[0]
	if status.RestartAttempts != 0 || status.TerminalError != "" {
		t.Fatalf("status=%#v", status)
	}
}
