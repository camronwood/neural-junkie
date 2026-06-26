package agent

import (
	"testing"

	"github.com/camronwood/neural-junkie/internal/protocol"
)

func TestRequiresDiagnoseGate_bootFix(t *testing.T) {
	state := &ImplementationSessionState{BootFixIntent: true}
	msg := protocol.NewMessage(protocol.MessageTypeQuestion, "ch", protocol.AgentInfo{ID: "u", Name: "User"}, "fix")
	if !requiresDiagnoseGate(msg, state, "/tmp") {
		t.Fatal("boot-fix should require diagnose gate")
	}
}

func TestResponseContainsDiagnosis(t *testing.T) {
	text := "Analysis:\n- Makefile missing start-all target\n\nPlanned edits:\n- Add start-all target to Makefile"
	if !responseContainsDiagnosis(text) {
		t.Fatal("expected diagnosis detection")
	}
}

func TestDiagnoseGateBlocksProposals(t *testing.T) {
	state := &ImplementationSessionState{DiagnosePhaseRequired: true}
	if !state.diagnoseGateBlocksProposals() {
		t.Fatal("should block before diagnosis")
	}
	state.DiagnosePhaseComplete = true
	if state.diagnoseGateBlocksProposals() {
		t.Fatal("should not block after diagnosis")
	}
}
