package ai

import (
	"context"
	"log"
	"net/http"
	"os"
	"strings"
	"time"
)

const ollamaImageGenerationTimeout = 10 * time.Minute

// OllamaImageGenerator generates images via Ollama's OpenAI-compatible API and
// unloads the model from memory after each use by default (frees VRAM for chat models).
type OllamaImageGenerator struct {
	inner        *OpenAICompatProvider
	baseEndpoint string
	model        string
	unloadAfter  bool
}

// NewOllamaImageGenerator targets Ollama's OpenAI-compatible POST /v1/images/generations endpoint.
func NewOllamaImageGenerator(endpoint, model string) ImageGenerator {
	endpoint = strings.TrimSpace(endpoint)
	if endpoint == "" {
		endpoint = "http://localhost:11434"
	}
	model = strings.TrimSpace(model)
	if model == "" {
		model = DefaultOllamaImageModel
	}
	base := strings.TrimRight(endpoint, "/")
	inner := NewOpenAICompatProvider(base+"/v1", "", model, nil)
	// A cold image-model load can take substantially longer than the shared
	// OpenAI-compatible chat timeout.
	inner.SetHTTPClient(&http.Client{Timeout: ollamaImageGenerationTimeout})
	return &OllamaImageGenerator{
		inner:        inner,
		baseEndpoint: base,
		model:        model,
		unloadAfter:  ollamaImageUnloadAfterUse(),
	}
}

func (g *OllamaImageGenerator) GenerateImage(ctx context.Context, prompt, size string) (string, string, error) {
	if !g.unloadAfter {
		// Kept resident after generate — free it when the app/hub exits.
		NoteOllamaModelUsed(g.baseEndpoint, g.model)
	}
	mime, data, err := g.inner.GenerateImage(ctx, prompt, size)
	if g.unloadAfter {
		unloadCtx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
		defer cancel()
		if unloadErr := UnloadOllamaModel(unloadCtx, g.baseEndpoint, g.model); unloadErr != nil {
			log.Printf("[ollama] unload image model %q: %v", g.model, unloadErr)
		}
	}
	return mime, data, err
}

// DefaultOllamaImageModel is used when no OLLAMA_IMAGE_MODEL / NEURAL_JUNKIE_IMAGE_MODEL is set.
// Requires a pulled Ollama image model (e.g. `ollama pull x/flux2-klein:4b`).
const DefaultOllamaImageModel = "x/flux2-klein:4b"

// ImageGeneratorFromEnv prefers local Ollama image models unless NEURAL_JUNKIE_IMAGE_PROVIDER=openai.
func ImageGeneratorFromEnv() ImageGenerator {
	switch strings.ToLower(strings.TrimSpace(os.Getenv("NEURAL_JUNKIE_IMAGE_PROVIDER"))) {
	case "openai":
		return openAIImageGenFromEnv()
	case "none", "disabled", "off":
		return nil
	default:
		return ollamaImageGenFromEnv()
	}
}

func ollamaImageGenFromEnv() ImageGenerator {
	endpoint := strings.TrimSpace(os.Getenv("OLLAMA_ENDPOINT"))
	model := strings.TrimSpace(os.Getenv("OLLAMA_IMAGE_MODEL"))
	if model == "" {
		model = strings.TrimSpace(os.Getenv("NEURAL_JUNKIE_IMAGE_MODEL"))
	}
	if model == "" {
		model = DefaultOllamaImageModel
	}
	return NewOllamaImageGenerator(endpoint, model)
}

func openAIImageGenFromEnv() ImageGenerator {
	key := strings.TrimSpace(os.Getenv("OPENAI_API_KEY"))
	if key == "" {
		return nil
	}
	endpoint := strings.TrimSpace(os.Getenv("OPENAI_BASE_URL"))
	if endpoint == "" {
		endpoint = "https://api.openai.com/v1"
	}
	model := strings.TrimSpace(os.Getenv("NEURAL_JUNKIE_IMAGE_MODEL"))
	if model == "" {
		model = "dall-e-3"
	}
	return NewOpenAICompatProvider(strings.TrimRight(endpoint, "/"), key, model, nil)
}
