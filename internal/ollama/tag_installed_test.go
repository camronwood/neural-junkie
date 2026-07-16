package ollama

import "testing"

func TestTagInstalled(t *testing.T) {
	installed := []string{
		"qwen3.5:9b",
		"qwen2.5-coder:14b",
		"mistral:7b",
		"llama3.1:8b",
	}

	tests := []struct {
		tag  string
		want bool
	}{
		{"qwen3.5:9b", true},
		{"qwen2.5-coder:14b", true},
		{"mistral:7b", true},
		{"qwen2.5:7b", false},
		{"llama3.1", true},
		{"llama3.1:8b", true},
		{"llama3.1:8b:latest", true},
	}
	for _, tc := range tests {
		if got := TagInstalled(installed, tc.tag); got != tc.want {
			t.Fatalf("TagInstalled(%q) = %v, want %v", tc.tag, got, tc.want)
		}
	}
}
