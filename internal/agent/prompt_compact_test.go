package agent

import (
	"strings"
	"testing"

	"github.com/camronwood/neural-junkie/internal/protocol"
)

func TestLooksLikeOllamaPromptLeak(t *testing.T) {
	if !looksLikeOllamaPromptLeak("Be mindful of the user's background and avoid jargon.") {
		t.Fatal("expected leak detection")
	}
	if looksLikeOllamaPromptLeak("DNA polymerase copies the template strand during replication.") {
		t.Fatal("expected valid biology answer")
	}
}

func TestBuildCompactOllamaPrompt(t *testing.T) {
	a := &Agent{
		Info: protocol.AgentInfo{
			Name:       "BiologyExpert",
			Type:       protocol.AgentTypeBiology,
			AIProvider: "Local Ollama",
			AIModel:    "nj-bio:8b",
		},
	}
	user := protocol.AgentInfo{ID: "u1", Name: "Camron", Type: protocol.AgentTypeGeneral}
	msg := protocol.NewMessage(protocol.MessageTypeChat, "dm-test", user, "What is CRISPR?")
	prompt := a.buildCompactOllamaPrompt(msg)
	if len(prompt) > 4000 {
		t.Fatalf("compact prompt too long: %d bytes", len(prompt))
	}
	if strings.Contains(prompt, "BEHAVIORAL RULES") {
		t.Fatal("compact prompt should not include full behavioral rules block")
	}
	if strings.Contains(prompt, "WORKSPACE CONTEXT") {
		t.Fatal("compact prompt should not include workspace context")
	}
}
