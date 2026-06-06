package hfhub

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/camronwood/neural-junkie/internal/ollama"
)

// ModelfileOpts optional template/parameters for composed models.
type ModelfileOpts struct {
	Template string
}

// ComposeModelfile builds a Modelfile with FROM base tag + ADAPTER path.
func ComposeModelfile(baseTag, adapterPath string, opts ModelfileOpts) string {
	baseTag = strings.TrimSpace(baseTag)
	adapterPath = strings.TrimSpace(adapterPath)
	var b strings.Builder
	fmt.Fprintf(&b, "FROM %q\n", baseTag)
	fmt.Fprintf(&b, "ADAPTER %q\n", adapterPath)
	if t := strings.TrimSpace(opts.Template); t != "" {
		fmt.Fprintf(&b, "TEMPLATE %q\n", t)
	}
	return b.String()
}

// llama3ChatTemplate returns the Llama 3 Instruct chat template for Modelfile TEMPLATE blocks.
func llama3ChatTemplate() string {
	return "{{ if .System }}<|begin_of_text|><|start_header_id|>system<|end_header_id|>\n\n{{ .System }}<|eot_id|>{{ end }}{{ if .Prompt }}<|start_header_id|>user<|end_header_id|>\n\n{{ .Prompt }}<|eot_id|><|start_header_id|>assistant<|end_header_id|>\n\n{{ end }}"
}

func adapterModelfileOpts(baseTag, ollamaTag string) ModelfileOpts {
	base := strings.ToLower(strings.TrimSpace(baseTag))
	tag := strings.ToLower(strings.TrimSpace(ollamaTag))
	if strings.Contains(base, "llama3") || tag == BiologyLoRATag {
		return ModelfileOpts{Template: llama3ChatTemplate()}
	}
	return ModelfileOpts{}
}

// AdapterConfigFilename is required by Ollama alongside adapter weights.
const AdapterConfigFilename = "adapter_config.json"

// ResolveAdapterDir returns the directory Ollama expects for ADAPTER (contains adapter_config.json + weights).
func ResolveAdapterDir(adapterPath string) (string, error) {
	adapterPath = strings.TrimSpace(adapterPath)
	if adapterPath == "" {
		return "", fmt.Errorf("adapter path is required")
	}
	info, err := os.Stat(adapterPath)
	if err != nil {
		return "", fmt.Errorf("adapter path not found: %w", err)
	}
	dir := adapterPath
	if !info.IsDir() {
		dir = filepath.Dir(adapterPath)
	}
	configPath := filepath.Join(dir, AdapterConfigFilename)
	if _, err := os.Stat(configPath); err != nil {
		return "", fmt.Errorf("%s missing in %s (download adapter_config.json from Hugging Face)", AdapterConfigFilename, dir)
	}
	return dir, nil
}

// ImportAdapterToOllama creates an Ollama model tag from a base tag + LoRA adapter directory or weights file.
func ImportAdapterToOllama(ctx context.Context, baseTag, adapterPath, ollamaTag string) error {
	baseTag = strings.TrimSpace(baseTag)
	adapterPath = strings.TrimSpace(adapterPath)
	ollamaTag = strings.TrimSpace(ollamaTag)
	if baseTag == "" || adapterPath == "" || ollamaTag == "" {
		return fmt.Errorf("base_ollama_tag, adapter path, and ollama_tag are required")
	}
	if !OllamaSafetensorLoRABaseSupported(baseTag) {
		return fmt.Errorf("ollama safetensors LoRA unsupported for base %q (Ollama supports Llama, Mistral, Gemma bases only; Qwen adapters require GGUF LoRA conversion)", baseTag)
	}
	adapterDir, err := ResolveAdapterDir(adapterPath)
	if err != nil {
		return err
	}
	if msg := WarnAdapterBaseMismatch(baseTag, adapterDir); msg != "" {
		fmt.Fprintf(os.Stderr, "⚠️  LoRA compose: %s\n", msg)
	}

	mgr := ollama.NewManager("")
	st := mgr.DetectInstallation()
	if !st.Installed {
		return fmt.Errorf("ollama is not installed")
	}
	ok, err := mgr.HasModel(ctx, baseTag)
	if err != nil {
		return fmt.Errorf("check base model: %w", err)
	}
	if !ok {
		return fmt.Errorf("base model %q is not installed in Ollama — pull it first (e.g. ollama pull %s)", baseTag, baseTag)
	}

	workDir, cleanup, err := stageAdapterForOllama(adapterDir)
	if err != nil {
		return err
	}
	defer cleanup()

	modelfile := composeStagedAdapterModelfile(baseTag, ollamaTag)
	modelfilePath := filepath.Join(workDir, "Modelfile")
	if err := os.WriteFile(modelfilePath, []byte(modelfile), 0o644); err != nil {
		return fmt.Errorf("write modelfile: %w", err)
	}
	return runOllamaCreate(ctx, st.Path, ollamaTag, modelfilePath)
}

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
	return runOllamaCreate(ctx, st.Path, ollamaTag, modelfilePath)
}

func runOllamaCreate(ctx context.Context, ollamaBin, ollamaTag, modelfilePath string) error {
	cmd := exec.CommandContext(ctx, ollamaBin, "create", ollamaTag, "-f", modelfilePath)
	cmd.Dir = filepath.Dir(modelfilePath)
	var stderr bytes.Buffer
	cmd.Stdout = os.Stdout
	cmd.Stderr = io.MultiWriter(os.Stderr, &stderr)
	if err := cmd.Run(); err != nil {
		if msg := strings.TrimSpace(stderr.String()); msg != "" {
			// Ollama often prints the real reason on stderr (e.g. unsupported architecture).
			if strings.Contains(msg, "Error:") {
				if i := strings.LastIndex(msg, "Error:"); i >= 0 {
					msg = strings.TrimSpace(msg[i:])
				}
			}
			return fmt.Errorf("%s", msg)
		}
		return fmt.Errorf("ollama create failed: %w", err)
	}
	return nil
}

// openBioGGUFModelfile builds a Modelfile with Llama 3 chat template for OpenBio GGUF imports.
func openBioGGUFModelfile(ggufPath string) string {
	return fmt.Sprintf(`FROM %q
TEMPLATE %q
PARAMETER stop "<|eot_id|>"
PARAMETER stop "<|start_header_id|>"
PARAMETER stop "<|end_header_id|>"
`, ggufPath, llama3ChatTemplate())
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
