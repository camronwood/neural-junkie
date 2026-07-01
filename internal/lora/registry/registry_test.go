package registry

import (
	"os"
	"path/filepath"
	"testing"
)

func TestRegisterAndActivate(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "lora-adapters.json")
	s, err := NewStore(path)
	if err != nil {
		t.Fatal(err)
	}
	e1, err := s.Register(RegisterInput{
		OllamaTag:     "nj-repo-test:14b",
		BaseOllamaTag: "llama3.1:8b",
		AgentID:       "agent-1",
		JobID:         "job-1",
		RowCount:      12,
		ArtifactDir:   dir,
	})
	if err != nil {
		t.Fatal(err)
	}
	if e1.Version != 1 {
		t.Fatalf("version = %d, want 1", e1.Version)
	}
	e2, err := s.Register(RegisterInput{
		OllamaTag:     "nj-repo-test:14b",
		BaseOllamaTag: "llama3.1:8b",
		AgentID:       "agent-1",
		JobID:         "job-2",
		RowCount:      20,
		ArtifactDir:   dir,
	})
	if err != nil {
		t.Fatal(err)
	}
	if e2.Version != 2 {
		t.Fatalf("version = %d, want 2", e2.Version)
	}
	active, ok := s.ActiveForTag("nj-repo-test:14b")
	if !ok || active.ID != e2.ID {
		t.Fatalf("active = %+v, ok=%v", active, ok)
	}
	got, err := s.Activate(e1.ID)
	if err != nil {
		t.Fatal(err)
	}
	if got.Status != StatusActive {
		t.Fatalf("status = %q", got.Status)
	}
	active, ok = s.ActiveForTag("nj-repo-test:14b")
	if !ok || active.ID != e1.ID {
		t.Fatalf("after rollback active = %+v", active)
	}
}

func TestStorePersists(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "lora-adapters.json")
	s1, _ := NewStore(path)
	_, _ = s1.Register(RegisterInput{OllamaTag: "nj-x:14b", BaseOllamaTag: "llama3.1:8b"})
	s2, err := NewStore(path)
	if err != nil {
		t.Fatal(err)
	}
	list := s2.List("", "")
	if len(list) != 1 {
		t.Fatalf("len = %d", len(list))
	}
	if _, err := os.Stat(path); err != nil {
		t.Fatal(err)
	}
}
