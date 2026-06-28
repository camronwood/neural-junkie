package agent

import (
	"os"
	"path/filepath"
	"testing"
)

func TestInferReproCommand_userMentioned(t *testing.T) {
	cmd := inferReproCommand("", nil, "please run npm run build and fix errors")
	if cmd != "npm run build" {
		t.Fatalf("got %q", cmd)
	}
}

func TestInferReproCommand_goDefault(t *testing.T) {
	dir := t.TempDir()
	_ = os.WriteFile(filepath.Join(dir, "go.mod"), []byte("module example.com/test\n\ngo 1.22\n"), 0o644)
	manifest := DetectStackManifest(dir)
	cmd := inferReproCommand(dir, manifest, "the build is broken")
	if cmd != "go build ./..." {
		t.Fatalf("got %q", cmd)
	}
}

func TestInferReproCommand_makeTargetFromError(t *testing.T) {
	cmd := inferReproCommand("", nil, "make: *** No rule to make target 'start-all'.  Stop.")
	if cmd != "make start-all" {
		t.Fatalf("got %q", cmd)
	}
}

func TestDefaultReproCommands_nodeBuild(t *testing.T) {
	dir := t.TempDir()
	_ = os.WriteFile(filepath.Join(dir, "package.json"), []byte(`{"scripts":{"build":"tsc"}}`), 0o644)
	manifest := DetectStackManifest(dir)
	cmds := manifest.DefaultReproCommands("app broken")
	if len(cmds) == 0 || cmds[0] != "npm run build" {
		t.Fatalf("cmds = %v", cmds)
	}
}

func TestFixLikeSessionSucceeded_requiresReproExitZero(t *testing.T) {
	st := &ImplementationSessionState{
		FixLikeIntent: true,
		ReproCommand:  "npm run build",
		ReproExitCode: 1,
	}
	if fixLikeSessionSucceeded(st) {
		t.Fatal("expected failure when repro exit != 0")
	}
	st.ReproExitCode = 0
	if !fixLikeSessionSucceeded(st) {
		t.Fatal("expected success when repro exit 0")
	}
}
