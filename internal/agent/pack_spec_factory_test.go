package agent

import (
	"testing"

	"github.com/camronwood/neural-junkie/internal/ai"
	"github.com/camronwood/neural-junkie/internal/packs"
	"github.com/camronwood/neural-junkie/internal/protocol"
)

func TestAgentFactoryFromPackSpecMusicPilot(t *testing.T) {
	mockAI := ai.NewMockProvider()
	hub := shouldRespondTestHub{}
	cases := []packs.AgentSpec{
		{Name: "MusicExpert", Type: "music", Implementation: "builtin/music"},
		{Name: "MusicExpert", Type: "", Implementation: "builtin/music"},
		{Name: "MusicExpert", Type: "expert", Implementation: "builtin/music"}, // Implementation wins
	}
	for _, spec := range cases {
		ag, err := AgentFactoryFromPackSpec(spec, "", mockAI, hub)
		if err != nil {
			t.Fatalf("spec=%+v: %v", spec, err)
		}
		if ag.Info.Type != protocol.AgentTypeMusic {
			t.Fatalf("spec=%+v: type=%q, want music", spec, ag.Info.Type)
		}
		if ag.Info.Name != "MusicExpert" {
			t.Fatalf("name=%q, want MusicExpert", ag.Info.Name)
		}
	}
}

func TestResolveAgentTypeFromPackSpecFallsBackToType(t *testing.T) {
	got := ResolveAgentTypeFromPackSpec(packs.AgentSpec{Type: "backend", Implementation: ""})
	if got != protocol.AgentTypeBackend {
		t.Fatalf("got %q, want backend", got)
	}
	got = ResolveAgentTypeFromPackSpec(packs.AgentSpec{Type: "frontend", Implementation: "sidecar/foo"})
	if got != protocol.AgentTypeFrontend {
		t.Fatalf("non-builtin impl should fall back to type, got %q", got)
	}
}
