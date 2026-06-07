package hfhub

import (
	"fmt"
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

	if err := materializeFile(configSrc, filepath.Join(workDir, AdapterConfigFilename)); err != nil {
		cleanup()
		return "", nil, err
	}
	dstWeights := filepath.Join(workDir, ollamaAdapterWeightsName)
	if err := materializeFile(weightsSrc, dstWeights); err != nil {
		cleanup()
		return "", nil, err
	}
	if err := assertRegularFile(dstWeights); err != nil {
		cleanup()
		return "", nil, fmt.Errorf("staged weights: %w", err)
	}
	return workDir, cleanup, nil
}

// materializeFile copies src to dst as a regular file (never a symlink).
// Ollama rejects ADAPTER paths that escape the staging dir via symlinks.
func materializeFile(src, dst string) error {
	src, err := filepath.EvalSymlinks(src)
	if err != nil {
		return fmt.Errorf("resolve %s: %w", filepath.Base(src), err)
	}
	data, err := os.ReadFile(src)
	if err != nil {
		return fmt.Errorf("read %s: %w", filepath.Base(src), err)
	}
	if err := os.WriteFile(dst, data, 0o644); err != nil {
		return fmt.Errorf("write %s: %w", filepath.Base(dst), err)
	}
	return assertRegularFile(dst)
}

func assertRegularFile(path string) error {
	st, err := os.Lstat(path)
	if err != nil {
		return err
	}
	if st.Mode()&os.ModeSymlink != 0 {
		return fmt.Errorf("%s is a symlink", filepath.Base(path))
	}
	if !st.Mode().IsRegular() {
		return fmt.Errorf("%s is not a regular file", filepath.Base(path))
	}
	return nil
}

func copyFile(src, dst string) error {
	return materializeFile(src, dst)
}
