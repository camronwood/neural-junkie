package hub

import (
	"context"
	"strings"
	"testing"

	"github.com/camronwood/neural-junkie/internal/agent"
	"github.com/camronwood/neural-junkie/internal/ai"
	"github.com/camronwood/neural-junkie/internal/config"
	"github.com/camronwood/neural-junkie/internal/mcp"
	biologymcp "github.com/camronwood/neural-junkie/internal/mcp/biology"
	"github.com/camronwood/neural-junkie/internal/protocol"
)

func TestFormatChannelToolsListWithBiologyTools(t *testing.T) {
	cfg := config.DefaultConfig()
	cfg.MCP.Enabled = true
	if err := cfg.InstallPack(config.PackLifeSciences); err != nil {
		t.Fatal(err)
	}
	cfg.Packs.Enabled[config.PackLifeSciences] = true
	cfg.SyncAgentsFromPacks()
	mcp.SetAppConfig(cfg)

	h := NewHub()
	ch, err := NewCommandHandler(h)
	if err != nil {
		t.Fatal(err)
	}

	bioMCP, err := biologymcp.NewBiologyMCP()
	if err != nil {
		t.Fatal(err)
	}
	ollama := ai.NewOllamaProviderWithConfig("http://localhost:11434", "koesn/llama3-openbiollm-8b:latest")
	bioAgent := agent.NewBiologyAgent("BiologyExpert", ollama, h)
	bioAgent.MCPServer = bioMCP
	bioAgent.Info.ID = "bio-test-id"
	ch.runtimeAgents[bioAgent.Info.ID] = bioAgent

	if err := h.RegisterAgent(&bioAgent.Info); err != nil {
		t.Fatal(err)
	}
	channelName := "test-tools-channel"
	h.CreateChannelWithType(channelName, "test", "", protocol.ChannelTypeCustom, "user")
	if err := h.AddAgentToChannel(bioAgent.Info.ID, channelName); err != nil {
		t.Fatal(err)
	}

	msg := &protocol.Message{Channel: channelName}
	out, err := ch.handleToolsList(context.Background(), msg)
	if err != nil {
		t.Fatal(err)
	}
	text := out.Content
	if !strings.Contains(text, "analyze_sequence") {
		t.Fatalf("expected analyze_sequence in output: %s", text)
	}
	if !strings.Contains(text, "BiologyExpert") {
		t.Fatalf("expected agent name: %s", text)
	}
}

func TestListChannelToolCapabilitiesSkipsModerator(t *testing.T) {
	h := NewHub()
	ch, err := NewCommandHandler(h)
	if err != nil {
		t.Fatal(err)
	}
	channelName := "tools-mod-test"
	h.CreateChannelWithType(channelName, "test", "", protocol.ChannelTypeCustom, "system")
	mod := &protocol.AgentInfo{ID: "mod-1", Name: "ChatModerator", Type: protocol.AgentTypeModerator, Status: "active"}
	_ = h.RegisterAgent(mod)
	_ = h.AddAgentToChannel("mod-1", channelName)

	resp, err := ch.ListChannelToolCapabilities(channelName)
	if err != nil {
		t.Fatal(err)
	}
	for _, ag := range resp.Agents {
		if ag.AgentType == string(protocol.AgentTypeModerator) {
			t.Fatal("moderator should be skipped")
		}
	}
}

func TestListChannelToolCapabilitiesResolvesDMAgentWithoutJoin(t *testing.T) {
	cfg := config.DefaultConfig()
	cfg.MCP.Enabled = true
	if err := cfg.InstallPack(config.PackLifeSciences); err != nil {
		t.Fatal(err)
	}
	cfg.Packs.Enabled[config.PackLifeSciences] = true
	cfg.SyncAgentsFromPacks()
	mcp.SetAppConfig(cfg)

	h := NewHub()
	ch, err := NewCommandHandler(h)
	if err != nil {
		t.Fatal(err)
	}

	bioMCP, err := biologymcp.NewBiologyMCP()
	if err != nil {
		t.Fatal(err)
	}
	ollama := ai.NewOllamaProviderWithConfig("http://localhost:11434", "koesn/llama3-openbiollm-8b:latest")
	bioAgent := agent.NewBiologyAgent("BiologyExpert", ollama, h)
	bioAgent.MCPServer = bioMCP
	bioAgent.Info.ID = "bio-dm-id"
	ch.runtimeAgents[bioAgent.Info.ID] = bioAgent
	if err := h.RegisterAgent(&bioAgent.Info); err != nil {
		t.Fatal(err)
	}

	dmName := "dm-camron-biologyexpert"
	h.CreateChannelWithType(dmName, "Direct message with BiologyExpert", "", protocol.ChannelTypeDM, "camron")
	// Intentionally do not AddAgentToChannel — simulates hub restart before re-join.

	resp, err := ch.ListChannelToolCapabilities(dmName)
	if err != nil {
		t.Fatal(err)
	}
	if len(resp.Agents) != 1 {
		t.Fatalf("expected 1 agent, got %d", len(resp.Agents))
	}
	if resp.Agents[0].ToolCount == 0 {
		t.Fatalf("expected biology tools, got %+v", resp.Agents[0])
	}
}
