package train

import (
	"os"
	"path/filepath"
	"runtime"
	"strings"
)

// ResolvePython returns the Python executable for LoRA training.
func ResolvePython(repoRoot string) string {
	if p := strings.TrimSpace(os.Getenv("NEURAL_JUNKIE_LORA_PYTHON")); p != "" {
		return p
	}
	if repoRoot != "" {
		for _, sub := range []string{
			filepath.Join(repoRoot, ".venv-lora-mlx", "bin", "python"),
			filepath.Join(repoRoot, ".venv-lora", "bin", "python"),
		} {
			if st, err := os.Stat(sub); err == nil && !st.IsDir() {
				return sub
			}
		}
	}
	return "python3"
}

// preferMLX reports whether MLX training should be preferred on this host.
func preferMLX() bool {
	if os.Getenv("NJ_LORA_FORCE_UNSLOTH") == "1" {
		return false
	}
	if os.Getenv("NJ_LORA_PREFER_MLX") == "1" {
		return true
	}
	return runtime.GOOS == "darwin" && runtime.GOARCH == "arm64"
}
