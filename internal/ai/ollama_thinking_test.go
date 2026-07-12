package ai

import (
	"testing"
	"time"
)

func TestOllamaHTTPTimeout_collabModels(t *testing.T) {
	cases := map[string]time.Duration{
		"qwen3.5:9b":        540 * time.Second,
		"gemma3:12b":        540 * time.Second,
		"qwen2.5-coder:14b": 540 * time.Second,
		"devstral:24b":      540 * time.Second,
		"llama3.1":          120 * time.Second,
		"qwen3.5:27b":       600 * time.Second,
	}
	for model, want := range cases {
		if got := ollamaHTTPTimeout(model); got != want {
			t.Errorf("ollamaHTTPTimeout(%q) = %v want %v", model, got, want)
		}
	}
}

func TestOllamaHTTPTimeout_exceedsCollabFileExecution(t *testing.T) {
	collabFileSec := 300
	for _, model := range []string{"qwen3.5:9b", "gemma3:12b", "qwen2.5-coder:14b"} {
		if got := ollamaHTTPTimeout(model); got < time.Duration(collabFileSec)*time.Second {
			t.Errorf("HTTP timeout for %q (%v) should be >= collab file execution (%ds)", model, got, collabFileSec)
		}
	}
}

func TestOllamaChatRequestExplicitlyDisablesDefaultThinking(t *testing.T) {
	p := NewOllamaProviderWithConfig("", "qwen3.5:9b")
	req := p.newChatRequest(nil, true)
	if req.Think == nil || *req.Think {
		t.Fatalf("qwen3.5 non-thinking tag must send think=false, got %#v", req.Think)
	}
}
