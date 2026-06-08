package agent

import (
	"testing"

	"github.com/camronwood/neural-junkie/internal/ai"
	"github.com/camronwood/neural-junkie/internal/protocol"
)

func TestShouldRespond_CLIInDMWithIDERoute(t *testing.T) {
	const dm = "dm-camron-cursor"
	ag := NewAgent(protocol.AgentTypeCLI, "Cursor", nil, ai.NewMockProvider(), shouldRespondTestHub{dmChannel: dm})
	msg := protocol.NewMessage(protocol.MessageTypeQuestion, dm,
		protocol.AgentInfo{ID: "human-user", Name: "camronwood", Type: "human"},
		"you here?")
	msg.Metadata = map[string]interface{}{"ide_route_agent_type": "frontend"}
	if !ag.shouldRespond(msg) {
		t.Fatal("expected CLI agent to respond in DM even when IDE routes to frontend")
	}
}
