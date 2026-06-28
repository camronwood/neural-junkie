package agent

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/camronwood/neural-junkie/internal/protocol"
)

func TestMaybeAskFixClarification_vagueMessage(t *testing.T) {
	dir := t.TempDir()
	msg := protocol.NewMessage(protocol.MessageTypeQuestion, "dm", protocol.AgentInfo{Type: "human"}, "something is broken")
	state := &ImplementationSessionState{
		FixLikeIntent: true,
		StackManifest: DetectStackManifest(dir),
	}
	if q, ask := maybeAskFixClarification(msg, state, dir); !ask || q == "" {
		t.Fatalf("expected clarify question, got ask=%v q=%q", ask, q)
	}
	if state.ClarifyQuestionsAsked != 1 {
		t.Fatalf("clarify count = %d", state.ClarifyQuestionsAsked)
	}
}

func TestMaybeAskFixClarification_skipsWhenReproKnown(t *testing.T) {
	dir := t.TempDir()
	_ = os.WriteFile(filepath.Join(dir, "package.json"), []byte(`{"scripts":{"build":"tsc"}}`), 0o644)
	msg := protocol.NewMessage(protocol.MessageTypeQuestion, "dm", protocol.AgentInfo{Type: "human"}, "npm run build fails with errors")
	state := &ImplementationSessionState{
		FixLikeIntent: true,
		StackManifest: DetectStackManifest(dir),
	}
	if _, ask := maybeAskFixClarification(msg, state, dir); ask {
		t.Fatal("should not ask when repro command is in message")
	}
}

func TestMessageImpliesFixLikeIntent(t *testing.T) {
	if !messageImpliesFixLikeIntent("the app won't boot", nil) {
		t.Fatal("boot message should be fix-like")
	}
	if messageImpliesFixLikeIntent("please add a dark mode theme toggle", nil) {
		t.Fatal("feature ask should not be fix-like")
	}
}
