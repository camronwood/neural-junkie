package ollama

import (
	"os"
	"path/filepath"
	"runtime"
)

// HostTriple matches Rust target_triple() / scripts/fetch-ollama.sh layout.
func HostTriple() string {
	switch runtime.GOOS {
	case "darwin":
		if runtime.GOARCH == "arm64" {
			return "aarch64-apple-darwin"
		}
		return "x86_64-apple-darwin"
	case "linux":
		if runtime.GOARCH == "amd64" {
			return "x86_64-unknown-linux-gnu"
		}
	case "windows":
		if runtime.GOARCH == "amd64" {
			return "x86_64-pc-windows-msvc"
		}
	}
	return ""
}

func bundledBinaryInDir(dir string) string {
	if dir == "" {
		return ""
	}
	candidates := []string{
		filepath.Join(dir, "ollama"),
		filepath.Join(dir, "bin", "ollama"),
		filepath.Join(dir, "ollama.exe"),
	}
	for _, c := range candidates {
		if st, err := os.Stat(c); err == nil && !st.IsDir() {
			return c
		}
	}
	return ""
}

// DevBundledBinaryPath returns the fetch-ollama binary under desktop/src-tauri/ollama/{triple}.
func DevBundledBinaryPath() string {
	triple := HostTriple()
	if triple == "" {
		return ""
	}
	rel := filepath.Join("desktop", "src-tauri", "ollama", triple)
	wd, err := os.Getwd()
	if err == nil {
		if p := bundledBinaryInDir(filepath.Join(wd, rel)); p != "" {
			return p
		}
	}
	if home, err := os.UserHomeDir(); err == nil {
		for _, base := range []string{
			filepath.Join(home, "development", "projects", "neural-junkie"),
			filepath.Join(home, "development", "neural-junkie"),
		} {
			if p := bundledBinaryInDir(filepath.Join(base, rel)); p != "" {
				return p
			}
		}
	}
	return ""
}

// NeuralJunkieModelsDir is the shared models path for bundled/dev Ollama (matches Tauri app data fallback).
func NeuralJunkieModelsDir() string {
	if home, err := os.UserHomeDir(); err == nil {
		return filepath.Join(home, ".neural-junkie", "ollama-models")
	}
	return filepath.Join(os.TempDir(), "neural-junkie-ollama-models")
}

func resolveBundledBinary() (path string, bundled bool) {
	// Prefer user-updated runtime so POST /api/ollama/update takes effect without rewriting the app bundle.
	if p := UserRuntimeBinary(); p != "" {
		return p, true
	}
	if p := BundledBinaryPath(); p != "" {
		return p, true
	}
	if p := DevBundledBinaryPath(); p != "" {
		return p, true
	}
	return "", false
}
