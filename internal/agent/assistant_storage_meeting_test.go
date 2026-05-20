package agent

import (
	"os"
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
