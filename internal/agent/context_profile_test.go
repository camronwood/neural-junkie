package agent

import (
	"testing"

	"github.com/camronwood/neural-junkie/internal/protocol"
)

func TestOllamaParameterSizeB(t *testing.T) {
	t.Parallel()
	size, ok := ollamaParameterSizeB("qwen3.5:9b")
	if !ok || size != 9 {
		t.Fatalf("qwen3.5:9b size=%v ok=%v", size, ok)
	}
	size, ok = ollamaParameterSizeB("qwen2.5-coder:14b-instruct")
	if !ok || size != 14 {
		t.Fatalf("14b size=%v ok=%v", size, ok)
	}
	if _, ok := ollamaParameterSizeB("claude-sonnet-4"); ok {
		t.Fatal("cloud model must not parse as ollama size")
	}
}

func TestResolveTurnContextProfile_qwen9bConstrainedUnder64KB(t *testing.T) {
	t.Parallel()
	a := &Agent{Info: protocol.AgentInfo{AIProvider: "ollama", AIModel: "qwen3.5:9b"}}
	msg := protocol.NewMessage(protocol.MessageTypeChat, "general", protocol.AgentInfo{Name: "Camron"}, "add HelloWorld")
	msg.Metadata = map[string]interface{}{"editor_mode": "agent", "composer_mode": "agent"}
	p := resolveTurnContextProfile(a, msg, 8192)
	if !p.Constrained {
		t.Fatal("expected constrained")
	}
	if p.MaxPromptBytes <= 0 || p.MaxPromptBytes >= 64*1024 {
		t.Fatalf("MaxPromptBytes=%d want well under 64KB", p.MaxPromptBytes)
	}
	if !p.NativeToolsOnly || !p.EmptyRetry {
		t.Fatalf("native/retry flags: %+v", p)
	}
	if p.EagerCodebase || p.EagerRepoConsult {
		t.Fatalf("unscoped retrieval must be off: %+v", p)
	}
}

func TestResolveTurnContextProfile_anthropicNotConstrained(t *testing.T) {
	t.Parallel()
	a := &Agent{Info: protocol.AgentInfo{AIProvider: "anthropic", AIModel: "claude-sonnet-4"}}
	msg := protocol.NewMessage(protocol.MessageTypeChat, "general", protocol.AgentInfo{Name: "Camron"}, "add HelloWorld")
	msg.Metadata = map[string]interface{}{"editor_mode": "agent", "composer_mode": "agent"}
	p := resolveTurnContextProfile(a, msg, 8192)
	if p.Constrained {
		t.Fatalf("cloud must not be constrained: %+v", p)
	}
	if p.MaxPromptBytes != 0 {
		t.Fatalf("unconstrained MaxPromptBytes=%d", p.MaxPromptBytes)
	}
}

func TestResolveTurnContextProfile_citedPathEnablesEagerCodebase(t *testing.T) {
	t.Parallel()
	a := &Agent{Info: protocol.AgentInfo{AIProvider: "ollama", AIModel: "qwen3.5:9b"}}
	msg := protocol.NewMessage(protocol.MessageTypeChat, "general", protocol.AgentInfo{Name: "Camron"},
		"add HelloWorld to core/sample/main.go")
	msg.Metadata = map[string]interface{}{"editor_mode": "agent", "composer_mode": "agent"}
	p := resolveTurnContextProfile(a, msg, 8192)
	if !p.Constrained || !p.EagerCodebase {
		t.Fatalf("cited path should allow codebase merge: %+v", p)
	}
	if p.EagerRepoConsult {
		t.Fatal("repo consult stays off on constrained IDE")
	}
}

func TestConstrainedMaxPromptBytesWellUnder64KB(t *testing.T) {
	t.Parallel()
	got := constrainedMaxPromptBytes(8192)
	if got >= 64*1024 || got < constrainedMaxPromptBytesMin {
		t.Fatalf("got %d", got)
	}
}

func TestIsIDEComposerTurn(t *testing.T) {
	t.Parallel()
	agentMsg := &protocol.Message{Metadata: map[string]interface{}{"editor_mode": "agent"}}
	if !isIDEComposerTurn(agentMsg) {
		t.Fatal("agent")
	}
	if isIDEComposerTurn(&protocol.Message{Content: "hi"}) {
		t.Fatal("bare chat is not an IDE composer turn")
	}
}
