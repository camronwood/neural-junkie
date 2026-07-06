package agent

import (
	"context"
	"encoding/json"
	"testing"

	"github.com/camronwood/neural-junkie/internal/protocol"
)

type musicGenTestHub struct {
	enabled bool
	posted  bool
	style   string
}

func (h *musicGenTestHub) SendMessage(*protocol.Message) error       { return nil }
func (h *musicGenTestHub) BroadcastDirect(string, *protocol.Message) {}
func (h *musicGenTestHub) Subscribe(string) (chan *protocol.Message, error) {
	return make(chan *protocol.Message), nil
}
func (h *musicGenTestHub) GetMessages(string, int) ([]*protocol.Message, error) { return nil, nil }
func (h *musicGenTestHub) GetChannelAgents(string) ([]protocol.AgentInfo, error) { return nil, nil }
func (h *musicGenTestHub) GetThreadParentAuthor(string) string                   { return "" }
func (h *musicGenTestHub) GetCommandHandler() CommandHandlerInterface            { return nil }
func (h *musicGenTestHub) GetAgentChannels(string) []string                        { return nil }
func (h *musicGenTestHub) GetChannelType(string) protocol.ChannelType              { return protocol.ChannelTypePublic }
func (h *musicGenTestHub) GetChannelSessionSummary(string) string                 { return "" }
func (h *musicGenTestHub) GetThreadMessages(string, int) ([]*protocol.Message, error) { return nil, nil }
func (h *musicGenTestHub) IsChannelHeld(string) bool                            { return false }
func (h *musicGenTestHub) ImageGenerationEnabled() bool                         { return false }
func (h *musicGenTestHub) GenerateAndPostImage(context.Context, string, protocol.AgentInfo, string, string) error {
	return nil
}
func (h *musicGenTestHub) MusicGenerationEnabled() bool { return h.enabled }
func (h *musicGenTestHub) GenerateAndPostMusic(_ context.Context, _ string, _ protocol.AgentInfo, req MusicGenerateRequest) error {
	h.posted = true
	h.style = req.StyleTags
	return nil
}
func (h *musicGenTestHub) ExtractAndPostMusicStems(context.Context, string, protocol.AgentInfo, MusicExtractRequest) error {
	return nil
}
func (h *musicGenTestHub) AskUserQuestion(string, string, string, string, []string) (string, error) {
	return "", nil
}


func TestAgentToolDefinitionsIncludesGenerateMusic(t *testing.T) {
	hub := &musicGenTestHub{enabled: true}
	a := &Agent{
		Info: protocol.AgentInfo{Name: "MusicExpert", Type: protocol.AgentTypeMusic},
		Hub:  hub,
	}
	names := map[string]bool{}
	for _, td := range a.agentToolDefinitions(nil) {
		names[td.Name] = true
	}
	if !names[generateMusicToolName] {
		t.Fatalf("expected generate_music tool, got %v", names)
	}
}

func TestExecuteGenerateMusicTool(t *testing.T) {
	hub := &musicGenTestHub{enabled: true}
	a := &Agent{
		Info:    protocol.AgentInfo{Name: "MusicExpert", Type: protocol.AgentTypeMusic},
		Hub:     hub,
		Context: &ConversationContext{CurrentChannel: "general"},
	}
	msg := &protocol.Message{Channel: "general"}
	input, _ := json.Marshal(map[string]string{
		"style_tags": "lo-fi chill",
		"lyrics":     "[Instrumental]",
	})
	if _, err := a.executeGenerateMusicTool(context.Background(), msg, input); err != nil {
		t.Fatal(err)
	}
	if !hub.posted || hub.style != "lo-fi chill" {
		t.Fatalf("expected music post, posted=%v style=%q", hub.posted, hub.style)
	}
}

func TestMusicGenerationDisabledForFrontend(t *testing.T) {
	hub := &musicGenTestHub{enabled: true}
	a := &Agent{
		Info: protocol.AgentInfo{Name: "Frontend", Type: protocol.AgentTypeFrontend},
		Hub:  hub,
	}
	for _, td := range a.agentToolDefinitions(nil) {
		if td.Name == generateMusicToolName {
			t.Fatal("frontend should not get generate_music")
		}
	}
}
