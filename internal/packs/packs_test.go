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
	if len(cat.Packs) < 3 {
		t.Fatal("expected at least 3 catalog entries")
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

func TestSoftwareDevelopmentNoLoRAAdapters(t *testing.T) {
	m, err := LoadBuiltinManifest("software-development")
	if err != nil {
		t.Fatal(err)
	}
	if len(m.LoRAAdapters) != 0 {
		t.Fatalf("expected no lora adapters on dev pack, got %d", len(m.LoRAAdapters))
	}
}

func TestSpecialistTuningLoRAAdapters(t *testing.T) {
	m, err := LoadBuiltinManifest("specialist-tuning")
	if err != nil {
		t.Fatal(err)
	}
	if len(m.LoRAAdapters) != 4 {
		t.Fatalf("expected 4 lora adapters, got %d", len(m.LoRAAdapters))
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
	m, err := LoadBuiltinManifest("life-sciences")
	if err != nil {
		t.Fatal(err)
	}
	if len(m.LoRAAdapters) != 0 {
		t.Fatalf("expected no lora adapters on life-sciences pack, got %d", len(m.LoRAAdapters))
	}
}
