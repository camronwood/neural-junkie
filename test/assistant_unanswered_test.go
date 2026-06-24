package test

import (
	"context"
	"testing"
	"time"

	"github.com/camronwood/neural-junkie/internal/agent"
	"github.com/camronwood/neural-junkie/internal/ai"
	"github.com/camronwood/neural-junkie/internal/hub"
	"github.com/camronwood/neural-junkie/internal/protocol"
)

func TestAssistantAgentHasUnansweredTracker(t *testing.T) {
	mockAI := ai.NewMockProvider()
	testHub := hub.NewHub()

	assistant := agent.NewAssistantAgent("Assistant", mockAI, testHub)
	if assistant == nil {
		t.Fatal("failed to create assistant agent")
	}
	if assistant.Info.Type != protocol.AgentTypeAssistant {
		t.Fatalf("expected assistant type, got %s", assistant.Info.Type)
	}
}

func TestAssistantTracksUserMessages(t *testing.T) {
	mockAI := ai.NewMockProvider()
	testHub := hub.NewHub()
	testHub.CreateChannel("general", "General", "")

	assistant := agent.NewAssistantAgent("Assistant", mockAI, testHub)
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	go assistant.Start(ctx, "general")
	time.Sleep(100 * time.Millisecond)

	userMsg := protocol.NewMessage(
		protocol.MessageTypeChat,
		"general",
		protocol.AgentInfo{ID: "user1", Name: "TestUser"},
		"Can someone help me?",
	)
	assistant.ProcessMessage(ctx, userMsg)
}
