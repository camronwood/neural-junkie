package ai

import (
	"context"
	"os"
	"strings"

	"github.com/camronwood/neural-junkie/internal/protocol"
)

const DefaultHuggingFaceRouterEndpoint = "https://router.huggingface.co/v1"

// HuggingFaceProvider calls HF Inference Router (OpenAI-compatible chat completions).
type HuggingFaceProvider struct {
	inner *OpenAICompatProvider
}

// NewHuggingFaceProvider builds a provider for the given router endpoint, token, and Hub model id (org/repo).
func NewHuggingFaceProvider(endpoint, apiKey, model string) *HuggingFaceProvider {
	if endpoint == "" {
		endpoint = DefaultHuggingFaceRouterEndpoint
	}
	if model == "" {
		model = "meta-llama/Meta-Llama-3-8B-Instruct"
	}
	return &HuggingFaceProvider{
		inner: NewOpenAICompatProvider(endpoint, apiKey, model, nil),
	}
}

// ResolveHFToken returns the first non-empty token from args or HF_TOKEN / HUGGING_FACE_HUB_TOKEN.
func ResolveHFToken(fromConfig string) string {
	if t := strings.TrimSpace(fromConfig); t != "" {
		return t
	}
	if t := strings.TrimSpace(os.Getenv("HF_TOKEN")); t != "" {
		return t
	}
	return strings.TrimSpace(os.Getenv("HUGGING_FACE_HUB_TOKEN"))
}

func (p *HuggingFaceProvider) GenerateResponse(ctx context.Context, prompt string, conversationHistory []protocol.Message) (string, error) {
	return p.inner.GenerateResponse(ctx, prompt, conversationHistory)
}

func (p *HuggingFaceProvider) GenerateVisionResponse(ctx context.Context, prompt string, imageData []byte, imageType string, conversationHistory []protocol.Message) (string, error) {
	return p.inner.GenerateVisionResponse(ctx, prompt, imageData, imageType, conversationHistory)
}

func (p *HuggingFaceProvider) GenerateMultimodal(ctx context.Context, prompt string, images []protocol.UserImagePart, conversationHistory []protocol.Message) (string, error) {
	return p.inner.GenerateMultimodal(ctx, prompt, images, conversationHistory)
}

func (p *HuggingFaceProvider) GenerateMultimodalStream(ctx context.Context, prompt string, images []protocol.UserImagePart, conversationHistory []protocol.Message) (<-chan StreamToken, error) {
	return p.inner.GenerateMultimodalStream(ctx, prompt, images, conversationHistory)
}

func (p *HuggingFaceProvider) GenerateResponseStream(ctx context.Context, prompt string, conversationHistory []protocol.Message) (<-chan StreamToken, error) {
	return p.inner.GenerateResponseStream(ctx, prompt, conversationHistory)
}

func (p *HuggingFaceProvider) SupportsStreaming() bool {
	return p.inner.SupportsStreaming()
}

func (p *HuggingFaceProvider) GetModel() string {
	return p.inner.GetModel()
}
