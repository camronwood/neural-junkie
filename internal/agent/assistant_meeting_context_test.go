package agent

import (
	"strings"
	"testing"
	"time"

	"github.com/camronwood/neural-junkie/internal/ai"
	"github.com/camronwood/neural-junkie/internal/protocol"
)

func stubAssistantPromptNow(t *testing.T, now time.Time) {
	t.Helper()
	original := assistantPromptNow
	assistantPromptNow = func() time.Time {
		return now
	}
	t.Cleanup(func() {
		assistantPromptNow = original
	})
}

func TestSelectMeetingNotesForPrompt_PrioritizesOlderTitleMatch(t *testing.T) {
	recent := &MeetingNote{
		Title:       "Reader Sync",
		Summary:     "Recent reader calibration discussion.",
		MeetingDate: time.Date(2026, 5, 26, 10, 0, 0, 0, time.UTC),
	}
	olderMatch := &MeetingNote{
		Title:       "PHOENIX TEAM MEETING",
		Summary:     "Calculation methodology and reader deployment strategy.",
		MeetingDate: time.Date(2026, 5, 19, 10, 0, 0, 0, time.UTC),
	}

	selected, matched := selectMeetingNotesForPrompt(
		[]*MeetingNote{recent, olderMatch},
		"do you see the PHOENIX TEAM MEETING notes from today?",
		3,
	)
	if !matched {
		t.Fatal("expected a matching note")
	}
	if len(selected) == 0 || selected[0] != olderMatch {
		t.Fatalf("expected older title match first, got %#v", selected)
	}
}

func TestSelectMeetingNotesForPrompt_PrefersLatestRecurringTitleMatch(t *testing.T) {
	april := &MeetingNote{
		Title:       "NotesPHOENIX_TEAM_MEETING_Apr_21_2026",
		Summary:     "PHOENIX TEAM MEETING reviewed product developments.",
		MeetingDate: time.Date(2026, 4, 21, 10, 0, 0, 0, time.UTC),
	}
	may := &MeetingNote{
		Title:       "NotesPHOENIX_TEAM_MEETING_May_19_2026",
		Summary:     "Team updates regarding calculation methodologies.",
		MeetingDate: time.Date(2026, 5, 19, 10, 0, 0, 0, time.UTC),
	}

	selected, matched := selectMeetingNotesForPrompt(
		[]*MeetingNote{april, may},
		"do you see the PHOENIX TEAM MEETING notes from today?",
		3,
	)
	if !matched {
		t.Fatal("expected matching notes")
	}
	if len(selected) == 0 || selected[0] != may {
		t.Fatalf("expected latest recurring title match first, got %#v", selected)
	}
}

func TestBuildMeetingContextPrompt_IncludesMatchedMeetingAndNoAccessGuardrail(t *testing.T) {
	t.Setenv("HOME", t.TempDir())
	stubAssistantPromptNow(t, time.Date(2026, 5, 26, 12, 0, 0, 0, time.UTC))
	assistant := NewAssistantAgent("Assistant", ai.NewMockProvider(), shouldRespondTestHub{})
	if assistant.storage == nil {
		t.Fatal("expected assistant storage")
	}

	if err := assistant.storage.SaveMeetingNote(&MeetingNote{
		Title:       "Reader Sync",
		Summary:     "Recent reader calibration discussion.",
		MeetingDate: time.Date(2026, 5, 26, 10, 0, 0, 0, time.UTC),
	}); err != nil {
		t.Fatal(err)
	}
	if err := assistant.storage.SaveMeetingNote(&MeetingNote{
		Title:       "PHOENIX TEAM MEETING",
		Summary:     "Discussed calculation methodologies and deployment strategy.",
		ActionItems: []string{"Revise official system documentation"},
		FullContent: "The PHOENIX team covered calculation methodologies, infrastructure architectural improvements, and reader deployment strategies.",
		MeetingDate: time.Date(2026, 5, 19, 10, 0, 0, 0, time.UTC),
	}); err != nil {
		t.Fatal(err)
	}

	msg := protocol.NewMessage(
		protocol.MessageTypeQuestion,
		"general",
		protocol.AgentInfo{ID: "user", Name: "Camron"},
		"do you see the PHOENIX TEAM MEETING notes from today?",
	)
	prompt := assistant.buildMeetingContextPrompt(msg)

	if !strings.Contains(prompt, "PHOENIX TEAM MEETING") {
		t.Fatalf("expected prompt to include matched meeting, got:\n%s", prompt)
	}
	if !strings.Contains(prompt, "do not say you cannot access meeting notes") {
		t.Fatalf("expected no-access guardrail in prompt, got:\n%s", prompt)
	}
	if readerIdx := strings.Index(prompt, "Reader Sync"); readerIdx >= 0 && strings.Index(prompt, "PHOENIX TEAM MEETING") > readerIdx {
		t.Fatalf("expected matched meeting before recent fallback, got:\n%s", prompt)
	}
}

func TestBuildMeetingContextPrompt_AnchorsTodayToCurrentDate(t *testing.T) {
	t.Setenv("HOME", t.TempDir())
	stubAssistantPromptNow(t, time.Date(2026, 5, 26, 12, 0, 0, 0, time.UTC))
	assistant := NewAssistantAgent("Assistant", ai.NewMockProvider(), shouldRespondTestHub{})
	if assistant.storage == nil {
		t.Fatal("expected assistant storage")
	}

	if err := assistant.storage.SaveMeetingNote(&MeetingNote{
		Title:       "PHOENIX TEAM MEETING",
		Summary:     "Team updates regarding calculation methodologies.",
		MeetingDate: time.Date(2026, 5, 19, 10, 0, 0, 0, time.UTC),
	}); err != nil {
		t.Fatal(err)
	}

	msg := protocol.NewMessage(
		protocol.MessageTypeQuestion,
		"general",
		protocol.AgentInfo{ID: "user", Name: "Camron"},
		"do you have meeting notes for today?",
	)
	prompt := assistant.buildMeetingContextPrompt(msg)

	if !strings.Contains(prompt, "Current date: Tuesday, May 26, 2026") {
		t.Fatalf("expected current date in prompt, got:\n%s", prompt)
	}
	if !strings.Contains(prompt, "No synced meeting notes are dated today (May 26, 2026)") {
		t.Fatalf("expected no-today guardrail in prompt, got:\n%s", prompt)
	}
	if !strings.Contains(prompt, "do not describe them as today's notes") {
		t.Fatalf("expected instruction to avoid treating older notes as today, got:\n%s", prompt)
	}
}
