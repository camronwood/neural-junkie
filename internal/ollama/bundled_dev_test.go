package ollama

import (
	"os"
	"path/filepath"
	"testing"
)

func TestResolveOllamaModelsDirOverride(t *testing.T) {
	t.Setenv("NJ_OLLAMA_MODELS", "/tmp/nj-custom-ollama-models")
	t.Setenv("PATH", "")
	dir := ResolveOllamaModelsDir()
	if dir != "/tmp/nj-custom-ollama-models" {
		t.Fatalf("expected override, got %q", dir)
	}
}

func TestResolveOllamaModelsDirSystemWhenOnPath(t *testing.T) {
	t.Setenv("NJ_OLLAMA_MODELS", "")
	tmp := t.TempDir()
	fakeBin := filepath.Join(tmp, "ollama")
	if err := os.WriteFile(fakeBin, []byte("#!/bin/sh\n"), 0o755); err != nil {
		t.Fatal(err)
	}
	t.Setenv("PATH", tmp)

	home := t.TempDir()
	t.Setenv("HOME", home)
	systemDir := filepath.Join(home, ".ollama", "models")
	if err := os.MkdirAll(systemDir, 0o755); err != nil {
		t.Fatal(err)
	}

	got := ResolveOllamaModelsDir()
	if got != systemDir {
		t.Fatalf("expected system dir %q, got %q", systemDir, got)
	}
}

func TestResolveOllamaModelsDirIsolatedWithoutSystemCLI(t *testing.T) {
	t.Setenv("NJ_OLLAMA_MODELS", "")
	t.Setenv("PATH", "")

	home := t.TempDir()
	t.Setenv("HOME", home)

	got := ResolveOllamaModelsDir()
	want := filepath.Join(home, ".neural-junkie", "ollama-models")
	if got != want {
		t.Fatalf("expected isolated dir %q, got %q", want, got)
	}
}
