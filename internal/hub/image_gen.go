package hub

import (
	"context"
	"fmt"
	"os"
	"strings"

	"github.com/camronwood/neural-junkie/internal/ai"
	"github.com/camronwood/neural-junkie/internal/protocol"
)

// ImageGenerationAvailable reports whether the hub can call an OpenAI-compatible Images API.
func ImageGenerationAvailable() bool {
	return OpenAIImageGenFromEnv() != nil
}

// OpenAIImageGenFromEnv builds an image generator from OPENAI_API_KEY (and optional
// OPENAI_BASE_URL, NEURAL_JUNKIE_IMAGE_MODEL).
func OpenAIImageGenFromEnv() ai.ImageGenerator {
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
	return ai.NewOpenAICompatProvider(strings.TrimRight(endpoint, "/"), key, model, nil)
}

func agentTypeSupportsImageGeneration(t protocol.AgentType) bool {
	switch t {
	case protocol.AgentTypeFrontend, protocol.AgentTypeAssistant, protocol.AgentTypeHelper:
		return true
	default:
		return false
	}
}

// enrichAgentImageGeneration sets SupportsImageGeneration on agents that can use hub image gen.
func enrichAgentImageGeneration(agent *protocol.AgentInfo) {
	if agent == nil {
		return
	}
	agent.SupportsImageGeneration = ImageGenerationAvailable() && agentTypeSupportsImageGeneration(agent.Type)
}

// ImageGenerationEnabled implements agent.HubClient.
func (h *Hub) ImageGenerationEnabled() bool {
	return ImageGenerationAvailable()
}

// GenerateAndPostImage generates an image and posts it to a channel.
func (h *Hub) GenerateAndPostImage(ctx context.Context, channel string, from protocol.AgentInfo, prompt, size string) error {
	gen := OpenAIImageGenFromEnv()
	if gen == nil {
		return fmt.Errorf("image generation not configured: set OPENAI_API_KEY on the hub server")
	}
	prompt = strings.TrimSpace(prompt)
	if prompt == "" {
		return fmt.Errorf("empty image prompt")
	}
	mime, b64, err := gen.GenerateImage(ctx, prompt, size)
	if err != nil {
		return err
	}
	out := protocol.NewMessage(protocol.MessageTypeChat, channel, from, "🖼️ Generated image.")
	out.Metadata["generated_image"] = map[string]interface{}{
		"mime": mime,
		"data": b64,
	}
	return h.SendMessage(out)
}
