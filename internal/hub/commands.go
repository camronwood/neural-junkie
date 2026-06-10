package hub

import (
	"context"
	"fmt"
	"log"
	"sync"

	"github.com/camronwood/neural-junkie/internal/agent"
	"github.com/camronwood/neural-junkie/internal/ai"
	"github.com/camronwood/neural-junkie/internal/config"
	"github.com/camronwood/neural-junkie/internal/mcp_export"
	"github.com/camronwood/neural-junkie/internal/protocol"
)

// CommandHandler handles chat commands
type CommandHandler struct {
	hub              *Hub
	appConfig        *config.Config
	providerCache    *ai.ProviderCache
	aiProvider       ai.AIProvider
	repoAgents       map[string]*agent.RepoAgent        // Track repo agents for management
	confluenceAgents map[string]*agent.ConfluenceAgent  // Track confluence agents for management
	cliAgents        map[string]*agent.Agent            // CLI proxy agents keyed by agent ID (one runtime per instance)
	runtimeAgents    map[string]*agent.Agent            // Track runtime specialist/moderator/assistant/CLI agents
	agentsMu         sync.RWMutex                       // protects repoAgents, confluenceAgents, cliAgents, runtimeAgents
	assistantAgent   *agent.AssistantAgent              // Track assistant agent for meeting notes
	exportStorage    *mcp_export.ExportStorage          // Export storage for MCP exports
	pendingReviews   map[string]*protocol.PendingReview // Track pending reviews by repo path
	pendingMutex     sync.Mutex                         // Protects pending reviews map
	// collaborateRedirect is set when /collaborate succeeds so /api/send can tell the client to switch rooms.
	collabRedirectMu      sync.Mutex
	collabRedirectChannel string
	collabRedirectID      string
	// dmRedirect is set when /create-expert succeeds so /api/send can open the new DM.
	dmRedirectMu      sync.Mutex
	dmRedirectChannel string
}

type commandExecutor func(ctx context.Context, msg *protocol.Message, parts []string) (*protocol.Message, error)

// NewCommandHandler creates a new command handler
func NewCommandHandler(hub *Hub) (*CommandHandler, error) {
	// Create AI provider for repo agents
	var aiProvider ai.AIProvider
	ollamaProvider, err := ai.NewOllamaProvider()
	if err != nil {
		log.Printf("Warning: failed to initialize Ollama provider for repo agents: %v (using mock provider)", err)
		aiProvider = ai.NewMockProvider()
	} else {
		aiProvider = ollamaProvider
		log.Printf("Ollama provider initialized for repo agents (model: %s)", ollamaProvider.GetModel())
	}

	// Initialize export storage
	exportStorage, err := mcp_export.NewExportStorage()
	if err != nil {
		return nil, fmt.Errorf("failed to create export storage: %w", err)
	}

	ch := &CommandHandler{
		hub:              hub,
		aiProvider:       aiProvider,
		repoAgents:       make(map[string]*agent.RepoAgent),
		confluenceAgents: make(map[string]*agent.ConfluenceAgent),
		cliAgents:        make(map[string]*agent.Agent),
		runtimeAgents:    make(map[string]*agent.Agent),
		exportStorage:    exportStorage,
		pendingReviews:   make(map[string]*protocol.PendingReview),
	}
	ch.validateCommandDefinitions()
	return ch, nil
}
