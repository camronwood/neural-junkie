package agent

import (
	"errors"
	"strings"
	"testing"

	"github.com/camronwood/neural-junkie/internal/protocol"
)

func TestClassifyVerifyFailure_build(t *testing.T) {
	out := "$ npm run build\nerror TS2304: Cannot find name 'foo'\nexit_code=1"
	info := classifyVerifyFailure(out, []string{"npm run build"})
	if info.Kind != RepairFailureBuild {
		t.Fatalf("kind=%q", info.Kind)
	}
}

func TestClassifyVerifyFailure_partialSuccess(t *testing.T) {
	out := "$ npm run build\nok\n---\n$ npm test --if-present\nFAIL src/app.test.ts\nexit_code=1"
	info := classifyVerifyFailure(out, []string{"npm run build", "npm test --if-present"})
	if info.Kind != RepairFailurePartialSuccess && info.Kind != RepairFailureTest {
		t.Fatalf("kind=%q", info.Kind)
	}
}

func TestFormatTypedRepairNote_includesCategory(t *testing.T) {
	note := formatVerifyRepairNote(VerifyFailureInfo{
		Kind:    RepairFailureBuild,
		Summary: "error TS2304",
	}, "raw")
	if !strings.Contains(note, "Feedback category: build") {
		t.Fatalf("note=%q", note)
	}
}

func TestOutcomeScore(t *testing.T) {
	if outcomeScore(map[string]interface{}{"outcome": "applied_and_verified"}) != 100 {
		t.Fatal("expected 100 for verified")
	}
	if outcomeScore(map[string]interface{}{"outcome": "no_changes"}) != 0 {
		t.Fatal("expected 0 for no changes")
	}
}

func TestImplementationBestOfKMetadata(t *testing.T) {
	msg := protocolNewImplMsg(t, map[string]interface{}{"implementation_best_of_k": 3})
	if k := implementationBestOfK(msg); k != 3 {
		t.Fatalf("k=%d", k)
	}
	msg2 := protocolNewImplMsg(t, map[string]interface{}{"implementation_best_of_k_boot_fix": true})
	if k := implementationBestOfK(msg2); k != defaultBootFixBestOfK {
		t.Fatalf("boot fix k=%d", k)
	}
}

func TestProposalErrorCircuitBreaker_repeatedIdenticalFailure(t *testing.T) {
	state := &ImplementationSessionState{}
	err := errors.New("grounding required: read the stack manifest and use read_file or glob_file_search before proposing edits")

	if state.recordProposalError(err) {
		t.Fatal("first failure should allow one repair attempt")
	}
	if !state.recordProposalError(err) {
		t.Fatal("second identical failure should trip the circuit breaker")
	}
	if state.ConsecutiveProposalErrors != identicalProposalErrorLimit {
		t.Fatalf("consecutive=%d want %d", state.ConsecutiveProposalErrors, identicalProposalErrorLimit)
	}
	if !state.CircuitBreakerFired {
		t.Fatal("expected circuit breaker state to be recorded in the outcome")
	}
	if len(state.PreflightErrors) != 1 {
		t.Fatalf("expected deduplicated preflight error, got %v", state.PreflightErrors)
	}
}

func TestProposalErrorCircuitBreaker_progressResetsFailure(t *testing.T) {
	state := &ImplementationSessionState{}
	first := errors.New("grounding required")
	second := errors.New("protected file")

	if state.recordProposalError(first) || state.recordProposalError(second) {
		t.Fatal("different failures must not trip the identical-error breaker")
	}
	state.clearProposalError()
	if state.LastProposalError != "" || state.ConsecutiveProposalErrors != 0 {
		t.Fatalf("clear did not reset proposal failure state: %+v", state)
	}
	if state.recordProposalError(first) {
		t.Fatal("first failure after progress should allow another repair attempt")
	}
}

func protocolNewImplMsg(t *testing.T, meta map[string]interface{}) *protocol.Message {
	t.Helper()
	msg := protocol.NewMessage(protocol.MessageTypeQuestion, "ch", protocol.AgentInfo{ID: "u", Name: "User"}, "fix boot")
	if meta != nil {
		msg.Metadata = meta
	}
	return msg
}
