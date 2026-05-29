package train

import (
	"os"
	"path/filepath"
	"strings"
)

// ResolvePython returns the Python executable for LoRA training.
// Prefers NEURAL_JUNKIE_LORA_PYTHON, then repoRoot/.venv-lora/bin/python, then python3.
func ResolvePython(repoRoot string) string {
	if p := strings.TrimSpace(os.Getenv("NEURAL_JUNKIE_LORA_PYTHON")); p != "" {
		return p
	}
	if repoRoot != "" {
		venv := filepath.Join(repoRoot, ".venv-lora", "bin", "python")
		if st, err := os.Stat(venv); err == nil && !st.IsDir() {
			return venv
		}
	}
	return "python3"
}
