package hub

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"

	"github.com/camronwood/neural-junkie/internal/delegation"
)

func handoffStorePath() (string, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}
	dir := filepath.Join(home, ".neural-junkie")
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return "", err
	}
	return filepath.Join(dir, "capability-handoffs.json"), nil
}

func (h *Hub) persistHandoffs() error {
	if h == nil {
		return nil
	}
	h.mu.RLock()
	records := make([]delegation.HandoffRecord, 0, len(h.handoffs))
	for _, record := range h.handoffs {
		records = append(records, record)
	}
	h.mu.RUnlock()
	path, err := handoffStorePath()
	if err != nil {
		return err
	}
	data, err := json.MarshalIndent(records, "", "  ")
	if err != nil {
		return err
	}
	tmp := path + ".tmp"
	if err := os.WriteFile(tmp, data, 0o600); err != nil {
		return err
	}
	if err := os.Rename(tmp, path); err != nil {
		return fmt.Errorf("persist handoffs: %w", err)
	}
	return nil
}

func (h *Hub) loadHandoffs() {
	path, err := handoffStorePath()
	if err != nil {
		return
	}
	data, err := os.ReadFile(path)
	if err != nil {
		return
	}
	var records []delegation.HandoffRecord
	if json.Unmarshal(data, &records) != nil {
		return
	}
	h.mu.Lock()
	for _, record := range records {
		if record.ID != "" {
			h.handoffs[record.ID] = record
		}
	}
	h.mu.Unlock()
}

func (h *Hub) upsertHandoff(record delegation.HandoffRecord) {
	if h == nil {
		return
	}
	h.mu.Lock()
	h.handoffs[record.ID] = record
	h.mu.Unlock()
	_ = h.persistHandoffs()
}

// ListHandoffs returns a snapshot for diagnostics and future UI history.
func (h *Hub) ListHandoffs() []delegation.HandoffRecord {
	if h == nil {
		return nil
	}
	h.mu.RLock()
	defer h.mu.RUnlock()
	out := make([]delegation.HandoffRecord, 0, len(h.handoffs))
	for _, record := range h.handoffs {
		out = append(out, record)
	}
	return out
}
