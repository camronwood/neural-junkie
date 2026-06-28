package agent

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/camronwood/neural-junkie/internal/ai"
	"github.com/camronwood/neural-junkie/internal/protocol"
)

func TestExtractMissingNpmModules_viteFormat(t *testing.T) {
	out := "The following dependencies are imported but could not be resolved:\n\n  react-bootstrap (imported by /Users/x/src/components/Header.tsx)\n\nAre they installed?"
	mods := extractMissingNpmModules(out)
	if len(mods) != 1 || mods[0] != "react-bootstrap" {
		t.Fatalf("mods = %v", mods)
	}
}

func TestExtractMissingNpmModules(t *testing.T) {
	out := "src/Header.tsx(2,31): error TS2307: Cannot find module 'react-bootstrap' or its corresponding type declarations."
	mods := extractMissingNpmModules(out)
	if len(mods) != 1 || mods[0] != "react-bootstrap" {
		t.Fatalf("mods = %v", mods)
	}
}

func TestAddMissingDependencyToPackageJSON(t *testing.T) {
	existing := []byte(`{"name":"demo","dependencies":{"react":"^18.0.0"}}`)
	body, ok := addMissingDependencyToPackageJSON(existing, "react-bootstrap")
	if !ok {
		t.Fatal("expected ok")
	}
	if !strings.Contains(body, `"react-bootstrap"`) {
		t.Fatalf("body = %s", body)
	}
	_, ok = addMissingDependencyToPackageJSON(existing, "react")
	if ok {
		t.Fatal("should not add existing dependency")
	}
}

func TestTryEarlyMissingNpmModuleFix(t *testing.T) {
	dir := t.TempDir()
	pkg := `{"name":"demo","scripts":{"build":"tsc"},"dependencies":{"react":"^18.0.0"}}`
	if err := os.WriteFile(filepath.Join(dir, "package.json"), []byte(pkg), 0o644); err != nil {
		t.Fatal(err)
	}

	ag := NewAgent(protocol.AgentTypeFrontend, "FrontendEngineer", nil, ai.NewMockProvider(), shouldRespondTestHub{})
	msg := protocol.NewMessage(protocol.MessageTypeQuestion, "dm-test",
		protocol.AgentInfo{ID: "human", Name: "camron", Type: "human"},
		"app won't boot")
	msg.Metadata = map[string]interface{}{
		"implementation_session": true,
		"workspace_context":      map[string]interface{}{"workspace_path": dir},
	}
	state := &ImplementationSessionState{
		BootFixIntent: true,
		StackManifest: DetectStackManifest(dir),
	}
	state.RecordReadPath("package.json")
	state.RecordCommandRun("npm run build", 1, "exit_code=1\nCannot find module 'react-bootstrap'")
	ctx := withImplementationSessionState(context.Background(), state)

	if !ag.tryEarlyMissingNpmModuleFix(ctx, msg, dir, state) {
		t.Fatal("expected early npm module fix")
	}
	if state.ProposedCount < 1 {
		t.Fatal("expected proposal")
	}
	if !state.DiagnosePhaseComplete {
		t.Fatal("expected diagnose gate cleared")
	}
}

func TestFormatBootFixInterimProgress(t *testing.T) {
	st := &ImplementationSessionState{}
	st.RecordCommandRun("npm run build", 1, "exit_code=1\nCannot find module 'react-bootstrap'")
	got := formatBootFixInterimProgress(st)
	if !strings.Contains(got, "missing_npm_module") {
		t.Fatalf("got %q", got)
	}
}

func TestCompleteBootFixDiagnoseFromBootstrap(t *testing.T) {
	st := &ImplementationSessionState{BootFixIntent: true}
	st.RecordCommandRun("npm run build", 1, "exit_code=1\nCannot find module 'react-bootstrap'")
	completeBootFixDiagnoseFromBootstrap(st)
	if !st.DiagnosePhaseComplete {
		t.Fatal("expected diagnose complete")
	}
}
