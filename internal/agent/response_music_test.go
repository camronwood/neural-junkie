package agent

import (
	"context"
	"strings"
	"testing"

	"github.com/camronwood/neural-junkie/internal/intent"
	"github.com/camronwood/neural-junkie/internal/protocol"
)

// Note: UserRequestsGeneratedMusic is a deprecated phrase-matching stub (always false) — // phrase-migration-shim
// see response_music.go. Music routing is now stamp-first via messageSuppressesMusicGeneration
// / tryHubMusicGenerationShortcut in music_gen_tools.go, exercised below with a stamped
// ActionMusic TurnDecision.

func stampMusicDecision(t *testing.T, msg *protocol.Message) {
	t.Helper()
	if err := protocol.StampTurnDecision(msg, intent.TurnDecision{
		SchemaVersion: intent.SchemaVersion, Interaction: intent.InteractionTask,
		RequestedAction: intent.ActionMusic, Action: intent.ActionMusic,
		Mutation: intent.MutationExternal, Confidence: 0.95, Source: intent.SourceLocalModel,
	}); err != nil {
		t.Fatal(err)
	}
}

func TestMusicStyleTagsFromMessage(t *testing.T) {
	if got := MusicStyleTagsFromMessage("Can you generate me a song?"); got != "" {
		t.Fatalf("bare request should yield empty tags, got %q", got)
	}
	got := MusicStyleTagsFromMessage("Generate a dark synthwave song with analog bass")
	if got == "" {
		t.Fatal("expected style tags from descriptive request")
	}
	if !strings.Contains(strings.ToLower(got), "synthwave") {
		t.Fatalf("expected synthwave in %q", got)
	}
}

func TestTryHubMusicGenerationShortcut(t *testing.T) {
	hub := &musicGenTestHub{enabled: true}
	a := &Agent{
		Info: protocol.AgentInfo{Name: "MusicExpert", Type: protocol.AgentTypeMusic},
		Hub:  hub,
	}
	msg := protocol.NewMessage(protocol.MessageTypeQuestion, "music", protocol.AgentInfo{Name: "Camron"}, "Can you generate me a song?")
	stampMusicDecision(t, msg)
	resp, ok := a.tryHubMusicGenerationShortcut(context.Background(), msg)
	if !ok {
		t.Fatal("expected shortcut to handle song request")
	}
	if resp == "" {
		t.Fatal("expected non-empty response")
	}
	if !hub.posted {
		t.Fatal("expected hub music generation")
	}
	if hub.style == "" {
		t.Fatal("expected style tags")
	}
}

func TestTryHubMusicGenerationShortcutPackDisabled(t *testing.T) {
	hub := &musicGenTestHub{enabled: false}
	a := &Agent{
		Info: protocol.AgentInfo{Name: "MusicExpert", Type: protocol.AgentTypeMusic},
		Hub:  hub,
	}
	msg := protocol.NewMessage(protocol.MessageTypeQuestion, "music", protocol.AgentInfo{Name: "Camron"}, "Can you generate me a song?")
	stampMusicDecision(t, msg)
	resp, ok := a.tryHubMusicGenerationShortcut(context.Background(), msg)
	if !ok {
		t.Fatal("expected shortcut when pack disabled for music agent")
	}
	if resp == "" || !strings.Contains(strings.ToLower(resp), "domain packs") {
		t.Fatalf("expected pack install hint, got %q", resp)
	}
}
