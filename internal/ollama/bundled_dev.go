package ollama

import (
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
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

// NeuralJunkieModelsDir is the isolated models path for bundled Ollama when no system install exists.
func NeuralJunkieModelsDir() string {
	if home, err := os.UserHomeDir(); err == nil {
		return filepath.Join(home, ".neural-junkie", "ollama-models")
	}
	return filepath.Join(os.TempDir(), "neural-junkie-ollama-models")
}

// SystemOllamaModelsDir is the default store used by a system `ollama` install (~/.ollama/models).
func SystemOllamaModelsDir() string {
	home, err := os.UserHomeDir()
	if err != nil {
		return ""
	}
	return filepath.Join(home, ".ollama", "models")
}

// SystemOllamaOnPath reports whether the `ollama` CLI is available (dev machines with a real install).
func SystemOllamaOnPath() bool {
	_, err := exec.LookPath("ollama")
	return err == nil
}

// ResolveOllamaModelsDir picks where bundled `ollama serve` should store models.
// Prefer the system store when `ollama` is on PATH so dev/benchmarks and the desktop app share pulls.
// NJ_OLLAMA_MODELS overrides everything (also honored by Tauri and ensure-ollama.sh).
func ResolveOllamaModelsDir() string {
	if p := strings.TrimSpace(os.Getenv("NJ_OLLAMA_MODELS")); p != "" {
		return p
	}
	if SystemOllamaOnPath() {
		if dir := SystemOllamaModelsDir(); dir != "" {
			return dir
		}
	}
	return NeuralJunkieModelsDir()
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
