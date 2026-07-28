package agent

import (
	"testing"

	"github.com/camronwood/neural-junkie/internal/ai"
	"github.com/camronwood/neural-junkie/internal/packs"
	"github.com/camronwood/neural-junkie/internal/protocol"
)

func TestAgentFactoryFromPackSpecRetiredAbilityTypes(t *testing.T) {
	mockAI := ai.NewMockProvider()
	hub := shouldRespondTestHub{}
	cases := []packs.AgentSpec{
		{Name: "MusicExpert", Type: "music", Implementation: "builtin/music"},
		{Name: "MapsExpert", Type: "maps", Implementation: "builtin/maps"},
		{Name: "WebBrowserExpert", Type: "browser", Implementation: "builtin/browser"},
		{Name: "CodeReviewer", Type: "code-review", Implementation: "builtin/code-review"},
	}
	for _, spec := range cases {
		ag, err := AgentFactoryFromPackSpec(spec, "", mockAI, hub)
		if err == nil || ag != nil {
			t.Fatalf("spec=%+v: expected error for retired ability-pack expert, got ag=%v err=%v", spec, ag, err)
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
