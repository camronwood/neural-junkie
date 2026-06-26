package agent

import (
	"strings"
	"testing"

	"github.com/camronwood/neural-junkie/internal/ai"
	"github.com/camronwood/neural-junkie/internal/protocol"
)

func TestTryBootFixImplementerRedirect_architectureDM(t *testing.T) {
	const dm = "dm-camron-softwarearchitect"
	ag := NewAgent(protocol.AgentTypeArchitecture, "SoftwareArchitect", nil, ai.NewMockProvider(), shouldRespondTestHub{dmChannel: dm})
	msg := protocol.NewMessage(protocol.MessageTypeQuestion, dm,
		protocol.AgentInfo{ID: "human-user", Name: "camronwood", Type: "human"},
		"This app is not booting up, can you fix the Makefile?")
	msg.Metadata = map[string]interface{}{
		"implementation_session": true,
		"editor_mode":            "agent",
		"workspace_context": map[string]interface{}{
			"workspace_path": t.TempDir(),
		},
	}
	resp, ok := ag.tryBootFixImplementerRedirect(msg)
	if !ok {
		t.Fatal("expected redirect for architecture DM boot-fix")
	}
	if resp == "" || !strings.Contains(resp, "FrontendEngineer") || !strings.Contains(resp, "SoftwareArchitect") {
		t.Fatalf("unexpected redirect: %q", resp)
	}
}

func TestTryBootFixImplementerRedirect_frontendDM(t *testing.T) {
	const dm = "dm-camron-frontendengineer"
	ag := NewAgent(protocol.AgentTypeFrontend, "FrontendEngineer", nil, ai.NewMockProvider(), shouldRespondTestHub{dmChannel: dm})
	msg := protocol.NewMessage(protocol.MessageTypeQuestion, dm,
		protocol.AgentInfo{ID: "human-user", Name: "camronwood", Type: "human"},
		"the app is not booting up")
	msg.Metadata = map[string]interface{}{
		"implementation_session": true,
		"editor_mode":            "agent",
	}
	resp, ok := ag.tryBootFixImplementerRedirect(msg)
	if ok {
		t.Fatalf("frontend DM should not redirect: %q", resp)
	}
}
