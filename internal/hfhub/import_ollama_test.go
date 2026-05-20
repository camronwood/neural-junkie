package hfhub

import (
	"strings"
	"testing"
)

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
