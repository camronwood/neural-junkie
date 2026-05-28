package agent

import (
	"strings"
	"testing"

	"github.com/camronwood/neural-junkie/internal/protocol"
)

func TestAssistantSkipPersonalContextEnrichment_CollabRecap(t *testing.T) {
	msg := protocol.NewMessage(
		protocol.MessageTypeCollabRecap,
		"collab-test",
		protocol.AgentInfo{ID: "system", Name: "System", Type: protocol.AgentTypeGeneral},
		"@Assistant deliver session recap. Discussion mentioned email templates in docs/tim.",
	)
	msg.SetCollaborationID("abc12345-0000-0000-0000-000000000001")
	if !assistantSkipPersonalContextEnrichment(msg) {
		t.Fatal("expected collab recap to skip personal enrichment")
	}
}

func TestBuildEmailContextPrompt_NoRecursionWhenNoEmails(t *testing.T) {
	a := &AssistantAgent{
		Agent: &Agent{
			Info: protocol.AgentInfo{Name: "Assistant", Type: protocol.AgentTypeAssistant},
		},
		storage: &AssistantStorage{},
	}
	msg := protocol.NewMessage(
		protocol.MessageTypeQuestion,
		"general",
		protocol.AgentInfo{ID: "human", Name: "Camron", Type: protocol.AgentTypeGeneral},
		"any emails this week?",
	)
	prompt := a.buildEmailContextPrompt(msg)
	if strings.Contains(prompt, "RECENT EMAILS") {
		t.Fatalf("expected fallback core prompt without email section, got %q", prompt[:minInt(200, len(prompt))])
	}
	if !strings.Contains(prompt, "You are the Assistant") {
		t.Fatalf("expected standard assistant prompt, got %q", prompt[:minInt(200, len(prompt))])
	}
}

func TestCollaborationWorkspaceFocusHint_ResourceAPI(t *testing.T) {
	hint := collaborationWorkspaceFocusHint("Investigate resource api schema registration")
	if !strings.Contains(hint, "resource-api/json_endpoints") {
		t.Fatalf("hint = %q", hint)
	}
	if !strings.Contains(hint, "main.go") {
		t.Fatalf("expected anti-hallucination hint for main.go, got %q", hint)
	}
}
