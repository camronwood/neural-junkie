package agent

import (
	"context"
	"fmt"
	"log"
	"strings"
	"time"

	"github.com/camronwood/neural-junkie/internal/ai"
	"github.com/camronwood/neural-junkie/internal/confluence"
	"github.com/camronwood/neural-junkie/internal/protocol"
	"github.com/google/uuid"
)

// ConfluenceAgent is a specialized agent that is an expert on a Confluence space
type ConfluenceAgent struct {
	*Agent
	spaceKey   string
	index      *confluence.ConfluenceIndex
	storage    *confluence.Storage
	client     *confluence.Client
	isIndexing bool
}

// NewConfluenceAgent creates a new Confluence space expert agent
func NewConfluenceAgent(name string, spaceKey string, ai ai.AIProvider, hub HubClient) (*ConfluenceAgent, error) {
	storage, err := confluence.NewStorage()
	if err != nil {
		return nil, fmt.Errorf("failed to create storage: %w", err)
	}

	client, err := confluence.NewClient()
	if err != nil {
		return nil, fmt.Errorf("failed to create Confluence client: %w", err)
	}

	baseAgent := &Agent{
		Info: protocol.AgentInfo{
			ID:                 uuid.New().String(),
			Name:               name,
			Type:               protocol.AgentTypeConfluence,
			Expertise:          []string{}, // Will be populated from space
			Status:             "active",
			Model:              ai.GetModel(),
			IsPaused:           false,
			IndexingStatus:     string(protocol.IndexingStatusIndexing),
			IndexProgress:      0,
			ConfluenceSpaceKey: spaceKey,
		},
		AI:  ai,
		Hub: hub,
		Context: &ConversationContext{
			History:    make(map[string][]*protocol.Message),
			MaxHistory: 50,
		},
		stopCh:            make(chan struct{}),
		msgCh:             make(chan *protocol.Message, 100),
		respondedMessages: make(map[string]bool),
	}

	confluenceAgent := &ConfluenceAgent{
		Agent:      baseAgent,
		spaceKey:   spaceKey,
		storage:    storage,
		client:     client,
		isIndexing: true,
	}

	return confluenceAgent, nil
}

// StartWithIndexing starts the agent and begins indexing the Confluence space
func (ca *ConfluenceAgent) StartWithIndexing(ctx context.Context, channel string) error {
	ca.Context.CurrentChannel = channel

	// Subscribe to channel messages
	subCh, err := ca.Hub.Subscribe(channel)
	if err != nil {
		return fmt.Errorf("failed to subscribe to channel: %w", err)
	}

	// Start indexing in background
	go ca.indexSpace(ctx)

	// Start message processing loop
	go func() {
		for {
			select {
			case <-ctx.Done():
				return
			case <-ca.stopCh:
				return
			case msg := <-subCh:
				if msg == nil {
					return
				}
				ca.handleMessage(ctx, msg)
			}
		}
	}()

	return nil
}

// indexSpace performs the space indexing
func (ca *ConfluenceAgent) indexSpace(ctx context.Context) {
	ca.isIndexing = true
	ca.Info.IndexingStatus = string(protocol.IndexingStatusIndexing)

	log.Printf("[ConfluenceAgent:%s] Starting indexing for space: %s", ca.Info.Name, ca.spaceKey)

	// Check if cached index exists
	if ca.storage.IndexExists(ca.spaceKey) {
		log.Printf("[ConfluenceAgent:%s] Loading cached index...", ca.Info.Name)

		index, err := ca.storage.LoadIndex(ca.spaceKey)
		if err != nil {
			log.Printf("[ConfluenceAgent:%s] Failed to load cached index: %v. Starting fresh indexing.", ca.Info.Name, err)
		} else {
			ca.index = index
			ca.updateExpertiseFromIndex()

			// Check staleness
			analyzer := confluence.NewAnalyzer(ca.client, ca.updateProgress)
			isStale, stalePages, err := analyzer.CheckStaleness(index)
			if err != nil {
				log.Printf("[ConfluenceAgent:%s] Failed to check staleness: %v", ca.Info.Name, err)
			} else if isStale {
				log.Printf("[ConfluenceAgent:%s] Index is stale (%d pages changed). Performing incremental update.", ca.Info.Name, len(stalePages))

				ca.Info.IndexingStatus = string(protocol.IndexingStatusReindexing)
				ca.sendStatusUpdate("Updating stale pages...")

				if err := analyzer.IncrementalUpdate(ctx, index, stalePages); err != nil {
					log.Printf("[ConfluenceAgent:%s] Incremental update failed: %v", ca.Info.Name, err)
				} else {
					// Save updated index
					if err := ca.storage.SaveIndex(index); err != nil {
						log.Printf("[ConfluenceAgent:%s] Failed to save updated index: %v", ca.Info.Name, err)
					}
					ca.updateExpertiseFromIndex()
				}
			} else {
				log.Printf("[ConfluenceAgent:%s] Cached index is up-to-date", ca.Info.Name)
			}

			ca.isIndexing = false
			ca.Info.IndexingStatus = string(protocol.IndexingStatusReady)
			ca.Info.IndexProgress = 100
			ca.sendStatusUpdate("Ready")
			return
		}
	}

	// Perform full indexing
	analyzer := confluence.NewAnalyzer(ca.client, ca.updateProgress)

	index, err := analyzer.AnalyzeSpace(ctx, ca.spaceKey)
	if err != nil {
		log.Printf("[ConfluenceAgent:%s] Indexing failed: %v", ca.Info.Name, err)
		ca.Info.IndexingStatus = string(protocol.IndexingStatusError)
		ca.isIndexing = false
		ca.sendStatusUpdate(fmt.Sprintf("Indexing failed: %v", err))
		return
	}

	ca.index = index
	ca.updateExpertiseFromIndex()

	// Save index to storage
	if err := ca.storage.SaveIndex(index); err != nil {
		log.Printf("[ConfluenceAgent:%s] Failed to save index: %v", ca.Info.Name, err)
	} else {
		log.Printf("[ConfluenceAgent:%s] Index saved successfully", ca.Info.Name)
	}

	ca.isIndexing = false
	ca.Info.IndexingStatus = string(protocol.IndexingStatusReady)
	ca.Info.IndexProgress = 100

	ca.sendStatusUpdate("Ready")
	log.Printf("[ConfluenceAgent:%s] Indexing complete. Indexed %d pages.", ca.Info.Name, index.PageCount)
}

// updateProgress updates the indexing progress
func (ca *ConfluenceAgent) updateProgress(progress int, message string) {
	ca.Info.IndexProgress = progress

	// Send periodic status updates (every 10%)
	if progress%10 == 0 || progress == 100 {
		ca.sendStatusUpdate(message)
	}

	log.Printf("[ConfluenceAgent:%s] %d%% - %s", ca.Info.Name, progress, message)
}

// updateExpertiseFromIndex updates agent expertise based on indexed content
func (ca *ConfluenceAgent) updateExpertiseFromIndex() {
	if ca.index == nil {
		return
	}

	expertise := []string{
		ca.index.SpaceName,
		"confluence",
		"documentation",
	}

	// Add common labels as expertise
	labelCounts := make(map[string]int)
	for label := range ca.index.Labels {
		labelCounts[label] = len(ca.index.Labels[label])
	}

	// Add top labels to expertise
	for label, count := range labelCounts {
		if count >= 3 { // Label used on at least 3 pages
			expertise = append(expertise, label)
		}
	}

	ca.Info.Expertise = expertise
}

// sendStatusUpdate sends a status update message
func (ca *ConfluenceAgent) sendStatusUpdate(message string) {
	if ca.Context.CurrentChannel == "" {
		return
	}

	statusMsg := &protocol.Message{
		ID:        uuid.New().String(),
		Type:      protocol.MessageTypeAgentStatus,
		Channel:   ca.Context.CurrentChannel,
		From:      ca.Info,
		Content:   message,
		Timestamp: time.Now(),
	}

	ca.Hub.SendMessage(statusMsg)
}

// handleMessage processes incoming messages
func (ca *ConfluenceAgent) handleMessage(ctx context.Context, msg *protocol.Message) {
	if msg.Type == protocol.MessageTypeAgentStatus && msg.Metadata != nil {
		if v, ok := msg.Metadata["history_resync"].(bool); ok && v && msg.Channel != "" {
			if hist, err := ca.Hub.GetMessages(msg.Channel, 20); err == nil {
				if ca.Context.History == nil {
					ca.Context.History = make(map[string][]*protocol.Message)
				}
				ca.Context.History[msg.Channel] = hist
			}
			return
		}
	}

	// Skip if paused
	if ca.Info.IsPaused {
		return
	}

	// Skip our own messages
	if msg.From.ID == ca.Info.ID {
		return
	}

	// Skip if still indexing (unless directly mentioned)
	if ca.isIndexing {
		if !msg.IsMentioned(ca.Info.ID) {
			return
		}
		// If mentioned while indexing, inform user
		response := fmt.Sprintf("I'm currently indexing the %s space. Please wait until indexing is complete.", ca.index.SpaceName)
		responseMsg := protocol.NewMessage(
			protocol.MessageTypeChat,
			msg.Channel,
			ca.Info,
			response,
		)
		responseMsg.ReplyTo = msg.ID
		ca.Hub.SendMessage(responseMsg)
		return
	}

	// Check if we should respond
	if !ca.shouldRespond(msg) {
		return
	}

	// Generate and send response
	ca.generateResponse(ctx, msg)
}

// shouldRespond determines if the agent should respond to a message
func (ca *ConfluenceAgent) shouldRespond(msg *protocol.Message) bool {
	// Always respond if mentioned
	if msg.IsMentioned(ca.Info.ID) {
		return true
	}

	// Don't respond to system messages
	if msg.Type == protocol.MessageTypeSystemInfo ||
		msg.Type == protocol.MessageTypeAgentJoin ||
		msg.Type == protocol.MessageTypeAgentLeave ||
		msg.Type == protocol.MessageTypeAgentStatus {
		return false
	}

	// Check if message is relevant to our expertise
	content := strings.ToLower(msg.Content)
	for _, expertise := range ca.Info.Expertise {
		if strings.Contains(content, strings.ToLower(expertise)) {
			return true
		}
	}

	// Check if it's a question about documentation
	if strings.Contains(content, "documentation") ||
		strings.Contains(content, "docs") ||
		strings.Contains(content, "confluence") ||
		strings.Contains(content, "wiki") {
		return true
	}

	return false
}

// generateResponse generates an AI response using Confluence context
func (ca *ConfluenceAgent) generateResponse(ctx context.Context, msg *protocol.Message) {
	// Send thinking indicator
	ca.sendThinkingStatus(msg, protocol.ThinkingStatusStarted)

	// Search for relevant pages
	searcher := confluence.NewSearcher(ca.index)
	searchResults := searcher.Search(msg.Content, 5)

	// Build context from search results
	contextParts := []string{
		fmt.Sprintf("You are an expert on the '%s' Confluence space.", ca.index.SpaceName),
		"",
		"Here are the most relevant pages from the documentation:",
		"",
	}

	if len(searchResults) > 0 {
		for i, result := range searchResults {
			contextParts = append(contextParts,
				fmt.Sprintf("## Page %d: %s", i+1, result.Page.Title),
				fmt.Sprintf("URL: %s", result.Page.URL),
				fmt.Sprintf("Last Updated: %s", result.Page.LastUpdated.Format("2006-01-02")),
				"",
				result.Snippet,
				"",
			)
		}
	} else {
		contextParts = append(contextParts, "No directly relevant pages found in the documentation.")
	}

	contextParts = append(contextParts,
		"",
		"Use this information to answer the user's question. If the information isn't in the documentation, say so clearly.",
	)

	context := strings.Join(contextParts, "\n")

	var prefix strings.Builder
	PrependRulesAndAttachmentsForMonolithic(&prefix, msg, &ca.Info)

	// Generate response using AI
	prompt := fmt.Sprintf("%s%s\n\nUser Question: %s", prefix.String(), context, msg.Content)

	// Convert history from []*protocol.Message to []protocol.Message
	history := ca.Context.History[msg.Channel]
	historyMsgs := make([]protocol.Message, len(history))
	for i, h := range history {
		historyMsgs[i] = *h
	}

	eff := ca.EffectiveAIProvider(ctx, msg)
	if eff == nil {
		eff = ca.GetAIProvider()
	}
	response, err := eff.GenerateResponse(ctx, prompt, historyMsgs)
	if err != nil {
		log.Printf("[ConfluenceAgent:%s] Failed to generate response: %v", ca.Info.Name, err)
		ca.sendThinkingStatus(msg, protocol.ThinkingStatusError)
		return
	}

	// Send response
	responseMsg := protocol.NewMessage(
		protocol.MessageTypeChat,
		msg.Channel,
		ca.Info,
		response,
	)
	responseMsg.ReplyTo = msg.ID

	if err := ca.Hub.SendMessage(responseMsg); err != nil {
		log.Printf("[ConfluenceAgent:%s] Failed to send response: %v", ca.Info.Name, err)
		ca.sendThinkingStatus(msg, protocol.ThinkingStatusError)
		return
	}

	ca.sendThinkingStatus(msg, protocol.ThinkingStatusCompleted)
}

// Reindex triggers a manual reindex of the space
func (ca *ConfluenceAgent) Reindex(ctx context.Context) error {
	if ca.isIndexing {
		return fmt.Errorf("already indexing")
	}

	go ca.indexSpace(ctx)
	return nil
}

// GetIndex returns the current index (for commands/inspection)
func (ca *ConfluenceAgent) GetIndex() *confluence.ConfluenceIndex {
	return ca.index
}

// Search performs a search on the indexed space
func (ca *ConfluenceAgent) Search(query string, limit int) []confluence.SearchResult {
	if ca.index == nil {
		return []confluence.SearchResult{}
	}

	searcher := confluence.NewSearcher(ca.index)
	return searcher.Search(query, limit)
}

// SetCredentials sets custom Confluence credentials for the agent
func (ca *ConfluenceAgent) SetCredentials(credentials map[string]string) {
	if ca.client != nil {
		ca.client.SetCredentials(credentials)
	}
}
