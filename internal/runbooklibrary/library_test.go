package runbooklibrary_test

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/camronwood/neural-junkie/internal/collaboration"
	"github.com/camronwood/neural-junkie/internal/runbooklibrary"
)

func TestSaveAndLoadUserDefinition(t *testing.T) {
	tmp := t.TempDir()
	home := filepath.Join(tmp, "home")
	if err := os.MkdirAll(filepath.Join(home, ".neural-junkie", "runbook-library"), 0o755); err != nil {
		t.Fatal(err)
	}
	t.Setenv("HOME", home)

	def := runbooklibrary.RunbookDefinition{
		Title:       "Test runbook",
		Description: "demo",
		Tasks: []collaboration.CollaborationTask{
			{ID: "t1", Title: "Step 1", Status: collaboration.TaskPending},
		},
	}
	saved, err := runbooklibrary.SaveUserDefinition(def)
	if err != nil {
		t.Fatal(err)
	}
	if saved.Version != 1 {
		t.Fatalf("version = %d", saved.Version)
	}
	loaded, err := runbooklibrary.LoadUserDefinition(saved.ID, 0)
	if err != nil {
		t.Fatal(err)
	}
	if loaded.Title != "Test runbook" {
		t.Fatalf("title = %q", loaded.Title)
	}
}

func TestInterpolateInputs(t *testing.T) {
	got := runbooklibrary.InterpolateString(
		"GET {{inputs.url}} for {{task.title}}",
		nil,
		collaboration.CollaborationTask{Title: "Health"},
		map[string]string{"url": "https://example.com"},
	)
	want := "GET https://example.com for Health"
	if got != want {
		t.Fatalf("got %q want %q", got, want)
	}
}

func TestValidateDefinitionRequiredInput(t *testing.T) {
	def := &runbooklibrary.RunbookDefinition{
		Tasks: []collaboration.CollaborationTask{{ID: "a", Title: "A"}},
		Inputs: []runbooklibrary.RunInputSpec{
			{Key: "url", Required: true},
		},
	}
	warns := runbooklibrary.ValidateDefinition(def, map[string]string{})
	found := false
	for _, w := range warns {
		if w == `required input "url" is missing` {
			found = true
		}
	}
	if !found {
		t.Fatalf("expected required input warning, got %v", warns)
	}
}
