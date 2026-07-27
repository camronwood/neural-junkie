package runbooklibrary_test

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/camronwood/neural-junkie/internal/collaboration"
	"github.com/camronwood/neural-junkie/internal/runbooklibrary"
)

func setupUserLibraryHome(t *testing.T) {
	t.Helper()
	tmp := t.TempDir()
	home := filepath.Join(tmp, "home")
	if err := os.MkdirAll(filepath.Join(home, ".neural-junkie", "runbook-library"), 0o755); err != nil {
		t.Fatal(err)
	}
	t.Setenv("HOME", home)
}

func TestNewDefinitionBundle_WrapsDefinitionWithSchemaVersion(t *testing.T) {
	def := runbooklibrary.RunbookDefinition{
		ID:    "d1",
		Title: "Demo",
		Tasks: []collaboration.CollaborationTask{{ID: "t1", Title: "Step 1"}},
	}
	bundle := runbooklibrary.NewDefinitionBundle(def)
	if bundle.SchemaVersion != runbooklibrary.DefinitionBundleSchemaVersion {
		t.Fatalf("schema version = %d", bundle.SchemaVersion)
	}
	if bundle.ExportedAt.IsZero() {
		t.Fatal("expected non-zero ExportedAt")
	}
	if bundle.Definition.ID != "d1" || bundle.Definition.Title != "Demo" {
		t.Fatalf("bundle did not preserve definition: %+v", bundle.Definition)
	}
}

func TestImportDefinitionBundle_MintsFreshIDByDefault(t *testing.T) {
	setupUserLibraryHome(t)

	def := runbooklibrary.RunbookDefinition{
		ID:    "original-id",
		Title: "Imported runbook",
		Tasks: []collaboration.CollaborationTask{{ID: "t1", Title: "Step 1"}},
	}
	saved, err := runbooklibrary.ImportDefinitionBundle(def, false)
	if err != nil {
		t.Fatal(err)
	}
	if saved.ID == "" || saved.ID == "original-id" {
		t.Fatalf("expected fresh ID, got %q", saved.ID)
	}
	if saved.Version != 1 {
		t.Fatalf("expected v1 for a fresh import, got %d", saved.Version)
	}
	if saved.Source != runbooklibrary.SourceUser {
		t.Fatalf("expected SourceUser, got %q", saved.Source)
	}

	loaded, err := runbooklibrary.LoadUserDefinition(saved.ID, 0)
	if err != nil {
		t.Fatal(err)
	}
	if loaded.Title != "Imported runbook" {
		t.Fatalf("title = %q", loaded.Title)
	}
}

func TestImportDefinitionBundle_KeepIDPreservesOriginalAndBumpsVersion(t *testing.T) {
	setupUserLibraryHome(t)

	def := runbooklibrary.RunbookDefinition{
		ID:    "stable-id",
		Title: "Round trip",
		Tasks: []collaboration.CollaborationTask{{ID: "t1", Title: "Step 1"}},
	}
	first, err := runbooklibrary.ImportDefinitionBundle(def, true)
	if err != nil {
		t.Fatal(err)
	}
	if first.ID != "stable-id" {
		t.Fatalf("expected preserved ID, got %q", first.ID)
	}
	if first.Version != 1 {
		t.Fatalf("expected v1, got %d", first.Version)
	}

	second, err := runbooklibrary.ImportDefinitionBundle(def, true)
	if err != nil {
		t.Fatal(err)
	}
	if second.ID != "stable-id" {
		t.Fatalf("expected preserved ID on re-import, got %q", second.ID)
	}
	if second.Version != 2 {
		t.Fatalf("expected version bumped to 2 on re-import, got %d", second.Version)
	}
}
