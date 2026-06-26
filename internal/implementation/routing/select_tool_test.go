package routing

import "testing"

func TestPickToolCapableModelSkipsNonToolFallback(t *testing.T) {
	in := Input{
		OllamaTagToolFilter: func(tag string) bool {
			return tag == "qwen2.5-coder:14b" || tag == "qwen3.5:9b"
		},
	}
	got := pickToolCapableModel(in, "deepseek-coder:6.7b")
	if got != "qwen2.5-coder:14b" {
		t.Fatalf("got %q, want qwen2.5-coder:14b", got)
	}
}
