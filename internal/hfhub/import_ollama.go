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

// ImportAdapterToOllama creates an Ollama model tag from a base tag + adapter safetensors.
func ImportAdapterToOllama(ctx context.Context, baseTag, adapterPath, ollamaTag string) error {
	baseTag = strings.TrimSpace(baseTag)
	adapterPath = strings.TrimSpace(adapterPath)
	ollamaTag = strings.TrimSpace(ollamaTag)
	if baseTag == "" || adapterPath == "" || ollamaTag == "" {
		return fmt.Errorf("base_ollama_tag, adapter path, and ollama_tag are required")
	}
	abs, err := filepath.Abs(adapterPath)
	if err != nil {
		return err
	}
	if _, err := os.Stat(abs); err != nil {
		return fmt.Errorf("adapter file not found: %w", err)
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

	modelfile := ComposeModelfile(baseTag, abs, adapterModelfileOpts(baseTag, ollamaTag))
	return runOllamaCreate(ctx, st.Path, ollamaTag, modelfile)
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
	return runOllamaCreate(ctx, st.Path, ollamaTag, modelfile)
}

func runOllamaCreate(ctx context.Context, ollamaBin, ollamaTag, modelfile string) error {
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

	cmd := exec.CommandContext(ctx, ollamaBin, "create", ollamaTag, "-f", modelfilePath)
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
