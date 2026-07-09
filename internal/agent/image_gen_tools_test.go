package agent

import (
	"context"
	"encoding/json"
	"strings"
	"testing"

	"github.com/camronwood/neural-junkie/internal/protocol"
)

type imageGenTestHub struct {
	enabled bool
	posted  bool
	prompt  string
}

func (h *imageGenTestHub) SendMessage(msg *protocol.Message) error   { return nil }
func (h *imageGenTestHub) BroadcastDirect(string, *protocol.Message) {}
func (h *imageGenTestHub) Subscribe(string) (chan *protocol.Message, error) {
	return make(chan *protocol.Message), nil
}
func (h *imageGenTestHub) GetMessages(string, int) ([]*protocol.Message, error)  { return nil, nil }
func (h *imageGenTestHub) GetChannelAgents(string) ([]protocol.AgentInfo, error) { return nil, nil }
func (h *imageGenTestHub) GetThreadParentAuthor(string) string                   { return "" }
func (h *imageGenTestHub) GetCommandHandler() CommandHandlerInterface            { return nil }
func (h *imageGenTestHub) GetAgentChannels(string) []string                      { return nil }
func (h *imageGenTestHub) GetChannelType(string) protocol.ChannelType {
	return protocol.ChannelTypePublic
}
func (h *imageGenTestHub) GetChannelSessionSummary(string) string { return "" }
func (h *imageGenTestHub) GetThreadMessages(string, int) ([]*protocol.Message, error) {
	return nil, nil
}
func (h *imageGenTestHub) IsChannelHeld(string) bool    { return false }
func (h *imageGenTestHub) ImageGenerationEnabled() bool { return h.enabled }
func (h *imageGenTestHub) GenerateAndPostImage(_ context.Context, _ string, _ protocol.AgentInfo, prompt, _ string) error {
	h.posted = true
	h.prompt = prompt
	return nil
}
func (h *imageGenTestHub) MusicGenerationEnabled() bool { return false }
func (h *imageGenTestHub) GenerateAndPostMusic(context.Context, string, protocol.AgentInfo, MusicGenerateRequest) error {
	return nil
}
func (h *imageGenTestHub) ExtractAndPostMusicStems(context.Context, string, protocol.AgentInfo, MusicExtractRequest) error {
	return nil
}
func (h *imageGenTestHub) AskUserQuestion(string, string, string, string, []string) (string, error) {
	return "", nil
}
func (h *imageGenTestHub) RequestToolApproval(string, string, string, string, map[string]interface{}) (bool, error) {
	return true, nil
}

func TestAgentToolDefinitionsIncludesGenerateImage(t *testing.T) {
	hub := &imageGenTestHub{enabled: true}
	a := &Agent{
		Info: protocol.AgentInfo{
			Name: "Frontend",
			Type: protocol.AgentTypeFrontend,
		},
		Hub: hub,
	}
	tools := a.agentToolDefinitions(nil)
	names := map[string]bool{}
	for _, td := range tools {
		names[td.Name] = true
	}
	if !names[generateImageToolName] {
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

func TestImageGenerationToolsEnabledForBackend(t *testing.T) {
	hub := &imageGenTestHub{enabled: true}
	a := &Agent{
		Info: protocol.AgentInfo{Name: "Backend", Type: protocol.AgentTypeBackend},
		Hub:  hub,
	}
	tools := a.agentToolDefinitions(nil)
	for _, td := range tools {
		if td.Name == generateImageToolName {
			return
		}
	}
	t.Fatalf("backend agent should get generate_image tool, got %+v", tools)
}

func TestTryHubImageGenerationShortcutSkippedForDeliveryMessage(t *testing.T) {
	hub := &imageGenTestHub{enabled: true}
	a := &Agent{
		Info: protocol.AgentInfo{Name: "Assistant", Type: protocol.AgentTypeAssistant},
		Hub:  hub,
	}
	msg := &protocol.Message{
		Channel: "dm-camron-assistant",
		Content: protocol.GeneratedImageDeliveryContent,
		Metadata: map[string]interface{}{
			"generated_image": map[string]interface{}{"mime": "image/png"},
		},
	}
	if _, ok := a.tryHubImageGenerationShortcut(context.Background(), msg); ok {
		t.Fatal("shortcut should not run on hub image delivery posts")
	}
	if hub.posted {
		t.Fatal("image should not be posted for delivery message")
	}
}

func TestTryHubImageGenerationShortcut(t *testing.T) {
	hub := &imageGenTestHub{enabled: true}
	a := &Agent{
		Info: protocol.AgentInfo{Name: "Assistant", Type: protocol.AgentTypeAssistant},
		Hub:  hub,
	}
	msg := &protocol.Message{Channel: "general", Content: "Can you generate me an image of a ship?"}
	resp, ok := a.tryHubImageGenerationShortcut(context.Background(), msg)
	if !ok {
		t.Fatal("expected shortcut to run")
	}
	if !hub.posted {
		t.Fatal("expected image post")
	}
	if !strings.Contains(resp, "posted") {
		t.Fatalf("unexpected response: %q", resp)
	}
}

func TestImageGenerationDisabledDuringImplementationSession(t *testing.T) {
	hub := &imageGenTestHub{enabled: true}
	a := &Agent{
		Info: protocol.AgentInfo{Name: "SoftwareArchitect", Type: protocol.AgentTypeArchitecture},
		Hub:  hub,
	}
	msg := &protocol.Message{
		Metadata: map[string]interface{}{
			protocol.IdeMetaImplementationSession: true,
			MetadataConversationMode:              ConversationModeCode,
			protocol.IdeMetaEditorMode:            "agent",
		},
	}
	if a.imageGenerationToolsEnabledForMessage(msg) {
		t.Fatal("image gen should be disabled during implementation session")
	}
	tools := a.agentToolDefinitions(msg)
	for _, td := range tools {
		if td.Name == generateImageToolName {
			t.Fatal("generate_image should not be in tool list during implementation session")
		}
	}
}

func TestImageGenerationToolsDisabledDuringCollabPlanning(t *testing.T) {
	hub := &imageGenTestHub{enabled: true}
	a := &Agent{
		Info: protocol.AgentInfo{Name: "SoftwareArchitect", Type: protocol.AgentTypeArchitecture},
		Hub:  hub,
	}
	msg := protocol.NewMessage(
		protocol.MessageTypeCollabDiscussion,
		"collab-test",
		protocol.AgentInfo{ID: "system", Name: "System"},
		"@SoftwareArchitect propose tasks",
	)
	msg.SetCollaborationPhase("planning")
	if a.imageGenerationToolsEnabledForMessage(msg) {
		t.Fatal("image gen should be disabled during collab planning")
	}
	for _, td := range a.agentToolDefinitions(msg) {
		if td.Name == generateImageToolName {
			t.Fatal("generate_image should not be in tool list during collab planning")
		}
	}
}

func TestTryHubImageGenerationShortcutSkippedDuringImplementationSession(t *testing.T) {
	hub := &imageGenTestHub{enabled: true}
	a := &Agent{
		Info: protocol.AgentInfo{Name: "SoftwareArchitect", Type: protocol.AgentTypeArchitecture},
		Hub:  hub,
	}
	msg := &protocol.Message{
		Channel: "general",
		Content: "Can you generate me an image of a ship?",
		Metadata: map[string]interface{}{
			protocol.IdeMetaImplementationSession: true,
			MetadataConversationMode:              ConversationModeCode,
		},
	}
	if _, ok := a.tryHubImageGenerationShortcut(context.Background(), msg); ok {
		t.Fatal("image shortcut should not run during implementation session")
	}
	if hub.posted {
		t.Fatal("image should not be posted during implementation session")
	}
}

func TestExecuteGenerateImageToolBlockedDuringCodeMode(t *testing.T) {
	hub := &imageGenTestHub{enabled: true}
	a := &Agent{
		Info: protocol.AgentInfo{Name: "Frontend", Type: protocol.AgentTypeFrontend},
		Hub:  hub,
	}
	msg := &protocol.Message{
		Channel:  "general",
		Metadata: map[string]interface{}{MetadataConversationMode: ConversationModeCode},
	}
	input, _ := json.Marshal(map[string]string{"prompt": "logo"})
	if _, err := a.executeGenerateImageTool(context.Background(), msg, input); err == nil {
		t.Fatal("expected error when generate_image called during code mode")
	}
	if hub.posted {
		t.Fatal("image should not be posted")
	}
}

func TestImageGenerationToolsDisabledForCLI(t *testing.T) {
	hub := &imageGenTestHub{enabled: true}
	a := &Agent{
		Info: protocol.AgentInfo{
			Name:                    "Cursor",
			Type:                    protocol.AgentTypeCLI,
			SupportsImageGeneration: false,
		},
		Hub: hub,
	}
	tools := a.agentToolDefinitions(nil)
	for _, td := range tools {
		if td.Name == generateImageToolName {
			t.Fatalf("CLI agent should not get hub image tools, got %+v", tools)
		}
	}
}
