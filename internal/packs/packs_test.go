package packs

import "testing"

func TestLoadBuiltinManifests(t *testing.T) {
	for _, id := range BuiltinIDs {
		m, err := LoadBuiltinManifest(id)
		if err != nil {
			t.Fatalf("pack %s: %v", id, err)
		}
		if m.ID != id {
			t.Fatalf("expected id %s, got %s", id, m.ID)
		}
	}
}

func TestLoadBuiltinCatalog(t *testing.T) {
	cat, err := LoadBuiltinCatalog()
	if err != nil {
		t.Fatal(err)
	}
	if len(cat.Packs) < 2 {
		t.Fatal("expected at least 2 catalog entries")
	}
}

func TestSoftwareDevCapabilities(t *testing.T) {
	m, err := LoadBuiltinManifest("software-development")
	if err != nil {
		t.Fatal(err)
	}
	if !m.HasCapability("git-rest") {
		t.Fatal("expected git-rest")
	}
	if m.DefaultLayoutProfile() != "ide" {
		t.Fatalf("expected ide layout, got %s", m.DefaultLayoutProfile())
	}
}

func TestSoftwareDevelopmentLoRAAdapters(t *testing.T) {
	m, err := LoadBuiltinManifest("software-development")
	if err != nil {
		t.Fatal(err)
	}
	if len(m.LoRAAdapters) != 3 {
		t.Fatalf("expected 3 lora adapters, got %d", len(m.LoRAAdapters))
	}
	want := map[string]string{
		"security":    "nj-security:14b",
		"code-review": "nj-code-review:14b",
		"backend":     "nj-backend:14b",
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
}

func TestLifeSciencesLoRAAdapters(t *testing.T) {
	m, err := LoadBuiltinManifest("life-sciences")
	if err != nil {
		t.Fatal(err)
	}
	if len(m.LoRAAdapters) != 1 {
		t.Fatalf("expected 1 lora adapter, got %d", len(m.LoRAAdapters))
	}
	la := m.LoRAAdapters[0]
	if la.OllamaTag != "nj-biology:8b" || la.BaseOllamaTag != "llama3:8b" {
		t.Fatalf("unexpected lora spec: %+v", la)
	}
}
