package meetnotes

import (
	"os"
	"path/filepath"
	"testing"
)

func TestStateStoreRoundTrip(t *testing.T) {
	dir := t.TempDir()
	s := &StateStore{BaseDir: dir}

	st, err := s.Load()
	if err != nil {
		t.Fatal(err)
	}
	st.ProcessedMessageIDs["msg1"] = true
	if err := s.Save(st); err != nil {
		t.Fatal(err)
	}

	loaded, err := s.Load()
	if err != nil {
		t.Fatal(err)
	}
	if !loaded.ProcessedMessageIDs["msg1"] {
		t.Fatal("expected msg1 to be processed")
	}
	if loaded.LastSyncAt.IsZero() {
		t.Fatal("expected LastSyncAt to be set")
	}

	if err := s.Clear(); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(filepath.Join(dir, syncStateFileName)); !os.IsNotExist(err) {
		t.Fatal("expected sync state file removed")
	}
}
