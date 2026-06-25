package hub

import (
	"context"
	"fmt"
	"log"
	"strings"

	"github.com/camronwood/neural-junkie/internal/ai"
	"github.com/camronwood/neural-junkie/internal/protocol"
)

// ImageGenerationAvailable reports whether the hub can generate images (local Ollama by default).
func ImageGenerationAvailable() bool {
	return ai.ImageGeneratorFromEnv() != nil
}

func agentTypeSupportsImageGeneration(t protocol.AgentType) bool {
	switch t {
	case protocol.AgentTypeCLI, protocol.AgentTypeModerator:
		return false
	default:
		return true
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
	gen := ai.ImageGeneratorFromEnv()
	if gen == nil {
		return fmt.Errorf("image generation disabled (set NEURAL_JUNKIE_IMAGE_PROVIDER or pull an Ollama image model)")
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
	meta := map[string]interface{}{
		"mime": mime,
		"data": b64,
	}
	if path, err := saveGeneratedImageFile(out.ID, mime, b64); err != nil {
		log.Printf("[hub] generated image file save failed (inline only): %v", err)
	} else {
		meta["path"] = path
	}
	out.Metadata["generated_image"] = meta
	return h.SendMessage(out)
}
