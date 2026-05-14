package agent

import (
	"context"
	"fmt"
	"log"
	"os"
	"strings"
	"sync"

	"github.com/camronwood/neural-junkie/internal/ai"
	"github.com/camronwood/neural-junkie/internal/protocol"
	"github.com/camronwood/neural-junkie/internal/repo"
	"github.com/google/uuid"
)

// RepoAgent is a specialized agent that is an expert on a specific repository
type RepoAgent struct {
	*Agent
	repoPath        string
	index           *repo.RepositoryIndex
	storage         *repo.Storage
	isIndexing      bool
	watcher         *repo.Watcher
	enableAutoWatch bool         // Enable automatic file watching and reindexing
	mu              sync.RWMutex // Protects index, isIndexing, watcher, enableAutoWatch
}

// NewRepoAgent creates a new repository expert agent
func NewRepoAgent(name string, repoPath string, ai ai.AIProvider, hub HubClient) (*RepoAgent, error) {
	// Validate repository path
	if repoPath == "" {
		return nil, fmt.Errorf("repository path cannot be empty")
	}

	// Check if path exists
	if _, err := os.Stat(repoPath); os.IsNotExist(err) {
		return nil, fmt.Errorf("repository path does not exist: %s", repoPath)
	}

	// Check if path is a directory
	if info, err := os.Stat(repoPath); err == nil && !info.IsDir() {
		return nil, fmt.Errorf("repository path is not a directory: %s", repoPath)
	}

	storage, err := repo.NewStorage()
	if err != nil {
		return nil, fmt.Errorf("failed to create storage: %w", err)
	}

	baseAgent := &Agent{
		Info: protocol.AgentInfo{
			ID:             uuid.New().String(),
			Name:           name,
			Type:           protocol.AgentTypeRepo,
			Expertise:      []string{}, // Will be populated from repo
			Status:         "active",
			Model:          ai.GetModel(),
			IsPaused:       false,
			IndexingStatus: string(protocol.IndexingStatusIndexing),
			IndexProgress:  0,
			RepositoryPath: repoPath,
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
		activeChannels:    make(map[string]context.CancelFunc),
		WorkspacePath:     os.Getenv("WORKSPACE_PATH"),
	}

	repoAgent := &RepoAgent{
		Agent:           baseAgent,
		repoPath:        repoPath,
		storage:         storage,
		isIndexing:      true,
		enableAutoWatch: false, // Disabled by default to save resources
	}

	return repoAgent, nil
}

// StartWithIndexing starts the agent and begins indexing the repository
func (ra *RepoAgent) StartWithIndexing(ctx context.Context, channel string) error {
	ra.Context.CurrentChannel = channel

	// Start on the initial channel using shared channel lifecycle logic.
	if err := ra.AddChannel(ctx, channel); err != nil {
		return err
	}
	go ra.discoverChannels(ctx)

	// In tests (mock providers), run indexing synchronously to avoid
	// goroutine races with temp directory cleanup.
	if _, ok := ra.AI.(*ai.MockProvider); ok {
		ra.indexRepository(ctx)
	} else {
		// Start indexing in background for normal runtime usage.
		go ra.indexRepository(ctx)
	}

	return nil
}

// indexRepository performs the repository indexing
func (ra *RepoAgent) indexRepository(ctx context.Context) {
	log.Printf("[%s] Starting repository indexing: %s", ra.Info.Name, ra.repoPath)

	analyzer := repo.NewAnalyzer(func(progress int, message string) {
		ra.updateIndexProgress(progress, message)
	})

	// Generate cache key from repository path
	cacheKey, err := ra.storage.GetCacheKeyForPath(ra.repoPath)
	if err != nil {
		log.Printf("[%s] Failed to generate cache key: %v", ra.Info.Name, err)
		ra.Info.IndexingStatus = string(protocol.IndexingStatusError)
		ra.Info.IndexProgress = 0
		ra.updateAgentStatus()
		return
	}

	// Try to load cached index first
	var index *repo.RepositoryIndex
	cacheLoaded := false

	if ra.storage.IndexExists(cacheKey) {
		log.Printf("[%s] Found cached index, checking if stale...", ra.Info.Name)
		ra.sendStatusMessage("🔍 Found cached index, checking freshness...")

		cachedIndex, loadErr := ra.storage.LoadIndex(cacheKey)
		if loadErr == nil {
			isStale, reason, checkErr := analyzer.IsIndexStale(ctx, ra.repoPath, cachedIndex)
			if checkErr != nil {
				log.Printf("[%s] Error checking staleness: %v, will reindex", ra.Info.Name, checkErr)
				ra.sendStatusMessage(fmt.Sprintf("⚠️  Error checking cache: %v, performing full analysis", checkErr))
			} else if !isStale {
				// Index is fresh, use it
				log.Printf("[%s] Cached index is fresh, loading instantly!", ra.Info.Name)
				ra.sendStatusMessage("✅ Loaded from cache (instant) - repository already indexed!")
				index = cachedIndex
				cacheLoaded = true
			} else {
				// Index is stale, do incremental update
				log.Printf("[%s] Cached index is stale (%s), performing incremental update...", ra.Info.Name, reason)
				ra.sendStatusMessage(fmt.Sprintf("🔄 Cache stale (%s), performing incremental update...", reason))
				ra.Info.IndexingStatus = string(protocol.IndexingStatusReindexing)
				ra.updateAgentStatus()
				index, err = analyzer.IncrementalAnalyze(ctx, ra.repoPath, cachedIndex)
				if err != nil {
					log.Printf("[%s] Incremental analysis failed: %v, falling back to full analysis", ra.Info.Name, err)
					ra.sendStatusMessage("⚠️  Incremental update failed, performing full analysis...")
					index = nil
				}
			}
		} else {
			log.Printf("[%s] Failed to load cached index: %v", ra.Info.Name, loadErr)
			ra.sendStatusMessage(fmt.Sprintf("⚠️  Failed to load cache: %v, performing full analysis", loadErr))
		}
	} else {
		log.Printf("[%s] No cached index found, performing full analysis", ra.Info.Name)
		ra.sendStatusMessage("📊 No cache found, performing full analysis (this may take 30-60 seconds)...")
	}

	// If no cached index or incremental failed, do full analysis
	if index == nil {
		log.Printf("[%s] Performing full repository analysis...", ra.Info.Name)
		index, err = analyzer.AnalyzeRepository(ctx, ra.repoPath)
		if err != nil {
			log.Printf("[%s] Indexing failed: %v", ra.Info.Name, err)
			ra.Info.IndexingStatus = string(protocol.IndexingStatusError)
			ra.Info.IndexProgress = 0
			ra.updateAgentStatus()
			ra.sendStatusMessage(fmt.Sprintf("❌ Indexing failed: %v", err))
			return
		}
	}

	// Save index to storage with cache key
	if err := ra.storage.SaveIndex(cacheKey, index); err != nil {
		log.Printf("[%s] Failed to save index: %v", ra.Info.Name, err)
	}

	// Save metadata
	metadata := &repo.RepoMetadata{
		Path:       ra.repoPath,
		CacheKey:   cacheKey,
		AgentNames: []string{ra.Info.Name},
	}
	if err := ra.storage.SaveMetadata(cacheKey, metadata); err != nil {
		log.Printf("[%s] Failed to save metadata: %v", ra.Info.Name, err)
	}

	ra.mu.Lock()
	ra.index = index
	ra.isIndexing = false
	indexName := ""
	if index != nil {
		indexName = index.Name
	}
	ra.Info.IndexingStatus = string(protocol.IndexingStatusReady)
	ra.Info.IndexProgress = 100
	ra.mu.Unlock()

	// Update expertise based on repository (buildExpertise locks internally)
	ra.Info.Expertise = ra.buildExpertise()

	ra.updateAgentStatus()

	if cacheLoaded {
		log.Printf("[%s] Cache loaded! Ready to answer questions about %s", ra.Info.Name, indexName)
	} else {
		log.Printf("[%s] Indexing complete! Ready to answer questions about %s", ra.Info.Name, indexName)
		ra.sendStatusMessage("✅ Indexing complete! Repository cached for future use.")
	}

	// Check for pending review and auto-respond
	ra.checkAndRespondToPendingReview(ctx)

	// Start file watcher if enabled
	ra.mu.Lock()
	shouldStartWatcher := ra.enableAutoWatch && ra.watcher == nil
	ra.mu.Unlock()
	if shouldStartWatcher {
		ra.startWatcher(ctx)
	}
}

// startWatcher starts the file system watcher
func (ra *RepoAgent) startWatcher(ctx context.Context) {
	watcher, err := repo.NewWatcher(ra.repoPath, func(path string) {
		// Trigger incremental reindex when files change
		ra.mu.RLock()
		indexing := ra.isIndexing
		ra.mu.RUnlock()
		if !indexing {
			log.Printf("[%s] File changes detected, triggering incremental reindex", ra.Info.Name)
			if err := ra.Reindex(ctx); err != nil {
				log.Printf("[%s] Failed to trigger auto-reindex: %v", ra.Info.Name, err)
			}
		}
	})

	if err != nil {
		log.Printf("[%s] Failed to start file watcher: %v", ra.Info.Name, err)
		return
	}

	ra.mu.Lock()
	ra.watcher = watcher
	ra.mu.Unlock()
	watcher.Start(ctx)
	log.Printf("[%s] File watcher started for %s", ra.Info.Name, ra.repoPath)
}

// EnableAutoWatch enables automatic file watching and reindexing
func (ra *RepoAgent) EnableAutoWatch(ctx context.Context) {
	ra.mu.Lock()
	if ra.enableAutoWatch {
		ra.mu.Unlock()
		return
	}
	ra.enableAutoWatch = true
	shouldStart := ra.watcher == nil && !ra.isIndexing
	ra.mu.Unlock()
	if shouldStart {
		ra.startWatcher(ctx)
	}
	log.Printf("[%s] Auto-watch enabled", ra.Info.Name)
}

// DisableAutoWatch disables automatic file watching
func (ra *RepoAgent) DisableAutoWatch() {
	ra.mu.Lock()
	if !ra.enableAutoWatch {
		ra.mu.Unlock()
		return
	}
	ra.enableAutoWatch = false
	watcher := ra.watcher
	ra.watcher = nil
	ra.mu.Unlock()
	if watcher != nil {
		watcher.Stop()
	}
	log.Printf("[%s] Auto-watch disabled", ra.Info.Name)
}

// Reindex triggers a reindex of the repository
func (ra *RepoAgent) Reindex(ctx context.Context) error {
	ra.mu.Lock()
	if ra.isIndexing {
		ra.mu.Unlock()
		return fmt.Errorf("already indexing")
	}
	ra.isIndexing = true
	ra.Info.IndexingStatus = string(protocol.IndexingStatusReindexing)
	ra.Info.IndexProgress = 0
	ra.mu.Unlock()
	ra.updateAgentStatus()

	go ra.indexRepository(ctx)

	return nil
}

// handleMessage overrides the base agent's handleMessage to check indexing status
func (ra *RepoAgent) handleMessage(ctx context.Context, msg *protocol.Message) {
	if msg.Type == protocol.MessageTypeAgentStatus && msg.Metadata != nil {
		if v, ok := msg.Metadata["history_resync"].(bool); ok && v && msg.Channel != "" {
			if hist, err := ra.Hub.GetMessages(msg.Channel, 20); err == nil {
				if ra.Context.History == nil {
					ra.Context.History = make(map[string][]*protocol.Message)
				}
				ra.Context.History[msg.Channel] = hist
			}
			return
		}
	}

	// Ignore own messages
	if msg.From.ID == ra.Info.ID {
		return
	}

	// Don't respond if paused
	if ra.Info.IsPaused {
		return
	}

	// Don't respond if still indexing
	ra.mu.RLock()
	indexing := ra.isIndexing
	ra.mu.RUnlock()
	if indexing {
		return
	}

	// Add to history before eligibility checks so we retain ordering for all traffic.
	ra.addToHistory(msg)

	// Decide if we should respond (before marking "responded" to avoid suppressing future unrelated work)
	if !ra.shouldRespondToRepo(msg) {
		return
	}

	// Check if we've already responded to this message
	ra.respondedMutex.Lock()
	if ra.respondedMessages[msg.ID] {
		ra.respondedMutex.Unlock()
		log.Printf("[%s] Skipping message %s (already responded)", ra.Info.Name, msg.ID[:8])
		return
	}
	ra.respondedMessages[msg.ID] = true
	ra.respondedMutex.Unlock()

	log.Printf("[%s] Processing message from %s: %s", ra.Info.Name, msg.From.Name, msg.Content)

	// Send thinking status
	ra.sendThinkingStatus(msg, protocol.ThinkingStatusStarted)

	// Generate response
	response, err := ra.generateRepoResponse(ctx, msg)
	if err != nil {
		log.Printf("[%s] Error generating response: %v", ra.Info.Name, err)
		ra.sendThinkingStatus(msg, protocol.ThinkingStatusError)
		return
	}

	// Send response
	responseMsg := protocol.NewMessage(
		protocol.MessageTypeChat,
		msg.Channel,
		ra.Info,
		response,
	)
	responseMsg.ReplyTo = msg.ID
	if collabID := msg.GetCollaborationID(); collabID != "" {
		responseMsg.SetCollaborationID(collabID)
		if phase := msg.GetCollaborationPhase(); phase != "" {
			responseMsg.SetCollaborationPhase(phase)
		}
		if taskID := msg.GetTaskID(); taskID != "" {
			responseMsg.SetTaskID(taskID)
		}
		if taskStatus := msg.GetTaskStatus(); taskStatus != "" {
			responseMsg.SetTaskStatus(taskStatus)
		}
		if taskOutput := msg.GetTaskOutput(); taskOutput != "" {
			responseMsg.SetTaskOutput(taskOutput)
		}
	}

	if err := ra.Hub.SendMessage(responseMsg); err != nil {
		log.Printf("[%s] Error sending message: %v", ra.Info.Name, err)
		ra.sendThinkingStatus(msg, protocol.ThinkingStatusError)
		return
	}
	if collabID := responseMsg.GetCollaborationID(); collabID != "" && ra.Collab != nil {
		if err := ra.Collab.RecordMessage(collabID, responseMsg); err != nil {
			log.Printf("[%s] Warning: failed to record collaboration message: %v", ra.Info.Name, err)
		}
		ra.Collab.AnalyzeConsensus(collabID, responseMsg)
	}
	ra.sendThinkingStatus(msg, protocol.ThinkingStatusCompleted)
}

// shouldRespondToRepo determines if the agent should respond based on repo context
func (ra *RepoAgent) shouldRespondToRepo(msg *protocol.Message) bool {
	// Never respond to commands - let the command handler process them
	if len(msg.Content) > 0 && msg.Content[0] == '/' {
		return false
	}

	// Collaboration coordination messages can originate from System and still
	// be actionable, so evaluate this before generic system-message rejection.
	if collabID := msg.GetCollaborationID(); collabID != "" && ra.Collab != nil {
		if ra.Collab.IsParticipant(collabID, ra.Info.ID) && ra.Collab.IsActive(collabID) {
			if msg.Type == protocol.MessageTypeCollabTask && msg.Metadata != nil {
				if assignee, ok := taskAssigneeFromMetadata(msg.Metadata); ok && assignee == ra.Info.ID {
					return true
				}
			}
			if ra.Collab.IsAgentTurn(collabID, ra.Info.ID) {
				return true
			}
			if msg.IsMentioned(ra.Info.ID) && ra.Collab.AgentOutOfTurnMentionAllowed(collabID) {
				return true
			}
		}
		return false
	}

	// Never respond to system messages (errors, notifications, etc.)
	if msg.From.Name == "System" || msg.From.ID == "system" {
		return false
	}

	// Check if message is from another agent (not a human)
	isFromAgent := msg.From.Type == protocol.AgentTypeFrontend ||
		msg.From.Type == protocol.AgentTypeBackend ||
		msg.From.Type == protocol.AgentTypeDatabase ||
		msg.From.Type == protocol.AgentTypeSecurity ||
		msg.From.Type == protocol.AgentTypeDevOps ||
		msg.From.Type == protocol.AgentTypeRepo

	// If message has @mentions, ONLY respond if explicitly mentioned
	// This works even for agent-to-agent communication if explicitly mentioned
	if msg.HasMentions() {
		if msg.IsMentioned(ra.Info.ID) {
			log.Printf("[%s] ✅ EXPLICITLY MENTIONED - will respond", ra.Info.Name)
			return true
		}
		// Not mentioned but message has mentions - don't respond
		log.Printf("[%s] 🚫 IGNORING message with mentions (not mentioned)", ra.Info.Name)
		return false
	}

	// If no mentions specified, don't respond to other agents to prevent loops
	// Only respond to human messages when not explicitly mentioned
	if isFromAgent {
		log.Printf("[%s] 🚫 IGNORING message from agent (not mentioned)", ra.Info.Name)
		return false
	}

	// Always respond if mentioned by name in the content
	if strings.Contains(strings.ToLower(msg.Content), strings.ToLower(ra.Info.Name)) {
		return true
	}

	// Respond to questions about the repository
	content := strings.ToLower(msg.Content)

	// Check if it's a question
	isQuestion := msg.Type == protocol.MessageTypeQuestion ||
		strings.Contains(content, "?") ||
		strings.HasPrefix(content, "how ") ||
		strings.HasPrefix(content, "what ") ||
		strings.HasPrefix(content, "why ") ||
		strings.HasPrefix(content, "where ") ||
		strings.HasPrefix(content, "when ")

	if !isQuestion {
		return false
	}

	// Check if the question mentions the repository or related terms
	ra.mu.RLock()
	index := ra.index
	ra.mu.RUnlock()
	if index != nil {
		repoName := strings.ToLower(index.Name)
		if strings.Contains(content, repoName) {
			return true
		}

		// Check for code patterns/frameworks
		for _, pattern := range index.CodePatterns {
			if strings.Contains(content, strings.ToLower(pattern)) {
				return true
			}
		}
	}

	// Check for generic code structure questions
	codeQuestions := []string{
		"entry point", "main file", "structure", "architecture",
		"how does", "what does", "where is", "file structure",
		"dependencies", "packages", "imports",
	}
	for _, keyword := range codeQuestions {
		if strings.Contains(content, keyword) {
			return true
		}
	}

	return false
}

// generateRepoResponse generates a response based on repository knowledge
func (ra *RepoAgent) generateRepoResponse(ctx context.Context, msg *protocol.Message) (string, error) {
	ra.mu.RLock()
	index := ra.index
	ra.mu.RUnlock()
	if index == nil {
		return "I'm still learning about this repository. Please wait a moment.", nil
	}

	prompt := ra.buildRepoPrompt(msg, index)

	// Get recent conversation history for context
	history := ra.Context.History[msg.Channel]
	if len(history) > 10 {
		history = history[len(history)-10:]
	}

	eff := ra.EffectiveAIProvider(ctx, msg)
	if eff == nil {
		eff = ra.GetAIProvider()
	}
	response, err := eff.GenerateResponse(ctx, prompt, historyToMessages(history))
	if err != nil {
		return "", err
	}

	return response, nil
}

// buildRepoPrompt constructs a specialized prompt with repository context
func (ra *RepoAgent) buildRepoPrompt(msg *protocol.Message, index *repo.RepositoryIndex) string {
	var prompt strings.Builder

	PrependRulesAndAttachmentsForMonolithic(&prompt, msg, &ra.Info)

	prompt.WriteString(fmt.Sprintf("You are %s, a repository expert agent with deep knowledge of the %s codebase.\n\n",
		ra.Info.Name, index.Name))

	// Add repository overview
	prompt.WriteString("## Repository Overview\n")
	prompt.WriteString(index.ArchitectureDoc)
	prompt.WriteString("\n\n")

	// Add relevant source code files based on the question
	relevantFiles := repo.SearchRelevantFiles(msg.Content, index, 5)
	if len(relevantFiles) > 0 {
		prompt.WriteString("## Relevant Source Code Files\n\n")
		prompt.WriteString("I have access to the following files that may be relevant to your question:\n\n")

		totalSize := 0
		maxContextSize := 15000 // Limit context to avoid overwhelming the AI

		for _, file := range relevantFiles {
			// Decompress the file content
			content, err := repo.DecompressContent(file.Content)
			if err != nil {
				log.Printf("[%s] Failed to decompress %s: %v", ra.Info.Name, file.Path, err)
				continue
			}

			// Check if adding this file would exceed our context limit
			if totalSize+len(content) > maxContextSize {
				// Include partial content if we haven't added any files yet
				if totalSize == 0 {
					remaining := maxContextSize - totalSize
					if remaining > 500 {
						prompt.WriteString(fmt.Sprintf("### %s (%s)\n```%s\n",
							file.Path, file.Language, strings.ToLower(file.Language)))
						prompt.WriteString(content[:remaining])
						prompt.WriteString("\n... (truncated)\n```\n\n")
					}
				}
				break
			}

			// Add the full file
			prompt.WriteString(fmt.Sprintf("### %s (%s)\n```%s\n",
				file.Path, file.Language, strings.ToLower(file.Language)))
			prompt.WriteString(content)
			prompt.WriteString("\n```\n\n")

			totalSize += len(content)
		}
	}

	// Add key file information (config files, README, etc.)
	if len(index.KeyFiles) > 0 {
		prompt.WriteString("## Configuration & Documentation Files\n")
		for filename, content := range index.KeyFiles {
			// Skip if too long
			if len(content) > 2000 {
				prompt.WriteString(fmt.Sprintf("\n### %s (truncated)\n", filename))
				prompt.WriteString(content[:2000])
				prompt.WriteString("\n... (truncated)\n\n")
			} else {
				prompt.WriteString(fmt.Sprintf("\n### %s\n", filename))
				prompt.WriteString(content)
				prompt.WriteString("\n")
			}
		}
		prompt.WriteString("\n")
	}

	// Add recent commits context
	if index.GitInfo != nil && len(index.GitInfo.RecentCommits) > 0 {
		prompt.WriteString("## Recent Changes\n")
		for i, commit := range index.GitInfo.RecentCommits {
			if i >= 5 {
				break
			}
			prompt.WriteString(fmt.Sprintf("- %s: %s (%s)\n",
				commit.Hash, commit.Message, commit.Date.Format("2006-01-02")))
		}
		prompt.WriteString("\n")
	}

	prompt.WriteString("## Your Role\n")
	prompt.WriteString("As a repository expert, you should:\n")
	prompt.WriteString("1. Answer questions about the code structure, architecture, and implementation\n")
	prompt.WriteString("2. Reference specific files and code snippets when relevant\n")
	prompt.WriteString("3. Explain how different parts of the codebase work together\n")
	prompt.WriteString("4. Provide concrete code examples from the repository\n")
	prompt.WriteString("5. Be specific and cite actual files/code/line numbers when possible\n")
	prompt.WriteString("6. If you reference code, quote the relevant parts directly\n\n")

	prompt.WriteString(fmt.Sprintf("## Question from %s\n%s\n\n", msg.From.Name, msg.Content))
	prompt.WriteString("Provide a helpful, specific answer with code examples based on your knowledge of this repository.")

	return prompt.String()
}

// buildExpertise builds the expertise list based on repository analysis
func (ra *RepoAgent) buildExpertise() []string {
	ra.mu.RLock()
	index := ra.index
	ra.mu.RUnlock()
	if index == nil {
		return []string{}
	}

	expertise := []string{
		index.Name + " codebase",
		"Repository structure",
		"Code architecture",
	}

	// Add code patterns as expertise
	expertise = append(expertise, index.CodePatterns...)

	return expertise
}

// updateIndexProgress updates the indexing progress and notifies the hub
func (ra *RepoAgent) updateIndexProgress(progress int, message string) {
	ra.Info.IndexProgress = progress
	ra.updateAgentStatus()
	log.Printf("[%s] Indexing: %d%% - %s", ra.Info.Name, progress, message)
}

// updateAgentStatus notifies the hub about agent status changes
func (ra *RepoAgent) updateAgentStatus() {
	// Send status update message to the hub
	if ra.Context.CurrentChannel == "" {
		return
	}

	statusMsg := protocol.NewMessage(
		protocol.MessageTypeAgentStatus,
		ra.Context.CurrentChannel,
		ra.Info,
		fmt.Sprintf("Status: %s - %s (%d%%)",
			ra.Info.IndexingStatus,
			ra.getStatusDescription(),
			ra.Info.IndexProgress),
	)

	// Add status metadata
	statusMsg.Metadata["indexing_status"] = ra.Info.IndexingStatus
	statusMsg.Metadata["index_progress"] = ra.Info.IndexProgress
	statusMsg.Metadata["status"] = ra.Info.Status

	if err := ra.Hub.SendMessage(statusMsg); err != nil {
		log.Printf("[%s] Failed to send status update: %v", ra.Info.Name, err)
	}
}

// sendStatusMessage sends a user-visible status message to the channel
func (ra *RepoAgent) sendStatusMessage(message string) {
	if ra.Context.CurrentChannel == "" {
		return
	}

	statusMsg := protocol.NewMessage(
		protocol.MessageTypeSystemInfo,
		ra.Context.CurrentChannel,
		ra.Info,
		message,
	)

	if err := ra.Hub.SendMessage(statusMsg); err != nil {
		log.Printf("[%s] Failed to send status message: %v", ra.Info.Name, err)
	}
}

// getStatusDescription returns a human-readable status description
func (ra *RepoAgent) getStatusDescription() string {
	switch protocol.IndexingStatus(ra.Info.IndexingStatus) {
	case protocol.IndexingStatusIndexing:
		return "Analyzing repository"
	case protocol.IndexingStatusReindexing:
		return "Updating repository index"
	case protocol.IndexingStatusReady:
		return "Ready"
	case protocol.IndexingStatusError:
		return "Error occurred"
	default:
		return "Unknown"
	}
}

// Pause pauses the agent from responding
func (ra *RepoAgent) Pause() {
	ra.Info.IsPaused = true
	ra.Info.Status = "paused"
	ra.updateAgentStatus()
	log.Printf("[%s] Agent paused", ra.Info.Name)
}

// Unpause unpauses the agent
func (ra *RepoAgent) Unpause() {
	ra.Info.IsPaused = false
	ra.Info.Status = "active"
	ra.updateAgentStatus()
	log.Printf("[%s] Agent unpaused", ra.Info.Name)
}

// Cleanup removes the agent's stored data
func (ra *RepoAgent) Cleanup() error {
	// Stop watcher if running
	if ra.watcher != nil {
		ra.watcher.Stop()
		ra.watcher = nil
	}

	// Delete stored index using cache key
	if ra.storage != nil && ra.repoPath != "" {
		cacheKey, err := ra.storage.GetCacheKeyForPath(ra.repoPath)
		if err != nil {
			log.Printf("[%s] Failed to generate cache key for cleanup: %v", ra.Info.Name, err)
			return err
		}
		return ra.storage.DeleteIndex(cacheKey)
	}
	return nil
}

// checkAndRespondToPendingReview checks if this agent was created for a pending review and responds
func (ra *RepoAgent) checkAndRespondToPendingReview(ctx context.Context) {
	// Get the command handler from the hub to check for pending reviews
	commandHandler := ra.Hub.GetCommandHandler()
	if commandHandler == nil {
		log.Printf("[%s] Cannot access command handler for pending review check", ra.Info.Name)
		return
	}

	// Check if there's a pending review for this repository path
	pendingReview := commandHandler.GetPendingReview(ra.repoPath)
	if pendingReview == nil {
		return // No pending review
	}

	log.Printf("[%s] Found pending review, generating auto-response...", ra.Info.Name)

	// Generate a response to the original question
	response, err := ra.generateAutoResponse(ctx, pendingReview.OriginalMessage)
	if err != nil {
		log.Printf("[%s] Failed to generate auto-response: %v", ra.Info.Name, err)
		ra.sendStatusMessage("❌ Failed to generate response to your question. Please ask again.")
		return
	}

	// Send the response
	responseMsg := protocol.NewMessage(
		protocol.MessageTypeAnswer,
		pendingReview.OriginalMessage.Channel,
		ra.Info,
		response,
	)
	responseMsg.ReplyTo = pendingReview.OriginalMessage.ID

	// If the original message was in a thread, keep the response in the thread
	if pendingReview.OriginalMessage.IsInThread() {
		responseMsg.ThreadID = pendingReview.OriginalMessage.ThreadID
		responseMsg.IsThreadReply = true
	}

	// Send the response
	if err := ra.Hub.SendMessage(responseMsg); err != nil {
		log.Printf("[%s] Failed to send auto-response: %v", ra.Info.Name, err)
		ra.sendStatusMessage("❌ Failed to send response. Please ask again.")
		return
	}

	log.Printf("[%s] Auto-response sent successfully", ra.Info.Name)

	// Remove the pending review
	commandHandler.RemovePendingReview(ra.repoPath)
}

// generateAutoResponse generates an automatic response to the original question
func (ra *RepoAgent) generateAutoResponse(ctx context.Context, originalMsg *protocol.Message) (string, error) {
	// Build context about the repository
	repoContext := ra.buildRepositoryContext()

	// Get index name safely
	ra.mu.RLock()
	indexName := "the repository"
	if ra.index != nil {
		indexName = ra.index.Name
	}
	ra.mu.RUnlock()

	// Create a focused prompt for the original question
	prompt := fmt.Sprintf(`You are a repository expert for %s. A user asked: "%s"

Repository Context:
%s

Please provide a comprehensive response to their question based on your analysis of the repository. Focus on:
1. Direct answers to their specific question
2. Relevant code examples or file references
3. Architecture insights if applicable
4. Best practices or recommendations

Be specific and reference actual files, functions, or patterns from the repository when relevant.`,
		indexName,
		originalMsg.Content,
		repoContext)

	// Generate response using AI
	response, err := ra.AI.GenerateResponse(ctx, prompt, []protocol.Message{})
	if err != nil {
		return "", fmt.Errorf("failed to generate response: %w", err)
	}

	return response, nil
}

// buildRepositoryContext creates a summary of the repository for AI context
func (ra *RepoAgent) buildRepositoryContext() string {
	ra.mu.RLock()
	index := ra.index
	ra.mu.RUnlock()
	if index == nil {
		return "Repository not yet indexed."
	}

	var context strings.Builder
	context.WriteString(fmt.Sprintf("Repository: %s\n", index.Name))
	context.WriteString(fmt.Sprintf("Path: %s\n", index.Path))
	context.WriteString(fmt.Sprintf("Files: %d\n", index.FileCount))
	context.WriteString(fmt.Sprintf("Size: %.1f MB\n", float64(index.TotalSize)/(1024*1024)))

	if len(index.CodePatterns) > 0 {
		context.WriteString(fmt.Sprintf("Technologies: %s\n", strings.Join(index.CodePatterns, ", ")))
	}

	if index.ArchitectureDoc != "" {
		context.WriteString(fmt.Sprintf("Architecture: %s\n", index.ArchitectureDoc))
	}

	// Add key files information
	if len(index.KeyFiles) > 0 {
		context.WriteString("Key Files:\n")
		for filename, content := range index.KeyFiles {
			// Truncate content for context
			if len(content) > 500 {
				content = content[:500] + "..."
			}
			context.WriteString(fmt.Sprintf("- %s: %s\n", filename, content))
		}
	}

	return context.String()
}
