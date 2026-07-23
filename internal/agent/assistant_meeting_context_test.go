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

func TestSelectMeetingNotesForPrompt_LastMeetingUsesChronology(t *testing.T) {
	older := &MeetingNote{
		Title:       "Notes: “PHOENIX TEAM MEETING” Apr 28, 2026",
		Summary:     "Last updates about infrastructure and the most recent summary of deployment strategy.",
		FullContent: "We discussed the last summary of POS detection and the most recent hardware plan.",
		MeetingDate: time.Date(2026, 4, 28, 16, 0, 0, 0, time.UTC),
	}
	newest := &MeetingNote{
		Title:       "Notes: “PHOENIX TEAM MEETING” Jul 21, 2026",
		Summary:     "Production migrations and QC reader deployment.",
		MeetingDate: time.Date(2026, 7, 21, 15, 0, 0, 0, time.UTC),
	}
	mid := &MeetingNote{
		Title:       "Notes: “PHOENIX TEAM MEETING” Jun 30, 2026",
		Summary:     "Technical migrations.",
		MeetingDate: time.Date(2026, 6, 30, 15, 0, 0, 0, time.UTC),
	}

	for _, q := range []string{
		"can you give me a summary of the notes from my last meeting/",
		"what was my last meeting",
		"most recent meeting",
	} {
		selected, matched := selectMeetingNotesForPrompt([]*MeetingNote{older, newest, mid}, q, 3)
		if matched {
			t.Fatalf("query %q: expected chronology fallback (matched=false), got matched=true", q)
		}
		if len(selected) == 0 || selected[0] != newest {
			t.Fatalf("query %q: expected Jul 21 first, got %#v", q, selected)
		}
	}
}

func TestSelectMeetingNotesForPrompt_NamedMeetingStillMatchesTitle(t *testing.T) {
	rovo := &MeetingNote{
		Title:       "Get_the_most_out_of_Rovo_AI",
		Summary:     "Rovo tips.",
		MeetingDate: time.Date(2026, 4, 14, 10, 0, 0, 0, time.UTC),
	}
	phoenix := &MeetingNote{
		Title:       "Notes: “PHOENIX TEAM MEETING” Jul 21, 2026",
		Summary:     "Production migrations.",
		MeetingDate: time.Date(2026, 7, 21, 15, 0, 0, 0, time.UTC),
	}
	selected, matched := selectMeetingNotesForPrompt(
		[]*MeetingNote{rovo, phoenix},
		"summarize the Aclaris IT Security meeting",
		3,
	)
	// No Aclaris note — should fall back to recent by date, not Rovo via "most".
	if matched {
		t.Fatal("expected no title match for Aclaris")
	}
	if len(selected) == 0 || selected[0] != phoenix {
		t.Fatalf("expected newest by date when no title match, got %#v", selected)
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

func TestBuildPromptForIntent_LowSignalStillEnrichesMeetingNotes(t *testing.T) {
	t.Setenv("HOME", t.TempDir())
	stubAssistantPromptNow(t, time.Date(2026, 7, 23, 12, 0, 0, 0, time.UTC))
	assistant := NewAssistantAgent("Assistant", ai.NewMockProvider(), shouldRespondTestHub{})
	if assistant.storage == nil {
		t.Fatal("expected assistant storage")
	}
	if err := assistant.storage.SaveMeetingNote(&MeetingNote{
		Title:       "PHOENIX TEAM MEETING",
		Summary:     "Production migrations and QC reader deployment.",
		MeetingDate: time.Date(2026, 7, 21, 15, 0, 0, 0, time.UTC),
	}); err != nil {
		t.Fatal(err)
	}

	msg := protocol.NewMessage(
		protocol.MessageTypeChat,
		"dm-camron-assistant",
		protocol.AgentInfo{ID: "user", Name: "Camron"},
		"can you give me a summary of the notes from my last meeting/",
	)
	prompt := assistant.buildPromptForIntent(msg, IntentLowSignal)

	if !strings.Contains(prompt, "MEETING NOTES CONTEXT") {
		t.Fatalf("expected meeting notes enrichment on casual/LowSignal turn, got:\n%s", prompt)
	}
	if !strings.Contains(prompt, "PHOENIX TEAM MEETING") {
		t.Fatalf("expected synced meeting in prompt, got:\n%s", prompt)
	}
	if strings.Contains(prompt, "Respond briefly and naturally to the user's latest message only.") {
		t.Fatalf("expected enriched prompt, not minimal casual prompt, got:\n%s", prompt)
	}
}
