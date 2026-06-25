package hub

import (
	"testing"

	"github.com/camronwood/neural-junkie/internal/protocol"
)

func TestEnrichAgentImageGeneration(t *testing.T) {
	t.Setenv("NEURAL_JUNKIE_IMAGE_PROVIDER", "")
	t.Setenv("OPENAI_API_KEY", "")

	frontend := &protocol.AgentInfo{Type: protocol.AgentTypeFrontend}
	enrichAgentImageGeneration(frontend)
	if !frontend.SupportsImageGeneration {
		t.Fatal("frontend agent should support local image generation by default")
	}

	biology := &protocol.AgentInfo{Type: protocol.AgentTypeBiology}
	enrichAgentImageGeneration(biology)
	if !biology.SupportsImageGeneration {
		t.Fatal("biology agent should support image generation")
	}

	cli := &protocol.AgentInfo{Type: protocol.AgentTypeCLI}
	enrichAgentImageGeneration(cli)
	if cli.SupportsImageGeneration {
		t.Fatal("CLI agent should not support hub image generation")
	}
}

func TestEnrichAgentImageGenerationDisabled(t *testing.T) {
	t.Setenv("NEURAL_JUNKIE_IMAGE_PROVIDER", "none")

	frontend := &protocol.AgentInfo{Type: protocol.AgentTypeFrontend}
	enrichAgentImageGeneration(frontend)
	if frontend.SupportsImageGeneration {
		t.Fatal("should not enable image generation when provider is none")
	}
}

func TestEnrichAgentImageGenerationOpenAIProvider(t *testing.T) {
	t.Setenv("NEURAL_JUNKIE_IMAGE_PROVIDER", "openai")
	t.Setenv("OPENAI_API_KEY", "test-key")

	frontend := &protocol.AgentInfo{Type: protocol.AgentTypeFrontend}
	enrichAgentImageGeneration(frontend)
	if !frontend.SupportsImageGeneration {
		t.Fatal("frontend agent should support image generation when OpenAI is configured")
	}
}

func TestEnrichAgentImageGenerationOpenAIWithoutKey(t *testing.T) {
	t.Setenv("NEURAL_JUNKIE_IMAGE_PROVIDER", "openai")
	t.Setenv("OPENAI_API_KEY", "")

	frontend := &protocol.AgentInfo{Type: protocol.AgentTypeFrontend}
	enrichAgentImageGeneration(frontend)
	if frontend.SupportsImageGeneration {
		t.Fatal("should not enable image generation without OPENAI_API_KEY when provider is openai")
	}
}
