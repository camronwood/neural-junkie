package hfhub

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestOllamaSafetensorLoRABaseSupported(t *testing.T) {
	if !OllamaSafetensorLoRABaseSupported("llama3:8b") {
		t.Fatal("expected llama3 supported")
	}
	if !OllamaSafetensorLoRABaseSupported("mistral:7b") {
		t.Fatal("expected mistral supported")
	}
	if OllamaSafetensorLoRABaseSupported("qwen2.5-coder:14b") {
		t.Fatal("expected qwen unsupported")
	}
}

func TestComposeStagedAdapterModelfile(t *testing.T) {
	s := composeStagedAdapterModelfile("llama3:8b", BiologyLoRATag)
	if !strings.Contains(s, `FROM "llama3:8b"`) {
		t.Fatalf("missing FROM: %q", s)
	}
	if !strings.Contains(s, "ADAPTER .") {
		t.Fatalf("expected ADAPTER . in staged modelfile: %q", s)
	}
	if !strings.Contains(s, "TEMPLATE") {
		t.Fatal("expected Llama 3 template")
	}
}

func TestStageAdapterForOllama(t *testing.T) {
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, AdapterConfigFilename), []byte(`{"peft_type":"LORA"}`), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "adapter_model.safetensors"), []byte("fake-weights"), 0o644); err != nil {
		t.Fatal(err)
	}
	workDir, cleanup, err := stageAdapterForOllama(dir)
	if err != nil {
		t.Fatal(err)
	}
	defer cleanup()
	if _, err := os.Stat(filepath.Join(workDir, AdapterConfigFilename)); err != nil {
		t.Fatalf("missing config in staging dir: %v", err)
	}
	weightsPath := filepath.Join(workDir, ollamaAdapterWeightsName)
	st, err := os.Lstat(weightsPath)
	if err != nil {
		t.Fatalf("missing weights in staging dir: %v", err)
	}
	if st.Mode()&os.ModeSymlink != 0 {
		t.Fatal("staged weights must not be a symlink")
	}
	data, err := os.ReadFile(weightsPath)
	if err != nil || string(data) != "fake-weights" {
		t.Fatalf("staged weights content: %q err=%v", data, err)
	}
}

func TestStageAdapterForOllamaSymlinkSource(t *testing.T) {
	dir := t.TempDir()
	realWeights := filepath.Join(dir, "real.safetensors")
	if err := os.WriteFile(realWeights, []byte("materialized"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, AdapterConfigFilename), []byte(`{"peft_type":"LORA"}`), 0o644); err != nil {
		t.Fatal(err)
	}
	link := filepath.Join(dir, "adapter_model.safetensors")
	if err := os.Symlink(realWeights, link); err != nil {
		t.Fatal(err)
	}
	workDir, cleanup, err := stageAdapterForOllama(dir)
	if err != nil {
		t.Fatal(err)
	}
	defer cleanup()
	st, err := os.Lstat(filepath.Join(workDir, ollamaAdapterWeightsName))
	if err != nil {
		t.Fatal(err)
	}
	if st.Mode()&os.ModeSymlink != 0 {
		t.Fatal("expected materialized file, got symlink")
	}
}

func TestImportAdapterToOllamaRejectsQwenBase(t *testing.T) {
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, AdapterConfigFilename), []byte(`{}`), 0o644); err != nil {
		t.Fatal(err)
	}
	weights := filepath.Join(dir, "adapter_model.safetensors")
	if err := os.WriteFile(weights, []byte("fake"), 0o644); err != nil {
		t.Fatal(err)
	}
	err := ImportAdapterToOllama(t.Context(), "qwen2.5-coder:14b", weights, "nj-security:14b")
	if err == nil {
		t.Fatal("expected error for qwen base")
	}
	if !strings.Contains(err.Error(), "unsupported") {
		t.Fatalf("expected unsupported message, got: %v", err)
	}
}
