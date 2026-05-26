package agent

import (
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestSaveMeetingNoteUpsertByGoogleDocID(t *testing.T) {
	tempDir := t.TempDir()
	originalHome := os.Getenv("HOME")
	os.Setenv("HOME", tempDir)
	defer os.Setenv("HOME", originalHome)

	storage, err := NewAssistantStorage()
	if err != nil {
		t.Fatal(err)
	}

	note := &MeetingNote{
		ID:          "meeting_1",
		Source:      "google",
		GoogleDocID: "doc_abc",
		Title:       "First title",
		Summary:     "v1",
		MeetingDate: time.Now(),
		IngestedAt:  time.Now(),
	}
	if err := storage.SaveMeetingNote(note); err != nil {
		t.Fatal(err)
	}

	updated := &MeetingNote{
		Source:      "google",
		GoogleDocID: "doc_abc",
		Title:       "Updated title",
		Summary:     "v2",
		MeetingDate: time.Now(),
		IngestedAt:  time.Now(),
	}
	if err := storage.SaveMeetingNote(updated); err != nil {
		t.Fatal(err)
	}

	notes, err := storage.LoadMeetingNotes()
	if err != nil {
		t.Fatal(err)
	}
	if len(notes) != 1 {
		t.Fatalf("expected 1 note, got %d", len(notes))
	}
	if notes[0].ID != "meeting_1" {
		t.Fatalf("expected id meeting_1, got %s", notes[0].ID)
	}
	if notes[0].Title != "Updated title" {
		t.Fatalf("expected updated title, got %s", notes[0].Title)
	}
}

func TestLoadConfigDefaultsMissingGoogleMeetNotesEnabled(t *testing.T) {
	t.Setenv("HOME", t.TempDir())
	storage, err := NewAssistantStorage()
	if err != nil {
		t.Fatal(err)
	}

	configPath := filepath.Join(storage.baseDir, "config.json")
	legacyConfig := []byte(`{
  "timezone": "UTC",
  "default_channel": "general",
  "reminder_advance": 15,
  "keywords": ["meeting"],
  "proactive_assistance": true
}`)
	if err := os.WriteFile(configPath, legacyConfig, 0644); err != nil {
		t.Fatal(err)
	}

	config, err := storage.LoadConfig()
	if err != nil {
		t.Fatal(err)
	}
	if !config.GoogleMeetNotesEnabled {
		t.Fatal("expected missing google_meet_notes_enabled to default to true")
	}
	if config.GoogleSyncIntervalMinutes != 15 {
		t.Fatalf("expected default sync interval, got %d", config.GoogleSyncIntervalMinutes)
	}
}

func TestLoadConfigPreservesExplicitGoogleMeetNotesDisabled(t *testing.T) {
	t.Setenv("HOME", t.TempDir())
	storage, err := NewAssistantStorage()
	if err != nil {
		t.Fatal(err)
	}

	configPath := filepath.Join(storage.baseDir, "config.json")
	disabledConfig := []byte(`{
  "timezone": "UTC",
  "default_channel": "general",
  "reminder_advance": 15,
  "keywords": ["meeting"],
  "google_meet_notes_enabled": false
}`)
	if err := os.WriteFile(configPath, disabledConfig, 0644); err != nil {
		t.Fatal(err)
	}

	config, err := storage.LoadConfig()
	if err != nil {
		t.Fatal(err)
	}
	if config.GoogleMeetNotesEnabled {
		t.Fatal("expected explicit google_meet_notes_enabled=false to be preserved")
	}
}
