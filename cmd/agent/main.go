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
	"sync"
	"syscall"
	"time"

	"github.com/camronwood/neural-junkie/internal/agent"
	"github.com/camronwood/neural-junkie/internal/ai"
	"github.com/camronwood/neural-junkie/internal/hub/wsclient"
	"github.com/camronwood/neural-junkie/internal/protocol"
)

var (
	agentType  = flag.String("type", "backend", "Agent type (frontend, backend, devops, security, architecture, code-review, database, rust, biology, repo)")
	agentName  = flag.String("name", "", "Agent name (optional, will be auto-generated)")
	channel    = flag.String("channel", "general", "Channel to join")
	serverAddr = flag.String("server", "http://localhost:18765", "Chat hub server address")
	useMock    = flag.Bool("mock", false, "Use mock AI responses (set to true for testing without API calls)")
	repoPath   = flag.String("repo-path", "", "Repository path (required for repo type agents)")
	modelName  = flag.String("model", "", "Ollama model to use (overrides OLLAMA_MODEL env var)")
	usePoll    = flag.Bool("poll", false, "Use HTTP polling instead of WebSocket for hub messages")
	apiKey     = flag.String("api-key", "", "Hub API key (Bearer nj_...) for automation")
)

type httpHubClient struct {
	baseURL     string
	client      *http.Client
	agentID     string
	apiKey      string
	usePoll     bool
	stopWS      chan struct{}
	holdMu      sync.RWMutex
	channelHeld map[string]bool

	agentWSMu sync.Mutex
	agentWSCh chan *protocol.Message
	agentSubs map[string][]chan *protocol.Message
}

func newHTTPHubClient(baseURL string, usePoll bool, apiKey string) *httpHubClient {
	return &httpHubClient{
		baseURL:     baseURL,
		client:      &http.Client{Timeout: 10 * time.Second},
		apiKey:      strings.TrimSpace(apiKey),
		usePoll:     usePoll,
		stopWS:      make(chan struct{}),
		channelHeld: make(map[string]bool),
		agentSubs:   make(map[string][]chan *protocol.Message),
	}
}

func (h *httpHubClient) SetAgentID(id string) {
	h.agentID = strings.TrimSpace(id)
}

func (h *httpHubClient) setChannelHeld(channel string, held bool) {
	h.holdMu.Lock()
	defer h.holdMu.Unlock()
	if h.channelHeld == nil {
		h.channelHeld = make(map[string]bool)
	}
	if held {
		h.channelHeld[channel] = true
	} else {
		delete(h.channelHeld, channel)
	}
}

func (h *httpHubClient) IsChannelHeld(channel string) bool {
	h.holdMu.RLock()
	defer h.holdMu.RUnlock()
	return h.channelHeld[channel]
}

func (h *httpHubClient) authRequest(req *http.Request) {
	if h.apiKey != "" {
		req.Header.Set("Authorization", "Bearer "+h.apiKey)
	}
}

func (h *httpHubClient) doGet(url string) (*http.Response, error) {
	req, err := http.NewRequest(http.MethodGet, url, nil)
	if err != nil {
		return nil, err
	}
	h.authRequest(req)
	return h.client.Do(req)
}

func (h *httpHubClient) doPost(url, contentType string, body []byte) (*http.Response, error) {
	req, err := http.NewRequest(http.MethodPost, url, bytes.NewReader(body))
	if err != nil {
		return nil, err
	}
	if contentType != "" {
		req.Header.Set("Content-Type", contentType)
	}
	h.authRequest(req)
	return h.client.Do(req)
}

func (h *httpHubClient) SendMessage(msg *protocol.Message) error {
	data, err := json.Marshal(msg)
	if err != nil {
		return err
	}

	resp, err := h.doPost(h.baseURL+"/api/send", "application/json", data)
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
	resp, err := h.doPost(h.baseURL+"/api/broadcast", "application/json", data)
	if err != nil {
		return
	}
	resp.Body.Close()
}

func (h *httpHubClient) Subscribe(channelName string) (chan *protocol.Message, error) {
	if !h.usePoll && h.agentID != "" {
		return h.subscribeAgentChannel(channelName)
	}
	if !h.usePoll {
		return wsclient.Subscribe(h.baseURL, channelName, h.stopWS)
	}

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
				if msg.Type == protocol.MessageTypeAgentStatus && msg.Metadata != nil {
					if v, ok := msg.Metadata[protocol.MetadataChannelHold].(bool); ok {
						h.setChannelHeld(channelName, v)
					}
				}
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

func (h *httpHubClient) subscribeAgentChannel(channelName string) (chan *protocol.Message, error) {
	h.agentWSMu.Lock()
	defer h.agentWSMu.Unlock()

	if h.agentWSCh == nil {
		onHold := func(channel string, held bool) {
			h.setChannelHeld(channel, held)
		}
		feed, err := wsclient.SubscribeAgent(h.baseURL, h.agentID, h.stopWS, onHold)
		if err != nil {
			return nil, err
		}
		h.agentWSCh = feed
		go h.fanOutAgentMessages()
	}

	out := make(chan *protocol.Message, 512)
	h.agentSubs[channelName] = append(h.agentSubs[channelName], out)
	return out, nil
}

func (h *httpHubClient) fanOutAgentMessages() {
	for msg := range h.agentWSCh {
		if msg == nil {
			break
		}
		h.agentWSMu.Lock()
		subs := append([]chan *protocol.Message(nil), h.agentSubs[msg.Channel]...)
		h.agentWSMu.Unlock()
		for _, sub := range subs {
			select {
			case sub <- msg:
			default:
			}
		}
	}
}

func (h *httpHubClient) GetMessages(channelName string, limit int) ([]*protocol.Message, error) {
	req, err := http.NewRequest(http.MethodGet, fmt.Sprintf("%s/api/messages?channel=%s&limit=%d", h.baseURL, channelName, limit), nil)
	if err != nil {
		return nil, err
	}
	h.authRequest(req)
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

func (h *httpHubClient) GetChannelMessagesMerged(channelName string, limit int) ([]*protocol.Message, error) {
	url := fmt.Sprintf("%s/api/channel-export?channel=%s&format=json", h.baseURL, channelName)
	req, err := http.NewRequest(http.MethodGet, url, nil)
	if err != nil {
		return nil, err
	}
	h.authRequest(req)
	if s := strings.TrimSpace(os.Getenv("NEURAL_JUNKIE_FULL_METADATA_SECRET")); s != "" {
		req.Header.Set("X-NJ-Full-Metadata", s)
	}
	resp, err := h.client.Do(req)
	if err != nil {
		return h.GetMessages(channelName, limit)
	}
	defer resp.Body.Close()
	var body struct {
		Messages []*protocol.Message `json:"messages"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&body); err != nil {
		return h.GetMessages(channelName, limit)
	}
	msgs := body.Messages
	if limit <= 0 {
		limit = 50
	}
	if len(msgs) <= limit {
		return msgs, nil
	}
	return msgs[len(msgs)-limit:], nil
}

func (h *httpHubClient) GetChannelAgents(channelName string) ([]protocol.AgentInfo, error) {
	resp, err := h.doGet(fmt.Sprintf("%s/api/agents", h.baseURL))
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
	resp, err := h.doGet(fmt.Sprintf("%s/api/threads/%s/parent-author", h.baseURL, threadID))
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
	resp, err := h.doGet(fmt.Sprintf("%s/api/agent-channels?agent_id=%s", h.baseURL, agentID))
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
func (h *httpHubClient) GetThreadMessages(threadID string, limit int) ([]*protocol.Message, error) {
	return nil, nil
}

func (h *httpHubClient) GetChannelType(channelName string) protocol.ChannelType {
	resp, err := h.doGet(fmt.Sprintf("%s/api/channels", h.baseURL))
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

func (h *httpHubClient) MusicGenerationEnabled() bool {
	return false
}

func (h *httpHubClient) GenerateAndPostMusic(ctx context.Context, channel string, from protocol.AgentInfo, req agent.MusicGenerateRequest) error {
	return fmt.Errorf("music generation requires an in-process hub connection")
}
func (h *httpHubClient) ExtractAndPostMusicStems(ctx context.Context, channel string, from protocol.AgentInfo, req agent.MusicExtractRequest) error {
	return fmt.Errorf("music stem extraction requires an in-process hub connection")
}

func (h *httpHubClient) AskUserQuestion(agentID, agentName, channel, question string, options []string) (string, error) {
	body, err := json.Marshal(map[string]interface{}{
		"agent_id":   agentID,
		"agent_name": agentName,
		"channel":    channel,
		"question":   question,
		"options":    options,
	})
	if err != nil {
		return "", err
	}
	resp, err := h.doPost(h.baseURL+"/api/user-questions", "application/json", body)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("user question request failed: %s", resp.Status)
	}
	var out struct {
		Answer string `json:"answer"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&out); err != nil {
		return "", err
	}
	return out.Answer, nil
}

func (h *httpHubClient) RequestToolApproval(agentID, agentName, channel, toolName string, toolInput map[string]interface{}) (bool, error) {
	body, err := json.Marshal(map[string]interface{}{
		"agent_id":   agentID,
		"agent_name": agentName,
		"channel":    channel,
		"tool_name":  toolName,
		"tool_input": toolInput,
		"mode":       "interactive",
	})
	if err != nil {
		return false, err
	}
	resp, err := h.doPost(h.baseURL+"/api/tool-approvals", "application/json", body)
	if err != nil {
		return false, err
	}
	defer resp.Body.Close()
	var out struct {
		Decision string `json:"decision"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&out); err != nil {
		return false, err
	}
	return out.Decision == "allow", nil
}

func main() {
	flag.Parse()

	// Validate agent type
	aType := protocol.AgentType(*agentType)
	validTypes := map[protocol.AgentType]bool{
		protocol.AgentTypeFrontend:     true,
		protocol.AgentTypeBackend:      true,
		protocol.AgentTypeDevOps:       true,
		protocol.AgentTypeDatabase:     true,
		protocol.AgentTypeSecurity:     true,
		protocol.AgentTypeRust:         true,
		protocol.AgentTypeArchitecture: true,
		protocol.AgentTypeCodeReview:   true,
		protocol.AgentTypeBiology:           true,
		protocol.AgentTypeGenomics:          true,
		protocol.AgentTypeStructuralBiology: true,
		protocol.AgentTypeCheminformatics:   true,
		protocol.AgentTypeRepo:         true,
		protocol.AgentTypeAssistant:    true,
	}

	if !validTypes[aType] {
		log.Fatalf("Invalid agent type: %s. Valid types: frontend, backend, devops, security, architecture, code-review, database, rust, biology, repo, assistant", *agentType)
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

	usePollTransport := *usePoll || strings.TrimSpace(os.Getenv("NEURAL_JUNKIE_AGENT_POLL")) == "1"
	key := strings.TrimSpace(*apiKey)
	if key == "" {
		key = strings.TrimSpace(os.Getenv("NEURAL_JUNKIE_API_KEY"))
	}
	if strings.TrimSpace(os.Getenv("NEURAL_JUNKIE_AUTH_REQUIRED")) == "1" && key == "" {
		log.Fatal("NEURAL_JUNKIE_AUTH_REQUIRED=1: set --api-key or NEURAL_JUNKIE_API_KEY for standalone agents")
	}
	hubClient := newHTTPHubClient(*serverAddr, usePollTransport, key)
	if !usePollTransport {
		log.Println("Using WebSocket transport for hub messages")
	} else {
		log.Println("Using HTTP polling for hub messages (legacy)")
	}

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

	hubClient.SetAgentID(agentInstance.Info.ID)

	// Register with hub
	registerData, _ := json.Marshal(agentInstance.Info)

	resp, err := hubClient.doPost(*serverAddr+"/api/agents", "application/json", registerData)
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

	resp, err = hubClient.doPost(*serverAddr+"/api/channels/join", "application/json", joinData)
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
	close(hubClient.stopWS)
	agentInstance.Stop()
}

func generateAgentName(agentType protocol.AgentType) string {
	names := map[protocol.AgentType][]string{
		protocol.AgentTypeFrontend:     {"Frontend Engineer", "UI Engineer", "Accessibility Reviewer", "Design Systems Partner"},
		protocol.AgentTypeBackend:      {"Backend Engineer", "API Architect", "Service Designer", "Integration Partner"},
		protocol.AgentTypeDevOps:       {"Platform Engineer", "Infrastructure Partner", "Deployment Reviewer", "CI/CD Specialist"},
		protocol.AgentTypeDatabase:     {"Database Specialist", "Data Architect", "Query Optimizer", "Migration Reviewer"},
		protocol.AgentTypeSecurity:     {"Security Reviewer", "InfoSec Partner", "Threat Modeler", "Auth Reviewer"},
		protocol.AgentTypeRust:         {"Rust Expert", "Rust Architect", "Cargo Master", "Ownership Guru"},
		protocol.AgentTypeArchitecture: {"Software Architect", "System Designer", "Architecture Reviewer", "Migration Planner"},
		protocol.AgentTypeCodeReview:   {"Code Reviewer", "Quality Reviewer", "Regression Reviewer", "Maintainability Reviewer"},
		protocol.AgentTypeRepo:         {"Repo Expert", "Code Navigator", "Project Guide", "Codebase Oracle"},
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
