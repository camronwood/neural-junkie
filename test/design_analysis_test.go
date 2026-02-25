package test

import (
	"context"
	"testing"

	"github.com/camronwood/neural-junkie/internal/agent"
	"github.com/camronwood/neural-junkie/internal/ai"
	"github.com/camronwood/neural-junkie/internal/protocol"
)

// MockHubClient implements the HubClient interface for testing
type MockHubClient struct {
	messages []*protocol.Message
}

func (m *MockHubClient) SendMessage(msg *protocol.Message) error {
	m.messages = append(m.messages, msg)
	return nil
}

func (m *MockHubClient) Subscribe(channelName string) (chan *protocol.Message, error) {
	return make(chan *protocol.Message), nil
}

func (m *MockHubClient) GetMessages(channelName string, limit int) ([]*protocol.Message, error) {
	return m.messages, nil
}

func (m *MockHubClient) GetChannelAgents(channelName string) ([]protocol.AgentInfo, error) {
	return []protocol.AgentInfo{}, nil
}

func (m *MockHubClient) GetThreadParentAuthor(threadID string) string {
	return ""
}

func (m *MockHubClient) GetCommandHandler() agent.CommandHandlerInterface {
	return nil
}

func (m *MockHubClient) BroadcastDirect(channelName string, msg *protocol.Message) {}
func (m *MockHubClient) GetAgentChannels(agentID string) []string { return []string{"general"} }
func (m *MockHubClient) GetChannelType(channelName string) protocol.ChannelType { return protocol.ChannelTypePublic }

func TestDesignAnalysis(t *testing.T) {
	// Create mock AI provider
	aiProvider := ai.NewMockProvider()

	// Create mock hub client
	hubClient := &MockHubClient{}

	// Create frontend agent
	agent := agent.NewFrontendAgent("FrontendExpert", aiProvider, hubClient)

	// Create a test message with design analysis metadata
	msg := &protocol.Message{
		ID:      "test-msg-1",
		Type:    protocol.MessageTypeChat,
		Channel: "test-channel",
		From: protocol.AgentInfo{
			ID:   "user-1",
			Name: "TestUser",
			Type: protocol.AgentTypeGeneral,
		},
		Content: "Please analyze this design mockup and generate a CSS style guide with HTML demo.",
		Metadata: map[string]interface{}{
			"design_analysis": true,
			"image_data":      []byte("fake-base64-image-data"),
			"image_type":      "image/png",
		},
	}

	// Test that the agent should respond to design analysis
	if !agent.ShouldRespond(msg) {
		t.Error("Agent should respond to design analysis requests")
	}

	// Test design analysis response generation
	ctx := context.Background()
	response, err := agent.GenerateResponse(ctx, msg)
	if err != nil {
		t.Errorf("Design analysis failed: %v", err)
	}

	if response == "" {
		t.Error("Design analysis should return a non-empty response")
	}

	// Verify that a design output message was sent
	if len(hubClient.messages) == 0 {
		t.Error("Expected design output message to be sent")
	}

	designMsg := hubClient.messages[0]
	if designMsg.Type != protocol.MessageTypeDesignOutput {
		t.Errorf("Expected design_output message type, got %s", designMsg.Type)
	}
}

func TestImageValidation(t *testing.T) {
	// Test image size validation
	largeImageData := make([]byte, 6*1024*1024) // 6MB, exceeds limit

	msg := &protocol.Message{
		ID:      "test-msg-2",
		Type:    protocol.MessageTypeChat,
		Channel: "test-channel",
		From: protocol.AgentInfo{
			ID:   "user-1",
			Name: "TestUser",
			Type: protocol.AgentTypeGeneral,
		},
		Content: "Analyze this large image",
		Metadata: map[string]interface{}{
			"design_analysis": true,
			"image_data":      largeImageData,
			"image_type":      "image/png",
		},
	}

	// The agent should still respond, but the command handler should validate size
	// This test just ensures the agent can handle large image metadata
	aiProvider := ai.NewMockProvider()
	hubClient := &MockHubClient{}
	agent := agent.NewFrontendAgent("FrontendExpert", aiProvider, hubClient)

	if !agent.ShouldRespond(msg) {
		t.Error("Agent should respond to design analysis requests regardless of image size")
	}
}
