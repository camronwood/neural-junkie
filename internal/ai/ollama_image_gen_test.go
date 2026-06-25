package ai

import (
	"testing"
)

func TestImageGeneratorFromEnvDefaultsToOllama(t *testing.T) {
	t.Setenv("NEURAL_JUNKIE_IMAGE_PROVIDER", "")
	t.Setenv("OPENAI_API_KEY", "")
	t.Setenv("OLLAMA_ENDPOINT", "http://127.0.0.1:11434")
	t.Setenv("OLLAMA_IMAGE_MODEL", "z-image")

	gen := ImageGeneratorFromEnv()
	if gen == nil {
		t.Fatal("expected default local Ollama image generator")
	}
	p, ok := gen.(*OllamaImageGenerator)
	if !ok {
		t.Fatalf("expected *OllamaImageGenerator, got %T", gen)
	}
	if p.inner.Endpoint != "http://127.0.0.1:11434/v1" {
		t.Fatalf("endpoint = %q", p.inner.Endpoint)
	}
	if p.model != "z-image" {
		t.Fatalf("model = %q", p.model)
	}
}

func TestImageGeneratorFromEnvOpenAIProvider(t *testing.T) {
	t.Setenv("NEURAL_JUNKIE_IMAGE_PROVIDER", "openai")
	t.Setenv("OPENAI_API_KEY", "sk-test")
	t.Setenv("OPENAI_BASE_URL", "https://api.example.com/v1")
	t.Setenv("NEURAL_JUNKIE_IMAGE_MODEL", "dall-e-3")

	gen := ImageGeneratorFromEnv()
	if gen == nil {
		t.Fatal("expected OpenAI image generator")
	}
	p, ok := gen.(*OpenAICompatProvider)
	if !ok {
		t.Fatalf("expected *OpenAICompatProvider, got %T", gen)
	}
	if p.Endpoint != "https://api.example.com/v1" {
		t.Fatalf("endpoint = %q", p.Endpoint)
	}
	if p.APIKey != "sk-test" {
		t.Fatalf("api key = %q", p.APIKey)
	}
}

func TestImageGeneratorFromEnvDisabled(t *testing.T) {
	t.Setenv("NEURAL_JUNKIE_IMAGE_PROVIDER", "none")
	if ImageGeneratorFromEnv() != nil {
		t.Fatal("expected nil when image provider disabled")
	}
}
