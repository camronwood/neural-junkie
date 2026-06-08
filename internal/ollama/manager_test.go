package ollama

import (
	"os"
	"path/filepath"
	"runtime"
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

func TestAutoInstallSupported_falseWhenBundled(t *testing.T) {
	dir := t.TempDir()
	bin := filepath.Join(dir, "ollama")
	if err := os.WriteFile(bin, []byte{0}, 0o755); err != nil {
		t.Fatal(err)
	}
	t.Setenv("NJ_BUNDLED_OLLAMA", bin)
	if AutoInstallSupported() {
		t.Fatal("expected auto install disabled when bundled")
	}
}

func TestSSESafeLine(t *testing.T) {
	got := SSESafeLine("hello\r\nworld\n")
	if got != "hello world" {
		t.Fatalf("SSESafeLine() = %q, want %q", got, "hello world")
	}
}

func TestAutoInstallSupported_trueOnSupportedPlatforms(t *testing.T) {
	if runtime.GOOS != "darwin" && runtime.GOOS != "linux" && runtime.GOOS != "windows" {
		t.Skip("platform-specific test")
	}
	if BundledBinaryPath() != "" {
		t.Skip("bundled ollama in env")
	}
	if !AutoInstallSupported() {
		t.Fatal("expected auto install supported on", runtime.GOOS)
	}
}
