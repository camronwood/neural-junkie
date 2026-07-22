package hub

import (
	"context"
	"path/filepath"
	"testing"

	"github.com/camronwood/neural-junkie/internal/agent"
	"github.com/camronwood/neural-junkie/internal/ai"
	"github.com/camronwood/neural-junkie/internal/protocol"
)

func TestAddAgentToChannelSubscribesDiscoveryDisabledExpert(t *testing.T) {
	h := NewHub()
	expert := agent.NewAgent(protocol.AgentTypeArchitecture, "CustomArchitect", []string{"architecture"}, ai.NewMockProvider(), h)
	expert.Info.ID = "custom-expert-old"
	expert.DisableChannelDiscovery = true
	h.commandHandler.RegisterRuntimeAgent(expert)
	if err := h.RegisterAgent(&expert.Info); err != nil {
		t.Fatal(err)
	}

	dm := h.CreateChannelWithType(
		"dm-camron-customarchitect",
		"Direct message with CustomArchitect",
		"",
		protocol.ChannelTypeDM,
		"camron",
	)
	if err := expert.Start(context.Background(), dm.Name); err != nil {
		t.Fatal(err)
	}
	defer expert.Stop()

	custom := h.CreateChannelWithType("design-room", "Design discussion", "", protocol.ChannelTypeCustom, "camron")
	if err := h.AddAgentToChannel(expert.Info.ID, custom.Name); err != nil {
		t.Fatal(err)
	}

	msg := protocol.NewMessage(
		protocol.MessageTypeQuestion,
		custom.Name,
		protocol.AgentInfo{ID: "human-1", Name: "camron", Type: "human"},
		"@CustomArchitect are you here?",
	)
	if err := h.SendMessage(msg); err != nil {
		t.Fatal(err)
	}
	waitForAgentChatReply(t, h, custom.Name, expert.Info.ID)
}

func TestSessionRestoreRebindsCustomExpertByStableName(t *testing.T) {
	original := NewHub()
	oldInfo := &protocol.AgentInfo{
		ID: "old-runtime-id", Name: "CustomArchitect",
		Type: protocol.AgentTypeArchitecture, Status: "active",
	}
	if err := original.RegisterAgent(oldInfo); err != nil {
		t.Fatal(err)
	}
	custom := original.CreateChannelWithType("restored-design", "Design discussion", "", protocol.ChannelTypeCustom, "camron")
	if err := original.AddAgentToChannel(oldInfo.ID, custom.Name); err != nil {
		t.Fatal(err)
	}

	path := filepath.Join(t.TempDir(), "last-session.json")
	if err := original.SaveSessionToFile(path); err != nil {
		t.Fatal(err)
	}

	restored := NewHub()
	if err := restored.LoadSessionFromFile(path); err != nil {
		t.Fatal(err)
	}
	expert := agent.NewAgent(protocol.AgentTypeArchitecture, "CustomArchitect", []string{"architecture"}, ai.NewMockProvider(), restored)
	expert.Info.ID = "new-runtime-id"
	expert.DisableChannelDiscovery = true
	restored.commandHandler.RegisterRuntimeAgent(expert)
	if err := restored.RegisterAgent(&expert.Info); err != nil {
		t.Fatal(err)
	}
	dm := restored.CreateChannelWithType(
		"dm-camron-customarchitect",
		"Direct message with CustomArchitect",
		"",
		protocol.ChannelTypeDM,
		"camron",
	)
	if err := expert.Start(context.Background(), dm.Name); err != nil {
		t.Fatal(err)
	}
	defer expert.Stop()

	restored.RebindRestoredChannelMembers()

	var rebound *protocol.Channel
	for _, ch := range restored.ListChannels() {
		if ch != nil && ch.Name == custom.Name {
			rebound = ch
			break
		}
	}
	if rebound == nil {
		t.Fatalf("restored channel %q not found", custom.Name)
	}
	if len(rebound.Members) != 1 || rebound.Members[0] != expert.Info.ID {
		t.Fatalf("restored members = %v, want [%s]", rebound.Members, expert.Info.ID)
	}

	msg := protocol.NewMessage(
		protocol.MessageTypeQuestion,
		custom.Name,
		protocol.AgentInfo{ID: "human-1", Name: "camron", Type: "human"},
		"@CustomArchitect can you respond after restart?",
	)
	if err := restored.SendMessage(msg); err != nil {
		t.Fatal(err)
	}
	waitForAgentChatReply(t, restored, custom.Name, expert.Info.ID)
}
