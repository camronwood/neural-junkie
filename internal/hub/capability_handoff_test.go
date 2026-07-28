package hub

import (
	"context"
	"strings"
	"testing"

	"github.com/camronwood/neural-junkie/internal/agent"
	"github.com/camronwood/neural-junkie/internal/config"
	"github.com/camronwood/neural-junkie/internal/delegation"
	"github.com/camronwood/neural-junkie/internal/protocol"
)

type handoffTestProvider struct{}

func (handoffTestProvider) GenerateResponse(context.Context, string, []protocol.Message) (string, error) {
	return "verified helper result", nil
}
func (handoffTestProvider) GenerateVisionResponse(context.Context, string, []byte, string, []protocol.Message) (string, error) {
	return "", nil
}
func (handoffTestProvider) GetModel() string { return "test" }

func TestCapabilityHandoffCreatesReturnsAndArchivesRoom(t *testing.T) {
	t.Setenv("HOME", t.TempDir())
	config.SetupTestOfficialPackCatalog(t)
	cfg := config.DefaultConfig()
	config.InstallTestPack(t, cfg, config.PackSoftwareDevelopment)
	config.InstallTestPack(t, cfg, config.PackWebBrowser)
	if err := cfg.SetPackEnabled(config.PackSoftwareDevelopment, true); err != nil {
		t.Fatal(err)
	}
	if err := cfg.SetPackEnabled(config.PackWebBrowser, true); err != nil {
		t.Fatal(err)
	}
	config.SetAppConfig(cfg)
	t.Cleanup(func() { config.SetAppConfig(nil) })

	h := NewHub()
	h.commandHandler.appConfig = cfg
	requester := agent.NewAgent(protocol.AgentTypeArchitecture, "Architect", nil, handoffTestProvider{}, h)
	requester.Info.ID = "requester"
	helper := agent.NewAgent(protocol.AgentTypeAssistant, "Assistant", nil, handoffTestProvider{}, h)
	helper.Info.ID = "assistant"
	if err := h.RegisterAgent(&requester.Info); err != nil {
		t.Fatal(err)
	}
	if err := h.RegisterAgent(&helper.Info); err != nil {
		t.Fatal(err)
	}
	h.commandHandler.runtimeAgents[requester.Info.ID] = requester
	h.commandHandler.runtimeAgents[helper.Info.ID] = helper
	directory := h.commandHandler.CapabilityDirectory(requester.Info.ID)
	if len(directory) == 0 {
		t.Fatal("expected capability-to-agent directory")
	}

	result, err := h.commandHandler.RequestCapabilityHelp(context.Background(), delegation.CapabilityHelpRequest{
		FromID:          requester.Info.ID,
		FromName:        requester.Info.Name,
		CreatedBy:       "camron",
		CapabilityID:    "web-browser",
		Task:            "Verify the page accessibility",
		SourceChannel:   "general",
		SourceMessageID: "message-1",
	})
	if err != nil {
		t.Fatal(err)
	}
	if result.Status != "completed" || result.Summary != "verified helper result" {
		t.Fatalf("result=%+v", result)
	}
	room, err := h.GetChannel(result.Channel)
	if err != nil {
		t.Fatal(err)
	}
	if !room.Archived || room.SourceChannel != "general" || room.Type != protocol.ChannelTypeDelegation {
		t.Fatalf("room=%+v", room)
	}
	for _, listed := range h.ListChannels() {
		if listed.Name == result.Channel {
			t.Fatal("archived handoff room must be hidden")
		}
	}
	for _, active := range h.GetAgentChannels(helper.Info.ID) {
		if active == result.Channel {
			t.Fatal("archived handoff room must not remain an active agent channel")
		}
	}
	messages, err := h.GetMessages("general", 20)
	if err != nil {
		t.Fatal(err)
	}
	var started, completed bool
	for _, msg := range messages {
		event, _ := msg.Metadata["handoff_event"].(string)
		started = started || event == "handoff_started"
		completed = completed || event == "handoff_completed"
	}
	if !started || !completed {
		t.Fatalf("missing lifecycle events: started=%v completed=%v", started, completed)
	}
}

func TestCapabilityHandoffRejectsRecursion(t *testing.T) {
	cfg := config.DefaultConfig()
	h := NewHub()
	h.commandHandler.appConfig = cfg
	_, err := h.commandHandler.RequestCapabilityHelp(context.Background(), delegation.CapabilityHelpRequest{
		Depth: 1, CapabilityID: "web-browser", Task: "nested", SourceChannel: "general",
	})
	if err == nil {
		t.Fatal("expected depth rejection")
	}
}

func TestCapabilityHandoffRejectsVagueTask(t *testing.T) {
	cfg := config.DefaultConfig()
	h := NewHub()
	h.commandHandler.appConfig = cfg
	_, err := h.commandHandler.RequestCapabilityHelp(context.Background(), delegation.CapabilityHelpRequest{
		FromID: "requester", FromName: "Architect", CapabilityID: "web-browser",
		Task: "debugging a failing pod or reviewing your CI/CD pipeline security", SourceChannel: "general",
	})
	if err == nil {
		t.Fatal("expected vague-task rejection")
	}
	if !strings.Contains(err.Error(), "topic menu") && !strings.Contains(err.Error(), "bounded task") {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestCapabilityHandoffExplainsWhenNoHelperIsRunning(t *testing.T) {
	h := NewHub()
	h.commandHandler.appConfig = config.DefaultConfig()
	_, err := h.commandHandler.RequestCapabilityHelp(context.Background(), delegation.CapabilityHelpRequest{
		FromID: "requester", FromName: "Architect", CapabilityID: "web-browser",
		Task: "Inspect the page", SourceChannel: "general",
	})
	if err == nil {
		t.Fatal("expected no-helper error")
	}
}
