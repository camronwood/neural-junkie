package packs

import (
	"path/filepath"
	"testing"
)

func officialTestManifest(t *testing.T, packID string) *Manifest {
	t.Helper()
	dir, err := filepath.Abs(filepath.Join("testdata", "official", packID))
	if err != nil {
		t.Fatal(err)
	}
	m, err := LoadManifest(dir)
	if err != nil {
		t.Fatalf("pack %s: %v", packID, err)
	}
	return m
}

func TestOfficialPackManifests(t *testing.T) {
	for _, id := range OfficialPackIDs {
		m := officialTestManifest(t, id)
		if m.ID != id {
			t.Fatalf("expected id %s, got %s", id, m.ID)
		}
	}
}

func TestSoftwareDevCapabilities(t *testing.T) {
	m := officialTestManifest(t, "software-development")
	if !m.HasCapability("git-rest") {
		t.Fatal("expected git-rest")
	}
	if m.DefaultLayoutProfile() != "ide" {
		t.Fatalf("expected ide layout, got %s", m.DefaultLayoutProfile())
	}
}

func TestSoftwareDevelopmentNoLoRAAdapters(t *testing.T) {
	m := officialTestManifest(t, "software-development")
	if len(m.LoRAAdapters) != 0 {
		t.Fatalf("expected no lora adapters on dev pack, got %d", len(m.LoRAAdapters))
	}
}

func TestSpecialistTuningLoRAAdapters(t *testing.T) {
	m := officialTestManifest(t, "specialist-tuning")
	if len(m.LoRAAdapters) != 9 {
		t.Fatalf("expected 9 lora adapters, got %d", len(m.LoRAAdapters))
	}
	want := map[string]string{
		"security":    "nj-security:14b",
		"code-review": "nj-code-review:14b",
		"backend":     "nj-backend:14b",
		"biology":     "nj-biology:8b",
	}
	got := make(map[string]string, len(m.LoRAAdapters))
	for _, la := range m.LoRAAdapters {
		got[la.AgentType] = la.OllamaTag
	}
	for agentType, tag := range want {
		if got[agentType] != tag {
			t.Fatalf("agent %s: got tag %q want %q (all: %+v)", agentType, got[agentType], tag, got)
		}
	}
	if !m.HasCapability("lora-training") || !m.HasCapability("lora-compose") {
		t.Fatal("expected lora capabilities")
	}
	if !m.HasCapability("personal-learning") {
		t.Fatal("expected personal-learning capability")
	}
}

func TestLifeSciencesNoLoRAAdapters(t *testing.T) {
	m := officialTestManifest(t, "life-sciences")
	if len(m.LoRAAdapters) != 0 {
		t.Fatalf("expected no lora adapters on life-sciences pack, got %d", len(m.LoRAAdapters))
	}
}

func TestLifeSciencesPackCapabilities(t *testing.T) {
	m := officialTestManifest(t, "life-sciences")
	for _, cap := range []string{"biology-api", "biology-workbench", "biology-sidecar"} {
		if !m.HasCapability(cap) {
			t.Fatalf("expected capability %s", cap)
		}
	}
	if _, ok := m.CapabilityDefs["biology-sidecar"]; !ok {
		t.Fatal("expected biology-sidecar capability_defs")
	}
	if _, ok := m.CapabilityDefs["structure-viewer"]; !ok {
		t.Fatal("expected structure-viewer capability_defs")
	}
}

func TestCADPackCapabilities(t *testing.T) {
	m := officialTestManifest(t, "cad")
	if len(m.LoRAAdapters) != 0 {
		t.Fatalf("expected no lora adapters on cad pack, got %d", len(m.LoRAAdapters))
	}
	for _, cap := range []string{"cad-api", "cad-viewer", "cad-workbench"} {
		if !m.HasCapability(cap) {
			t.Fatalf("expected capability %s", cap)
		}
	}
	if m.DefaultLayoutProfile() != "team" {
		t.Fatalf("expected team layout, got %s", m.DefaultLayoutProfile())
	}
	if m.ExpertSlug != "cad" {
		t.Fatalf("expected expert_slug cad, got %s", m.ExpertSlug)
	}
}

func TestMusicCreationPackCapabilities(t *testing.T) {
	m := officialTestManifest(t, "music-creation")
	for _, cap := range []string{"music-generation", "music-workbench", "music-sidecar"} {
		if !m.HasCapability(cap) {
			t.Fatalf("expected capability %s", cap)
		}
	}
	if _, ok := m.CapabilityDefs["music-sidecar"]; !ok {
		t.Fatal("expected music-sidecar capability_defs")
	}
	if m.ExpertSlug != "music" {
		t.Fatalf("expected expert_slug music, got %s", m.ExpertSlug)
	}
}

func TestPackIDForAgentType(t *testing.T) {
	if got := PackIDForAgentType("backend"); got != "software-development" {
		t.Fatalf("got %q", got)
	}
	if got := PackIDForAgentType("biology"); got != "life-sciences" {
		t.Fatalf("got %q", got)
	}
	if got := PackIDForAgentType("music"); got != "music-creation" {
		t.Fatalf("got %q", got)
	}
	if got := PackIDForAgentType("unknown"); got != "" {
		t.Fatalf("got %q", got)
	}
}
