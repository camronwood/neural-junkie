package agent

import (
	"strings"
	"testing"

	"github.com/camronwood/neural-junkie/internal/ai"
	"github.com/camronwood/neural-junkie/internal/protocol"
)

func TestBuildCollabRecapPrompt_Compact(t *testing.T) {
	msg := protocol.NewMessage(
		protocol.MessageTypeCollabRecap,
		"collab-test",
		protocol.AgentInfo{ID: "system", Name: "System", Type: protocol.AgentTypeGeneral},
		strings.Repeat("context ", 5000),
	)
	prompt := buildCollabRecapPrompt("Assistant", msg)
	if strings.Contains(prompt, "=== YOUR CAPABILITIES ===") {
		t.Fatal("expected compact recap prompt without assistant capabilities catalog")
	}
	if !strings.Contains(prompt, ai.SystemPromptSeparator) {
		t.Fatal("expected system/user separator")
	}
	if len(prompt) > collabRecapPromptMaxBytes+512 {
		t.Fatalf("prompt too large: %d bytes", len(prompt))
	}
}
