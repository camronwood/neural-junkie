package agent

import (
	"strings"
	"testing"

	"github.com/camronwood/neural-junkie/internal/ai"
	"github.com/camronwood/neural-junkie/internal/config"
	"github.com/camronwood/neural-junkie/internal/mcp"
	biologymcp "github.com/camronwood/neural-junkie/internal/mcp/biology"
	"github.com/camronwood/neural-junkie/internal/protocol"
)

func TestEffectiveToolLoopModelKoesnUsesQwenFallback(t *testing.T) {
	eff := ai.NewOllamaProviderWithConfig("http://localhost:11434", "koesn/llama3-openbiollm-8b:latest")
	model, fallback := effectiveToolLoopModel(eff)
	if !fallback {
		t.Fatal("expected fallback for koesn chat model")
	}
	if model != ai.OllamaBiologyFallbackModel {
		t.Fatalf("got tool loop model %q, want %q", model, ai.OllamaBiologyFallbackModel)
	}
}

func TestEffectiveToolLoopModelQwenNoFallback(t *testing.T) {
	eff := ai.NewOllamaProviderWithConfig("http://localhost:11434", "qwen2.5:7b")
	model, fallback := effectiveToolLoopModel(eff)
	if fallback {
		t.Fatal("expected no fallback for qwen")
	}
	if model != "qwen2.5:7b" {
		t.Fatalf("got %q", model)
	}
}

func TestDescribeToolCapabilitiesBiologyMCP(t *testing.T) {
	cfg := config.DefaultConfig()
	cfg.MCP.Enabled = true
	cfg.Packs.Enabled[config.PackLifeSciences] = true
	cfg.SyncAgentsFromPacks()
	mcp.SetAppConfig(cfg)
	bioMCP, err := biologymcp.NewBiologyMCP()
	if err != nil {
		t.Fatal(err)
	}
	ollama := ai.NewOllamaProviderWithConfig("http://localhost:11434", "koesn/llama3-openbiollm-8b:latest")
	a := &Agent{
		Info: protocol.AgentInfo{
			ID:         "bio-1",
			Name:       "BiologyExpert",
			Type:       protocol.AgentTypeBiology,
			AIProvider: "ollama",
			AIModel:    "koesn/llama3-openbiollm-8b:latest",
		},
		AI:        ollama,
		MCPServer: bioMCP,
	}
	cap := a.DescribeToolCapabilities()
	if cap.ToolCount < 2 {
		t.Fatalf("expected at least 2 tools, got %d: %+v", cap.ToolCount, cap.Tools)
	}
	names := make(map[string]bool)
	for _, tool := range cap.Tools {
		names[tool.Name] = true
		if tool.Source != "mcp" {
			t.Fatalf("expected mcp source for %s", tool.Name)
		}
	}
	if !names["analyze_sequence"] || !names["fold_protein"] {
		t.Fatalf("missing biology tools: %+v", names)
	}
	if !cap.ToolLoopUsesFallback {
		t.Fatal("expected tool loop fallback for koesn")
	}
	if cap.ToolLoopModel != ai.OllamaBiologyFallbackModel {
		t.Fatalf("tool loop model %q", cap.ToolLoopModel)
	}
}

func TestParseToolInputSchema(t *testing.T) {
	schema := `{"type":"object","properties":{"sequence":{"type":"string","description":"Raw sequence"}},"required":["sequence"]}`
	params := parseToolInputSchema([]byte(schema))
	if len(params) != 1 || params[0].Name != "sequence" || !params[0].Required {
		t.Fatalf("got %+v", params)
	}
}

func TestCapabilitiesFromAgentInfoCLI(t *testing.T) {
	cap := CapabilitiesFromAgentInfo(&protocol.AgentInfo{
		ID: "c1", Name: "Cursor", Type: protocol.AgentTypeCLI, AIProvider: "cursor-cli",
	})
	if cap.ToolCount != 0 {
		t.Fatal("CLI should have no hub tools")
	}
	if len(cap.Notes) == 0 || !strings.Contains(cap.Notes[0], "CLI") {
		t.Fatalf("notes: %v", cap.Notes)
	}
}
