package hub

import (
	"testing"

	"github.com/camronwood/neural-junkie/internal/protocol"
)

func TestEnrichAgentImageGeneration(t *testing.T) {
	t.Setenv("OPENAI_API_KEY", "test-key")

	frontend := &protocol.AgentInfo{Type: protocol.AgentTypeFrontend}
	enrichAgentImageGeneration(frontend)
	if !frontend.SupportsImageGeneration {
		t.Fatal("frontend agent should support image generation when OPENAI_API_KEY is set")
	}

	repo := &protocol.AgentInfo{Type: protocol.AgentTypeRepo}
	enrichAgentImageGeneration(repo)
	if repo.SupportsImageGeneration {
		t.Fatal("repo agent should not support image generation")
	}
}

func TestEnrichAgentImageGenerationWithoutKey(t *testing.T) {
	t.Setenv("OPENAI_API_KEY", "")

	frontend := &protocol.AgentInfo{Type: protocol.AgentTypeFrontend}
	enrichAgentImageGeneration(frontend)
	if frontend.SupportsImageGeneration {
		t.Fatal("should not enable image generation without OPENAI_API_KEY")
	}
}
