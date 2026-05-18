package agent

import (
	"context"
	"encoding/json"
	"testing"

	"github.com/camronwood/neural-junkie/internal/protocol"
)

type imageGenTestHub struct {
	enabled bool
	posted  bool
	prompt  string
}

func (h *imageGenTestHub) SendMessage(msg *protocol.Message) error { return nil }
func (h *imageGenTestHub) BroadcastDirect(string, *protocol.Message) {}
func (h *imageGenTestHub) Subscribe(string) (chan *protocol.Message, error) {
	return make(chan *protocol.Message), nil
}
func (h *imageGenTestHub) GetMessages(string, int) ([]*protocol.Message, error) { return nil, nil }
func (h *imageGenTestHub) GetChannelAgents(string) ([]protocol.AgentInfo, error) { return nil, nil }
func (h *imageGenTestHub) GetThreadParentAuthor(string) string                   { return "" }
func (h *imageGenTestHub) GetCommandHandler() CommandHandlerInterface            { return nil }
func (h *imageGenTestHub) GetAgentChannels(string) []string                        { return nil }
func (h *imageGenTestHub) GetChannelType(string) protocol.ChannelType              { return protocol.ChannelTypePublic }
func (h *imageGenTestHub) ImageGenerationEnabled() bool                            { return h.enabled }
func (h *imageGenTestHub) GenerateAndPostImage(_ context.Context, _ string, _ protocol.AgentInfo, prompt, _ string) error {
	h.posted = true
	h.prompt = prompt
	return nil
}

func TestAgentToolDefinitionsIncludesGenerateImage(t *testing.T) {
	hub := &imageGenTestHub{enabled: true}
	a := &Agent{
		Info: protocol.AgentInfo{
			Name:                    "Frontend",
			Type:                    protocol.AgentTypeFrontend,
			SupportsImageGeneration: true,
		},
		Hub: hub,
	}
	tools := a.agentToolDefinitions()
	if len(tools) != 1 || tools[0].Name != generateImageToolName {
		t.Fatalf("expected generate_image tool, got %+v", tools)
	}
}

func TestExecuteGenerateImageTool(t *testing.T) {
	hub := &imageGenTestHub{enabled: true}
	a := &Agent{
		Info: protocol.AgentInfo{Name: "Frontend", Type: protocol.AgentTypeFrontend},
		Hub:  hub,
	}
	msg := &protocol.Message{Channel: "general"}
	input, _ := json.Marshal(map[string]string{"prompt": "a blue hexagon logo"})
	result, err := a.executeGenerateImageTool(context.Background(), msg, input)
	if err != nil {
		t.Fatal(err)
	}
	if !hub.posted || hub.prompt != "a blue hexagon logo" {
		t.Fatalf("expected image post, posted=%v prompt=%q", hub.posted, hub.prompt)
	}
	if result == "" {
		t.Fatal("expected non-empty tool result")
	}
}

func TestImageGenerationToolsDisabledWithoutFlag(t *testing.T) {
	hub := &imageGenTestHub{enabled: true}
	a := &Agent{
		Info: protocol.AgentInfo{Name: "Backend", Type: protocol.AgentTypeBackend},
		Hub:  hub,
	}
	if len(a.agentToolDefinitions()) != 0 {
		t.Fatalf("backend agent should not get image tools")
	}
}