package agent

import (
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

func protocolNewImplMsg(t *testing.T, meta map[string]interface{}) *protocol.Message {
	t.Helper()
	msg := protocol.NewMessage(protocol.MessageTypeQuestion, "ch", protocol.AgentInfo{ID: "u", Name: "User"}, "fix boot")
	if meta != nil {
		msg.Metadata = meta
	}
	return msg
}
