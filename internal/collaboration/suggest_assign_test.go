package collaboration

import (
	"testing"

	"github.com/camronwood/neural-junkie/internal/protocol"
)

func TestSuggestAssigneeSecurity(t *testing.T) {
	pool := []CollaborationAgent{
		{AgentID: "r1", AgentName: "RustExpert", AgentType: protocol.AgentTypeRust, Expertise: []string{"rust", "cli"}},
		{AgentID: "s1", AgentName: "SecurityExpert", AgentType: protocol.AgentTypeSecurity, Expertise: []string{"auth", "encryption"}},
	}
	s := SuggestAssignee(pool, "Threat model", "Review OAuth and JWT handling for vulnerabilities", nil)
	if s == nil || s.AgentID != "s1" {
		t.Fatalf("expected security expert, got %#v", s)
	}
}
