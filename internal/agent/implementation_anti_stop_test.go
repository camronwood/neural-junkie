package agent

import (
	"testing"

	"github.com/camronwood/neural-junkie/internal/protocol"
)

func TestResponseIsAdvisoryOnly(t *testing.T) {
	if !responseIsAdvisoryOnly("You could try updating the Makefile.") {
		t.Fatal("expected advisory")
	}
	if responseIsAdvisoryOnly("[FILE_CHANGE]\npath=Makefile\n[/FILE_CHANGE]") {
		t.Fatal("file change should not be advisory")
	}
}

func TestResponseClaimsPrematureDone(t *testing.T) {
	if !responseClaimsPrematureDone("I think this is fixed now.") {
		t.Fatal("expected premature done")
	}
}

func TestShouldRejectPrematureStop_verifyFailed(t *testing.T) {
	a := &Agent{}
	state := &ImplementationSessionState{VerifyFailed: true}
	msg := protocol.NewMessage(protocol.MessageTypeQuestion, "ch", protocol.AgentInfo{ID: "u", Name: "User"}, "fix")
	reject, _ := shouldRejectPrematureStop(a, msg, state, false, 0, 5)
	if !reject {
		t.Fatal("expected reject when verify failed")
	}
}
