package agent

import (
	"context"
	"testing"
	"time"

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

func TestSuppressProposalsSince_rollsBackCounts(t *testing.T) {
	state := &ImplementationSessionState{
		ProposedCount:   3,
		RegisteredFiles: []string{"a.go", "b.go"},
		FilesChanged:    []string{"a.go", "b.go", "c.go"},
	}
	state.suppressProposalsSince(1, 1, 1)
	if state.ProposedCount != 1 {
		t.Fatalf("ProposedCount=%d", state.ProposedCount)
	}
	if len(state.RegisteredFiles) != 1 || state.RegisteredFiles[0] != "a.go" {
		t.Fatalf("RegisteredFiles=%v", state.RegisteredFiles)
	}
	if len(state.FilesChanged) != 1 || state.FilesChanged[0] != "a.go" {
		t.Fatalf("FilesChanged=%v", state.FilesChanged)
	}
}

func TestSessionProposedAny_unifiesPredicates(t *testing.T) {
	if (&ImplementationSessionState{}).sessionProposedAny() {
		t.Fatal("empty state should be false")
	}
	if !(&ImplementationSessionState{ProposedCount: 1}).sessionProposedAny() {
		t.Fatal("ProposedCount>0 should be true")
	}
	if !(&ImplementationSessionState{FilesChanged: []string{"x.go"}}).sessionProposedAny() {
		t.Fatal("FilesChanged should be true")
	}
	if !(&ImplementationSessionState{RegisteredFiles: []string{"x.go"}}).sessionProposedAny() {
		t.Fatal("RegisteredFiles should be true")
	}
}

func TestStampImplSessionAbortOutcome(t *testing.T) {
	outcome := map[string]interface{}{"outcome": "no_changes"}
	ctx, cancel := context.WithTimeout(context.Background(), time.Nanosecond)
	defer cancel()
	<-ctx.Done()
	stampImplSessionAbortOutcome(outcome, ctx, context.Background())
	if outcome["outcome"] != "timed_out" {
		t.Fatalf("outcome=%v want timed_out", outcome["outcome"])
	}

	parent, pcancel := context.WithCancel(context.Background())
	pcancel()
	child, ccancel := context.WithCancel(parent)
	defer ccancel()
	outcome2 := map[string]interface{}{"outcome": "no_changes"}
	stampImplSessionAbortOutcome(outcome2, child, parent)
	if outcome2["outcome"] != "cancelled" {
		t.Fatalf("outcome=%v want cancelled", outcome2["outcome"])
	}
}

func TestRequiresDiagnoseGate_greenfieldSwiftDoesNotRequire(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	state := &ImplementationSessionState{StackManifest: &StackManifest{}}
	msg := protocol.NewMessage(protocol.MessageTypeQuestion, "ch", protocol.AgentInfo{ID: "u", Name: "User"}, iosTriviaPrompt)
	if requiresDiagnoseGate(msg, state, dir) {
		t.Fatal("pure greenfield Swift creates must not require diagnose gate")
	}
}

func TestRequiresDiagnoseGate_multiFileExistingTargetsStillRequire(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	writeMultifileTestFile(t, dir, "tailwind.config.js", `module.exports = { content: ["./src/**/*"] };`)
	writeMultifileTestFile(t, dir, "src/App.tsx", `export default function App() { return <div />; }`)
	state := &ImplementationSessionState{
		StackManifest: &StackManifest{
			HasReact:       true,
			HasTailwind:    true,
			TailwindConfig: "tailwind.config.js",
			EntryPoint:     "src/App.tsx",
		},
	}
	msg := protocol.NewMessage(protocol.MessageTypeQuestion, "ch", protocol.AgentInfo{ID: "u", Name: "User"}, "implement light/dark theme toggle in the sidebar")
	if !requiresDiagnoseGate(msg, state, dir) {
		t.Fatal("multi-file edits of existing targets should still require diagnose gate")
	}
}
