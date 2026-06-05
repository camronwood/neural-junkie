package ollama

import (
	"os"
	"path/filepath"
	"testing"
)

func TestBundledBinaryPath_prefersEnv(t *testing.T) {
	dir := t.TempDir()
	bin := filepath.Join(dir, "ollama")
	if err := os.WriteFile(bin, []byte{0}, 0o755); err != nil {
		t.Fatal(err)
	}
	t.Setenv("NJ_BUNDLED_OLLAMA", bin)
	if got := BundledBinaryPath(); got != bin {
		t.Fatalf("BundledBinaryPath() = %q, want %q", got, bin)
	}
}

func TestDetectInstallation_usesBundledBinary(t *testing.T) {
	dir := t.TempDir()
	bin := filepath.Join(dir, "ollama")
	if err := os.WriteFile(bin, []byte{0}, 0o755); err != nil {
		t.Fatal(err)
	}
	t.Setenv("NJ_BUNDLED_OLLAMA", bin)

	m := NewManager("")
	status := m.DetectInstallation()
	if !status.Installed {
		t.Fatal("expected installed")
	}
	if !status.Bundled {
		t.Fatal("expected bundled")
	}
	if status.Path != bin {
		t.Fatalf("path = %q, want %q", status.Path, bin)
	}
}

func TestInstallOllama_noOpWhenBundled(t *testing.T) {
	dir := t.TempDir()
	bin := filepath.Join(dir, "ollama")
	if err := os.WriteFile(bin, []byte{0}, 0o755); err != nil {
		t.Fatal(err)
	}
	t.Setenv("NJ_BUNDLED_OLLAMA", bin)

	m := NewManager("")
	if err := m.InstallOllama(t.Context(), nil); err != nil {
		t.Fatalf("InstallOllama: %v", err)
	}
}
