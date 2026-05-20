package meetnotes

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"time"
)

const syncStateFileName = "google_sync_state.json"

// SyncState tracks processed Gmail messages and last sync time.
type SyncState struct {
	ProcessedMessageIDs map[string]bool `json:"processed_message_ids"`
	LastSyncAt          time.Time       `json:"last_sync_at"`
}

// StateStore persists sync progress.
type StateStore struct {
	BaseDir string
}

func (s *StateStore) path() string {
	return filepath.Join(s.BaseDir, syncStateFileName)
}

func (s *StateStore) Load() (*SyncState, error) {
	data, err := os.ReadFile(s.path())
	if err != nil {
		if os.IsNotExist(err) {
			return &SyncState{ProcessedMessageIDs: make(map[string]bool)}, nil
		}
		return nil, fmt.Errorf("read sync state: %w", err)
	}
	var st SyncState
	if err := json.Unmarshal(data, &st); err != nil {
		return nil, fmt.Errorf("parse sync state: %w", err)
	}
	if st.ProcessedMessageIDs == nil {
		st.ProcessedMessageIDs = make(map[string]bool)
	}
	return &st, nil
}

func (s *StateStore) Save(st *SyncState) error {
	if err := os.MkdirAll(s.BaseDir, 0755); err != nil {
		return err
	}
	st.LastSyncAt = time.Now()
	data, err := json.MarshalIndent(st, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(s.path(), data, 0644)
}

func (s *StateStore) Clear() error {
	return os.Remove(s.path())
}

func (s *StateStore) LastSyncAt() *time.Time {
	st, err := s.Load()
	if err != nil || st == nil || st.LastSyncAt.IsZero() {
		return nil
	}
	t := st.LastSyncAt
	return &t
}
