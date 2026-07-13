package workflow

import (
	"encoding/json"
	"os"
	"path/filepath"
	"sync"
	"time"
)

// Event is one append-only workflow transition.
type Event struct {
	TS         time.Time              `json:"ts"`
	Type       string                 `json:"type"`
	CollabID   string                 `json:"collab_id,omitempty"`
	FromPhase  string                 `json:"from,omitempty"`
	ToPhase    string                 `json:"to,omitempty"`
	TaskID     string                 `json:"task_id,omitempty"`
	DispatchToken string              `json:"dispatch_token,omitempty"`
	OutputPreview string              `json:"output_preview,omitempty"`
	Attrs      map[string]interface{} `json:"attrs,omitempty"`
}

var writeMu sync.Mutex

func eventsDir() (string, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(home, ".neural-junkie", "workflow-events"), nil
}

// Append writes one event to workflow-events/{collabID}.jsonl.
func Append(collabID string, ev Event) error {
	if collabID == "" {
		return nil
	}
	if ev.TS.IsZero() {
		ev.TS = time.Now().UTC()
	}
	ev.CollabID = collabID
	dir, err := eventsDir()
	if err != nil {
		return err
	}
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return err
	}
	line, err := json.Marshal(ev)
	if err != nil {
		return err
	}
	writeMu.Lock()
	defer writeMu.Unlock()
	f, err := os.OpenFile(filepath.Join(dir, collabID+".jsonl"), os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0o644)
	if err != nil {
		return err
	}
	defer f.Close()
	_, err = f.Write(append(line, '\n'))
	return err
}

// LogPhaseTransition records a collaboration phase change.
func LogPhaseTransition(collabID, from, to string) {
	_ = Append(collabID, Event{Type: "phase_transition", FromPhase: from, ToPhase: to})
}

// LogTaskDispatched records a task dispatch with idempotency token.
func LogTaskDispatched(collabID, taskID, token string) {
	_ = Append(collabID, Event{Type: "task_dispatched", TaskID: taskID, DispatchToken: token})
}

// LogTaskCompleted records task completion.
func LogTaskCompleted(collabID, taskID, preview string) {
	_ = Append(collabID, Event{Type: "task_completed", TaskID: taskID, OutputPreview: preview})
}

// LogTaskFailed records hub-side task failure (timeout, blocked).
func LogTaskFailed(collabID, taskID, reason string) {
	_ = Append(collabID, Event{Type: "task_failed", TaskID: taskID, Attrs: map[string]interface{}{"reason": reason}})
}

// EventLogPath returns the JSONL path for a collaboration.
func EventLogPath(collabID string) (string, error) {
	dir, err := eventsDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(dir, collabID+".jsonl"), nil
}
