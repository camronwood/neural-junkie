package turnledger

import (
	"strings"
	"testing"
	"time"
)

func TestSafeChannelFile(t *testing.T) {
	if got := SafeChannelFile("dm-camron-frontend"); got != "dm-camron-frontend" {
		t.Fatalf("got %q", got)
	}
	if got := SafeChannelFile("general/#weird"); got != "general__weird" {
		t.Fatalf("got %q", got)
	}
}

func TestAppendAndReadTail(t *testing.T) {
	SetDirForTest(t.TempDir())
	t.Cleanup(func() { SetDirForTest("") })

	ch := "dm-test-ledger"
	for i := 0; i < 5; i++ {
		if err := Append(ch, Entry{
			TS:          time.Now().UTC(),
			Speaker:     "User",
			SpeakerType: "human",
			MsgType:     "question",
			Excerpt:     "Talk about ThemeSettings component please",
		}); err != nil {
			t.Fatalf("append: %v", err)
		}
	}
	tail, err := ReadTail(ch, 3)
	if err != nil {
		t.Fatal(err)
	}
	if len(tail) != 3 {
		t.Fatalf("tail len=%d want 3", len(tail))
	}
	if len(tail[0].Entities) == 0 {
		t.Fatalf("expected ThemeSettings entity extraction, got %#v", tail[0])
	}
	found := false
	for _, e := range tail[0].Entities {
		if e == "ThemeSettings" {
			found = true
			break
		}
	}
	if !found {
		t.Fatalf("entities=%v want ThemeSettings", tail[0].Entities)
	}
}

func TestFormatOverlay(t *testing.T) {
	block := FormatOverlay([]Entry{
		{Speaker: "Camron", SpeakerType: "human", Excerpt: "Build ThemeSettings", Entities: []string{"ThemeSettings"}},
		{Speaker: "FrontendEngineer", SpeakerType: "agent", Excerpt: "Here is ThemeSettings"},
	}, 12)
	for _, part := range []string{"TURN LEDGER", "Camron", "FrontendEngineer", "ThemeSettings"} {
		if !strings.Contains(block, part) {
			t.Fatalf("overlay missing %q: %q", part, block)
		}
	}
}

func TestExtractEntities(t *testing.T) {
	ents := ExtractEntities("Please update `ThemeSettings` and the FocusTrap hook")
	joined := strings.Join(ents, ",")
	if !strings.Contains(joined, "ThemeSettings") || !strings.Contains(joined, "FocusTrap") {
		t.Fatalf("ents=%v", ents)
	}
}
