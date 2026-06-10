package main

import (
	"context"
	"log"
	"strings"

	"github.com/camronwood/neural-junkie/internal/agent"
	"github.com/camronwood/neural-junkie/internal/ai"
	"github.com/camronwood/neural-junkie/internal/hub"
	"github.com/camronwood/neural-junkie/internal/protocol"
)

func initializeModeratorAgent() {
	log.Println("🤖 Initializing moderator agent...")

	// Create AI provider for moderator
	var aiProvider ai.AIProvider
	ollamaProvider, err := ai.NewOllamaProvider()
	if err != nil {
		log.Printf("⚠️  Warning: Failed to initialize Ollama provider for moderator: %v", err)
		log.Println("⚠️  Using mock AI provider for moderator. Make sure Ollama is running on localhost:11434")
		aiProvider = ai.NewMockProvider()
	} else {
		aiProvider = ollamaProvider
		log.Printf("✅ Ollama provider initialized for moderator (model: %s)", ollamaProvider.GetModel())
	}

	// Create moderator agent
	moderator := agent.NewModeratorAgent("ChatModerator", aiProvider, chatHub)
	moderator.SetCollabClient(chatHub.NewCollaborationClientAdapter())

	// Register moderator with hub
	if err := chatHub.RegisterAgent(&moderator.Info); err != nil {
		log.Printf("❌ Failed to register moderator agent: %v", err)
		return
	}
	if commandHandler := chatHub.GetCommandHandler(); commandHandler != nil {
		if ch, ok := commandHandler.(*hub.CommandHandler); ok {
			ch.RegisterRuntimeAgent(moderator.Agent)
		}
	}

	// Join general channel with greeting
	if err := chatHub.JoinChannel(moderator.Info.ID, "general",
		"👋 ChatModerator online! I'm here to help with chat features and commands. Type @ChatModerator to ask me anything about using this chat system!"); err != nil {
		log.Printf("❌ Failed to join moderator to general channel: %v", err)
		return
	}

	// Start moderator in general channel
	ctx := context.Background()
	go func() {
		if err := moderator.Start(ctx, "general"); err != nil {
			log.Printf("❌ Failed to start moderator agent: %v", err)
			return
		}
	}()

	log.Println("✅ Moderator agent started successfully")
}

// initializeAssistantAgent creates and starts the system assistant agent

func initializeAssistantAgent() {
	log.Println("🤖 Initializing assistant agent...")

	// Create AI provider for assistant - use Ollama since Claude API key is invalid
	var aiProvider ai.AIProvider

	// Use Ollama for assistant since Claude API key is invalid
	ollamaProvider, err := ai.NewOllamaProvider()
	if err != nil {
		log.Printf("⚠️  Warning: Failed to initialize Ollama provider for assistant: %v", err)
		log.Println("⚠️  Using mock AI provider for assistant.")
		aiProvider = ai.NewMockProvider()
	} else {
		aiProvider = ollamaProvider
		log.Printf("✅ Ollama provider initialized for assistant (model: %s, endpoint: %s)", ollamaProvider.GetModel(), ollamaProvider.GetEndpoint())
	}

	// Create assistant agent
	assistant := agent.NewAssistantAgent("Assistant", aiProvider, chatHub)
	assistant.SetCollabClient(chatHub.NewCollaborationClientAdapter())

	// Register assistant with hub
	if err := chatHub.RegisterAgent(&assistant.Info); err != nil {
		log.Printf("❌ Failed to register assistant agent: %v", err)
		return
	}

	// Register assistant with command handler for meeting notes functionality
	if commandHandler := chatHub.GetCommandHandler(); commandHandler != nil {
		if ch, ok := commandHandler.(*hub.CommandHandler); ok {
			ch.SetAssistantAgent(assistant)
			ch.RegisterRuntimeAgent(assistant.Agent)
		}
	}

	// Join general channel with greeting
	if err := chatHub.JoinChannel(assistant.Info.ID, "general",
		"👋 Personal Assistant online! I can help with reminders, tasks, notes, and more. Ask me '/help-assistant' to learn what I can do!"); err != nil {
		log.Printf("❌ Failed to join assistant to general channel: %v", err)
		return
	}

	// Rebind assistant to restored DM channels after restart/session restore.
	for _, ch := range chatHub.ListChannels() {
		if ch == nil || ch.Type != protocol.ChannelTypeDM {
			continue
		}
		nameLower := strings.ToLower(ch.Name)
		descLower := strings.ToLower(ch.Description)
		if strings.Contains(nameLower, "assistant") || strings.Contains(descLower, "assistant") {
			if err := chatHub.JoinChannel(assistant.Info.ID, ch.Name); err != nil {
				log.Printf("⚠️  Failed to rejoin assistant to DM channel %s: %v", ch.Name, err)
			} else {
				log.Printf("✅ Assistant rejoined restored DM channel: %s", ch.Name)
			}
		}
	}

	// Start assistant in general channel
	ctx := context.Background()
	go func() {
		if err := assistant.Start(ctx, "general"); err != nil {
			log.Printf("❌ Failed to start assistant agent: %v", err)
			return
		}
	}()

	log.Println("✅ Assistant agent started successfully")
}
