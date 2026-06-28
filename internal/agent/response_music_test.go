package agent

import (
	"context"
	"strings"
	"testing"

	"github.com/camronwood/neural-junkie/internal/protocol"
)

func TestUserRequestsGeneratedMusic(t *testing.T) {
	if !UserRequestsGeneratedMusic("Can you generate me a song?") {
		t.Fatal("expected song generation request")
	}
	if !UserRequestsGeneratedMusic("Can you genereate me a song?") {
		t.Fatal("expected typo-tolerant genereate")
	}
	if !UserRequestsGeneratedMusic("compose an upbeat jazz track") {
		t.Fatal("expected compose track request")
	}
	if UserRequestsGeneratedMusic("what time is it?") {
		t.Fatal("expected false for unrelated question")
	}
	if UserRequestsGeneratedMusic("🎵 Generated song.") {
		t.Fatal("delivery boilerplate should not count as a request")
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
	resp, ok := a.tryHubMusicGenerationShortcut(context.Background(), msg)
	if !ok {
		t.Fatal("expected shortcut when pack disabled for music agent")
	}
	if resp == "" || !strings.Contains(strings.ToLower(resp), "domain packs") {
		t.Fatalf("expected pack install hint, got %q", resp)
	}
}
