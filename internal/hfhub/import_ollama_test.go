package hfhub

import (
	"strings"
	"testing"
)

func TestComposeModelfile(t *testing.T) {
	s := ComposeModelfile("qwen2.5-coder:14b", "/tmp/lora-adapter", ModelfileOpts{})
	if !strings.Contains(s, `FROM "qwen2.5-coder:14b"`) {
		t.Fatalf("missing FROM base tag: %q", s)
	}
	if !strings.Contains(s, `ADAPTER "/tmp/lora-adapter"`) {
		t.Fatalf("missing ADAPTER path: %q", s)
	}
}

func TestOpenBioGGUFModelfile(t *testing.T) {
	s := openBioGGUFModelfile("/tmp/test.gguf")
	if !strings.Contains(s, `FROM "/tmp/test.gguf"`) {
		t.Fatalf("missing FROM: %q", s)
	}
	if !strings.Contains(s, "TEMPLATE") {
		t.Fatal("expected Llama 3 TEMPLATE in modelfile")
	}
	if !strings.Contains(s, "<|eot_id|>") {
		t.Fatal("expected eot_id stop in modelfile")
	}
}

func TestDefaultOllamaTagOpenBio(t *testing.T) {
	tag := DefaultOllamaTag("aaditya/OpenBioLLM-Llama3-8B-GGUF", "openbiollm-llama3-8b.Q4_K_M.gguf")
	if tag != "nj-bio:8b" {
		t.Fatalf("tag = %q", tag)
	}
}

func TestDefaultAdapterOllamaTag(t *testing.T) {
	entry := &LibraryModel{
		DefaultOllamaTag: "nj-security:14b",
		RepoID:           "org/lora",
	}
	if got := DefaultAdapterOllamaTag(entry, "adapter_model.safetensors"); got != "nj-security:14b" {
		t.Fatalf("got %q", got)
	}
}

func TestAdapterModelfileBiology(t *testing.T) {
	s := ComposeModelfile("llama3:8b", "/tmp/adapter.safetensors", adapterModelfileOpts("llama3:8b", BiologyLoRATag))
	if !strings.Contains(s, "TEMPLATE") {
		t.Fatalf("expected Llama 3 template for biology adapter: %q", s)
	}
	if !strings.Contains(s, "<|start_header_id|>") {
		t.Fatal("expected Llama 3 header tokens in biology adapter modelfile")
	}
}

func TestBiologyLoRATag(t *testing.T) {
	if got := SpecialistLoRATag("biology"); got != BiologyLoRATag {
		t.Fatalf("biology tag = %q", got)
	}
}

func TestSpecialistAndRepoLoRATags(t *testing.T) {
	if got := SpecialistLoRATag("security"); got != "nj-security:14b" {
		t.Fatalf("specialist tag = %q", got)
	}
	if got := RepoLoRATag("/Users/me/projects/My-App"); got != "nj-repo-my-app:14b" {
		t.Fatalf("repo tag = %q", got)
	}
}

func TestLibraryValidatesAdapterEntries(t *testing.T) {
	models, err := Library()
	if err != nil {
		t.Fatal(err)
	}
	foundAdapter := false
	for _, m := range models {
		if IsAdapterEntry(&m) {
			foundAdapter = true
			if m.BaseOllamaTag == "" {
				t.Fatalf("adapter %q missing base tag", m.RepoID)
			}
		}
	}
	if !foundAdapter {
		t.Fatal("expected at least one adapter catalog entry")
	}
}
