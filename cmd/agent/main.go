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
	"strings"
	"syscall"
	"time"

	"github.com/camronwood/neural-junkie/internal/agent"
	"github.com/camronwood/neural-junkie/internal/ai"
	"github.com/camronwood/neural-junkie/internal/protocol"
)

var (
	agentType  = flag.String("type", "backend", "Agent type (frontend, backend, devops, database, security, rust, repo)")
	agentName  = flag.String("name", "", "Agent name (optional, will be auto-generated)")
	channel    = flag.String("channel", "general", "Channel to join")
	serverAddr = flag.String("server", "http://localhost:18765", "Chat hub server address")
	useMock    = flag.Bool("mock", false, "Use mock AI responses (set to true for testing without API calls)")
	repoPath   = flag.String("repo-path", "", "Repository path (required for repo type agents)")
	modelName  = flag.String("model", "", "Ollama model to use (overrides OLLAMA_MODEL env var)")
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
	data, err := json.Marshal(msg)
	if err != nil {
		return
	}
	resp, err := h.client.Post(h.baseURL+"/api/broadcast", "application/json", bytes.NewBuffer(data))
	if err != nil {
		return
	}
	resp.Body.Close()
}

func (h *httpHubClient) Subscribe(channelName string) (chan *protocol.Message, error) {
	// For HTTP client, we'll poll for new messages
	// In a real implementation, this would use WebSockets
	ch := make(chan *protocol.Message, 100)

	go func() {
		ticker := time.NewTicker(1 * time.Second)
		defer ticker.Stop()

		lastCheck := time.Now()
		seenMessages := make(map[string]bool) // Track message IDs we've already sent

		for range ticker.C {
			messages, err := h.GetMessages(channelName, 20)
			if err != nil {
				continue
			}

			// Send only new messages that we haven't seen before
			for _, msg := range messages {
				if seenMessages[msg.ID] {
					continue
				}
				// Only mark seen when we actually deliver; otherwise a message that
				// fails the timestamp gate would be dropped forever on the next poll.
				if msg.Timestamp.After(lastCheck) {
					seenMessages[msg.ID] = true
					ch <- msg
				}

				// Clean up old entries to prevent memory leak
				if len(seenMessages) > 100 {
					// Clear half of the map
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
	req, err := http.NewRequest(http.MethodGet, fmt.Sprintf("%s/api/messages?channel=%s&limit=%d", h.baseURL, channelName, limit), nil)
	if err != nil {
		return nil, err
	}
	if s := strings.TrimSpace(os.Getenv("NEURAL_JUNKIE_FULL_METADATA_SECRET")); s != "" {
		req.Header.Set("X-NJ-Full-Metadata", s)
	}
	resp, err := h.client.Do(req)
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
	// HTTP clients don't have direct access to command handler
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

func (h *httpHubClient) GetChannelSessionSummary(channel string) string { return "" }

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

func (h *httpHubClient) ImageGenerationEnabled() bool {
	return false
}

func (h *httpHubClient) GenerateAndPostImage(ctx context.Context, channel string, from protocol.AgentInfo, prompt, size string) error {
	return fmt.Errorf("image generation requires an in-process hub connection")
}

func main() {
	flag.Parse()

	// Validate agent type
	aType := protocol.AgentType(*agentType)
	validTypes := map[protocol.AgentType]bool{
		protocol.AgentTypeFrontend:  true,
		protocol.AgentTypeBackend:   true,
		protocol.AgentTypeDevOps:    true,
		protocol.AgentTypeDatabase:  true,
		protocol.AgentTypeSecurity:  true,
		protocol.AgentTypeRust:      true,
		protocol.AgentTypeBiology:   true,
		protocol.AgentTypeRepo:      true,
		protocol.AgentTypeAssistant: true,
	}

	if !validTypes[aType] {
		log.Fatalf("Invalid agent type: %s. Valid types: frontend, backend, devops, database, security, rust, biology, repo, assistant", *agentType)
	}

	// Validate repo path for repo agents
	if aType == protocol.AgentTypeRepo && *repoPath == "" {
		log.Fatalf("Repository path is required for repo agents. Use --repo-path flag")
	}

	// Generate name if not provided
	name := *agentName
	if name == "" {
		name = generateAgentName(aType)
	}

	// Create AI provider
	var aiProvider ai.AIProvider
	var err error

	if *useMock {
		aiProvider = ai.NewMockProvider()
		log.Println("Using mock AI provider")
	} else if *modelName != "" {
		// Explicit model override via --model flag
		aiProvider = ai.NewOllamaProviderWithConfig("", *modelName)
		log.Printf("Using Ollama AI provider (model: %s)", *modelName)
	} else {
		aiProvider, err = ai.NewOllamaProvider()
		if err != nil {
			log.Fatalf("Failed to create Ollama provider: %v", err)
		}
		log.Printf("Using Ollama AI provider (model: %s)", aiProvider.GetModel())
	}

	// Create hub client
	hubClient := newHTTPHubClient(*serverAddr)

	// Register agent with hub
	log.Printf("Creating %s agent: %s", aType, name)

	// Create specialized agent
	var agentInstance *agent.Agent
	var repoAgent *agent.RepoAgent

	if aType == protocol.AgentTypeRepo {
		// Create repository expert agent
		repoAgent, err = agent.NewRepoAgent(name, *repoPath, aiProvider, hubClient)
		if err != nil {
			log.Fatalf("Failed to create repo agent: %v", err)
		}
		agentInstance = repoAgent.Agent
	} else {
		// Create regular agent
		agentInstance, err = agent.AgentFactory(aType, name, aiProvider, hubClient)
		if err != nil {
			log.Fatalf("Failed to create agent: %v", err)
		}
	}

	// Register with hub
	registerData, _ := json.Marshal(agentInstance.Info)

	resp, err := http.Post(*serverAddr+"/api/agents", "application/json", bytes.NewBuffer(registerData))
	if err != nil {
		log.Printf("Warning: Failed to register with hub: %v", err)
	} else {
		resp.Body.Close()
	}

	// Join channel with greeting (single join message, no duplicate)
	greeting := fmt.Sprintf("👋 %s (%s) has joined the channel. I specialize in: %s",
		agentInstance.Info.Name,
		agentInstance.Info.Type,
		formatExpertise(agentInstance.Info.Expertise))
	joinData, _ := json.Marshal(map[string]interface{}{
		"agent_id": agentInstance.Info.ID,
		"channel":  *channel,
		"greeting": greeting,
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

	if aType == protocol.AgentTypeRepo && repoAgent != nil {
		if err := repoAgent.StartWithIndexing(ctx, *channel); err != nil {
			log.Fatalf("Failed to start repo agent: %v", err)
		}
	} else {
		if err := agentInstance.Start(ctx, *channel); err != nil {
			log.Fatalf("Failed to start agent: %v", err)
		}
	}

	log.Printf("✅ Agent %s is now active in channel: %s", name, *channel)
	log.Printf("Press Ctrl+C to stop")

	// Wait for interrupt signal
	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, os.Interrupt, syscall.SIGTERM)
	<-sigCh

	log.Println("Shutting down agent...")
	agentInstance.Stop()
}

func generateAgentName(agentType protocol.AgentType) string {
	names := map[protocol.AgentType][]string{
		protocol.AgentTypeFrontend: {"React Specialist", "Vue Expert", "UI/UX Master", "Frontend Guru"},
		protocol.AgentTypeBackend:  {"API Architect", "Backend Expert", "Microservices Pro", "Go Master"},
		protocol.AgentTypeDevOps:   {"DevOps Engineer", "Cloud Architect", "Infrastructure Expert", "CI/CD Specialist"},
		protocol.AgentTypeDatabase: {"Database Expert", "SQL Master", "Data Architect", "Query Optimizer"},
		protocol.AgentTypeSecurity: {"Security Expert", "InfoSec Specialist", "Cybersecurity Pro", "Auth Master"},
		protocol.AgentTypeRust:     {"Rust Expert", "Rust Architect", "Cargo Master", "Ownership Guru"},
		protocol.AgentTypeRepo:     {"Repo Expert", "Code Navigator", "Project Guide", "Codebase Oracle"},
	}

	nameList := names[agentType]
	if len(nameList) == 0 {
		return "Agent"
	}
	return nameList[time.Now().Unix()%int64(len(nameList))]
}

func formatExpertise(expertise []string) string {
	if len(expertise) == 0 {
		return "general development"
	}
	if len(expertise) <= 3 {
		return fmt.Sprintf("%v", expertise)
	}
	return fmt.Sprintf("%s, %s, %s and %d more", expertise[0], expertise[1], expertise[2], len(expertise)-3)
}
