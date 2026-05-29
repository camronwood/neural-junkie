package ai

import "testing"

func TestOllamaWithModel(t *testing.T) {
	base := NewOllamaProviderWithConfig("http://localhost:11434", "qwen2.5-coder:14b")
	clone := OllamaWithModel(base, "nj-security:14b")
	if clone.Model != "nj-security:14b" {
		t.Fatalf("model = %q", clone.Model)
	}
	if base.Model != "qwen2.5-coder:14b" {
		t.Fatal("base model should be unchanged")
	}
	if OllamaWithModel(base, "") != base {
		t.Fatal("empty model should return base")
	}
}
