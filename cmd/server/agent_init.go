package main

import (
	"context"
	"log"
	"strings"

	"github.com/camronwood/neural-junkie/internal/agent"
	"github.com/camronwood/neural-junkie/internal/ai"
	"github.com/camronwood/neural-junkie/internal/config"
	"github.com/camronwood/neural-junkie/internal/hub"
	"github.com/camronwood/neural-junkie/internal/protocol"
)

// initializeAssistantAgent creates and starts the system assistant agent

func initializeAssistantAgent() {
	log.Println("🤖 Initializing assistant agent...")

	aiProvider := assistantAIProvider()

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
			wireAgentWorkspaceBackend(assistant.Agent)
		}
	}

	// Join general channel with greeting
	if err := chatHub.JoinChannel(assistant.Info.ID, "general",
		"👋 Personal Assistant online! I can help with reminders, tasks, notes, platform questions, and more. Ask me '/help-assistant' to learn what I can do!"); err != nil {
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

func assistantAIProvider() ai.AIProvider {
	if appConfig != nil && globalProviderCache != nil {
		acfg := config.AgentConfig{
			Type:       "assistant",
			Name:       "Assistant",
			Enabled:    true,
			ProviderID: appConfig.AI.DefaultProviderID,
			Model:      config.UtilityOllamaModel,
		}
		for _, a := range appConfig.Agents {
			if strings.EqualFold(strings.TrimSpace(a.Type), "assistant") {
				acfg = a
				break
			}
		}
		if p, err := globalProviderCache.GetForAgent(appConfig, acfg); err == nil && p != nil {
			log.Printf("✅ Assistant provider from config (model: %s)", p.GetModel())
			return p
		}
	}
	ollamaProvider, err := ai.NewOllamaProvider()
	if err != nil {
		log.Printf("⚠️  Warning: Failed to initialize Ollama provider for assistant: %v", err)
		log.Println("⚠️  Using mock AI provider for assistant.")
		return ai.NewMockProvider()
	}
	log.Printf("✅ Ollama provider initialized for assistant (model: %s, endpoint: %s)", ollamaProvider.GetModel(), ollamaProvider.GetEndpoint())
	return ollamaProvider
}
