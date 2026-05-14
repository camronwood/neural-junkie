package config

import "testing"

func TestFirstOllamaEndpoint(t *testing.T) {
	c := DefaultConfig()
	got := c.FirstOllamaEndpoint()
	if got != "http://localhost:11434" {
		t.Fatalf("FirstOllamaEndpoint: got %q want default Ollama URL", got)
	}
}
