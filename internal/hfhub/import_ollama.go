package hfhub

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/camronwood/neural-junkie/internal/ollama"
)

// ImportToOllama creates an Ollama model tag from a downloaded GGUF via `ollama create`.
func ImportToOllama(ctx context.Context, ggufPath, ollamaTag string) error {
	ggufPath = strings.TrimSpace(ggufPath)
	ollamaTag = strings.TrimSpace(ollamaTag)
	if ggufPath == "" || ollamaTag == "" {
		return fmt.Errorf("gguf path and ollama_tag are required")
	}
	abs, err := filepath.Abs(ggufPath)
	if err != nil {
		return err
	}
	if _, err := os.Stat(abs); err != nil {
		return fmt.Errorf("gguf file not found: %w", err)
	}

	mgr := ollama.NewManager("")
	st := mgr.DetectInstallation()
	if !st.Installed {
		return fmt.Errorf("ollama is not installed")
	}

	modelfile := openBioGGUFModelfile(abs)
	tmp, err := os.CreateTemp("", "nj-modelfile-*.txt")
	if err != nil {
		return err
	}
	modelfilePath := tmp.Name()
	defer os.Remove(modelfilePath)
	if _, err := tmp.WriteString(modelfile); err != nil {
		tmp.Close()
		return err
	}
	if err := tmp.Close(); err != nil {
		return err
	}

	cmd := exec.CommandContext(ctx, st.Path, "create", ollamaTag, "-f", modelfilePath)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("ollama create failed: %w", err)
	}
	return nil
}

// openBioGGUFModelfile builds a Modelfile with Llama 3 chat template for OpenBio GGUF imports.
func openBioGGUFModelfile(ggufPath string) string {
	return fmt.Sprintf(`FROM %q
TEMPLATE """{{ if .System }}<|begin_of_text|><|start_header_id|>system<|end_header_id|>

{{ .System }}<|eot_id|>{{ end }}{{ if .Prompt }}<|start_header_id|>user<|end_header_id|>

{{ .Prompt }}<|eot_id|><|start_header_id|>assistant<|end_header_id|>

{{ end }}"""
PARAMETER stop "<|eot_id|>"
PARAMETER stop "<|start_header_id|>"
PARAMETER stop "<|end_header_id|>"
`, ggufPath)
}

// DefaultOllamaTag suggests an Ollama tag from repo_id and filename.
func DefaultOllamaTag(repoID, filename string) string {
	repoLower := strings.ToLower(repoID)
	if strings.Contains(repoLower, "openbiollm") {
		return "nj-bio:8b"
	}
	base := strings.TrimSuffix(filename, filepath.Ext(filename))
	base = strings.ToLower(base)
	if len(base) > 48 {
		base = base[:48]
	}
	shortRepo := repoID
	if i := strings.LastIndex(repoID, "/"); i >= 0 {
		shortRepo = repoID[i+1:]
	}
	shortRepo = strings.ToLower(strings.ReplaceAll(shortRepo, ".", "-"))
	return fmt.Sprintf("%s-%s", shortRepo, base)
}
