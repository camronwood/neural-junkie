package agent

import (
	"testing"

	"github.com/camronwood/neural-junkie/internal/ai"
	"github.com/camronwood/neural-junkie/internal/config"
	"github.com/camronwood/neural-junkie/internal/mcp"
	"github.com/camronwood/neural-junkie/internal/protocol"
)

func TestToolCapableProviderGemmaUsesReAct(t *testing.T) {
	cfg := config.DefaultConfig()
	mcp.SetAppConfig(cfg)
	t.Cleanup(func() { mcp.SetAppConfig(nil) })

	eff := ai.NewOllamaProviderWithConfig("http://localhost:11434", "gemma3:12b")
	eff.MarkNativeToolsUnsupported()
	a := &Agent{Info: protocol.AgentInfo{Name: "Test", Type: protocol.AgentTypeAssistant}}
	got := a.toolCapableProvider(t.Context(), eff)
	if _, ok := got.(*ai.ReActToolProvider); !ok {
		t.Fatalf("expected ReActToolProvider, got %T", got)
	}
}

func TestToolCapableProviderKoesnUsesSwap(t *testing.T) {
	cfg := config.DefaultConfig()
	mcp.SetAppConfig(cfg)
	t.Cleanup(func() { mcp.SetAppConfig(nil) })

	eff := ai.NewOllamaProviderWithConfig("http://localhost:11434", "koesn/llama3-openbiollm-8b:latest")
	a := &Agent{Info: protocol.AgentInfo{Name: "Bio", Type: protocol.AgentTypeBiology}}
	got := a.toolCapableProvider(t.Context(), eff)
	if _, ok := got.(*ai.ReActToolProvider); ok {
		t.Fatal("expected swap provider, not ReAct")
	}
	if got.GetModel() == eff.GetModel() {
		t.Fatalf("expected swap model, still on %q", got.GetModel())
	}
}

func TestToolCapableProviderLMStudioUsesNative(t *testing.T) {
	eff := ai.NewLMStudioProviderWithConfig("http://localhost:1234/v1", "qwen2.5-7b-instruct")
	a := &Agent{Info: protocol.AgentInfo{Name: "Bio", Type: protocol.AgentTypeBiology}}
	got := a.toolCapableProvider(t.Context(), eff)
	if got != eff {
		t.Fatalf("expected native LM Studio provider, got %T", got)
	}
}

func TestToolCapableProviderLMStudioUnsupportedUsesReAct(t *testing.T) {
	eff := ai.NewLMStudioProviderWithConfig("http://localhost:1234/v1", "small-model")
	eff.MarkNativeToolsUnsupported()
	a := &Agent{Info: protocol.AgentInfo{Name: "Bio", Type: protocol.AgentTypeBiology}}
	got := a.toolCapableProvider(t.Context(), eff)
	if _, ok := got.(*ai.ReActToolProvider); !ok {
		t.Fatalf("expected ReActToolProvider, got %T", got)
	}
}

func TestToolCapableProviderLMStudioNoOllamaSwap(t *testing.T) {
	eff := ai.NewLMStudioProviderWithConfig("http://localhost:1234/v1", "chat-only")
	a := &Agent{Info: protocol.AgentInfo{Name: "Bio", Type: protocol.AgentTypeBiology}}
	got := a.ollamaToolSwapProvider(t.Context(), eff)
	if got != nil {
		t.Fatalf("expected no Ollama swap for LM Studio, got %T model %q", got, got.GetModel())
	}
}

func TestToolCapableProviderOpenAICompatUnsupportedUsesReAct(t *testing.T) {
	eff := ai.NewOpenAICompatProvider("http://localhost:1234/v1", "", "chat-only", nil)
	eff.MarkNativeToolsUnsupported()
	a := &Agent{Info: protocol.AgentInfo{Name: "Test", Type: protocol.AgentTypeAssistant}}
	got := a.toolCapableProvider(t.Context(), eff)
	if _, ok := got.(*ai.ReActToolProvider); !ok {
		t.Fatalf("expected ReActToolProvider, got %T", got)
	}
}

func TestEffectiveToolLoopModelGemmaReactMode(t *testing.T) {
	cfg := config.DefaultConfig()
	mcp.SetAppConfig(cfg)
	t.Cleanup(func() { mcp.SetAppConfig(nil) })

	eff := ai.NewOllamaProviderWithConfig("http://localhost:11434", "gemma3:12b")
	eff.MarkNativeToolsUnsupported()
	a := &Agent{Info: protocol.AgentInfo{Type: protocol.AgentTypeAssistant}}
	model, fallback, mode := a.effectiveToolLoopRouting(eff)
	if fallback {
		t.Fatal("unexpected fallback")
	}
	if model != "gemma3:12b" || mode != "react" {
		t.Fatalf("model=%q mode=%q", model, mode)
	}
}
