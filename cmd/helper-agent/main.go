package main

import (
	"bytes"
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/camronwood/neural-junkie/internal/agent"
	"github.com/camronwood/neural-junkie/internal/ai"
	"github.com/camronwood/neural-junkie/internal/protocol"
)

var (
	helperName = flag.String("name", "", "Helper agent name (required, e.g., 'day-one')")
	channel    = flag.String("channel", "general", "Channel to join")
	serverAddr = flag.String("server", "http://localhost:8080", "Chat hub server address")
	useMock    = flag.Bool("mock", false, "Use mock AI responses (set to true for testing without API calls)")
)

type httpHubClient struct {
	baseURL string
	client  *http.Client
}

func newHTTPHubClient(baseURL string) *httpHubClient {
	return &httpHubClient{
		baseURL: baseURL,
		client:  &http.Client{Timeout: 10 * time.Second},
	}
}

func (h *httpHubClient) SendMessage(msg *protocol.Message) error {
	data, err := json.Marshal(msg)
	if err != nil {
		return err
	}

	resp, err := h.client.Post(h.baseURL+"/api/send", "application/json", bytes.NewBuffer(data))
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("server returned status %d", resp.StatusCode)
	}

	return nil
}

func (h *httpHubClient) BroadcastDirect(channelName string, msg *protocol.Message) {
	_ = h.SendMessage(msg)
}

func (h *httpHubClient) Subscribe(channelName string) (chan *protocol.Message, error) {
	ch := make(chan *protocol.Message, 100)

	go func() {
		ticker := time.NewTicker(1 * time.Second)
		defer ticker.Stop()

		lastCheck := time.Now()
		seenMessages := make(map[string]bool)

		for range ticker.C {
			messages, err := h.GetMessages(channelName, 20)
			if err != nil {
				continue
			}

			for _, msg := range messages {
				if seenMessages[msg.ID] {
					continue
				}

				seenMessages[msg.ID] = true

				if msg.Timestamp.After(lastCheck) {
					ch <- msg
				}

				if len(seenMessages) > 100 {
					count := 0
					for id := range seenMessages {
						delete(seenMessages, id)
						count++
						if count >= 50 {
							break
						}
					}
				}
			}
			lastCheck = time.Now()
		}
	}()

	return ch, nil
}

func (h *httpHubClient) GetMessages(channelName string, limit int) ([]*protocol.Message, error) {
	resp, err := h.client.Get(fmt.Sprintf("%s/api/messages?channel=%s&limit=%d", h.baseURL, channelName, limit))
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	var messages []*protocol.Message
	if err := json.NewDecoder(resp.Body).Decode(&messages); err != nil {
		return nil, err
	}

	return messages, nil
}

func (h *httpHubClient) GetChannelAgents(channelName string) ([]protocol.AgentInfo, error) {
	resp, err := h.client.Get(fmt.Sprintf("%s/api/agents", h.baseURL))
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	var agents []protocol.AgentInfo
	if err := json.NewDecoder(resp.Body).Decode(&agents); err != nil {
		return nil, err
	}

	return agents, nil
}

func (h *httpHubClient) GetThreadParentAuthor(threadID string) string {
	resp, err := h.client.Get(fmt.Sprintf("%s/api/threads/%s/parent-author", h.baseURL, threadID))
	if err != nil {
		return ""
	}
	defer resp.Body.Close()

	var result map[string]string
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return ""
	}

	return result["author_id"]
}

func (h *httpHubClient) GetCommandHandler() agent.CommandHandlerInterface {
	return nil
}

func (h *httpHubClient) GetAgentChannels(agentID string) []string {
	resp, err := h.client.Get(fmt.Sprintf("%s/api/agent-channels?agent_id=%s", h.baseURL, agentID))
	if err != nil {
		return nil
	}
	defer resp.Body.Close()

	var result struct {
		Channels []string `json:"channels"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil
	}
	return result.Channels
}

func (h *httpHubClient) GetChannelType(channelName string) protocol.ChannelType {
	resp, err := h.client.Get(fmt.Sprintf("%s/api/channels", h.baseURL))
	if err != nil {
		return protocol.ChannelTypePublic
	}
	defer resp.Body.Close()

	var channels []struct {
		Name string               `json:"name"`
		Type protocol.ChannelType `json:"type"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&channels); err != nil {
		return protocol.ChannelTypePublic
	}

	for _, ch := range channels {
		if ch.Name == channelName {
			return ch.Type
		}
	}
	return protocol.ChannelTypePublic
}

func main() {
	flag.Parse()

	if *helperName == "" {
		log.Fatal("Helper agent name is required. Use --name flag (e.g., --name day-one)")
	}

	// Create storage manager
	storage, err := agent.NewHelperAgentStorage()
	if err != nil {
		log.Fatalf("Failed to create storage: %v", err)
	}

	// Load helper agent configuration
	log.Printf("Loading helper agent configuration: %s", *helperName)
	config, err := storage.LoadConfig(*helperName)
	if err != nil {
		// List available agents
		available, listErr := storage.ListConfigs()
		if listErr != nil {
			available = []string{"(unable to list available agents)"}
		}
		log.Fatalf("Failed to load helper agent config: %v\n"+
			"Available agents: %v\n"+
			"Make sure the helper agent exists at: ~/.neural-junkie/helpers/",
			err, available)
	}

	log.Printf("✓ Loaded config: %s", config.Name)
	log.Printf("  Description: %s", config.Description)
	log.Printf("  Expertise areas: %d", len(config.Expertise))
	log.Printf("  Keywords: %d", len(config.Keywords))
	log.Printf("  Knowledge path: %s", config.KnowledgePath)

	// Create AI provider
	var aiProvider ai.AIProvider

	if *useMock {
		aiProvider = ai.NewMockProvider()
		log.Println("Using mock AI provider")
	} else {
		aiProvider, err = ai.NewOllamaProvider()
		if err != nil {
			log.Fatalf("Failed to create Ollama provider: %v", err)
		}
		log.Println("Using Ollama AI provider")
	}

	// Create hub client
	hubClient := newHTTPHubClient(*serverAddr)

	// Create helper agent
	log.Printf("Creating helper agent: %s", config.Name)
	helperAgent, err := agent.NewHelperAgent(config, aiProvider, hubClient)
	if err != nil {
		log.Fatalf("Failed to create helper agent: %v", err)
	}

	log.Printf("✓ Loaded %d knowledge documents", len(helperAgent.Knowledge.Documents))

	// Register with hub
	registerData, _ := json.Marshal(helperAgent.Info)

	resp, err := http.Post(*serverAddr+"/api/agents", "application/json", bytes.NewBuffer(registerData))
	if err != nil {
		log.Printf("Warning: Failed to register with hub: %v", err)
	} else {
		resp.Body.Close()
	}

	// Join channel
	joinData, _ := json.Marshal(map[string]string{
		"agent_id": helperAgent.Info.ID,
		"channel":  *channel,
	})

	resp, err = http.Post(*serverAddr+"/api/channels/join", "application/json", bytes.NewBuffer(joinData))
	if err != nil {
		log.Printf("Warning: Failed to join channel: %v", err)
	} else {
		resp.Body.Close()
	}

	// Start agent
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	if err := helperAgent.Agent.Start(ctx, *channel); err != nil {
		log.Fatalf("Failed to start helper agent: %v", err)
	}

	// Send join message
	joinMsg := protocol.NewMessage(
		protocol.MessageTypeAgentJoin,
		*channel,
		helperAgent.Info,
		fmt.Sprintf("👋 %s has joined the channel. I'm here to help with: %s",
			helperAgent.Info.Name,
			formatExpertise(helperAgent.Info.Expertise)),
	)

	if err := hubClient.SendMessage(joinMsg); err != nil {
		log.Printf("Warning: Failed to send join message: %v", err)
	}

	log.Printf("✅ Helper agent '%s' is now active in channel: %s", config.Name, *channel)
	log.Printf("📚 Knowledge base loaded with %d documents", len(helperAgent.Knowledge.Documents))
	log.Printf("Press Ctrl+C to stop")

	// Wait for interrupt signal
	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, os.Interrupt, syscall.SIGTERM)
	<-sigCh

	log.Println("Shutting down helper agent...")
	helperAgent.Agent.Stop()
}

func formatExpertise(expertise []string) string {
	if len(expertise) == 0 {
		return "general topics"
	}
	if len(expertise) <= 3 {
		return fmt.Sprintf("%v", expertise)
	}
	return fmt.Sprintf("%s, %s, %s and %d more areas", expertise[0], expertise[1], expertise[2], len(expertise)-3)
}
