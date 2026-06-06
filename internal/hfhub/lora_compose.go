package hfhub

import (
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
)

const ollamaAdapterWeightsName = "model.safetensors"

// OllamaSafetensorLoRABaseSupported reports whether Ollama can compose a Hugging Face
// safetensors LoRA on this base tag (see ollama modelfile ADAPTER docs).
func OllamaSafetensorLoRABaseSupported(baseTag string) bool {
	b := normalizeOllamaTag(baseTag)
	switch {
	case strings.Contains(b, "llama"),
		strings.Contains(b, "mistral"),
		strings.Contains(b, "mixtral"),
		strings.Contains(b, "gemma"),
		strings.Contains(b, "codestral"),
		strings.Contains(b, "devstral"):
		return true
	default:
		return false
	}
}

// composeStagedAdapterModelfile builds a Modelfile for a staged adapter directory (ADAPTER .).
func composeStagedAdapterModelfile(baseTag, ollamaTag string) string {
	baseTag = strings.TrimSpace(baseTag)
	opts := adapterModelfileOpts(baseTag, ollamaTag)
	var b strings.Builder
	fmt.Fprintf(&b, "FROM %q\n", baseTag)
	fmt.Fprintln(&b, "ADAPTER .")
	if t := strings.TrimSpace(opts.Template); t != "" {
		fmt.Fprintf(&b, "TEMPLATE %q\n", t)
	}
	return b.String()
}

func findAdapterWeightsFile(dir string) (string, error) {
	dir = strings.TrimSpace(dir)
	for _, name := range []string{"adapter_model.safetensors", "model.safetensors", "adapter.safetensors"} {
		p := filepath.Join(dir, name)
		if st, err := os.Stat(p); err == nil && st.Mode().IsRegular() && st.Size() > 0 {
			return p, nil
		}
	}
	entries, err := os.ReadDir(dir)
	if err != nil {
		return "", err
	}
	for _, e := range entries {
		if e.IsDir() {
			continue
		}
		if strings.HasSuffix(strings.ToLower(e.Name()), ".safetensors") {
			return filepath.Join(dir, e.Name()), nil
		}
	}
	return "", fmt.Errorf("no .safetensors weights found in %s", dir)
}

// stageAdapterForOllama copies adapter_config.json + weights into an isolated directory
// using the layout Ollama expects (model.safetensors + adapter_config.json, ADAPTER .).
func stageAdapterForOllama(adapterDir string) (workDir string, cleanup func(), err error) {
	adapterDir, err = filepath.Abs(adapterDir)
	if err != nil {
		return "", nil, err
	}
	configSrc := filepath.Join(adapterDir, AdapterConfigFilename)
	if _, err := os.Stat(configSrc); err != nil {
		return "", nil, fmt.Errorf("%s missing in %s", AdapterConfigFilename, adapterDir)
	}
	weightsSrc, err := findAdapterWeightsFile(adapterDir)
	if err != nil {
		return "", nil, err
	}

	workDir, err = os.MkdirTemp("", "nj-lora-compose-")
	if err != nil {
		return "", nil, err
	}
	cleanup = func() { _ = os.RemoveAll(workDir) }

	if err := copyFile(configSrc, filepath.Join(workDir, AdapterConfigFilename)); err != nil {
		cleanup()
		return "", nil, err
	}
	if err := copyFile(weightsSrc, filepath.Join(workDir, ollamaAdapterWeightsName)); err != nil {
		cleanup()
		return "", nil, err
	}
	return workDir, cleanup, nil
}

func copyFile(src, dst string) error {
	in, err := os.Open(src)
	if err != nil {
		return fmt.Errorf("copy %s: %w", filepath.Base(src), err)
	}
	defer in.Close()

	out, err := os.Create(dst)
	if err != nil {
		return fmt.Errorf("create %s: %w", filepath.Base(dst), err)
	}
	if _, err := io.Copy(out, in); err != nil {
		out.Close()
		return fmt.Errorf("write %s: %w", filepath.Base(dst), err)
	}
	return out.Close()
}
