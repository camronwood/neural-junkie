package agent

import (
	"testing"

	"github.com/camronwood/neural-junkie/internal/protocol"
)

func TestResolveImplementationTrustMode_agentAutoApply(t *testing.T) {
	msg := protocol.NewMessage(protocol.MessageTypeQuestion, "dm-u-arch", protocol.AgentInfo{ID: "u", Name: "User"}, "fix boot")
	msg.Metadata = map[string]interface{}{
		"editor_mode":            "agent",
		"editor_agent_trust":     "interactive",
		"implementation_session": true,
	}
	if got := resolveImplementationTrustMode(msg); got != editorTrustAutoApply {
		t.Fatalf("resolveImplementationTrustMode() = %q want %q", got, editorTrustAutoApply)
	}
}

func TestResolveImplementationTrustMode_askInteractive(t *testing.T) {
	msg := protocol.NewMessage(protocol.MessageTypeQuestion, "dm-u-arch", protocol.AgentInfo{ID: "u", Name: "User"}, "thoughts?")
	msg.Metadata = map[string]interface{}{
		"editor_mode":        "ask",
		"editor_agent_trust": "auto_apply_edits",
	}
	if got := resolveImplementationTrustMode(msg); got != "" {
		t.Fatalf("resolveImplementationTrustMode() = %q want empty (interactive)", got)
	}
}

func TestMessageSuggestsMissingDependencies(t *testing.T) {
	if !messageSuggestsMissingDependencies("also got this: Cannot find module 'react-bootstrap'", nil) {
		t.Fatal("expected missing dependency signal")
	}
	if messageSuggestsMissingDependencies("hello", nil) {
		t.Fatal("did not expect missing dependency signal")
	}
}
