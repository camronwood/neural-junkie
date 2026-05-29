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
