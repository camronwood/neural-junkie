package hub

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"github.com/camronwood/neural-junkie/internal/ai"
	"github.com/camronwood/neural-junkie/internal/config"
	"github.com/camronwood/neural-junkie/internal/protocol"
)

func TestEnsureHiddenRepoAgent_Idempotent(t *testing.T) {
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "README.md"), []byte("# test"), 0o644); err != nil {
		t.Fatal(err)
	}

	h := NewHub()
	ch, err := NewCommandHandler(h)
	if err != nil {
		t.Fatal(err)
	}
	ch.appConfig = config.DefaultConfig()

	first, err := ch.EnsureHiddenRepoAgent(context.Background(), dir, EnsureHiddenRepoAgentOptions{})
	if err != nil {
		t.Fatal(err)
	}
	if first == nil {
		t.Fatal("expected hidden repo agent")
	}
	if !first.Info.ConsultOnly {
		t.Fatal("expected consult-only agent")
	}

	second, err := ch.EnsureHiddenRepoAgent(context.Background(), dir, EnsureHiddenRepoAgentOptions{})
	if err != nil {
		t.Fatal(err)
	}
	if second == nil || second.Info.ID != first.Info.ID {
		t.Fatalf("expected same agent id, got first=%v second=%v", first.Info.ID, second.Info.ID)
	}
}

func TestHiddenRepoAgentName(t *testing.T) {
	name := hiddenRepoAgentName("/tmp/neural-junkie")
	if name == "" {
		t.Fatal("expected non-empty name")
	}
	if name == "neural-junkie" {
		t.Fatal("expected internal prefix normalization")
	}
}

func TestNormalizeRepoAgentPath(t *testing.T) {
	abs, err := filepath.Abs(".")
	if err != nil {
		t.Fatal(err)
	}
	got := normalizeRepoAgentPath(abs, false)
	if got == "" {
		t.Fatal("expected normalized local path")
	}
	remote := normalizeRepoAgentPath("/home/user/project", true)
	if remote != "/home/user/project" {
		t.Fatalf("remote path = %q", remote)
	}
}

func TestConsultRepoForPath_NoAgent(t *testing.T) {
	h := NewHub()
	ch, err := NewCommandHandler(h)
	if err != nil {
		t.Fatal(err)
	}
	_, _, err = ch.ConsultRepoForPath(context.Background(), t.TempDir(), "where is main?", "general")
	if err == nil {
		t.Fatal("expected error when no repo agent")
	}
}

func TestSkipDelegationTarget_ConsultOnly(t *testing.T) {
	if !skipDelegationTarget(protocol.AgentInfo{ConsultOnly: true}) {
		t.Fatal("consult-only agents should be skipped for keyword delegation")
	}
	if skipDelegationTarget(protocol.AgentInfo{Name: "FrontendEngineer", Type: protocol.AgentTypeFrontend}) {
		t.Fatal("specialists should not be skipped")
	}
}

func TestCommandHandlerImplementsConsultInterface(t *testing.T) {
	h := NewHub()
	ch, err := NewCommandHandler(h)
	if err != nil {
		t.Fatal(err)
	}
	_ = ch.aiProvider
	if ch.aiProvider == nil {
		ch.aiProvider = ai.NewMockProvider()
	}
	_, _, err = ch.ConsultRepoForPath(context.Background(), "/nonexistent", "q", "")
	if err == nil {
		t.Fatal("expected missing agent error")
	}
}
