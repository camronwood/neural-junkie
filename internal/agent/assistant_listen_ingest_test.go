package agent

import (
	"context"
	"os"
	"testing"

	"github.com/camronwood/neural-junkie/internal/ai"
)

func TestIngestDesktopListenTranscript(t *testing.T) {
	t.Setenv("HOME", t.TempDir())
	assistant := NewAssistantAgent("Assistant", ai.NewMockProvider(), shouldRespondTestHub{})
	if assistant.storage == nil {
		t.Fatal("expected storage")
	}

	note, err := assistant.IngestDesktopListenTranscript(context.Background(), "Smoke listen",
		"We discussed desktop listen. Action item: Camron to verify whisper stub.")
	if err != nil {
		t.Fatal(err)
	}
	if note.Source != "desktop_listen" {
		t.Fatalf("source=%q", note.Source)
	}
	if note.ID == "" {
		t.Fatal("expected note id")
	}
	if note.FullContent == "" {
		t.Fatal("expected full content")
	}

	notes, err := assistant.storage.LoadMeetingNotes()
	if err != nil {
		t.Fatal(err)
	}
	if len(notes) != 1 || notes[0].Source != "desktop_listen" {
		t.Fatalf("notes=%#v", notes)
	}
}

func TestIngestDesktopListenTranscriptEmpty(t *testing.T) {
	t.Setenv("HOME", t.TempDir())
	assistant := NewAssistantAgent("Assistant", ai.NewMockProvider(), shouldRespondTestHub{})
	_, err := assistant.IngestDesktopListenTranscript(context.Background(), "x", "   ")
	if err == nil {
		t.Fatal("expected error for empty transcript")
	}
}

func TestDefaultConfigDesktopListenEnabled(t *testing.T) {
	cfg := DefaultAssistantConfig()
	if !cfg.DesktopListenEnabled {
		t.Fatal("expected desktop_listen_enabled default true")
	}
	_ = os.DevNull
}
