package ai

import "testing"

func TestOllamaSmallChatModel(t *testing.T) {
	if !OllamaSmallChatModel("qwen2.5:7b") {
		t.Fatal("expected qwen2.5:7b to be small chat model")
	}
	if OllamaSmallChatModel("qwen2.5-coder:14b") {
		t.Fatal("did not expect 14b to be small chat model")
	}
	if !OllamaModelPrefersCompactPrompt("nj-bio:8b") {
		t.Fatal("expected nj-bio compact")
	}
}
