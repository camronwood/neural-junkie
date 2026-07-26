package agent

import (
	"context"
	"encoding/json"
	"strings"
	"testing"

	"github.com/camronwood/neural-junkie/internal/ai"
	"github.com/camronwood/neural-junkie/internal/protocol"
)

type imageGenTestHub struct {
	hubArenaNoop
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

func TestAgentToolDefinitionsSuppressesToolsDuringCollabPlanning(t *testing.T) {
	a := &Agent{
		Info: protocol.AgentInfo{Name: "SoftwareArchitect", Type: protocol.AgentTypeArchitecture},
		Hub:  &imageGenTestHub{enabled: true},
	}
	msg := protocol.NewMessage(
		protocol.MessageTypeCollabDiscussion,
		"collab-test",
		protocol.AgentInfo{ID: "system", Name: "System", Type: protocol.AgentTypeGeneral},
		"Propose a minimal plan.",
	)
	msg.SetCollaborationID("550e8400-e29b-41d4-a716-446655440000")
	msg.SetCollaborationPhase("planning")

	if tools := a.agentToolDefinitions(msg); len(tools) != 0 {
		t.Fatalf("collaboration planning must not expose tools, got %+v", tools)
	}
}

func TestExecuteGenerateImageTool(t *testing.T) {
	hub := &imageGenTestHub{enabled: true}
	a := &Agent{
		Info: protocol.AgentInfo{Name: "Frontend", Type: protocol.AgentTypeFrontend},
		Hub:  hub,
	}
	msg := &protocol.Message{Channel: "general", Content: "generate an image of a blue hexagon logo"}
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

func TestTryHubImageGenerationShortcut_IndirectCoverRequest(t *testing.T) {
	hub := &imageGenTestHub{enabled: true}
	a := &Agent{
		Info: protocol.AgentInfo{Name: "BookWriter", Type: protocol.AgentTypeExpert},
		Hub:  hub,
	}
	msg := protocol.NewMessage(
		protocol.MessageTypeQuestion,
		"dm-camron-bookwriter",
		protocol.AgentInfo{ID: "user", Name: "Camron", Type: protocol.AgentTypeGeneral},
		"ok lets see what a sample outline and cover art image will look like",
	)
	goal := deriveTurnGoal(a, msg, IntentTask)
	if goal.Action != ActionImage {
		t.Fatalf("action = %q, want image", goal.Action)
	}
	ctx := contextWithTurnGoal(context.Background(), goal)
	if _, ok := a.tryHubImageGenerationShortcut(ctx, msg); !ok {
		t.Fatal("expected indirect cover request to use image shortcut")
	}
	if !hub.posted {
		t.Fatal("expected cover image to be posted")
	}
}

func TestCompleteMixedImageResponseIncludesTextDeliverable(t *testing.T) {
	a := &Agent{}
	msg := &protocol.Message{Content: "Show me a sample outline and cover art image."}
	got := a.completeMixedImageResponse(
		context.Background(),
		msg,
		"Write the requested outline.",
		nil,
		ai.NewMockProvider(),
		"Done — I've posted the generated image to the channel.",
	)
	if !strings.Contains(got, "mock response") || !strings.Contains(got, "posted the generated image") {
		t.Fatalf("mixed response did not preserve both deliverables: %q", got)
	}

	imageOnly := &protocol.Message{Content: "Generate cover art for the book."}
	const imageResponse = "Done — image posted."
	if got := a.completeMixedImageResponse(
		context.Background(), imageOnly, "prompt", nil, ai.NewMockProvider(), imageResponse,
	); got != imageResponse {
		t.Fatalf("image-only response unexpectedly called companion model: %q", got)
	}
}

func TestImageGenerationDisabledForChatThemeAsk(t *testing.T) {
	hub := &imageGenTestHub{enabled: true}
	a := &Agent{
		Info: protocol.AgentInfo{Name: "BackendEngineer", Type: protocol.AgentTypeBackend},
		Hub:  hub,
	}
	msg := &protocol.Message{
		Channel: "dm-test",
		Content: "I want to add theme support to this app I am working on now",
		Metadata: map[string]interface{}{
			MetadataConversationMode: ConversationModeChat,
		},
	}
	if a.imageGenerationToolsEnabledForMessage(msg) {
		t.Fatal("theme support chat ask must not expose generate_image")
	}
	vis := &protocol.Message{
		Channel: "dm-test",
		Content: "can you see my workspace I have open?",
		Metadata: map[string]interface{}{
			MetadataConversationMode: ConversationModeCode,
		},
	}
	if a.imageGenerationToolsEnabledForMessage(vis) {
		t.Fatal("workspace visibility must not expose generate_image")
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

func TestExplicitImageIntentOverridesAmbientIDEMetadata(t *testing.T) {
	hub := &imageGenTestHub{enabled: true}
	a := &Agent{
		Info: protocol.AgentInfo{Name: "Frontend", Type: protocol.AgentTypeFrontend},
		Hub:  hub,
	}
	msg := &protocol.Message{
		Channel: "general",
		Content: "Generate an image of a blue ship.",
		Metadata: map[string]interface{}{
			MetadataConversationMode:   ConversationModeCode,
			protocol.IdeMetaEditorMode: "agent",
		},
	}
	if !a.imageGenerationToolsEnabledForMessage(msg) {
		t.Fatal("explicit image intent should override ambient code/editor metadata")
	}
	if _, ok := a.tryHubImageGenerationShortcut(context.Background(), msg); !ok {
		t.Fatal("expected explicit image shortcut")
	}
	if !hub.posted {
		t.Fatal("expected image to be posted")
	}
}

func TestExplicitImageIntentDoesNotOverrideActiveCollaboration(t *testing.T) {
	a := &Agent{
		Info: protocol.AgentInfo{Name: "Frontend", Type: protocol.AgentTypeFrontend},
		Hub:  &imageGenTestHub{enabled: true},
	}
	msg := &protocol.Message{Channel: "collab", Content: "Generate an image of the design."}
	msg.SetCollaborationID("collab-1")
	msg.SetCollaborationPhase("executing")
	if a.imageGenerationToolsEnabledForMessage(msg) {
		t.Fatal("active collaboration must retain control of explicit image requests")
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

	nudge := protocol.NewMessage(
		protocol.MessageTypeQuestion,
		"collab-test",
		protocol.AgentInfo{ID: "user", Name: "User"},
		"@BackendEngineer please post your planning turn with concrete Task lines.",
	)
	nudge.SetCollaborationPhase("planning")
	if a.imageGenerationToolsEnabledForMessage(nudge) {
		t.Fatal("image gen must be disabled for planning-phase question nudges")
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
