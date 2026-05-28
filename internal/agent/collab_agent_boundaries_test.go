package agent

import (
	"strings"
	"testing"

	"github.com/camronwood/neural-junkie/internal/protocol"
)

func TestCollabLaneFor_ArchitectVsDevOpsDistinct(t *testing.T) {
	arch := collabLaneFor(protocol.AgentTypeArchitecture, "SoftwareArchitect")
	dev := collabLaneFor(protocol.AgentTypeDevOps, "PlatformEngineer")
	if strings.Contains(arch.owns, "kubectl") || strings.Contains(arch.owns, "CI/CD") {
		t.Fatalf("architect lane should not own infra: %q", arch.owns)
	}
	if strings.Contains(dev.owns, "schema shape") {
		t.Fatalf("devops lane should not own schema design: %q", dev.owns)
	}
	if !strings.Contains(dev.defers, "schema") && !strings.Contains(dev.defers, "Architect") {
		t.Fatalf("devops should defer schema: %q", dev.defers)
	}
}

func TestAppendCollaborationLaneInstructions_ListsPeers(t *testing.T) {
	var b strings.Builder
	info := CollaborationInfo{
		ID:    "c1",
		Phase: "planning",
		Agents: []CollaborationAgentSummary{
			{Name: "Assistant", Type: string(protocol.AgentTypeAssistant), Role: "Facilitation"},
			{Name: "Gemini", Type: string(protocol.AgentTypeCLI), Role: "Implementation"},
			{Name: "PlatformEngineer", Type: string(protocol.AgentTypeDevOps), Role: "Platform"},
		},
	}
	self := protocol.AgentInfo{Name: "Assistant", Type: protocol.AgentTypeAssistant}
	appendCollaborationLaneInstructions(&b, info, self)
	out := b.String()
	if !strings.Contains(out, "YOUR LANE") || !strings.Contains(out, "@Gemini") || !strings.Contains(out, "@PlatformEngineer") {
		t.Fatalf("expected lane sections and peers, got:\n%s", out)
	}
	if strings.Contains(out, "@Assistant") {
		t.Fatal("should not list self as peer")
	}
}
