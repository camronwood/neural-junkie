package train

import (
	"os"
	"path/filepath"
	"testing"
)

func TestResolvePythonEnvOverride(t *testing.T) {
	t.Setenv("NEURAL_JUNKIE_LORA_PYTHON", "/custom/python")
	if got := ResolvePython("/tmp/repo"); got != "/custom/python" {
		t.Fatalf("got %q want /custom/python", got)
	}
}

func TestResolvePythonVenv(t *testing.T) {
	t.Setenv("NEURAL_JUNKIE_LORA_PYTHON", "")
	dir := t.TempDir()
	venvDir := filepath.Join(dir, ".venv-lora", "bin")
	if err := os.MkdirAll(venvDir, 0755); err != nil {
		t.Fatal(err)
	}
	py := filepath.Join(venvDir, "python")
	if err := os.WriteFile(py, []byte("#!/bin/sh\n"), 0755); err != nil {
		t.Fatal(err)
	}
	if got := ResolvePython(dir); got != py {
		t.Fatalf("got %q want %q", got, py)
	}
}

func TestResolvePythonFallback(t *testing.T) {
	t.Setenv("NEURAL_JUNKIE_LORA_PYTHON", "")
	if got := ResolvePython(t.TempDir()); got != "python3" {
		t.Fatalf("got %q want python3", got)
	}
}
